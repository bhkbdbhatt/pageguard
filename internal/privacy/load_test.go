package privacy

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pageguard/pageguard/internal/domain"
)

func TestParseYAML(t *testing.T) {
	src := `
version: 1
name: "test"
regulations: [gdpr, ccpa]
cookie_rules:
  - name_pattern: "^_ga"
    category: analytics
settings:
  opt_out_url_patterns: ["(?i)do-not-sell"]
  pass_threshold: 80
`
	c, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if c.Name != "test" {
		t.Errorf("name = %q", c.Name)
	}
	if !reflect.DeepEqual(c.Regulations, []Regulation{GDPR, CCPA}) {
		t.Errorf("regulations = %v", c.Regulations)
	}
	if len(c.CookieRules) != 1 || c.CookieRules[0].Category != domain.TagAnalytics {
		t.Errorf("cookie rules = %+v", c.CookieRules)
	}
	if c.Settings.PassThreshold != 80 {
		t.Errorf("threshold = %d", c.Settings.PassThreshold)
	}
	if len(c.optOutRe) != 1 {
		t.Errorf("opt-out regexes = %d", len(c.optOutRe))
	}
}

func TestParseJSON(t *testing.T) {
	src := `{
		"name": "test",
		"regulations": ["eprivacy"],
		"settings": { "consent_states": ["accept_all"] }
	}`
	c, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(c.Regulations) != 1 || c.Regulations[0] != EPrivacy {
		t.Errorf("regulations = %v", c.Regulations)
	}
	// defaults applied
	if c.Version != 1 {
		t.Errorf("version = %d, want 1", c.Version)
	}
}

func TestParseUnknownField(t *testing.T) {
	if _, err := Parse([]byte(`name: x` + "\n" + `bogus: true`)); err == nil {
		t.Fatal("expected error for unknown YAML field")
	}
	if _, err := Parse([]byte(`{"name":"x","bogus":true}`)); err == nil {
		t.Fatal("expected error for unknown JSON field")
	}
}

func TestParseMissingName(t *testing.T) {
	if _, err := Parse([]byte(`version: 1`)); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseInvalidRegex(t *testing.T) {
	if _, err := Parse([]byte("name: x\ncookie_rules:\n  - name_pattern: \"(\"")); err == nil {
		t.Fatal("expected error for invalid cookie rule regex")
	}
}

func TestParseTooLarge(t *testing.T) {
	big := make([]byte, MaxPolicyBytes+1)
	for i := range big {
		big[i] = ' '
	}
	if _, err := Parse(big); err == nil {
		t.Fatal("expected error for oversized document")
	}
}

func TestLoadFileExamplesEquivalent(t *testing.T) {
	yamlPath := filepath.Join("..", "..", "examples", "privacy", "acme.yaml")
	jsonPath := filepath.Join("..", "..", "examples", "privacy", "acme.json")

	y, err := LoadFile(yamlPath)
	if err != nil {
		t.Fatalf("load yaml: %v", err)
	}
	j, err := LoadFile(jsonPath)
	if err != nil {
		t.Fatalf("load json: %v", err)
	}
	yb, _ := json.Marshal(y)
	jb, _ := json.Marshal(j)
	if !bytes.Equal(yb, jb) {
		t.Errorf("example yaml and json differ:\nyaml=%s\njson=%s", yb, jb)
	}
}

func TestLoadFileMissing(t *testing.T) {
	if _, err := LoadFile(filepath.Join("testdata", "nope.yaml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
