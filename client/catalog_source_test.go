package client

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/thecoooler/blitz-client/resources/xml"
)

func TestReadAvailableNationsFallsBackAndRejectsEmpty(t *testing.T) {
	f := newFixture(t)
	source := directorySource{dir: f.data}
	nations, err := ReadAvailableNations(source)
	if err != nil {
		t.Fatal(err)
	}
	expected := make([]string, len(Nations()))
	for index, nation := range Nations() {
		expected[index] = nation.ID
	}
	if !reflect.DeepEqual(nations, expected) {
		t.Fatalf("fallback nations: %v", nations)
	}

	path := f.write("available_nations.yaml", "available_nations:\n  - ussr\n  - usa\n")
	nations, err = ReadAvailableNations(source)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nations, []string{"ussr", "usa"}) {
		t.Fatalf("nations: %v", nations)
	}

	f.write("available_nations.yaml", "available_nations: []\n")
	_, err = ReadAvailableNations(source)
	requireError(t, err, "CLIENT_YAML_INVALID", path)
}

func TestDecodeLocalizationKeepsNullsAndProjectsStrings(t *testing.T) {
	localization, err := DecodeLocalization([]byte("\"#name\": \"Танк\"\n\"#null\": null\n"))
	if err != nil {
		t.Fatal(err)
	}
	if value := localization["#name"]; value == nil || *value != "Танк" {
		t.Fatalf("missing value: %#v", localization)
	}
	if value, ok := localization["#null"]; !ok || value != nil {
		t.Fatalf("null entry was not preserved: %#v", localization)
	}
	resolved := localization.Resolved()
	if resolved["#name"] != "Танк" {
		t.Fatalf("resolved names: %v", resolved)
	}
	if _, ok := resolved["#null"]; ok {
		t.Fatalf("null entered the resolved projection: %v", resolved)
	}
}

func TestReadLocalizationMissingIsEmpty(t *testing.T) {
	localization, err := ReadLocalization(directorySource{dir: newFixture(t).data})
	if err != nil {
		t.Fatal(err)
	}
	if len(localization) != 0 {
		t.Fatalf("missing localization: %v", localization)
	}
}

func TestReadMapsCatalogResolvesAliasesAndValidatesEntries(t *testing.T) {
	f := newFixture(t)
	f.write("maps.yaml", "maps:\n"+
		"    first:\n"+
		"        id: 7\n"+
		"        localName: ONE\n"+
		"        frame: 4\n"+
		"        avaliableInTrainingRoom: true\n"+
		"        availableModes: [1, 2]\n"+
		"    second:\n"+
		"        id: 8\n"+
		"        localName: TWO\n"+
		"        spriteFrame: 5\n"+
		"        availableInTrainingRoom: true\n")
	maps, err := ReadMapsCatalog(f.data)
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 2 || maps[0].SpriteFrame != 4 || !maps[0].AvailableInTrainingRoom ||
		!reflect.DeepEqual(maps[0].AvailableModes, []int64{1, 2}) {
		t.Fatalf("first map: %+v", maps)
	}
	if maps[1].SpriteFrame != 5 || !maps[1].AvailableInTrainingRoom || len(maps[1].AvailableModes) != 0 {
		t.Fatalf("second map: %+v", maps[1])
	}

	f.write("maps.yaml", "maps:\n    broken:\n        localName: NAME\n        frame: 1\n")
	_, err = ReadMapsCatalog(f.data)
	requireError(t, err, "CLIENT_YAML_INVALID", filepath.Join(f.data, "maps.yaml"))
	f.write("maps.yaml", "maps:\n    broken:\n        id: 1\n        localName: NAME\n")
	_, err = ReadMapsCatalog(f.data)
	requireError(t, err, "CLIENT_YAML_INVALID", filepath.Join(f.data, "maps.yaml"))
}

func TestReadVehicleListSharesIdentityAndRejectsDuplicates(t *testing.T) {
	f := newFixture(t)
	source := directorySource{dir: f.data}
	if list, err := ReadVehicleList(source, "usa"); err != nil || list != nil {
		t.Fatalf("missing list: %v, %v", list, err)
	}

	path := f.list("usa", "<nextAvailableId>9</nextAvailableId>"+
		"<First><id>7</id><userString>#first</userString><level>3</level></First>"+
		"<Second><id>8</id></Second>")
	list, err := ReadVehicleList(source, "usa")
	if err != nil {
		t.Fatal(err)
	}
	if list.Path != path || len(list.Entries) != 2 {
		t.Fatalf("list: %+v", list)
	}
	first := list.Entries[0]
	if first.TechName != "First" || first.ID != "7" || first.UserString != "#first" || !first.HasLevel || first.Level != "3" {
		t.Fatalf("entry: %+v", first)
	}

	f.list("usa", "<First><id>7</id></First><Second><id>7</id></Second>")
	_, err = ReadVehicleList(source, "usa")
	requireError(t, err, "CLIENT_XML_INVALID", path)

	f.list("usa", "<First><id>7</id></First><First><id>8</id></First>")
	_, err = ReadVehicleList(source, "usa")
	requireError(t, err, "CLIENT_XML_INVALID", path)
}

func TestCrewSkillSourceFeedsLocalizedAndRawProjections(t *testing.T) {
	f := newFixture(t)
	source := "<root><skillsByClasses><heavyTank>sniper</heavyTank></skillsByClasses><skills>" +
		"<sniper><userString>#sniper</userString><effectDescription>#effect</effectDescription>" +
		"<gunReloadTimeFactorPerLevel>0.01</gunReloadTimeFactorPerLevel></sniper></skills></root>"
	f.write("XML/item_defs/tankmen/avatar.xml", source)
	root, err := xml.Parse([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := ParseCrewSkills(root)
	if err != nil {
		t.Fatal(err)
	}
	localized, err := readCrewSkills(f.data, map[string]string{"#sniper": "Снайпер", "#effect": "Эффект"})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 1 || len(localized) != 1 || raw[0].ID != localized[0].ID {
		t.Fatalf("projections: %+v, %+v", raw, localized)
	}
	if raw[0].UserString != "#sniper" || raw[0].Parameters["gunReloadTimeFactorPerLevel"] != 0.01 {
		t.Fatalf("raw projection: %+v", raw[0])
	}
	if localized[0].UserString != "Снайпер" || localized[0].Description != "Эффект" || localized[0].Parameters != nil {
		t.Fatalf("localized projection: %+v", localized[0])
	}
}
