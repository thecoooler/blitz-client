package export

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func pngBytes(t *testing.T, width, height int, fill func(x, y int) color.NRGBA) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.SetNRGBA(x, y, fill(x, y))
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func atlasPixel(x, y int) color.NRGBA { return color.NRGBA{uint8(x * 40), uint8(y * 40), 7, 255} }

func iconFixture(t *testing.T) (*filesFixture, []iconSet) {
	f := newFilesFixture(t)
	f.packed("game/Data/Gfx/A/texture0@2x.png.dvpl", pngBytes(t, 4, 4, atlasPixel), 2)
	f.write("game/Data/Gfx/A/texture0.png", pngBytes(t, 2, 2, atlasPixel))
	f.packed("game/Data/Gfx/A/red@2x.txt.dvpl", "1\ntexture0@2x.png\n6 5\n1\n2 1 2 2 3 2 0 frame0\n", 1)
	f.write("game/Data/Gfx/A/red.txt", "1\ntexture0.png\n3 3\n1\n0 0 1 1 1 1 0 frame0\n")
	f.write("game/Data/Gfx/A/notes.yaml", "ignored")
	f.write("game/Data/Gfx/B1/.keep", "")
	f.write("game/Data/Gfx/B2/plain@2x.png", pngBytes(t, 3, 2, func(int, int) color.NRGBA { return color.NRGBA{1, 2, 3, 128} }))
	return f, []iconSet{{"a", []string{"Gfx/A"}}, {"b", []string{"Gfx/B1", "Gfx/B2"}}}
}

func readCatalog(t *testing.T, path string) IconCatalog {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var catalog IconCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	return catalog
}

func readPNG(t *testing.T, path string) image.Image {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestExportsIconSetsWithCatalog(t *testing.T) {
	f, sets := iconFixture(t)
	output := f.path("output")
	report, err := exportIcons(f.path("game/Data"), output, sets)
	if err != nil {
		t.Fatal(err)
	}
	if report.IconCount != 2 || report.WrittenCount != 2 || report.UnchangedCount != 0 || report.RemovedCount != 0 {
		t.Fatalf("unexpected report %+v", report)
	}
	catalog := readCatalog(t, report.Catalog)
	if catalog.FormatVersion != 1 || catalog.Count != 2 || len(catalog.Sets) != 2 {
		t.Fatalf("unexpected catalog %+v", catalog)
	}
	red := catalog.Sets[0].Icons
	want := IconEntry{ID: "red", File: "a/red.png", Width: 6, Height: 5, Scale: 2, Source: IconSource{
		Kind: "sprite", Path: "Gfx/A/red@2x.txt", Texture: "Gfx/A/texture0@2x.png",
		Frame: &IconFrame{X: 2, Y: 1, Width: 2, Height: 2, OffsetX: 3, OffsetY: 2}}}
	if len(red) != 1 || red[0].Source.Frame == nil || *red[0].Source.Frame != *want.Source.Frame {
		t.Fatalf("unexpected set a icons %+v", red)
	}
	red[0].Source.Frame, want.Source.Frame = nil, nil
	if red[0] != want {
		t.Fatalf("got %+v, want %+v", red[0], want)
	}
	plain := catalog.Sets[1]
	if plain.ID != "b" || len(plain.Sources) != 2 || len(plain.Icons) != 1 || plain.Icons[0] != (IconEntry{
		ID: "plain", File: "b/plain.png", Width: 3, Height: 2, Scale: 2,
		Source: IconSource{Kind: "image", Path: "Gfx/B2/plain@2x.png"}}) {
		t.Fatalf("unexpected set b %+v", plain)
	}

	img := readPNG(t, filepath.Join(output, "assets/icons/a/red.png"))
	if img.Bounds() != image.Rect(0, 0, 6, 5) {
		t.Fatalf("unexpected canvas %v", img.Bounds())
	}
	for y := range 5 {
		for x := range 6 {
			got := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			want := color.NRGBA{}
			if x >= 3 && x < 5 && y >= 2 && y < 4 {
				want = atlasPixel(x-3+2, y-2+1)
			}
			if got != want {
				t.Fatalf("pixel %d,%d = %v, want %v", x, y, got, want)
			}
		}
	}
	if got := color.NRGBAModel.Convert(readPNG(t, filepath.Join(output, "assets/icons/b/plain.png")).At(1, 1)); got != (color.NRGBA{1, 2, 3, 128}) {
		t.Fatalf("standalone image changed: %v", got)
	}
	if exists(filepath.Join(output, "assets/icons/a/texture0.png")) {
		t.Fatal("atlas exported as an icon")
	}
}

func TestIconExportKeepsUnchangedAndRemovesStaleIcons(t *testing.T) {
	f, sets := iconFixture(t)
	output := f.path("output")
	if _, err := exportIcons(f.path("game/Data"), output, sets); err != nil {
		t.Fatal(err)
	}
	unrelated := f.write("output/assets/icons/b/custom.png", "user file")
	if err := os.Remove(f.path("game/Data/Gfx/B2/plain@2x.png")); err != nil {
		t.Fatal(err)
	}
	report, err := exportIcons(f.path("game/Data"), output, sets)
	if err != nil {
		t.Fatal(err)
	}
	if report.IconCount != 1 || report.WrittenCount != 0 || report.UnchangedCount != 1 || report.RemovedCount != 1 {
		t.Fatalf("unexpected report %+v", report)
	}
	if exists(filepath.Join(output, "assets/icons/b/plain.png")) || !exists(unrelated) {
		t.Fatal("stale icon kept or unrelated file removed")
	}
	if catalog := readCatalog(t, report.Catalog); catalog.Count != 1 || len(catalog.Sets[1].Icons) != 0 {
		t.Fatalf("unexpected catalog %+v", catalog)
	}
}

func TestIconExportRejectsDuplicatesAndMissingSources(t *testing.T) {
	f, sets := iconFixture(t)
	f.write("game/Data/Gfx/B1/plain.png", pngBytes(t, 1, 1, atlasPixel))
	if _, err := exportIcons(f.path("game/Data"), f.path("output"), sets); errorCode(err) != "ICON_EXPORT_DUPLICATE" {
		t.Fatalf("expected ICON_EXPORT_DUPLICATE, got %v", err)
	}
	missing := []iconSet{{"missing", []string{"Gfx/Missing"}}}
	if _, err := exportIcons(f.path("game/Data"), f.path("output"), missing); errorCode(err) != "IO_ERROR" {
		t.Fatalf("expected IO_ERROR, got %v", err)
	}
}

func TestIconExportRejectsFramesOutsideAtlas(t *testing.T) {
	f, sets := iconFixture(t)
	f.packed("game/Data/Gfx/A/red@2x.txt.dvpl", "1\ntexture0@2x.png\n6 5\n1\n3 3 2 2 0 0 0\n", 1)
	if _, err := exportIcons(f.path("game/Data"), f.path("output"), sets); errorCode(err) != "ICON_EXPORT_INVALID" {
		t.Fatalf("expected ICON_EXPORT_INVALID, got %v", err)
	}
}
