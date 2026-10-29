package export

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StrictShellDocument retains named shells and the document's special icons
// dictionary. Names are source identifiers and remain dynamic.
type StrictShellDocument struct {
	Icons Optional[map[string]Optional[string]]
	Items map[string]Optional[StrictShell]
}

func (d *StrictShellDocument) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("shell document must be an object")
	}
	var fields map[string]json.RawMessage
	if err := decodeStrictJSON(data, &fields); err != nil {
		return err
	}
	result := StrictShellDocument{Items: make(map[string]Optional[StrictShell])}
	for key, raw := range fields {
		if key == "icons" {
			if err := decodeStrictJSON(raw, &result.Icons); err != nil {
				return fmt.Errorf("shell icons: %w", err)
			}
		} else {
			var item Optional[StrictShell]
			if err := decodeStrictJSON(raw, &item); err != nil {
				return fmt.Errorf("shell %q: %w", key, err)
			}
			result.Items[key] = item
		}
	}
	*d = result
	return nil
}
func (d StrictShellDocument) MarshalJSON() ([]byte, error) {
	result := make(map[string]json.RawMessage, len(d.Items)+1)
	for key, item := range d.Items {
		if key == "icons" || !item.Present() {
			return nil, fmt.Errorf("invalid shell key or absent shell value %q", key)
		}
		b, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		result[key] = b
	}
	if d.Icons.Present() {
		b, err := json.Marshal(d.Icons)
		if err != nil {
			return nil, err
		}
		result["icons"] = b
	}
	return json.Marshal(result)
}
