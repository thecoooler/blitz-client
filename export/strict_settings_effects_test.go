package export

import (
	"encoding/json"
	"testing"
)

func TestStrictEffectsFullExport(t *testing.T) {
	raw := fullExportData(t)
	var doc struct {
		Settings map[string]json.RawMessage `json:"settings"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		target any
	}{
		{"chassis_effects", new(StrictChassisEffects)},
		{"gun_effects", new(map[string]Optional[StrictGunEffect])},
		{"shot_effects", new(map[string]Optional[StrictShotEffect])},
		{"vehicle_effects", new(map[string]Optional[StrictVehicleEffect])},
	} {
		t.Run(test.name, func(t *testing.T) {
			original, ok := doc.Settings[test.name]
			if !ok {
				t.Fatal("missing document")
			}
			if err := decodeStrictJSON(original, test.target); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(test.target)
			if err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, original, encoded)
		})
	}
}

func TestStrictEffectRejectsUnknown(t *testing.T) {
	var effect StrictGunEffect
	if err := decodeStrictJSON([]byte(`{"blitz":{"unknown":"x"}}`), &effect); err == nil {
		t.Fatal("unknown field accepted")
	}
}
