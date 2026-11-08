package export

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestStrictGameModeDecoding(t *testing.T) {
	const input = `{"format":"blitz-client","version":1,"included":["data"],"nations":[],"tanks":[],"modes":[{"id":"regular","name":"battleType/regular","description":"hangar/battleType/regularDescription","shortDescription":null},{"id":"tournament","name":"battleResults/Footer/Tournament","description":null,"shortDescription":null}],"issues":[]}`
	s, err := DecodeStrictDataExport(bytes.NewReader([]byte(input)))
	if err != nil {
		t.Fatal(err)
	}
	modes, ok := s.Modes.Get()
	if !ok || len(modes) != 2 || modes[0].ID != "regular" || modes[1].ID != "tournament" {
		t.Fatalf("modes: %+v", modes)
	}
	if description, ok := modes[0].Description.Get(); !ok || description != "hangar/battleType/regularDescription" {
		t.Fatalf("regular description: %+v", modes[0])
	}
	if !modes[0].ShortDescription.Null() || !modes[1].Description.Null() {
		t.Fatalf("null descriptions lost: %+v", modes)
	}
	encoded, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var want, got any
	if err := json.Unmarshal([]byte(input), &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip: %s", encoded)
	}
}
