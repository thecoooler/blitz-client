package export

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestStrictProvisions(t *testing.T) {
	good := `{"common":{"one":{"isAbility":null,"script":"Simple","descrParam":{"name":"a"}},"two":{"script":{"#text":"X","bonusValues":{"crewLevelIncrease":"0"}},"descrParam":[{"value":"2"},null,{"value":"1"}]}}}`
	var provisions StrictProvisions
	if err := decodeStrictJSON([]byte(good), &provisions); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(provisions)
	if err != nil {
		t.Fatal(err)
	}
	var expected, actual any
	if err := json.Unmarshal([]byte(good), &expected); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("round trip: %s", encoded)
	}
	for _, input := range []string{
		`{"common":{"one":{"unknown":1}}}`,
		`{"common":{"one":{"isAbility":"true"}}}`,
		`{"common":{"one":{"script":{"#text":"X","bonusValues":{"unknown":"1"}}}}}`,
	} {
		if err := decodeStrictJSON([]byte(input), &provisions); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
}

func TestStrictProvisionsFullExport(t *testing.T) {
	forEachFullExport(t, func(t *testing.T, data []byte) {
		var root struct {
			Provisions json.RawMessage `json:"provisions"`
		}
		if err := json.Unmarshal(data, &root); err != nil {
			t.Fatal(err)
		}
		var provisions StrictProvisions
		if err := decodeStrictJSON(root.Provisions, &provisions); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(provisions)
		if err != nil {
			t.Fatal(err)
		}
		var expected, actual any
		for i, b := range [][]byte{root.Provisions, encoded} {
			decoder := json.NewDecoder(bytes.NewReader(b))
			decoder.UseNumber()
			if i == 0 {
				err = decoder.Decode(&expected)
			} else {
				err = decoder.Decode(&actual)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Fatal("provisions round trip differs")
		}
	})
}
