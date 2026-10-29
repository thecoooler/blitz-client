package export

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestXMLScalarSpelling(t *testing.T) {
	for _, tc := range []struct {
		kind  XMLScalarKind
		valid []string
		bad   []string
	}{
		{XMLIntegerKind, []string{"0", "-12", "+3", "007"}, []string{"", "1.0", "1e3", " 1", "x"}},
		{XMLNumberKind, []string{"0", "-1.5", ".5", "5.", "1e-3", "+2E+4"}, []string{"", ".", "1e", "1 2", "inf", "0x10", "1_000"}},
		{XMLBoolKind, []string{"true", "false", "True", "False"}, []string{"", "TRUE", "1", "fasle"}},
		{XMLLocalizationKeyKind, []string{"#menu:key", "#ns:a/b-c.d e", "#ns: x"}, []string{"#ns:", "#:key", "ns:key", "#n s:key", ""}},
		{XMLNumberVectorKind, []string{"1", "0 0.5 1", "50 60\n\t\t100 90\n", " 1  2 "}, []string{"", " ", "1 x", "1,2"}},
	} {
		for _, s := range tc.valid {
			if !tc.kind.Matches(s) {
				t.Errorf("%s rejected %q", tc.kind, s)
			}
		}
		for _, s := range tc.bad {
			if tc.kind.Matches(s) {
				t.Errorf("%s accepted %q", tc.kind, s)
			}
		}
	}
}

func TestXMLScalarValues(t *testing.T) {
	type doc struct {
		I Optional[XMLInteger]         `json:"i,omitzero"`
		N Optional[XMLNumber]          `json:"n,omitzero"`
		B Optional[XMLBool]            `json:"b,omitzero"`
		L Optional[XMLLocalizationKey] `json:"l,omitzero"`
		V Optional[XMLNumberVector]    `json:"v,omitzero"`
	}
	input := `{"i":"+9007199254740993","n":"0.10","b":"True","l":"#menu:tank/name","v":"1 2.50\n3"}`
	var d doc
	if err := decodeStrictJSON([]byte(input), &d); err != nil {
		t.Fatal(err)
	}
	i, _ := d.I.Get()
	if n, err := i.Int64(); err != nil || n != 9007199254740993 {
		t.Fatalf("integer %d %v", n, err)
	}
	n, _ := d.N.Get()
	if f, err := n.Float64(); err != nil || f != 0.1 || n != "0.10" {
		t.Fatalf("number %v %v", f, err)
	}
	b, _ := d.B.Get()
	if v, err := b.Bool(); err != nil || !v {
		t.Fatalf("bool %v %v", v, err)
	}
	l, _ := d.L.Get()
	if l.Namespace() != "menu" || l.Key() != "tank/name" {
		t.Fatalf("key %q %q", l.Namespace(), l.Key())
	}
	v, _ := d.V.Get()
	if fs, err := v.Float64s(); err != nil || len(fs) != 3 || fs[1] != 2.5 || v.Numbers()[1] != "2.50" {
		t.Fatalf("vector %v %v", fs, err)
	}
	out, err := json.Marshal(d)
	if err != nil || string(out) != input {
		t.Fatalf("round trip %s %v", out, err)
	}
	for _, bad := range []string{`{"i":"1.5"}`, `{"n":"abc"}`, `{"n":1}`, `{"b":"yes"}`, `{"l":"plain"}`, `{"v":"1 x"}`} {
		if err := decodeStrictJSON([]byte(bad), &d); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

type lenientInner struct {
	Value Optional[XMLNumber] `json:"value,omitzero"`
	Name  string              `json:"name"`
}

type lenientDoc struct {
	Number Optional[XMLNumber]                       `json:"number,omitzero"`
	Inner  Optional[lenientInner]                    `json:"inner,omitzero"`
	Dict   Optional[map[string]Optional[XMLInteger]] `json:"dict,omitzero"`
	List   Optional[[]Optional[XMLInteger]]          `json:"list,omitzero"`
	Rows   Optional[[]lenientInner]                  `json:"rows,omitzero"`
}

func TestLenientDecoding(t *testing.T) {
	input := `{"number":"x1","inner":{"value":"2","name":"a","new":{"deep":[1]}},"dict":{"a":"1","b":"two"},"list":["1","bad","3"],"rows":[{"name":"keep"},{"value":"1"}],"future":true}`
	var strict lenientDoc
	err := decodeStrictJSON([]byte(input), &strict)
	var schemaErr *SchemaError
	if !errors.As(err, &schemaErr) || schemaErr.Path != "$.number" {
		t.Fatalf("strict error %v", err)
	}
	var d lenientDoc
	warnings, err := decodeSchemaJSON([]byte(input), &d, schemaOptions{lenient: true})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, w := range warnings {
		got = append(got, string(w.Kind)+" "+w.Path+" "+w.SchemaPath+" "+w.Dropped)
	}
	want := []string{
		"unexpected-value $.number $.number $.number",
		"unknown-field $.inner.new $.inner.new ",
		`unexpected-value $.dict.b $.dict{} $.dict.b`,
		"unexpected-value $.list[1] $.list[] $.list[1]",
		`unexpected-value $.rows[1] $.rows[] $.rows[1]`,
		"unknown-field $.future $.future ",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("warnings:\n%s", strings.Join(got, "\n"))
	}
	out, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	const kept = `{"inner":{"value":"2","name":"a"},"dict":{"a":"1"},"list":["1","3"],"rows":[{"name":"keep"}]}`
	if string(out) != kept {
		t.Fatalf("kept %s", out)
	}
	// Malformed documents fail in either mode.
	for _, bad := range []string{`{"number":"1","number":"2"}`, `{"number":"1"} {}`, `[`} {
		if _, err := decodeSchemaJSON([]byte(bad), &d, schemaOptions{lenient: true}); err == nil {
			t.Errorf("lenient accepted %s", bad)
		}
	}
}

func TestLenientPartialExport(t *testing.T) {
	doc := `{"format":"blitz-client","version":1,"included":["data"],"nations":["ussr"],"tanks":[{"id":"ussr:1","nation":"ussr","clientId":1,"techName":"T","future":1,"configuration":{"hull":{"weight":"heavy"}}}],"issues":[],"newSection":{}}`
	if _, err := DecodePartialClientExport(strings.NewReader(doc)); err == nil {
		t.Fatal("strict accepted unknown data")
	}
	s, warnings, err := DecodePartialClientExportLenient(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 3 || warnings[0].Path != "$.tanks[0].future" || warnings[1].Path != "$.tanks[0].configuration.hull.weight" || warnings[2].Path != "$.newSection" {
		t.Fatalf("warnings %v", warnings)
	}
	if len(s.Tanks) != 1 || s.Tanks[0].TechName != "T" {
		t.Fatalf("tanks %+v", s.Tanks)
	}
	if _, _, err := DecodePartialClientExportLenient(strings.NewReader(strings.Replace(doc, `"version":1`, `"version":"1"`, 1))); err == nil {
		t.Fatal("lenient accepted an invalid root field")
	}
}

func TestOptionalStates(t *testing.T) {
	var absent Optional[string]
	if absent.Present() || absent.Null() || !absent.IsZero() {
		t.Fatal("zero value must be absent")
	}
	if n := NullValue[string](); !n.Present() || !n.Null() {
		t.Fatal("null state")
	}
	if v, ok := Some("x").Get(); !ok || v != "x" {
		t.Fatal("value state")
	}
}
