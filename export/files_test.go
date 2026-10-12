package export

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/resources"
)

type filesFixture struct {
	t    *testing.T
	root string
}

func newFilesFixture(t *testing.T) *filesFixture {
	root := t.TempDir()
	for _, dir := range []string{"game/Data", "packs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return &filesFixture{t, root}
}

func (f *filesFixture) path(relative string) string { return filepath.Join(f.root, relative) }

func (f *filesFixture) write(relative, data string) string {
	path := f.path(relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return path
}

func (f *filesFixture) packed(relative, data string, compression uint32) string {
	encoded, err := resources.EncodeDVPL([]byte(data), resources.DVPL(compression))
	if err != nil {
		f.t.Fatal(err)
	}
	return f.write(relative, string(encoded))
}

func (f *filesFixture) options() ClientFilesExportOptions {
	options := NewClientFilesExportOptions(f.path("output"))
	options.GameDir = f.path("game")
	options.CacheDir = f.path("packs")
	return options
}

func (f *filesFixture) resources() {
	// The shadowed broken installation file must never be decoded.
	f.write("game/Data/UI/battle.yaml.dvpl", "broken installation copy")
	f.packed("packs/UI/battle.yaml.dvpl", "cache config", 1)
	f.write("game/Data/Gfx/image.webp", "old image")
	f.packed("packs/Gfx/image.webp.dvpl", "new image", 2)
	f.packed("game/Data/3d/tank.sc2.dvpl", "scene", 3)
	f.write("game/Data/note.txt", "text")
	f.write("game/Data/twin.yaml.dvpl", "invalid packed sibling")
	f.write("game/Data/twin.yaml", "plain config wins")
	f.write("game/Data/unknown.bin", "unknown bytes")
}

func errorCode(err error) string {
	if typed := blitz.AsError(err); typed != nil {
		return typed.Code
	}
	return ""
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func readString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestMergesUnpackedPathsCacheWinsAndRepeatedExportKeepsIdenticalFiles(t *testing.T) {
	f := newFilesFixture(t)
	f.resources()
	f.write("output/unrelated.txt", "keep me")
	var progress []int
	report, err := ExportClientFilesWithProgress(f.options(), func(report *ClientFilesExportReport) error {
		progress = append(progress, report.WrittenCount+report.UnchangedCount)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.FileCount != 6 || report.WrittenCount != 6 || report.CacheFileCount != 2 || report.DVPLCount != 3 {
		t.Fatalf("%+v", report)
	}
	if !slices.Equal(progress, []int{0, 1, 2, 3, 4, 5, 6}) {
		t.Fatal(progress)
	}
	for path, expected := range map[string]string{
		"UI/battle.yaml": "cache config", "Gfx/image.webp": "new image", "3d/tank.sc2": "scene",
		"twin.yaml": "plain config wins", "unknown.bin": "unknown bytes", "unrelated.txt": "keep me",
	} {
		if got := readString(t, filepath.Join(report.OutputDir, path)); got != expected {
			t.Fatalf("%s: %q", path, got)
		}
		if exists(filepath.Join(report.OutputDir, path+".dvpl")) {
			t.Fatal(path)
		}
	}
	before, _ := os.Stat(filepath.Join(report.OutputDir, "UI/battle.yaml"))
	again, err := ExportClientFiles(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if again.WrittenCount != 0 || again.UnchangedCount != 6 {
		t.Fatalf("%+v", again)
	}
	after, _ := os.Stat(filepath.Join(report.OutputDir, "UI/battle.yaml"))
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("unchanged file rewritten")
	}
}

func TestSelectsMultipleCategoriesByUnpackedExtension(t *testing.T) {
	f := newFilesFixture(t)
	f.resources()
	options := f.options()
	options.Types = []ClientFileType{FileTypeConfigs, FileTypeModels}
	report, err := ExportClientFiles(options)
	if err != nil {
		t.Fatal(err)
	}
	if report.FileCount != 4 || exists(filepath.Join(report.OutputDir, "Gfx")) || exists(filepath.Join(report.OutputDir, "unknown.bin")) {
		t.Fatalf("%+v", report)
	}
	for file, expected := range map[string]ClientFileType{
		"a.YAML.DVPL": FileTypeConfigs, "a.actions": FileTypeConfigs, "a.dds.dvpl": FileTypeImages,
		"a.scg": FileTypeModels, "a.animation": FileTypeModels, "a.pck.dvpl": FileTypeSounds,
		"a.ttf": FileTypeFonts, "a.slh": FileTypeShaders, "no-extension": FileTypeOther, ".dvpl": FileTypeOther,
	} {
		if got := FileTypeOf(file); got != expected {
			t.Fatalf("%s: %s", file, got)
		}
	}
	if _, err := ParseClientFileType("unknown"); err == nil {
		t.Fatal("unknown type parsed")
	}
	if kind, err := ParseClientFileType(" Images "); err != nil || kind != FileTypeImages {
		t.Fatal(kind, err)
	}
}

func TestInvalidDVPLPreservesExistingDestinationAndReportsSource(t *testing.T) {
	f := newFilesFixture(t)
	source := f.packed("game/Data/config.yaml.dvpl", "config", 0)
	data, _ := os.ReadFile(source)
	data[0] ^= 1
	os.WriteFile(source, data, 0o644)
	output := f.write("output/config.yaml", "previous export")
	_, err := ExportClientFiles(f.options())
	canonical, _ := filepath.EvalSymlinks(source)
	if errorCode(err) != "DVPL_CRC_FAILED" || blitz.AsError(err).Path != canonical {
		t.Fatal(err)
	}
	if readString(t, output) != "previous export" {
		t.Fatal("destination changed")
	}
	os.WriteFile(source, []byte("invalid footer"), 0o644)
	if _, err := ExportClientFiles(f.options()); errorCode(err) != "DVPL_INVALID" {
		t.Fatal(err)
	}
}

func TestRejectsSourceOverlapAndMissingExplicitCacheBeforeWriting(t *testing.T) {
	f := newFilesFixture(t)
	options := f.options()
	for _, output := range []string{f.path("game/Data/nested"), f.root, f.path("packs/export")} {
		options.OutputDir = output
		if _, err := ExportClientFiles(options); errorCode(err) != "FILE_EXPORT_OUTPUT_INVALID" {
			t.Fatal(output, err)
		}
	}
	if exists(f.path("game/Data/nested")) || exists(f.path("packs/export")) {
		t.Fatal("created output")
	}
	file := f.write("not-a-directory", "keep me")
	options.OutputDir = filepath.Join(file, "export")
	if _, err := ExportClientFiles(options); errorCode(err) != "FILE_EXPORT_OUTPUT_INVALID" {
		t.Fatal(err)
	}
	if readString(t, file) != "keep me" {
		t.Fatal("file changed")
	}
	options = f.options()
	options.CacheDir = f.path("missing-packs")
	if _, err := ExportClientFiles(options); errorCode(err) != "FILE_EXPORT_SOURCE_NOT_FOUND" {
		t.Fatal(err)
	}
	options.NoCache = true
	if _, err := ExportClientFiles(options); errorCode(err) != "FILE_EXPORT_OPTIONS_INVALID" {
		t.Fatal(err)
	}
	options.CacheDir = ""
	f.write("game/Data/config.yaml", "installation")
	f.write("packs/config.yaml", "cache")
	report, err := ExportClientFiles(options)
	if err != nil {
		t.Fatal(err)
	}
	if report.CacheDir != nil || readString(t, filepath.Join(report.OutputDir, "config.yaml")) != "installation" {
		t.Fatalf("%+v", report)
	}
}

func TestSkipsSourceSymlinksAndRejectsOutputSymlinksAndSourceAliases(t *testing.T) {
	f := newFilesFixture(t)
	real := f.write("game/Data/UI/config.yaml", "config")
	if err := os.Symlink(real, f.path("game/Data/linked.yaml")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	report, err := ExportClientFiles(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if report.SkippedSymlinkCount != 1 || exists(filepath.Join(report.OutputDir, "linked.yaml")) {
		t.Fatalf("%+v", report)
	}
	os.Symlink(f.path("game/Data"), f.path("alias"))
	options := f.options()
	options.OutputDir = f.path("alias/nested")
	if _, err := ExportClientFiles(options); errorCode(err) != "FILE_EXPORT_OUTPUT_INVALID" {
		t.Fatal(err)
	}
	options = f.options()
	os.RemoveAll(options.OutputDir)
	os.MkdirAll(options.OutputDir, 0o755)
	outside := f.write("outside/config.yaml", "untouched")
	os.Symlink(filepath.Dir(outside), filepath.Join(options.OutputDir, "UI"))
	if _, err := ExportClientFiles(options); errorCode(err) != "FILE_EXPORT_OUTPUT_INVALID" {
		t.Fatal(err)
	}
	if readString(t, outside) != "untouched" {
		t.Fatal("wrote through symlink")
	}
}

func TestProgressFailureStopsBeforeWritingNextFile(t *testing.T) {
	f := newFilesFixture(t)
	f.write("game/Data/first.txt", "first")
	f.write("game/Data/second.txt", "second")
	options := f.options()
	_, err := ExportClientFilesWithProgress(options, func(report *ClientFilesExportReport) error {
		if report.WrittenCount == 1 {
			return blitz.NewError("CANCELLED", "stop")
		}
		return nil
	})
	if errorCode(err) != "CANCELLED" || !exists(filepath.Join(options.OutputDir, "first.txt")) || exists(filepath.Join(options.OutputDir, "second.txt")) {
		t.Fatal(err)
	}
}

func TestExclusionsMatchUnpackedPathsAndPreserveExportDryRunSelection(t *testing.T) {
	f := newFilesFixture(t)
	f.resources()
	// Neither source may be decoded or used as fallback when this path is excluded.
	f.write("game/Data/skip.yaml", "installation")
	f.write("packs/skip.yaml.dvpl", "invalid DVPL")
	f.write("game/Data/Gfx/.hidden.webp", "hidden image")
	options := f.options()
	options.Exclude = []string{"**/*.yaml", "Gfx/*.[w]eb?"}
	options.DryRun = true
	preview, err := ExportClientFiles(options)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ExcludedFileCount != 5 || preview.FileCount != 3 || exists(options.OutputDir) {
		t.Fatalf("%+v", preview)
	}
	options.DryRun = false
	report, err := ExportClientFiles(options)
	if err != nil {
		t.Fatal(err)
	}
	if report.FileCount != preview.FileCount || report.SourceBytes != preview.SourceBytes ||
		report.ExcludedFileCount != preview.ExcludedFileCount || report.CacheFileCount != 0 {
		t.Fatalf("%+v", report)
	}
	if exists(filepath.Join(report.OutputDir, "skip.yaml")) || exists(filepath.Join(report.OutputDir, "Gfx")) ||
		!exists(filepath.Join(report.OutputDir, "3d/tank.sc2")) {
		t.Fatal("exclusions not applied")
	}
	// A single star does not cross directory boundaries.
	options.DryRun = true
	options.Exclude = []string{"*.yaml", "3d/**"}
	report, err = ExportClientFiles(options)
	if err != nil || report.ExcludedFileCount != 3 || report.FileCount != 5 {
		t.Fatalf("%+v %v", report, err)
	}
}

func TestInvalidExclusionPatternsFailBeforeCreatingOutput(t *testing.T) {
	f := newFilesFixture(t)
	options := f.options()
	for _, pattern := range []string{"[", "a**b", ""} {
		options.Exclude = []string{pattern}
		if _, err := ExportClientFiles(options); errorCode(err) != "FILE_EXPORT_PATTERN_INVALID" || exists(options.OutputDir) {
			t.Fatal(pattern, err)
		}
	}
}
