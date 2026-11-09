package export

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// exportEnvelopeError marks a document that fails the shared v1 envelope
// contract. Directory readers report it without adding the file path, so both
// entry points produce the same message.
type exportEnvelopeError struct{ err error }

func (e *exportEnvelopeError) Error() string { return e.err.Error() }
func (e *exportEnvelopeError) Unwrap() error { return e.err }

// validateExportEnvelope applies the one document contract shared by every
// reader: the blitz-client v1 format and a non-empty, valid included selection.
func validateExportEnvelope(format string, version int, included []string) error {
	if format != "blitz-client" || version != 1 {
		return &exportEnvelopeError{fmt.Errorf("unsupported client export %q version %d", format, version)}
	}
	if len(included) == 0 {
		return &exportEnvelopeError{fmt.Errorf("snapshot is missing included sections")}
	}
	if _, err := ParseExportIncludes(included); err != nil {
		return &exportEnvelopeError{err}
	}
	return nil
}

// readExportDocument reads data.json from directory and returns its path for
// error messages. Every reader that accepts a directory uses this helper.
func readExportDocument(directory string) (path string, data []byte, err error) {
	path = filepath.Join(directory, "data.json")
	data, err = os.ReadFile(path)
	return path, data, err
}

// ReadClientExport reads data.json in directory with DecodeClientExport and
// returns the untyped Snapshot projection: game data stays as values and is not
// validated. Media paths in the result are relative to directory.
func ReadClientExport(directory string) (*Snapshot, error) {
	path, data, err := readExportDocument(directory)
	if err != nil {
		return nil, err
	}
	s, err := DecodeClientExport(bytes.NewReader(data))
	if err != nil {
		var envelope *exportEnvelopeError
		if errors.As(err, &envelope) {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// DecodeClientExport reads one v1 data.json document of any --include selection
// and returns the untyped Snapshot projection: game data stays as values and is
// not validated. Unknown fields are allowed for additive schema evolution;
// unsupported formats, versions and included selections are rejected. Numbers
// keep their exact spelling; float64 would round large YAML integers.
func DecodeClientExport(r io.Reader) (*Snapshot, error) {
	var s Snapshot
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	if err := decoder.Decode(&s); err != nil {
		return nil, err
	}
	if err := validateExportEnvelope(s.Format, s.Version, s.Included); err != nil {
		return nil, err
	}
	return &s, nil
}
