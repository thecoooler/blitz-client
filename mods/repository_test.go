package mods

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryURLs(t *testing.T) {
	cases := []struct{ raw, revision, archive string }{
		{"https://github.com/org/repo.git", "feature/ui", "https://api.github.com/repos/org/repo/zipball/feature%2Fui"},
		{"https://gitlab.com/org/group/repo", "main", "https://gitlab.com/api/v4/projects/org%2Fgroup%2Frepo/repository/archive.zip?sha=main"},
		{"https://example.com/mods.zip", "", "https://example.com/mods.zip"},
	}
	for _, c := range cases {
		r, err := NewGitRepository(c.raw, c.revision)
		if err != nil {
			t.Fatal(err)
		}
		archive, err := r.ArchiveURL()
		if err != nil || archive != c.archive {
			t.Fatalf("archive %s %v", archive, err)
		}
	}
	for _, raw := range []string{"http://github.com/org/repo", "https://user:pass@github.com/org/repo", "https://github.com/org/repo/tree/main", "https://github.com/org/repo#fragment", "https://example.com/repo", "https://gitlab.com/org/../repo"} {
		if _, err := NewGitRepository(raw, ""); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if _, err := NewGitRepository("https://example.com/mods.zip", "main"); err == nil {
		t.Fatal("ZIP revision accepted")
	}
}
func makeRepositoryZIP(t *testing.T, names []string) string {
	t.Helper()
	archive := filepath.Join(t.TempDir(), "mods.zip")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	for _, name := range names {
		item, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		item.Write([]byte("id: test\nname: Test\nfiles: []\n"))
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return archive
}
func TestRepositoryExtractionAndRejectedArchives(t *testing.T) {
	archive := makeRepositoryZIP(t, []string{"root/README.md", "root/mods/example/mod.yaml"})
	output := t.TempDir()
	if err := extractRepository(archive, output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(output, "mods/example/mod.yaml")); err != nil {
		t.Fatal(err)
	}
	for _, names := range [][]string{
		{"root/mods/example/mod.yaml", "../escape"},
		{"root/mods/example/mod.yaml", "root/mods/example/MOD.YAML"},
		{"root/mods/example/mod.yaml", "other/mods/second/mod.yaml"},
		{"root/mods/example/readme.txt"},
	} {
		if err := extractRepository(makeRepositoryZIP(t, names), t.TempDir()); err == nil {
			t.Fatalf("accepted unsafe archive %v", names)
		}
	}
}
func TestFailedRefreshPreservesCache(t *testing.T) {
	repo, _ := NewGitRepository("https://example.com/mods.zip", "")
	cache := RepositoryCache{Root: t.TempDir()}
	root, _ := cache.sourceDir(repo)
	os.MkdirAll(filepath.Join(root, "old/mods"), 0700)
	os.WriteFile(filepath.Join(root, "current"), []byte("old"), 0600)
	path, err := cache.CachedMods(repo)
	if err != nil || !strings.HasSuffix(path, filepath.Join("old", "mods")) {
		t.Fatalf("cache: %s %v", path, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = cache.Refresh(ctx, repo); err == nil {
		t.Fatal("canceled refresh succeeded")
	}
	// Corrupt extraction never touches the published generation.
	if err = extractRepository(makeRepositoryZIP(t, []string{"root/mods/broken/README"}), t.TempDir()); err == nil {
		t.Fatal("bad refresh succeeded")
	}
	path, err = cache.CachedMods(repo)
	if err != nil || !strings.HasSuffix(path, filepath.Join("old", "mods")) {
		t.Fatal("old cache lost")
	}
	os.WriteFile(filepath.Join(root, "current"), []byte("../escape"), 0600)
	if _, err = cache.CachedMods(repo); err == nil {
		t.Fatal("unsafe cache pointer accepted")
	}
}
