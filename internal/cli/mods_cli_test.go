package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/mods"
	"github.com/thecoooler/blitz-client/resources"
)

type modFixture struct {
	t    *testing.T
	root string
}

func newModFixture(t *testing.T) *modFixture {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "game/Data/config.yaml"), "original\n")
	os.MkdirAll(filepath.Join(root, "mods"), 0o755)
	return &modFixture{t, root}
}

func (f *modFixture) path(relative string) string { return filepath.Join(f.root, relative) }
func (f *modFixture) game() string                { return f.path("game") }

func (f *modFixture) manifest(dir, id, find string) {
	writeFile(f.t, filepath.Join(f.root, "mods", dir, "mod.yaml"), fmt.Sprintf(
		"id: %s\nname: %s\nfiles:\n  config.yaml:\n    - replace:\n        search: %s\n        replace: \"changed\"\n",
		strconv.Quote(id), strconv.Quote(id), strconv.Quote(find)))
}

func (f *modFixture) validate(path string, json bool, env ...string) result {
	args := []string{"mod", "validate", path, "--game-dir", f.game()}
	if json {
		args = append(args, "--json")
	}
	return run(f.t, env, args...)
}

func errorCode(err error) string {
	if typed := blitz.AsError(err); typed != nil {
		return typed.Code
	}
	return ""
}

func TestRecursivelyValidatesRootAndNestedModsWithoutWritingResources(t *testing.T) {
	f := newModFixture(t)
	f.manifest("", "Root", "original")
	f.manifest("group/deep", "Nested", "original")
	result := f.validate(f.path("mods"), true)
	value := result.json(t)
	if result.code != 0 || value["valid"] != true || len(value["mods"].([]any)) != 2 {
		t.Fatalf("%s", result.stdout)
	}
	for _, item := range value["mods"].([]any) {
		if get(item, "report", "valid") != true {
			t.Fatal(item)
		}
	}
	if readFile(t, f.path("game/Data/config.yaml")) != "original\n" || exists(f.path("game/.tanks-blitz-client")) {
		t.Fatal("validation wrote files")
	}
	if text := f.validate(f.path("mods"), false); !strings.Contains(string(text.stdout), "2 mods · 2 passed · 0 failed") {
		t.Fatalf("%s", text.stdout)
	}
}

func TestCollectionContinuesAfterBadManifestAndFailedResource(t *testing.T) {
	f := newModFixture(t)
	f.manifest("a-invalid", "Invalid", "original")
	writeFile(t, f.path("mods/a-invalid/mod.yaml"), "files: [invalid")
	f.manifest("b-failed", "Failed", "missing text")
	f.manifest("z-valid", "Valid", "original")
	result := f.validate(f.path("mods"), true)
	value := result.json(t)
	if result.code != 4 || value["valid"] != false || get(value, "mods", 0, "error", "code") != "MANIFEST_INVALID" ||
		get(value, "mods", 1, "report", "files", 0, "error_code") != "REPLACE_MATCH_COUNT" ||
		get(value, "mods", 2, "report", "valid") != true {
		t.Fatalf("%d %s", result.code, result.stdout)
	}
}

func TestSingleModAndExplicitManifestKeepExistingReport(t *testing.T) {
	f := newModFixture(t)
	f.manifest("single", "Single", "original")
	for _, path := range []string{f.path("mods/single"), f.path("mods/single/mod.yaml")} {
		result := f.validate(path, true)
		value := result.json(t)
		if result.code != 0 || value["mod_id"] != "Single" || value["valid"] != true || value["mods"] != nil {
			t.Fatalf("%s", result.stdout)
		}
	}
	if value := f.validate(f.path("mods"), true).json(t); len(value["mods"].([]any)) != 1 {
		t.Fatal(value)
	}
}

func TestEmptyDirectoryIsAnError(t *testing.T) {
	f := newModFixture(t)
	result := f.validate(f.path("mods"), true)
	if result.code != 2 || get(result.json(t), "error", "code") != "MANIFEST_NOT_FOUND" {
		t.Fatalf("%s", result.stdout)
	}
}

func TestDirectorySymlinkCyclesAreNotFollowed(t *testing.T) {
	f := newModFixture(t)
	f.manifest("single", "Single", "original")
	if err := os.Symlink(f.path("mods"), f.path("mods/loop")); err != nil {
		t.Skip(err)
	}
	result := f.validate(f.path("mods"), true)
	if result.code != 0 || len(result.json(t)["mods"].([]any)) != 1 {
		t.Fatalf("%s", result.stdout)
	}
}

func TestInsertActionsRoundTripAndComposeWithOtherMods(t *testing.T) {
	f := newModFixture(t)
	for _, item := range [][3]string{{"A", "before", "prefix:"}, {"B", "after", ":suffix"}} {
		f.manifest(item[0], item[0], "original")
		manifest := f.path("mods/" + item[0] + "/mod.yaml")
		text := strings.ReplaceAll(readFile(t, manifest), "- replace:", "- "+item[1]+":")
		text = strings.ReplaceAll(text, `replace: "changed"`, "replace: "+strconv.Quote(item[2]))
		writeFile(t, manifest, text)
		if report, err := mods.ValidateMod(manifest, f.game()); err != nil || !report.Valid {
			t.Fatal(err)
		}
		if _, err := mods.InstallMod(manifest, f.game()); err != nil {
			t.Fatal(err)
		}
	}
	target := f.path("game/Data/config.yaml")
	if readFile(t, target) != "prefix:original:suffix\n" {
		t.Fatal(readFile(t, target))
	}
	mods.RemoveMod("A", f.game())
	if readFile(t, target) != "original:suffix\n" {
		t.Fatal(readFile(t, target))
	}
	mods.RemoveMod("B", f.game())
	if readFile(t, target) != "original\n" {
		t.Fatal(readFile(t, target))
	}
}

func TestInsertActionsRequireOneExactMatchAndRequiredFields(t *testing.T) {
	f := newModFixture(t)
	for _, action := range []string{"before", "after"} {
		for _, find := range []string{"missing", "i"} {
			f.manifest(action, action, find)
			manifest := f.path("mods/" + action + "/mod.yaml")
			text := strings.ReplaceAll(readFile(t, manifest), "- replace:", "- "+action+":")
			writeFile(t, manifest, text)
			report, err := mods.ValidateMod(manifest, f.game())
			if err != nil || report.Valid || *report.Files[0].ErrorCode != "REPLACE_MATCH_COUNT" {
				t.Fatal(action, find, err)
			}
			writeFile(t, manifest, strings.ReplaceAll(text, `replace: "changed"`, ""))
			if _, err := mods.ValidateMod(manifest, f.game()); errorCode(err) != "MANIFEST_INVALID_ACTION" {
				t.Fatal(err)
			}
		}
	}
}

func TestClearHandlesPlainAndDVPLBinaryResourcesAndRestoresOriginalBytes(t *testing.T) {
	f := newModFixture(t)
	binary := []byte{0xff, 0x00, 0xfe, 0x80}
	for index, kind := range []resources.ContainerKind{resources.Plain, resources.DVPL(0), resources.DVPL(1), resources.DVPL(2)} {
		name := fmt.Sprintf("image%d.webp", index)
		target := f.path("game/Data/" + name)
		if kind.DVPL {
			target += ".dvpl"
		}
		original, _ := resources.EncodeDVPL(binary, kind)
		os.WriteFile(target, original, 0o644)
		dir := f.path(fmt.Sprintf("mods/Clear%d", index))
		writeFile(t, filepath.Join(dir, "mod.yaml"), fmt.Sprintf("id: Clear%d\nname: Clear\nfiles:\n  %s:\n    - clear: {}\n", index, strconv.Quote(name)))
		if report, err := mods.ValidateMod(dir, f.game()); err != nil || !report.Valid {
			t.Fatal(err)
		}
		if readFile(t, target) != string(original) {
			t.Fatal("validation changed target")
		}
		if _, err := mods.InstallMod(dir, f.game()); err != nil {
			t.Fatal(err)
		}
		payload, outputKind, err := resources.DecodeDVPL([]byte(readFile(t, target)))
		if err != nil || len(payload) != 0 || outputKind != kind {
			t.Fatal(err, payload, outputKind)
		}
		mods.RemoveMod(fmt.Sprintf("Clear%d", index), f.game())
		if readFile(t, target) != string(original) {
			t.Fatal("not restored")
		}
	}
}

func TestYAMLNormalizedReplacePreservesOtherBytesAndReplacement(t *testing.T) {
	f := newModFixture(t)
	target := f.path("game/Data/config.yaml")
	original := "untouched:  \r\n    first: \t\r\n        second:  \r\nlast: \t\r\n"
	writeFile(t, target, original)
	manifest := f.path("mods/mod.yaml")
	writeFile(t, manifest, `version: 1
id: Normalized
name: Normalized
files:
  config.yaml:
    - replace:
        ignore-trailing-whitespace: true
        search: |2
              first:
                  second:
        replace: "    replaced:  \n"
`)
	if report, err := mods.ValidateMod(manifest, f.game()); err != nil || !report.Valid {
		t.Fatal(err)
	}
	if cli := f.validate(f.path("mods"), true); cli.code != 0 {
		t.Fatalf("%s", cli.stdout)
	}
	mods.InstallMod(manifest, f.game())
	if readFile(t, target) != "untouched:  \r\n    replaced:  \nlast: \t\r\n" {
		t.Fatalf("%q", readFile(t, target))
	}
	mods.RemoveMod("Normalized", f.game())
	if readFile(t, target) != original {
		t.Fatal("not restored")
	}
}

func TestNormalizationIsOptInAndKeepsIndentationAndCase(t *testing.T) {
	f := newModFixture(t)
	target := f.path("game/Data/config.yaml")
	dir := f.path("mods/single")
	f.manifest("single", "Single", "original")
	manifest := filepath.Join(dir, "mod.yaml")
	text := "id: Single\nname: Single\nfiles:\n  config.yaml:\n    - replace:\n        search: \"    first:\\n        second:\\n\"\n        replace: ''\n"
	writeFile(t, target, "    first: \r\n        second:\t\r\n")
	writeFile(t, manifest, text)
	if report, _ := mods.ValidateMod(dir, f.game()); report.Valid {
		t.Fatal("exact match should fail")
	}
	writeFile(t, manifest, text+"        ignore-trailing-whitespace: true\n")
	if report, err := mods.ValidateMod(dir, f.game()); err != nil || !report.Valid {
		t.Fatal(err)
	}
	for _, mismatch := range []string{
		"    first: \r\n    second:\t\r\n",
		"    First: \r\n        second:\t\r\n",
		"    first: \r\n\r\n        second:\t\r\n",
		"    first: \r\n        second:\t\r\n    first:\n        second:\n",
	} {
		writeFile(t, target, mismatch)
		if report, _ := mods.ValidateMod(dir, f.game()); report.Valid {
			t.Fatalf("%q matched", mismatch)
		}
	}
}

func TestYAMLOrderedInsertionsPreserveMatchedWhitespaceAndSurviveRecomposition(t *testing.T) {
	f := newModFixture(t)
	original := "original \t\r\n"
	target := f.path("game/Data/config.yaml")
	writeFile(t, target, original)
	dir := f.path("mods/group/deep")
	writeFile(t, filepath.Join(dir, "mod.yml"), `id: A
name: Insertions
files:
  config.yaml:
    - before:
        ignore-trailing-whitespace: true
        search: "original\n"
        insert: "prefix\n"
    - after:
        ignore-trailing-whitespace: true
        search: "original\n"
        insert: "suffix\n"
`)
	if report, err := mods.ValidateMod(dir, f.game()); err != nil || !report.Valid {
		t.Fatal(err)
	}
	if f.validate(f.path("mods"), true).code != 0 {
		t.Fatal("collection validation failed")
	}
	if report, err := mods.InstallMod(dir, f.game()); err != nil || len(report.Targets) != 1 {
		t.Fatal(err)
	}
	if readFile(t, target) != "prefix\noriginal \t\r\nsuffix\n" {
		t.Fatalf("%q", readFile(t, target))
	}
	f.manifest("B", "B", "suffix")
	mods.InstallMod(f.path("mods/B"), f.game())
	if readFile(t, target) != "prefix\noriginal \t\r\nchanged\n" {
		t.Fatalf("%q", readFile(t, target))
	}
	mods.RemoveMod("B", f.game())
	if readFile(t, target) != "prefix\noriginal \t\r\nsuffix\n" {
		t.Fatalf("%q", readFile(t, target))
	}
	if report, err := mods.RemoveMod("A", f.game()); err != nil || len(report.RestoredFiles) != 1 || readFile(t, target) != original {
		t.Fatal(err)
	}
}

func TestYAMLRejectsAmbiguousNamesInvalidFlagsAndUnknownFields(t *testing.T) {
	f := newModFixture(t)
	dir := f.path("mods")
	manifest := filepath.Join(dir, "mod.yaml")
	base := "id: Test\nname: Test\nfiles:\n  config.yaml:\n    - replace:\n        search: original\n        replace: changed\n"
	for _, invalid := range []string{
		base + "        ignore-trailing-whitespace: yesplease\n",
		base + "        ignore-trailing-whitespac: true\n",
		strings.ReplaceAll(base, "search: original", "search: '   '") + "        ignore-trailing-whitespace: true\n",
		"id: Test\nname: Test\nfiles:\n  config.yaml:\n    - clear:\n        ignore-trailing-whitespace: true\n",
	} {
		writeFile(t, manifest, invalid)
		if _, err := mods.LoadManifest(dir); err == nil {
			t.Fatalf("%q loaded", invalid)
		}
	}
	writeFile(t, manifest, base)
	writeFile(t, filepath.Join(dir, "mod.yml"), base)
	if _, err := mods.LoadManifest(dir); errorCode(err) != "MANIFEST_AMBIGUOUS" {
		t.Fatal(err)
	}
	result := f.validate(dir, true)
	if result.code != 2 || get(result.json(t), "error", "code") != "MANIFEST_AMBIGUOUS" {
		t.Fatalf("%s", result.stdout)
	}
}

func TestOlderInstalledStateWithoutWhitespaceFlagCanBeRemoved(t *testing.T) {
	f := newModFixture(t)
	f.manifest("Legacy", "Legacy", "original")
	if _, err := mods.InstallMod(f.path("mods/Legacy"), f.game()); err != nil {
		t.Fatal(err)
	}
	statePath := f.path("game/.tanks-blitz-client/state.json")
	var state map[string]any
	json.Unmarshal([]byte(readFile(t, statePath)), &state)
	for _, file := range get(state, "mods", 0, "files").([]any) {
		delete(file.(map[string]any)["operation"].(map[string]any), "ignore_trailing_whitespace")
	}
	data, _ := json.Marshal(state)
	os.WriteFile(statePath, data, 0o644)
	if _, err := mods.RemoveMod("Legacy", f.game()); err != nil || readFile(t, f.path("game/Data/config.yaml")) != "original\n" {
		t.Fatal(err)
	}
}

func git(t *testing.T, repo string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-c", "user.name=Test", "-c", "user.email=test@example.invalid"}, args...)...)
	command.Dir = repo
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, output)
	}
}

func TestYAMLGitUpdatePreservesManualOperationsAndRefreshesGeneratedPatches(t *testing.T) {
	f := newModFixture(t)
	repo := f.path("repo")
	os.MkdirAll(repo, 0o755)
	git(t, repo, "init", "--quiet")
	writeFile(t, filepath.Join(repo, "config.yaml"), "original\n")
	writeFile(t, filepath.Join(repo, "mod.yaml"), "name: Original\n")
	writeFile(t, filepath.Join(repo, "extra resource.yaml"), "extra\n")
	git(t, repo, "add", "config.yaml", "mod.yaml", "extra resource.yaml")
	git(t, repo, "commit", "--quiet", "-m", "base")
	dir := f.path("mods")
	manifest := filepath.Join(dir, "mod.yaml")
	writeFile(t, manifest, "version: 1\nid: Git\nname: Git\nfiles:\n  \"manual.yaml\":\n    - replace:\n        ignore-trailing-whitespace: true\n        search: 'manual'\n        replace: 'changed'\n")
	writeFile(t, f.path("game/Data/manual.yaml"), "manual \n")
	writeFile(t, filepath.Join(repo, "mod.yaml"), "name: Updated\n")
	writeFile(t, filepath.Join(repo, "config.yaml"), "updated\n")
	writeFile(t, filepath.Join(repo, "extra resource.yaml"), "updated extra\n")
	writeFile(t, f.path("game/Data/extra resource.yaml"), "extra\n")
	first, err := mods.UpdateFromGitDiff(dir, repo, nil)
	if err != nil || len(first.Files) != 3 {
		t.Fatal(err)
	}
	generated := readFile(t, manifest)
	if !strings.Contains(generated, "\npatch: mod.patch\n") || strings.Contains(generated, "\n  config.yaml:\n") ||
		!strings.Contains(generated, "\n  manual.yaml:\n") {
		t.Fatal(generated)
	}
	var oldPatch string
	for _, file := range first.Files {
		if file.Patch != nil {
			oldPatch = filepath.Join(dir, *file.Patch)
			break
		}
	}
	if !exists(oldPatch) {
		t.Fatal(oldPatch)
	}
	if report, err := mods.ValidateMod(dir, f.game()); err != nil || !report.Valid {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "config.yaml"), "updated again\n")
	if _, err := mods.UpdateFromGitDiff(manifest, repo, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, manifest), "\npatch: mod.patch\n") || !strings.Contains(readFile(t, oldPatch), "+updated again") {
		t.Fatal(readFile(t, manifest))
	}
	loaded, err := mods.LoadManifest(dir)
	if err != nil || len(loaded.Files) != 3 {
		t.Fatal(err)
	}
	for _, file := range loaded.Files {
		if file.Name == "manual.yaml" && !file.IgnoreTrailingWhitespace {
			t.Fatal("manual flag lost")
		}
	}
	if report, err := mods.ValidateMod(dir, f.game()); err != nil || !report.Valid {
		t.Fatal(err)
	}
}

func TestTOMLManifestsAreNotLoadedOrDiscovered(t *testing.T) {
	f := newModFixture(t)
	dir := f.path("mods")
	legacy := filepath.Join(dir, "mod.toml")
	writeFile(t, legacy, "id = 'Legacy'\nname = 'Legacy'\n")
	if _, err := mods.LoadManifest(legacy); errorCode(err) != "MANIFEST_FORMAT_UNSUPPORTED" {
		t.Fatal(err)
	}
	if _, err := mods.LoadManifest(dir); errorCode(err) != "MANIFEST_NOT_FOUND" {
		t.Fatal(err)
	}
	result := f.validate(dir, true)
	if result.code != 2 || get(result.json(t), "error", "code") != "MANIFEST_NOT_FOUND" {
		t.Fatalf("%s", result.stdout)
	}
}

func TestValidationTextIsCompactAndDeduplicatesFileErrors(t *testing.T) {
	f := newModFixture(t)
	f.manifest("single", "Single", "original")
	if single := f.validate(f.path("mods/single"), false); string(single.stdout) != "✓ Single (1 file)\n" {
		t.Fatalf("%q", single.stdout)
	}
	writeFile(t, f.path("mods/failed/mod.yaml"), "id: Failed\nname: Failed\nfiles:\n  config.yaml:\n    - replace:\n        search: missing\n        replace: first\n    - replace:\n        search: missing\n        replace: second\n")
	result := f.validate(f.path("mods"), false)
	text := string(result.stdout)
	if result.code != 4 || !strings.Contains(text, "✗ Failed (1 error)\n  config.yaml\n  MOD_CONFLICT:") ||
		strings.Count(text, "config.yaml") != 1 || !strings.Contains(text, "2 mods · 1 passed · 1 failed") ||
		strings.Contains(text, f.root) || strings.Contains(text, "\x1b") {
		t.Fatalf("%d %s", result.code, text)
	}
}

func TestValidationColorsRespectPipesNoColorAndJSON(t *testing.T) {
	f := newModFixture(t)
	f.manifest("single", "Single", "missing")
	validate := func(force, noColor, json bool) result {
		var env []string
		if force {
			env = append(env, "CLICOLOR_FORCE=1")
		}
		if noColor {
			env = append(env, "NO_COLOR=1")
		}
		return f.validate(f.path("mods/single"), json, env...)
	}
	if strings.Contains(string(validate(false, false, false).stdout), "\x1b") {
		t.Fatal("colored pipe output")
	}
	if !strings.Contains(string(validate(true, false, false).stdout), "\x1b[31m✗\x1b[0m") {
		t.Fatal("forced color missing")
	}
	if strings.Contains(string(validate(true, true, false).stdout), "\x1b") {
		t.Fatal("NO_COLOR ignored")
	}
	jsonResult := validate(true, false, true)
	if strings.Contains(string(jsonResult.stdout), "\x1b") || jsonResult.json(t)["valid"] != false {
		t.Fatalf("%s", jsonResult.stdout)
	}
	writeFile(t, f.path("mods/single/mod.yaml"), "files: [invalid")
	failure := validate(true, false, false)
	if failure.code != 2 || !strings.Contains(string(failure.stderr), "\x1b[31mMANIFEST_INVALID\x1b[0m") {
		t.Fatalf("%d %q", failure.code, failure.stderr)
	}
}

func TestValidationGroupsParseAndResourceFailuresBelowSuccesses(t *testing.T) {
	f := newModFixture(t)
	f.manifest("a-invalid", "Invalid", "original")
	writeFile(t, f.path("mods/a-invalid/mod.yaml"), "files: [invalid")
	f.manifest("m-failed", "Failed", "missing")
	f.manifest("z-valid", "Valid", "original")
	result := f.validate(f.path("mods"), false)
	text := string(result.stdout)
	positions := []int{
		strings.Index(text, "✓ Valid"), strings.Index(text, "Failures (2)"), strings.Index(text, "✗ a-invalid"),
		strings.Index(text, "✗ Failed"), strings.Index(text, "3 mods · 1 passed · 2 failed"),
	}
	for index := 1; index < len(positions); index++ {
		if positions[index-1] < 0 || positions[index-1] >= positions[index] {
			t.Fatalf("%s", text)
		}
	}
	if result.code != 4 || strings.Count(text, "MANIFEST_INVALID") != 1 || strings.Count(text, "REPLACE_MATCH_COUNT") != 1 {
		t.Fatalf("%s", text)
	}
	report := f.validate(f.path("mods"), true).json(t)
	if get(report, "mods", 0, "error", "code") != "MANIFEST_INVALID" || get(report, "mods", 2, "report", "mod_id") != "Valid" {
		t.Fatal(report)
	}
	f.manifest("z-valid", "Valid", "missing")
	text = string(f.validate(f.path("mods"), false).stdout)
	if !strings.HasPrefix(text, "Failures (3)\n") || !strings.Contains(text, "3 mods · 0 passed · 3 failed") {
		t.Fatalf("%s", text)
	}
}

func TestGitPatchWithContextAndRepeatedHunksRoundTrips(t *testing.T) {
	f := newModFixture(t)
	target := f.path("game/Data/config.yaml")
	original := "header\ncontext\nold\nend\nseparator\ncontext\nold\nend\nfooter\n"
	writeFile(t, target, original)
	dir := f.path("mods/Patched")
	writeFile(t, filepath.Join(dir, "patches/config.patch"), "diff --git a/config.yaml b/config.yaml\n--- a/config.yaml\n+++ b/config.yaml\n@@ -2,3 +2,3 @@\n context\n-old\n+new\n end\n@@ -6,3 +6,3 @@\n context\n-old\n+new\n end\n")
	writeFile(t, filepath.Join(dir, "mod.yaml"), "id: Patched\nname: Patched\npatch: patches/config.patch\n")
	if report, err := mods.ValidateMod(dir, f.game()); err != nil || !report.Valid || readFile(t, target) != original {
		t.Fatal(err)
	}
	mods.InstallMod(dir, f.game())
	if readFile(t, target) != strings.ReplaceAll(original, "old\n", "new\n") {
		t.Fatal(readFile(t, target))
	}
	mods.RemoveMod("Patched", f.game())
	if readFile(t, target) != original {
		t.Fatal("not restored")
	}
}

func resourcePatch(path, before, after string) string {
	return fmt.Sprintf("diff --git a/%[1]s b/%[1]s\n--- a/%[1]s\n+++ b/%[1]s\n@@ -1 +1 @@\n-%[2]s\n+%[3]s\n", path, before, after)
}

func TestMultifilePatchRunsBeforeTextOperationsAndSurvivesSourceRemoval(t *testing.T) {
	f := newModFixture(t)
	dir := f.path("mods/Multi")
	second := f.path("game/Data/second.yaml.dvpl")
	packed, _ := resources.EncodeDVPL([]byte("second\n"), resources.DVPL(2))
	os.WriteFile(second, packed, 0o644)
	writeFile(t, filepath.Join(dir, "mod.patch"), resourcePatch("config.yaml", "original", "patched")+resourcePatch("second.yaml", "second", "changed"))
	writeFile(t, filepath.Join(dir, "mod.yaml"), "id: Multi\nname: Multi\npatch: mod.patch\nfiles:\n  config.yaml:\n    - replace:\n        search: patched\n        replace: final\n")
	if report, err := mods.ValidateMod(dir, f.game()); err != nil || !report.Valid {
		t.Fatal(err)
	}
	if _, err := mods.InstallMod(dir, f.game()); err != nil {
		t.Fatal(err)
	}
	decoded, _, _ := resources.DecodeDVPL([]byte(readFile(t, second)))
	if readFile(t, f.path("game/Data/config.yaml")) != "final\n" || string(decoded) != "changed\n" {
		t.Fatal("patch not applied")
	}
	os.RemoveAll(dir)
	if _, err := mods.RemoveMod("Multi", f.game()); err != nil {
		t.Fatal(err)
	}
	if readFile(t, f.path("game/Data/config.yaml")) != "original\n" || readFile(t, second) != string(packed) {
		t.Fatal("not restored")
	}
}

func TestPatchListsKeepOrderAndFoldersAreRecursiveSortedAndDeduplicated(t *testing.T) {
	f := newModFixture(t)
	dir := f.path("mods/Ordered")
	writeFile(t, filepath.Join(dir, "first.patch"), resourcePatch("config.yaml", "original", "first"))
	writeFile(t, filepath.Join(dir, "patches/01.patch"), resourcePatch("config.yaml", "first", "second"))
	writeFile(t, filepath.Join(dir, "patches/nested/02.diff"), resourcePatch("config.yaml", "second", "third"))
	writeFile(t, filepath.Join(dir, "patches/ignored.txt"), "not a patch")
	writeFile(t, filepath.Join(dir, "mod.yaml"), "id: Ordered\nname: Ordered\npatch:\n  - first.patch\n  - patches\n  - patches/01.patch\n")
	if manifest, err := mods.LoadManifest(dir); err != nil || len(manifest.Files) != 3 {
		t.Fatal(err)
	}
	if report, err := mods.ValidateMod(dir, f.game()); err != nil || !report.Valid {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "mod.yaml"), "id: Ordered\nname: Ordered\npatch: [patches, first.patch]\n")
	if report, err := mods.ValidateMod(dir, f.game()); err != nil || report.Valid {
		t.Fatal(err)
	}
}

func TestPatchesRejectUnsupportedChangesAndUnsafePaths(t *testing.T) {
	f := newModFixture(t)
	dir := f.path("mods/Unsafe")
	writeFile(t, filepath.Join(dir, "mod.yaml"), "id: Unsafe\nname: Unsafe\npatch: mod.patch\n")
	for _, diff := range []string{
		resourcePatch("../outside", "original", "changed"),
		"diff --git a/config.yaml b/config.yaml\nnew file mode 100644\n--- /dev/null\n+++ b/config.yaml\n@@ -0,0 +1 @@\n+new\n",
		"diff --git a/config.yaml b/other.yaml\nrename from config.yaml\nrename to other.yaml\n",
		"diff --git a/config.yaml b/config.yaml\nBinary files a/config.yaml and b/config.yaml differ\n",
	} {
		writeFile(t, filepath.Join(dir, "mod.patch"), diff)
		if _, err := mods.LoadManifest(dir); err == nil {
			t.Fatalf("%q loaded", diff)
		}
	}
	writeFile(t, filepath.Join(dir, "mod.yaml"), "id: Unsafe\nname: Unsafe\npatch: ../outside.patch\n")
	if _, err := mods.LoadManifest(dir); errorCode(err) != "MANIFEST_INVALID_PATCH_PATH" {
		t.Fatal(err)
	}
}

func TestGitGenerationSupportsPatchOnlyModsAndPreservesManualSources(t *testing.T) {
	f := newModFixture(t)
	repo, dir := f.path("repo"), f.path("mods/Generated")
	os.MkdirAll(repo, 0o755)
	git(t, repo, "init", "--quiet")
	writeFile(t, filepath.Join(repo, "config.yaml"), "original\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "--quiet", "-m", "base")
	// Headers alone are sufficient when creating the first generated patch.
	manifest := filepath.Join(dir, "mod.yaml")
	writeFile(t, manifest, "id: Generated\nname: Generated\n")
	writeFile(t, filepath.Join(repo, "config.yaml"), "generated\n")
	if _, err := mods.UpdateFromGitDiff(dir, repo, nil); err != nil {
		t.Fatal(err)
	}
	if report, err := mods.ValidateMod(dir, f.game()); err != nil || !report.Valid {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "manual.diff"), resourcePatch("config.yaml", "generated", "manual"))
	writeFile(t, manifest, "id: Generated\nname: Generated\npatch: [mod.patch, manual.diff]\n")
	updated, err := mods.UpdateFromGitDiff(dir, repo, nil)
	if err != nil || !reflect.DeepEqual(updated.Patch, []string{"mod.patch", "manual.diff"}) {
		t.Fatal(err)
	}
	if report, err := mods.ValidateMod(dir, f.game()); err != nil || !report.Valid {
		t.Fatal(err)
	}
	manifestBefore, patchBefore := readFile(t, manifest), readFile(t, filepath.Join(dir, "mod.patch"))
	git(t, repo, "restore", "config.yaml")
	if _, err := mods.UpdateFromGitDiff(dir, repo, nil); errorCode(err) != "GIT_DIFF_EMPTY" {
		t.Fatal(err)
	}
	if readFile(t, manifest) != manifestBefore || readFile(t, filepath.Join(dir, "mod.patch")) != patchBefore {
		t.Fatal("failed update wrote files")
	}
}

func TestModListReportsSavedMetadataAndReinstallStatus(t *testing.T) {
	f := newModFixture(t)
	modCommand := func(args ...string) result {
		return run(t, nil, append(append([]string{"mod"}, args...), "--game-dir", f.game())...)
	}
	empty := modCommand("list", "--json")
	if empty.code != 0 || !reflect.DeepEqual(empty.json(t)["mods"], []any{}) || exists(f.path("game/.tanks-blitz-client")) {
		t.Fatalf("%s", empty.stdout)
	}
	if string(modCommand("list").stdout) != "No mods installed\n" {
		t.Fatal("empty text")
	}
	writeFile(t, f.path("mods/a/mod.yaml"), "id: A\nname: First\nfiles:\n  config.yaml:\n    - replace:\n        search: original\n        replace: intermediate\n    - replace:\n        search: intermediate\n        replace: changed\n")
	writeFile(t, f.path("game/Data/second.yaml"), "second\n")
	writeFile(t, f.path("mods/b/mod.yaml"), "id: B\nname: Second\nfiles:\n  second.yaml:\n    - clear: {}\n")
	mods.InstallMod(f.path("mods/b"), f.game())
	mods.InstallMod(f.path("mods/a"), f.game())
	reinstall := modCommand("install", f.path("mods/a"), "--json")
	report := reinstall.json(t)
	if reinstall.code != 0 || report["installed"] != true || report["reinstalled"] != true ||
		readFile(t, f.path("game/Data/config.yaml")) != "changed\n" {
		t.Fatalf("%s", reinstall.stdout)
	}
	if text := modCommand("install", f.path("mods/a")); text.code != 0 || !strings.HasPrefix(string(text.stdout), "Reinstalled A — First\n") {
		t.Fatalf("%s", text.stdout)
	}
	os.RemoveAll(f.path("mods"))
	listed := modCommand("list", "--json")
	var expected any
	json.Unmarshal([]byte(`[{"id":"A","name":"First","files":["config.yaml"]},{"id":"B","name":"Second","files":["second.yaml"]}]`), &expected)
	if listed.code != 0 || !reflect.DeepEqual(listed.json(t)["mods"], expected) {
		t.Fatalf("%s", listed.stdout)
	}
	if string(modCommand("list").stdout) != "A — First (1 file)\nB — Second (1 file)\n" || modCommand("list", "unexpected").code != 2 {
		t.Fatal("list text")
	}
}

func TestModListReportsInvalidStateInsteadOfHidingInstalledMods(t *testing.T) {
	f := newModFixture(t)
	statePath := f.path("game/.tanks-blitz-client/state.json")
	for _, item := range [][2]string{
		{"{broken", "STATE_INVALID"},
		{"null", "STATE_INVALID"},
		{`{"version":"2","mods":[]}`, "STATE_INVALID"},
		{`{"version":2,"mods":{}}`, "STATE_INVALID"},
		{`{"version":999,"mods":[{"id":"A"}]}`, "STATE_VERSION_UNSUPPORTED"},
	} {
		writeFile(t, statePath, item[0])
		output := run(t, nil, "mod", "list", "--json", "--game-dir", f.game())
		if output.code != 2 || get(output.json(t), "error", "code") != item[1] || readFile(t, statePath) != item[0] {
			t.Fatalf("%s: %s", item[0], output.stdout)
		}
	}
}
