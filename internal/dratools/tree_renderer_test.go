package dratools

import (
	"strings"
	"testing"
)

func TestTreeRenderer(t *testing.T) {
	root := &Node{
		Type:      BioProjectResourceType,
		Accession: "PRJNA000001",
		Children: []*Node{
			{
				Relation:  DBXrefRelation,
				Type:      SRARunResourceType,
				Accession: "DRR000001",
				Children: []*Node{
					{
						Relation: DownloadRelation,
						Type:     FileTypeSRA,
						URL:      "https://example.test/DRR000001.sra",
					},
				},
			},
		},
	}
	got := NewTreeRenderer(FileTypeSRA, 5).Render(root)
	for _, want := range []string{"bioproject PRJNA000001", "sra-run DRR000001", "sra https://example.test/DRR000001.sra"} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered tree does not contain %q:\n%s", want, got)
		}
	}
}
