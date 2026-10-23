package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/pageguard/pageguard/pkg/pluginsdk"
)

// NodeValidator runs a TypeScript/JavaScript validator out-of-process.
//
// Contract (one-shot per invocation): the host writes a single JSON request to
// stdin and the plugin writes a single JSON response to stdout.
//
//	{"type":"validate","protocolVersion":1,"observation":{...},
//	 "scope":{...},"rules":[...]}
//	{"ok":true,"findings":[{...}]}
type NodeValidator struct {
	Manifest pluginsdk.Manifest
	Dir      string
	// NodeBin overrides the interpreter; defaults to "node".
	NodeBin string
}

// ID returns the plugin ID.
func (v *NodeValidator) ID() string { return v.Manifest.ID }

type nodeRequest struct {
	Type            string                 `json:"type"`
	ProtocolVersion int                    `json:"protocolVersion"`
	Observation     *pluginsdk.Observation `json:"observation"`
	Scope           pluginsdk.Scope        `json:"scope"`
	Rules           []pluginsdk.RuleSpec   `json:"rules,omitempty"`
}

type nodeResponse struct {
	OK       bool                `json:"ok"`
	Error    string              `json:"error,omitempty"`
	Findings []pluginsdk.Finding `json:"findings"`
}

// Validate executes the plugin for one observation.
func (v *NodeValidator) Validate(ctx context.Context, obs *pluginsdk.Observation, scope pluginsdk.Scope) ([]pluginsdk.Finding, error) {
	bin := v.NodeBin
	if bin == "" {
		bin = "node"
	}
	req := nodeRequest{
		Type:            "validate",
		ProtocolVersion: pluginsdk.ProtocolVersion,
		Observation:     obs,
		Scope:           scope,
		Rules:           v.Manifest.Rules,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	entry := filepath.Join(v.Dir, v.Manifest.Entry)
	cmd := exec.CommandContext(ctx, bin, entry)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("plugin %s: run failed: %w: %s", v.Manifest.ID, err, stderr.String())
	}
	var resp nodeResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("plugin %s: invalid response: %w", v.Manifest.ID, err)
	}
	if !resp.OK {
		return nil, fmt.Errorf("plugin %s: %s", v.Manifest.ID, resp.Error)
	}
	return resp.Findings, nil
}
