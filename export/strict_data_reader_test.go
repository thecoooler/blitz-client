package export

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestStrictDataReaderRejectsUnvalidatedSections(t *testing.T) {
	const doc = `{"format":"blitz-client","version":1,"included":["data"],"nations":[],"tanks":[],"settings":{"future":{"any":true}},"issues":[]}`
	partial, err := DecodePartialClientExport(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(partial.UnvalidatedSections) != 1 || partial.UnvalidatedSections[0] != "settings.future" {
		t.Fatalf("gaps: %v", partial.UnvalidatedSections)
	}
	if _, err := DecodeStrictDataExport(strings.NewReader(doc)); err == nil || !strings.Contains(err.Error(), "settings.future") {
		t.Fatalf("error: %v", err)
	}
	withoutGap := strings.Replace(doc, `"settings":{"future":{"any":true}}`, `"settings":{}`, 1)
	if _, err := DecodeStrictDataExport(strings.NewReader(withoutGap)); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeStrictDataExport(strings.NewReader(strings.Replace(withoutGap, `["data"]`, `["data","icons"]`, 1))); err == nil {
		t.Fatal("accepted media export")
	}
	if _, err := DecodeStrictDataExport(strings.NewReader(strings.Replace(withoutGap, `"settings":{}`, `"crewSkillTraining":null`, 1))); err == nil {
		t.Fatal("accepted unvalidated training")
	}
}

func TestStrictDataReaderPreservesValues(t *testing.T) {
	const input = `{"format":"blitz-client","version":1,"included":["data"],"nations":["ussr"],"tanks":[{"id":"ussr:1","nation":"ussr","clientId":1,"techName":"Tank","parameters":{"chassis":{"textureScale":9007199254740993}},"configuration":{"chassis":[{"name":"B","properties":null},{"name":"A","properties":{}}]}}],"hangarParameters":{"keyedArchive":{"missing":{"keyedArchive":{}},"explicit":{"keyedArchive":{"digits":null}}}},"issues":[]}`
	s, err := DecodeStrictDataExport(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	parameters, ok := s.Tanks[0].Parameters.Get()
	if !ok {
		t.Fatal("missing parameters")
	}
	chassis, ok := parameters.Chassis.Get()
	if !ok {
		t.Fatal("missing chassis")
	}
	n, ok := chassis.TextureScale.Get()
	if !ok || n != "9007199254740993" {
		t.Fatalf("number: %q", n)
	}
	config, ok := s.Tanks[0].Configuration.Get()
	if !ok {
		t.Fatal("missing configuration")
	}
	choices, ok := config.Chassis.Get()
	if !ok || len(choices) != 2 {
		t.Fatal("missing choices")
	}
	b, ok := choices[0].Get()
	if !ok || !b.Properties.Null() {
		t.Fatal("null selection properties lost")
	}
	bname, _ := b.Name.Get()
	a, _ := choices[1].Get()
	aname, _ := a.Name.Get()
	if bname != "B" || aname != "A" {
		t.Fatalf("selection order: %s %s", bname, aname)
	}
	hangar, ok := s.HangarParameters.Get()
	if !ok {
		t.Fatal("missing hangar")
	}
	archive, ok := hangar.KeyedArchive.Get()
	if !ok {
		t.Fatal("missing archive")
	}
	missing, ok := archive["missing"].KeyedArchive.Get()
	if !ok || missing.Digits.Present() {
		t.Fatal("invented digits")
	}
	explicit, ok := archive["explicit"].KeyedArchive.Get()
	if !ok || !explicit.Digits.Null() {
		t.Fatal("null digits lost")
	}
	output, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var expected, actual any
	if err := decodeStrictJSON([]byte(input), &expected); err != nil {
		t.Fatal(err)
	}
	if err := decodeStrictJSON(output, &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("round-trip mismatch: %s", output)
	}
	if _, err := DecodeStrictDataExport(strings.NewReader(input + ` {}`)); err == nil {
		t.Fatal("accepted trailing value")
	}
}
