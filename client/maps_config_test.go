package client

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// buildMapsConfig serializes the KA container and MapsConfig protobuf used by
// the client cache.
func buildMapsConfig(t *testing.T, progression int64, maps []MapInfo) []byte {
	t.Helper()
	info := []byte{}
	for _, entry := range maps {
		packed := []byte{}
		for _, level := range entry.Levels {
			packed = appendConfigVarint(packed, uint64(level))
		}
		message := appendConfigBytes(nil, 1, []byte(entry.Name))
		if entry.Levels != nil {
			message = appendConfigBytes(message, 2, packed)
		}
		if entry.NewbieVehicleLevel != nil {
			message = appendConfigVarint(message, 3<<3)
			message = appendConfigVarint(message, uint64(*entry.NewbieVehicleLevel))
		}
		if entry.NewbieBattleCount != nil {
			message = appendConfigVarint(message, 4<<3)
			message = appendConfigVarint(message, uint64(*entry.NewbieBattleCount))
		}
		if entry.GameModes != nil {
			modes := []byte{}
			for _, mode := range entry.GameModes {
				modes = appendConfigVarint(modes, uint64(mode))
			}
			message = appendConfigBytes(message, 5, modes)
		}
		info = appendConfigBytes(info, 2, message)
	}
	config := appendConfigVarint(nil, 1<<3)
	config = appendConfigVarint(config, uint64(progression))
	config = append(config, info...)
	return wrapConfigContainer(t, config)
}

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

func wrapConfigContainer(t *testing.T, payload []byte) []byte {
	t.Helper()
	out := []byte("KA")
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint32(out, 1)
	out = append(out, 4)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(mapsConfigPayload)))
	out = append(out, []byte(mapsConfigPayload)...)
	out = append(out, 4)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(payload)))
	return append(out, payload...)
}

func int64Ptr(value int64) *int64 { return &value }

func TestReadMapsConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "maps_config_Russia_1L.dat")
	data := buildMapsConfig(t, 1000, []MapInfo{
		{Name: "glacier_02", Levels: []int64{5, 6, 7}, NewbieVehicleLevel: int64Ptr(5), NewbieBattleCount: int64Ptr(60), GameModes: []int64{25, 4, 5}},
		{Name: "glacier_br", GameModes: []int64{48}},
	})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := ReadMapsConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.ProgressionThreshold != 1000 || len(config.Maps) != 2 {
		t.Fatalf("config: %+v", config)
	}
	tournament, ok := config.Map("glacier_02")
	if !ok {
		t.Fatal("glacier_02 missing")
	}
	if !reflect.DeepEqual(tournament.Levels, []int64{5, 6, 7}) ||
		tournament.NewbieVehicleLevel == nil || *tournament.NewbieVehicleLevel != 5 ||
		tournament.NewbieBattleCount == nil || *tournament.NewbieBattleCount != 60 ||
		!reflect.DeepEqual(tournament.GameModes, []int64{25, 4, 5}) {
		t.Fatalf("glacier_02: %+v", tournament)
	}
	br, ok := config.Map("glacier_br")
	if !ok || br.Levels != nil || br.NewbieVehicleLevel != nil || br.NewbieBattleCount != nil ||
		!reflect.DeepEqual(br.GameModes, []int64{48}) {
		t.Fatalf("glacier_br: %+v", br)
	}
	if name, ok := GameModeName(48); !ok || name != "battle-royale" {
		t.Fatalf("mode 48: %q %v", name, ok)
	}
	if name, ok := GameModeName(4); !ok || name != "tournament" {
		t.Fatalf("mode 4: %q %v", name, ok)
	}
}

func TestReadMapsConfigDirectoryPicksNewest(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "maps_config_Russia_1L.dat")
	newer := filepath.Join(dir, "maps_config_Russia_2L.dat")
	if err := os.WriteFile(older, buildMapsConfig(t, 1, []MapInfo{{Name: "old"}}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, buildMapsConfig(t, 2, []MapInfo{{Name: "new"}}), 0o600); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(older, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, stamp.Add(time.Minute), stamp.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	config, err := ReadMapsConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if config.ProgressionThreshold != 2 || len(config.Maps) != 1 || config.Maps[0].Name != "new" {
		t.Fatalf("expected the newest cache, got %+v", config)
	}
}

func TestReadMapsConfigRejectsInvalid(t *testing.T) {
	for name, data := range map[string][]byte{
		"not a container": []byte("nope"),
		"truncated entry": append([]byte("KA\x01\x00\x01\x00\x00\x00\x04\xff"), 0, 0, 0, 0),
		"no payload":      []byte("KA\x01\x00\x00\x00\x00\x00"),
	} {
		path := filepath.Join(t.TempDir(), "maps_config.dat")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadMapsConfig(path); err == nil {
			t.Fatalf("%s: accepted invalid cache", name)
		}
	}
}
