package export

import (
	"os"
	"testing"
)

// fullExportData reads the data.json named by BLITZ_EXPORT_DATA. Generate it with
// `blitz-client client export --include data` to validate the strict schema
// against installed game resources without committing them.
func fullExportData(t *testing.T) []byte {
	t.Helper()
	path := os.Getenv("BLITZ_EXPORT_DATA")
	if path == "" {
		t.Skip("set BLITZ_EXPORT_DATA to a data.json from client export --include data")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
