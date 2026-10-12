package export

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/rs"
	"github.com/thecoooler/blitz-client/resources/archive"
	"github.com/thecoooler/blitz-client/resources/xml"
)

type exportedArmor struct {
	values      object
	mesh        object
	faceCount   int
	vertexCount int
}

type scenePart struct {
	part      string
	transform transform
}

var partNumericID = map[string]uint8{"gun": 0, "hull": 1, "turret": 2}

func exportArmor(sc2Data, scgData, xmlData []byte, nation, tankID, sc2Source, scgSource, xmlSource string) (*exportedArmor, error) {
	root, err := xml.Parse(xmlData)
	if err != nil {
		return nil, err
	}
	valueParts, err := armorValues(root)
	if err != nil {
		return nil, err
	}
	armorByPart, err := armorThickness(root)
	if err != nil {
		return nil, err
	}
	offsets, err := mountOffsets(root)
	if err != nil {
		return nil, err
	}
	scene, err := archive.ParseSC2Scene(sc2Data)
	if err != nil {
		return nil, err
	}
	parts, err := sceneParts(&scene, partCollisionModels(root))
	if err != nil {
		return nil, err
	}
	geometry, err := archive.ParseSCG(scgData)
	if err != nil {
		return nil, err
	}
	geometryIDs := make([]uint64, 0, len(parts))
	for id := range parts {
		geometryIDs = append(geometryIDs, id)
	}
	sort.Slice(geometryIDs, func(i, j int) bool { return geometryIDs[i] < geometryIDs[j] })
	for _, id := range geometryIDs {
		if !geometry.ContainsGroup(id) {
			return nil, fmt.Errorf("SC2 geometry ID %d has no SCG polygon group", id)
		}
	}

	var positions, normals []float32
	indices := []uint32{}
	armorIDs := []int32{}
	thicknessMM := []int32{}
	partIDs := []uint8{}
	faces := []any{}
	partsMeta := object{}

	for _, geometryID := range geometryIDs {
		part := parts[geometryID]
		group, err := geometry.Decode(geometryID)
		if err != nil {
			return nil, err
		}
		t := part.transform
		offset, ok := offsets[part.part]
		if !ok {
			return nil, fmt.Errorf("missing XML mount position for %s", part.part)
		}
		for axis, value := range offset {
			t.translation[axis] += value
		}
		localPositions := group.Positions
		localNormals := group.Normals
		if localNormals == nil {
			return nil, fmt.Errorf("SCG group %d lacks normal attribute", geometryID)
		}
		zoneValues := group.ArmorZones
		if zoneValues == nil {
			return nil, fmt.Errorf("SCG group %d lacks armor-zone attribute", geometryID)
		}
		zones := make([]int32, 0, len(zoneValues))
		for index, zoneValue := range zoneValues {
			if index >= len(localPositions) {
				break
			}
			if !allFinite(localPositions[index][:]...) || !finite32(zoneValue) {
				return nil, fmt.Errorf("non-finite SCG vertex data in group %d", geometryID)
			}
			zone := float32(math.Round(float64(zoneValue)))
			if abs32(zoneValue-zone) > 1e-5 || zone < math.MinInt32 || zone > math.MaxInt32 {
				return nil, fmt.Errorf("non-integer SCG armor label %s in group %d", rustFloat32(zoneValue), geometryID)
			}
			zones = append(zones, saturatingInt32(float64(zone)))
		}
		numericPart := partNumericID[part.part]
		firstFace := len(faces)
		firstIndex := len(indices)
		for start := 0; start+3 <= len(group.Indices); start += 3 {
			triangle := group.Indices[start : start+3]
			armorID := zones[triangle[0]]
			if zones[triangle[1]] != armorID || zones[triangle[2]] != armorID {
				return nil, fmt.Errorf("triangle has mixed armor labels in group %d", geometryID)
			}
			millimeters, ok := armorByPart[part.part][armorID]
			if !ok {
				return nil, fmt.Errorf("SCG label armor_%d has no XML thickness for %s", armorID, part.part)
			}
			firstTriangleIndex := len(indices)
			// The shipped collision meshes contain degenerate triangles whose
			// stored normals are NaN; derive the plane normal instead, or skip
			// the face when the triangle has no area to render.
			var repaired *vec3
			if !allFinite(localNormals[triangle[0]][:]...) || !allFinite(localNormals[triangle[1]][:]...) ||
				!allFinite(localNormals[triangle[2]][:]...) {
				normal, ok := triangleFaceNormal(localPositions, triangle)
				if !ok {
					continue
				}
				repaired = &normal
			}
			for _, index := range triangle {
				position, err := transformPosition(localPositions[index], &t)
				if err != nil {
					return nil, err
				}
				positions = append(positions, position[:]...)
				source := localNormals[index]
				if repaired != nil {
					source = *repaired
				}
				normal, err := transformNormal(source, &t)
				if err != nil {
					return nil, err
				}
				normals = append(normals, normal[:]...)
				indices = append(indices, uint32(len(indices)))
				armorIDs = append(armorIDs, armorID)
				thicknessMM = append(thicknessMM, millimeters)
				partIDs = append(partIDs, numericPart)
			}
			faces = append(faces, object{
				"firstIndex":  firstTriangleIndex,
				"firstVertex": firstTriangleIndex,
				"indexCount":  3,
				"part":        part.part,
				"partId":      numericPart,
				"armorId":     armorID,
				"mm":          millimeters,
			})
		}
		matrix, err := worldMatrix(&t)
		if err != nil {
			return nil, err
		}
		partsMeta[part.part] = object{
			"node":       part.transform.node,
			"geometryId": geometryID,
			"transform": object{
				"translation":            jsonx.Widen(t.translation[:]),
				"rotationQuaternionXYZW": jsonx.Widen(t.rotation[:]),
				"scale":                  jsonx.Widen(t.scale[:]),
				"matrixColumnMajor":      jsonx.Widen(matrix[:]),
			},
			"mountOffsetFromVehicleXml": jsonx.Widen(offset[:]),
			"firstFace":                 firstFace,
			"faceCount":                 len(faces) - firstFace,
			"firstIndex":                firstIndex,
			"indexCount":                len(indices) - firstIndex,
		}
	}

	values := object{
		"vehicle": nation + "." + tankID,
		"source":  xmlSource,
		"parts":   valueParts,
	}
	mesh := object{
		"formatVersion":   2,
		"tank":            nation + "-" + tankID,
		"coordinateSpace": "assembled SCG model coordinates; SC2 world transforms plus XML component mount positions applied",
		"primitive":       "triangles",
		"source": object{
			"scene":      sc2Source,
			"geometry":   scgSource,
			"vehicleXml": xmlSource,
			"provenance": "Decoded from the locally installed World of Tanks Blitz client; collision SCG zone labels joined to XML armor values and XML turret/gun mount positions; no tank-viewer site assets or code used.",
		},
		"attributeEncoding": object{
			"positions":   "flat xyz floats; triangle soup",
			"normals":     "flat xyz floats; triangle soup",
			"indices":     "sequential indices; each face uses 3 vertices",
			"partIds":     object{"0": "gun", "1": "hull", "2": "turret"},
			"armorIds":    "XML armor_N suffix carried by SCG per-vertex zone labels",
			"thicknessMm": "XML armor_N thickness repeated per vertex",
		},
		"parts":       partsMeta,
		"positions":   jsonx.Widen(positions),
		"normals":     jsonx.Widen(normals),
		"indices":     indices,
		"partIds":     uintList(partIDs),
		"armorIds":    armorIDs,
		"thicknessMm": thicknessMM,
		"faces":       faces,
	}
	return &exportedArmor{values: values, mesh: mesh, faceCount: len(faces), vertexCount: len(positions) / 3}, nil
}

// uintList keeps []uint8 encoded as a number array rather than base64.
func uintList(values []uint8) []uint32 {
	out := make([]uint32, len(values))
	for index, value := range values {
		out[index] = uint32(value)
	}
	return out
}

// rustFloat32 formats a float32 like Rust's Display.
func rustFloat32(value float32) string {
	switch {
	case value != value:
		return "NaN"
	case math.IsInf(float64(value), 1):
		return "inf"
	case math.IsInf(float64(value), -1):
		return "-inf"
	}
	return strconv.FormatFloat(float64(value), 'f', -1, 32)
}

type armorNodes struct {
	hull, turret, gun *xml.Node
}

func findArmorNodes(root *xml.Node) armorNodes {
	var nodes armorNodes
	nodes.hull = root.Child("hull")
	if turrets := root.Child("turrets0"); turrets != nil && len(turrets.Children) > 0 {
		nodes.turret = turrets.Children[0]
	}
	if nodes.turret != nil {
		if guns := nodes.turret.Child("guns"); guns != nil && len(guns.Children) > 0 {
			nodes.gun = guns.Children[0]
		}
	}
	return nodes
}

type partModel struct {
	part, model string
}

func partCollisionModels(root *xml.Node) []partModel {
	nodes := findArmorNodes(root)
	var out []partModel
	for _, candidate := range []struct {
		part string
		node *xml.Node
	}{{"hull", nodes.hull}, {"turret", nodes.turret}, {"gun", nodes.gun}} {
		if candidate.node == nil {
			continue
		}
		if model, ok := collisionModelName(candidate.node); ok {
			out = append(out, partModel{candidate.part, model})
		}
	}
	return out
}

func collisionModelName(node *xml.Node) (string, bool) {
	hitTester := node.Child("hitTester")
	if hitTester == nil {
		return "", false
	}
	model, ok := xml.ChildText(hitTester, "collisionModel")
	if !ok {
		return "", false
	}
	file := model[strings.LastIndexByte(model, '/')+1:]
	stem, _, _ := strings.Cut(file, ".")
	return asciiLower(stem), true
}

func sceneParts(scene *archive.Value, expected []partModel) (map[uint64]scenePart, error) {
	result := map[uint64]scenePart{}
	sceneNames := []string{}
	nodes, _ := scene.Get("#hierarchy").AsArray()
	for index := range nodes {
		node := &nodes[index]
		name := nodeName(node, "")
		sceneNames = append(sceneNames, name)
		partName := ""
		for _, candidate := range expected {
			if asciiFold(candidate.model, name) {
				partName = candidate.part
				break
			}
		}
		if partName == "" {
			continue
		}
		components, ok := node.Get("components").AsMap()
		if !ok {
			return nil, fmt.Errorf("SC2 collision node %s has no components", name)
		}
		var geometryIDs []uint64
		for _, component := range sortedValues(components) {
			if !isComponent(component, "RenderComponent") {
				continue
			}
			batches, _ := renderBatches(component)
			for _, batch := range batches {
				if id, ok := batch.Get("rb.datasource").AsU64(); ok && !containsID(geometryIDs, id) {
					geometryIDs = append(geometryIDs, id)
				}
			}
		}
		if len(geometryIDs) != 1 {
			parts := make([]string, len(geometryIDs))
			for index, id := range geometryIDs {
				parts[index] = strconv.FormatUint(id, 10)
			}
			return nil, fmt.Errorf("expected one SCG datasource on SC2 node %s, got BTreeSetCompat([%s])", name, strings.Join(parts, ", "))
		}
		t, err := nodeTransformOf(node)
		if err != nil {
			return nil, err
		}
		result[geometryIDs[0]] = scenePart{part: partName, transform: t}
	}
	found := map[string]bool{}
	for _, part := range result {
		found[part.part] = true
	}
	for _, required := range []string{"hull", "gun"} {
		if found[required] {
			continue
		}
		model := "?"
		for _, candidate := range expected {
			if candidate.part == required {
				model = candidate.model
				break
			}
		}
		quoted := make([]string, len(sceneNames))
		for index, name := range sceneNames {
			quoted[index] = rs.Quote(name)
		}
		return nil, fmt.Errorf("collision scene has no %s node (expected %s); scene nodes: [%s]",
			required, rs.Quote(model), strings.Join(quoted, ", "))
	}
	return result, nil
}

// asciiFold compares like eq_ignore_ascii_case, which unlike strings.EqualFold
// does not fold non-ASCII letters.
func asciiFold(a, b string) bool {
	return len(a) == len(b) && asciiLower(a) == asciiLower(b)
}

func containsID(ids []uint64, id uint64) bool {
	for _, existing := range ids {
		if existing == id {
			return true
		}
	}
	return false
}

func armorValues(root *xml.Node) (object, error) {
	nodes := findArmorNodes(root)
	hull, err := parseArmorPart(nodes.hull, "Корпус")
	if err != nil {
		return nil, err
	}
	turret, err := parseArmorPart(nodes.turret, "Башня")
	if err != nil {
		return nil, err
	}
	gun, err := parseArmorPart(nodes.gun, "Орудие")
	if err != nil {
		return nil, err
	}
	return object{"hull": hull, "turret": turret, "gun": gun}, nil
}

const f32Epsilon float32 = 1.1920929e-07

func parseRustF64(text string) (float64, bool) { return parseRustFloat(text, 64) }

// parseRustFloat parses like Rust's f32/f64 from_str: decimal or inf/nan
// spellings, overflow to infinity, and no hexadecimal or underscores.
func parseRustFloat(text string, bits int) (float64, bool) {
	if strings.ContainsAny(text, "_xX") {
		return 0, false
	}
	value, err := strconv.ParseFloat(text, bits)
	if err != nil {
		if numErr, ok := err.(*strconv.NumError); ok && numErr.Err == strconv.ErrRange {
			return value, true
		}
		return 0, false
	}
	return value, true
}

// saturatingInt32 converts like Rust's `as i32`.
func saturatingInt32(value float64) int32 {
	switch {
	case value != value:
		return 0
	case value >= math.MaxInt32:
		return math.MaxInt32
	case value <= math.MinInt32:
		return math.MinInt32
	}
	return int32(value)
}

func parseArmorPart(node *xml.Node, label string) (object, error) {
	if node == nil {
		return nil, fmt.Errorf("vehicle XML is missing armor part %s", label)
	}
	armor := node.Child("armor")
	if armor == nil {
		return nil, fmt.Errorf("vehicle XML is missing armor values for %s", label)
	}
	values := []any{}
	for _, entry := range armor.Children {
		text := strings.TrimSpace(entry.Text)
		if text == "" {
			continue
		}
		value, ok := parseRustF64(text)
		if !ok {
			continue
		}
		values = append(values, object{"id": entry.Name, "mm": saturatingInt32(value)})
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("vehicle XML has no armor values for %s", label)
	}
	primary, _ := xml.ChildText(node, "primaryArmor")
	return object{"label": label, "primaryArmor": strings.Fields(primary), "values": values}, nil
}

func armorThickness(root *xml.Node) (map[string]map[int32]int32, error) {
	nodes := findArmorNodes(root)
	result := map[string]map[int32]int32{}
	for _, part := range []struct {
		label string
		node  *xml.Node
	}{{"hull", nodes.hull}, {"turret", nodes.turret}, {"gun", nodes.gun}} {
		var armor *xml.Node
		if part.node != nil {
			armor = part.node.Child("armor")
		}
		values, err := parseThickness(armor, part.label)
		if err != nil {
			return nil, err
		}
		result[part.label] = values
	}
	return result, nil
}

func parseThickness(node *xml.Node, label string) (map[int32]int32, error) {
	if node == nil {
		return nil, fmt.Errorf("XML armor block missing: %s", label)
	}
	values := map[int32]int32{}
	for _, entry := range node.Children {
		number, ok := strings.CutPrefix(entry.Name, "armor_")
		if !ok {
			continue
		}
		id, err := rs.ParseInt(number, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid armor label %s", entry.Name)
		}
		text := strings.TrimSpace(entry.Text)
		if text == "" {
			return nil, fmt.Errorf("missing numeric thickness for %s/%s", label, entry.Name)
		}
		thickness, ok := parseRustF64(text)
		if !ok {
			return nil, fmt.Errorf("invalid armor thickness for %s/%s", label, entry.Name)
		}
		values[int32(id)] = saturatingInt32(thickness)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("empty armor block: %s", label)
	}
	return values, nil
}

func mountOffsets(root *xml.Node) (map[string]vec3, error) {
	hull := root.Child("hull")
	if hull == nil {
		return nil, fmt.Errorf("vehicle XML has no hull")
	}
	positions := hull.Child("turretPositions")
	if positions == nil {
		return nil, fmt.Errorf("XML mount position missing: hull/turretPositions")
	}
	turretText, ok := xml.ChildText(positions, "turret")
	turret, err := clientPosition(turretText, ok, "hull/turretPositions/turret")
	if err != nil {
		return nil, err
	}
	turretNode := findArmorNodes(root).turret
	if turretNode == nil {
		return nil, fmt.Errorf("vehicle turret definition not found in XML")
	}
	gunText, ok := xml.ChildText(turretNode, "gunPosition")
	gunLocal, err := clientPosition(gunText, ok, "turret/gunPosition")
	if err != nil {
		return nil, err
	}
	return map[string]vec3{
		"hull":   {0, 0, 0},
		"turret": turret,
		"gun":    add3(turret, gunLocal),
	}, nil
}

func clientPosition(value string, present bool, label string) (vec3, error) {
	if !present {
		return vec3{}, fmt.Errorf("XML mount position missing: %s", label)
	}
	invalid := fmt.Errorf("invalid XML mount position for %s: %s", label, rs.Quote(value))
	var numbers []float32
	for _, item := range strings.Fields(value) {
		number, ok := parseRustFloat(item, 32)
		if !ok {
			return vec3{}, invalid
		}
		numbers = append(numbers, float32(number))
	}
	if len(numbers) != 3 || !allFinite(numbers...) {
		return vec3{}, invalid
	}
	return vec3{numbers[0], numbers[2], numbers[1]}, nil
}

func triangleFaceNormal(positions []vec3, triangle []uint32) (vec3, bool) {
	start, second, third := positions[triangle[0]], positions[triangle[1]], positions[triangle[2]]
	normal := cross3(sub3(second, start), sub3(third, start))
	length := length3(normal)
	if length <= f32Epsilon {
		return vec3{}, false
	}
	return vec3{normal[0] / length, normal[1] / length, normal[2] / length}, true
}
