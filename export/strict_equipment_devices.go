package export

import (
	"encoding/json"
	"fmt"
)

// StrictEquipmentDevices holds dynamic device names and the XML document's
// separate nextAvailableId scalar, without flattening either into raw JSON.
type StrictEquipmentDevices struct {
	NextAvailableID Optional[XMLInteger]                       `json:"nextAvailableId,omitzero"`
	Items           map[string]Optional[StrictEquipmentDevice] `json:"-"`
}

func (d StrictEquipmentDevices) MarshalJSON() ([]byte, error) {
	result := make(map[string]json.RawMessage, len(d.Items)+1)
	for key, item := range d.Items {
		if key == "nextAvailableId" || !item.Present() {
			return nil, fmt.Errorf("invalid device key or absent device value %q", key)
		}
		b, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		result[key] = b
	}
	if d.NextAvailableID.Present() {
		b, err := json.Marshal(d.NextAvailableID)
		if err != nil {
			return nil, err
		}
		result["nextAvailableId"] = b
	}
	return json.Marshal(result)
}
