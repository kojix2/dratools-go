package dratools

import "testing"

func TestResourceTypeFor(t *testing.T) {
	tests := map[string]string{
		"DRR000001":   SRARunResourceType,
		"SRX000001":   SRAExperimentResourceType,
		"ERS000001":   SRASampleResourceType,
		"DRP000001":   SRAStudyResourceType,
		"SRA000001":   SRASubmissionResourceType,
		"PRJNA341783": BioProjectResourceType,
		"SAMD000001":  BioSampleResourceType,
		"SAMN000001":  BioSampleResourceType,
	}
	for accession, want := range tests {
		got, err := ResourceTypeFor(accession)
		if err != nil {
			t.Fatalf("ResourceTypeFor(%q) returned error: %v", accession, err)
		}
		if got != want {
			t.Fatalf("ResourceTypeFor(%q) = %q, want %q", accession, got, want)
		}
	}
}

func TestResourceTypeForUnsupported(t *testing.T) {
	if _, err := ResourceTypeFor("GSE149689"); err == nil {
		t.Fatal("expected unsupported accession error")
	}
}
