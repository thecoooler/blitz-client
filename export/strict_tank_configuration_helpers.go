package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// NamedShots retains name-keyed shot-like elements found directly under a gun
// in some source files. Other fields remain validated against the fixed schema.
func (g *StrictTankConfigurationTurrets0EntryPropertiesGunsEntryProperties) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := decodeStrictJSON(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("gun properties must be an object")
	}
	named := map[string]Optional[StrictTankConfigurationNamedShot]{}
	for key, value := range fields {
		if strings.HasPrefix(key, "_") {
			var shot Optional[StrictTankConfigurationNamedShot]
			if err := decodeStrictJSON(value, &shot); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
			named[key] = shot
			delete(fields, key)
		}
	}
	data, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	type alias StrictTankConfigurationTurrets0EntryPropertiesGunsEntryProperties
	var decoded alias
	if err = decodeStrictJSON(data, &decoded); err != nil {
		return err
	}
	if len(named) > 0 {
		decoded.NamedShots = named
	}
	*g = StrictTankConfigurationTurrets0EntryPropertiesGunsEntryProperties(decoded)
	return nil
}
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

func validateTankOrderedSelections(data []byte) error {
	var root json.RawMessage = data
	var walk func(json.RawMessage, string) error
	walk = func(raw json.RawMessage, path string) error {
		b := bytes.TrimSpace(raw)
		if len(b) == 0 {
			return fmt.Errorf("%s: empty value", path)
		}
		switch b[0] {
		case '{':
			var m map[string]json.RawMessage
			if err := json.Unmarshal(b, &m); err != nil {
				return err
			}
			for k, v := range m {
				ordered := k == "guns" || k == "shots" || k == "chassis" || k == "engines" || k == "fuelTanks" || k == "radios" || k == "turrets" || strings.HasPrefix(k, "turrets") && strings.Trim(strings.TrimPrefix(k, "turrets"), "0123456789") == ""
				if ordered {
					if err := validateTankOrderedSelection(v, path+"."+k); err != nil {
						return err
					}
				}
				if err := walk(v, path+"."+k); err != nil {
					return err
				}
			}
		case '[':
			var a []json.RawMessage
			if err := json.Unmarshal(b, &a); err != nil {
				return err
			}
			for i, v := range a {
				if err := walk(v, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(root, "configuration")
}
func validateTankOrderedSelection(raw json.RawMessage, path string) error {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 {
		return fmt.Errorf("%s: empty selection", path)
	}
	if b[0] == '{' {
		var v map[string]json.RawMessage
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		items, ok := v["#items"]
		if !ok {
			return fmt.Errorf("%s: ordered container missing #items", path)
		}
		b = bytes.TrimSpace(items)
	}
	if len(b) == 0 || b[0] != '[' {
		return fmt.Errorf("%s: ordered container must be an array or #items object", path)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(b, &entries); err != nil {
		return err
	}
	for i, raw := range entries {
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(raw, &entry); err != nil || entry == nil {
			return fmt.Errorf("%s[%d]: expected entry object", path, i)
		}
		name, ok := entry["name"]
		if !ok {
			return fmt.Errorf("%s[%d]: missing name", path, i)
		}
		var s string
		if err := json.Unmarshal(name, &s); err != nil || bytes.Equal(bytes.TrimSpace(name), []byte("null")) {
			return fmt.Errorf("%s[%d]: name must be a string", path, i)
		}
		if _, ok := entry["properties"]; !ok {
			return fmt.Errorf("%s[%d]: missing properties", path, i)
		}
	}
	return nil
}
