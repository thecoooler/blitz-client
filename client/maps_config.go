package client

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	blitz "github.com/thecoooler/blitz-client"
)

// mapConfigContainerMagic starts the serialized client cache container that
// holds one named binary payload.
var mapConfigContainerMagic = []byte("KA")

// mapsConfigPayload names the protobuf payload of maps_config_*.dat.
const mapsConfigPayload = "protobufConfig"

// MapsConfig is the decoded maps_config_*.dat server cache. The MapsConfig
// message itself carries only a progression threshold and the map list.
type MapsConfig struct {
	// ProgressionThreshold is the serialized MapsConfig.progressionThreshold.
	ProgressionThreshold int64
	// Maps is MapsConfig.mapList in source order.
	Maps []MapInfo
}

// MapInfo is one MapInfo record of MapsConfig.mapList.
type MapInfo struct {
	// Name is MapInfo.name, the maps.yaml map key.
	Name string
	// Levels is MapInfo.levels, the vehicle levels the map is enabled for.
	Levels []int64
	// NewbieVehicleLevel and NewbieBattleCount are the corresponding MapInfo
	// fields; nil when the client omitted them.
	NewbieVehicleLevel *int64
	NewbieBattleCount  *int64
	// GameModes are the serialized battle mode IDs (see GameModeName) in the
	// order the server wrote them.
	GameModes []int64
}

// Map returns the record for a maps.yaml map name.
func (c *MapsConfig) Map(name string) (MapInfo, bool) {
	for _, entry := range c.Maps {
		if entry.Name == name {
			return entry, true
		}
	}
	return MapInfo{}, false
}

// ReadMapsConfig reads the client's maps_config_*.dat cache. A directory reads
// its newest maps_config_*.dat, so callers can point at the client cache folder.
// Maps absent from the configuration are absent from the result; they are not
// errors.
func ReadMapsConfig(path string) (*MapsConfig, error) {
	resolved, err := resolveMapsConfig(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, blitz.IOError("could not read maps configuration", resolved, err)
	}
	invalid := func(err error) (*MapsConfig, error) {
		return nil, blitz.ErrorAt("CLIENT_CONFIG_INVALID", err.Error(), resolved)
	}
	payload, err := mapConfigPayload(data)
	if err != nil {
		return invalid(err)
	}
	config, err := decodeMapsConfig(payload)
	if err != nil {
		return invalid(err)
	}
	return config, nil
}

// resolveMapsConfig accepts a cache file or a directory holding one.
func resolveMapsConfig(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", blitz.IOError("could not read maps configuration", path, err)
	}
	if !info.IsDir() {
		return path, nil
	}
	matches, err := filepath.Glob(filepath.Join(path, "maps_config_*.dat"))
	if err != nil {
		return "", blitz.IOError("could not read maps configuration", path, err)
	}
	var newest string
	var newestTime int64
	for _, match := range matches {
		stat, err := os.Stat(match)
		if err != nil || !stat.Mode().IsRegular() {
			continue
		}
		if stamp := stat.ModTime().UnixNano(); stamp > newestTime {
			newest, newestTime = match, stamp
		}
	}
	if newest == "" {
		return "", blitz.ErrorAt("CLIENT_CONFIG_INVALID", "no maps_config_*.dat file in directory", path)
	}
	return newest, nil
}

// mapConfigPayload returns the protobufConfig payload of a KA container.
func mapConfigPayload(data []byte) ([]byte, error) {
	if len(data) < 8 || string(data[:2]) != string(mapConfigContainerMagic) {
		return nil, fmt.Errorf("maps configuration is not a KA container")
	}
	if version := binary.LittleEndian.Uint16(data[2:4]); version != 1 {
		return nil, fmt.Errorf("unsupported KA container version %d", version)
	}
	count := binary.LittleEndian.Uint32(data[4:8])
	if count > 1024 {
		return nil, fmt.Errorf("KA container has %d entries", count)
	}
	offset := 8
	for entry := uint32(0); entry < count; entry++ {
		name, next, err := containerString(data, offset)
		if err != nil {
			return nil, err
		}
		offset = next
		payload, afterType, err := containerBlob(data, offset)
		if err != nil {
			return nil, err
		}
		offset = afterType
		if name == mapsConfigPayload {
			return payload, nil
		}
	}
	return nil, fmt.Errorf("KA container has no %s payload", mapsConfigPayload)
}

// containerString reads a uint32 length plus name after the entry kind byte.
func containerString(data []byte, offset int) (string, int, error) {
	if offset >= len(data) {
		return "", 0, fmt.Errorf("truncated KA entry")
	}
	offset++ // entry kind
	if offset+4 > len(data) {
		return "", 0, fmt.Errorf("truncated KA entry length")
	}
	size := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4
	if size < 0 || size > len(data)-offset {
		return "", 0, fmt.Errorf("KA entry name exceeds the container")
	}
	return string(data[offset : offset+size]), offset + size, nil
}

// containerBlob reads a payload kind byte, then a uint32 length and the payload.
func containerBlob(data []byte, offset int) ([]byte, int, error) {
	if offset >= len(data) {
		return nil, 0, fmt.Errorf("truncated KA payload")
	}
	offset++ // payload kind
	if offset+4 > len(data) {
		return nil, 0, fmt.Errorf("truncated KA payload length")
	}
	size := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4
	if size < 0 || size > len(data)-offset {
		return nil, 0, fmt.Errorf("KA payload exceeds the container")
	}
	return data[offset : offset+size], offset + size, nil
}

// decodeMapsConfig parses the MapsConfig protobuf message. Unknown protobuf
// fields are skipped so a newer client stays readable.
func decodeMapsConfig(data []byte) (*MapsConfig, error) {
	config := &MapsConfig{}
	seen := map[string]bool{}
	err := eachProtoField(data, func(number int, wire int, value uint64, raw []byte) error {
		switch number {
		case 1:
			if wire != 0 {
				return fmt.Errorf("progressionThreshold has wire type %d", wire)
			}
			config.ProgressionThreshold = int64(value)
		case 2:
			if wire != 2 {
				return fmt.Errorf("mapList has wire type %d", wire)
			}
			info, err := decodeMapInfo(raw)
			if err != nil {
				return err
			}
			if info.Name == "" {
				return fmt.Errorf("maps configuration entry has no name")
			}
			if seen[info.Name] {
				return fmt.Errorf("maps configuration repeats map %q", info.Name)
			}
			seen[info.Name] = true
			config.Maps = append(config.Maps, info)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return config, nil
}

func decodeMapInfo(data []byte) (MapInfo, error) {
	info := MapInfo{}
	err := eachProtoField(data, func(number int, wire int, value uint64, raw []byte) error {
		switch number {
		case 1:
			if wire != 2 {
				return fmt.Errorf("map name has wire type %d", wire)
			}
			info.Name = string(raw)
		case 2:
			switch wire {
			case 0:
				info.Levels = append(info.Levels, int64(value))
			case 2:
				levels, err := decodePackedIntegers(raw)
				if err != nil {
					return err
				}
				info.Levels = append(info.Levels, levels...)
			default:
				return fmt.Errorf("map levels have wire type %d", wire)
			}
		case 3:
			if wire != 0 {
				return fmt.Errorf("newbieVehicleLevel has wire type %d", wire)
			}
			level := int64(value)
			info.NewbieVehicleLevel = &level
		case 4:
			if wire != 0 {
				return fmt.Errorf("newbieBattleCount has wire type %d", wire)
			}
			count := int64(value)
			info.NewbieBattleCount = &count
		case 5:
			switch wire {
			case 0:
				info.GameModes = append(info.GameModes, int64(value))
			case 2:
				modes, err := decodePackedIntegers(raw)
				if err != nil {
					return err
				}
				info.GameModes = append(info.GameModes, modes...)
			default:
				return fmt.Errorf("map game modes have wire type %d", wire)
			}
		}
		return nil
	})
	return info, err
}

// eachProtoField walks one protobuf message, passing varints and
// length-delimited payloads. Fixed-width fields are skipped.
func eachProtoField(data []byte, visit func(number int, wire int, value uint64, raw []byte) error) error {
	offset := 0
	for offset < len(data) {
		key, next, err := protoVarint(data, offset)
		if err != nil {
			return err
		}
		offset = next
		number, wire := int(key>>3), int(key&7)
		if number == 0 {
			return fmt.Errorf("protobuf field number 0")
		}
		switch wire {
		case 0:
			value, after, err := protoVarint(data, offset)
			if err != nil {
				return err
			}
			offset = after
			if err := visit(number, wire, value, nil); err != nil {
				return err
			}
		case 1:
			if offset+8 > len(data) {
				return fmt.Errorf("truncated protobuf fixed64")
			}
			offset += 8
		case 2:
			size, after, err := protoVarint(data, offset)
			if err != nil {
				return err
			}
			if size > uint64(len(data)-after) {
				return fmt.Errorf("protobuf length exceeds the message")
			}
			raw := data[after : after+int(size)]
			offset = after + int(size)
			if err := visit(number, wire, 0, raw); err != nil {
				return err
			}
		case 5:
			if offset+4 > len(data) {
				return fmt.Errorf("truncated protobuf fixed32")
			}
			offset += 4
		default:
			return fmt.Errorf("unsupported protobuf wire type %d", wire)
		}
	}
	return nil
}

func decodePackedIntegers(data []byte) ([]int64, error) {
	values := []int64{}
	offset := 0
	for offset < len(data) {
		value, next, err := protoVarint(data, offset)
		if err != nil {
			return nil, err
		}
		offset = next
		values = append(values, int64(value))
	}
	return values, nil
}

func protoVarint(data []byte, offset int) (uint64, int, error) {
	var value uint64
	for shift := uint(0); shift < 64; shift += 7 {
		if offset >= len(data) {
			return 0, 0, fmt.Errorf("truncated protobuf varint")
		}
		byteValue := data[offset]
		offset++
		value |= uint64(byteValue&0x7f) << shift
		if byteValue&0x80 == 0 {
			return value, offset, nil
		}
	}
	return 0, 0, fmt.Errorf("protobuf varint overflows 64 bits")
}
