package dratools

import (
	"fmt"
	"strings"
)

type AccessionResolver struct {
	Client    ResourceClient
	Collector *RunRecordCollector
}

type ResourceClient interface {
	FetchResourceRecord(resourceType, accession string) (Record, error)
	FetchDBLinks(resourceType, accession, target string) ([]map[string]any, error)
	FetchResourceRecordsBulk(resourceType string, accessions []string, includeDBXrefs bool) (map[string]Record, error)
	FetchDBLinkCounts(items []map[string]string) (map[string]map[string]int, error)
}

func NewAccessionResolver(client ResourceClient) *AccessionResolver {
	return &AccessionResolver{
		Client:    client,
		Collector: NewRunRecordCollector(client),
	}
}

func (r *AccessionResolver) ResolveDownloads(accession, fileType string) ([]DownloadCandidate, error) {
	accession = strings.ToUpper(accession)
	record, err := r.FetchRecordFor(accession)
	if err != nil {
		return nil, err
	}
	return r.ResolveDownloadsFromRecord(accession, record, fileType)
}

func (r *AccessionResolver) ResolveDownloadsFromRecord(accession string, record Record, fileType string) ([]DownloadCandidate, error) {
	tree, err := r.ResolveTreeFromRecord(record, fileType, false, nil)
	if err != nil {
		return nil, err
	}
	downloads := tree.Downloads()
	if len(downloads) == 0 {
		return nil, newError("not_found", fmt.Sprintf("download URL not found: %s (type=%s)", strings.ToUpper(accession), fileType))
	}
	return downloads, nil
}

func (r *AccessionResolver) ResolveTree(accession, fileType string, tolerant bool, directRunFetchLimit *int) (*Node, error) {
	accession = strings.ToUpper(accession)
	record, err := r.FetchRecordFor(accession)
	if err != nil {
		return nil, err
	}
	return r.ResolveTreeFromRecord(record, fileType, tolerant, directRunFetchLimit)
}

func (r *AccessionResolver) ResolveTreeFromRecord(record Record, fileType string, tolerant bool, directRunFetchLimit *int) (*Node, error) {
	tree, err := r.Collector.Explore(record, tolerant, directRunFetchLimit)
	if err != nil {
		return nil, err
	}
	attachDownloads(tree, fileType)
	return tree, nil
}

func (r *AccessionResolver) FetchRecordFor(accession string) (Record, error) {
	resourceType, err := ResourceTypeFor(accession)
	if err != nil {
		return nil, err
	}
	return r.Client.FetchResourceRecord(resourceType, accession)
}

func (r *AccessionResolver) DirectRunAccessionsFor(accession string) ([]string, error) {
	accession = strings.ToUpper(accession)
	resourceType, err := ResourceTypeFor(accession)
	if err != nil {
		return nil, err
	}
	if resourceType == SRARunResourceType {
		return []string{accession}, nil
	}
	xrefs, err := r.Client.FetchDBLinks(resourceType, accession, SRARunResourceType)
	if err != nil {
		return nil, err
	}
	var runs []string
	for _, xref := range xrefs {
		runs = append(runs, xrefAccession(xref))
	}
	return uniqueStrings(runs), nil
}

func (r *AccessionResolver) DirectRunCountFor(accession string) (int, error) {
	accession = strings.ToUpper(accession)
	resourceType, err := ResourceTypeFor(accession)
	if err != nil {
		return 0, err
	}
	if resourceType == SRARunResourceType {
		return 1, nil
	}
	counts, err := r.Client.FetchDBLinkCounts([]map[string]string{{"type": resourceType, "id": accession}})
	if err != nil {
		return 0, err
	}
	return counts[resourceType+"\x00"+accession][SRARunResourceType], nil
}

func attachDownloads(node *Node, fileType string) {
	if node == nil {
		return
	}
	if node.IsRun() && node.Record != nil {
		for _, download := range BuildDownloadCandidates(node.Record) {
			if fileType != FileTypeAll && download.Type != fileType {
				continue
			}
			d := download
			node.Children = append(node.Children, &Node{
				Relation:  DownloadRelation,
				Type:      download.Type,
				Accession: download.RunAccession,
				URL:       download.URL,
				Download:  &d,
			})
		}
	}
	for _, child := range node.Children {
		if !child.IsDownload() {
			attachDownloads(child, fileType)
		}
	}
}
