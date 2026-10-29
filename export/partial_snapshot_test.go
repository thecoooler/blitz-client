package export

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestPartialSnapshotRoundTrip(t *testing.T) {
	input := `{"format":"blitz-client","version":1,"included":["data"],"nations":["ussr"],"tanks":[{"id":"ussr:1","nation":"ussr","clientId":1,"techName":"Tank","metadata":{"level":"0","price":null},"parameters":null,"configuration":{"chassis":[{"name":"B","properties":{"level":"9007199254740993"}},{"name":"A","properties":null}]}}],"maps":null,"components":{"ussr":{"guns":{"ids":{"gun":"9007199254740993"}}}},"issues":[]}`
	s, err := DecodePartialClientExport(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.UnvalidatedSections) != 0 {
		t.Fatalf("unvalidated: %v", s.UnvalidatedSections)
	}
	if !s.Maps.Present() || !s.Maps.Null() || !s.Tanks[0].Parameters.Null() {
		t.Fatal("null not retained")
	}
	metadata, ok := s.Tanks[0].Metadata.Get()
	if !ok || !metadata.Price.Null() {
		t.Fatal("metadata null not retained")
	}
	got, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]json.RawMessage
	var actual map[string]json.RawMessage
	if err := json.Unmarshal([]byte(input), &expected); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &actual); err != nil {
		t.Fatal(err)
	}
	// Object key order is immaterial; raw subtree comparison also proves that
	// ordered selections and numbers were not changed.
	for _, k := range []string{"components", "tanks", "maps", "issues"} {
		var a, b bytes.Buffer
		if err := json.Compact(&a, actual[k]); err != nil {
			t.Fatal(err)
		}
		if err := json.Compact(&b, expected[k]); err != nil {
			t.Fatal(err)
		}
		if k == "tanks" { // metadata object fields can be emitted in struct order.
			if !bytes.Contains(a.Bytes(), []byte(`"name":"B","properties":{"level":"9007199254740993"}`)) || !bytes.Contains(a.Bytes(), []byte(`"name":"A"`)) {
				t.Fatalf("tank configuration lost: %s", &a)
			}
			continue
		}
		if !bytes.Equal(a.Bytes(), b.Bytes()) {
			t.Fatalf("%s: got %s want %s", k, &a, &b)
		}
	}
}

func TestPartialSnapshotRejectsInvalidTypedData(t *testing.T) {
	base := `{"format":"blitz-client","version":1,"included":["data"],"nations":[],"tanks":[],"issues":[]}`
	cases := []struct{ name, input string }{
		{"duplicate", strings.Replace(base, `"issues":[]`, `"issues":[],"issues":[]`, 1)},
		{"trailing", base + ` {}`},
		{"null nation", strings.Replace(base, `"nations":[]`, `"nations":[null]`, 1)},
		{"null identity", strings.Replace(base, `"tanks":[]`, `"tanks":[{"id":null,"nation":"x","clientId":1,"techName":"x"}]`, 1)},
		{"case", strings.Replace(base, `"format"`, `"Format"`, 1)},
		{"invalid map", strings.Replace(base, `"issues":[]`, `"maps":{"unexpected":1},"issues":[]`, 1)},
		{"invalid version", strings.Replace(base, `"version":1`, `"version":2`, 1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodePartialClientExport(strings.NewReader(tc.input)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

// Set BLITZ_EXPORT_DATA to a full data.json to validate the installed client.
func TestPartialSnapshotCurrentExport(t *testing.T) {
	path := os.Getenv("BLITZ_EXPORT_DATA")
	if path == "" {
		t.Skip("BLITZ_EXPORT_DATA not set")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s, err := DecodeStrictDataExport(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("strict data tanks: %d", len(s.Tanks))
	got, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&a); err != nil {
		t.Fatal(err)
	}
	decoder = json.NewDecoder(bytes.NewReader(got))
	decoder.UseNumber()
	if err := decoder.Decode(&b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("partial snapshot did not round-trip")
	}
}
