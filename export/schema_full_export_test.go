package export

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Tests in this file validate real exports listed in BLITZ_EXPORT_DATA.

func TestFullExportRoundTrip(t *testing.T) {
	forEachFullExport(t, func(t *testing.T, data []byte) {
		s, err := DecodePartialClientExport(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		for _, section := range s.UnvalidatedSections {
			if section != "tanks[].icons" && section != "tanks[].model" && section != "crewSkillTraining" {
				t.Errorf("unvalidated data section %s", section)
			}
		}
		t.Logf("included %v, tanks %d, raw sections %v", s.Included, len(s.Tanks), s.UnvalidatedSections)
		assertJSONRoundTrip(t, data, s)
		if len(s.UnvalidatedSections) == 0 && len(s.Included) == 1 && s.Included[0] == "data" {
			strict, err := DecodeStrictDataExport(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			assertJSONRoundTrip(t, data, strict)
		}
	})
}

func TestFullExportLenientWithoutWarnings(t *testing.T) {
	forEachFullExport(t, func(t *testing.T, data []byte) {
		s, warnings, err := DecodePartialClientExportLenient(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		for i, w := range warnings {
			if i == 10 {
				break
			}
			t.Error(w)
		}
		assertJSONRoundTrip(t, data, s)
	})
}

// Injects an unknown key into objects of every observed key set. Lenient
// decoding must report only the probes and keep everything else unchanged;
// strict decoding must reject the document.
func TestFullExportLenientIgnoresUnknownFields(t *testing.T) {
	const probe = "__probe__"
	forEachFullExport(t, func(t *testing.T, data []byte) {
		root := decodeGenericJSON(t, data)
		perShape := map[string]int{}
		probes := 0
		var inject func(v any, path string)
		inject = func(v any, path string) {
			switch x := v.(type) {
			case map[string]any:
				keys := make([]string, 0, len(x))
				for key, child := range x {
					keys = append(keys, key)
					inject(child, path+jsonPathKey(key))
				}
				sort.Strings(keys)
				shape := strings.Join(keys, "\x00")
				if perShape[shape] < 2 {
					perShape[shape]++
					x[probe] = map[string]any{"probe": []any{json.Number("1"), "a", nil}}
					probes++
				}
			case []any:
				for i, child := range x {
					inject(child, path+"["+strconv.Itoa(i)+"]")
				}
			}
		}
		inject(root, "$")
		probed, err := json.Marshal(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodePartialClientExport(bytes.NewReader(probed)); err == nil {
			t.Fatal("strict decoding accepted unknown fields")
		}
		s, warnings, err := DecodePartialClientExportLenient(bytes.NewReader(probed))
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range warnings {
			if !strings.Contains(w.Path, probe) || (w.Dropped != "" && !strings.Contains(w.Dropped, probe)) {
				t.Fatalf("warning outside probes: %s", w)
			}
		}
		if len(warnings) < probes/2 {
			t.Fatalf("%d warnings for %d probes", len(warnings), probes)
		}
		t.Logf("%d probes, %d warnings", probes, len(warnings))
		got, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		decoded := decodeGenericJSON(t, got)
		stripKey(decoded, probe)
		if !reflect.DeepEqual(decodeGenericJSON(t, data), decoded) {
			t.Fatal("lenient decoding changed data outside the probes")
		}
	})
}

// The declared XML scalar types must accept every value of every export. With
// exports of two or more game versions, the declared types must also equal
// the inferred ones (go run ./internal/tools/exportschema -write reproduces
// them).
func TestFullExportScalarTypes(t *testing.T) {
	paths := fullExportPaths()
	analysis := &SchemaAnalysis{}
	forEachFullExport(t, func(t *testing.T, data []byte) {
		a, err := AnalyzeClientExportSchema(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		analysis.Merge(a)
	})
	if len(analysis.Warnings) > 0 {
		t.Fatalf("schema warnings, e.g. %s", analysis.Warnings[0])
	}
	drift := 0
	for _, key := range analysis.SortedFields() {
		usage := analysis.Fields[key]
		if usage.Mismatches > 0 {
			t.Errorf("%s.%s: %d values rejected by %s", key.Owner, key.Field, usage.Mismatches, usage.Declared)
		}
		if usage.Inferred() != usage.Declared {
			drift++
			if len(paths) >= 2 {
				t.Errorf("%s.%s: declared %s, values fit %s", key.Owner, key.Field, usage.Declared, usage.Inferred())
			}
		}
	}
	t.Logf("%d XML fields, %d could be stricter for these exports", len(analysis.Fields), drift)
}

func BenchmarkDecodeFullExport(b *testing.B) {
	paths := fullExportPaths()
	if len(paths) == 0 {
		b.Skip("set BLITZ_EXPORT_DATA")
	}
	data := readFile(b, paths[0])
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		if _, err := DecodePartialClientExport(bytes.NewReader(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func readFile(tb testing.TB, path string) []byte {
	tb.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatal(err)
	}
	return data
}

func jsonPathKey(key string) string {
	if isPathIdentifier(key) {
		return "." + key
	}
	return "[" + strconv.Quote(key) + "]"
}

func decodeGenericJSON(t *testing.T, data []byte) any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var v any
	if err := decoder.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func stripKey(v any, key string) {
	switch x := v.(type) {
	case map[string]any:
		delete(x, key)
		for _, child := range x {
			stripKey(child, key)
		}
	case []any:
		for _, child := range x {
			stripKey(child, key)
		}
	}
}

func assertJSONRoundTrip(t *testing.T, input []byte, value any) {
	t.Helper()
	output, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decodeGenericJSON(t, input), decodeGenericJSON(t, output)) {
		t.Fatal("re-encoded export differs from the input")
	}
}
