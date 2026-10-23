package privacy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFile reads and parses a privacy compliance policy. JSON documents
// (leading '{') are decoded strictly as JSON; everything else is decoded as
// YAML. Unknown fields are rejected in both cases.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("privacy: read: %w", err)
	}
	if len(data) > MaxPolicyBytes {
		return nil, fmt.Errorf("privacy: %s exceeds %d bytes", path, MaxPolicyBytes)
	}
	return Parse(data)
}

// Parse decodes a policy from YAML or JSON and prepares it (defaults applied,
// regular expressions compiled, config validated).
func Parse(data []byte) (*Config, error) {
	if len(data) > MaxPolicyBytes {
		return nil, fmt.Errorf("privacy: document exceeds %d bytes", MaxPolicyBytes)
	}
	c := &Config{}
	if isJSON(data) {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(c); err != nil {
			return nil, fmt.Errorf("privacy: parse json: %w", err)
		}
		if _, err := dec.Token(); err != io.EOF {
			if err == nil {
				return nil, fmt.Errorf("privacy: unexpected trailing content after document")
			}
			return nil, fmt.Errorf("privacy: trailing content: %w", err)
		}
	} else {
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(c); err != nil {
			return nil, fmt.Errorf("privacy: parse yaml: %w", err)
		}
	}
	if err := c.Prepare(); err != nil {
		return nil, err
	}
	return c, nil
}

func isJSON(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}
