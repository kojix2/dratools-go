package dratools

import (
	"strconv"
	"strings"
)

type TreeRenderer struct {
	FileType         string
	SummaryThreshold int
}

func NewTreeRenderer(fileType string, summaryThreshold int) TreeRenderer {
	if summaryThreshold <= 0 {
		summaryThreshold = 5
	}
	return TreeRenderer{FileType: fileType, SummaryThreshold: summaryThreshold}
}

func (r TreeRenderer) Render(root *Node) string {
	lines := []string{r.labelFor(root)}
	r.renderChildren(root.Children, "", &lines)
	return strings.Join(lines, "\n")
}

func (r TreeRenderer) renderChildren(children []*Node, prefix string, lines *[]string) {
	displayChildren := r.summarizedChildren(children)
	for i, child := range displayChildren {
		last := i == len(displayChildren)-1
		connector := "├─ "
		if last {
			connector = "└─ "
		}
		*lines = append(*lines, prefix+connector+r.labelFor(child))
		if len(child.Children) == 0 {
			continue
		}
		childPrefix := prefix + "│  "
		if last {
			childPrefix = prefix + "   "
		}
		r.renderChildren(child.Children, childPrefix, lines)
	}
}

func (r TreeRenderer) summarizedChildren(children []*Node) []*Node {
	if !r.summarizableRunGroup(children) {
		return children
	}
	leafLabel := "no " + r.FileType + " downloads"
	for _, child := range children {
		if child.Record == nil {
			leafLabel = r.FileType + " downloads not expanded"
			break
		}
	}
	return []*Node{{
		Type:      SRARunResourceType,
		Accession: strconv.Itoa(len(children)) + " records",
		Children:  []*Node{{Type: leafLabel}},
	}}
}

func (r TreeRenderer) summarizableRunGroup(children []*Node) bool {
	if len(children) <= r.SummaryThreshold {
		return false
	}
	for _, child := range children {
		if !child.IsRun() || len(child.Downloads()) > 0 || len(child.Errors()) > 0 {
			return false
		}
	}
	return true
}

func (r TreeRenderer) labelFor(node *Node) string {
	if node == nil {
		return ""
	}
	if node.IsDownload() {
		return compactJoin(" ", node.Type, node.URL)
	}
	parts := []string{}
	if node.Relation == ChildBioProjectRelation {
		parts = append(parts, "childBioProject")
	}
	if node.Relation != ChildBioProjectRelation {
		parts = append(parts, node.Type)
	}
	parts = append(parts, node.Accession)
	parts = append(parts, node.ObjectTyp)
	if node.Err != "" {
		parts = append(parts, "error: "+node.Err)
	}
	return compactJoin(" ", parts...)
}

func compactJoin(sep string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, sep)
}
