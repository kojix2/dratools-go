package dratools

import "fmt"

type RunRecordCollector struct {
	Client ResourceClient
}

func NewRunRecordCollector(client ResourceClient) *RunRecordCollector {
	return &RunRecordCollector{Client: client}
}

var traversableXrefTypes = map[string]bool{
	SRARunResourceType:        true,
	SRAExperimentResourceType: true,
	SRASampleResourceType:     true,
	SRAStudyResourceType:      true,
	SRASubmissionResourceType: true,
	BioProjectResourceType:    true,
	BioSampleResourceType:     true,
}

func (c *RunRecordCollector) Explore(record Record, tolerant bool, directRunFetchLimit *int) (*Node, error) {
	return c.explore(record, map[string]bool{}, RootRelation, tolerant, directRunFetchLimit)
}

func (c *RunRecordCollector) explore(record Record, seen map[string]bool, relation string, tolerant bool, directRunFetchLimit *int) (*Node, error) {
	node := nodeFromRecord(record, relation)
	if record[TypeKey] == SRARunResourceType {
		return node, nil
	}

	xrefs := mapSlice(record[DBXrefsKey])
	runXrefs := selectXrefs(xrefs, SRARunResourceType)
	if directRunFetchLimit != nil && len(runXrefs) > *directRunFetchLimit {
		for _, xref := range runXrefs {
			node.Children = append(node.Children, nodeFromXref(xref, DBXrefRelation, ""))
		}
		return node, nil
	}

	directChildren, err := c.exploreRunXrefs(runXrefs, seen, tolerant, directRunFetchLimit)
	if err != nil {
		return nil, err
	}
	if hasRunOrRunRecords(directChildren) {
		node.Children = append(node.Children, directChildren...)
		return node, nil
	}

	children, err := c.recursiveChildren(record, xrefs, seen, tolerant, directRunFetchLimit)
	if err != nil {
		return nil, err
	}
	node.Children = append(node.Children, children...)
	return node, nil
}

func (c *RunRecordCollector) recursiveChildren(record Record, xrefs []map[string]any, seen map[string]bool, tolerant bool, directRunFetchLimit *int) ([]*Node, error) {
	recursiveXrefs := make([]map[string]any, 0, len(xrefs))
	for _, xref := range xrefs {
		if traversableXrefTypes[stringValue(xref, TypeKey)] {
			recursiveXrefs = append(recursiveXrefs, xref)
		}
	}
	if err := validateRecursiveNonRunXrefCount(record, recursiveXrefs); err != nil {
		return nil, err
	}
	dbChildren, err := c.exploreEdges(recursiveXrefs, DBXrefRelation, seen, tolerant, directRunFetchLimit)
	if err != nil {
		return nil, err
	}
	childChildren, err := c.exploreEdges(mapSlice(record[ChildBioProjectsKey]), ChildBioProjectRelation, seen, tolerant, directRunFetchLimit)
	if err != nil {
		return nil, err
	}
	return append(dbChildren, childChildren...), nil
}

func validateRecursiveNonRunXrefCount(record Record, xrefs []map[string]any) error {
	limit, err := maxRecursiveNonRunXrefs()
	if err != nil || limit == nil {
		return err
	}
	count := 0
	for _, xref := range xrefs {
		if stringValue(xref, TypeKey) != SRARunResourceType {
			count++
		}
	}
	if count <= *limit {
		return nil
	}
	accession := recordAccession(record)
	if accession == "" {
		accession = "record"
	}
	msg := fmt.Sprintf("%s has %d linked non-run records; refine to an experiment/sample accession before run expansion, or set %s=unlimited", accession, count, maxRecursiveNonRunXrefsEnv)
	return newError("invalid_record", msg)
}

func (c *RunRecordCollector) exploreEdges(xrefs []map[string]any, relation string, seen map[string]bool, tolerant bool, directRunFetchLimit *int) ([]*Node, error) {
	var children []*Node
	for _, xref := range xrefs {
		if !traversableXrefTypes[stringValue(xref, TypeKey)] {
			continue
		}
		key := xrefAccession(xref)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		child, err := c.exploreXref(xref, relation, seen, tolerant, directRunFetchLimit)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	return children, nil
}

func (c *RunRecordCollector) exploreRunXrefs(xrefs []map[string]any, seen map[string]bool, tolerant bool, directRunFetchLimit *int) ([]*Node, error) {
	var fetchable []map[string]any
	var accessions []string
	for _, xref := range xrefs {
		if !traversableXrefTypes[stringValue(xref, TypeKey)] {
			continue
		}
		accession := xrefAccession(xref)
		if accession == "" || seen[accession] {
			continue
		}
		seen[accession] = true
		fetchable = append(fetchable, xref)
		accessions = append(accessions, accession)
	}
	if len(fetchable) == 0 {
		return nil, nil
	}
	records, err := c.Client.FetchResourceRecordsBulk(SRARunResourceType, accessions, false)
	if err != nil {
		return nil, err
	}
	children := make([]*Node, 0, len(fetchable))
	for _, xref := range fetchable {
		accession := xrefAccession(xref)
		if record, ok := records[accession]; ok {
			child, err := c.explore(record, seen, DBXrefRelation, tolerant, directRunFetchLimit)
			if err != nil {
				return nil, err
			}
			children = append(children, child)
			continue
		}
		if tolerant {
			children = append(children, nodeFromXref(xref, DBXrefRelation, "not found: "+accession))
			continue
		}
		return nil, newError("not_found", "not found: sra-run/"+accession)
	}
	return children, nil
}

func (c *RunRecordCollector) exploreXref(xref map[string]any, relation string, seen map[string]bool, tolerant bool, directRunFetchLimit *int) (*Node, error) {
	accession := xrefAccession(xref)
	if accession == "" {
		return nil, newError("invalid_record", "xref has no identifier")
	}
	record, err := c.Client.FetchResourceRecord(stringValue(xref, TypeKey), accession)
	if err != nil {
		if tolerant {
			return nodeFromXref(xref, relation, err.Error()), nil
		}
		return nil, err
	}
	return c.explore(record, seen, relation, tolerant, directRunFetchLimit)
}

func nodeFromRecord(record Record, relation string) *Node {
	n := &Node{
		Relation:  relation,
		Type:      stringValue(record, TypeKey),
		Accession: recordAccession(record),
		ObjectTyp: stringValue(record, "objectType"),
	}
	if n.Type == SRARunResourceType {
		n.Record = record
	}
	return n
}

func nodeFromXref(xref map[string]any, relation, errMsg string) *Node {
	return &Node{
		Relation:  relation,
		Type:      stringValue(xref, TypeKey),
		Accession: xrefAccession(xref),
		Err:       errMsg,
	}
}

func selectXrefs(xrefs []map[string]any, typ string) []map[string]any {
	out := make([]map[string]any, 0)
	for _, xref := range xrefs {
		if stringValue(xref, TypeKey) == typ {
			out = append(out, xref)
		}
	}
	return out
}

func hasRunOrRunRecords(nodes []*Node) bool {
	for _, node := range nodes {
		if node.IsRun() || len(node.RunRecords()) > 0 {
			return true
		}
	}
	return false
}
