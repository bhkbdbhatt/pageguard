package journey

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const minimalYAML = `
version: 1
name: "smoke"
steps:
  - action: navigate
    value: "https://acme.test/"
`

const minimalJSON = `{
  "version": 1,
  "name": "smoke",
  "steps": [{"action": "navigate", "value": "https://acme.test/"}]
}`

func TestParseYAMLMinimal(t *testing.T) {
	j, err := Parse([]byte(minimalYAML))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if j.Name != "smoke" || len(j.Steps) != 1 || j.Steps[0].Action != ActionNavigate {
		t.Fatalf("unexpected journey: %+v", j)
	}
	if j.Defaults.TimeoutMs != DefaultTimeoutMs {
		t.Errorf("default timeout = %d, want %d", j.Defaults.TimeoutMs, DefaultTimeoutMs)
	}
	if j.Steps[0].TimeoutMs != DefaultTimeoutMs {
		t.Errorf("step timeout not inherited: %d", j.Steps[0].TimeoutMs)
	}
}

func TestParseJSONMinimal(t *testing.T) {
	j, err := Parse([]byte(minimalJSON))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if j.Name != "smoke" || j.Version != 1 {
		t.Fatalf("unexpected journey: %+v", j)
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	for name, doc := range map[string]string{
		"yaml": minimalYAML + "\nbogus: true\n",
		"json": `{"version":1,"name":"x","steps":[{"action":"wait","selector":"#a"}],"bogus":1}`,
	} {
		if _, err := Parse([]byte(doc)); err == nil {
			t.Errorf("%s: expected unknown-field error", name)
		}
	}
}

func TestParseRejectsOversize(t *testing.T) {
	big := strings.Repeat("a", MaxJourneyBytes+1)
	if _, err := Parse([]byte(big)); err == nil {
		t.Fatal("expected size error")
	}
}

func TestLoadFile(t *testing.T) {
	j, err := LoadFile(filepath.Join("..", "..", "examples", "journey", "checkout.yaml"))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(j.Steps) != 7 {
		t.Fatalf("steps = %d, want 7", len(j.Steps))
	}
	if j.Steps[0].Value != "https://acme.com/" {
		t.Errorf("relative navigate not resolved: %q", j.Steps[0].Value)
	}
}

func TestLoadFileMissing(t *testing.T) {
	if _, err := LoadFile("does-not-exist.yaml"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestExampleJourneyEquivalent proves the YAML and JSON example fixtures parse
// to the same journey.
func TestExampleJourneyEquivalent(t *testing.T) {
	yamlPath := filepath.Join("..", "..", "examples", "journey", "checkout.yaml")
	jsonPath := filepath.Join("..", "..", "examples", "journey", "checkout.json")
	if _, err := os.Stat(yamlPath); err != nil {
		t.Skipf("example missing: %v", err)
	}
	fromYAML, err := LoadFile(yamlPath)
	if err != nil {
		t.Fatalf("load yaml: %v", err)
	}
	fromJSON, err := LoadFile(jsonPath)
	if err != nil {
		t.Fatalf("load json: %v", err)
	}
	if !reflect.DeepEqual(fromYAML, fromJSON) {
		t.Fatalf("yaml and json fixtures differ:\nyaml=%+v\njson=%+v", fromYAML, fromJSON)
	}
}
