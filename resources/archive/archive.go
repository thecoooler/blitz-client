package archive

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"unicode/utf16"

	"github.com/thecoooler/blitz-client/internal/lossy"
)

const (
	maxDepth      = 100
	maxValueCount = 16_000_000
)

// ParseKeyedArchive decodes a standalone keyed archive, such as the client's
// device settings, without interpreting its entries.
func ParseKeyedArchive(data []byte) (Value, error) {
	r := &reader{data: data}
	value, err := r.archive(nil, 0)
	if err != nil {
		return Value{}, err
	}
	if r.remaining() != 0 {
		return Value{}, fmt.Errorf("unparsed keyed archive bytes remain: %d", r.remaining())
	}
	return value, nil
}

// ParseSC2Scene decodes the root keyed archive of an SC2 scene.
func ParseSC2Scene(data []byte) (Value, error) {
	r := &reader{data: data}
	magic, err := r.take(4)
	if err != nil {
		return Value{}, err
	}
	if string(magic) != "SFV2" {
		return Value{}, fmt.Errorf("SC2 input does not begin with SFV2")
	}
	if _, err := r.u32(); err != nil { // version
		return Value{}, err
	}
	if _, err := r.u32(); err != nil { // node count
		return Value{}, err
	}
	if _, err := r.archive(nil, 0); err != nil { // version tags
		return Value{}, err
	}
	descriptorLength, err := r.u32()
	if err != nil {
		return Value{}, err
	}
	if _, err := r.take(int(descriptorLength)); err != nil {
		return Value{}, err
	}
	root, err := r.archive(nil, 0)
	if err != nil {
		return Value{}, err
	}
	if r.remaining() != 0 {
		return Value{}, fmt.Errorf("unparsed SC2 bytes remain: %d", r.remaining())
	}
	return root, nil
}

// ParseSCG decodes an SCG polygon-group file. Group identities are validated
// immediately; geometry is decoded on demand.
func ParseSCG(data []byte) (*Geometry, error) {
	r := &reader{data: data}
	magic, err := r.take(4)
	if err != nil {
		return nil, err
	}
	if string(magic) != "SCPG" {
		return nil, fmt.Errorf("SCG input does not begin with SCPG")
	}
	version, err := r.u32()
	if err != nil {
		return nil, err
	}
	count, err := r.u32()
	if err != nil {
		return nil, err
	}
	countCopy, err := r.u32()
	if err != nil {
		return nil, err
	}
	if version != 1 || count != countCopy || count > maxValueCount {
		return nil, fmt.Errorf("unsupported SCG header: version=%d, counts=%d/%d", version, count, countCopy)
	}
	groups := make([]Value, 0, count)
	for range count {
		group, err := r.archive(nil, 0)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	if r.remaining() != 0 {
		return nil, fmt.Errorf("unparsed SCG bytes remain: %d", r.remaining())
	}
	return geometryFromGroups(groups)
}

type reader struct {
	data     []byte
	position int
}

func (r *reader) remaining() int {
	if r.position > len(r.data) {
		return 0
	}
	return len(r.data) - r.position
}

func (r *reader) take(length int) ([]byte, error) {
	end := r.position + length
	if length < 0 || end < r.position {
		return nil, fmt.Errorf("binary read offset overflow")
	}
	if end > len(r.data) {
		return nil, fmt.Errorf("read outside file: offset=%d, size=%d, length=%d", r.position, length, len(r.data))
	}
	start := r.position
	r.position = end
	return r.data[start:end], nil
}

func (r *reader) u8() (byte, error) {
	b, err := r.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *reader) u16() (uint16, error) {
	b, err := r.take(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

func (r *reader) u32() (uint32, error) {
	b, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (r *reader) u64() (uint64, error) {
	b, err := r.take(8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func (r *reader) f32() (float32, error) {
	value, err := r.u32()
	return math.Float32frombits(value), err
}

func (r *reader) string() (string, error) {
	length, err := r.u32()
	if err != nil {
		return "", err
	}
	if uint64(length) > uint64(r.remaining()) {
		return "", fmt.Errorf("string length %d exceeds remaining archive bytes", length)
	}
	b, err := r.take(int(length))
	if err != nil {
		return "", err
	}
	return lossy.String(b), nil
}

func (r *reader) count() (int, error) {
	count, err := r.u32()
	if err != nil {
		return 0, err
	}
	if count > maxValueCount {
		return 0, fmt.Errorf("keyed archive value count %d exceeds safety limit", count)
	}
	return int(count), nil
}

func (r *reader) archive(parentNames map[uint32]string, depth int) (Value, error) {
	if depth > maxDepth {
		return Value{}, fmt.Errorf("SC2 keyed archives exceed the nesting limit")
	}
	signature, err := r.take(2)
	if err != nil {
		return Value{}, err
	}
	if string(signature) != "KA" {
		return Value{}, fmt.Errorf("invalid keyed archive signature at byte %d", r.position-2)
	}
	version, err := r.u16()
	if err != nil {
		return Value{}, err
	}
	switch version {
	case 1:
		count, err := r.count()
		if err != nil {
			return Value{}, err
		}
		entries := make(map[string]Value)
		for range count {
			keyType, err := r.u8()
			if err != nil {
				return Value{}, err
			}
			key, err := r.value(keyType, parentNames, depth+1, false)
			if err != nil {
				return Value{}, err
			}
			valueType, err := r.u8()
			if err != nil {
				return Value{}, err
			}
			value, err := r.value(valueType, parentNames, depth+1, false)
			if err != nil {
				return Value{}, err
			}
			entries[valueKey(key)] = value
		}
		return Value{Kind: Map, Map: entries}, nil
	case 2:
		fastCount, err := r.count()
		if err != nil {
			return Value{}, err
		}
		names := make([]string, 0, fastCount)
		for range fastCount {
			length, err := r.u16()
			if err != nil {
				return Value{}, err
			}
			b, err := r.take(int(length))
			if err != nil {
				return Value{}, err
			}
			names = append(names, lossy.String(b))
		}
		namesByID := make(map[uint32]string, fastCount)
		ids := make([]uint32, 0, fastCount)
		for range fastCount {
			id, err := r.u32()
			if err != nil {
				return Value{}, err
			}
			ids = append(ids, id)
		}
		for index, id := range ids {
			namesByID[id] = names[index]
		}
		return r.fastEntries(namesByID, depth, false, "unknown SC2 fast-name key id %d")
	case 0x0102:
		if parentNames == nil {
			return Value{}, fmt.Errorf("nested SC2 archive is missing its fast-name table")
		}
		return r.fastEntries(parentNames, depth, true, "unknown nested SC2 key id %d")
	case 0xff02:
		return Value{Kind: Map, Map: map[string]Value{}}, nil
	default:
		return Value{}, fmt.Errorf("unsupported keyed archive version 0x%04x", version)
	}
}

func (r *reader) fastEntries(names map[uint32]string, depth int, fastString bool, unknown string) (Value, error) {
	count, err := r.count()
	if err != nil {
		return Value{}, err
	}
	entries := make(map[string]Value)
	for range count {
		keyID, err := r.u32()
		if err != nil {
			return Value{}, err
		}
		key, ok := names[keyID]
		if !ok {
			return Value{}, fmt.Errorf(unknown, keyID)
		}
		valueType, err := r.u8()
		if err != nil {
			return Value{}, err
		}
		value, err := r.value(valueType, names, depth+1, fastString)
		if err != nil {
			return Value{}, err
		}
		entries[key] = value
	}
	return Value{Kind: Map, Map: entries}, nil
}

func (r *reader) value(typeID byte, names map[uint32]string, depth int, fastString bool) (Value, error) {
	if depth > maxDepth {
		return Value{}, fmt.Errorf("SC2 keyed archives exceed the nesting limit")
	}
	switch typeID {
	case 0:
		return Value{Kind: Null}, nil
	case 1:
		b, err := r.u8()
		return Value{Kind: Bool, Bool: b != 0}, err
	case 2:
		v, err := r.u32()
		return Value{Kind: Signed, Signed: int64(int32(v))}, err
	case 3:
		v, err := r.f32()
		return Value{Kind: Float, Float: float64(v)}, err
	case 4, 20:
		if typeID == 4 && fastString {
			s, err := r.fastName(names)
			return Value{Kind: String, String: s}, err
		}
		s, err := r.string()
		return Value{Kind: String, String: s}, err
	case 5:
		count, err := r.u32()
		if err != nil {
			return Value{}, err
		}
		b, err := r.take(int(uint64(count) * 2))
		if err != nil {
			return Value{}, err
		}
		units := make([]uint16, count)
		for index := range units {
			units[index] = binary.LittleEndian.Uint16(b[index*2:])
		}
		return Value{Kind: String, String: string(utf16.Decode(units))}, nil
	case 6:
		length, err := r.u32()
		if err != nil {
			return Value{}, err
		}
		b, err := r.take(int(length))
		return Value{Kind: Bytes, Bytes: append([]byte(nil), b...)}, err
	case 7:
		v, err := r.u32()
		return Value{Kind: Unsigned, Unsigned: uint64(v)}, err
	case 8:
		expected, err := r.u32()
		if err != nil {
			return Value{}, err
		}
		start := r.position
		nested, err := r.archive(names, depth+1)
		if err != nil {
			return Value{}, err
		}
		if consumed := r.position - start; consumed != int(expected) {
			return Value{}, fmt.Errorf("nested archive length mismatch: %d != %d", consumed, expected)
		}
		return nested, nil
	case 9:
		v, err := r.u64()
		return Value{Kind: Signed, Signed: int64(v)}, err
	case 10:
		v, err := r.u64()
		return Value{Kind: Unsigned, Unsigned: v}, err
	case 11:
		return r.floatArray(2)
	case 12:
		return r.floatArray(3)
	case 13, 14:
		return r.floatArray(4)
	case 15:
		return r.floatArray(9)
	case 16:
		return r.floatArray(16)
	case 17:
		b, err := r.take(4)
		return Value{Kind: Bytes, Bytes: append([]byte(nil), b...)}, err
	case 18:
		s, err := r.fastName(names)
		return Value{Kind: String, String: s}, err
	case 19:
		return r.floatArray(6)
	case 21:
		v, err := r.u64()
		return Value{Kind: Float, Float: math.Float64frombits(v)}, err
	case 22:
		b, err := r.u8()
		return Value{Kind: Signed, Signed: int64(int8(b))}, err
	case 23:
		b, err := r.u8()
		return Value{Kind: Unsigned, Unsigned: uint64(b)}, err
	case 24:
		v, err := r.u16()
		return Value{Kind: Signed, Signed: int64(int16(v))}, err
	case 25:
		v, err := r.u16()
		return Value{Kind: Unsigned, Unsigned: uint64(v)}, err
	case 27:
		count, err := r.count()
		if err != nil {
			return Value{}, err
		}
		values := make([]Value, 0, count)
		for range count {
			itemType, err := r.u8()
			if err != nil {
				return Value{}, err
			}
			item, err := r.value(itemType, names, depth+1, fastString)
			if err != nil {
				return Value{}, err
			}
			values = append(values, item)
		}
		return Value{Kind: Array, Array: values}, nil
	case 29:
		position, err := r.floatArray(3)
		if err != nil {
			return Value{}, err
		}
		scale, err := r.floatArray(3)
		if err != nil {
			return Value{}, err
		}
		rotation, err := r.floatArray(4)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: Map, Map: map[string]Value{
			"position": position, "scale": scale, "rotation": rotation,
		}}, nil
	default:
		return Value{}, fmt.Errorf("unsupported SC2 keyed archive value type %d", typeID)
	}
}

func (r *reader) fastName(names map[uint32]string) (string, error) {
	index, err := r.u32()
	if err != nil {
		return "", err
	}
	name, ok := names[index]
	if !ok {
		return "", fmt.Errorf("unknown SC2 fast-name id %d", index)
	}
	return name, nil
}

func (r *reader) floatArray(count int) (Value, error) {
	values := make([]Value, 0, count)
	for range count {
		v, err := r.f32()
		if err != nil {
			return Value{}, err
		}
		values = append(values, Value{Kind: Float, Float: float64(v)})
	}
	return Value{Kind: Array, Array: values}, nil
}

func valueKey(value Value) string {
	switch value.Kind {
	case String:
		return value.String
	case Signed:
		return strconv.FormatInt(value.Signed, 10)
	case Unsigned:
		return strconv.FormatUint(value.Unsigned, 10)
	case Bytes:
		return fmt.Sprintf("%x", value.Bytes)
	case Bool:
		return strconv.FormatBool(value.Bool)
	default:
		return value.Debug()
	}
}
