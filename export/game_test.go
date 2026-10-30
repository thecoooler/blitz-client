package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseVersionText(t *testing.T) {
	str := func(s string) *string { return &s }
	for _, test := range []struct {
		text                                       string
		version, build, branch, platform, verbatim *string
	}{
		{"26.10.70_533068 master MacOS", str("26.10.70"), str("533068"), str("master"), str("MacOS"), str("26.10.70_533068 master MacOS")},
		{"\ufeff 26.10.0_ruby  release Windows 11\r\nsecond line", str("26.10.0_ruby"), nil, str("release"), str("Windows 11"), str("26.10.0_ruby  release Windows 11")},
		{"11.3_", str("11.3_"), nil, nil, nil, str("11.3_")},
		{" \n", nil, nil, nil, nil, nil},
	} {
		var g SnapshotGame
		parseVersionText(&g, test.text)
		got := []*string{g.Version, g.Build, g.Branch, g.Platform, g.VersionText}
		want := []*string{test.version, test.build, test.branch, test.platform, test.verbatim}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %v, want %v", test.text, derefAll(got), derefAll(want))
		}
	}
}

func derefAll(values []*string) []any {
	out := []any{}
	for _, v := range values {
		if v == nil {
			out = append(out, nil)
		} else {
			out = append(out, *v)
		}
	}
	return out
}

func TestGameSectionExport(t *testing.T) {
	f, o := fullDataFixture(t)
	o.Include = []string{"data"}
	f.write("game/Data/version.txt", "26.10.70_533068 master MacOS")
	// The downloaded overlay wins over the installed version.
	f.packed("packs/version.txt.dvpl", "26.10.71_533100 master MacOS", 1)
	report, err := ExportClient(o)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ReadClientExport(o.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	g := s.Game
	if g == nil || g.VersionText == nil || *g.VersionText != "26.10.71_533100 master MacOS" || *g.Version != "26.10.71" || *g.Build != "533100" || *g.Branch != "master" || *g.Platform != "MacOS" {
		t.Fatalf("version: %+v", g)
	}
	if report.Game == nil || *report.Game.Version != "26.10.71" {
		t.Fatalf("report game: %+v", report.Game)
	}
	if _, err := time.Parse(time.RFC3339, s.ExportedAt); err != nil {
		t.Fatalf("exportedAt %q: %v", s.ExportedAt, err)
	}
	// Test binaries carry no module version.
	if !s.ExporterVersion.Null() {
		t.Fatalf("exporterVersion: %+v", s.ExporterVersion)
	}
	var root map[string]any
	readDataJSON(t, o.OutputDir, "data.json", &root)
	if v, ok := root["exporterVersion"]; !ok || v != nil {
		t.Fatalf("data.json exporterVersion must be an explicit null: %v, %v", v, ok)
	}
}

func TestStrictReaderDecodesGameSection(t *testing.T) {
	const doc = `{"format":"blitz-client","version":1,"included":["data"],"game":{"versionText":"26.10.70_533068 master MacOS","version":"26.10.70","build":"533068","branch":"master","platform":"MacOS"},"exportedAt":"2026-10-29T00:00:00Z","exporterVersion":"v1.2.0","nations":[],"tanks":[],"issues":[]}`
	s, err := DecodeStrictDataExport(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	game, ok := s.Game.Get()
	if !ok || *game.Build != "533068" || game.Platform == nil {
		t.Fatalf("game: %+v", s.Game)
	}
	if at, _ := s.ExportedAt.Get(); at != "2026-10-29T00:00:00Z" {
		t.Fatalf("exportedAt: %+v", s.ExportedAt)
	}
	if v, _ := s.ExporterVersion.Get(); v != "v1.2.0" {
		t.Fatalf("exporterVersion: %+v", s.ExporterVersion)
	}
	nulls := strings.NewReplacer(`"build":"533068"`, `"build":null`, `"v1.2.0"`, `null`).Replace(doc)
	if s, err := DecodeStrictDataExport(strings.NewReader(nulls)); err != nil {
		t.Fatal(err)
	} else if game, _ := s.Game.Get(); game.Build != nil || !s.ExporterVersion.Null() {
		t.Fatalf("nulls: %+v, %+v", game, s.ExporterVersion)
	}
}

func TestGameSectionWithoutSources(t *testing.T) {
	_, o := fullDataFixture(t)
	o.Include = []string{"icons"}
	o.NoCache, o.CacheDir = true, ""
	if _, err := ExportClient(o); err != nil {
		t.Fatal(err)
	}
	s, err := ReadClientExport(o.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.Game, &SnapshotGame{}) {
		t.Fatalf("game: %+v", s.Game)
	}
}

// An export written before game metadata existed keeps its fields when read
// and encoded again.
func TestReadClientExportKeepsMissingGameFields(t *testing.T) {
	dir := t.TempDir()
	const doc = `{"format":"blitz-client","version":1,"included":["data"],"nations":[],"tanks":[],"issues":[]}`
	if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ReadClientExport(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"game", "exportedAt", "exporterVersion"} {
		if _, ok := got[key]; ok {
			t.Errorf("re-encoded old export gained %q: %s", key, b)
		}
	}
}
