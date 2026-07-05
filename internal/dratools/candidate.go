package dratools

import (
	"net/url"
	"path"
	"strings"
)

type DownloadCandidate struct {
	RunAccession string `json:"run_accession"`
	Type         string `json:"type"`
	URL          string `json:"url"`
	FTPURL       string `json:"ftp_url"`
	Size         any    `json:"size"`
	MD5          string `json:"md5"`
}

func BuildDownloadCandidates(runRecord Record) []DownloadCandidate {
	runAccession := recordAccession(runRecord)
	items := mapSlice(runRecord[DistributionKey])
	seen := map[string]bool{}
	out := make([]DownloadCandidate, 0, len(items))
	for _, item := range items {
		fileType := fileTypeFromDistribution(item)
		if fileType == "" {
			continue
		}
		d := DownloadCandidate{
			RunAccession: runAccession,
			Type:         fileType,
			URL:          stringValue(item, ContentURLKey),
			Size:         item[ContentSizeKey],
			MD5:          firstNonEmpty(stringValue(item, MD5Key), stringValue(item, MD5SumKey)),
		}
		key := d.Type + "\x00" + d.URL + "\x00" + d.FTPURL
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, d)
	}
	return out
}

func fileTypeFromDistribution(item map[string]any) string {
	switch strings.ToLower(stringValue(item, EncodingFormatKey)) {
	case FileTypeSRA:
		return FileTypeSRA
	case FileTypeFASTQ:
		return FileTypeFASTQ
	default:
		return ""
	}
}

func (d DownloadCandidate) URLForProtocol(protocol string) (string, error) {
	switch protocol {
	case "ftp":
		if d.FTPURL != "" {
			return d.FTPURL, nil
		}
		return d.URL, nil
	case "http", "https":
		if d.URL != "" {
			return d.URL, nil
		}
		return d.FTPURL, nil
	default:
		return "", newError("invalid_protocol", "unknown protocol: "+protocol)
	}
}

func (d DownloadCandidate) FilenameForProtocol(protocol string) (string, error) {
	raw, err := d.URLForProtocol(protocol)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	return path.Base(u.Path), nil
}

func (d DownloadCandidate) IsDirectoryURL(protocol string) bool {
	raw, err := d.URLForProtocol(protocol)
	if err != nil {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.HasSuffix(u.Path, "/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
