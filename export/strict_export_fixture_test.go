package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// forEachFullExport runs fn for every data.json listed in BLITZ_EXPORT_DATA
// (separated by the OS path list separator). Generate them with
// `blitz-client client export`, with or without media, to validate the schema
// against installed game resources without committing them.
func forEachFullExport(t *testing.T, fn func(t *testing.T, data []byte)) {
	t.Helper()
	paths := fullExportPaths()
	if len(paths) == 0 {
		t.Skip("set BLITZ_EXPORT_DATA to data.json files from client export")
	}
	for _, path := range paths {
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			fn(t, data)
		})
	}
}

func fullExportPaths() []string {
	var paths []string
	for _, path := range strings.Split(os.Getenv("BLITZ_EXPORT_DATA"), string(os.PathListSeparator)) {
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}
