package export

// StrictMaps describes the maps.yaml part of a v1 data export. Map names are
// dynamic keys; their values are structured records.
type StrictMaps struct {
	Maps Optional[map[string]StrictMap] `json:"maps,omitzero"`
}

type StrictMap struct {
	AssaultRespawnPoints     Optional[StrictMapRespawnPoints] `json:"assaultRespawnPoints,omitzero"`
	AvailableModes           Optional[[]StrictInteger]        `json:"availableModes,omitzero"`
	AvaliableInTrainingRoom  Optional[bool]                   `json:"avaliableInTrainingRoom,omitzero"`
	ID                       Optional[StrictInteger]          `json:"id,omitzero"`
	Levels                   Optional[[]StrictInteger]        `json:"levels,omitzero"`
	LocalName                Optional[string]                 `json:"localName,omitzero"`
	ShadowMapsAvailable      Optional[bool]                   `json:"shadowMapsAvailable,omitzero"`
	SpriteFrame              Optional[StrictInteger]          `json:"spriteFrame,omitzero"`
	SupremacyPointsThreshold Optional[StrictInteger]          `json:"supremacyPointsThreshold,omitzero"`
	Tags                     Optional[string]                 `json:"tags,omitzero"`
	VehicleCamouflageKind    Optional[string]                 `json:"vehicleCamouflageKind,omitzero"`
}

type StrictMapRespawnPoints struct {
	Allies  Optional[[]StrictMapRespawn] `json:"allies,omitzero"`
	Enemies Optional[[]StrictMapRespawn] `json:"enemies,omitzero"`
}

type StrictMapRespawn struct {
	Points        Optional[[]StrictMapPoint] `json:"points,omitzero"`
	RespawnNumber Optional[StrictInteger]    `json:"respawnNumber,omitzero"`
}

// StrictMapPoint preserves the exact JSON numbers in a two-coordinate point.
type StrictMapPoint [2]StrictNumber

// StrictHangarParameters describes hangarParamsConfig.yaml, whose parameter
// names are dynamic and whose values use keyedArchive wrappers.
type StrictHangarParameters struct {
	KeyedArchive Optional[map[string]StrictHangarParameter] `json:"keyedArchive,omitzero"`
}

type StrictHangarParameter struct {
	KeyedArchive Optional[StrictHangarArchive] `json:"keyedArchive,omitzero"`
}

type StrictHangarArchive struct {
	BiggerIsBetter Optional[StrictHangarBool]   `json:"biggerIsBetter,omitzero"`
	Digits         Optional[StrictHangarInt32]  `json:"digits,omitzero"`
	Units          Optional[StrictHangarString] `json:"units,omitzero"`
}

type StrictHangarBool struct {
	Bool Optional[bool] `json:"bool,omitzero"`
}
type StrictHangarInt32 struct {
	Int32 Optional[StrictInteger] `json:"int32,omitzero"`
}
type StrictHangarString struct {
	String Optional[string] `json:"string,omitzero"`
}
