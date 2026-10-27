//go:build windows

package replay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenReplayAllowsConcurrentFinalization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.replay")
	if err := os.WriteFile(path, sampleReplay(), 0600); err != nil {
		t.Fatal(err)
	}
	file, err := openReplay(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	// A writer and finalizer must remain unblocked while our reader is open.
	writer, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	finalized := filepath.Join(filepath.Dir(path), "finalized.replay")
	if err := os.Rename(path, finalized); err != nil {
		t.Fatalf("reader blocked rename: %v", err)
	}
	if err := os.Remove(finalized); err != nil {
		t.Fatalf("reader blocked removal: %v", err)
	}
}
