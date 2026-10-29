package export

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StrictEquipmentDevices holds dynamic device names and the XML document's
// separate nextAvailableId scalar, without flattening either into raw JSON.
type StrictEquipmentDevices struct {
	NextAvailableID Optional[string]
	Items           map[string]Optional[StrictEquipmentDevice]
}

func (d *StrictEquipmentDevices) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("equipment devices must be an object")
	}
	var fields map[string]json.RawMessage
	if err := decodeStrictJSON(data, &fields); err != nil {
		return err
	}
	result := StrictEquipmentDevices{Items: make(map[string]Optional[StrictEquipmentDevice])}
	for key, raw := range fields {
		if key == "nextAvailableId" {
			if err := decodeStrictJSON(raw, &result.NextAvailableID); err != nil {
				return fmt.Errorf("nextAvailableId: %w", err)
			}
		} else {
			var device Optional[StrictEquipmentDevice]
			if err := decodeStrictJSON(raw, &device); err != nil {
				return fmt.Errorf("device %q: %w", key, err)
			}
			result.Items[key] = device
		}
	}
	*d = result
	return nil
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

func (v *StrictEquipmentPreset) UnmarshalJSON(data []byte) error {
	type plain StrictEquipmentPreset
	return decodeNonNullObject(data, (*plain)(v))
}
