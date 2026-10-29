package export

import (
	"encoding/json"
	"testing"
)

func TestStrictTankConfigurationRoundTrip(t *testing.T) {
	input := []byte(`{"crew":{"loader":[null,"radio"]},"hull":{"armor":{"armor_1":{"#text":"10","vehicleDamageFactor":"0.5"},"newPart":"45"}},"engines":[{"name":"engineB","properties":"shared"},{"name":"engineA","properties":{"#text":"shared","unlocks":{"engine":{"#text":"engineB","cost":"0"}}}}],"turrets0":[{"name":"t","properties":{"guns":[{"name":"gunB","properties":{"#text":"shared","shots":[{"name":"shotB","properties":{"shell":{"#text":"shared","price":{"#text":"10","gold":null}}}}]}}]}}]}`)
	var value StrictTankConfiguration
	if err := json.Unmarshal(input, &value); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	assertEquivalentJSON(t, input, out)
	// One installed XML places a name-keyed shell directly under a gun. Its
	// component identifier is data rather than a fixed struct field.
	input = []byte(`{"turrets0":[{"name":"t","properties":{"guns":[{"name":"g","properties":{"_different_shell":{"shell":{"#text":"shared","price":"0"}}}}]}}]}`)
	if err := json.Unmarshal(input, &value); err != nil {
		t.Fatal(err)
	}
	out, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	assertEquivalentJSON(t, input, out)
}
func TestStrictTankConfigurationRejectInvalid(t *testing.T) {
	for _, input := range []string{`null`, `{"crew":{"loader":123}}`, `{"hull":{"unknown":"bad"}}`, `{"engines":[{"name":1}]}`, `{"engines":[{"name":"missing-properties"}]}`, `{"engines":[{"name":null,"properties":null}]}`, `{"turrets0":[{"name":"t","properties":{"guns":{}}}]}`} {
		var value StrictTankConfiguration
		if err := json.Unmarshal([]byte(input), &value); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}
func TestStrictTankConfigurationFullExport(t *testing.T) {
	forEachFullExport(t, func(t *testing.T, data []byte) {
		var snapshot struct {
			Tanks []struct {
				Configuration json.RawMessage `json:"configuration"`
				ID            string          `json:"id"`
			} `json:"tanks"`
		}
		if err := json.Unmarshal(data, &snapshot); err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Tanks) == 0 {
			t.Fatal("no tanks")
		}
		for _, tank := range snapshot.Tanks {
			var value StrictTankConfiguration
			if err := json.Unmarshal(tank.Configuration, &value); err != nil {
				t.Fatalf("%s: %v", tank.ID, err)
			}
			out, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("%s: %v", tank.ID, err)
			}
			assertEquivalentJSON(t, tank.Configuration, out)
		}
	})
}
