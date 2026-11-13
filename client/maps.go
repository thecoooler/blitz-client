package client

import (
	"fmt"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/rs"
	"go.yaml.in/yaml/v3"
)

// MapsDocument is the decoded maps.yaml document. Order preserves the source
// order of the map names; Maps holds the entry per name.
type MapsDocument struct {
	Path  string
	Order []string
	Maps  map[string]MapDocumentEntry
}

// MapDocumentEntry contains the maps.yaml fields shared by catalog discovery
// and the export data document. Frame and SpriteFrame are kept separately so
// each projection can apply its own alias precedence.
type MapDocumentEntry struct {
	Name                     string
	ID                       *int64
	LocalName                string
	AvailableModes           []int64
	Frame                    *int64
	SpriteFrame              *int64
	AvailableInTrainingRoom  *bool
	AvaliableInTrainingRoom  *bool
	Levels                   []int64
	ShadowMapsAvailable      *bool
	SupremacyPointsThreshold *int64
	Tags                     string
	VehicleCamouflageKind    string
	AssaultRespawnPoints     *MapRespawnSides
}

// MapRespawnSides is the assault respawn point table of one map.
type MapRespawnSides struct {
	Allies  []MapRespawn
	Enemies []MapRespawn
}

// MapRespawn is one spawn group: its points and optional respawn number.
type MapRespawn struct {
	Points        [][2]float64
	RespawnNumber *int64
}

// yamlMapDocumentEntry mirrors one maps.yaml entry while decoding.
type yamlMapDocumentEntry struct {
	AssaultRespawnPoints     *yamlMapRespawnSides `yaml:"assaultRespawnPoints"`
	AvailableModes           []int64              `yaml:"availableModes"`
	AvailableInTrainingRoom  *bool                `yaml:"availableInTrainingRoom"`
	AvaliableInTrainingRoom  *bool                `yaml:"avaliableInTrainingRoom"`
	ID                       *int64               `yaml:"id"`
	Levels                   []int64              `yaml:"levels"`
	LocalName                string               `yaml:"localName"`
	ShadowMapsAvailable      *bool                `yaml:"shadowMapsAvailable"`
	Frame                    *int64               `yaml:"frame"`
	SpriteFrame              *int64               `yaml:"spriteFrame"`
	SupremacyPointsThreshold *int64               `yaml:"supremacyPointsThreshold"`
	Tags                     string               `yaml:"tags"`
	VehicleCamouflageKind    string               `yaml:"vehicleCamouflageKind"`
}

type yamlMapRespawnSides struct {
	Allies  []yamlMapRespawn `yaml:"allies"`
	Enemies []yamlMapRespawn `yaml:"enemies"`
}

type yamlMapRespawn struct {
	Points        [][2]float64 `yaml:"points"`
	RespawnNumber *int64       `yaml:"respawnNumber"`
}

// ReadMapsDocument reads maps.yml or maps.yaml, preferring maps.yml like the
// catalog. A missing document returns (nil, nil).
func ReadMapsDocument(source CatalogSource) (*MapsDocument, error) {
	relative := "maps.yml"
	if !source.Exists(relative) {
		relative = "maps.yaml"
		if !source.Exists(relative) {
			return nil, nil
		}
	}
	data, err := source.Open(relative)
	if err != nil {
		return nil, err
	}
	document, err := DecodeMapsDocument(data)
	if err != nil {
		return nil, blitz.ErrorAt("CLIENT_YAML_INVALID", err.Error(), source.Location(relative))
	}
	document.Path = source.Location(relative)
	return document, nil
}

// DecodeMapsDocument decodes the maps.yaml document. The frame/spriteFrame and
// training-room spelling aliases are resolved here, and the same numeric field
// rules apply to every consumer.
func DecodeMapsDocument(data []byte) (*MapsDocument, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	document := &MapsDocument{Maps: map[string]MapDocumentEntry{}}
	if len(root.Content) == 0 {
		return document, nil
	}
	top := root.Content[0]
	if top.Kind == 0 || (top.Kind == yaml.ScalarNode && top.Value == "") {
		return document, nil
	}
	if top.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("maps.yaml root must be a mapping")
	}
	var mapsNode *yaml.Node
	for index := 0; index+1 < len(top.Content); index += 2 {
		if top.Content[index].Value == "maps" {
			mapsNode = top.Content[index+1]
			break
		}
	}
	if mapsNode == nil {
		return document, nil
	}
	if mapsNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("maps must be a mapping")
	}
	for index := 0; index+1 < len(mapsNode.Content); index += 2 {
		name := mapsNode.Content[index].Value
		if _, exists := document.Maps[name]; exists {
			return nil, fmt.Errorf("duplicate map entry %s", rs.Quote(name))
		}
		var raw yamlMapDocumentEntry
		if err := mapsNode.Content[index+1].Decode(&raw); err != nil {
			return nil, fmt.Errorf("map %s: %w", rs.Quote(name), err)
		}
		document.Order = append(document.Order, name)
		document.Maps[name] = MapDocumentEntry{
			Name:                     name,
			ID:                       raw.ID,
			LocalName:                raw.LocalName,
			AvailableModes:           raw.AvailableModes,
			Frame:                    raw.Frame,
			SpriteFrame:              raw.SpriteFrame,
			AvailableInTrainingRoom:  raw.AvailableInTrainingRoom,
			AvaliableInTrainingRoom:  raw.AvaliableInTrainingRoom,
			Levels:                   raw.Levels,
			ShadowMapsAvailable:      raw.ShadowMapsAvailable,
			SupremacyPointsThreshold: raw.SupremacyPointsThreshold,
			Tags:                     raw.Tags,
			VehicleCamouflageKind:    raw.VehicleCamouflageKind,
			AssaultRespawnPoints:     raw.AssaultRespawnPoints.document(),
		}
	}
	return document, nil
}

func (sides *yamlMapRespawnSides) document() *MapRespawnSides {
	if sides == nil {
		return nil
	}
	return &MapRespawnSides{Allies: respawnDocuments(sides.Allies), Enemies: respawnDocuments(sides.Enemies)}
}

func respawnDocuments(groups []yamlMapRespawn) []MapRespawn {
	if len(groups) == 0 {
		return nil
	}
	out := make([]MapRespawn, len(groups))
	for index, group := range groups {
		out[index] = MapRespawn{Points: group.Points, RespawnNumber: group.RespawnNumber}
	}
	return out
}

// ReadMapsCatalog reads maps.yaml without localization, so LocalName keeps
// the raw scene path. Names are the maps.yaml entry keys.
func ReadMapsCatalog(dataDir string) ([]ClientMap, error) {
	return readMaps(dataDir, nil)
}

func readMaps(dataDir string, localized map[string]string) ([]ClientMap, error) {
	document, err := ReadMapsDocument(directorySource{dir: dataDir})
	if err != nil {
		return nil, err
	}
	if document == nil {
		return nil, blitz.ErrorAt("CLIENT_DATA_INVALID",
			"maps.yml or maps.yaml was not found in the client resource directory", dataDir)
	}
	maps, err := mapsFromDocument(document, localized)
	if err != nil {
		return nil, blitz.ErrorAt("CLIENT_YAML_INVALID", err.Error(), document.Path)
	}
	return maps, nil
}

func mapsFromDocument(document *MapsDocument, localized map[string]string) ([]ClientMap, error) {
	maps := []ClientMap{}
	for _, name := range document.Order {
		entry := document.Maps[name]
		if entry.ID == nil {
			return nil, fmt.Errorf("map %s is missing id", rs.Quote(name))
		}
		frame := entry.Frame
		if frame == nil {
			frame = entry.SpriteFrame
		}
		if frame == nil {
			return nil, fmt.Errorf("map %s has neither frame nor spriteFrame", rs.Quote(name))
		}
		modes := entry.AvailableModes
		if modes == nil {
			modes = []int64{}
		}
		training := false
		for _, value := range []*bool{entry.AvailableInTrainingRoom, entry.AvaliableInTrainingRoom} {
			if value != nil && *value {
				training = true
			}
		}
		localName := entry.LocalName
		if localized != nil {
			if translated, ok := localized["#maps:"+name+":"+localName]; ok {
				localName = translated
			}
		}
		maps = append(maps, ClientMap{
			Name:                    name,
			ID:                      *entry.ID,
			LocalName:               localName,
			AvailableInTrainingRoom: training,
			AvailableModes:          modes,
			SpriteFrame:             *frame,
			VehicleCamouflageKind:   entry.VehicleCamouflageKind,
		})
	}
	if len(maps) == 0 {
		return nil, fmt.Errorf("maps.yaml contains no map entries")
	}
	return maps, nil
}
