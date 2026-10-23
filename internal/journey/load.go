package journey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// MaxJourneyBytes caps the size of a journey document accepted by Parse/LoadFile.
const MaxJourneyBytes = 1 << 20 // 1 MiB

// LoadFile reads and parses a journey from disk. JSON documents (leading '{')
// are decoded strictly as JSON; everything else is decoded as YAML (a superset
// of JSON). Unknown fields are rejected in both cases.
func LoadFile(path string) (*Journey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("journey: read: %w", err)
	}
	if len(data) > MaxJourneyBytes {
		return nil, fmt.Errorf("journey: %s exceeds %d bytes", path, MaxJourneyBytes)
	}
	return Parse(data)
}

// Parse decodes a journey from YAML or JSON and prepares it (defaults applied,
// steps validated).
func Parse(data []byte) (*Journey, error) {
	if len(data) > MaxJourneyBytes {
		return nil, fmt.Errorf("journey: document exceeds %d bytes", MaxJourneyBytes)
	}
	j := &Journey{}
	if isJSON(data) {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(j); err != nil {
			return nil, fmt.Errorf("journey: parse json: %w", err)
		}
		if err := ensureEOF(dec); err != nil {
			return nil, err
		}
	} else {
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(j); err != nil {
			return nil, fmt.Errorf("journey: parse yaml: %w", err)
		}
	}
	if err := j.Prepare(); err != nil {
		return nil, err
	}
	return j, nil
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

func ensureEOF(dec *json.Decoder) error {
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("journey: unexpected trailing content after document")
		}
		return fmt.Errorf("journey: trailing content: %w", err)
	}
	return nil
}
