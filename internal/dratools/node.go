package dratools

const (
	RootRelation            = "root"
	DBXrefRelation          = "db_xref"
	ChildBioProjectRelation = "child_bioproject"
	DownloadRelation        = "download"
)

type Node struct {
	Relation  string
	Type      string
	Accession string
	ObjectTyp string
	Record    Record
	URL       string
	Err       string
	Children  []*Node
	Download  *DownloadCandidate
}

func (n *Node) IsRun() bool {
	return n != nil && n.Type == SRARunResourceType
}

func (n *Node) IsDownload() bool {
	return n != nil && n.Relation == DownloadRelation
}

func (n *Node) RunRecords() []Record {
	if n == nil {
		return nil
	}
	var out []Record
	if n.IsRun() && n.Record != nil {
		out = append(out, n.Record)
	}
	for _, child := range n.Children {
		out = append(out, child.RunRecords()...)
	}
	return out
}

func (n *Node) RunAccessions() []string {
	if n == nil {
		return nil
	}
	var out []string
	if n.IsRun() && n.Accession != "" {
		out = append(out, n.Accession)
	}
	for _, child := range n.Children {
		if child.IsDownload() {
			continue
		}
		out = append(out, child.RunAccessions()...)
	}
	return out
}

func (n *Node) Downloads() []DownloadCandidate {
	if n == nil {
		return nil
	}
	var out []DownloadCandidate
	if n.Download != nil {
		out = append(out, *n.Download)
	}
	for _, child := range n.Children {
		out = append(out, child.Downloads()...)
	}
	return out
}

func (n *Node) Errors() []string {
	if n == nil {
		return nil
	}
	var out []string
	if n.Err != "" {
		out = append(out, n.Err)
	}
	for _, child := range n.Children {
		out = append(out, child.Errors()...)
	}
	return out
}
