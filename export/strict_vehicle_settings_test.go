package export

import (
	"encoding/json"
	"testing"
)

func TestStrictVehicleSettings(t *testing.T) {
	fixture := []byte(`{"extras":{"new_name":{"#text":"x","@index":"0","sounds":{"critical":null}}},"materials":{"new":{"chanceToHitByExplosion":["0.4","0.5"]}},"balance":{"modulesWeightMultipliers":{"chassis":{"#items":[],"#text":"0"}}}}`)
	var v StrictVehicleSettings
	if err := json.Unmarshal(fixture, &v); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	assertEquivalentJSON(t, fixture, out)
	for _, in := range []string{`null`, `{"extras":{"x":{"newField":1}}}`, `{"balance":{"modulesWeightMultipliers":{"chassis":{"#items":[{}]}}}}`} {
		if err := json.Unmarshal([]byte(in), &v); err == nil {
			t.Errorf("accepted %s", in)
		}
	}
}
func TestStrictVehicleSettingsFullExport(t *testing.T) {
	data := fullExportData(t)
	var snapshot struct {
		Settings map[string]json.RawMessage `json:"settings"`
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	raw, ok := snapshot.Settings["vehicle"]
	if !ok {
		t.Fatal("settings.vehicle absent")
	}
	var v StrictVehicleSettings
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	assertEquivalentJSON(t, raw, out)
}
