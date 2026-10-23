package tagpolicy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// MaxPolicyBytes caps the size of a policy document accepted by Parse/LoadFile.
const MaxPolicyBytes = 1 << 20 // 1 MiB

// LoadFile reads and parses a policy document from disk. The format is detected
// from the content: JSON documents (leading '{') are decoded strictly as JSON,
// everything else is decoded as YAML (a superset of JSON). Unknown fields are
// rejected in both cases so typos surface immediately.
func LoadFile(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tagpolicy: read policy: %w", err)
	}
	if len(data) > MaxPolicyBytes {
		return nil, fmt.Errorf("tagpolicy: policy %s exceeds %d bytes", path, MaxPolicyBytes)
	}
	return Parse(data)
}

// Parse decodes a policy from YAML or JSON and prepares it (defaults applied,
// regular expressions compiled). YAML is the recommended authoring format but
// JSON is accepted for machine-generated policies.
func Parse(data []byte) (*Policy, error) {
	if len(data) > MaxPolicyBytes {
		return nil, fmt.Errorf("tagpolicy: policy exceeds %d bytes", MaxPolicyBytes)
	}
	p := &Policy{}
	if isJSON(data) {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(p); err != nil {
			return nil, fmt.Errorf("tagpolicy: parse json: %w", err)
		}
		if err := ensureEOF(dec); err != nil {
			return nil, err
		}
	} else {
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(p); err != nil {
			return nil, fmt.Errorf("tagpolicy: parse yaml: %w", err)
		}
	}
	if err := p.Prepare(); err != nil {
		return nil, err
	}
	return p, nil
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

// ensureEOF rejects trailing content after the first JSON value.
func ensureEOF(dec *json.Decoder) error {
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return errors.New("tagpolicy: unexpected trailing content after policy")
		}
		return fmt.Errorf("tagpolicy: trailing content: %w", err)
	}
	return nil
}
