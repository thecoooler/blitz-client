package archive

import (
	"encoding/binary"
	"math"
	"reflect"
	"strings"
	"testing"

	ka "github.com/thecoooler/blitz-client/internal/katest"
)

func mustParse(t *testing.T, data []byte) *Geometry {
	t.Helper()
	geometry, err := ParseSCG(data)
	if err != nil {
		t.Fatal(err)
	}
	return geometry
}

func TestDecodesBothIndexWidthsAndOptionalStreams(t *testing.T) {
	for _, wide := range []bool{false, true} {
		geometry := mustParse(t, ka.SCG(ka.Triangle(7, wide)))
		if !geometry.ContainsGroup(7) || geometry.ContainsGroup(8) {
			t.Fatal("group identity")
		}
		decoded, err := geometry.Decode(7)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(decoded.Positions, [][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}) ||
			!reflect.DeepEqual(decoded.Indices, []uint32{0, 1, 2}) {
			t.Fatalf("%+v", decoded)
		}
		if decoded.Normals != nil || decoded.Colors != nil || decoded.UVs != nil ||
			decoded.Tangents != nil || decoded.Binormals != nil || decoded.ArmorZones != nil {
			t.Fatal("unexpected optional stream")
		}
		if _, err := geometry.Decode(8); err == nil || !strings.Contains(err.Error(), "missing SCG geometry group 8") {
			t.Fatal(err)
		}
	}
}

func TestDecodesPackedAttributesWithoutApplyingRepairPolicy(t *testing.T) {
	format := uint64(1 | 1<<1 | 1<<2 | 1<<3 | 1<<4 | 1<<7 | 1<<8 | 1<<9)
	vertices := ka.Floats(1, 2, 3, float32(math.NaN()), 0, 1)
	vertices = append(vertices, 0, 64, 128, 255)
	vertices = append(vertices, ka.Floats(0.25, 0.75, 7, 8, 1, 0, 0, 0, 1, 0, 3)...)
	decoded, err := mustParse(t, ka.SCG(ka.Group(9, format, 1, vertices, false, 0, 0, 0))).Decode(9)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Positions[0] != [3]float32{1, 2, 3} || !math.IsNaN(float64(decoded.Normals[0][0])) ||
		decoded.Colors[0] != [4]float32{0, 64. / 255, 128. / 255, 1} || decoded.UVs[0] != [2]float32{0.25, 0.75} ||
		decoded.Tangents[0] != [3]float32{1, 0, 0} || decoded.Binormals[0] != [3]float32{0, 1, 0} ||
		!reflect.DeepEqual(decoded.ArmorZones, []float32{3}) {
		t.Fatalf("%+v", decoded)
	}
}

func TestValidatesGroupIdentityGloballyButGeometryOnlyWhenSelected(t *testing.T) {
	geometry := mustParse(t, ka.SCG(ka.Triangle(1, false), ka.Fields{{Key: "#id", Value: ka.Number(99)}}))
	if decoded, err := geometry.Decode(1); err != nil || len(decoded.Indices) != 3 {
		t.Fatal(err)
	}
	if _, err := geometry.Decode(99); err == nil || !strings.Contains(err.Error(), "vertexFormat") {
		t.Fatal(err)
	}
	if _, err := ParseSCG(ka.SCG(ka.Triangle(1, false), ka.Triangle(1, true))); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatal(err)
	}
	if _, err := ParseSCG(ka.SCG(ka.Triangle(1, false).Without("#id"))); err == nil || !strings.Contains(err.Error(), "no #id") {
		t.Fatal(err)
	}
	for _, id := range []ka.Value{ka.Bytes(binary.LittleEndian.AppendUint64(nil, 7)), ka.Signed(7)} {
		if !mustParse(t, ka.SCG(ka.Triangle(7, false).Replace("#id", id))).ContainsGroup(7) {
			t.Fatal("id")
		}
	}
	if _, err := ParseSCG(ka.SCG(ka.Triangle(1, false).Replace("#id", ka.Signed(-1)))); err == nil {
		t.Fatal("negative id accepted")
	}
}

func TestRejectsInvalidSelectedLayoutsBuffersAndIndicesWithoutPanics(t *testing.T) {
	cases := []struct {
		key      string
		value    ka.Value
		expected string
	}{
		{"vertexFormat", ka.Number(0), "lacks positions"},
		{"vertexFormat", ka.Number(1 | 1<<20), "unsupported vertex format"},
		{"vertexFormat", ka.Number(1 | 1<<40), "unsupported vertex format"},
		{"vertexCount", ka.Number(math.MaxUint64), "size"},
		{"vertices", ka.Bytes(make([]byte, 35)), "vertex buffer size mismatch"},
		{"rhi_primitiveType", ka.Number(2), "not a triangle list"},
		{"indexFormat", ka.Number(2), "unknown index format"},
		{"indexCount", ka.Number(math.MaxUint64), "size"},
		{"indices", ka.Bytes(make([]byte, 5)), "index buffer size mismatch"},
		{"indices", ka.Bytes{0, 0, 1, 0, 3, 0}, "index out of range"},
	}
	for _, test := range cases {
		_, err := mustParse(t, ka.SCG(ka.Triangle(42, false).Replace(test.key, test.value))).Decode(42)
		if err == nil || !strings.Contains(err.Error(), "group 42") || !strings.Contains(err.Error(), test.expected) {
			t.Fatalf("%s: %v", test.key, err)
		}
	}
	_, err := mustParse(t, ka.SCG(ka.Group(42, 1, 3, make([]byte, 36), true, 0, 1, 2, 0))).Decode(42)
	if err == nil || !strings.Contains(err.Error(), "divisible by 3") {
		t.Fatal(err)
	}
}

func TestPreservesEmptyGroupsAndRejectsInvalidArchiveFraming(t *testing.T) {
	decoded, err := mustParse(t, ka.SCG(ka.Group(1, 1, 0, nil, false))).Decode(1)
	if err != nil || len(decoded.Positions) != 0 || len(decoded.Indices) != 0 {
		t.Fatal(err)
	}
	if mustParse(t, ka.SCG()).ContainsGroup(1) {
		t.Fatal("empty file has groups")
	}
	valid := ka.SCG(ka.Triangle(1, false))
	for _, length := range []int{0, 4, 15, len(valid) - 1} {
		if _, err := ParseSCG(valid[:length]); err == nil {
			t.Fatalf("prefix %d parsed", length)
		}
	}
	for _, mutate := range []func([]byte) []byte{
		func(b []byte) []byte { binary.LittleEndian.PutUint32(b[4:], 2); return b },
		func(b []byte) []byte { binary.LittleEndian.PutUint32(b[12:], 2); return b },
		func(b []byte) []byte { return append(b, 0) },
	} {
		if _, err := ParseSCG(mutate(append([]byte(nil), valid...))); err == nil {
			t.Fatal("invalid framing parsed")
		}
	}
}

func TestPreservesFullWidthIndices(t *testing.T) {
	decoded, err := mustParse(t, ka.SCG(ka.Group(1, 1, 65_537, make([]byte, 65_537*12), true, 0, 65_536, 1))).Decode(1)
	if err != nil || !reflect.DeepEqual(decoded.Indices, []uint32{0, 65_536, 1}) {
		t.Fatal(err)
	}
}
