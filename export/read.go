package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadClientExport reads the same snapshot regardless of its selected sections.
// Media paths in the result are relative to directory. Unknown fields are allowed
// for additive schema evolution; unknown format versions are rejected.
func ReadClientExport(directory string) (*Snapshot, error) {
	path := filepath.Join(directory, "data.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Snapshot
	// Numbers keep their exact spelling; float64 would round large YAML integers.
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if s.Format != "blitz-client" || s.Version != 1 {
		return nil, fmt.Errorf("unsupported client export %q version %d", s.Format, s.Version)
	}
	if len(s.Included) == 0 {
		return nil, fmt.Errorf("snapshot is missing included sections")
	}
	if _, err := ParseExportIncludes(s.Included); err != nil {
		return nil, err
	}
	return &s, nil
}
