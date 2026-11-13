package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"image/color"

	"github.com/thecoooler/blitz-client/client"
)

func mapsFixture(t *testing.T) *filesFixture {
	t.Helper()
	f := newFilesFixture(t)
	f.write("game/Data/maps.yaml", `maps:
    karelia:
        id: 1
        localName: "17_karelia_ka/17_karelia_ka.sc2"
        spriteFrame: 1
        availableModes: [0, 1]
        vehicleCamouflageKind: "summer"
    skit:
        id: 2
        localName: "29_skit_sk/29_skit_sk.sc2"
        spriteFrame: 0
        vehicleCamouflageKind: "desert"
    lost:
        id: 3
        localName: "40_lost_ls/40_lost_ls.sc2"
        spriteFrame: 0
`)
	// The atlas holds two 2x2 frames inside the 4x2 image.
	f.packed("game/Data/Gfx/UI/MapsListDialog/back_maps@2x.packed.webp.dvpl",
		packedWebP(t, "1\n\n4 2\n2\n0 0 2 2 0 0 0 \n2 0 2 2 0 0 0 \n"), 1)
	f.write("game/Data/Gfx/UI/BattleScreenHUD/minimap/karelia/MiniMapSmall@2x.packed.webp",
		packedWebP(t, "1\n\n2 2\n1\n0 0 2 2 0 0 0 \n"))
	// skit only has the SD variant; lost has no minimap at all.
	f.write("game/Data/Gfx/UI/BattleScreenHUD/minimap/skit/MiniMapSmall.packed.webp",
		packedWebP(t, "1\n\n2 2\n1\n0 0 2 2 0 0 0 \n"))
	f.write("game/Data/3d/Maps/17_karelia_ka/blitz/map_effects.yaml", `keyedArchive:
    fx1:
        string: "ground"
    fx0:
        string: "stone"
`)
	return f
}

func mapIndex(t *testing.T, path string) []MapExport {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entries []MapExport
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestExportMapsRendersPreviewsAndMinimaps(t *testing.T) {
	f := mapsFixture(t)
	output := f.path("output")
	snapshot := &Snapshot{}
	report, err := exportMaps(f.path("game/Data"), output, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if report.Maps != 3 || report.Previews != 3 || report.Minimaps != 2 {
		t.Fatalf("unexpected report %+v", report)
	}
	entries := mapIndex(t, filepath.Join(output, "maps.json"))
	if len(entries) != 3 {
		t.Fatalf("unexpected index %+v", entries)
	}
	if entries[0].Name != "karelia" || entries[0].ID != 1 || entries[0].Scene != "17_karelia_ka/17_karelia_ka.sc2" ||
		entries[0].SpriteFrame != 1 || entries[0].AvailableInTrainingRoom || entries[0].Season != "summer" {
		t.Fatalf("unexpected entry %+v", entries[0])
	}
	if len(entries[0].AvailableModes) != 2 || entries[0].AvailableModes[0] != 0 || entries[0].AvailableModes[1] != 1 {
		t.Fatalf("modes must be kept: %+v", entries[0])
	}
	if entries[0].Preview == nil || *entries[0].Preview != "assets/maps/karelia/preview.png" ||
		entries[0].Minimap == nil || *entries[0].Minimap != "assets/maps/karelia/minimap.png" {
		t.Fatalf("unexpected links %+v", entries[0])
	}
	// The sources name the client resources the images were rendered from.
	if entries[0].PreviewSource == nil ||
		entries[0].PreviewSource.Kind != "sprite" ||
		entries[0].PreviewSource.Path != "Gfx/UI/MapsListDialog/back_maps@2x.packed.webp" ||
		entries[0].PreviewSource.Frame == nil ||
		*entries[0].PreviewSource.Frame != (IconFrame{X: 2, Y: 0, Width: 2, Height: 2}) {
		t.Fatalf("unexpected preview source %+v", entries[0].PreviewSource)
	}
	if entries[0].MinimapSource == nil ||
		entries[0].MinimapSource.Path != "Gfx/UI/BattleScreenHUD/minimap/karelia/MiniMapSmall@2x.packed.webp" ||
		entries[0].MinimapSource.Frame == nil ||
		*entries[0].MinimapSource.Frame != (IconFrame{Width: 2, Height: 2}) {
		t.Fatalf("unexpected minimap source %+v", entries[0].MinimapSource)
	}
	if len(entries[0].Effects) != 2 || entries[0].Effects[0] != "stone" || entries[0].Effects[1] != "ground" {
		t.Fatalf("unexpected effects %+v", entries[0].Effects)
	}
	// skit used the SD minimap fallback.
	if entries[1].Minimap == nil || *entries[1].Minimap != "assets/maps/skit/minimap.png" || entries[1].Season != "desert" {
		t.Fatalf("unexpected skit entry %+v", entries[1])
	}
	// lost has no map kind at all, so its season stays empty.
	if entries[2].Season != "" {
		t.Fatalf("unexpected lost season %+v", entries[2])
	}
	// lost has no minimap: null plus an issue, but its preview renders.
	if entries[2].Minimap != nil || entries[2].Preview == nil {
		t.Fatalf("unexpected lost entry %+v", entries[2])
	}
	if len(snapshot.Issues) != 1 || snapshot.Issues[0].Section != "maps" ||
		!strings.Contains(snapshot.Issues[0].Message, "map image is not installed") {
		t.Fatalf("expected one missing-minimap issue, got %v", snapshot.Issues)
	}
	// Previews are rendered onto the descriptor canvas (4x2), minimaps onto
	// their own 2x2 canvas; all crops show the atlas color.
	previews := map[string][2]int{*entries[0].Preview: {4, 2}, *entries[0].Minimap: {2, 2}}
	for file, size := range previews {
		img := readPNG(t, filepath.Join(output, file))
		if img.Bounds().Dx() != size[0] || img.Bounds().Dy() != size[1] {
			t.Fatalf("%s has size %v", file, img.Bounds())
		}
		if got := color.NRGBAModel.Convert(img.At(0, 0)); got != (color.NRGBA{60, 100, 7, 255}) {
			t.Fatalf("%s has color %v", file, got)
		}
	}
}

func TestExportMapsPrefersAt2xAtlas(t *testing.T) {
	f := mapsFixture(t)
	// The SD atlas has a single frame, so karelia (spriteFrame 1) only
	// resolves when the @2x atlas is preferred.
	f.write("game/Data/Gfx/UI/MapsListDialog/back_maps.packed.webp",
		packedWebP(t, "1\n\n2 2\n1\n0 0 2 2 0 0 0 \n"))
	snapshot := &Snapshot{}
	report, err := exportMaps(f.path("game/Data"), f.path("output"), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if report.Previews != 3 {
		t.Fatalf("unexpected report %+v", report)
	}
	for _, issue := range snapshot.Issues {
		if strings.Contains(issue.Message, "preview atlas") {
			t.Fatalf("SD atlas was used: %v", snapshot.Issues)
		}
	}
}

func TestExportMapsWithoutClientData(t *testing.T) {
	f := newFilesFixture(t)
	snapshot := &Snapshot{}
	report, err := exportMaps(f.path("game/Data"), f.path("output"), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if report.Maps != 0 || report.Previews != 0 || report.Minimaps != 0 || report.File == "" {
		t.Fatalf("unexpected report %+v", report)
	}
	if len(mapIndex(t, report.File)) != 0 {
		t.Fatal("expected an empty index")
	}
	if len(snapshot.Issues) != 1 || snapshot.Issues[0].Section != "maps" ||
		!strings.Contains(snapshot.Issues[0].Message, "maps.yml or maps.yaml was not found") {
		t.Fatalf("expected a missing-data issue, got %v", snapshot.Issues)
	}
}

func TestExportMapsSkipsFramesOutsideTheAtlas(t *testing.T) {
	f := mapsFixture(t)
	// A spriteFrame past the atlas loses only its preview.
	f.write("game/Data/Gfx/UI/BattleScreenHUD/minimap/lost/MiniMapSmall.packed.webp",
		packedWebP(t, "1\n\n2 2\n1\n0 0 2 2 0 0 0 \n"))
	f.write("game/Data/maps.yaml", `maps:
    karelia:
        id: 1
        localName: "17_karelia_ka/17_karelia_ka.sc2"
        spriteFrame: 5
`)
	snapshot := &Snapshot{}
	report, err := exportMaps(f.path("game/Data"), f.path("output"), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if report.Previews != 0 || report.Minimaps != 1 {
		t.Fatalf("unexpected report %+v", report)
	}
	entries := mapIndex(t, report.File)
	if entries[0].Preview != nil || entries[0].Minimap == nil {
		t.Fatalf("unexpected entry %+v", entries[0])
	}
	if len(snapshot.Issues) != 1 || !strings.Contains(snapshot.Issues[0].Message, "preview atlas has no frame 5") {
		t.Fatalf("expected a frame issue, got %v", snapshot.Issues)
	}
}

func TestMapSeasonsInheritFromScene(t *testing.T) {
	got := mapSeasons([]client.ClientMap{
		{Name: "base", LocalName: "scene/a"},
		{Name: "base_02", LocalName: "scene/a", VehicleCamouflageKind: "winter"},
		{Name: "other", LocalName: "scene/b", VehicleCamouflageKind: " summer "},
		{Name: "other_02", LocalName: "scene/b"},
		{Name: "lost", LocalName: "scene/c"},
	})
	want := []string{"winter", "winter", "summer", "summer", ""}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("seasons = %v, want %v", got, want)
		}
	}
}
