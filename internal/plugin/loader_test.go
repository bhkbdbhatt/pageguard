package plugin

import (
	"context"
	"os/exec"
	"testing"

	"github.com/pageguard/pageguard/pkg/pluginsdk"
)

func TestLoadDirRegistersExamplePlugins(t *testing.T) {
	reg := NewRegistry()
	manifests, err := LoadDir(reg, "../../plugins", "node")
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(manifests) != 2 {
		t.Fatalf("loaded %d manifests, want 2: %+v", len(manifests), manifests)
	}
	if _, ok := reg.Get("example-tag-rules"); !ok {
		t.Error("declarative plugin not registered")
	}
	if _, ok := reg.Get("example-node-validator"); !ok {
		t.Error("node plugin not registered")
	}
}

func TestLoadDirMissingDirIsNotAnError(t *testing.T) {
	reg := NewRegistry()
	if _, err := LoadDir(reg, "../../does-not-exist", "node"); err != nil {
		t.Fatalf("missing dir should be ignored, got %v", err)
	}
}

// TestNodeValidatorEndToEnd executes the example Node validator. It is skipped
// when Node.js is unavailable.
func TestNodeValidatorEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	reg := NewRegistry()
	if _, err := LoadDir(reg, "../../plugins", "node"); err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	v, ok := reg.Get("example-node-validator")
	if !ok {
		t.Fatal("plugin not registered")
	}

	obs := &pluginsdk.Observation{
		URL: "https://x/", FinalURL: "https://x/", StatusCode: 200,
		Requests: []pluginsdk.NetworkRequest{{URL: "http://cdn.example/a.js", Host: "cdn.example", ThirdParty: true}},
	}
	findings, err := v.Validate(context.Background(), obs, pluginsdk.Scope{RunID: "r"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "example-node.insecure-request" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected insecure-request finding, got %+v", findings)
	}
}
