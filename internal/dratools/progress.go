package dratools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	grab "github.com/cavaliergopher/grab/v3"
)

const (
	progressBarWidth     = 12
	progressLabelMaxSize = 24
)

func interactiveWriter(w io.Writer) bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (s *DownloadService) waitForDownload(resp *grab.Response, outputPath string, attempt int) error {
	if !s.ProgressEnabled || s.ProgressOutput == nil {
		return resp.Err()
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	s.printDownloadProgress(resp, outputPath, attempt)
	for {
		select {
		case <-resp.Done:
			s.clearDownloadProgress()
			return resp.Err()
		case <-ticker.C:
			s.printDownloadProgress(resp, outputPath, attempt)
		}
	}
}

func (s *DownloadService) printDownloadProgress(resp *grab.Response, outputPath string, attempt int) {
	if resp == nil || s.ProgressOutput == nil {
		return
	}
	size := resp.Size()
	done := resp.BytesComplete()
	status := "download"
	if attempt > 0 {
		status = fmt.Sprintf("retry %d/%d", attempt, downloadRetryCount)
	}
	fmt.Fprintf(
		s.ProgressOutput,
		"\r\033[2K%s %s %s %s %s/s",
		MessagePrefix,
		status,
		fileProgressLabel(outputPath),
		progressBar(done, size),
		formatBytes(int64(resp.BytesPerSecond())),
	)
}

func (s *DownloadService) clearDownloadProgress() {
	if s.ProgressOutput != nil {
		fmt.Fprint(s.ProgressOutput, "\r\033[2K")
	}
}

func progressBar(done, size int64) string {
	if size <= 0 {
		return "[" + strings.Repeat("?", progressBarWidth) + "]"
	}
	if done > size {
		done = size
	}
	filled := int(float64(done) / float64(size) * progressBarWidth)
	if filled > progressBarWidth {
		filled = progressBarWidth
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat("-", progressBarWidth-filled) + fmt.Sprintf("] %3.0f%%", float64(done)/float64(size)*100)
}

func progressBytes(done, size int64) string {
	if size <= 0 {
		return formatBytes(done)
	}
	return formatBytes(done) + "/" + formatBytes(size)
}

func fileProgressLabel(path string) string {
	label := filepath.Base(path)
	if label == "." || label == string(filepath.Separator) || label == "" {
		label = path
	}
	runes := []rune(label)
	if len(runes) <= progressLabelMaxSize {
		return label
	}
	if progressLabelMaxSize <= 1 {
		return string(runes[:progressLabelMaxSize])
	}
	return "…" + string(runes[len(runes)-progressLabelMaxSize+1:])
}
