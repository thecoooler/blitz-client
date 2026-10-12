package yamlv

import (
	"os"
	"testing"
)

// The expected output was produced by the Rust tool (serde_yaml_ng) from the
// same input after inserting mod.patch as the first patch source.
func TestReserializesLikeSerdeYAML(t *testing.T) {
	input, err := os.ReadFile("testdata/reserialize.in.yaml")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("testdata/reserialize.out.yaml")
	if err != nil {
		t.Fatal(err)
	}
	node, err := ParseDocument(input)
	if err != nil {
		t.Fatal(err)
	}
	document, err := Build(node, "")
	if err != nil {
		t.Fatal(err)
	}
	document.Set("patch", NewSeq([]string{"mod.patch", "extra.patch"}))
	output, err := Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != string(expected) {
		t.Fatalf("got:\n%s\nwant:\n%s", output, expected)
	}
}

func TestResolvesPlainScalarsLikeSerdeYAML(t *testing.T) {
	for text, kind := range map[string]Kind{
		"": Null, "~": Null, "null": Null, "NULL": Null, "true": Bool, "False": Bool, "yes": String, "on": String,
		"1": Int, "+1": Int, "-1": Int, "0x1F": Int, "-0o17": Int, "0b101": Int, "012": String, "-012": String,
		"1.": Float, ".5": Float, "1e5": Float, ".inf": Float, "-.Inf": Float, ".nan": Float, "inf": String,
		"1e400": String, "99999999999999999999": Float, "0x": String, "1_000": String, "+-1": String,
	} {
		if got := ResolvePlain(text).Kind; got != kind {
			t.Errorf("%q: got %v want %v", text, got, kind)
		}
	}
}
