package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/resources"
)

type fixture struct {
	t    *testing.T
	data string
}

func newFixture(t *testing.T) *fixture {
	data := filepath.Join(t.TempDir(), "Data")
	if err := os.MkdirAll(filepath.Join(data, "XML/item_defs/vehicles"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, data: data}
}

func (f *fixture) write(relative string, data string) string {
	path := filepath.Join(f.data, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return path
}

func (f *fixture) writeDVPL(relative string, data string) string {
	encoded, err := resources.EncodeDVPL([]byte(data), resources.DVPL(2))
	if err != nil {
		f.t.Fatal(err)
	}
	return f.write(relative+".dvpl", string(encoded))
}

func (f *fixture) list(nation, entries string) string {
	return f.write("XML/item_defs/vehicles/"+nation+"/list.xml", "<root>"+entries+"</root>")
}

// pair writes a showroom model; discovery checks existence, not contents.
func (f *fixture) pair(folder, id string) {
	f.write("3d/Tanks/"+folder+"/"+id+".sc2", "")
	f.write("3d/Tanks/"+folder+"/"+id+".scg", "")
}

func (f *fixture) maps() {
	f.write("maps.yaml", "maps:\n    arena:\n        id: 7\n        localName: NAME\n        frame: 4\n        availableModes: [1, 2]\n        availableInTrainingRoom: true\n")
}

func completeTank(name, id string) string {
	return "<" + name + ">" +
		"<id>" + id + "</id><userString>#vehicles:" + name + "</userString>" +
		"<shortUserString>#short:" + name + "</shortUserString>" +
		"<description>#description:" + name + "</description>" +
		"<price>100<gold/></price><sellPrice>50</sellPrice>" +
		"<notInShop>true</notInShop><enrichmentPermanentCost>20</enrichmentPermanentCost>" +
		"<tags>lightTank regular</tags><level>3</level>" +
		"<combatRole><default>support</default></combatRole>" +
		"<configurationModes>default special</configurationModes>" +
		"<leasing>30.09.2026 12:34</leasing></" + name + ">"
}

func requireError(t *testing.T, err error, code, path string) *blitz.Error {
	t.Helper()
	typed := blitz.AsError(err)
	if typed == nil || typed.Code != code || (path != "" && typed.Path != path) {
		t.Fatalf("expected %s at %s, got %v", code, path, err)
	}
	return typed
}

func jsonValue(t *testing.T, value any) any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func ids[T any](items []T, id func(T) string) []string {
	out := make([]string, len(items))
	for index, item := range items {
		out[index] = id(item)
	}
	return out
}

func TestModelsDoNotRequireCatalogFieldsOrOptionalDisplayFields(t *testing.T) {
	f := newFixture(t)
	listPath := f.list("usa", "<Sparse_Tank/><Empty_Fields><userString> </userString><level/></Empty_Fields><Unpaired/>")
	f.pair("USA", "Sparse_Tank")
	f.pair("USA", "Empty_Fields")

	models, err := DiscoverClientTanks(f.data)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "Sparse_Tank" || models[0].Name != "Sparse Tank" || models[1].Name != "Empty Fields" {
		t.Fatalf("%+v", models)
	}
	for _, model := range models {
		if model.Level != nil || model.UserString != nil {
			t.Fatalf("%+v", model)
		}
	}
	_, err = DiscoverClientCatalog(f.data)
	requireError(t, err, "CLIENT_XML_INVALID", listPath)
}

func TestUnmodeledInvalidMetadataDoesNotBlockModelListing(t *testing.T) {
	f := newFixture(t)
	listPath := f.list("usa", completeTank("Modeled", "42")+
		"<Unmodeled><userString>unknown</userString><id>43</id><price>invalid</price></Unmodeled>")
	f.pair("USA", "Modeled")

	models, err := ListClientTanks(f.data)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "Modeled" || models[0].Name != "Modeled" || *models[0].UserString != "#vehicles:Modeled" {
		t.Fatalf("%+v", models)
	}
	_, err = DiscoverClientCatalog(f.data)
	typed := requireError(t, err, "CLIENT_XML_INVALID", listPath)
	if !strings.Contains(typed.Message, "invalid price") {
		t.Fatal(typed)
	}
}

func TestOnlyCompletePairsAreDiscoveredFromPlainAndDVPLResources(t *testing.T) {
	for _, compressedList := range []bool{false, true} {
		f := newFixture(t)
		list := "<root><First><userString>#first</userString><level>4</level></First><Second/><Scene_Only/><Geometry_Only/></root>"
		stringsYAML := "\"#first\": \"Первый\"\n"
		if compressedList {
			f.writeDVPL("XML/item_defs/vehicles/usa/list.xml", list)
			f.write("Strings/ru.yaml", stringsYAML)
		} else {
			f.write("XML/item_defs/vehicles/usa/list.xml", list)
			f.writeDVPL("Strings/ru.yaml", stringsYAML)
		}
		firstScene := f.write("3d/Tanks/USA/First.sc2", "")
		firstGeometry := f.writeDVPL("3d/Tanks/USA/First.scg", "")
		secondScene := f.writeDVPL("3d/Tanks/USA/Second.sc2", "")
		secondGeometry := f.write("3d/Tanks/USA/Second.scg", "")
		f.write("3d/Tanks/USA/Scene_Only.sc2", "")
		f.write("3d/Tanks/USA/Geometry_Only.scg", "")

		models, err := DiscoverClientTanks(f.data)
		if err != nil {
			t.Fatal(err)
		}
		if len(models) != 2 || models[0].Name != "Первый" || *models[0].Level != "4" ||
			models[0].ModelFiles != (TankModelFiles{firstScene, firstGeometry}) ||
			models[1].ModelFiles != (TankModelFiles{secondScene, secondGeometry}) {
			t.Fatalf("%+v", models)
		}
	}
}

func TestCatalogPreservesSchemaLocalizationAndDistinctModelIDs(t *testing.T) {
	f := newFixture(t)
	f.list("usa", completeTank("Model_Key", "42"))
	f.pair("USA", "Model_Key")
	f.maps()
	f.writeDVPL("Strings/ru.yaml", "\"#vehicles:Model_Key\": \"Танк\"\n\"#short:Model_Key\": \"Т\"\n\"#description:Model_Key\": \"Описание\"\n\"#maps:arena:NAME\": \"Карта\"\n")

	catalog, err := DiscoverClientCatalog(f.data)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tanks) != 1 || len(catalog.Maps) != 1 {
		t.Fatalf("%+v", catalog)
	}
	var expectedTank, expectedMap any
	json.Unmarshal([]byte(`{
		"id": "42", "userString": "Танк", "shortUserString": "Т",
		"description": "Описание", "price": {"value": 100, "is_gold": true},
		"sellPrice": {"value": 50, "is_gold": false}, "notInShop": true,
		"enrichmentPermanentCost": 20, "tags": ["lightTank", "regular"],
		"level": 3, "nation": "usa", "combatRole": {"default": "support"},
		"configurationModes": ["default", "special"], "leasing": "2026-09-30T12:34:00"
	}`), &expectedTank)
	json.Unmarshal([]byte(`{"id": 7, "localName": "Карта", "availableInTrainingRoom": true, "availableModes": [1, 2], "spriteFrame": 4}`), &expectedMap)
	if got := jsonValue(t, catalog.Tanks[0]); !reflect.DeepEqual(got, expectedTank) {
		t.Fatalf("%v", got)
	}
	if got := jsonValue(t, catalog.Maps[0]); !reflect.DeepEqual(got, expectedMap) {
		t.Fatalf("%v", got)
	}
	models, err := DiscoverClientTanks(f.data)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "Model_Key" || models[0].Name != "Танк" || *models[0].UserString != "#vehicles:Model_Key" {
		t.Fatalf("%+v", models)
	}
	if !reflect.DeepEqual(jsonValue(t, catalog.ModelTanks), jsonValue(t, models)) {
		t.Fatal("model projections differ")
	}
}

func TestModelsPreserveLegacyNationOrderAndXMLOrder(t *testing.T) {
	f := newFixture(t)
	f.write("available_nations.yaml", "available_nations:\n  - ussr\n  - usa\n  - unknown\n")
	f.list("ussr", completeTank("RU_Second", "1")+completeTank("RU_First", "2"))
	f.list("usa", completeTank("US_Second", "3")+completeTank("US_First", "4"))
	f.list("unknown", completeTank("Unknown", "5"))
	for _, pair := range [][2]string{{"USSR", "RU_Second"}, {"USSR", "RU_First"}, {"USA", "US_Second"}, {"USA", "US_First"}, {"unknown", "Unknown"}} {
		f.pair(pair[0], pair[1])
	}
	f.maps()

	catalog, err := DiscoverClientCatalog(f.data)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(catalog.Tanks, func(tank ClientTankMetadata) string { return tank.ID }); !slices.Equal(got, []string{"1", "2", "3", "4", "5"}) {
		t.Fatal(got)
	}
	models, err := DiscoverClientTanks(f.data)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(models, func(tank ClientTank) string { return tank.ID }); !slices.Equal(got, []string{"US_Second", "US_First", "RU_Second", "RU_First"}) {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(jsonValue(t, catalog.ModelTanks), jsonValue(t, models)) {
		t.Fatal("model projections differ")
	}
}

func TestFullCatalogStillRejectsInvalidOptionalAndRequiredMetadata(t *testing.T) {
	valid := completeTank("Tank", "42")
	for _, replacement := range [][2]string{
		{"<id>42</id>", ""},
		{"<userString>#vehicles:Tank</userString>", ""},
		{"<price>100<gold/></price>", "<price>bad</price>"},
		{"<enrichmentPermanentCost>20</enrichmentPermanentCost>", "<enrichmentPermanentCost>bad</enrichmentPermanentCost>"},
		{"<level>3</level>", "<level>bad</level>"},
		{"<tags>lightTank regular</tags>", "<tags/>"},
		{"<configurationModes>default special</configurationModes>", "<configurationModes/>"},
		{"<sellPrice>50</sellPrice>", "<sellPrice>bad</sellPrice>"},
		{"<leasing>30.09.2026 12:34</leasing>", "<leasing>bad</leasing>"},
	} {
		f := newFixture(t)
		listPath := f.list("usa", strings.ReplaceAll(valid, replacement[0], replacement[1]))
		f.pair("USA", "Tank")
		if models, err := DiscoverClientTanks(f.data); err != nil || len(models) != 1 {
			t.Fatalf("model discovery for %s: %v", replacement[0], err)
		}
		_, err := DiscoverClientCatalog(f.data)
		requireError(t, err, "CLIENT_XML_INVALID", listPath)
	}
}

func TestModelDiscoveryKeepsXMLAndDVPLErrors(t *testing.T) {
	f := newFixture(t)
	path := f.write("XML/item_defs/vehicles/usa/list.xml", "<root><Tank></root>")
	_, err := DiscoverClientTanks(f.data)
	requireError(t, err, "CLIENT_XML_INVALID", path)
	_, err = DiscoverClientCatalog(f.data)
	requireError(t, err, "CLIENT_XML_INVALID", path)

	f = newFixture(t)
	path = f.writeDVPL("XML/item_defs/vehicles/usa/list.xml", "<root/>")
	data, _ := os.ReadFile(path)
	data[0] ^= 0xff
	os.WriteFile(path, data, 0o644)
	_, err = DiscoverClientTanks(f.data)
	requireError(t, err, "DVPL_CRC_FAILED", path)
	_, err = DiscoverClientCatalog(f.data)
	requireError(t, err, "DVPL_CRC_FAILED", path)
}

func TestModelDiscoveryDoesNotRequireMaps(t *testing.T) {
	f := newFixture(t)
	f.list("usa", completeTank("Tank", "42"))
	f.pair("USA", "Tank")
	if models, err := DiscoverClientTanks(f.data); err != nil || len(models) != 1 {
		t.Fatal(err)
	}
	_, err := DiscoverClientCatalog(f.data)
	requireError(t, err, "CLIENT_DATA_INVALID", f.data)
	f.maps()
	if catalog, err := DiscoverClientCatalog(f.data); err != nil || len(catalog.Tanks) != 1 {
		t.Fatal(err)
	}
}

func TestNormalizesDataAndAppDirectories(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "Game", "Data")
	app := filepath.Join(root, "Tanks Blitz.app", "Contents", "Resources", "Data")
	for _, dir := range []string{data, app} {
		os.MkdirAll(dir, 0o755)
	}
	canonicalRoot, _ := filepath.EvalSymlinks(root)
	for input, expected := range map[string][2]string{
		data:                                   {filepath.Join(canonicalRoot, "Game"), filepath.Join(canonicalRoot, "Game", "Data")},
		filepath.Join(root, "Game"):            {filepath.Join(canonicalRoot, "Game"), filepath.Join(canonicalRoot, "Game", "Data")},
		app:                                    {filepath.Join(canonicalRoot, "Tanks Blitz.app"), filepath.Join(canonicalRoot, "Tanks Blitz.app", "Contents", "Resources", "Data")},
		filepath.Join(root, "Tanks Blitz.app"): {filepath.Join(canonicalRoot, "Tanks Blitz.app"), filepath.Join(canonicalRoot, "Tanks Blitz.app", "Contents", "Resources", "Data")},
	} {
		resolved, err := ResolveGameDir(input)
		if err != nil {
			t.Fatal(err)
		}
		if resolved.GameDir != expected[0] || resolved.ResourceDir != expected[1] || resolved.Source != "explicit" {
			t.Fatalf("%s: %+v", input, resolved)
		}
	}
	_, err := ResolveGameDir(filepath.Join(root, "missing"))
	requireError(t, err, "GAME_DIR_NOT_FOUND", filepath.Join(root, "missing"))
}
