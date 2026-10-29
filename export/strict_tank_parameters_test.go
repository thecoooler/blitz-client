package export

import (
	"encoding/json"
	"testing"
)

func TestStrictTankParametersRoundTrip(t *testing.T) {
	input := []byte(`{"resourcesPath":{"bigIconPath":"x","customEffects":{"hull":{"arbitrary":null}}},"collision":{"gun_1":{"points":[9007199254740993,0.000000000000000001]}},"gunMarks":{"gradeOnePath":"a","gun_01":{"skin":{"scale":[1,2]}}},"detachedTurret":{"cylinderPos":[0,1,2],"turret_23":{"shouldClipGun":false}},"mechanics":null}`)
	var value StrictTankParameters
	if err := json.Unmarshal(input, &value); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	assertEquivalentJSON(t, input, out)
}
func TestStrictTankParametersRejectInvalid(t *testing.T) {
	for _, input := range []string{`null`, `{"resourcesPath":{"BIGICONPATH":"a"}}`, `{"resourcesPath":{"bigIconPath":1}}`, `{"collision":{"gun":{"bbox":{"min":["1"]}}}}`, `{"detachedTurret":{"unknown":1}}`, `{"gunMarks":{"gun_1":42}}`} {
		var value StrictTankParameters
		if err := json.Unmarshal([]byte(input), &value); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}
func TestStrictTankParametersFullExport(t *testing.T) {
	forEachFullExport(t, func(t *testing.T, data []byte) {
		var snapshot struct {
			Tanks []struct {
				Parameters json.RawMessage `json:"parameters"`
				ID         string          `json:"id"`
			} `json:"tanks"`
		}
		if err := json.Unmarshal(data, &snapshot); err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Tanks) == 0 {
			t.Fatal("no tanks")
		}
		for _, tank := range snapshot.Tanks {
			var value StrictTankParameters
			if err := json.Unmarshal(tank.Parameters, &value); err != nil {
				t.Fatalf("%s: %v", tank.ID, err)
			}
			out, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("%s: %v", tank.ID, err)
			}
			assertEquivalentJSON(t, tank.Parameters, out)
		}
	})
}
