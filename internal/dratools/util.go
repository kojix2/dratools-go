package dratools

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Record map[string]any

func stringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	switch v := m[key].(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func recordAccession(record Record) string {
	for _, key := range []string{AccessionKey, IdentifierKey, IDKey, PrimaryIDKey} {
		if value := strings.TrimSpace(stringValue(record, key)); value != "" {
			return value
		}
	}
	return ""
}

func xrefAccession(xref map[string]any) string {
	for _, key := range []string{IDKey, IdentifierKey, AccessionKey} {
		if value := strings.TrimSpace(stringValue(xref, key)); value != "" {
			return value
		}
	}
	return ""
}

func mapSlice(value any) []map[string]any {
	switch v := value.(type) {
	case nil:
		return nil
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case []map[string]any:
		return v
	case map[string]any:
		return []map[string]any{v}
	default:
		return nil
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
