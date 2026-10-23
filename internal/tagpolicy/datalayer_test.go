package tagpolicy

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

func TestSplitPath(t *testing.T) {
	cases := map[string][]string{
		"":               nil,
		"a":              {"a"},
		"a.b.c":          {"a", "b", "c"},
		"items[0].sku":   {"items", "0", "sku"},
		"items.0.sku":    {"items", "0", "sku"},
		"cart.items[12]": {"cart", "items", "12"},
	}
	for in, want := range cases {
		if got := splitPath(in); !reflect.DeepEqual(got, want) {
			t.Errorf("splitPath(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolvePath(t *testing.T) {
	root := map[string]any{
		"page": map[string]any{"type": "product"},
		"items": []any{
			map[string]any{"sku": "A1"},
			map[string]any{"sku": "B2"},
		},
	}
	cases := []struct {
		path  string
		want  any
		found bool
	}{
		{"page.type", "product", true},
		{"items[0].sku", "A1", true},
		{"items.1.sku", "B2", true},
		{"page.missing", nil, false},
		{"items[9].sku", nil, false},
		{"page.type.deeper", nil, false},
	}
	for _, c := range cases {
		got, ok := resolvePath(root, c.path)
		if ok != c.found || !reflect.DeepEqual(got, c.want) {
			t.Errorf("resolvePath(%q) = (%v, %v), want (%v, %v)", c.path, got, ok, c.want, c.found)
		}
	}
}

func TestEntryEventAndValue(t *testing.T) {
	entry := domain.DataLayerEntry{Name: "dataLayer[0]", Value: json.RawMessage(`{"event":"purchase","cart":{"total":42}}`)}
	if got := entryEvent(entry); got != "purchase" {
		t.Errorf("entryEvent = %q, want purchase", got)
	}
	v, ok := entryValue(entry)
	if !ok {
		t.Fatal("entryValue not ok")
	}
	if got, _ := resolvePath(v, "cart.total"); got != float64(42) {
		t.Errorf("cart.total = %v, want 42", got)
	}

	if _, ok := entryValue(domain.DataLayerEntry{Name: "empty"}); ok {
		t.Error("entryValue of empty should not be ok")
	}
	if got := entryEvent(domain.DataLayerEntry{Name: "bad", Value: json.RawMessage(`[1,2]`)}); got != "" {
		t.Errorf("entryEvent of array = %q, want empty", got)
	}
}
