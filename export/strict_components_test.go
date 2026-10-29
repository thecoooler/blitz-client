package export

import (
	"encoding/json"
	"testing"
)

func TestStrictComponentsRoundTrip(t *testing.T) {
	raw := []byte(`{"guns":{"ids":{"foo":"1"},"nextAvailableId":"2","shared":{"foo":{"shots":[{"name":"b","properties":{"speed":"2"}},{"name":"a","properties":{"speed":"1"}}],"tags":[null,"two"],"price":null}}},"shells":{"icons":{"ap":"x.png"},"a":{"price":{"#text":"2","gold":null}},"b":{"price":"2"}},"turrets":{"shared":null}}`)
	var data StrictNationComponents
	if err := decodeStrictJSON(raw, &data); err != nil {
		t.Fatal(err)
	}
	guns, ok := data.Guns.Get()
	if !ok {
		t.Fatal("missing guns")
	}
	shared, ok := guns.Shared.Get()
	if !ok {
		t.Fatal("missing shared")
	}
	gun, ok := shared["foo"].Get()
	if !ok {
		t.Fatal("missing gun")
	}
	shots, ok := gun.Shots.Get()
	if !ok || len(shots) != 2 {
		t.Fatal("missing shots")
	}
	first, ok := shots[0].Get()
	if !ok {
		t.Fatal("null shot")
	}
	name, _ := first.Name.Get()
	if name != "b" {
		t.Fatal("selection order was not preserved")
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, raw, encoded)
}

func TestStrictComponentsRejectInvalid(t *testing.T) {
	for _, raw := range []string{
		`{"guns":{"shared":{"foo":{"unknown":"1"}}}}`,
		`{"guns":{"shared":{"foo":{"Shots":[]}}}}`,
		`{"guns":{"shared":{"foo":{"tags":{}}}}}`,
		`{"shells":{"foo":{"price":{"bogus":1}}}}`,
		`{"shells":{"foo":"wrong"}}`,
	} {
		var data StrictNationComponents
		if err := decodeStrictJSON([]byte(raw), &data); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestStrictComponentsFullExport(t *testing.T) {
	forEachFullExport(t, func(t *testing.T, raw []byte) {
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		original, ok := doc["components"]
		if !ok {
			t.Fatal("missing components")
		}
		var data map[string]Optional[StrictNationComponents]
		if err := decodeStrictJSON(original, &data); err != nil {
			t.Fatal(err)
		}
		if len(data) != 9 {
			t.Fatalf("expected 9 nations, got %d", len(data))
		}
		encoded, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, original, encoded)
	})
}
