package export

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/rs"
	"github.com/thecoooler/blitz-client/resources/archive"
)

// object is a JSON object; encoding/json sorts its keys like serde_json's
// default BTreeMap-backed Map.
type object = map[string]any

type transform struct {
	translation vec3
	rotation    [4]float32
	scale       vec3
	node        string
}

type materialInfo struct {
	maps           map[string]string
	alphaTest      bool
	alphaThreshold float32
}

type binding struct {
	geometryID uint64
	materialID uint64
	node       string
	transform  transform
	material   materialInfo
}

type exportedModel struct {
	metadata          object
	binary            []byte
	pbrBinary         []byte
	textures          []textureSource
	vertexCount       int
	triangleCount     int
	materialCount     int
	fallbackMaps      int
	generatedUVGroups int
}

type textureSource struct {
	file     string
	source   string // empty for generated fallback textures
	role     string
	lossless bool
}

// textureSet is a sorted role→path list ordered like Rust's BTreeMap.
type textureSet [][2]string

func newTextureSet(maps map[string]string) textureSet {
	set := make(textureSet, 0, len(maps))
	for key, value := range maps {
		set = append(set, [2]string{key, value})
	}
	slices.SortFunc(set, func(a, b [2]string) int { return strings.Compare(a[0], b[0]) })
	return set
}

func compareTextureSets(a, b textureSet) int {
	for index := 0; index < len(a) && index < len(b); index++ {
		if c := strings.Compare(a[index][0], b[index][0]); c != 0 {
			return c
		}
		if c := strings.Compare(a[index][1], b[index][1]); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(a), len(b))
}

func (s textureSet) key() string {
	var builder strings.Builder
	for _, pair := range s {
		builder.WriteString(strconv.Quote(pair[0]))
		builder.WriteByte('=')
		builder.WriteString(strconv.Quote(pair[1]))
		builder.WriteByte(';')
	}
	return builder.String()
}

func baseColorOnly(maps map[string]string) map[string]string {
	path, ok := maps["baseColor"]
	if !ok {
		path = "@fallback/baseColor"
	}
	return map[string]string{"baseColor": path}
}

func exportModel(sc2Data, scgData []byte, sc2Source, scgSource, tankID string, includePBR bool,
	textureFormat string, webpQuality uint8, compress bool) (*exportedModel, error) {
	if textureFormat != "webp" && textureFormat != "png" {
		return nil, fmt.Errorf("texture format must be png or webp")
	}
	scene, err := archive.ParseSC2Scene(sc2Data)
	if err != nil {
		return nil, err
	}
	materials, err := materialNodes(&scene)
	if err != nil {
		return nil, err
	}
	bindings, err := visibleBatches(&scene, materials, includePBR)
	if err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return nil, fmt.Errorf("SC2 scene contains no visible render batches")
	}
	geometry, err := archive.ParseSCG(scgData)
	if err != nil {
		return nil, err
	}
	var missing []uint64
	for _, b := range bindings {
		if !geometry.ContainsGroup(b.geometryID) && !slices.Contains(missing, b.geometryID) {
			missing = append(missing, b.geometryID)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		parts := make([]string, len(missing))
		for index, id := range missing {
			parts[index] = strconv.FormatUint(id, 10)
		}
		return nil, fmt.Errorf("SC2 batches reference missing SCG geometry groups: [%s]", strings.Join(parts, ", "))
	}

	selectedMaps := make([]textureSet, len(bindings))
	for index, b := range bindings {
		if includePBR {
			selectedMaps[index] = newTextureSet(b.material.maps)
		} else {
			selectedMaps[index] = newTextureSet(baseColorOnly(b.material.maps))
		}
	}
	var textureSets []textureSet
	seenSets := map[string]bool{}
	for _, set := range selectedMaps {
		if !seenSets[set.key()] {
			seenSets[set.key()] = true
			textureSets = append(textureSets, set)
		}
	}
	slices.SortFunc(textureSets, compareTextureSets)
	textureIDs := map[string]uint32{}
	for index, set := range textureSets {
		textureIDs[set.key()] = uint32(index)
	}

	var positions, normals, tangents, binormals, uvs, vertexColors []float32
	var vertexTextureIDs, indices []uint32
	groupRanges := []any{}
	generatedUVGroups := 0

	ordered := make([]int, len(bindings))
	for index := range ordered {
		ordered[index] = index
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		left, right := bindings[ordered[i]], bindings[ordered[j]]
		if left.geometryID != right.geometryID {
			return left.geometryID < right.geometryID
		}
		return left.node < right.node
	})

	for _, bindingIndex := range ordered {
		b := bindings[bindingIndex]
		group, err := geometry.Decode(b.geometryID)
		if err != nil {
			return nil, err
		}
		vertexCount := len(group.Positions)
		hasUVs := group.UVs != nil
		localPositions := group.Positions
		localNormals := group.Normals
		if localNormals == nil {
			localNormals = make([]vec3, vertexCount)
		}
		localUVs := group.UVs
		if localUVs == nil {
			localUVs = make([]vec2, vertexCount)
		}
		for _, position := range localPositions {
			if !allFinite(position[:]...) {
				return nil, fmt.Errorf("non-finite vertex attribute in SCG group %d", b.geometryID)
			}
		}
		for _, uv := range localUVs {
			if !allFinite(uv[:]...) {
				return nil, fmt.Errorf("non-finite vertex attribute in SCG group %d", b.geometryID)
			}
		}
		localColors := group.Colors
		if localColors == nil {
			localColors = make([][4]float32, vertexCount)
			for index := range localColors {
				localColors[index] = [4]float32{1, 1, 1, 1}
			}
		}
		var localTangents, localBinormals []vec3
		if includePBR {
			localTangents, localBinormals = group.Tangents, group.Binormals
			if localTangents == nil {
				localTangents = make([]vec3, vertexCount)
			}
			if localBinormals == nil {
				localBinormals = make([]vec3, vertexCount)
			}
		}
		localIndices := group.Indices
		if !hasUVs {
			localUVs = planarUVs(localPositions)
			generatedUVGroups++
		}
		rebuilt := fallbackNormals(localPositions, localIndices)
		for index, normal := range localNormals {
			length := length3(normal)
			if length <= 1e-15 || !finite32(length) {
				localNormals[index] = rebuilt[index]
			} else {
				localNormals[index] = scale3(normal, 1/length)
			}
		}
		var fallbackTangents, fallbackBinormals []vec3
		if includePBR {
			fallbackTangents, fallbackBinormals = fallbackTangentBasis(localPositions, localNormals, localUVs, localIndices)
		}

		vertexBase := uint32(len(positions) / 3)
		firstIndex := len(indices)
		textureID, ok := textureIDs[selectedMaps[bindingIndex].key()]
		if !ok {
			return nil, fmt.Errorf("internal texture set lookup failed")
		}
		for vertex := 0; vertex < vertexCount; vertex++ {
			position, err := transformPosition(localPositions[vertex], &b.transform)
			if err != nil {
				return nil, err
			}
			positions = append(positions, position[:]...)
			normal, err := transformNormal(localNormals[vertex], &b.transform)
			if err != nil {
				return nil, err
			}
			normals = append(normals, normal[:]...)
			if includePBR {
				tangent := localTangents[vertex]
				if length := length3(tangent); !finite32(length) || length <= 1e-15 {
					tangent = fallbackTangents[vertex]
				}
				binormal := localBinormals[vertex]
				if length := length3(binormal); !finite32(length) || length <= 1e-15 {
					binormal = fallbackBinormals[vertex]
				}
				direction, err := transformDirection(tangent, &b.transform)
				if err != nil {
					return nil, err
				}
				tangents = append(tangents, direction[:]...)
				if direction, err = transformDirection(binormal, &b.transform); err != nil {
					return nil, err
				}
				binormals = append(binormals, direction[:]...)
			}
			uvs = append(uvs, localUVs[vertex][:]...)
			vertexColors = append(vertexColors, localColors[vertex][:]...)
			vertexTextureIDs = append(vertexTextureIDs, textureID)
		}
		for _, index := range localIndices {
			indices = append(indices, vertexBase+index)
		}
		groupRanges = append(groupRanges, object{
			"geometryId":   b.geometryID,
			"node":         b.node,
			"materialId":   b.materialID,
			"textureId":    textureID,
			"firstIndex":   firstIndex,
			"indexCount":   len(indices) - firstIndex,
			"generatedUVs": !hasUVs,
		})
	}

	materialsJSON := buildMaterialJSON(textureSets, bindings, selectedMaps, textureFormat, includePBR)
	var textures []textureSource
	fallbackMaps := 0
	for index, set := range textureSets {
		maps := materialsJSON[index]["maps"].(object)
		for _, pair := range set {
			role := pair[0]
			entry := maps[role].(object)
			fallback := entry["fallback"].(bool)
			if fallback {
				fallbackMaps++
			}
			source := ""
			if path, ok := entry["materialPath"].(string); ok {
				source = setExtension(strings.ReplaceAll(path, `\`, "/"), "dx11.dds.dvpl")
			}
			textures = append(textures, textureSource{
				file:     entry["file"].(string),
				source:   source,
				role:     role,
				lossless: includePBR && role != "baseColor" && textureFormat == "webp",
			})
		}
	}

	var maxTextureID, maxIndex uint32
	for _, value := range vertexTextureIDs {
		maxTextureID = max(maxTextureID, value)
	}
	for _, value := range indices {
		maxIndex = max(maxIndex, value)
	}
	textureType, textureWidth := "uint32", 4
	if maxTextureID <= math.MaxUint8 {
		textureType, textureWidth = "uint8", 1
	} else if maxTextureID <= math.MaxUint16 {
		textureType, textureWidth = "uint16", 2
	}
	indexType, indexWidth := "uint32", 4
	if maxIndex <= math.MaxUint16 {
		indexType, indexWidth = "uint16", 2
	}
	var buffer []byte
	arrays := object{}
	buffer = pushF32Array(buffer, arrays, "positions", positions)
	buffer = pushF32Array(buffer, arrays, "normals", normals)
	buffer = pushF32Array(buffer, arrays, "uvs", uvs)
	buffer = pushF32Array(buffer, arrays, "vertexColors", vertexColors)
	buffer, textureOffset := alignBuffer(buffer, textureWidth)
	buffer = appendUints(buffer, vertexTextureIDs, textureWidth)
	arrays["textureIds"] = arrayDescriptor(textureType, textureOffset, len(vertexTextureIDs))
	buffer, indexOffset := alignBuffer(buffer, indexWidth)
	buffer = appendUints(buffer, indices, indexWidth)
	arrays["indices"] = arrayDescriptor(indexType, indexOffset, len(indices))

	var pbrBinary []byte
	pbrDescriptors := object{}
	if includePBR {
		pbrBinary = pushF32Array([]byte{}, pbrDescriptors, "tangents", tangents)
		pbrBinary = pushF32Array(pbrBinary, pbrDescriptors, "binormals", binormals)
	}

	textureIDJSON := object{}
	for index, set := range textureSets {
		maps := object{}
		for _, pair := range set {
			maps[pair[0]] = pair[1]
		}
		textureIDJSON[strconv.Itoa(index)] = maps
	}
	attributeEncoding := object{
		"positions":    "flat xyz floats",
		"normals":      "flat xyz floats",
		"uvs":          "flat uv pairs from SCG TEXCOORD0; planar UVs generated where the client group has no UV stream",
		"vertexColors": "flat rgba floats from SCG vertex color; default white when absent",
		"textureIds":   textureIDJSON,
	}
	if includePBR {
		attributeEncoding["tangents"] = "flat xyz floats from SCG TANGENT"
		attributeEncoding["binormals"] = "flat xyz floats from SCG BINORMAL"
	}
	sourceName := tankID
	if name := rustFileName(sc2Source); name != "" {
		sourceName = name
	}
	sourceName = trimEndMatches(trimEndMatches(sourceName, ".sc2.dvpl"), ".sc2")
	var webp any
	if textureFormat == "webp" {
		webp = webpQuality
	}
	materialList := make([]any, len(materialsJSON))
	for index, material := range materialsJSON {
		materialList[index] = material
	}
	metadata := object{
		"formatVersion":   5,
		"pbrEnabled":      includePBR,
		"textureFormat":   textureFormat,
		"webpQuality":     webp,
		"pbrWebpLossless": includePBR && textureFormat == "webp",
		"tank":            sourceName,
		"coordinateSpace": "SC2 world/model coordinates; render-node world transforms applied",
		"primitive":       "triangles",
		"source": object{
			"scene":             sc2Source,
			"geometry":          scgSource,
			"provenance":        "Decoded from the locally installed World of Tanks Blitz client; no tank-viewer site assets or code used.",
			"omittedSceneNodes": []string{"crash state", "HP_* marker helpers"},
		},
		"materials":             materialList,
		"attributeEncoding":     attributeEncoding,
		"groups":                groupRanges,
		"generatedUvGroupCount": generatedUVGroups,
	}
	binaryFile, pbrFile := "model.bin", "model.pbr.bin"
	if compress {
		binaryFile, pbrFile = "model.bin.br", "model.pbr.bin.br"
	}
	metadata["binaryGeometry"] = object{"file": binaryFile, "byteLength": len(buffer), "arrays": arrays}
	if includePBR {
		metadata["pbrBinaryGeometry"] = object{"file": pbrFile, "byteLength": len(pbrBinary), "arrays": pbrDescriptors}
	}

	return &exportedModel{
		metadata:          metadata,
		binary:            buffer,
		pbrBinary:         pbrBinary,
		textures:          textures,
		vertexCount:       len(positions) / 3,
		triangleCount:     len(indices) / 3,
		materialCount:     len(materialsJSON),
		fallbackMaps:      fallbackMaps,
		generatedUVGroups: generatedUVGroups,
	}, nil
}

func trimEndMatches(value, suffix string) string {
	for suffix != "" && strings.HasSuffix(value, suffix) {
		value = value[:len(value)-len(suffix)]
	}
	return value
}

// rustFileName mirrors Path::file_name: the final normal component, or "" for
// paths ending in "..".
func rustFileName(path string) string {
	trimmed := strings.TrimRight(path, "/"+string(filepath.Separator))
	if trimmed == "" {
		return ""
	}
	name := filepath.Base(trimmed)
	if name == ".." || name == "." {
		return ""
	}
	return name
}

// rustFileStem mirrors Path::file_stem.
func rustFileStem(name string) string {
	if name == ".." {
		return name
	}
	index := strings.LastIndexByte(name, '.')
	if index <= 0 {
		return name
	}
	return name[:index]
}

// setExtension mirrors PathBuf::set_extension for a non-empty extension.
func setExtension(path, extension string) string {
	name := rustFileName(path)
	if name == "" {
		return path
	}
	stem := rustFileStem(name)
	trimmed := strings.TrimRight(path, "/")
	return trimmed[:len(trimmed)-len(name)] + stem + "." + extension
}

func dataNodeID(id *archive.Value, label string) (uint64, error) {
	if id.Kind == archive.Bytes && len(id.Bytes) == 8 {
		return binary.LittleEndian.Uint64(id.Bytes), nil
	}
	if value, ok := id.AsU64(); ok {
		return value, nil
	}
	return 0, fmt.Errorf("unexpected SC2 %s id: %s", label, id.Debug())
}

func materialNodes(scene *archive.Value) (map[uint64]*archive.Value, error) {
	result := map[uint64]*archive.Value{}
	nodes, _ := scene.Get("#dataNodes").AsArray()
	for index := range nodes {
		node := &nodes[index]
		id := node.Get("#id")
		if id == nil {
			continue
		}
		value, err := dataNodeID(id, "data-node")
		if err != nil {
			return nil, err
		}
		result[value] = node
	}
	return result, nil
}

func nodeName(node *archive.Value, fallback string) string {
	if name, ok := node.Get("name").AsString(); ok {
		return name
	}
	return fallback
}

// sortedValues returns map values in key order.
func sortedValues(entries map[string]archive.Value) []*archive.Value {
	keys := archive.SortedKeys(entries)
	out := make([]*archive.Value, len(keys))
	for index, key := range keys {
		value := entries[key]
		out[index] = &value
	}
	return out
}

func renderBatches(component *archive.Value) ([]*archive.Value, bool) {
	batches, ok := component.Get("rc.renderObj").Get("ro.batches").AsMap()
	if !ok {
		return nil, false
	}
	return sortedValues(batches), true
}

func isComponent(component *archive.Value, typename string) bool {
	name, ok := component.Get("comp.typename").AsString()
	return ok && name == typename
}

func visibleBatches(scene *archive.Value, materials map[uint64]*archive.Value, includePBR bool) ([]binding, error) {
	var result []binding
	seen := map[string]bool{}
	var visit func(nodes []archive.Value, parents []string) error
	visit = func(nodes []archive.Value, parents []string) error {
		for index := range nodes {
			node := &nodes[index]
			name := nodeName(node, "unnamed")
			path := append(append([]string(nil), parents...), name)
			skip := false
			for _, part := range path {
				lowered := asciiLower(part)
				if strings.Contains(lowered, "crash") || strings.HasPrefix(lowered, "hp_") {
					skip = true
				}
			}
			if skip {
				continue
			}
			nodeTransform, err := nodeTransformOf(node)
			if err != nil {
				return err
			}
			if components, ok := node.Get("components").AsMap(); ok {
				for _, component := range sortedValues(components) {
					if !isComponent(component, "RenderComponent") {
						continue
					}
					batches, ok := renderBatches(component)
					if !ok {
						continue
					}
					for _, batch := range batches {
						geometryID, ok := batch.Get("rb.datasource").AsU64()
						if !ok {
							continue
						}
						materialID, ok := batch.Get("rb.nmatname").AsU64()
						if !ok {
							return fmt.Errorf("SC2 render batch has invalid material id")
						}
						material, err := materialInfoOf(materialID, materials)
						if err != nil {
							return err
						}
						signature := fmt.Sprintf("%d|%s|%t|%x|%q|%v|%v|%v", geometryID, newTextureSet(material.maps).key(),
							material.alphaTest, math.Float32bits(material.alphaThreshold), nodeTransform.node,
							floatBits(nodeTransform.translation[:]), floatBits(nodeTransform.rotation[:]), floatBits(nodeTransform.scale[:]))
						if seen[signature] {
							continue
						}
						seen[signature] = true
						if !includePBR {
							material.maps = baseColorOnly(material.maps)
						}
						result = append(result, binding{
							geometryID: geometryID,
							materialID: materialID,
							node:       "/" + strings.Join(path, "/"),
							transform:  nodeTransform,
							material:   material,
						})
					}
				}
			}
			if children, ok := node.Get("#hierarchy").AsArray(); ok {
				if err := visit(children, path); err != nil {
					return err
				}
			}
		}
		return nil
	}
	nodes, _ := scene.Get("#hierarchy").AsArray()
	if err := visit(nodes, nil); err != nil {
		return nil, err
	}
	return result, nil
}

func floatBits(values []float32) []uint32 {
	out := make([]uint32, len(values))
	for index, value := range values {
		out[index] = math.Float32bits(value)
	}
	return out
}

func nodeTransformOf(node *archive.Value) (transform, error) {
	name := nodeName(node, "unnamed")
	components, ok := node.Get("components").AsMap()
	if !ok {
		return transform{}, fmt.Errorf("node %s has no component map", rs.Quote(name))
	}
	var matches []*archive.Value
	for _, component := range sortedValues(components) {
		if isComponent(component, "TransformComponent") {
			matches = append(matches, component)
		}
	}
	if len(matches) != 1 {
		return transform{}, fmt.Errorf("expected one transform component on node %s, got %d", rs.Quote(name), len(matches))
	}
	component := matches[0]
	result := transform{node: name}
	var err error
	translation, err := vector(component.Get("tc.worldTranslation"), []float32{0, 0, 0}, name, "tc.worldTranslation")
	if err != nil {
		return transform{}, err
	}
	rotation, err := vector(component.Get("tc.worldRotation"), []float32{0, 0, 0, 1}, name, "tc.worldRotation")
	if err != nil {
		return transform{}, err
	}
	scale, err := vector(component.Get("tc.worldScale"), []float32{1, 1, 1}, name, "tc.worldScale")
	if err != nil {
		return transform{}, err
	}
	copy(result.translation[:], translation)
	copy(result.rotation[:], rotation)
	copy(result.scale[:], scale)
	return result, nil
}

func vector(value *archive.Value, fallback []float32, node, key string) ([]float32, error) {
	if value == nil {
		return fallback, nil
	}
	items, ok := value.AsArray()
	if !ok || len(items) != len(fallback) {
		return nil, fmt.Errorf("invalid %s on node %s", key, rs.Quote(node))
	}
	result := make([]float32, len(items))
	for index := range items {
		number, ok := items[index].AsF64()
		if !ok {
			return nil, fmt.Errorf("invalid %s on node %s", key, rs.Quote(node))
		}
		if result[index] = float32(number); !finite32(result[index]) {
			return nil, fmt.Errorf("non-finite %s on node %s", key, rs.Quote(node))
		}
	}
	return result, nil
}

var materialSlots = []struct {
	role       string
	candidates []string
}{
	// Older client assets carry only the pre-PBR slot names, so each role
	// accepts candidate slots in priority order.
	{"baseColor", []string{"baseColorMap", "albedo"}},
	{"normal", []string{"baseNormalMap", "normalmap"}},
	{"roughnessMetallic", []string{"baseRMMap"}},
	{"misc", []string{"miscMap"}},
}

func materialInfoOf(id uint64, materials map[uint64]*archive.Value) (materialInfo, error) {
	instance, ok := materials[id]
	if !ok {
		return materialInfo{}, fmt.Errorf("SC2 render batch references missing material id %d", id)
	}
	parent := instance
	if parentID, ok := instance.Get("parentMaterialKey").AsU64(); ok {
		if found, ok := materials[parentID]; ok {
			parent = found
		}
	}
	var configs []*archive.Value
	if entries, ok := parent.AsMap(); ok {
		for _, key := range archive.SortedKeys(entries) {
			value := entries[key]
			if strings.HasPrefix(key, "configArchive_") && value.Kind == archive.Map {
				configs = append(configs, &value)
			}
		}
	}
	if _, ok := parent.Get("textures").AsMap(); len(configs) == 0 && ok {
		configs = append(configs, parent)
	}
	sort.SliceStable(configs, func(i, j int) bool {
		return isDefaultConfig(configs[i]) && !isDefaultConfig(configs[j])
	})
	best := -1
	var bestMaps map[string]string
	var bestConfig *archive.Value
	for _, config := range configs {
		maps := map[string]string{}
		textures := config.Get("textures")
		for _, slot := range materialSlots {
			for _, candidate := range slot.candidates {
				if path, ok := textures.Get(candidate).AsString(); ok && path != "" {
					maps[slot.role] = path
					break
				}
			}
		}
		available := len(maps)
		if available == len(materialSlots) {
			best, bestMaps, bestConfig = available, maps, config
			break
		}
		if best < 0 || available > best {
			best, bestMaps, bestConfig = available, maps, config
		}
	}
	if bestConfig == nil {
		maps := map[string]string{}
		for _, slot := range materialSlots {
			maps[slot.role] = "@fallback/" + slot.role
		}
		return materialInfo{maps: maps, alphaThreshold: 0.35}, nil
	}
	threshold := float32(0.35)
	if data, ok := bestConfig.Get("properties").Get("alphatestThreshold").AsBytes(); ok && len(data) >= 4 {
		value := math.Float32frombits(binary.LittleEndian.Uint32(data[len(data)-4:]))
		if finite32(value) {
			threshold = value
		}
	}
	alphaTest := bestConfig.Get("enabledPresets").Get("AlphaTest").AsBool()
	for _, slot := range materialSlots {
		if _, ok := bestMaps[slot.role]; !ok {
			bestMaps[slot.role] = "@fallback/" + slot.role
		}
	}
	return materialInfo{maps: bestMaps, alphaTest: alphaTest, alphaThreshold: threshold}, nil
}

func isDefaultConfig(config *archive.Value) bool {
	name, ok := config.Get("configName").AsString()
	return ok && name == "Default"
}

func buildMaterialJSON(textureSets []textureSet, bindings []binding, selectedMaps []textureSet,
	textureFormat string, includePBR bool) []object {
	out := make([]object, len(textureSets))
	for id, maps := range textureSets {
		bindingIndex := 0
		for index, selected := range selectedMaps {
			if compareTextureSets(selected, maps) == 0 {
				bindingIndex = index
				break
			}
		}
		b := bindings[bindingIndex]
		mapJSON := object{}
		for _, pair := range maps {
			role, path := pair[0], pair[1]
			fallback := strings.HasPrefix(path, "@fallback/")
			folder := "pbr"
			if role == "baseColor" {
				folder = "base"
			}
			var file string
			var materialPath any
			if fallback {
				file = "textures/" + folder + "/__fallback_" + role + ".png"
			} else {
				basename := rustFileName(path)
				if basename == "" {
					basename = path
				}
				file = "textures/" + folder + "/" + rustFileStem(basename) + "." + textureFormat
				materialPath = path
			}
			mapJSON[role] = object{"materialPath": materialPath, "file": file, "fallback": fallback}
		}
		item := object{
			"id":             id,
			"maps":           mapJSON,
			"alphaTest":      b.material.alphaTest,
			"alphaThreshold": jsonx.F64(b.material.alphaThreshold),
		}
		if includePBR {
			item["normalEncoding"] = "DXT5nm AG"
			item["roughnessMetallicEncoding"] = "DXT5nm AG (roughness, metallic)"
			item["miscEncoding"] = "DXT5nm AG (ambient occlusion, emissive)"
		}
		out[id] = item
	}
	return out
}

func planarUVs(positions []vec3) []vec2 {
	minimum := vec3{float32(math.Inf(1)), float32(math.Inf(1)), float32(math.Inf(1))}
	maximum := vec3{float32(math.Inf(-1)), float32(math.Inf(-1)), float32(math.Inf(-1))}
	for _, position := range positions {
		for axis := range 3 {
			minimum[axis] = min32(minimum[axis], position[axis])
			maximum[axis] = max32(maximum[axis], position[axis])
		}
	}
	axes := []int{0, 1, 2}
	span := func(axis int) float32 { return maximum[axis] - minimum[axis] }
	sort.SliceStable(axes, func(i, j int) bool { return totalLess(span(axes[j]), span(axes[i])) })
	spans := [2]float32{max32(span(axes[0]), 1e-12), max32(span(axes[1]), 1e-12)}
	out := make([]vec2, len(positions))
	for index, position := range positions {
		out[index] = vec2{
			(position[axes[0]] - minimum[axes[0]]) / spans[0],
			(position[axes[1]] - minimum[axes[1]]) / spans[1],
		}
	}
	return out
}

// totalLess orders floats like f32::total_cmp.
func totalLess(a, b float32) bool {
	key := func(v float32) int32 {
		bits := int32(math.Float32bits(v))
		return bits ^ int32(uint32(bits>>31)>>1)
	}
	return key(a) < key(b)
}

func fallbackNormals(positions []vec3, indices []uint32) []vec3 {
	sums := make([]vec3, len(positions))
	for start := 0; start+3 <= len(indices); start += 3 {
		triangle := indices[start : start+3]
		a, b, c := positions[triangle[0]], positions[triangle[1]], positions[triangle[2]]
		face := cross3(sub3(b, a), sub3(c, a))
		for _, index := range triangle {
			for axis := range 3 {
				sums[index][axis] += face[axis]
			}
		}
	}
	out := make([]vec3, len(sums))
	for index, normal := range sums {
		if length := length3(normal); length > 1e-15 {
			out[index] = scale3(normal, 1/length)
		} else {
			out[index] = vec3{0, 0, 1}
		}
	}
	return out
}

func fallbackTangentBasis(positions, normals []vec3, uvs []vec2, indices []uint32) ([]vec3, []vec3) {
	tangentSums := make([]vec3, len(positions))
	binormalSums := make([]vec3, len(positions))
	for start := 0; start+3 <= len(indices); start += 3 {
		ia, ib, ic := indices[start], indices[start+1], indices[start+2]
		ab := sub3(positions[ib], positions[ia])
		ac := sub3(positions[ic], positions[ia])
		uvAB := vec2{uvs[ib][0] - uvs[ia][0], uvs[ib][1] - uvs[ia][1]}
		uvAC := vec2{uvs[ic][0] - uvs[ia][0], uvs[ic][1] - uvs[ia][1]}
		determinant := mul(uvAB[0], uvAC[1]) - mul(uvAB[1], uvAC[0])
		if abs32(determinant) <= 1e-12 {
			continue
		}
		inverse := recip(determinant)
		tangent := scale3(sub3(scale3(ab, uvAC[1]), scale3(ac, uvAB[1])), inverse)
		binormal := scale3(sub3(scale3(ac, uvAB[0]), scale3(ab, uvAC[0])), inverse)
		for _, index := range []uint32{ia, ib, ic} {
			tangentSums[index] = add3(tangentSums[index], tangent)
			binormalSums[index] = add3(binormalSums[index], binormal)
		}
	}
	tangents := make([]vec3, len(normals))
	binormals := make([]vec3, len(normals))
	for index, normal := range normals {
		tangent := sub3(tangentSums[index], scale3(normal, dot3(normal, tangentSums[index])))
		length := length3(tangent)
		if length <= 1e-15 {
			axis := vec3{0, 1, 0}
			if abs32(normal[2]) < 0.9 {
				axis = vec3{0, 0, 1}
			}
			tangent = cross3(axis, normal)
			length = max32(length3(tangent), 1)
		}
		tangent = scale3(tangent, recip(length))
		cross := cross3(normal, tangent)
		handedness := float32(1)
		if dot3(cross, binormalSums[index]) < 0 {
			handedness = -1
		}
		tangents[index] = tangent
		binormals[index] = scale3(cross, handedness)
	}
	return tangents, binormals
}

func transformPosition(position vec3, t *transform) (vec3, error) {
	scaled := vec3{mul(position[0], t.scale[0]), mul(position[1], t.scale[1]), mul(position[2], t.scale[2])}
	rotated, err := rotate(scaled, t.rotation, t.node)
	if err != nil {
		return vec3{}, err
	}
	return add3(rotated, t.translation), nil
}

func transformNormal(normal vec3, t *transform) (vec3, error) {
	for _, scale := range t.scale {
		if abs32(scale) <= 1e-15 {
			return vec3{}, fmt.Errorf("zero scale on %s", t.node)
		}
	}
	inverseScaled := vec3{normal[0] / t.scale[0], normal[1] / t.scale[1], normal[2] / t.scale[2]}
	rotated, err := rotate(inverseScaled, t.rotation, t.node)
	if err != nil {
		return vec3{}, err
	}
	length := length3(rotated)
	if length <= 1e-15 {
		return vec3{}, fmt.Errorf("zero-length normal on %s", t.node)
	}
	return scale3(rotated, recip(length)), nil
}

func transformDirection(direction vec3, t *transform) (vec3, error) {
	origin, err := transformPosition(vec3{}, t)
	if err != nil {
		return vec3{}, err
	}
	endpoint, err := transformPosition(direction, t)
	if err != nil {
		return vec3{}, err
	}
	result := sub3(endpoint, origin)
	length := length3(result)
	if length <= 1e-15 || !finite32(length) {
		return vec3{}, fmt.Errorf("invalid tangent direction on %s", t.node)
	}
	return scale3(result, recip(length)), nil
}

func normalizedQuaternion(q [4]float32, node string) ([4]float32, error) {
	var sum float32
	for _, component := range q {
		sum += mul(component, component)
	}
	norm := sqrt32(sum)
	if norm <= 1e-15 {
		return q, fmt.Errorf("zero-length quaternion on %s", node)
	}
	return [4]float32{q[0] / norm, q[1] / norm, q[2] / norm, q[3] / norm}, nil
}

func rotate(value vec3, quaternion [4]float32, node string) (vec3, error) {
	q, err := normalizedQuaternion(quaternion, node)
	if err != nil {
		return vec3{}, err
	}
	qx, qy, qz, qw := q[0], q[1], q[2], q[3]
	x, y, z := value[0], value[1], value[2]
	tx := mul(2, mul(qy, z)-mul(qz, y))
	ty := mul(2, mul(qz, x)-mul(qx, z))
	tz := mul(2, mul(qx, y)-mul(qy, x))
	return vec3{
		x + mul(qw, tx) + mul(qy, tz) - mul(qz, ty),
		y + mul(qw, ty) + mul(qz, tx) - mul(qx, tz),
		z + mul(qw, tz) + mul(qx, ty) - mul(qy, tx),
	}, nil
}

func worldMatrix(t *transform) ([16]float32, error) {
	q, err := normalizedQuaternion(t.rotation, t.node)
	if err != nil {
		return [16]float32{}, err
	}
	qx, qy, qz, qw := q[0], q[1], q[2], q[3]
	sx, sy, sz := t.scale[0], t.scale[1], t.scale[2]
	return [16]float32{
		mul(1-mul(2, mul(qy, qy)+mul(qz, qz)), sx),
		mul(mul(2, mul(qx, qy)+mul(qz, qw)), sx),
		mul(mul(2, mul(qx, qz)-mul(qy, qw)), sx),
		0,
		mul(mul(2, mul(qx, qy)-mul(qz, qw)), sy),
		mul(1-mul(2, mul(qx, qx)+mul(qz, qz)), sy),
		mul(mul(2, mul(qy, qz)+mul(qx, qw)), sy),
		0,
		mul(mul(2, mul(qx, qz)+mul(qy, qw)), sz),
		mul(mul(2, mul(qy, qz)-mul(qx, qw)), sz),
		mul(1-mul(2, mul(qx, qx)+mul(qy, qy)), sz),
		0,
		t.translation[0], t.translation[1], t.translation[2], 1,
	}, nil
}

func alignBuffer(buffer []byte, alignment int) ([]byte, int) {
	padding := (alignment - len(buffer)%alignment) % alignment
	buffer = append(buffer, make([]byte, padding)...)
	return buffer, len(buffer)
}

func arrayDescriptor(kind string, offset, count int) object {
	return object{"type": kind, "byteOffset": offset, "count": count}
}

func pushF32Array(buffer []byte, descriptors object, name string, values []float32) []byte {
	buffer, offset := alignBuffer(buffer, 4)
	for _, value := range values {
		buffer = binary.LittleEndian.AppendUint32(buffer, math.Float32bits(value))
	}
	descriptors[name] = arrayDescriptor("float32", offset, len(values))
	return buffer
}

func appendUints(buffer []byte, values []uint32, width int) []byte {
	for _, value := range values {
		switch width {
		case 1:
			buffer = append(buffer, byte(value))
		case 2:
			buffer = binary.LittleEndian.AppendUint16(buffer, uint16(value))
		default:
			buffer = binary.LittleEndian.AppendUint32(buffer, value)
		}
	}
	return buffer
}
