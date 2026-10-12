// Package katest encodes real keyed-archive bytes for tests, so assertions
// cross the archive and export interfaces instead of constructing decoded state.
package katest

import (
	"encoding/binary"
	"math"
)

// Value is a keyed-archive value to encode.
type Value interface{ encode(out []byte) []byte }

type (
	Number uint64
	Signed int64
	Bytes  []byte
	Text   string
	Map    Fields
	Array  []Value
)

// Field is one ordered archive entry.
type Field struct {
	Key   string
	Value Value
}

// Fields is an ordered list of archive entries.
type Fields []Field

func (v Number) encode(out []byte) []byte {
	return binary.LittleEndian.AppendUint64(append(out, 10), uint64(v))
}

func (v Signed) encode(out []byte) []byte {
	return binary.LittleEndian.AppendUint64(append(out, 9), uint64(v))
}

func (v Bytes) encode(out []byte) []byte {
	out = binary.LittleEndian.AppendUint32(append(out, 6), uint32(len(v)))
	return append(out, v...)
}

func (v Text) encode(out []byte) []byte {
	out = binary.LittleEndian.AppendUint32(append(out, 4), uint32(len(v)))
	return append(out, v...)
}

func (v Map) encode(out []byte) []byte {
	nested := Archive(Fields(v))
	out = binary.LittleEndian.AppendUint32(append(out, 8), uint32(len(nested)))
	return append(out, nested...)
}

func (v Array) encode(out []byte) []byte {
	out = binary.LittleEndian.AppendUint32(append(out, 27), uint32(len(v)))
	for _, item := range v {
		out = item.encode(out)
	}
	return out
}

// Archive encodes a version-1 keyed archive.
func Archive(fields Fields) []byte {
	out := []byte("KA")
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(fields)))
	for _, field := range fields {
		out = Text(field.Key).encode(out)
		out = field.Value.encode(out)
	}
	return out
}

// SCG encodes an SCG file with the given groups.
func SCG(groups ...Fields) []byte {
	out := []byte("SCPG")
	out = binary.LittleEndian.AppendUint32(out, 1)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(groups)))
	out = binary.LittleEndian.AppendUint32(out, uint32(len(groups)))
	for _, group := range groups {
		out = append(out, Archive(group)...)
	}
	return out
}

// Floats encodes little-endian float32 values.
func Floats(values ...float32) []byte {
	out := make([]byte, 0, len(values)*4)
	for _, value := range values {
		out = binary.LittleEndian.AppendUint32(out, math.Float32bits(value))
	}
	return out
}

// Group returns the fields of a triangle-list polygon group.
func Group(id, format, vertexCount uint64, vertices []byte, wide bool, indices ...uint32) Fields {
	var indexBytes []byte
	for _, index := range indices {
		if wide {
			indexBytes = binary.LittleEndian.AppendUint32(indexBytes, index)
		} else {
			indexBytes = binary.LittleEndian.AppendUint16(indexBytes, uint16(index))
		}
	}
	indexFormat := uint64(0)
	if wide {
		indexFormat = 1
	}
	return Fields{
		{"#id", Number(id)},
		{"vertexFormat", Number(format)},
		{"vertexCount", Number(vertexCount)},
		{"vertices", Bytes(vertices)},
		{"rhi_primitiveType", Number(1)},
		{"indexFormat", Number(indexFormat)},
		{"indexCount", Number(len(indices))},
		{"indices", Bytes(indexBytes)},
	}
}

// Triangle returns a position-only group with one triangle.
func Triangle(id uint64, wide bool) Fields {
	return Group(id, 1, 3, Floats(0, 0, 0, 1, 0, 0, 0, 1, 0), wide, 0, 1, 2)
}

// Replace sets the value of an existing field and returns the fields.
func (f Fields) Replace(key string, value Value) Fields {
	out := append(Fields(nil), f...)
	for index := range out {
		if out[index].Key == key {
			out[index].Value = value
			return out
		}
	}
	panic("missing field " + key)
}

// Without removes a field.
func (f Fields) Without(key string) Fields {
	var out Fields
	for _, field := range f {
		if field.Key != key {
			out = append(out, field)
		}
	}
	return out
}
