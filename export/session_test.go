package export

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// sessionTree creates the smallest game tree the shared session needs: a
// installation directory with Data and a separate packs directory.
func sessionTree(t *testing.T) (root, game, data, packs string) {
	t.Helper()
	root = t.TempDir()
	game = filepath.Join(root, "game")
	data = filepath.Join(game, "Data")
	packs = filepath.Join(root, "packs")
	for _, directory := range []string{data, packs} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root, game, data, packs
}

func canonical(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestOpenExportRootsResolvesExplicitCacheAndNewOutput(t *testing.T) {
	root, game, data, packs := sessionTree(t)
	output := filepath.Join(root, "new", "output")
	roots, err := openExportRoots(exportSessionOptions{
		GameDir:            game,
		CacheDir:           packs,
		OutputDir:          output,
		RequireEmptyOutput: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if roots.ResourceDir != canonical(t, data) {
		t.Fatalf("resource dir: %s", roots.ResourceDir)
	}
	if roots.CacheDir != canonical(t, packs) {
		t.Fatalf("cache dir: %s", roots.CacheDir)
	}
	if roots.OutputDir != filepath.Join(canonical(t, root), "new", "output") || roots.OutputExisted {
		t.Fatalf("output: %s existed=%v", roots.OutputDir, roots.OutputExisted)
	}
	if got := roots.sources(); len(got) != 2 || got[0] != roots.ResourceDir || got[1] != roots.CacheDir {
		t.Fatalf("sources: %v", got)
	}
}

func TestOpenExportRootsNoCacheDisablesOverlay(t *testing.T) {
	root, game, _, packs := sessionTree(t)
	roots, err := openExportRoots(exportSessionOptions{
		GameDir:   game,
		NoCache:   true,
		OutputDir: filepath.Join(root, "output"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if roots.CacheDir != "" {
		t.Fatalf("cache not disabled: %s", roots.CacheDir)
	}
	if got := roots.sources(); len(got) != 1 {
		t.Fatalf("sources: %v", got)
	}
	// The explicit path is ignored once NoCache wins the policy. The entry
	// points reject the combination before reaching the session.
	if cache, err := selectExportCache(&packs, packs, true); err != nil || cache != "" {
		t.Fatalf("no-cache selected %q: %v", cache, err)
	}
}

func TestSelectExportCacheUsesDefaultOnlyWhenItExists(t *testing.T) {
	_, _, _, packs := sessionTree(t)
	if cache, err := selectExportCache(&packs, "", false); err != nil || cache != canonical(t, packs) {
		t.Fatalf("existing default cache: %q %v", cache, err)
	}
	missing := filepath.Join(t.TempDir(), "packs")
	if cache, err := selectExportCache(&missing, "", false); err != nil || cache != "" {
		t.Fatalf("missing default cache used: %q %v", cache, err)
	}
	if cache, err := selectExportCache(&packs, "", true); err != nil || cache != "" {
		t.Fatalf("no-cache used default: %q %v", cache, err)
	}
}

func TestOpenExportRootsRejectsCacheConflict(t *testing.T) {
	for _, test := range []struct {
		cache   string
		noCache bool
		want    bool
	}{
		{"cache", true, true},
		{"cache", false, false},
		{"", true, false},
		{"", false, false},
	} {
		if got := exportCacheConflict(test.cache, test.noCache); got != test.want {
			t.Fatalf("exportCacheConflict(%q, %v) = %v, want %v", test.cache, test.noCache, got, test.want)
		}
	}
}

func TestOpenExportRootsRejectsMissingExplicitCache(t *testing.T) {
	root, game, _, _ := sessionTree(t)
	_, err := openExportRoots(exportSessionOptions{
		GameDir:   game,
		CacheDir:  filepath.Join(root, "missing-packs"),
		OutputDir: filepath.Join(root, "output"),
	})
	if errorCode(err) != "FILE_EXPORT_SOURCE_NOT_FOUND" {
		t.Fatal(err)
	}
}

func TestOpenExportRootsMissingOutputIsNew(t *testing.T) {
	root, game, _, _ := sessionTree(t)
	output := filepath.Join(root, "absent", "nested")
	roots, err := openExportRoots(exportSessionOptions{
		GameDir:            game,
		OutputDir:          output,
		RequireEmptyOutput: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if roots.OutputDir != filepath.Join(canonical(t, root), "absent", "nested") || roots.OutputExisted {
		t.Fatalf("output: %s existed=%v", roots.OutputDir, roots.OutputExisted)
	}
}

func TestOpenExportRootsRejectsOutputFile(t *testing.T) {
	root, game, _, _ := sessionTree(t)
	file := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(file, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := openExportRoots(exportSessionOptions{
		GameDir:   game,
		OutputDir: filepath.Join(file, "output"),
	})
	if errorCode(err) != "FILE_EXPORT_OUTPUT_INVALID" {
		t.Fatal(err)
	}
	if readString(t, file) != "keep me" {
		t.Fatal("output file changed")
	}
}

func TestOpenExportRootsRejectsOverlap(t *testing.T) {
	root, game, data, packs := sessionTree(t)
	for _, output := range []string{
		filepath.Join(data, "nested"),
		filepath.Join(packs, "nested"),
		root,
	} {
		_, err := openExportRoots(exportSessionOptions{
			GameDir:   game,
			CacheDir:  packs,
			OutputDir: output,
		})
		var overlap *outputOverlapError
		if !errors.As(err, &overlap) {
			t.Fatalf("%s: %v", output, err)
		}
	}
}

func TestOpenExportRootsRejectsOverlapThroughSymlinkedData(t *testing.T) {
	root, _, data, _ := sessionTree(t)
	install := filepath.Join(root, "install")
	if err := os.MkdirAll(install, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(data, filepath.Join(install, "Data")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	_, err := openExportRoots(exportSessionOptions{
		GameDir:   install,
		OutputDir: filepath.Join(data, "nested"),
	})
	var overlap *outputOverlapError
	if !errors.As(err, &overlap) {
		t.Fatalf("overlap through symlinked Data accepted: %v", err)
	}
}

func TestOpenExportRootsSnapshotRequiresEmptyOutput(t *testing.T) {
	root, game, _, _ := sessionTree(t)
	output := filepath.Join(root, "output")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "existing.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := openExportRoots(exportSessionOptions{
		GameDir:            game,
		OutputDir:          output,
		RequireEmptyOutput: true,
	})
	var notEmpty *outputNotEmptyError
	if !errors.As(err, &notEmpty) {
		t.Fatalf("non-empty output accepted: %v", err)
	}
	// Files export merges into the same directory instead.
	roots, err := openExportRoots(exportSessionOptions{
		GameDir:   game,
		OutputDir: output,
	})
	if err != nil || roots.OutputDir != canonical(t, output) {
		t.Fatalf("files export rejected existing output: %v", err)
	}
}
