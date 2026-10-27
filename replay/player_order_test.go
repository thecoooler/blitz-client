package replay

import (
	"reflect"
	"slices"
	"testing"

	"github.com/thecoooler/blitz-client/client"
)

func TestBattlePlayerOrderMatchesScreenshot(t *testing.T) {
	teams := [][]ArenaPlayer{
		{{VehicleLevel: 10, VehicleName: "60TP Lewandowskiego"}, {VehicleLevel: 10, VehicleName: "BZ-75"}, {VehicleLevel: 10, VehicleName: "Foch 155"}, {VehicleLevel: 10, VehicleName: "Grille 15"}, {VehicleLevel: 10, VehicleName: "Leopard 1"}, {VehicleLevel: 10, VehicleName: "MBT-B"}, {VehicleLevel: 10, VehicleName: "Т-100 ЛТ"}, {VehicleLevel: 9, VehicleName: "B-C 25 t Roland"}, {VehicleLevel: 9, VehicleName: "T30"}, {VehicleLevel: 9, VehicleName: "ИС-3-II"}},
		{{VehicleLevel: 10, VehicleName: "Carro 45t"}, {VehicleLevel: 10, VehicleName: "Chieftain Mk. 6"}, {VehicleLevel: 10, VehicleName: "FV215b 183"}, {VehicleLevel: 10, VehicleName: "Grille 15"}, {VehicleLevel: 10, VehicleName: "MBT-B"}, {VehicleLevel: 10, VehicleName: "Sheridan Ракетный"}, {VehicleLevel: 10, VehicleName: "WZ-113"}, {VehicleLevel: 9, VehicleName: "AMX 30 1er prot."}, {VehicleLevel: 9, VehicleName: "Tortoise"}, {VehicleLevel: 9, VehicleName: "ЛВ-915 Кратос"}},
	}
	for _, want := range teams {
		shuffled := slices.Clone(want)
		slices.Reverse(shuffled)
		source := slices.Clone(shuffled)
		if got := OrderBattlePlayers(shuffled); !reflect.DeepEqual(got, want) {
			t.Fatalf("screenshot order: got %+v want %+v", got, want)
		}
		if !reflect.DeepEqual(shuffled, source) {
			t.Fatal("sort mutated source snapshot")
		}
	}
}

func TestBattlePlayerOrderDestroyedAndMissingMetadata(t *testing.T) {
	team := []ArenaPlayer{
		{Nickname: "Dead IX", VehicleLevel: 9, VehicleName: "T30", Destroyed: true},
		{Nickname: "Tie first", VehicleLevel: 10, VehicleName: "Grille 15"},
		{Nickname: "Unknown"},
		{Nickname: "Live IX", VehicleLevel: 9, VehicleName: "B-C 25 t Roland"},
		{Nickname: "Dead X", VehicleLevel: 10, VehicleName: "BZ-75", Destroyed: true},
		{Nickname: "Tie second", VehicleLevel: 10, VehicleName: "Grille 15"},
	}
	var names []string
	for _, p := range OrderBattlePlayers(team) {
		names = append(names, p.Nickname)
	}
	want := []string{"Tie first", "Tie second", "Live IX", "Unknown", "Dead X", "Dead IX"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("order: %v", names)
	}
}

func TestArenaPlayerVehicleDescriptorAndCatalogProjection(t *testing.T) {
	p, err := decodeArenaPlayer([]byte{0x12, 2, 0x31, 0x2e})
	if err != nil || p.VehicleTypeID != 11825 {
		t.Fatalf("vehicle descriptor: %+v %v", p, err)
	}
	for _, body := range [][]byte{{0x12, 0}, {0x12, 1, 0x31}, {0x12, 2, 0x32, 0x2e}} {
		p, err := decodeArenaPlayer(body)
		if err != nil || p.VehicleTypeID != 0 {
			t.Fatalf("unrecognized descriptor: %+v %v", p, err)
		}
	}
	watcher := NewLiveReplayWatcher(t.TempDir())
	watcher.vehicleCatalog = map[uint32]client.VehicleDisplay{11825: {Name: "BZ-75", Level: 10}}
	roster := &BattleRoster{Allies: []ArenaPlayer{{VehicleTypeID: 11825}}, Enemies: []ArenaPlayer{{VehicleTypeID: 999}}}
	watcher.describePlayers(roster)
	if roster.Allies[0].VehicleLevel != 10 || roster.Allies[0].VehicleName != "BZ-75" || roster.Enemies[0].VehicleLevel != 0 {
		t.Fatalf("projection: %+v", roster)
	}
	if err := watcher.LoadVehicleCatalog(t.TempDir()); err == nil {
		t.Fatal("invalid catalog unexpectedly loaded")
	}
	watcher.describePlayers(roster)
	if roster.Allies[0].VehicleName != "BZ-75" {
		t.Fatal("failed reload discarded usable metadata")
	}
}
