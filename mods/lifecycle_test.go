package mods

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	blitz "github.com/thecoooler/blitz-client"
)

// Fixed DVPL bytes: expected output does not use the production encoder.
var (
	packedOriginal = []byte{
		240, 2, 112, 97, 99, 107, 101, 100, 58, 32, 111, 114, 105, 103, 105, 110, 97, 108, 10, 17, 0,
		0, 0, 19, 0, 0, 0, 33, 171, 11, 170, 2, 0, 0, 0, 68, 86, 80, 76,
	}
	packedInstalled = []byte{
		240, 3, 112, 97, 99, 107, 101, 100, 58, 32, 105, 110, 115, 116, 97, 108, 108, 101, 100, 10, 18,
		0, 0, 0, 20, 0, 0, 0, 218, 131, 7, 197, 2, 0, 0, 0, 68, 86, 80, 76,
	}
)

type scenario struct {
	t         *testing.T
	root      string
	originals map[string][]byte
}

func lifecyclePatch(path string, line int, before, after string) string {
	return fmt.Sprintf("diff --git a/%[1]s b/%[1]s\n--- a/%[1]s\n+++ b/%[1]s\n@@ -%[2]d +%[2]d @@\n-%[3]s\n+%[4]s\n", path, line, before, after)
}

func newScenario(t *testing.T) *scenario {
	s := &scenario{t: t, root: t.TempDir(), originals: map[string][]byte{
		"shared.yaml":      []byte("alpha: original\nseparator\nbeta: original\n"),
		"old-only.yaml":    []byte("old: original\n"),
		"new-only.yaml":    []byte("new: original\n"),
		"packed.yaml.dvpl": packedOriginal,
		"untouched.bin":    {0xff, 0x00, 0x80, 0xfe},
	}}
	os.MkdirAll(s.data(""), 0o755)
	for name, data := range s.originals {
		os.WriteFile(s.data(name), data, 0o644)
	}
	// Two versions share an id but change both the content and affected files.
	s.writeMod("alpha-v1", "Alpha", lifecyclePatch("shared.yaml", 1, "alpha: original", "alpha: v1")+
		lifecyclePatch("old-only.yaml", 1, "old: original", "old: v1"))
	s.writeMod("alpha-v2", "Alpha", lifecyclePatch("shared.yaml", 1, "alpha: original", "alpha: v2")+
		lifecyclePatch("new-only.yaml", 1, "new: original", "new: v2"))
	s.writeMod("beta", "Beta", lifecyclePatch("shared.yaml", 3, "beta: original", "beta: installed")+
		lifecyclePatch("packed.yaml", 1, "packed: original", "packed: installed"))
	return s
}

func (s *scenario) data(name string) string { return filepath.Join(s.root, "game/Data", name) }
func (s *scenario) game() string            { return filepath.Join(s.root, "game") }
func (s *scenario) mod(dir string) string   { return filepath.Join(s.root, "mods", dir) }
func (s *scenario) statePath() string {
	return filepath.Join(s.root, "game/.tanks-blitz-client/state.json")
}

func (s *scenario) read(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		s.t.Fatal(err)
	}
	return data
}

func (s *scenario) writeMod(dir, id, diff string) {
	path := s.mod(dir)
	os.MkdirAll(path, 0o755)
	os.WriteFile(filepath.Join(path, "mod.yaml"), []byte(fmt.Sprintf("id: %s\nname: %s\npatch: mod.patch\n", id, id)), 0o644)
	os.WriteFile(filepath.Join(path, "mod.patch"), []byte(diff), 0o644)
}

func (s *scenario) install(dir string) {
	s.t.Helper()
	report, err := InstallMod(s.mod(dir), s.game())
	if err != nil || !report.Installed {
		s.t.Fatalf("%s: %v", dir, err)
	}
}

func (s *scenario) remove(id string) {
	s.t.Helper()
	if report, err := RemoveMod(id, s.game()); err != nil || !report.Removed {
		s.t.Fatalf("%s: %v", id, err)
	}
}

func (s *scenario) assertFiles(alpha int, beta bool) {
	s.t.Helper()
	state := func(version int, values ...string) string { return values[version] }
	betaText := "original"
	if beta {
		betaText = "installed"
	}
	expected := map[string]string{
		"shared.yaml":   fmt.Sprintf("alpha: %s\nseparator\nbeta: %s\n", state(alpha, "original", "v1", "v2"), betaText),
		"old-only.yaml": fmt.Sprintf("old: %s\n", state(alpha, "original", "v1", "original")),
		"new-only.yaml": fmt.Sprintf("new: %s\n", state(alpha, "original", "original", "v2")),
	}
	for name, text := range expected {
		if got := string(s.read(s.data(name))); got != text {
			s.t.Fatalf("%s: Alpha=%d Beta=%v: %q", name, alpha, beta, got)
		}
	}
	packed := packedOriginal
	if beta {
		packed = packedInstalled
	}
	if !bytes.Equal(s.read(s.data("packed.yaml.dvpl")), packed) {
		s.t.Fatalf("packed.yaml.dvpl: Alpha=%d Beta=%v", alpha, beta)
	}
	if !bytes.Equal(s.read(s.data("untouched.bin")), s.originals["untouched.bin"]) {
		s.t.Fatal("untouched.bin changed")
	}
	// Complete removal must restore original bytes, including compression.
	if alpha == 0 && !beta {
		for name, original := range s.originals {
			if !bytes.Equal(s.read(s.data(name)), original) {
				s.t.Fatal(name)
			}
		}
	}
	if (alpha != 0 || beta) && !isFile(s.statePath()) {
		s.t.Fatal("installed mods must have saved state")
	}
	if isFile(s.statePath()) {
		var stored struct {
			Mods []struct {
				ID string `json:"id"`
			} `json:"mods"`
		}
		json.Unmarshal(s.read(s.statePath()), &stored)
		ids := []string{}
		for _, item := range stored.Mods {
			ids = append(ids, item.ID)
		}
		slices.Sort(ids)
		var expectedIDs []string
		if alpha != 0 {
			expectedIDs = append(expectedIDs, "Alpha")
		}
		if beta {
			expectedIDs = append(expectedIDs, "Beta")
		}
		if !slices.Equal(ids, expectedIDs) {
			s.t.Fatal(ids)
		}
	}
}

func code(err error) string {
	if typed := blitz.AsError(err); typed != nil {
		return typed.Code
	}
	return ""
}

func TestModVersionsSurviveRepeatedInstallRemoveAndRecomposition(t *testing.T) {
	s := newScenario(t)
	s.assertFiles(0, false)
	s.install("alpha-v1")
	s.assertFiles(1, false)
	s.install("beta")
	s.assertFiles(1, true)
	s.remove("Alpha")
	s.assertFiles(0, true)
	s.install("alpha-v2")
	s.assertFiles(2, true)
	s.remove("Beta")
	s.assertFiles(2, false)
	s.install("beta")
	s.assertFiles(2, true)
	// Removing source files must not affect subsequent recomposition.
	os.RemoveAll(s.mod("alpha-v2"))
	s.remove("Alpha")
	s.assertFiles(0, true)
	s.install("alpha-v1")
	s.assertFiles(1, true)
	s.remove("Beta")
	s.assertFiles(1, false)
	s.remove("Alpha")
	s.assertFiles(0, false)
	// Start again in reverse installation order after all snapshots were released.
	s.install("beta")
	s.assertFiles(0, true)
	s.install("alpha-v1")
	s.assertFiles(1, true)
	s.remove("Alpha")
	s.assertFiles(0, true)
	s.remove("Beta")
	s.assertFiles(0, false)
}

func TestInvalidNewVersionDoesNotPartiallyInstallOrChangeOtherMods(t *testing.T) {
	s := newScenario(t)
	s.install("alpha-v1")
	s.install("beta")
	s.remove("Alpha")
	s.assertFiles(0, true)
	before := s.read(s.statePath())
	// The shared resource can be prepared, but the next resource cannot match.
	s.writeMod("alpha-v2", "Alpha", lifecyclePatch("shared.yaml", 1, "alpha: original", "alpha: v2")+
		lifecyclePatch("new-only.yaml", 1, "missing context", "new: v2"))
	if _, err := InstallMod(s.mod("alpha-v2"), s.game()); code(err) != "PATCH_CONTEXT_MISMATCH" {
		t.Fatal(err)
	}
	s.assertFiles(0, true)
	if !bytes.Equal(s.read(s.statePath()), before) {
		t.Fatal("state changed")
	}
	// A failed attempt must not prevent a later installation of the old version.
	s.install("alpha-v1")
	s.assertFiles(1, true)
	s.remove("Beta")
	s.assertFiles(1, false)
	s.remove("Alpha")
	s.assertFiles(0, false)
}

func TestReinstallReplacesVersionsAndPreservesOtherMods(t *testing.T) {
	s := newScenario(t)
	if listed, err := ListInstalledMods(s.game()); err != nil || len(listed.Mods) != 0 {
		t.Fatal(err)
	}
	s.install("beta")
	s.install("alpha-v1")
	s.assertFiles(1, true)
	if report, err := ValidateMod(s.mod("alpha-v2"), s.game()); err != nil || !report.Valid {
		t.Fatal(err)
	}
	for _, step := range []struct {
		dir     string
		version int
	}{{"alpha-v2", 2}, {"alpha-v2", 2}, {"alpha-v1", 1}, {"alpha-v2", 2}} {
		report, err := InstallMod(s.mod(step.dir), s.game())
		if err != nil || !report.Installed || !report.Reinstalled {
			t.Fatal(err)
		}
		s.assertFiles(step.version, true)
		listed, _ := ListInstalledMods(s.game())
		if listed.Mods[0].ID != "Alpha" || listed.Mods[1].ID != "Beta" {
			t.Fatalf("%+v", listed.Mods)
		}
		expected := []string{"new-only.yaml", "shared.yaml"}
		if step.version == 1 {
			expected = []string{"old-only.yaml", "shared.yaml"}
		}
		if !reflect.DeepEqual(listed.Mods[0].Files, expected) {
			t.Fatal(listed.Mods[0].Files)
		}
	}
	os.RemoveAll(filepath.Join(s.root, "mods"))
	listed, _ := ListInstalledMods(s.game())
	if len(listed.Mods) != 2 || listed.Mods[1].Name != "Beta" || !reflect.DeepEqual(listed.Mods[1].Files, []string{"packed.yaml", "shared.yaml"}) {
		t.Fatalf("%+v", listed.Mods)
	}
	s.remove("Beta")
	s.assertFiles(2, false)
	s.remove("Alpha")
	s.assertFiles(0, false)
	if listed, _ := ListInstalledMods(s.game()); len(listed.Mods) != 0 {
		t.Fatal(listed.Mods)
	}
}

func TestFailedReinstallPreservesOldVersionAndSavedState(t *testing.T) {
	s := newScenario(t)
	s.install("alpha-v1")
	s.install("beta")
	before := s.read(s.statePath())
	s.writeMod("alpha-v2", "Alpha", lifecyclePatch("shared.yaml", 1, "alpha: original", "alpha: v2")+
		lifecyclePatch("new-only.yaml", 1, "missing", "new: v2"))
	if _, err := InstallMod(s.mod("alpha-v2"), s.game()); code(err) != "PATCH_CONTEXT_MISMATCH" {
		t.Fatal(err)
	}
	s.assertFiles(1, true)
	if !bytes.Equal(s.read(s.statePath()), before) {
		t.Fatal("state changed")
	}
	s.remove("Alpha")
	s.assertFiles(0, true)
	s.remove("Beta")
	s.assertFiles(0, false)
}

func TestReinstallRejectsExternalChangesToOldOnlyFiles(t *testing.T) {
	s := newScenario(t)
	s.install("alpha-v1")
	s.install("beta")
	target := s.data("old-only.yaml")
	original := s.read(target)
	before := s.read(s.statePath())
	os.WriteFile(target, []byte("changed outside installer\n"), 0o644)
	if _, err := InstallMod(s.mod("alpha-v2"), s.game()); code(err) != "INSTALLED_FILE_CHANGED" {
		t.Fatal(err)
	}
	if report, err := ValidateMod(s.mod("alpha-v2"), s.game()); err != nil || report.Valid {
		t.Fatal(err)
	}
	if string(s.read(target)) != "changed outside installer\n" || !bytes.Equal(s.read(s.statePath()), before) {
		t.Fatal("files changed")
	}
	os.WriteFile(target, original, 0o644)
	s.assertFiles(1, true)
}

func TestDroppingAPatchRequiredByAnotherModKeepsOldInstallation(t *testing.T) {
	s := newScenario(t)
	s.install("alpha-v1")
	s.writeMod("beta", "Beta", lifecyclePatch("old-only.yaml", 1, "old: v1", "old: v1 with beta"))
	s.install("beta")
	before := map[string][]byte{}
	for name := range s.originals {
		before[name] = s.read(s.data(name))
	}
	stateBefore := s.read(s.statePath())
	validation, err := ValidateMod(s.mod("alpha-v2"), s.game())
	if err != nil || validation.Valid {
		t.Fatal(err)
	}
	found := false
	for _, file := range validation.Files {
		found = found || (file.Path == "old-only.yaml" && !file.Valid)
	}
	if !found {
		t.Fatalf("%+v", validation.Files)
	}
	if _, err := InstallMod(s.mod("alpha-v2"), s.game()); code(err) != "PATCH_CONTEXT_MISMATCH" {
		t.Fatal(err)
	}
	for name, data := range before {
		if !bytes.Equal(s.read(s.data(name)), data) {
			t.Fatal(name)
		}
	}
	if !bytes.Equal(s.read(s.statePath()), stateBefore) {
		t.Fatal("state changed")
	}
	s.remove("Beta")
	s.assertFiles(1, false)
	s.remove("Alpha")
	s.assertFiles(0, false)
}

func TestDiscoveryReturnsPathsEvenForBrokenModsAndConflictingManifests(t *testing.T) {
	s := newScenario(t)
	root := filepath.Join(s.root, "mods")
	empty := filepath.Join(s.root, "empty")
	os.MkdirAll(empty, 0o755)
	if paths, err := DiscoverMods(empty); err != nil || len(paths) != 0 {
		t.Fatal(paths, err)
	}
	bad := filepath.Join(root, "nested/broken")
	os.MkdirAll(bad, 0o755)
	os.WriteFile(filepath.Join(bad, "mod.yaml"), []byte("files: [broken"), 0o644)
	os.WriteFile(filepath.Join(bad, "mod.yml"), []byte("id: Other\nname: Other\n"), 0o644)
	paths, err := DiscoverMods(root)
	expected := []string{
		filepath.Join(root, "alpha-v1/mod.yaml"), filepath.Join(root, "alpha-v2/mod.yaml"),
		filepath.Join(root, "beta/mod.yaml"), filepath.Join(bad, "mod.yaml"),
	}
	if err != nil || !reflect.DeepEqual(paths, expected) {
		t.Fatal(paths, err)
	}
	if _, err := LoadManifest(paths[len(paths)-1]); err == nil {
		t.Fatal("conflicting manifests loaded")
	}
	os.Remove(filepath.Join(bad, "mod.yml"))
	if _, err := LoadManifest(filepath.Join(bad, "mod.yaml")); code(err) != "MANIFEST_INVALID" {
		t.Fatal(err)
	}
	if manifest, err := LoadManifest(paths[0]); err != nil || manifest.ID != "Alpha" {
		t.Fatal(err)
	}
	if _, err := DiscoverMods(filepath.Join(s.root, "missing")); err == nil {
		t.Fatal("missing directory discovered")
	}
	if os.Symlink(root, filepath.Join(bad, "cycle")) == nil {
		if again, err := DiscoverMods(root); err != nil || !reflect.DeepEqual(again, paths) {
			t.Fatal(again, err)
		}
	}
}

func TestReinstallRestoresDroppedBinaryClearFromOriginalBytes(t *testing.T) {
	s := newScenario(t)
	s.install("beta")
	os.WriteFile(filepath.Join(s.mod("alpha-v1"), "mod.yaml"), []byte("id: Alpha\nname: Alpha\nfiles:\n  untouched.bin:\n    - clear: {}\n"), 0o644)
	s.install("alpha-v1")
	if len(s.read(s.data("untouched.bin"))) != 0 {
		t.Fatal("clear not applied")
	}
	if report, err := InstallMod(s.mod("alpha-v2"), s.game()); err != nil || !report.Reinstalled {
		t.Fatal(err)
	}
	s.assertFiles(2, true)
	s.remove("Alpha")
	s.assertFiles(0, true)
	s.remove("Beta")
	s.assertFiles(0, false)
}
