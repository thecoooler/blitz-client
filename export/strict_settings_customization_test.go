package export

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestStrictCustomizationSettingsFullExport(t *testing.T) {
	data := fullExportData(t)
	var root struct {
		Settings map[string]json.RawMessage `json:"settings"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		target any
	}{{"camouflages", new(SettingCamouflages)}, {"player_emblems", new(SettingPlayerEmblems)}, {"roles", new(SettingRoles)}} {
		t.Run(test.name, func(t *testing.T) {
			raw := root.Settings[test.name]
			if len(raw) == 0 {
				t.Fatal("missing document")
			}
			if err := decodeStrictJSON(raw, test.target); err != nil {
				t.Fatal(err)
			}
			out, err := json.Marshal(test.target)
			if err != nil {
				t.Fatal(err)
			}
			var a, b any
			d := json.NewDecoder(bytes.NewReader(raw))
			d.UseNumber()
			if err = d.Decode(&a); err != nil {
				t.Fatal(err)
			}
			d = json.NewDecoder(bytes.NewReader(out))
			d.UseNumber()
			if err = d.Decode(&b); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(a, b) {
				t.Fatal("round trip differs")
			}
		})
	}
}
