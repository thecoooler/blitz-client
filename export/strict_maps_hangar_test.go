package export

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestStrictMapsHangarRoundTrip(t *testing.T) {
	input := `{"maps":{"maps":{"map_a":{"id":9007199254740993,"avaliableInTrainingRoom":false,"levels":null,"assaultRespawnPoints":{"allies":[{"points":[[1.2300,-4e3]],"respawnNumber":0}],"enemies":[]}}}},"hangar":{"keyedArchive":{"one":{"keyedArchive":{"digits":{"int32":2},"units":{"string":"m"}}},"empty":{"keyedArchive":{}}}}}`
	var data struct {
		Maps   StrictMaps             `json:"maps"`
		Hangar StrictHangarParameters `json:"hangar"`
	}
	if err := decodeStrictJSON([]byte(input), &data); err != nil {
		t.Fatal(err)
	}
	maps, ok := data.Maps.Maps.Get()
	if !ok {
		t.Fatal("missing maps")
	}
	record := maps["map_a"]
	if !record.Levels.Null() || record.Tags.Present() {
		t.Fatal("null and absent fields were not distinguished")
	}
	id, _ := record.ID.Get()
	if id != "9007199254740993" {
		t.Fatalf("ID = %s", id)
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, []byte(input), encoded)
	if !bytes.Contains(encoded, []byte("1.2300")) || !bytes.Contains(encoded, []byte("-4e3")) {
		t.Fatalf("number spelling lost: %s", encoded)
	}
}

func TestStrictMapsRejectWrongShapes(t *testing.T) {
	for _, input := range []string{
		`{"maps":{"a":{"id":"1"}}}`,
		`{"maps":{"a":null}}`,
		`{"maps":{"a":{"newField":1}}}`,
		`{"maps":{"a":{"assaultRespawnPoints":{"allies":[{"points":[[1]]}]}}}}`,
		`{"maps":{"a":{"assaultRespawnPoints":{"allies":[null]}}}}`,
	} {
		var maps StrictMaps
		if err := decodeStrictJSON([]byte(input), &maps); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
	var h StrictHangarParameters
	if err := decodeStrictJSON([]byte(`{"keyedArchive":{"a":{"keyedArchive":{"digits":{"new":1}}}}}`), &h); err == nil {
		t.Fatal("accepted unknown hangar field")
	}
}

// Set BLITZ_EXPORT_DATA to an actual --include data export to test all installed
// map and hangar records without committing game assets to the repository.
func TestStrictMapsHangarFullExport(t *testing.T) {
	raw := fullExportData(t)
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"maps", "hangarParameters"} {
		t.Run(name, func(t *testing.T) {
			original, ok := envelope[name]
			if !ok {
				t.Fatalf("missing %s", name)
			}
			var target any
			if name == "maps" {
				target = new(StrictMaps)
			} else {
				target = new(StrictHangarParameters)
			}
			if err := decodeStrictJSON(original, target); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(target)
			if err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, original, encoded)
		})
	}
}

func assertJSONEqual(t *testing.T, a, b []byte) {
	t.Helper()
	var x, y any
	for _, pair := range []struct {
		raw   []byte
		value *any
	}{{a, &x}, {b, &y}} {
		decoder := json.NewDecoder(strings.NewReader(string(pair.raw)))
		decoder.UseNumber()
		if err := decoder.Decode(pair.value); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(x, y) {
		t.Fatal("JSON values changed after round-trip")
	}
}
