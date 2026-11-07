package cli

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDataExportCLI(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "Data")
	out := filepath.Join(root, "full")
	files := map[string]string{
		"available_nations.yaml":                                  "available_nations:\n  - ussr\n",
		"maps.yaml":                                               "maps:\n    test:\n        id: 1\n        frame: 0\n",
		"Strings/ru.yaml":                                         "name: Танк",
		"XML/item_defs/tankmen/avatar.xml":                        `<root><skillsByClasses><heavyTank>desperado</heavyTank></skillsByClasses><skills><desperado><userString>name</userString><gunReloadTimeFactorPerLevel>0.01</gunReloadTimeFactorPerLevel></desperado></skills></root>`,
		"XML/item_defs/vehicles/ussr/list.xml":                    `<root><Tank><id>0</id><userString>name</userString><price>100<gold/></price><enrichmentPermanentCost>0</enrichmentPermanentCost><tags>heavyTank</tags><level>1</level><configurationModes>production</configurationModes></Tank></root>`,
		"XML/item_defs/vehicles/ussr/Tank.xml":                    `<root><hull/></root>`,
		"XML/item_defs/vehicles/ussr/components/guns.xml":         `<root><shared/></root>`,
		"XML/item_defs/vehicles/ussr/components/shells.xml":       `<root/>`,
		"3d/Tanks/Parameters/ussr/Tank.yaml":                      "mechanics: []",
		"XML/item_defs/vehicles/common/optional_devices.xml":      `<root/>`,
		"XML/item_defs/vehicles/common/optional_device_slots.xml": `<root><presets/></root>`,
		"XML/item_defs/vehicles/common/provisions/list.xml":       `<root><items path="common.xml"/></root>`,
		"XML/item_defs/vehicles/common/provisions/common.xml":     `<root/>`,
	}
	for rel, contents := range files {
		p := filepath.Join(game, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	var config bytes.Buffer
	writer := zlib.NewWriter(&config)
	if _, err := writer.Write([]byte("(dS'gradeUpSkillCost'\n(lI25000\nas.")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "server_config.dat")
	if err := os.WriteFile(configPath, config.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	r := run(t, nil, "--json", "client", "export", "--game-dir", game, "--output-dir", out, "--no-cache", "--include", "data", "--include", "icons", "--no-pbr", "--server-config", configPath)
	if r.code != 0 {
		t.Fatalf("%s %s", r.stdout, r.stderr)
	}
	if r.json(t)["tanks"] != float64(1) {
		t.Fatal("wrong format")
	}
	if included := r.json(t)["included"].([]any); len(included) != 2 || included[0] != "data" || included[1] != "icons" {
		t.Fatalf("repeated --include lost sections: %v", included)
	}
	if _, err := os.Stat(filepath.Join(out, "data.json")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		CrewSkills []struct {
			ID     string
			Levels []struct{ XP int64 }
		}
		CrewSkillTraining struct{ MaxLevel int }
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.CrewSkills) != 1 || snapshot.CrewSkills[0].ID != "desperado" || len(snapshot.CrewSkills[0].Levels) != 1 || snapshot.CrewSkills[0].Levels[0].XP != 25000 || snapshot.CrewSkillTraining.MaxLevel != 1 {
		t.Fatalf("crew skills/config flag lost: %+v", snapshot)
	}
	for _, args := range [][]string{{"--data-only"}, {"--no-cache", "--cache-dir", root}, {"--include", "unknown"}, {"--include", "all,data"}, {"--skip-images"}} {
		r := run(t, nil, append([]string{"client", "export"}, args...)...)
		if r.code == 0 {
			t.Fatalf("invalid arguments accepted: %v", args)
		}
	}
}

func TestUnifiedExportDefaultAndRemovedCommands(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "Data")
	files := map[string]string{
		"available_nations.yaml":                                  "available_nations:\n  - ussr\n",
		"XML/item_defs/vehicles/ussr/list.xml":                    "<root/>",
		"XML/item_defs/vehicles/ussr/components/guns.xml":         "<root><shared/></root>",
		"XML/item_defs/vehicles/ussr/components/shells.xml":       "<root/>",
		"Strings/ru.yaml":                                         "{}",
		"XML/item_defs/vehicles/common/optional_devices.xml":      "<root/>",
		"XML/item_defs/vehicles/common/optional_device_slots.xml": "<root><presets/></root>",
		"XML/item_defs/vehicles/common/provisions/list.xml":       "<root/>",
	}
	for rel, body := range files {
		path := filepath.Join(game, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	r := run(t, nil, "--json", "client", "export", "--game-dir", game, "--output-dir", filepath.Join(root, "output"), "--no-cache")
	if r.code != 0 {
		t.Fatalf("%s %s", r.stdout, r.stderr)
	}
	included := r.json(t)["included"].([]any)
	if len(included) != 4 || included[0] != "data" || included[1] != "icons" || included[2] != "models" || included[3] != "maps" {
		t.Fatalf("wrong defaults: %v", included)
	}
	for _, command := range [][]string{{"client", "data", "export"}, {"client", "tanks", "export"}, {"client", "files", "export"}} {
		if r := run(t, nil, command...); r.code == 0 {
			t.Fatalf("obsolete export accepted: %v", command)
		}
	}
}
