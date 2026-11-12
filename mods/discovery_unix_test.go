//go:build unix

package mods

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverModsInKeepsHealthySiblingsWhenNestedDirectoryIsUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	root := t.TempDir()
	healthy := filepath.Join(root, "healthy")
	blocked := filepath.Join(root, "blocked")
	for _, dir := range []string{healthy, blocked} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(healthy, "mod.yaml"), []byte("id: healthy"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })

	report := DiscoverModsIn([]string{root})
	if len(report.Manifests) != 1 {
		t.Fatalf("manifests = %v", report.Manifests)
	}
	if len(report.Errors) != 1 || report.Errors[0].Directory != blocked {
		t.Fatalf("errors = %+v", report.Errors)
	}
}
