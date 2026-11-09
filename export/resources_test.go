package export

import (
	"path/filepath"
	"testing"

	"github.com/thecoooler/blitz-client/client"
)

func TestIconExportReadsCacheOverlay(t *testing.T) {
	f := newFilesFixture(t)
	f.write("game/Data/Gfx/A/fromData.png", pngBytes(t, 2, 2, atlasPixel))
	f.packed("packs/Gfx/A/fromData.png.dvpl", pngBytes(t, 4, 4, atlasPixel), 1)
	f.packed("packs/Gfx/A/fromCache.png.dvpl", pngBytes(t, 3, 2, atlasPixel), 2)
	f.packed("packs/Gfx/B/only.png.dvpl", pngBytes(t, 1, 1, atlasPixel), 1)

	resources, err := client.OpenResources(f.path("game/Data"), f.path("packs"))
	if err != nil {
		t.Fatal(err)
	}
	sets := existingIconSets(resources, []iconSet{
		{"a", []string{"Gfx/A"}, nil, nil},
		{"b", []string{"Gfx/B"}, nil, nil},
	})
	if len(sets) != 2 {
		t.Fatalf("cache-only set dropped: %+v", sets)
	}
	report, _, err := exportIconsWithResources(resources, f.path("game/Data"), f.path("output"), sets, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.IconCount != 3 {
		t.Fatalf("unexpected report %+v", report)
	}
	if img := readPNG(t, filepath.Join(f.path("output"), "assets/icons/a/fromData.png")); img.Bounds().Dx() != 4 {
		t.Fatalf("cache override not used: %v", img.Bounds())
	}
	if img := readPNG(t, filepath.Join(f.path("output"), "assets/icons/a/fromCache.png")); img.Bounds().Dx() != 3 {
		t.Fatalf("cache-only icon not exported: %v", img.Bounds())
	}
	if img := readPNG(t, filepath.Join(f.path("output"), "assets/icons/b/only.png")); img.Bounds().Dx() != 1 {
		t.Fatalf("cache-only directory not exported: %v", img.Bounds())
	}
}
