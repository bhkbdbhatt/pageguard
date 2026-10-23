package tagpolicy

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExamplesEquivalent(t *testing.T) {
	yamlPath := filepath.Join("..", "..", "examples", "tagpolicy", "acme.yaml")
	jsonPath := filepath.Join("..", "..", "examples", "tagpolicy", "acme.json")

	fromYAML, err := LoadFile(yamlPath)
	if err != nil {
		t.Fatalf("load yaml: %v", err)
	}
	fromJSON, err := LoadFile(jsonPath)
	if err != nil {
		t.Fatalf("load json: %v", err)
	}

	yb, err := json.Marshal(fromYAML)
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}
	jb, err := json.Marshal(fromJSON)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	if !bytes.Equal(yb, jb) {
		t.Fatalf("YAML and JSON policies differ:\n yaml: %s\n json: %s", yb, jb)
	}

	if len(fromYAML.MustExist) != 5 || len(fromYAML.MustNotExist) != 2 || len(fromYAML.ValueRules) != 4 {
		t.Fatalf("unexpected example shape: %d/%d/%d", len(fromYAML.MustExist), len(fromYAML.MustNotExist), len(fromYAML.ValueRules))
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	if _, err := Parse([]byte("version: 1\nname: p\nbogus: 1\n")); err == nil {
		t.Error("yaml: expected error for unknown field")
	}
	if _, err := Parse([]byte(`{"version":1,"name":"p","bogus":1}`)); err == nil {
		t.Error("json: expected error for unknown field")
	}
}

func TestParseRejectsInvalidPolicy(t *testing.T) {
	cases := map[string]string{
		"missing name":  "version: 1\n",
		"invalid regex": "version: 1\nname: p\nmust_exist:\n  - id: a\n    target: tag\n    match:\n      tag_pattern: \"(\"\n",
		"bad target":    "version: 1\nname: p\nmust_exist:\n  - id: a\n    target: widget\n    match:\n      tag: x\n",
		"duplicate id":  "version: 1\nname: p\nmust_exist:\n  - id: a\n    target: tag\n    match:\n      tag: x\nmust_not_exist:\n  - id: a\n    target: tag\n    match:\n      tag: y\n",
		"bad severity":  "version: 1\nname: p\nsettings:\n  unexpected_tags: nope\n",
	}
	for name, doc := range cases {
		if _, err := Parse([]byte(doc)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestParseAppliesDefaults(t *testing.T) {
	p, err := Parse([]byte("version: 1\nname: p\nmust_exist:\n  - id: a\n    target: tag\n    match:\n      tag: x\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Settings.MaxOccurrences != 1 {
		t.Errorf("max_occurrences = %d, want 1", p.Settings.MaxOccurrences)
	}
	if p.Settings.UnexpectedTags != SevInfo {
		t.Errorf("unexpected_tags = %q, want info", p.Settings.UnexpectedTags)
	}
	if p.Settings.DuplicateTags != SevMedium {
		t.Errorf("duplicate_tags = %q, want medium", p.Settings.DuplicateTags)
	}
	if p.MustExist[0].Severity != SevMedium {
		t.Errorf("default rule severity = %q, want medium", p.MustExist[0].Severity)
	}
	if p.ConsentRequirements.Severity != SevMedium {
		t.Errorf("default consent severity = %q, want medium", p.ConsentRequirements.Severity)
	}
}

func TestParseRejectsTrailingJSON(t *testing.T) {
	if _, err := Parse([]byte(`{"version":1,"name":"p"} {"version":1}`)); err == nil {
		t.Error("expected error for trailing JSON content")
	}
}

func TestParseRejectsOversize(t *testing.T) {
	big := bytes.Repeat([]byte("a"), MaxPolicyBytes+1)
	if _, err := Parse(big); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected oversize error, got %v", err)
	}
}

func TestLoadFileMissing(t *testing.T) {
	if _, err := LoadFile(filepath.Join(os.TempDir(), "does-not-exist-tagpolicy.yaml")); err == nil {
		t.Error("expected error for missing file")
	}
}
