package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pageguard/pageguard/pkg/pluginsdk"
)

// LoadDir scans dir for plugin subdirectories, each containing a manifest.json,
// and registers validators in the registry. Reporter/integration manifests are
// returned but not yet wired (Modules 3 and 5 own their execution).
//
// A missing directory is not an error.
func LoadDir(reg *Registry, dir, nodeBin string) ([]pluginsdk.Manifest, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var loaded []pluginsdk.Manifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(dir, e.Name(), "manifest.json")
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var m pluginsdk.Manifest
		if err := json.Unmarshal(raw, &m); err != nil {
			return loaded, fmt.Errorf("plugin %s: invalid manifest: %w", e.Name(), err)
		}
		if m.ID == "" {
			m.ID = e.Name()
		}
		pluginDir := filepath.Join(dir, e.Name())

		switch {
		case m.Kind == pluginsdk.KindValidator && m.Runtime == "node":
			reg.Register(&NodeValidator{Manifest: m, Dir: pluginDir, NodeBin: nodeBin})
		case m.Kind == pluginsdk.KindValidator && len(m.Rules) > 0:
			reg.Register(&DeclarativeValidator{PluginID: m.ID, Rules: m.Rules})
		default:
			// Validator without rules/runtime, or a non-validator kind: skip.
			continue
		}
		loaded = append(loaded, m)
	}
	return loaded, nil
}
