package export

import (
	"go.yaml.in/yaml/v3"
)

// RawMaps mirrors the maps.yaml document carried by the data section. The
// dynamic keys are the map names; the field set and the client's
// avaliableInTrainingRoom spelling follow the source document, which the
// strict reader validates as StrictMaps.
type RawMaps struct {
	Maps map[string]RawMap `json:"maps"`
}

type RawMap struct {
	AssaultRespawnPoints     *RawRespawnSides `json:"assaultRespawnPoints,omitempty"`
	AvailableModes           []int64          `json:"availableModes,omitempty"`
	AvaliableInTrainingRoom  *bool            `json:"avaliableInTrainingRoom,omitempty"`
	ID                       *int64           `json:"id,omitempty"`
	Levels                   []int64          `json:"levels,omitempty"`
	LocalName                string           `json:"localName,omitempty"`
	ShadowMapsAvailable      *bool            `json:"shadowMapsAvailable,omitempty"`
	SpriteFrame              *int64           `json:"spriteFrame,omitempty"`
	SupremacyPointsThreshold *int64           `json:"supremacyPointsThreshold,omitempty"`
	Tags                     string           `json:"tags,omitempty"`
	VehicleCamouflageKind    string           `json:"vehicleCamouflageKind,omitempty"`
}

type RawRespawnSides struct {
	Allies  []RawRespawn `json:"allies,omitempty"`
	Enemies []RawRespawn `json:"enemies,omitempty"`
}

type RawRespawn struct {
	Points        [][2]float64 `json:"points,omitempty"`
	RespawnNumber *int64       `json:"respawnNumber,omitempty"`
}

// yamlMaps mirrors the YAML layout of maps.yaml for decoding.
type yamlMaps struct {
	Maps map[string]yamlMap `yaml:"maps"`
}

type yamlMap struct {
	AssaultRespawnPoints     *yamlRespawnSides `yaml:"assaultRespawnPoints"`
	AvailableModes           []int64           `yaml:"availableModes"`
	AvaliableInTrainingRoom  *bool             `yaml:"avaliableInTrainingRoom"`
	ID                       *int64            `yaml:"id"`
	Levels                   []int64           `yaml:"levels"`
	LocalName                string            `yaml:"localName"`
	ShadowMapsAvailable      *bool             `yaml:"shadowMapsAvailable"`
	SpriteFrame              *int64            `yaml:"spriteFrame"`
	SupremacyPointsThreshold *int64            `yaml:"supremacyPointsThreshold"`
	Tags                     string            `yaml:"tags"`
	VehicleCamouflageKind    string            `yaml:"vehicleCamouflageKind"`
}

type yamlRespawnSides struct {
	Allies  []yamlRespawn `yaml:"allies"`
	Enemies []yamlRespawn `yaml:"enemies"`
}

type yamlRespawn struct {
	Points        [][2]float64 `yaml:"points"`
	RespawnNumber *int64       `yaml:"respawnNumber"`
}

// parseRawMaps decodes maps.yaml into the typed data-section document.
func parseRawMaps(data []byte) (*RawMaps, error) {
	var parsed yamlMaps
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	raw := &RawMaps{Maps: make(map[string]RawMap, len(parsed.Maps))}
	for name, entry := range parsed.Maps {
		values := RawMap{
			AvailableModes:           entry.AvailableModes,
			AvaliableInTrainingRoom:  entry.AvaliableInTrainingRoom,
			ID:                       entry.ID,
			Levels:                   entry.Levels,
			LocalName:                entry.LocalName,
			ShadowMapsAvailable:      entry.ShadowMapsAvailable,
			SpriteFrame:              entry.SpriteFrame,
			SupremacyPointsThreshold: entry.SupremacyPointsThreshold,
			Tags:                     entry.Tags,
			VehicleCamouflageKind:    entry.VehicleCamouflageKind,
		}
		if entry.AssaultRespawnPoints != nil {
			sides := &RawRespawnSides{Allies: respawns(entry.AssaultRespawnPoints.Allies), Enemies: respawns(entry.AssaultRespawnPoints.Enemies)}
			values.AssaultRespawnPoints = sides
		}
		raw.Maps[name] = values
	}
	return raw, nil
}

func respawns(groups []yamlRespawn) []RawRespawn {
	if len(groups) == 0 {
		return nil
	}
	out := make([]RawRespawn, len(groups))
	for i, group := range groups {
		out[i] = RawRespawn{Points: group.Points, RespawnNumber: group.RespawnNumber}
	}
	return out
}
