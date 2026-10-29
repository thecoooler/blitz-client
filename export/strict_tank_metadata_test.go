package export

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestStrictTankMetadataRoundTrip(t *testing.T) {
	for _, input := range []string{
		`{"level":"0","price":"123","combatRole":{"arcade":"cavalry","unknownMode":null}}`,
		`{"price":{"#text":"100","gold":null},"sellPrice":{"#text":"2250","gold":{"dateFactors":{"factor":{"expirationDate":"26.06.2025","value":"1"}}}},"description":null}`,
	} {
		var metadata StrictTankMetadata
		if err := json.Unmarshal([]byte(input), &metadata); err != nil {
			t.Fatalf("decode %s: %v", input, err)
		}
		out, err := json.Marshal(metadata)
		if err != nil {
			t.Fatal(err)
		}
		assertEquivalentJSON(t, []byte(input), out)
	}
}

func TestStrictTankMetadataRejectsBadShapes(t *testing.T) {
	for _, input := range []string{
		`null`, `[]`, `{"level":3}`, `{"price":false}`, `{"price":{"unknown":"x"}}`,
		`{"sellPrice":{"gold":{"dateFactors":{"factor":{"value":3}}}}}`,
		`{"other":"value"}`, `{"level":"1","level":"2"}`,
	} {
		var metadata StrictTankMetadata
		if err := json.Unmarshal([]byte(input), &metadata); err == nil {
			t.Errorf("accepted invalid %s", input)
		}
	}
}

func TestStrictTankMetadataFullExport(t *testing.T) {
	data := fullExportData(t)
	var snapshot struct {
		Tanks []struct {
			Metadata json.RawMessage `json:"metadata"`
			ID       string          `json:"id"`
		} `json:"tanks"`
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Tanks) == 0 {
		t.Fatal("no tanks")
	}
	for _, tank := range snapshot.Tanks {
		var metadata StrictTankMetadata
		if err := json.Unmarshal(tank.Metadata, &metadata); err != nil {
			t.Fatalf("%s: %v", tank.ID, err)
		}
		out, err := json.Marshal(metadata)
		if err != nil {
			t.Fatalf("%s: %v", tank.ID, err)
		}
		assertEquivalentJSON(t, tank.Metadata, out)
	}
}

func assertEquivalentJSON(t *testing.T, left, right []byte) {
	t.Helper()
	decode := func(b []byte) interface{} {
		var v interface{}
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		if err := d.Decode(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if !reflect.DeepEqual(decode(left), decode(right)) {
		t.Fatalf("JSON differs: %s vs %s", left, right)
	}
}
