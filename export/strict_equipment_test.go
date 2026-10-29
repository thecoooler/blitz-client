package export

import (
	"encoding/json"
	"testing"
)

func TestStrictEquipmentRoundTrip(t *testing.T) {
	raw := []byte(`{"devices":{"nextAvailableId":"127","unit":{"id":"1","script":{"#text":"effect","rammingAbsorptionPercent":null}}},"slots":{"presets":{"empty":{"level0":{"attack":{"device0":null,"device1":null}}}}}}`)
	var data StrictEquipment
	if err := decodeStrictJSON(raw, &data); err != nil {
		t.Fatal(err)
	}
	devices, ok := data.Devices.Get()
	if !ok {
		t.Fatal("no devices")
	}
	id, _ := devices.NextAvailableID.Get()
	if id != "127" {
		t.Fatalf("next id: %s", id)
	}
	device, ok := devices.Items["unit"].Get()
	if !ok {
		t.Fatal("no unit")
	}
	script, ok := device.Script.Get()
	if !ok || !script.RammingAbsorptionPercent.Null() {
		t.Fatal("null not preserved")
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, raw, encoded)
}

func TestStrictEquipmentRejectInvalid(t *testing.T) {
	for _, raw := range []string{
		`{"devices":{"foo":{"script":{"bogus":1}}}}`,
		`{"devices":{"foo":{"script":{"Weight":"1"}}}}`,
		`{"devices":{"foo":"bad"}}`,
		`{"slots":{"presets":{"foo":null}}}`,
		`{"devices":{"nextAvailableId":"1","nextAvailableId":"2"}}`,
	} {
		var data StrictEquipment
		if err := decodeStrictJSON([]byte(raw), &data); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestStrictEquipmentFullExport(t *testing.T) {
	forEachFullExport(t, func(t *testing.T, raw []byte) {
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		original, ok := doc["equipment"]
		if !ok {
			t.Fatal("missing equipment")
		}
		var data StrictEquipment
		if err := decodeStrictJSON(original, &data); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, original, encoded)
	})
}
