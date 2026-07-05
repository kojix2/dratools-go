package dratools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildDownloadCandidates(t *testing.T) {
	record := Record{
		AccessionKey: "DRR000001",
		DistributionKey: []any{
			map[string]any{
				EncodingFormatKey: "sra",
				ContentURLKey:     "https://example.test/DRR000001.sra",
				ContentSizeKey:    float64(123),
				MD5SumKey:         "abc",
			},
			map[string]any{
				EncodingFormatKey: "fastq",
				ContentURLKey:     "https://example.test/DRR000001_1.fastq.gz",
			},
			map[string]any{
				EncodingFormatKey: "bam",
				ContentURLKey:     "https://example.test/ignored.bam",
			},
		},
	}
	got := BuildDownloadCandidates(record)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].RunAccession != "DRR000001" || got[0].Type != FileTypeSRA || got[0].MD5 != "abc" {
		t.Fatalf("unexpected first candidate: %#v", got[0])
	}
	if got[1].Type != FileTypeFASTQ {
		t.Fatalf("unexpected second candidate: %#v", got[1])
	}
}

func TestDownloadCandidateURLForProtocol(t *testing.T) {
	d := DownloadCandidate{URL: "https://example.test/a.sra", FTPURL: "ftp://example.test/a.sra"}
	got, err := d.URLForProtocol("ftp")
	if err != nil {
		t.Fatal(err)
	}
	if got != d.FTPURL {
		t.Fatalf("ftp URL = %q, want %q", got, d.FTPURL)
	}
}

func TestDownloadCandidateJSONIncludesEmptyOptionalValues(t *testing.T) {
	payload, err := json.Marshal(DownloadCandidate{
		RunAccession: "DRR000001",
		Type:         FileTypeSRA,
		URL:          "https://example.test/DRR000001.sra",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(payload)
	for _, key := range []string{`"ftp_url"`, `"size"`, `"md5"`} {
		if !strings.Contains(got, key) {
			t.Fatalf("JSON %s does not include %s", got, key)
		}
	}
}

func TestDownloadURLRejectsInvalidMD5Checksum(t *testing.T) {
	service := NewDownloadService()
	err := service.downloadURL("https://example.test/DRR000001.sra", "DRR000001.sra", "not-hex", true)
	if err == nil {
		t.Fatal("expected invalid md5 checksum error")
	}
	if !strings.Contains(err.Error(), "invalid md5 checksum") {
		t.Fatalf("error = %v, want invalid md5 checksum", err)
	}
}
