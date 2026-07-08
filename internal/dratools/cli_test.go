package dratools

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRunURLContinuesAfterAccessionFailure(t *testing.T) {
	client := &fakeResourceClient{records: map[string]Record{
		"DRR000001": testRunRecord("DRR000001"),
		"DRR000003": testRunRecord("DRR000003"),
	}}

	var stdout, stderr bytes.Buffer
	a := testApp(client, &stdout, &stderr)

	status := a.run([]string{"url", "DRR000001", "DRR000002", "DRR000003"})

	if status != 1 {
		t.Fatalf("status = %d, want 1", status)
	}
	wantLines := []string{
		"https://example.test/DRR000001.sra",
		"https://example.test/DRR000003.sra",
	}
	if got := strings.Split(strings.TrimSpace(stdout.String()), "\n"); !reflect.DeepEqual(got, wantLines) {
		t.Fatalf("stdout lines = %#v, want %#v\nstdout:\n%s", got, wantLines, stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, "[dratools] url: DRR000002:") {
		t.Fatalf("stderr does not include accession-scoped error:\n%s", got)
	}
	if strings.Contains(stderr.String(), "one or more accessions failed") {
		t.Fatalf("stderr includes duplicate aggregate error:\n%s", stderr.String())
	}
}

func TestRunURLJSONWritesSuccessfulAccessionsWhenLaterAccessionFails(t *testing.T) {
	client := &fakeResourceClient{records: map[string]Record{
		"DRR000001": testRunRecord("DRR000001"),
	}}

	var stdout, stderr bytes.Buffer
	a := testApp(client, &stdout, &stderr)

	status := a.run([]string{"url", "--json", "DRR000001", "DRR000002"})

	if status != 1 {
		t.Fatalf("status = %d, want 1", status)
	}
	var candidates []DownloadCandidate
	if err := json.Unmarshal(stdout.Bytes(), &candidates); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if len(candidates) != 1 || candidates[0].RunAccession != "DRR000001" {
		t.Fatalf("candidates = %#v, want one DRR000001 candidate", candidates)
	}
	if got := stderr.String(); !strings.Contains(got, "[dratools] url: DRR000002:") {
		t.Fatalf("stderr does not include accession-scoped error:\n%s", got)
	}
}

func testApp(client ResourceClient, stdout, stderr *bytes.Buffer) *app {
	return &app{
		resolver:   NewAccessionResolver(client),
		downloader: NewDownloadService(),
		stdin:      os.Stdin,
		stdout:     stdout,
		stderr:     stderr,
	}
}

type fakeResourceClient struct {
	records map[string]Record
}

func (c *fakeResourceClient) FetchResourceRecord(resourceType, accession string) (Record, error) {
	record, ok := c.records[accession]
	if !ok {
		return nil, newError("not_found", "not found: "+resourceType+"/"+accession)
	}
	return record, nil
}

func (c *fakeResourceClient) FetchDBLinks(resourceType, accession, target string) ([]map[string]any, error) {
	return nil, newError("not_found", "not found: "+resourceType+"/"+accession)
}

func (c *fakeResourceClient) FetchResourceRecordsBulk(resourceType string, accessions []string, includeDBXrefs bool) (map[string]Record, error) {
	out := map[string]Record{}
	for _, accession := range accessions {
		if record, ok := c.records[accession]; ok {
			out[accession] = record
		}
	}
	return out, nil
}

func (c *fakeResourceClient) FetchDBLinkCounts(items []map[string]string) (map[string]map[string]int, error) {
	return map[string]map[string]int{}, nil
}

func testRunRecord(accession string) Record {
	return Record{
		AccessionKey: accession,
		TypeKey:      SRARunResourceType,
		DistributionKey: []any{
			map[string]any{
				EncodingFormatKey: FileTypeSRA,
				ContentURLKey:     "https://example.test/" + accession + ".sra",
			},
		},
	}
}
