package utils

import (
	"bytes"
	"encoding/json"

	"github.com/tailscale/hujson"
)

// HuJSON implements a HuJSON parser.
type HuJSON struct{}

// ConfigParser returns a HuJSON ConfigParser.
func ConfigParser() *HuJSON {
	return &HuJSON{}
}

// Unmarshal parses the given JSON bytes.
func (p *HuJSON) Unmarshal(b []byte) (map[string]interface{}, error) {
	jsonBytes, err := hujson.Standardize(b)
	if err != nil {
		return nil, err
	}

	var out map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Marshal marshals the given config map to JSON bytes.
func (p *HuJSON) Marshal(o map[string]interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	encoder.Encode(o)
	return buf.Bytes(), nil
}
