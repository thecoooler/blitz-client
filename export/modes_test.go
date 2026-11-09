package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"image/color"
)

func modesFixture(t *testing.T) *filesFixture {
	t.Helper()
	f := newFilesFixture(t)
	f.write("game/Data/Strings/ru.yaml", `"battleType/regular": "Обычный бой"
"hangar/battleType/regularDescription": "Базовый режим боя."
"battleType/tournament": "Турнир"
"battleResults/Footer/Tournament": "Турнир"
"battleType/Lunar": "Лунный бой"
"hangar/battleType/LunarDescription": "Лунное описание"
"hangar/battleType/moonFightShortDescription": "Кратко о луне"
`)
	root := "game/Data/Gfx/Lobby/backgrounds/GameMode/"
	// The SD variant must not win over the @2x variant.
	f.write(root+"icon_regular.packed.webp", packedWebP(t, "1\n\n1 1\n1\n0 0 1 1 0 0 0 \n"))
	f.packed(root+"icon_regular@2x.packed.webp.dvpl", packedWebP(t, "1\n\n2 2\n1\n0 0 2 2 0 0 0 \n"), 1)
	f.packed(root+"bg_regular_card@2x.packed.webp.dvpl", packedWebP(t, "1\n\n4 2\n1\n0 0 4 2 0 0 0 \n"), 1)
	for index := 1; index <= 3; index++ {
		f.packed(root+"bg_regular_layer"+string(rune('0'+index))+"@2x.packed.webp.dvpl",
			packedWebP(t, "1\n\n3 1\n1\n0 0 3 1 0 0 0 \n"), 1)
	}
	// Modes without a card banner still export their logo.
	f.packed(root+"icon_tournament@2x.packed.webp.dvpl", packedWebP(t, "1\n\n2 2\n1\n0 0 2 2 0 0 0 \n"), 1)
	// The artwork folder uses the lower-case mode ID.
	f.write(root+"icon_lunar.png", pngBytes(t, 1, 1, func(x, y int) color.NRGBA { return color.NRGBA{1, 2, 3, 255} }))
	f.write(root+"bg_lunar_card.packed.webp", packedWebP(t, "1\n\n2 1\n1\n0 0 2 1 0 0 0 \n"))
	f.write(root+"bg_lunar_layer1.packed.webp", packedWebP(t, "1\n\n1 1\n1\n0 0 1 1 0 0 0 \n"))
	return f
}

func readModeIndex(t *testing.T, path string) []GameModeExport {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entries []GameModeExport
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestExportGameModesDefinesLocalizationAndArtwork(t *testing.T) {
	f := modesFixture(t)
	snapshot := &Snapshot{}
	report, err := exportGameModes(collectMechanicFiles(t, f), f.path("game/Data"), f.path("output"), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if report.Modes != 14 || report.Logos != 3 || report.Banners != 2 || report.Layers != 4 {
		t.Fatalf("unexpected report %+v", report)
	}
	entries := readModeIndex(t, report.File)
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	want := []string{"regular", "training", "rating", "tournament", "arcade", "scuffle", "hellgames", "Lunar", "nanomaps", "vampiric", "bossmode", "gravitizing", "tentoten", "battle-royale"}
	if len(ids) != len(want) {
		t.Fatalf("unexpected modes %v", ids)
	}
	for index := range want {
		if ids[index] != want[index] {
			t.Fatalf("unexpected mode order %v", ids)
		}
	}

	regular := entries[0]
	if regular.NameKey != "battleType/regular" || regular.Name != "Обычный бой" ||
		regular.DescriptionKey == nil || *regular.DescriptionKey != "hangar/battleType/regularDescription" ||
		regular.Description != "Базовый режим боя." || regular.ShortDescriptionKey != nil || regular.ShortDescription != "" {
		t.Fatalf("unexpected regular entry %+v", regular)
	}
	if regular.Logo == nil || *regular.Logo != "assets/modes/regular/logo.png" ||
		regular.Banner == nil || *regular.Banner != "assets/modes/regular/banner.png" ||
		len(regular.Layers) != 3 {
		t.Fatalf("unexpected regular links %+v", regular)
	}
	if regular.LogoSource == nil || regular.LogoSource.Path != "Gfx/Lobby/backgrounds/GameMode/icon_regular@2x.packed.webp" {
		t.Fatalf("unexpected logo source %+v", regular.LogoSource)
	}
	if regular.BannerSource == nil || regular.BannerSource.Path != "Gfx/Lobby/backgrounds/GameMode/bg_regular_card@2x.packed.webp" {
		t.Fatalf("unexpected banner source %+v", regular.BannerSource)
	}
	if regular.Layers[0].Image != "assets/modes/regular/layer1.png" ||
		regular.Layers[0].Source.Path != "Gfx/Lobby/backgrounds/GameMode/bg_regular_layer1@2x.packed.webp" {
		t.Fatalf("unexpected layer %+v", regular.Layers[0])
	}
	if logo := readPNG(t, filepath.Join(f.path("output"), *regular.Logo)); logo.Bounds().Dx() != 2 || logo.Bounds().Dy() != 2 {
		t.Fatalf("@2x logo was not chosen: %v", logo.Bounds())
	}
	if banner := readPNG(t, filepath.Join(f.path("output"), *regular.Banner)); banner.Bounds().Dx() != 4 || banner.Bounds().Dy() != 2 {
		t.Fatalf("unexpected banner size %v", banner.Bounds())
	}
	if files, err := os.ReadDir(f.path("output/assets/modes/regular")); err != nil || len(files) != 5 {
		t.Fatalf("unexpected regular files: %v %v", files, err)
	}

	tournament := entries[3]
	if tournament.NameKey != "battleResults/Footer/Tournament" || tournament.Name != "Турнир" ||
		tournament.DescriptionKey != nil || tournament.Description != "" {
		t.Fatalf("unexpected tournament entry %+v", tournament)
	}
	if tournament.Logo == nil || *tournament.Logo != "assets/modes/tournament/logo.png" ||
		tournament.Banner != nil || len(tournament.Layers) != 0 {
		t.Fatalf("unexpected tournament artwork %+v", tournament)
	}

	lunar := entries[7]
	if lunar.NameKey != "battleType/Lunar" || lunar.Name != "Лунный бой" ||
		lunar.Description == "" || lunar.ShortDescription != "Кратко о луне" {
		t.Fatalf("unexpected lunar entry %+v", lunar)
	}
	if lunar.Logo == nil || *lunar.Logo != "assets/modes/lunar/logo.png" ||
		lunar.Banner == nil || *lunar.Banner != "assets/modes/lunar/banner.png" ||
		len(lunar.Layers) != 1 || lunar.Layers[0].Image != "assets/modes/lunar/layer1.png" {
		t.Fatalf("lower-case mode folder not used %+v", lunar)
	}

	training := entries[1]
	if training.Logo != nil || training.Banner != nil || len(training.Layers) != 0 {
		t.Fatalf("missing artwork must be null %+v", training)
	}
	if len(snapshot.Issues) != 0 {
		t.Fatalf("missing optional artwork must not be an issue: %+v", snapshot.Issues)
	}
}

func TestExportModesSection(t *testing.T) {
	f, options := fullDataFixture(t)
	options.Include = []string{"modes"}
	// A modes-only export must not read the full data document set.
	for _, relative := range []string{
		"game/Data/XML/item_defs/vehicles/common/optional_devices.xml",
		"game/Data/XML/item_defs/vehicles/common/optional_device_slots.xml",
		"game/Data/XML/item_defs/vehicles/common/provisions/list.xml",
	} {
		if err := os.Remove(f.path(relative)); err != nil {
			t.Fatal(err)
		}
	}
	f.packed("game/Data/Gfx/Lobby/backgrounds/GameMode/icon_regular@2x.packed.webp.dvpl",
		packedWebP(t, "1\n\n2 2\n1\n0 0 2 2 0 0 0 \n"), 1)
	f.packed("game/Data/Gfx/Lobby/backgrounds/GameMode/bg_regular_card@2x.packed.webp.dvpl",
		packedWebP(t, "1\n\n4 2\n1\n0 0 4 2 0 0 0 \n"), 1)
	report, err := ExportClient(options)
	if err != nil {
		t.Fatal(err)
	}
	if report.ModeExport == nil || report.ModeExport.Modes != 14 ||
		report.ModeExport.Logos != 1 || report.ModeExport.Banners != 1 {
		t.Fatalf("unexpected report %+v", report)
	}
	entries := readModeIndex(t, filepath.Join(options.OutputDir, "modes.json"))
	if len(entries) != 14 || entries[0].ID != "regular" {
		t.Fatalf("unexpected entries %+v", entries)
	}
	if entries[0].Logo == nil || *entries[0].Logo != "assets/modes/regular/logo.png" ||
		entries[0].Banner == nil || *entries[0].Banner != "assets/modes/regular/banner.png" {
		t.Fatalf("unexpected links %+v", entries[0])
	}
	snapshot, err := ReadClientExport(options.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Modes != nil {
		t.Fatalf("modes-only export repeated mode definitions in data.json: %+v", snapshot.Modes)
	}
}
