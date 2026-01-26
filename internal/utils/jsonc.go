package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/matthewmueller/jsonc"
)

// JsonC implements a JsonC parser.
type JsonC struct{}

// ConfigParser returns a HuJSON ConfigParser.
func ConfigParser() *JsonC {
	return &JsonC{}
}

// Unmarshal parses the given JSON bytes.
func (p *JsonC) Unmarshal(b []byte) (map[string]interface{}, error) {
	jsonBytes, err := jsonc.Standardize(b)
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
func (p *JsonC) Marshal(o map[string]interface{}) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

type JsonPatchOperation struct {
	Op    string      `json:"op"`
	From  string      `json:"from,omitempty"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type JsonPatch []JsonPatchOperation

func PatchFile(fp string, patch JsonPatch) error {
	b, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", fp, err)
	}

	parsed, err := jsonc.Parse(b)
	if err != nil {
		return fmt.Errorf("parsing HuJSON file %s: %w", fp, err)
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshaling JSON patch for file %s: %w", fp, err)
	}

	if err := parsed.Patch(patchBytes); err != nil {
		return fmt.Errorf("applying JSON patch to file %s: %w", fp, err)
	}

	parsed.Format()
	packed := parsed.Pack()

	if err := os.WriteFile(fp, packed, 0o644); err != nil {
		return fmt.Errorf("writing patched HuJSON file %s: %w", fp, err)
	}

	return nil
}
