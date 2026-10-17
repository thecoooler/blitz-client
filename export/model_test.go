package export

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/thecoooler/blitz-client/internal/jsonx"
	ka "github.com/thecoooler/blitz-client/internal/katest"
)

type f = ka.Field

func sceneNode(name string, geometryID uint64) ka.Value {
	return ka.Map{
		f{Key: "name", Value: ka.Text(name)},
		f{Key: "components", Value: ka.Map{
			f{Key: "transform", Value: ka.Map{f{Key: "comp.typename", Value: ka.Text("TransformComponent")}}},
			f{Key: "render", Value: ka.Map{
				f{Key: "comp.typename", Value: ka.Text("RenderComponent")},
				f{Key: "rc.renderObj", Value: ka.Map{f{Key: "ro.batches", Value: ka.Map{
					f{Key: "batch0", Value: ka.Map{
						f{Key: "rb.datasource", Value: ka.Number(geometryID)},
						f{Key: "rb.nmatname", Value: ka.Number(100)},
					}},
				}}}},
			}},
		}},
	}
}

func scene(nodes ...ka.Value) []byte {
	out := []byte("SFV2")
	out = binary.LittleEndian.AppendUint32(out, 1)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(nodes)))
	out = append(out, ka.Archive(nil)...)
	out = binary.LittleEndian.AppendUint32(out, 0)
	return append(out, ka.Archive(ka.Fields{
		{Key: "#dataNodes", Value: ka.Array{ka.Map{f{Key: "#id", Value: ka.Number(100)}}}},
		{Key: "#hierarchy", Value: ka.Array(nodes)},
	})...)
}

func testModel(scg []byte, includePBR bool) (*exportedModel, error) {
	return exportModel(scene(sceneNode("hull", 1)), scg, "fixture.sc2", "fixture.scg", "Fixture", includePBR, "png", 90, false)
}

func binaryFloats(data []byte, arrays any, name string) []float32 {
	descriptor := arrays.(object)[name].(object)
	offset, count := descriptor["byteOffset"].(int), descriptor["count"].(int)
	out := make([]float32, count)
	for index := range out {
		out[index] = math.Float32frombits(binary.LittleEndian.Uint32(data[offset+index*4:]))
	}
	return out
}

func allFiniteSlice(values []float32) bool { return allFinite(values...) }

func TestShowroomExportPreservesMissingAttributeRepairsAndUnusedGeometry(t *testing.T) {
	exported, err := testModel(ka.SCG(ka.Triangle(1, false), ka.Fields{{Key: "#id", Value: ka.Number(99)}}), true)
	if err != nil {
		t.Fatal(err)
	}
	if exported.vertexCount != 3 || exported.triangleCount != 1 || exported.generatedUVGroups != 1 {
		t.Fatalf("%+v", exported)
	}
	group := exported.metadata["groups"].([]any)[0].(object)
	if group["geometryId"] != uint64(1) || group["generatedUVs"] != true {
		t.Fatal(group)
	}
	arrays := exported.metadata["binaryGeometry"].(object)["arrays"]
	if got := binaryFloats(exported.binary, arrays, "positions"); !reflect.DeepEqual(got, []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}) {
		t.Fatal(got)
	}
	if got := binaryFloats(exported.binary, arrays, "normals"); !reflect.DeepEqual(got, []float32{0, 0, 1, 0, 0, 1, 0, 0, 1}) {
		t.Fatal(got)
	}
	for _, value := range binaryFloats(exported.binary, arrays, "vertexColors") {
		if value != 1 {
			t.Fatal("vertex colors")
		}
	}
	if !allFiniteSlice(binaryFloats(exported.binary, arrays, "uvs")) {
		t.Fatal("uvs")
	}
	pbrArrays := exported.metadata["pbrBinaryGeometry"].(object)["arrays"]
	for _, name := range []string{"tangents", "binormals"} {
		values := binaryFloats(exported.pbrBinary, pbrArrays, name)
		nonzero := false
		for _, value := range values {
			nonzero = nonzero || value != 0
		}
		if len(values) != 9 || !allFiniteSlice(values) || !nonzero {
			t.Fatal(name, values)
		}
	}
}

func armorGroup(id uint64, zones [3]float32, degenerate bool) ka.Fields {
	positions := [3][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}
	if degenerate {
		positions = [3][3]float32{}
	}
	var vertices []byte
	nan := float32(math.NaN())
	for index, position := range positions {
		vertices = append(vertices, ka.Floats(position[:]...)...)
		vertices = append(vertices, ka.Floats(nan, 0, 0)...)
		vertices = append(vertices, ka.Floats(zones[index])...)
	}
	return ka.Group(id, 1|1<<1|1<<9, 3, vertices, true, 0, 1, 2)
}

const vehicleXML = `<root>
<hull><armor><armor_3>50</armor_3></armor><turretPositions><turret>0 0 0</turret></turretPositions><hitTester><collisionModel>hull.sc2</collisionModel></hitTester></hull>
<turrets0><turret><armor><armor_3>40</armor_3></armor><gunPosition>0 0 0</gunPosition><guns><gun><armor><armor_3>20</armor_3></armor><hitTester><collisionModel>gun.sc2</collisionModel></hitTester></gun></guns></turret></turrets0>
</root>`

func testArmor(groups ...ka.Fields) (*exportedArmor, error) {
	return exportArmor(scene(sceneNode("gun", 2), sceneNode("hull", 1)), ka.SCG(groups...), []byte(vehicleXML),
		"usa", "Fixture", "fixture.sc2", "fixture.scg", "fixture.xml")
}

func jsonOf(t *testing.T, value any) string {
	t.Helper()
	data, err := jsonx.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestArmorExportPreservesNaNNormalRepairsThicknessAndDegenerateFaceSkips(t *testing.T) {
	exported, err := testArmor(armorGroup(2, [3]float32{3, 3, 3}, true), armorGroup(1, [3]float32{3, 3, 3}, false),
		ka.Fields{{Key: "#id", Value: ka.Number(99)}})
	if err != nil {
		t.Fatal(err)
	}
	if exported.faceCount != 1 || exported.vertexCount != 3 {
		t.Fatalf("%+v", exported)
	}
	for key, expected := range map[string]string{
		"positions":   "[0.0,0.0,0.0,1.0,0.0,0.0,0.0,1.0,0.0]",
		"normals":     "[0.0,0.0,1.0,0.0,0.0,1.0,0.0,0.0,1.0]",
		"thicknessMm": "[50,50,50]",
		"armorIds":    "[3,3,3]",
		"partIds":     "[1,1,1]",
	} {
		if got := jsonOf(t, exported.mesh[key]); got != expected {
			t.Fatal(key, got)
		}
	}
	parts := exported.mesh["parts"].(object)
	if parts["gun"].(object)["faceCount"] != 0 || parts["hull"].(object)["faceCount"] != 1 {
		t.Fatal(parts)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(jsonOf(t, exported.values)), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["vehicle"] != "usa.Fixture" {
		t.Fatal(decoded)
	}
}

func TestArmorPolicyKeepsRequiredStreamsAndZoneValidation(t *testing.T) {
	gun := armorGroup(2, [3]float32{3, 3, 3}, false)
	nan := float32(math.NaN())
	for _, test := range []struct {
		hull     ka.Fields
		expected string
	}{
		{ka.Triangle(1, false), "normal attribute"},
		{ka.Group(1, 3, 3, make([]byte, 72), false, 0, 1, 2), "armor-zone attribute"},
		{armorGroup(1, [3]float32{3, 4, 3}, false), "mixed armor labels"},
		{armorGroup(1, [3]float32{3.5, 3.5, 3.5}, false), "non-integer"},
		{armorGroup(1, [3]float32{nan, nan, nan}, false), "non-finite"},
		{armorGroup(1, [3]float32{4, 4, 4}, false), "no XML thickness"},
	} {
		_, err := testArmor(test.hull, gun)
		if err == nil || !strings.Contains(err.Error(), test.expected) {
			t.Fatalf("%s: %v", test.expected, err)
		}
	}
}

func TestShowroomPolicyKeepsNonfiniteRepairsAndPositionUVErrors(t *testing.T) {
	format := uint64(1 | 1<<1 | 1<<3 | 1<<7 | 1<<8)
	nan := float32(math.NaN())
	var vertices []byte
	for _, vertex := range [][5]float32{{0, 0, 0, 0, 0}, {1, 0, 0, 1, 0}, {0, 1, 0, 0, 1}} {
		vertices = append(vertices, ka.Floats(vertex[0], vertex[1], vertex[2])...)
		vertices = append(vertices, ka.Floats(nan, 0, 0)...)
		vertices = append(vertices, ka.Floats(vertex[3], vertex[4])...)
		vertices = append(vertices, ka.Floats(nan, 0, 0, nan, 0, 0)...)
	}
	exported, err := testModel(ka.SCG(ka.Group(1, format, 3, vertices, false, 0, 1, 2)), true)
	if err != nil {
		t.Fatal(err)
	}
	if exported.generatedUVGroups != 0 {
		t.Fatal("generated UVs")
	}
	arrays := exported.metadata["binaryGeometry"].(object)["arrays"]
	if got := binaryFloats(exported.binary, arrays, "normals"); !reflect.DeepEqual(got, []float32{0, 0, 1, 0, 0, 1, 0, 0, 1}) {
		t.Fatal(got)
	}
	for _, name := range []string{"tangents", "binormals"} {
		if !allFiniteSlice(binaryFloats(exported.pbrBinary, exported.metadata["pbrBinaryGeometry"].(object)["arrays"], name)) {
			t.Fatal(name)
		}
	}
	// First position and first UV in the interleaved stream.
	for _, offset := range []int{0, 24} {
		invalid := append([]byte(nil), vertices...)
		binary.LittleEndian.PutUint32(invalid[offset:], math.Float32bits(nan))
		_, err := testModel(ka.SCG(ka.Group(1, format, 3, invalid, false, 0, 1, 2)), true)
		if err == nil || !strings.Contains(err.Error(), "non-finite vertex attribute") {
			t.Fatal(offset, err)
		}
	}
}

func TestFallbackPNGsAndTexturePaths(t *testing.T) {
	path := t.TempDir() + "/fallback.png"
	if err := writeFallbackPNG(path, "baseColor"); err != nil {
		t.Fatal(err)
	}
	if err := writeFallbackPNG(path, "unknown"); err == nil {
		t.Fatal("unknown role accepted")
	}
	if got := setExtension("a/b/tex.tex", "dx11.dds.dvpl"); got != "a/b/tex.dx11.dds.dvpl" {
		t.Fatal(got)
	}
	if got := setExtension("tex", "dds"); got != "tex.dds" {
		t.Fatal(got)
	}
}
