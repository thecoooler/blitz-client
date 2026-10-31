package export

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorldCatalogExport(t *testing.T) {
	f, o := fullDataFixture(t)
	f.write("game/Data/maps.yaml", "maps:\n  karelia:\n    id: 1\n    localName: '17/map.sc2'\n    spriteFrame: 1\n    extra: [1, 2]\n  other:\n    id: 2\n    spriteFrame: 2\n")
	f.write("game/Data/Strings/ru.yaml", "\"#maps:karelia:17/map.sc2\": \"Балтийский щит\"\n\"Maps/Description/karelia\": \"Описание\"\n\"TankMechanics/Test\": \"Механика\"\n\"TankMechanics/TestDescription\": \"Описание механики\"\n\"TankMechanics/Unused\": \"Другая\"\n")
	f.write("game/Data/Gfx/Lobby/backgrounds/premium/maps/karelia.txt", "1\natlas.png\n2 2\n2\n0 0 1 1 0 0 0\n1 0 1 1 1 0 0\n")
	f.write("game/Data/Gfx/Lobby/backgrounds/premium/maps/atlas.png", pngBytes(t, 2, 2, func(x, y int) color.NRGBA { return color.NRGBA{R: 1, A: 255} }))
	f.write("packs/Gfx/Lobby/backgrounds/premium/maps/atlas.png", pngBytes(t, 2, 2, func(x, y int) color.NRGBA { return color.NRGBA{R: 200, A: 255} }))
	f.write("game/Data/Gfx/Lobby/icons/tank-mechanics/test/icon.png", pngBytes(t, 1, 1, atlasPixel))
	f.write("game/Data/Gfx/Lobby/icons/tank-mechanics/test/broken.txt", "1\nmissing.png\n1 1\n1\n0 0 1 1 0 0 0\n")
	if _, err := ExportClient(o); err != nil {
		t.Fatal(err)
	}
	s, err := ReadClientExport(o.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	w := s.World
	if w == nil || len(w.Maps) != 2 || len(w.Mechanics) != 2 {
		t.Fatalf("catalog: %+v", w)
	}
	m := w.Maps[0]
	if m.Key != "karelia" || m.NameKey != "#maps:karelia:17/map.sc2" || m.DescriptionKey != "Maps/Description/karelia" || m.TacticsKey != "" || len(m.Images) != 2 {
		t.Fatalf("map: %+v", m)
	}
	if w.Mechanics[0].ID != "Test" || len(w.Mechanics[0].Tanks) != 2 || w.Mechanics[0].DescriptionKey != "TankMechanics/TestDescription" || len(w.Mechanics[1].Tanks) != 0 {
		t.Fatalf("mechanics: %+v", w.Mechanics)
	}
	if len(w.MechanicsMedia) != 1 || len(w.MediaIssues) != 1 || !strings.Contains(w.MediaIssues[0].Message, "missing sprite atlas") {
		t.Fatalf("media: %+v %+v", w.MechanicsMedia, w.MediaIssues)
	}
	file, err := os.Open(filepath.Join(o.OutputDir, filepath.FromSlash(m.Images[0].File)))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	r, _, _, _ := img.At(0, 0).RGBA()
	if r != 200*257 {
		t.Fatalf("atlas overlay pixel: %d", r)
	}
	// The strict reader also understands the additive catalog.
	encoded, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	doc := fmt.Sprintf(`{"format":"blitz-client","version":1,"included":["data"],"nations":[],"tanks":[],"world":%s,"issues":[]}`, encoded)
	if _, err := DecodeStrictDataExport(strings.NewReader(doc)); err != nil {
		t.Fatal(err)
	}
}

func TestWorldDataOnlyDoesNotReadImages(t *testing.T) {
	f, o := fullDataFixture(t)
	o.Include = []string{"data"}
	f.write("game/Data/maps.yaml", "maps: {map: {id: 1, spriteFrame: 1}}")
	f.write("game/Data/Gfx/Lobby/icons/tank-mechanics/bad.png", "invalid")
	if _, err := ExportClient(o); err != nil {
		t.Fatal(err)
	}
	s, err := ReadClientExport(o.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	if s.World == nil || len(s.World.MediaIssues) != 0 || len(s.World.MechanicsMedia) != 0 {
		t.Fatalf("world: %+v", s.World)
	}
}
