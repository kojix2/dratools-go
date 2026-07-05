package dratools

import (
	"strings"
	"testing"
)

func TestProgressBarKnownSize(t *testing.T) {
	got := progressBar(50, 100)
	if !strings.Contains(got, "50%") {
		t.Fatalf("progress bar %q does not include percentage", got)
	}
	if !strings.Contains(got, strings.Repeat("=", progressBarWidth/2)) {
		t.Fatalf("progress bar %q does not show filled segment", got)
	}
}

func TestProgressBarUnknownSize(t *testing.T) {
	got := progressBar(50, -1)
	if !strings.Contains(got, strings.Repeat("?", progressBarWidth)) {
		t.Fatalf("progress bar %q does not show unknown size marker", got)
	}
}

func TestProgressBytes(t *testing.T) {
	if got := progressBytes(1024, 2048); got != "1.0 KiB/2.0 KiB" {
		t.Fatalf("progressBytes = %q, want 1.0 KiB/2.0 KiB", got)
	}
}
