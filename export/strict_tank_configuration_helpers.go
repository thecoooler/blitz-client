package export

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (g StrictTankConfigurationTurrets0EntryPropertiesGunsEntryProperties) MarshalJSON() ([]byte, error) {
	type alias StrictTankConfigurationTurrets0EntryPropertiesGunsEntryProperties
	raw, err := json.Marshal(alias(g))
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	for key, value := range g.NamedShots {
		if !strings.HasPrefix(key, "_") || !value.Present() {
			return nil, fmt.Errorf("invalid named shot %q", key)
		}
		v, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		fields[key] = v
	}
	return json.Marshal(fields)
}
