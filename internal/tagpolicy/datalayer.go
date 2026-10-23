package tagpolicy

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/pageguard/pageguard/internal/domain"
)

// entryValue decodes a data-layer entry's raw JSON into a generic value. It
// returns false when the entry carries no usable value.
func entryValue(entry domain.DataLayerEntry) (any, bool) {
	raw := entry.Value
	if len(raw) == 0 {
		return nil, false
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false
	}
	return v, true
}

// entryEvent returns the top-level "event" name of a data-layer entry, or "".
func entryEvent(entry domain.DataLayerEntry) string {
	v, ok := entryValue(entry)
	if !ok {
		return ""
	}
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	ev, _ := m["event"].(string)
	return ev
}

// splitPath tokenises a dotted path, treating bracketed indices as segments so
// that "items[0].sku" and "items.0.sku" are equivalent.
func splitPath(path string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range path {
		switch r {
		case '.':
			flush()
		case '[':
			flush()
		case ']':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

// resolvePath navigates a decoded JSON value by a dotted path. It returns false
// when any segment is missing or the shape does not allow traversal.
func resolvePath(root any, path string) (any, bool) {
	if path == "" {
		return root, true
	}
	cur := root
	for _, tok := range splitPath(path) {
		switch node := cur.(type) {
		case map[string]any:
			next, ok := node[tok]
			if !ok {
				return nil, false
			}
			cur = next
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i < 0 || i >= len(node) {
				return nil, false
			}
			cur = node[i]
		default:
			return nil, false
		}
	}
	return cur, true
}
