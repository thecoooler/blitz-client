package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func mechanicsFixture(t *testing.T) *filesFixture {
	t.Helper()
	f := newFilesFixture(t)
	f.write("game/Data/3d/Tanks/Parameters/ussr/TankA.yaml", "mechanics: [{name: DrumShot}, {name: PumpShot}]\n")
	f.write("game/Data/3d/Tanks/Parameters/ussr/TankB.yaml", "mechanics: [{name: LVArmor}, {name: AlteredSpotDuration}]\n")
	f.write("game/Data/Strings/ru.yaml", `"TankMechanics/DrumShot": "Барабанное орудие"
"TankMechanics/DrumShotDescription": "Описание барабана"
"TankMechanics/ArmorPumpShot": "Барабанное орудие с дозарядкой"
"TankMechanics/PumpShotDescription": "Описание дозарядки"
"TankMechanics/ArmorMoving": "Закалённые доспехи"
"TankMechanics/ArmorMovingDescription": "Описание брони"
"TankMechanics/ReducedSpotDuration": "Ускоренный выход из засвета"
"TankMechanics/ReducedSpotDurationDescription": "Описание засвета"
`)
	for _, directory := range []string{"drum-reload", "pump-action", "armor-moving", "reduced-spot-duration"} {
		f.packed("game/Data/Gfx/Lobby/icons/tank-mechanics/"+directory+"/bg_"+directory+"@2x.packed.webp.dvpl",
			packedWebP(t, "1\n\n4 2\n1\n0 0 4 2 0 0 0 \n"), 1)
		f.packed("game/Data/Gfx/Lobby/icons/tank-mechanics/"+directory+"/icon_"+directory+"_l@2x.packed.webp.dvpl",
			packedWebP(t, "1\n\n2 2\n1\n0 0 2 2 0 0 0 \n"), 1)
	}
	// Smaller resolutions must not be chosen when @2x and _l exist.
	f.write("game/Data/Gfx/Lobby/icons/tank-mechanics/drum-reload/bg_drum-reload.packed.webp",
		packedWebP(t, "1\n\n1 1\n1\n0 0 1 1 0 0 0 \n"))
	f.write("game/Data/Gfx/Lobby/icons/tank-mechanics/drum-reload/icon_drum-reload_s@2x.packed.webp",
		packedWebP(t, "1\n\n1 1\n1\n0 0 1 1 0 0 0 \n"))
	return f
}

func collectMechanicFiles(t *testing.T, f *filesFixture) map[string]sourceFile {
	t.Helper()
	files := map[string]sourceFile{}
	skipped := 0
	err := collectFiles(f.path("game/Data"), ClientFilesExportOptions{
		Types: []ClientFileType{FileTypeConfigs, FileTypeImages},
	}, false, files, &skipped)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func readMechanicIndex(t *testing.T, path string) []MechanicExport {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entries []MechanicExport
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestExportMechanicsIndexesTanksLocalizationAndArtwork(t *testing.T) {
	f := mechanicsFixture(t)
	snapshot := &Snapshot{Tanks: []SnapshotTank{
		{ID: "ussr:1", Nation: "ussr", TechName: "TankA"},
		{ID: "ussr:2", Nation: "ussr", TechName: "TankB"},
	}}
	report, err := exportMechanics(collectMechanicFiles(t, f), f.path("game/Data"), f.path("output"), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if report.Mechanics != 4 || report.Images != 4 || report.Icons != 4 {
		t.Fatalf("unexpected report %+v", report)
	}
	entries := readMechanicIndex(t, report.File)
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	if !slices.Equal(ids, []string{"AlteredSpotDuration", "DrumShot", "LVArmor", "PumpShot"}) {
		t.Fatalf("unexpected order %v", ids)
	}

	drum := entries[1]
	if !slices.Equal(drum.Tanks, []string{"ussr:1"}) ||
		drum.NameKey != "TankMechanics/DrumShot" || drum.Name != "Барабанное орудие" ||
		drum.DescriptionKey != "TankMechanics/DrumShotDescription" || drum.Description != "Описание барабана" {
		t.Fatalf("unexpected drum entry %+v", drum)
	}
	if drum.Image == nil || *drum.Image != "assets/mechanics/drum-reload/image.png" ||
		drum.Icon == nil || *drum.Icon != "assets/mechanics/drum-reload/icon.png" {
		t.Fatalf("unexpected drum links %+v", drum)
	}
	if drum.ImageSource == nil || drum.ImageSource.Path != "Gfx/Lobby/icons/tank-mechanics/drum-reload/bg_drum-reload@2x.packed.webp" {
		t.Fatalf("unexpected image source %+v", drum.ImageSource)
	}
	if drum.IconSource == nil || drum.IconSource.Path != "Gfx/Lobby/icons/tank-mechanics/drum-reload/icon_drum-reload_l@2x.packed.webp" {
		t.Fatalf("unexpected icon source %+v", drum.IconSource)
	}
	if image := readPNG(t, filepath.Join(f.path("output"), *drum.Image)); image.Bounds().Dx() != 4 || image.Bounds().Dy() != 2 {
		t.Fatalf("unexpected image size %v", image.Bounds())
	}
	if icon := readPNG(t, filepath.Join(f.path("output"), *drum.Icon)); icon.Bounds().Dx() != 2 || icon.Bounds().Dy() != 2 {
		t.Fatalf("unexpected icon size %v", icon.Bounds())
	}
	if files, err := os.ReadDir(f.path("output/assets/mechanics/drum-reload")); err != nil || len(files) != 2 {
		t.Fatalf("only the main image and icon are exported: %v %v", files, err)
	}

	pump := entries[3]
	if pump.NameKey != "TankMechanics/ArmorPumpShot" || pump.DescriptionKey != "TankMechanics/PumpShotDescription" ||
		pump.Name != "Барабанное орудие с дозарядкой" || pump.Description != "Описание дозарядки" ||
		pump.Image == nil || *pump.Image != "assets/mechanics/pump-action/image.png" {
		t.Fatalf("unexpected pump entry %+v", pump)
	}

	armor := entries[2]
	if !slices.Equal(armor.Tanks, []string{"ussr:2"}) ||
		armor.NameKey != "TankMechanics/ArmorMoving" || armor.Name != "Закалённые доспехи" ||
		armor.Image == nil || *armor.Image != "assets/mechanics/armor-moving/image.png" {
		t.Fatalf("unexpected LVArmor entry %+v", armor)
	}

	spot := entries[0]
	if spot.NameKey != "TankMechanics/ReducedSpotDuration" || spot.Name != "Ускоренный выход из засвета" ||
		spot.Image == nil || *spot.Image != "assets/mechanics/reduced-spot-duration/image.png" {
		t.Fatalf("unexpected AlteredSpotDuration entry %+v", spot)
	}
	if len(snapshot.Issues) != 0 {
		t.Fatalf("unexpected issues %+v", snapshot.Issues)
	}
}

func TestExportMechanicsWithoutArtwork(t *testing.T) {
	f := newFilesFixture(t)
	f.write("game/Data/3d/Tanks/Parameters/ussr/Tank.yaml", "mechanics:\n  - name: GunCooling\n")
	snapshot := &Snapshot{Tanks: []SnapshotTank{{ID: "ussr:1", Nation: "ussr", TechName: "Tank"}}}
	report, err := exportMechanics(collectMechanicFiles(t, f), f.path("game/Data"), f.path("output"), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if report.Mechanics != 1 || report.Images != 0 || report.Icons != 0 {
		t.Fatalf("unexpected report %+v", report)
	}
	entries := readMechanicIndex(t, report.File)
	entry := entries[0]
	if entry.ID != "GunCooling" || entry.Image != nil || entry.Icon != nil ||
		entry.NameKey != "TankMechanics/GunCooling" || entry.Name != "" || entry.Description != "" {
		t.Fatalf("unexpected entry %+v", entry)
	}
	if len(snapshot.Issues) != 0 {
		t.Fatalf("missing optional artwork must not be an issue: %+v", snapshot.Issues)
	}
}

func TestExportMechanicsSection(t *testing.T) {
	f, options := fullDataFixture(t)
	options.Include = []string{"mechanics"}
	// A mechanics-only export must not read the full data document set.
	for _, relative := range []string{
		"game/Data/XML/item_defs/vehicles/common/optional_devices.xml",
		"game/Data/XML/item_defs/vehicles/common/optional_device_slots.xml",
		"game/Data/XML/item_defs/vehicles/common/provisions/list.xml",
	} {
		if err := os.Remove(f.path(relative)); err != nil {
			t.Fatal(err)
		}
	}
	f.packed("game/Data/Gfx/Lobby/icons/tank-mechanics/test/bg_test@2x.packed.webp.dvpl",
		packedWebP(t, "1\n\n4 2\n1\n0 0 4 2 0 0 0 \n"), 1)
	f.packed("game/Data/Gfx/Lobby/icons/tank-mechanics/test/icon_test_l@2x.packed.webp.dvpl",
		packedWebP(t, "1\n\n2 2\n1\n0 0 2 2 0 0 0 \n"), 1)
	report, err := ExportClient(options)
	if err != nil {
		t.Fatal(err)
	}
	if report.MechanicExport == nil || report.MechanicExport.Mechanics != 1 ||
		report.MechanicExport.Images != 1 || report.MechanicExport.Icons != 1 {
		t.Fatalf("unexpected report %+v", report)
	}
	entries := readMechanicIndex(t, filepath.Join(options.OutputDir, "mechanics.json"))
	if len(entries) != 1 || entries[0].ID != "Test" || !slices.Equal(entries[0].Tanks, []string{"ussr:0", "usa:0"}) {
		t.Fatalf("unexpected entries %+v", entries)
	}
	if entries[0].Image == nil || *entries[0].Image != "assets/mechanics/test/image.png" ||
		entries[0].Icon == nil || *entries[0].Icon != "assets/mechanics/test/icon.png" {
		t.Fatalf("unexpected links %+v", entries[0])
	}
}

func TestMechanicKebabCaseFallback(t *testing.T) {
	for value, want := range map[string]string{
		"LTCamo":                     "lt-camo",
		"TTHOnMaps":                  "tth-on-maps",
		"TwinGuns":                   "twin-guns",
		"ShotDispersionStabilizator": "shot-dispersion-stabilizator",
	} {
		if got := kebabCase(value); got != want {
			t.Fatalf("kebabCase(%q) = %q, want %q", value, got, want)
		}
	}
}
