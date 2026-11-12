package mods

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/thecoooler/blitz-client/internal/pathx"
)

func TestDiscoverModsInKeepsHealthyRootsAndReportsMissingOnes(t *testing.T) {
	root := t.TempDir()
	modsDir := filepath.Join(root, "mods")
	one := filepath.Join(modsDir, "one")
	other := filepath.Join(root, "other")
	for _, dir := range []string{one, other} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(one, "mod.yaml"), []byte("broken: [yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "mod.yaml"), []byte("id: two"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(root, "missing")

	report := DiscoverModsIn([]string{modsDir, one, other, missing})
	if len(report.Manifests) != 2 {
		t.Fatalf("manifests = %v", report.Manifests)
	}
	if !slices.IsSortedFunc(report.Manifests, pathx.Compare) {
		t.Fatalf("manifests are not sorted: %v", report.Manifests)
	}
	if len(report.Errors) != 1 || report.Errors[0].Directory != missing {
		t.Fatalf("errors = %+v", report.Errors)
	}
}

func TestDiscoverModsInCanonicalizesOverlappingRoots(t *testing.T) {
	root := t.TempDir()
	modDir := filepath.Join(root, "nested")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(modDir, "mod.yaml")
	if err := os.WriteFile(manifest, []byte("id: nested"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := DiscoverModsIn([]string{root, modDir})
	if len(report.Manifests) != 1 {
		t.Fatalf("manifests = %v", report.Manifests)
	}
	want, err := pathx.Canonicalize(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if report.Manifests[0] != want {
		t.Fatalf("manifest = %q, want %q", report.Manifests[0], want)
	}
}
