package export

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StrictMaps describes the maps.yaml part of a v1 data export. Map names are
// dynamic keys; their values are structured records.
type StrictMaps struct {
	Maps Optional[map[string]StrictMap] `json:"maps,omitempty"`
}

type StrictMap struct {
	AssaultRespawnPoints     Optional[StrictMapRespawnPoints] `json:"assaultRespawnPoints,omitempty"`
	AvailableModes           Optional[[]StrictInteger]        `json:"availableModes,omitempty"`
	AvaliableInTrainingRoom  Optional[bool]                   `json:"avaliableInTrainingRoom,omitempty"`
	ID                       Optional[StrictInteger]          `json:"id,omitempty"`
	Levels                   Optional[[]StrictInteger]        `json:"levels,omitempty"`
	LocalName                Optional[string]                 `json:"localName,omitempty"`
	ShadowMapsAvailable      Optional[bool]                   `json:"shadowMapsAvailable,omitempty"`
	SpriteFrame              Optional[StrictInteger]          `json:"spriteFrame,omitempty"`
	SupremacyPointsThreshold Optional[StrictInteger]          `json:"supremacyPointsThreshold,omitempty"`
	Tags                     Optional[string]                 `json:"tags,omitempty"`
	VehicleCamouflageKind    Optional[string]                 `json:"vehicleCamouflageKind,omitempty"`
}

type StrictMapRespawnPoints struct {
	Allies  Optional[[]StrictMapRespawn] `json:"allies,omitempty"`
	Enemies Optional[[]StrictMapRespawn] `json:"enemies,omitempty"`
}

type StrictMapRespawn struct {
	Points        Optional[[]StrictMapPoint] `json:"points,omitempty"`
	RespawnNumber Optional[StrictInteger]    `json:"respawnNumber,omitempty"`
}

// StrictMapPoint preserves the exact JSON numbers in a two-coordinate point.
type StrictMapPoint [2]StrictNumber

func (p *StrictMapPoint) UnmarshalJSON(data []byte) error {
	var values []StrictNumber
	if err := decodeStrictJSON(data, &values); err != nil {
		return err
	}
	if len(values) != 2 {
		return fmt.Errorf("map point must contain two coordinates")
	}
	copy(p[:], values)
	return nil
}

// StrictHangarParameters describes hangarParamsConfig.yaml, whose parameter
// names are dynamic and whose values use keyedArchive wrappers.
type StrictHangarParameters struct {
	KeyedArchive Optional[map[string]StrictHangarParameter] `json:"keyedArchive,omitempty"`
}

type StrictHangarParameter struct {
	KeyedArchive Optional[StrictHangarArchive] `json:"keyedArchive,omitempty"`
}

type StrictHangarArchive struct {
	BiggerIsBetter Optional[StrictHangarBool]   `json:"biggerIsBetter,omitempty"`
	Digits         Optional[StrictHangarInt32]  `json:"digits,omitempty"`
	Units          Optional[StrictHangarString] `json:"units,omitempty"`
}

type StrictHangarBool struct {
	Bool Optional[bool] `json:"bool,omitempty"`
}
type StrictHangarInt32 struct {
	Int32 Optional[StrictInteger] `json:"int32,omitempty"`
}
type StrictHangarString struct {
	String Optional[string] `json:"string,omitempty"`
}

// Reject JSON null for dictionary and array records: encoding/json otherwise
// silently turns it into an empty struct, losing the distinction.
func decodeNonNullObject(data []byte, target any) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("expected object, got null")
	}
	return decodeStrictJSON(data, target)
}
func (v *StrictMap) UnmarshalJSON(data []byte) error {
	type plain StrictMap
	return decodeNonNullObject(data, (*plain)(v))
}
func (v *StrictMapRespawn) UnmarshalJSON(data []byte) error {
	type plain StrictMapRespawn
	return decodeNonNullObject(data, (*plain)(v))
}
func (v *StrictHangarParameter) UnmarshalJSON(data []byte) error {
	type plain StrictHangarParameter
	return decodeNonNullObject(data, (*plain)(v))
}

// Keep encoding/json imported here for the compile-time marshaler assertion.
var _ json.Unmarshaler = (*StrictMapPoint)(nil)
