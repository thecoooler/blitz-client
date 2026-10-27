package client

import "testing"

func TestMapDisplayCatalogRequiresUnambiguousLocalizedNames(t *testing.T) {
	f := newFixture(t)
	f.write("maps.yaml", "maps:\n    arena:\n        id: 20\n        localName: NAME\n        frame: 4\n    missing:\n        id: 21\n        localName: MISSING\n        frame: 1\n    duplicate1:\n        id: 22\n        localName: A\n        frame: 1\n    duplicate2:\n        id: 22\n        localName: B\n        frame: 1\n")
	f.writeDVPL("Strings/ru.yaml", "\"#maps:arena:NAME\": \"Карта\"\n\"#maps:duplicate1:A\": \"Другая\"\n")
	maps, err := LoadMapDisplayCatalog(f.data)
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 1 || maps[20] != "Карта" {
		t.Fatalf("catalog: %+v", maps)
	}
}
