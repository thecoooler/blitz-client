package export

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// mapsConfigFile serializes a KA container holding a minimal MapsConfig with
// every MapInfo field used by the exporter.
func mapsConfigFile(t *testing.T, progression int64, entries map[string]mapInfoFixture) string {
	t.Helper()
	payload := []byte{}
	for name, entry := range entries {
		levels := []byte{}
		for _, level := range entry.levels {
			levels = appendConfigVarint(levels, uint64(level))
		}
		modes := []byte{}
		for _, mode := range entry.modes {
			modes = appendConfigVarint(modes, uint64(mode))
		}
		info := appendConfigBytes(nil, 1, []byte(name))
		if entry.levels != nil {
			info = appendConfigBytes(info, 2, levels)
		}
		if entry.newbieVehicleLevel != nil {
			info = appendConfigVarint(info, 3<<3)
			info = appendConfigVarint(info, uint64(*entry.newbieVehicleLevel))
		}
		if entry.newbieBattleCount != nil {
			info = appendConfigVarint(info, 4<<3)
			info = appendConfigVarint(info, uint64(*entry.newbieBattleCount))
		}
		if entry.modes != nil {
			info = appendConfigBytes(info, 5, modes)
		}
		payload = appendConfigBytes(payload, 2, info)
	}
	config := appendConfigVarint(nil, 1<<3)
	config = appendConfigVarint(config, uint64(progression))
	config = append(config, payload...)

	container := []byte("KA")
	container = binary.LittleEndian.AppendUint16(container, 1)
	container = binary.LittleEndian.AppendUint32(container, 1)
	container = append(container, 4)
	container = binary.LittleEndian.AppendUint32(container, uint32(len("protobufConfig")))
	container = append(container, []byte("protobufConfig")...)
	container = append(container, 4)
	container = binary.LittleEndian.AppendUint32(container, uint32(len(config)))
	container = append(container, config...)

	path := filepath.Join(t.TempDir(), "maps_config_Russia_1L.dat")
	if err := os.WriteFile(path, container, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

type mapInfoFixture struct {
	levels             []int64
	newbieVehicleLevel *int64
	newbieBattleCount  *int64
	modes              []int64
}

func int64Ptr(value int64) *int64 { return &value }

func appendConfigVarint(buffer []byte, value uint64) []byte {
	for value >= 0x80 {
		buffer = append(buffer, byte(value)|0x80)
		value >>= 7
	}
	return append(buffer, byte(value))
}

func appendConfigBytes(buffer []byte, field int, payload []byte) []byte {
	buffer = appendConfigVarint(buffer, uint64(field)<<3|2)
	buffer = appendConfigVarint(buffer, uint64(len(payload)))
	return append(buffer, payload...)
}

func TestExportMapsProjectsMapInfoFromConfig(t *testing.T) {
	f := mapsFixture(t)
	config := mapsConfigFile(t, 1000, map[string]mapInfoFixture{
		"karelia": {levels: []int64{5, 6, 7}, newbieVehicleLevel: int64Ptr(5), newbieBattleCount: int64Ptr(60), modes: []int64{1, 4, 5}},
		"skit":    {modes: []int64{48}},
	})
	snapshot := &Snapshot{}
	report, err := exportMapsWithConfig(f.path("game/Data"), f.path("output"), config, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !report.WithMapConfig || report.ProgressionThreshold == nil || *report.ProgressionThreshold != 1000 {
		t.Fatalf("unexpected report %+v", report)
	}
	entries := mapIndex(t, filepath.Join(f.path("output"), "maps.json"))
	if !reflect.DeepEqual(entries[0].GameModes, []string{"regular", "tournament", "clan"}) ||
		!reflect.DeepEqual(entries[0].Levels, []int64{5, 6, 7}) ||
		entries[0].NewbieVehicleLevel == nil || *entries[0].NewbieVehicleLevel != 5 ||
		entries[0].NewbieBattleCount == nil || *entries[0].NewbieBattleCount != 60 {
		t.Fatalf("karelia MapInfo: %+v", entries[0])
	}
	if !reflect.DeepEqual(entries[1].GameModes, []string{"battle-royale"}) ||
		entries[1].Levels != nil || entries[1].NewbieVehicleLevel != nil || entries[1].NewbieBattleCount != nil {
		t.Fatalf("skit MapInfo: %+v", entries[1])
	}
	// "lost" is absent from the configuration: null plus one issue.
	if entries[2].GameModes != nil || entries[2].Levels != nil {
		t.Fatalf("lost MapInfo: %+v", entries[2])
	}
	found := false
	for _, issue := range snapshot.Issues {
		if issue.Section == "maps" && issue.Message == "map lost is absent from the maps configuration; its MapInfo fields are null" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an absent-map issue, got %v", snapshot.Issues)
	}
}

func TestExportMapsWithoutConfigLeavesMapInfoNull(t *testing.T) {
	f := mapsFixture(t)
	snapshot := &Snapshot{}
	report, err := exportMaps(f.path("game/Data"), f.path("output"), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if report.WithMapConfig || report.ProgressionThreshold != nil {
		t.Fatalf("report should not claim a map config: %+v", report)
	}
	for _, entry := range mapIndex(t, report.File) {
		if entry.GameModes != nil || entry.Levels != nil || entry.NewbieVehicleLevel != nil || entry.NewbieBattleCount != nil {
			t.Fatalf("%s has MapInfo without a config: %+v", entry.Name, entry)
		}
	}
}
