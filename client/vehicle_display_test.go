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
	for id, want := range map[uint32]VehicleDisplay{7297: {Name: "60TP", Level: 10}, 11825: {Name: "BZ-75", Level: 3}, 2593: {Name: "T30", Level: 3}} {
		if got := catalog[id]; got != want {
			t.Fatalf("descriptor %d: got %+v want %+v", id, got, want)
		}
	}
}
