package dratools

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := map[int64]string{
		0:           "0 B",
		999:         "999 B",
		1024:        "1.0 KiB",
		1024 * 1024: "1.0 MiB",
	}
	for input, want := range tests {
		if got := formatBytes(input); got != want {
			t.Fatalf("formatBytes(%d) = %q, want %q", input, got, want)
		}
	}
}
