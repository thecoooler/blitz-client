package client

import (
	"strings"
	"testing"
)

func TestVehicleDisplayCatalogUsesProtocolNationIDsAndLocalizedNames(t *testing.T) {
	f := newFixture(t)
	f.maps()
	f.list("european", strings.Replace(completeTank("Polish", "28"), "<level>3</level>", "<level>10</level>", 1))
	f.list("china", completeTank("Chinese", "46"))
	f.list("usa", completeTank("American", "10"))
	f.writeDVPL("Strings/ru.yaml", "\"#vehicles:Polish\": \"60TP Lewandowskiego\"\n\"#short:Polish\": \"60TP\"\n\"#vehicles:Chinese\": \"BZ-75\"\n\"#vehicles:American\": \"T30\"\n")
	catalog, err := LoadVehicleDisplayCatalog(f.data)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[uint32]VehicleDisplay{
		7297:  {Name: "60TP", Level: 10, Class: VehicleClassLight},
		11825: {Name: "BZ-75", Level: 3, Class: VehicleClassLight},
		2593:  {Name: "T30", Level: 3, Class: VehicleClassLight},
	} {
		if got := catalog[id]; got != want {
			t.Fatalf("descriptor %d: got %+v want %+v", id, got, want)
		}
	}
}

// The vehicle class comes from the tag list: the other tags describe rarity or
// role, and an unknown class stays empty instead of being guessed.
func TestVehicleDisplayCatalogReadsTheVehicleClass(t *testing.T) {
	f := newFixture(t)
	f.maps()
	tanks := []struct{ id, tag, name string }{
		{"10", "heavyTank collectible", "Heavy"},
		{"11", "mediumTank", "Medium"},
		{"12", "lightTank", "Light"},
		{"13", "AT-SPG premium", "Destroyer"},
		{"14", "collectible", "Unknown"},
	}
	var list strings.Builder
	for _, tank := range tanks {
		list.WriteString(strings.Replace(completeTank(tank.name, tank.id), "<tags>lightTank regular</tags>", "<tags>"+tank.tag+"</tags>", 1))
	}
	f.list("ussr", list.String())
	f.writeDVPL("Strings/ru.yaml", "\"#vehicles:Heavy\": \"Тяжёлый\"\n\"#short:Heavy\": \"Тяж\"\n"+
		"\"#vehicles:Medium\": \"Средний\"\n\"#short:Medium\": \"Сред\"\n"+
		"\"#vehicles:Light\": \"Лёгкий\"\n\"#short:Light\": \"Лёг\"\n"+
		"\"#vehicles:Destroyer\": \"ПТ\"\n\"#short:Destroyer\": \"ПТ\"\n"+
		"\"#vehicles:Unknown\": \"Прочий\"\n\"#short:Unknown\": \"Проч\"\n")
	catalog, err := LoadVehicleDisplayCatalog(f.data)
	if err != nil {
		t.Fatal(err)
	}
	classes := map[string]VehicleClass{}
	for _, display := range catalog {
		classes[display.Name] = display.Class
	}
	want := map[string]VehicleClass{"Тяж": VehicleClassHeavy, "Сред": VehicleClassMedium, "Лёг": VehicleClassLight, "ПТ": VehicleClassDestroyer, "Проч": ""}
	for name, class := range want {
		if classes[name] != class {
			t.Fatalf("class of %q: %q, want %q", name, classes[name], class)
		}
	}
}
