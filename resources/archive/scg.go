package archive

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
)

var attributeBytes = [20]int{12, 12, 4, 8, 8, 8, 8, 12, 12, 4, 16, 12, 4, 8, 16, 16, 12, 12, 12, 12}

const knownVertexFormat = uint64(1)<<len(attributeBytes) - 1

// Geometry owns group identity; structural decoding remains lazy so unused
// geometry does not impose requirements on showroom or collision exports.
type Geometry struct {
	groups map[uint64]Value
}

// DecodedGroup holds geometry whose binary layout and index invariants have
// been checked. Optional attributes are nil when absent and floats retain NaNs
// for caller-owned repairs.
type DecodedGroup struct {
	Positions  [][3]float32
	Normals    [][3]float32
	Colors     [][4]float32
	UVs        [][2]float32
	Tangents   [][3]float32
	Binormals  [][3]float32
	ArmorZones []float32
	Indices    []uint32
}

func geometryFromGroups(groups []Value) (*Geometry, error) {
	indexed := make(map[uint64]Value, len(groups))
	for _, group := range groups {
		id, err := groupID(group)
		if err != nil {
			return nil, err
		}
		if _, exists := indexed[id]; exists {
			return nil, fmt.Errorf("duplicate SCG polygon group id %d", id)
		}
		indexed[id] = group
	}
	return &Geometry{groups: indexed}, nil
}

// ContainsGroup reports whether a polygon group with the id exists.
func (g *Geometry) ContainsGroup(id uint64) bool {
	_, ok := g.groups[id]
	return ok
}

// Decode validates and decodes one polygon group.
func (g *Geometry) Decode(id uint64) (*DecodedGroup, error) {
	group, ok := g.groups[id]
	if !ok {
		return nil, fmt.Errorf("missing SCG geometry group %d", id)
	}
	decoded, err := decodeGroup(&group)
	if err != nil {
		return nil, fmt.Errorf("SCG group %d: %w", id, err)
	}
	return decoded, nil
}

func groupID(group Value) (uint64, error) {
	value := group.Get("#id")
	if value == nil {
		return 0, fmt.Errorf("SCG polygon group has no #id")
	}
	if value.Kind == Bytes && len(value.Bytes) == 8 {
		return binary.LittleEndian.Uint64(value.Bytes), nil
	}
	if id, ok := value.AsU64(); ok {
		return id, nil
	}
	return 0, fmt.Errorf("unexpected SCG polygon group id: %s", value.Debug())
}

func decodeGroup(group *Value) (*DecodedGroup, error) {
	format, err := number(group, "vertexFormat")
	if err != nil {
		return nil, err
	}
	if format&^knownVertexFormat != 0 {
		return nil, fmt.Errorf("unsupported vertex format bits: 0x%x", format)
	}
	if format&1 == 0 {
		return nil, fmt.Errorf("lacks positions")
	}
	var offsets [len(attributeBytes)]int
	stride := 0
	for bit, size := range attributeBytes {
		offsets[bit] = -1
		if format&(1<<bit) != 0 {
			offsets[bit] = stride
			stride += size
		}
	}
	vertexCount, err := count(group, "vertexCount")
	if err != nil {
		return nil, err
	}
	vertices, err := byteField(group, "vertices")
	if err != nil {
		return nil, err
	}
	if hi, expected := bits.Mul64(uint64(stride), vertexCount); hi != 0 || expected > math.MaxInt {
		return nil, fmt.Errorf("vertex buffer size overflow")
	} else if uint64(len(vertices)) != expected {
		return nil, fmt.Errorf("vertex buffer size mismatch")
	}
	primitive, err := number(group, "rhi_primitiveType")
	if err != nil {
		return nil, err
	}
	if primitive != 1 {
		return nil, fmt.Errorf("not a triangle list")
	}

	indexCount, err := count(group, "indexCount")
	if err != nil {
		return nil, err
	}
	indexBytes, err := byteField(group, "indices")
	if err != nil {
		return nil, err
	}
	indexFormat, err := number(group, "indexFormat")
	if err != nil {
		return nil, err
	}
	var width uint64
	switch indexFormat {
	case 0:
		width = 2
	case 1:
		width = 4
	default:
		return nil, fmt.Errorf("unknown index format %d", indexFormat)
	}
	if hi, expected := bits.Mul64(indexCount, width); hi != 0 || expected > math.MaxInt {
		return nil, fmt.Errorf("index buffer size overflow")
	} else if uint64(len(indexBytes)) != expected {
		return nil, fmt.Errorf("index buffer size mismatch")
	}
	if indexCount%3 != 0 {
		return nil, fmt.Errorf("triangle index count is not divisible by 3")
	}
	indices := make([]uint32, indexCount)
	for index := range indices {
		if width == 2 {
			indices[index] = uint32(binary.LittleEndian.Uint16(indexBytes[index*2:]))
		} else {
			indices[index] = binary.LittleEndian.Uint32(indexBytes[index*4:])
		}
		if uint64(indices[index]) >= vertexCount {
			return nil, fmt.Errorf("index out of range")
		}
	}

	n := int(vertexCount)
	decoded := &DecodedGroup{
		Positions: floats3(vertices, stride, offsets[0], n),
		Normals:   floats3(vertices, stride, offsets[1], n),
		Tangents:  floats3(vertices, stride, offsets[7], n),
		Binormals: floats3(vertices, stride, offsets[8], n),
		Indices:   indices,
	}
	if offset := offsets[2]; offset >= 0 {
		decoded.Colors = make([][4]float32, n)
		for vertex := range n {
			for axis := range 4 {
				decoded.Colors[vertex][axis] = float32(vertices[vertex*stride+offset+axis]) / 255
			}
		}
	}
	if offset := offsets[3]; offset >= 0 {
		decoded.UVs = make([][2]float32, n)
		for vertex := range n {
			for axis := range 2 {
				decoded.UVs[vertex][axis] = readFloat(vertices, vertex*stride+offset+axis*4)
			}
		}
	}
	if offset := offsets[9]; offset >= 0 {
		decoded.ArmorZones = make([]float32, n)
		for vertex := range n {
			decoded.ArmorZones[vertex] = readFloat(vertices, vertex*stride+offset)
		}
	}
	return decoded, nil
}

func readFloat(data []byte, offset int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(data[offset:]))
}

func floats3(vertices []byte, stride, offset, n int) [][3]float32 {
	if offset < 0 {
		return nil
	}
	values := make([][3]float32, n)
	for vertex := range n {
		for axis := range 3 {
			values[vertex][axis] = readFloat(vertices, vertex*stride+offset+axis*4)
		}
	}
	return values
}

func number(group *Value, field string) (uint64, error) {
	if value, ok := group.Get(field).AsU64(); ok {
		return value, nil
	}
	return 0, fmt.Errorf("missing or invalid field %s", field)
}

func count(group *Value, field string) (uint64, error) {
	value, err := number(group, field)
	if err != nil {
		return 0, err
	}
	if value > math.MaxInt {
		return 0, fmt.Errorf("%s exceeds addressable size", field)
	}
	return value, nil
}

func byteField(group *Value, field string) ([]byte, error) {
	if value, ok := group.Get(field).AsBytes(); ok {
		return value, nil
	}
	return nil, fmt.Errorf("missing or invalid byte field %s", field)
}
