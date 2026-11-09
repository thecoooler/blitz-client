package export

import (
	"github.com/thecoooler/blitz-client/client"
)

// RawMaps mirrors the maps.yaml document carried by the data section. The
// dynamic keys are the map names; the field set and the client's
// avaliableInTrainingRoom spelling follow the source document, which the
// catalog decoder validates and aliases.
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

// rawMapsFromDocument projects the shared maps document onto the data-section
// wire shape. The frame and training-room spelling aliases are resolved by the
// catalog decoder; this projection keeps the document's existing field names.
func rawMapsFromDocument(document *client.MapsDocument) *RawMaps {
	raw := &RawMaps{Maps: make(map[string]RawMap, len(document.Maps))}
	for _, name := range document.Order {
		entry := document.Maps[name]
		frame := entry.Frame
		if frame == nil {
			frame = entry.SpriteFrame
		}
		training := entry.AvaliableInTrainingRoom
		if training == nil {
			training = entry.AvailableInTrainingRoom
		}
		values := RawMap{
			AvailableModes:           entry.AvailableModes,
			AvaliableInTrainingRoom:  training,
			ID:                       entry.ID,
			Levels:                   entry.Levels,
			LocalName:                entry.LocalName,
			ShadowMapsAvailable:      entry.ShadowMapsAvailable,
			SpriteFrame:              frame,
			SupremacyPointsThreshold: entry.SupremacyPointsThreshold,
			Tags:                     entry.Tags,
			VehicleCamouflageKind:    entry.VehicleCamouflageKind,
		}
		if entry.AssaultRespawnPoints != nil {
			values.AssaultRespawnPoints = &RawRespawnSides{
				Allies:  rawRespawns(entry.AssaultRespawnPoints.Allies),
				Enemies: rawRespawns(entry.AssaultRespawnPoints.Enemies),
			}
		}
		raw.Maps[name] = values
	}
	return raw
}

func rawRespawns(groups []client.MapRespawn) []RawRespawn {
	if len(groups) == 0 {
		return nil
	}
	out := make([]RawRespawn, len(groups))
	for index, group := range groups {
		out[index] = RawRespawn{Points: group.Points, RespawnNumber: group.RespawnNumber}
	}
	return out
}
