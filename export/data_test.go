package export

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	ka "github.com/thecoooler/blitz-client/internal/katest"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestCrewSkillsExport(t *testing.T) {
	for _, withConfig := range []bool{false, true} {
		t.Run(fmt.Sprint(withConfig), func(t *testing.T) {
			f, o := fullDataFixture(t)
			o.Include = []string{"data"}
			// The downloaded DVPL definition must override the installed resource.
			f.write("game/Data/XML/item_defs/tankmen/avatar.xml", `<root><skills><old/></skills></root>`)
			f.packed("packs/XML/item_defs/tankmen/avatar.xml.dvpl", `<root><skillsByClasses><heavyTank>desperado</heavyTank></skillsByClasses><skills><desperado><userString>#name</userString><type>continuous</type><vehicleHealthFraction>0.15</vehicleHealthFraction><gunReloadTimeFactorPerLevel>0.01</gunReloadTimeFactorPerLevel></desperado><sniper><chancePerLevel>0.02</chancePerLevel></sniper></skills></root>`, 1)
			if withConfig {
				var payload bytes.Buffer
				writer := zlib.NewWriter(&payload)
				if _, err := writer.Write([]byte("(dS'gradeUpSkillCost'\n(lI25000\naI50000\nas.")); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				f.write("server_config.dat", payload.String())
				o.ServerConfig = f.path("server_config.dat")
			}
			if _, err := ExportClient(o); err != nil {
				t.Fatal(err)
			}
			s, err := ReadClientExport(o.OutputDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(s.CrewSkills) != 2 || s.CrewSkills[0].ID != "desperado" || s.CrewSkills[1].ID != "sniper" {
				t.Fatalf("skills/order: %+v", s.CrewSkills)
			}
			skill := s.CrewSkills[0]
			if skill.UserString != "#name" || skill.TankClass != "heavyTank" || skill.Type != "continuous" || skill.Parameters["vehicleHealthFraction"] != json.Number("0.15") {
				t.Fatalf("skill fields: %+v", skill)
			}
			if withConfig {
				if s.CrewSkillTraining == nil || s.CrewSkillTraining.MaxLevel != 2 || len(skill.Levels) != 2 || skill.Levels[1].XP != 50000 || skill.Levels[1].Parameters["gunReloadTimeFactorPerLevel"] != 0.02 {
					t.Fatalf("training: %+v, skill: %+v", s.CrewSkillTraining, skill)
				}
			} else if s.CrewSkillTraining != nil || skill.Levels != nil {
				t.Fatal("invented training without server config")
			}
		})
	}
}

func TestHangarParametersExport(t *testing.T) {
	for _, include := range [][]string{{"data"}, {"icons"}} {
		t.Run(strings.Join(include, ","), func(t *testing.T) {
			f, o := fullDataFixture(t)
			o.Include = include
			f.write("game/Data/hangarParamsConfig.yaml", "keyedArchive: {gunAimingTime: {keyedArchive: {digits: {int32: 9}}}}\n")
			f.packed("packs/hangarParamsConfig.yaml.dvpl", "keyedArchive: {gunAimingTime: {keyedArchive: {digits: {int32: 1}, units: {string: SECOND}}}, unknown: [null, 2]}\n", 1)
			if _, err := ExportClient(o); err != nil {
				t.Fatal(err)
			}
			s, err := ReadClientExport(o.OutputDir)
			if err != nil {
				t.Fatal(err)
			}
			if include[0] == "icons" {
				if s.HangarParameters != nil {
					t.Fatal("icons read hangar configuration")
				}
				return
			}
			archive := s.HangarParameters.(map[string]any)["keyedArchive"].(map[string]any)
			digits := archive["gunAimingTime"].(map[string]any)["keyedArchive"].(map[string]any)["digits"].(map[string]any)["int32"]
			if digits != json.Number("1") || archive["unknown"].([]any)[0] != nil {
				t.Fatalf("raw config/overlay lost: %#v", archive)
			}
		})
	}
}

func fullDataFixture(t *testing.T) (*filesFixture, ClientExportOptions) {
	f := newFilesFixture(t)
	f.write("game/Data/available_nations.yaml", "available_nations:\n  - ussr\n  - usa\n")
	f.packed("game/Data/Strings/ru.yaml.dvpl", "\"#name\": \"Танк\"\n", 1)
	for _, nation := range []string{"ussr", "usa"} {
		base := "game/Data/XML/item_defs/vehicles/" + nation + "/"
		f.write(base+"list.xml", `<root><Tank><id>0</id><price>10<gold/></price><tags>heavyTank collectible</tags><configurationModes>production development</configurationModes><combatRole><arcade>support</arcade></combatRole></Tank></root>`)
		f.write(base+"Tank.xml", `<root><hull/><speedLimits><forward>48</forward><forward>50</forward></speedLimits><turrets0><Turret><guns><GunB>shared<reloadTime>4</reloadTime><shots><ShellB><shell>shared<price>2<gold/></price></shell></ShellB><ShellA><shell/></ShellA></shots></GunB><GunA>shared</GunA></guns></Turret></turrets0><optDevicePreset>default</optDevicePreset></root>`)
		f.write(base+"components/guns.xml", `<root><ids><GunB>1</GunB></ids><shared><GunB><shots><ShellB><speed>900</speed></ShellB></shots><clip><count>2</count></clip><burst/></GunB><GunA><reloadTime>5</reloadTime></GunA></shared></root>`)
		f.write(base+"components/shells.xml", `<root><icons/><ShellB><damage><armor>100</armor></damage></ShellB><ShellA><kind>ARMOR_PIERCING</kind></ShellA></root>`)
		f.write("game/Data/3d/Tanks/Parameters/"+nation+"/Tank.yaml", "resourcesPath:\n  bigIconPath: '~res:/Gfx/"+nation+"'\n  smallIconPath: '~res:/Gfx/missing'\n  nested: {foo: bar}\nmechanics: [{name: Test}]\nlevels: {1: [1, null, 3]}\n")
		f.packed("packs/Gfx/"+nation+".packed.webp.dvpl", "icon "+nation, 1)
	}
	common := "game/Data/XML/item_defs/vehicles/common/"
	f.write(common+"consumables/common.xml", `<root><RepairKit><id>1</id><userString>#kit</userString><icon>kit</icon></RepairKit></root>`)
	f.write("game/Data/Gfx/UI/InventoryIcons/Big/Consumables/kit.txt", "1\nkit-atlas.png\n2 2\n1\n0 0 1 1 1 1 0 frame0\n")
	f.write("game/Data/Gfx/UI/InventoryIcons/Big/Consumables/kit-atlas.png", pngBytes(t, 2, 2, func(x, y int) color.NRGBA {
		return color.NRGBA{uint8(x), uint8(y), 3, 255}
	}))
	f.write("game/Data/UI/Styles/Lobby/Inventory/InventoryBigStyles.style",
		`.equipment_item.kit #icon { bg-sprite: "~res:/Gfx/UI/InventoryIcons/Big/Consumables/kit.txt"; }`)
	f.write(common+"optional_devices.xml", `<root><nextAvailableId>2</nextAvailableId><Rammer><script><reloadFactor>0.9</reloadFactor></script></Rammer></root>`)
	f.write(common+"optional_device_slots.xml", `<root><presets><default><slot1>Rammer Armor</slot1><slot2/><slot3/><slot4/><slot5/><slot6/><slot7/><slot8/><slot9/></default></presets></root>`)
	f.write(common+"provisions/list.xml", `<root><items path="common.xml"/><items path="mode.xml"/></root>`)
	for _, mode := range []string{"common", "mode"} {
		f.write(common+"provisions/"+mode+".xml", `<root><Food><vehicleFilter><include><nations>ussr</nations><levels>5 6</levels></include><exclude><vehicles>Tank2</vehicles></exclude></vehicleFilter><script><crewLevelIncrease>10</crewLevelIncrease></script></Food></root>`)
	}
	o := NewClientExportOptions(f.path("output"))
	o.GameDir = f.path("game")
	o.CacheDir = f.path("packs")
	o.Include = []string{"data", "icons"}
	return f, o
}

func readDataJSON(t *testing.T, root, rel string, v any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}
func TestUnifiedExport(t *testing.T) {
	f, o := fullDataFixture(t)
	f.write("packs/XML/item_defs/vehicles/ussr/components/guns.xml", `<root><shared><GunB><reloadTime>7</reloadTime></GunB><GunA><reloadTime>5</reloadTime></GunA></shared></root>`)
	report, err := ExportClient(o)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ReadClientExport(o.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Version != 1 || s.Format != "blitz-client" || len(s.Tanks) != 2 || len(report.Issues) != 2 {
		t.Fatalf("wrong snapshot: %+v", report)
	}
	a, b := s.Tanks[0], s.Tanks[1]
	if a.ID != "ussr:0" || b.ID != "usa:0" || a.TechName != b.TechName {
		t.Fatal("identities/namespaces lost")
	}
	metadata := a.Metadata.(map[string]any)
	if _, ok := metadata["id"]; ok {
		t.Fatal("duplicated numeric id")
	}
	if metadata["price"].(map[string]any)["#text"] != "10" || metadata["price"].(map[string]any)["gold"] != nil {
		t.Fatal("gold price lost")
	}
	cfg := a.Configuration.(map[string]any)
	if !reflect.DeepEqual(cfg["speedLimits"].(map[string]any)["forward"], []any{"48", "50"}) {
		t.Fatal("repeated XML scalar/list lost")
	}
	if cfg["hull"] != nil {
		t.Fatal("empty element changed")
	}
	turrets := cfg["turrets0"].([]any)
	guns := turrets[0].(map[string]any)["properties"].(map[string]any)["guns"].([]any)
	gun := guns[0].(map[string]any)
	if gun["name"] != "GunB" || guns[1].(map[string]any)["name"] != "GunA" {
		t.Fatal("gun order lost")
	}
	props := gun["properties"].(map[string]any)
	if props["#text"] != "shared" || props["reloadTime"] != "4" {
		t.Fatal("shared and local overrides lost")
	}
	shots := props["shots"].([]any)
	if shots[0].(map[string]any)["name"] != "ShellB" || shots[1].(map[string]any)["name"] != "ShellA" || shots[1].(map[string]any)["properties"].(map[string]any)["shell"] != nil {
		t.Fatal("ordered nullable shots lost")
	}
	shared := s.Components["ussr"]["guns"].(map[string]any)["shared"].(map[string]any)
	if shared["GunB"].(map[string]any)["reloadTime"] != "7" {
		t.Fatal("cache overlay lost")
	}
	if s.Components["usa"]["shells"] == nil || len(s.Provisions) != 2 || s.Provisions["common"].(map[string]any)["Food"].(map[string]any)["vehicleFilter"] == nil {
		t.Fatal("catalog/filter/include lost")
	}
	if a.Parameters.(map[string]any)["levels"].(map[string]any)["1"].([]any)[1] != nil {
		t.Fatal("YAML null lost")
	}
	icon, err := os.ReadFile(filepath.Join(o.OutputDir, *a.Icons["big"]))
	if err != nil || string(icon) != "icon ussr" {
		t.Fatalf("DVPL icon %q %v", icon, err)
	}
	entries, err := os.ReadDir(o.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		switch entry.Name() {
		case "data.json", "images", "items.json", "assets":
		default:
			t.Fatalf("duplicate/fragmented output: %s", entry.Name())
		}
	}
	if report.ItemsFile == "" || report.IconExport == nil || report.IconExport.IconCount != 1 {
		t.Fatalf("item icon export missing: %+v", report)
	}
	var items ItemsFile
	readDataJSON(t, o.OutputDir, "items.json", &items)
	if len(items.Consumables) != 1 || items.Consumables[0].Icon == nil || *items.Consumables[0].Icon != "assets/icons/consumables/kit.png" {
		t.Fatalf("item icon link lost: %+v", items.Consumables)
	}
	if _, err := os.Stat(filepath.Join(o.OutputDir, *items.Consumables[0].Icon)); err != nil {
		t.Fatal(err)
	}
	if _, err := ExportClient(o); err == nil {
		t.Fatal("nonempty output accepted")
	}
}

func TestExportSectionSelection(t *testing.T) {
	for _, parts := range [][]string{{"data"}, {"icons"}, {"models"}, {"data", "models"}, {"all"}, nil} {
		t.Run(strings.Join(parts, "+"), func(t *testing.T) {
			f, o := fullDataFixture(t)
			o.Include = parts
			// Minimal real model, no external codecs: all materials use generated PNGs.
			f.write("packs/3d/Tanks/USSR/Tank.sc2", string(scene(sceneNode("hull", 1))))
			f.packed("packs/3d/Tanks/USSR/Tank.scg.dvpl", string(ka.SCG(ka.Triangle(1, false))), 1)
			o.ModelOptions.TextureFormat = "png"
			o.ModelOptions.Compress = false
			if len(parts) == 1 && parts[0] == "models" {
				// Media-only export must not require full website data or tank parameters.
				if err := os.Remove(f.path("game/Data/3d/Tanks/Parameters/ussr/Tank.yaml")); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(f.path("game/Data/XML/item_defs/vehicles/common/optional_devices.xml")); err != nil {
					t.Fatal(err)
				}
			}
			report, err := ExportClient(o)
			if err != nil {
				t.Fatal(err)
			}
			var s Snapshot
			readDataJSON(t, o.OutputDir, "data.json", &s)
			includes, _ := ParseExportIncludes(parts)
			has := func(part string) bool {
				for _, v := range includes {
					if v == part {
						return true
					}
				}
				return false
			}
			if (s.Tanks[0].Metadata != nil) != has("data") || (s.Tanks[0].Icons != nil) != has("icons") || (s.Tanks[0].Model != nil) != has("models") {
				t.Fatalf("section selection failed: %+v", s.Tanks[0])
			}
			if has("models") {
				if report.Models != 1 {
					t.Fatalf("model export failed: %+v", report)
				}
				if _, err := os.Stat(filepath.Join(o.OutputDir, "assets/tanks/catalog.json")); !os.IsNotExist(err) {
					t.Fatal("duplicate model catalog")
				}
				modelPath := s.Tanks[0].Model.Model
				var model map[string]any
				readDataJSON(t, o.OutputDir, modelPath, &model)
				if _, err := os.Stat(filepath.Join(o.OutputDir, "assets/tanks/ussr/Tank/model.pbr.bin")); err != nil {
					t.Fatalf("PBR missing: %v", err)
				}
				if model["source"].(map[string]any)["scene"] != "3d/Tanks/USSR/Tank.sc2" {
					t.Fatal("model embeds a temporary/machine source path")
				}
				if model["binaryGeometry"] == nil {
					t.Fatal("model consumer cannot find geometry")
				}
				var raw map[string]any
				readDataJSON(t, o.OutputDir, "data.json", &raw)
				if _, ok := raw["tanks"].([]any)[0].(map[string]any)["model"].(map[string]any)["id"]; ok {
					t.Fatal("duplicated model identity")
				}
			}
		})
	}
}

func TestExportFailures(t *testing.T) {
	for _, scenario := range []string{"missing", "duplicate", "include", "overlap", "crew-classes", "server-config", "server-config-section"} {
		t.Run(scenario, func(t *testing.T) {
			f, o := fullDataFixture(t)
			o.Include = []string{"data"}
			switch scenario {
			case "crew-classes":
				f.write("game/Data/XML/item_defs/tankmen/avatar.xml", `<root><skillsByClasses><heavyTank>sniper</heavyTank><lightTank>sniper</lightTank></skillsByClasses><skills><sniper/></skills></root>`)
			case "server-config":
				o.ServerConfig = f.path("broken.dat")
				f.write("broken.dat", "invalid")
			case "server-config-section":
				o.Include = []string{"icons"}
				o.ServerConfig = f.path("missing.dat")
			case "missing":
				if err := os.Remove(f.path("game/Data/XML/item_defs/vehicles/ussr/Tank.xml")); err != nil {
					t.Fatal(err)
				}
			case "duplicate":
				f.write("game/Data/XML/item_defs/vehicles/ussr/list.xml", `<root><Tank><id>0</id></Tank><Other><id>0</id></Other></root>`)
			case "include":
				f.write("game/Data/XML/item_defs/vehicles/common/provisions/list.xml", `<root><items path="../escape.xml"/></root>`)
			case "overlap":
				o.OutputDir = f.path("game/Data/export")
			}
			if _, err := ExportClient(o); err == nil {
				t.Fatal("invalid/incomplete export accepted")
			}
			if entries, _ := os.ReadDir(o.OutputDir); len(entries) != 0 {
				t.Fatal("failed export left partial output")
			}
		})
	}
}

func TestModelFailuresForEveryTankAbortAndCleanUp(t *testing.T) {
	f, o := fullDataFixture(t)
	o.Include = []string{"icons", "models"}
	f.write("packs/3d/Tanks/USSR/Tank.sc2", string(scene(sceneNode("hull", 1))))
	f.packed("packs/3d/Tanks/USSR/Tank.scg.dvpl", string(ka.SCG(ka.Triangle(1, false))), 1)
	o.ModelOptions.TextureFormat = "png"
	t.Setenv("PATH", t.TempDir()) // no brotli
	_, err := ExportClient(o)
	if err == nil || !strings.Contains(err.Error(), "BROTLI_NOT_FOUND") {
		t.Fatalf("missing brotli was not fatal: %v", err)
	}
	if _, err := os.Stat(o.OutputDir); !os.IsNotExist(err) {
		t.Fatal("failed export left partial media")
	}
}

func TestOptionalClientResources(t *testing.T) {
	f, o := fullDataFixture(t)
	o.Include = []string{"icons"}
	if err := os.Remove(f.path("game/Data/available_nations.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(f.path("game/Data/3d/Tanks/Parameters/usa/Tank.yaml")); err != nil {
		t.Fatal(err)
	}
	report, err := ExportClient(o)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ReadClientExport(o.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	// Built-in nations are used; nations without list.xml are skipped.
	if !reflect.DeepEqual(s.Nations, []string{"usa", "ussr"}) {
		t.Fatalf("nations: %v", s.Nations)
	}
	found := false
	for _, issue := range report.Issues {
		found = found || (issue.Tank == "usa:0" && strings.Contains(issue.Message, "parameters"))
	}
	if !found {
		t.Fatalf("missing parameters not reported: %+v", report.Issues)
	}
}

func TestYAMLScalarKeysAndNonfiniteValues(t *testing.T) {
	var v any
	if err := yaml.Unmarshal([]byte("0.5: a\ntrue: b\n~: c\n1: d\nx: .inf\ny: [-.inf, .nan]\n"), &v); err != nil {
		t.Fatal(err)
	}
	got, err := dataJSONValue(v)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"0.5": "a", "true": "b", "null": "c", "1": "d", "x": ".inf", "y": []any{"-.inf", ".nan"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
	if _, err := json.Marshal(got); err != nil {
		t.Fatal(err)
	}
}

func TestParseIncludes(t *testing.T) {
	for _, input := range [][]string{nil, {"all"}, {"models, data"}, {"icons", "data", "icons"}} {
		got, err := ParseExportIncludes(input)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) == 0 {
			t.Fatal("empty selection")
		}
		if len(input) == 0 && !reflect.DeepEqual(got, []string{"data", "icons", "models"}) {
			t.Fatal(got)
		}
	}
	for _, input := range [][]string{{""}, {"all,data"}, {"unknown"}, {"data,"}} {
		if _, err := ParseExportIncludes(input); err == nil {
			t.Fatalf("accepted %v", input)
		}
	}
}

func TestReadClientExportRejectsUnknownFormats(t *testing.T) {
	for _, body := range []string{
		`{"format":"other","version":1,"included":["data"]}`,
		`{"format":"blitz-client","version":2,"included":["data"]}`,
		`{"format":"blitz-client","version":1}`,
		`{"format":"blitz-client","version":1,"included":["unknown"]}`,
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadClientExport(dir); err == nil {
			t.Fatalf("unsupported export accepted: %s", body)
		}
	}
}

func TestResolverFindsUnpackedCacheTextures(t *testing.T) {
	files := map[string]sourceFile{
		filepath.FromSlash("3d/Tanks/USSR/images/hull.dx11.dds"): {path: "/cache/hull.dx11.dds", fromCache: true},
	}
	resolve := filesResolver(files)
	// Material paths are rewritten to packed names relative to the scene directory.
	if got := resolve("3d/Tanks/USSR/../USSR/images/hull.dx11.dds.dvpl"); got != "/cache/hull.dx11.dds" {
		t.Fatalf("resolved %q", got)
	}
	if got := resolve("../outside.dds"); got != "" {
		t.Fatalf("escaped root: %q", got)
	}
}

func TestReadClientExportKeepsLargeIntegers(t *testing.T) {
	dir := t.TempDir()
	body := `{"format":"blitz-client","version":1,"included":["data"],"tanks":[{"id":"ussr:0","nation":"ussr","clientId":0,"techName":"T","parameters":{"mask":9007199254740993}}]}`
	if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := ReadClientExport(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Tanks[0].Parameters.(map[string]any)["mask"]; got != json.Number("9007199254740993") {
		t.Fatalf("large integer rounded: %v", got)
	}
}

func TestSymlinkedDataStillRejectsOverlappingOutput(t *testing.T) {
	f, o := fullDataFixture(t)
	o.Include = []string{"data"}
	// The install directory is real; only Data inside it is a symlink.
	if err := os.MkdirAll(f.path("install"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(f.path("game/Data"), f.path("install/Data")); err != nil {
		t.Skip(err)
	}
	o.GameDir = f.path("install")
	o.OutputDir = f.path("game/Data/export")
	if _, err := ExportClient(o); err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("overlap through symlink accepted: %v", err)
	}
}

func TestUnreadableModelResourceIsFatal(t *testing.T) {
	f, o := fullDataFixture(t)
	o.Include = []string{"models"}
	o.ModelOptions.TextureFormat = "png"
	o.ModelOptions.Compress = false
	f.write("packs/3d/Tanks/USSR/Tank.sc2", string(scene(sceneNode("hull", 1))))
	f.packed("packs/3d/Tanks/USSR/Tank.scg.dvpl", string(ka.SCG(ka.Triangle(1, false))), 1)
	if err := os.Chmod(f.path("packs/3d/Tanks/USSR/Tank.scg.dvpl"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(f.path("packs/3d/Tanks/USSR/Tank.scg.dvpl")); err == nil {
		t.Skip("permissions are not enforced")
	}
	if _, err := ExportClient(o); err == nil || !strings.Contains(err.Error(), "IO_ERROR") {
		t.Fatalf("unreadable resource was not fatal: %v", err)
	}
}
