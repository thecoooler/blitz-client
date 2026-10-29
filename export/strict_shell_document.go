package export

import (
	"encoding/json"
	"fmt"
)

// StrictShellDocument retains named shells and the document's special icons
// dictionary. Names are source identifiers and remain dynamic.
type StrictShellDocument struct {
	Icons Optional[map[string]Optional[string]] `json:"icons,omitzero"`
	Items map[string]Optional[StrictShell]      `json:"-"`
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
