package export

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestStrictNumber(t *testing.T) {
	for _, input := range []string{"9007199254740993", "-1.250e+30", "0"} {
		var n StrictNumber
		if err := json.Unmarshal([]byte(input), &n); err != nil || string(n) != input {
			t.Fatalf("%s: %q, %v", input, n, err)
		}
		encoded, err := json.Marshal(n)
		if err != nil || string(encoded) != input {
			t.Fatalf("%s: %s, %v", input, encoded, err)
		}
	}
	for _, input := range []string{`"1"`, "null", "true", "01", "{}"} {
		var n StrictNumber
		if err := json.Unmarshal([]byte(input), &n); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
}

func TestStrictCrewSkill(t *testing.T) {
	input := `{"id":"skill","userString":"#crew:name","description":"#crew:description","tip":"","features":"","tankClass":"","type":"continuous","parameters":{"chancePerLevel":9007199254740993},"levels":null}`
	var skill StrictCrewSkill
	if err := json.Unmarshal([]byte(input), &skill); err != nil {
		t.Fatal(err)
	}
	if number, ok := skill.Parameters.ChancePerLevel.Get(); !ok || number != "9007199254740993" || skill.Levels != nil {
		t.Fatalf("lost data: %+v", skill)
	}
	encoded, err := json.Marshal(skill)
	if err != nil || !bytes.Contains(encoded, []byte(`9007199254740993`)) {
		t.Fatalf("round trip: %s, %v", encoded, err)
	}
	for _, bad := range []string{
		`{"id":null,"userString":"#crew:name","description":"#crew:description","tip":"","features":"","tankClass":"","type":"x","parameters":{},"levels":null}`,
		`{"ID":"skill","userString":"#crew:name","description":"#crew:description","tip":"","features":"","tankClass":"","type":"x","parameters":{},"levels":null}`,
		`{"id":"skill","userString":"#crew:name","description":"#crew:description","tip":"","features":"","tankClass":"","type":"x","parameters":{"future":1},"levels":null}`,
		`{"id":"skill","userString":"#crew:name","description":"#crew:description","tip":"","features":"","tankClass":"","type":"x","parameters":{"chancePerLevel":"1"},"levels":null}`,
	} {
		if err := json.Unmarshal([]byte(bad), &skill); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

// Set BLITZ_EXPORT_DATA to exercise the installed client corpus.
func TestStrictCrewSkillFullExport(t *testing.T) {
	forEachFullExport(t, func(t *testing.T, data []byte) {
		var source struct {
			CrewSkills []json.RawMessage `json:"crewSkills"`
		}
		if err := json.Unmarshal(data, &source); err != nil {
			t.Fatal(err)
		}
		if len(source.CrewSkills) == 0 {
			t.Fatal("no crew skills")
		}
		for _, raw := range source.CrewSkills {
			var skill StrictCrewSkill
			if err := json.Unmarshal(raw, &skill); err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(skill)
			if err != nil {
				t.Fatal(err)
			}
			var original, roundTrip any
			for i, payload := range [][]byte{raw, got} {
				decoder := json.NewDecoder(bytes.NewReader(payload))
				decoder.UseNumber()
				if i == 0 {
					err = decoder.Decode(&original)
				} else {
					err = decoder.Decode(&roundTrip)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			a, _ := json.Marshal(original)
			b, _ := json.Marshal(roundTrip)
			if !bytes.Equal(a, b) {
				t.Fatalf("skill %s round trip differs: %s / %s", skill.ID, a, b)
			}
		}
	})
}
