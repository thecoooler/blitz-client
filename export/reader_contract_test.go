package export

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// representativeDataOnlyExport is a complete v1 data-only document: it carries
// every required root field and decodes through the strict schema.
const representativeDataOnlyExport = `{"format":"blitz-client","version":1,"included":["data"],"nations":["ussr"],"tanks":[{"id":"ussr:1","nation":"ussr","clientId":1,"techName":"Tank","metadata":{"level":"0","price":null},"parameters":null,"configuration":{"chassis":[{"name":"B","properties":{"level":"9007199254740993"}},{"name":"A","properties":null}]}}],"maps":null,"components":{"ussr":{"guns":{"ids":{"gun":"9007199254740993"}}}},"issues":[]}`

type readerEnvelope struct {
	Format   string
	Version  int
	Included []string
}

func envelopeOf(format string, version int, included []string) readerEnvelope {
	return readerEnvelope{Format: format, Version: version, Included: included}
}

func requireEnvelope(t *testing.T, got readerEnvelope, want readerEnvelope) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("envelope %+v, want %+v", got, want)
	}
}

// One representative export must expose the same envelope through every reader,
// on both the directory and io.Reader entry points.
func TestExportReadersShareOneDocumentContract(t *testing.T) {
	directory := t.TempDir()
	data := []byte(representativeDataOnlyExport)
	if err := os.WriteFile(filepath.Join(directory, "data.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	want := envelopeOf("blitz-client", 1, []string{"data"})

	snapshot, err := ReadClientExport(directory)
	if err != nil {
		t.Fatal(err)
	}
	requireEnvelope(t, envelopeOf(snapshot.Format, snapshot.Version, snapshot.Included), want)

	decoded, err := DecodeClientExport(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	requireEnvelope(t, envelopeOf(decoded.Format, decoded.Version, decoded.Included), want)

	partial, err := ReadPartialClientExport(directory)
	if err != nil {
		t.Fatal(err)
	}
	requireEnvelope(t, envelopeOf(partial.Format, partial.Version, partial.Included), want)

	partialDecoded, err := DecodePartialClientExport(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	requireEnvelope(t, envelopeOf(partialDecoded.Format, partialDecoded.Version, partialDecoded.Included), want)

	lenient, warnings, err := ReadPartialClientExportLenient(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	requireEnvelope(t, envelopeOf(lenient.Format, lenient.Version, lenient.Included), want)

	lenientDecoded, warnings, err := DecodePartialClientExportLenient(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	requireEnvelope(t, envelopeOf(lenientDecoded.Format, lenientDecoded.Version, lenientDecoded.Included), want)

	strict, err := ReadStrictDataExport(directory)
	if err != nil {
		t.Fatal(err)
	}
	requireEnvelope(t, envelopeOf(strict.Format, strict.Version, strict.Included), want)

	strictDecoded, err := DecodeStrictDataExport(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	requireEnvelope(t, envelopeOf(strictDecoded.Format, strictDecoded.Version, strictDecoded.Included), want)
}

func TestExportReadersRejectTheSameMalformedEnvelopes(t *testing.T) {
	cases := []struct{ name, body string }{
		{"wrong format", `{"format":"other","version":1,"included":["data"],"nations":[],"tanks":[],"issues":[]}`},
		{"wrong version", `{"format":"blitz-client","version":2,"included":["data"],"nations":[],"tanks":[],"issues":[]}`},
		{"missing included", `{"format":"blitz-client","version":1,"nations":[],"tanks":[],"issues":[]}`},
		{"unknown included section", `{"format":"blitz-client","version":1,"included":["unknown"],"nations":[],"tanks":[],"issues":[]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, "data.json"), []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			readers := map[string]func() error{
				"ReadClientExport": func() error {
					_, err := ReadClientExport(directory)
					return err
				},
				"DecodeClientExport": func() error {
					_, err := DecodeClientExport(strings.NewReader(tc.body))
					return err
				},
				"ReadPartialClientExport": func() error {
					_, err := ReadPartialClientExport(directory)
					return err
				},
				"DecodePartialClientExport": func() error {
					_, err := DecodePartialClientExport(strings.NewReader(tc.body))
					return err
				},
				"ReadPartialClientExportLenient": func() error {
					_, _, err := ReadPartialClientExportLenient(directory)
					return err
				},
				"DecodePartialClientExportLenient": func() error {
					_, _, err := DecodePartialClientExportLenient(strings.NewReader(tc.body))
					return err
				},
				"ReadStrictDataExport": func() error {
					_, err := ReadStrictDataExport(directory)
					return err
				},
				"DecodeStrictDataExport": func() error {
					_, err := DecodeStrictDataExport(strings.NewReader(tc.body))
					return err
				},
				"AnalyzeClientExportSchema": func() error {
					_, err := AnalyzeClientExportSchema(strings.NewReader(tc.body))
					return err
				},
			}
			for name, read := range readers {
				if err := read(); err == nil {
					t.Errorf("%s accepted a malformed envelope", name)
				}
			}
		})
	}
}

func TestEnvelopePolicyMessages(t *testing.T) {
	cases := []struct {
		name     string
		format   string
		version  int
		included []string
		want     string
	}{
		{"wrong format", "other", 1, []string{"data"}, `unsupported client export "other" version 1`},
		{"wrong version", "blitz-client", 2, []string{"data"}, `unsupported client export "blitz-client" version 2`},
		{"missing included", "blitz-client", 1, nil, "snapshot is missing included sections"},
		{"empty included", "blitz-client", 1, []string{}, "snapshot is missing included sections"},
		{"unknown included section", "blitz-client", 1, []string{"unknown"}, `unknown export section "unknown"; expected all, data, icons, models, maps, mechanics, modes`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateExportEnvelope(tc.format, tc.version, tc.included)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}

func TestDecodeClientExportKeepsNumbersAndUnknownFields(t *testing.T) {
	const body = `{"format":"blitz-client","version":1,"included":["data"],"nations":[],"tanks":[{"id":"ussr:0","nation":"ussr","clientId":0,"techName":"T","parameters":{"mask":9007199254740993},"future":true}],"issues":[],"future":{"kept":true}}`
	s, err := DecodeClientExport(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Tanks[0].Parameters.(map[string]any)["mask"]; got != json.Number("9007199254740993") {
		t.Fatalf("large integer rounded: %v", got)
	}
	if s.Tanks[0].Metadata != nil {
		t.Fatal("unknown field decoded into the snapshot")
	}
}
