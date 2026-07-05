package dratools

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	grab "github.com/cavaliergopher/grab/v3"
)

// downloadRetryCount is how many extra attempts a download gets after the
// first failure. grab resumes the partial file on each retry, so retrying is
// cheap and does not re-fetch already-downloaded bytes.
const downloadRetryCount = 3

type DownloadService struct {
	Client          *http.Client
	ProgressOutput  io.Writer
	ProgressEnabled bool
}

type DownloadResult struct {
	Path    string
	Skipped bool
}

func NewDownloadService() *DownloadService {
	return &DownloadService{Client: &http.Client{}}
}

func (s *DownloadService) ProbeDownload(download DownloadCandidate, protocol string, timeout time.Duration) error {
	raw, err := download.URLForProtocol(protocol)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", Name+"/"+Version)
	req.Header.Set("Range", "bytes=0-0")
	resp, err := s.Client.Do(req)
	if err != nil {
		return wrapError("network", "failed to fetch "+raw, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return newError("network", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, raw))
}

func (s *DownloadService) ContentLengths(download DownloadCandidate, protocol string, timeout time.Duration) []int64 {
	raw, err := download.URLForProtocol(protocol)
	if err != nil || !isHTTPURL(raw) {
		return []int64{-1}
	}
	if download.IsDirectoryURL(protocol) {
		files, err := s.directoryFileURLs(raw, timeout)
		if err != nil || len(files) == 0 {
			return []int64{-1}
		}
		out := make([]int64, 0, len(files))
		for _, fileURL := range files {
			out = append(out, s.safeHeadContentLength(fileURL, timeout))
		}
		return out
	}
	return []int64{s.safeHeadContentLength(raw, timeout)}
}

func (s *DownloadService) SaveDownload(download DownloadCandidate, outdir, protocol string, verify, force, skipExisting bool) (DownloadResult, error) {
	if err := os.MkdirAll(outdir, 0o755); err != nil {
		return DownloadResult{}, err
	}
	raw, err := download.URLForProtocol(protocol)
	if err != nil {
		return DownloadResult{}, err
	}
	if download.IsDirectoryURL(protocol) {
		return DownloadResult{}, newError("invalid_record", "download URL points to a directory: "+raw)
	}
	filename, err := download.FilenameForProtocol(protocol)
	if err != nil {
		return DownloadResult{}, err
	}
	outputPath := filepath.Join(outdir, filename)
	if force {
		_ = os.Remove(outputPath)
	}
	if shouldSkip, err := s.shouldSkipExisting(outputPath, download, raw, skipExisting); err != nil {
		return DownloadResult{}, err
	} else if shouldSkip {
		return DownloadResult{Path: outputPath, Skipped: true}, nil
	}
	if err := s.downloadURL(raw, outputPath, download.MD5, verify); err != nil {
		return DownloadResult{}, err
	}
	return DownloadResult{Path: outputPath}, nil
}

func (s *DownloadService) shouldSkipExisting(outputPath string, download DownloadCandidate, raw string, skipExisting bool) (bool, error) {
	info, err := os.Stat(outputPath)
	if err != nil {
		return false, nil
	}
	if info.IsDir() {
		return false, nil
	}
	if skipExisting {
		return true, nil
	}
	if strings.TrimSpace(download.MD5) != "" {
		ok, err := md5Matches(outputPath, download.MD5)
		return ok, err
	}
	remote := s.safeHeadContentLength(raw, 10*time.Second)
	if remote < 0 {
		return false, nil
	}
	if info.Size() == remote {
		return true, nil
	}
	if info.Size() > remote {
		return false, newError("invalid_record", fmt.Sprintf("existing file is larger than remote file: %s (local=%d, remote=%d); use --force to re-download", outputPath, info.Size(), remote))
	}
	return false, nil
}

// downloadURL fetches raw into outputPath using grab, which handles resuming
// partial files, HTTP range requests, and (when a checksum is supplied)
// post-download md5 verification. Failed attempts are retried; because grab
// resumes, a retry continues from where the previous attempt stopped.
func (s *DownloadService) downloadURL(raw, outputPath, expectedMD5 string, verify bool) error {
	client := grab.NewClient()
	client.UserAgent = Name + "/" + Version
	if s.Client != nil {
		client.HTTPClient = s.Client
	}

	var lastErr error
	for attempt := 0; attempt <= downloadRetryCount; attempt++ {
		req, err := grab.NewRequest(outputPath, raw)
		if err != nil {
			return wrapError("network", "failed to build request "+raw, err)
		}
		if verify {
			if sum := strings.TrimSpace(expectedMD5); sum != "" {
				decoded, err := hex.DecodeString(sum)
				if err != nil {
					return wrapError("checksum", "invalid md5 checksum "+sum, err)
				}
				// deleteOnError=true removes the file if the md5 mismatches.
				req.SetChecksum(md5.New(), decoded, true)
			}
		}
		resp := client.Do(req)
		if err := s.waitForDownload(resp, outputPath, attempt); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return wrapError("network", "failed to download "+raw, lastErr)
}

func (s *DownloadService) safeHeadContentLength(raw string, timeout time.Duration) int64 {
	length, err := s.headContentLength(raw, timeout)
	if err != nil {
		return -1
	}
	return length
}

func (s *DownloadService) headContentLength(raw string, timeout time.Duration) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, raw, nil)
	if err != nil {
		return -1, err
	}
	req.Header.Set("User-Agent", Name+"/"+Version)
	resp, err := s.Client.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return -1, newError("network", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, raw))
	}
	return resp.ContentLength, nil
}

var fastqHrefPattern = regexp.MustCompile(`(?i)href=(["'])([^"']*\.fastq[^"']*)["']`)

func (s *DownloadService) directoryFileURLs(raw string, timeout time.Duration) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", Name+"/"+Version)
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newError("network", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, raw))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, match := range fastqHrefPattern.FindAllSubmatch(body, -1) {
		href := html.UnescapeString(string(match[2]))
		rel, err := url.Parse(href)
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(rel).String()
		if !seen[resolved] {
			seen[resolved] = true
			out = append(out, resolved)
		}
	}
	return out, nil
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func md5Matches(path, expected string) (bool, error) {
	actual, err := fileMD5(path)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(actual, strings.TrimSpace(expected)), nil
}

func fileMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
