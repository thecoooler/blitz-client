package export

import (
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
	f.write("game/Data/languages.yaml", "[\n    \"ru\",\n    \"en\"\n]\n")
	f.write("game/Data/regions_publish.yaml", "Common:\n  public: \"true\"\nRegions:\n  Russia: {realm: ru, flag: ru}\n  Other: {flag: none}\n  Belarus: {realm: by}\n  Again: {realm: ru}\n")
	f.write("game/Data/3d/Configs/Presets/hg_halloween_26/presets.yaml", "{}\n")
	f.packed("packs/3d/Configs/Presets/hangar_main_2024/presets.yaml.dvpl", "{}\n", 1)
	f.write("game/Data/3d/Configs/Presets/nested/deeper/presets.yaml", "{}\n")
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
	if !reflect.DeepEqual(g.Realms, []string{"ru", "by"}) || !reflect.DeepEqual(g.Languages, []string{"ru", "en"}) || !reflect.DeepEqual(g.HangarPresets, []string{"hangar_main_2024", "hg_halloween_26"}) {
		t.Fatalf("game: %+v", g)
	}
	// version.txt and the hangar preset come from packs.
	if g.CacheFiles != 2 {
		t.Fatalf("cache files: %d", g.CacheFiles)
	}
	if report.Game == nil || *report.Game.Version != "26.10.71" {
		t.Fatalf("report game: %+v", report.Game)
	}
	if _, err := time.Parse(time.RFC3339, s.ExportedAt); err != nil {
		t.Fatalf("exportedAt %q: %v", s.ExportedAt, err)
	}
}

func TestStrictReaderDecodesGameSection(t *testing.T) {
	const doc = `{"format":"blitz-client","version":1,"included":["data"],"game":{"versionText":"26.10.70_533068 master MacOS","version":"26.10.70","build":"533068","branch":"master","platform":"MacOS","realms":["ru"],"languages":["ru"],"hangarPresets":[],"cacheFiles":0},"exportedAt":"2026-10-29T00:00:00Z","nations":[],"tanks":[],"issues":[]}`
	s, err := DecodeStrictDataExport(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	game, ok := s.Game.Get()
	if !ok || *game.Build != "533068" || game.Platform == nil || !reflect.DeepEqual(game.Realms, []string{"ru"}) {
		t.Fatalf("game: %+v", s.Game)
	}
	if at, _ := s.ExportedAt.Get(); at != "2026-10-29T00:00:00Z" {
		t.Fatalf("exportedAt: %+v", s.ExportedAt)
	}
	nulls := strings.Replace(doc, `"build":"533068"`, `"build":null`, 1)
	if s, err := DecodeStrictDataExport(strings.NewReader(nulls)); err != nil {
		t.Fatal(err)
	} else if game, _ := s.Game.Get(); game.Build != nil {
		t.Fatalf("null build: %+v", game)
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
	want := &SnapshotGame{Realms: []string{}, Languages: []string{}, HangarPresets: []string{}}
	if !reflect.DeepEqual(s.Game, want) {
		t.Fatalf("game: %+v", s.Game)
	}
}

func TestMalformedGameResourceFails(t *testing.T) {
	f, o := fullDataFixture(t)
	f.write("game/Data/regions_publish.yaml", "Regions: [ru]\n")
	if _, err := ExportClient(o); err == nil {
		t.Fatal("malformed regions_publish.yaml was accepted")
	}
}
