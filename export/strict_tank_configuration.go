// Tank configuration types based on observed v1 game resources.
// Fields reflect the current game resources, not inferred game semantics.
package export

import (
	"bytes"
	"fmt"
)

type StrictTankConfigurationCamouflage struct {
	ExclusionMask Optional[string] `json:"exclusionMask,omitempty"`
	PriceFactor   Optional[string] `json:"priceFactor,omitempty"`
	Tiling        Optional[string] `json:"tiling,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesAODecalsDecalTransform struct {
	Row0 Optional[string] `json:"row0,omitempty"`
	Row1 Optional[string] `json:"row1,omitempty"`
	Row2 Optional[string] `json:"row2,omitempty"`
	Row3 Optional[string] `json:"row3,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesAODecalsDecal struct {
	Transform Optional[StrictTankConfigurationChassisEntryPropertiesAODecalsDecalTransform] `json:"transform,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesAODecals struct {
	Decal Optional[StrictTankConfigurationChassisEntryPropertiesAODecalsDecal] `json:"decal,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesArmorArmorValue struct {
	Text                    Optional[string] `json:"#text,omitempty"`
	ChanceToHitByExplosion  Optional[string] `json:"chanceToHitByExplosion,omitempty"`
	ChanceToHitByProjectile Optional[string] `json:"chanceToHitByProjectile,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesEffects struct {
	Dust    Optional[string] `json:"dust,omitempty"`
	LodDist Optional[string] `json:"lodDist,omitempty"`
	Mud     Optional[string] `json:"mud,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesGroundNodesGroupEntry struct {
	Count      Optional[string] `json:"count,omitempty"`
	IsLeft     Optional[string] `json:"isLeft,omitempty"`
	MaxOffset  Optional[string] `json:"maxOffset,omitempty"`
	MinOffset  Optional[string] `json:"minOffset,omitempty"`
	StartIndex Optional[string] `json:"startIndex,omitempty"`
	Template   Optional[string] `json:"template,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesGroundNodes struct {
	Group Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesGroundNodesGroupEntry]] `json:"group,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesHitTester struct {
	CollisionModel Optional[string] `json:"collisionModel,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesInternalTrackLimitationEntry struct {
	BackwardSpeedLimit Optional[string] `json:"backwardSpeedLimit,omitempty"`
	ForwardSpeedLimit  Optional[string] `json:"forwardSpeedLimit,omitempty"`
	Track              Optional[string] `json:"track,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesInternalTrack struct {
	Limitation Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesInternalTrackLimitationEntry]] `json:"limitation,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesModels struct {
	Destroyed Optional[string] `json:"destroyed,omitempty"`
	Exploded  Optional[string] `json:"exploded,omitempty"`
	Undamaged Optional[string] `json:"undamaged,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesShotDispersionFactors struct {
	VehicleMovement Optional[string] `json:"vehicleMovement,omitempty"`
	VehicleRotation Optional[string] `json:"vehicleRotation,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesSplineDescAtlas struct {
	UTiles Optional[string] `json:"UTiles,omitempty"`
	VTiles Optional[string] `json:"VTiles,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesSplineDesc struct {
	Atlas              Optional[StrictTankConfigurationChassisEntryPropertiesSplineDescAtlas] `json:"atlas,omitempty"`
	Left               Optional[string]                                                       `json:"left,omitempty"`
	LodDist            Optional[string]                                                       `json:"lodDist,omitempty"`
	Right              Optional[string]                                                       `json:"right,omitempty"`
	Segment2ModelLeft  Optional[string]                                                       `json:"segment2ModelLeft,omitempty"`
	Segment2ModelRight Optional[string]                                                       `json:"segment2ModelRight,omitempty"`
	Segment2Offset     Optional[string]                                                       `json:"segment2Offset,omitempty"`
	SegmentLength      Optional[string]                                                       `json:"segmentLength,omitempty"`
	SegmentModelLeft   Optional[string]                                                       `json:"segmentModelLeft,omitempty"`
	SegmentModelRight  Optional[string]                                                       `json:"segmentModelRight,omitempty"`
	SegmentOffset      Optional[string]                                                       `json:"segmentOffset,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesSplineDescEntryAtlas struct {
	UTiles Optional[string] `json:"UTiles,omitempty"`
	VTiles Optional[string] `json:"VTiles,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesSplineDescEntry struct {
	Atlas             Optional[StrictTankConfigurationChassisEntryPropertiesSplineDescEntryAtlas] `json:"atlas,omitempty"`
	Left              Optional[string]                                                            `json:"left,omitempty"`
	LodDist           Optional[string]                                                            `json:"lodDist,omitempty"`
	Right             Optional[string]                                                            `json:"right,omitempty"`
	SegmentLength     Optional[string]                                                            `json:"segmentLength,omitempty"`
	SegmentModelLeft  Optional[string]                                                            `json:"segmentModelLeft,omitempty"`
	SegmentModelRight Optional[string]                                                            `json:"segmentModelRight,omitempty"`
	SegmentOffset     Optional[string]                                                            `json:"segmentOffset,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesTraces struct {
	BufferPrefs  Optional[string] `json:"bufferPrefs,omitempty"`
	DecalGroup   Optional[string] `json:"decalGroup,omitempty"`
	DecalTexture Optional[string] `json:"decalTexture,omitempty"`
	LodDist      Optional[string] `json:"lodDist,omitempty"`
	Size         Optional[string] `json:"size,omitempty"`
	TextureSet   Optional[string] `json:"textureSet,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesTrackNodesNodeEntry struct {
	BackwardElastK Optional[string] `json:"backwardElastK,omitempty"`
	Damping        Optional[string] `json:"damping,omitempty"`
	Elasticity     Optional[string] `json:"elasticity,omitempty"`
	ForwardElastK  Optional[string] `json:"forwardElastK,omitempty"`
	IsLeft         Optional[string] `json:"isLeft,omitempty"`
	LeftSibling    Optional[string] `json:"leftSibling,omitempty"`
	Name           Optional[string] `json:"name,omitempty"`
	Offset         Optional[string] `json:"offset,omitempty"`
	RightSibling   Optional[string] `json:"rightSibling,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesTrackNodes struct {
	BackwardElastK Optional[string]                                                                       `json:"backwardElastK,omitempty"`
	Damping        Optional[string]                                                                       `json:"damping,omitempty"`
	Elasticity     Optional[string]                                                                       `json:"elasticity,omitempty"`
	Enable         Optional[string]                                                                       `json:"enable,omitempty"`
	ForwardElastK  Optional[string]                                                                       `json:"forwardElastK,omitempty"`
	Gravity        Optional[string]                                                                       `json:"gravity,omitempty"`
	LinkBones      Optional[string]                                                                       `json:"linkBones,omitempty"`
	MaxAmplitude   Optional[string]                                                                       `json:"maxAmplitude,omitempty"`
	MaxOffset      Optional[string]                                                                       `json:"maxOffset,omitempty"`
	Node           Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesTrackNodesNodeEntry]] `json:"node,omitempty"`
	Offset         Optional[string]                                                                       `json:"offset,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesTrackNodesEntryNodeEntry struct {
	BackwardElastK Optional[string] `json:"backwardElastK,omitempty"`
	ForwardElastK  Optional[string] `json:"forwardElastK,omitempty"`
	IsLeft         Optional[string] `json:"isLeft,omitempty"`
	Name           Optional[string] `json:"name,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesTrackNodesEntry struct {
	BackwardElastK Optional[string]                                                                            `json:"backwardElastK,omitempty"`
	Damping        Optional[string]                                                                            `json:"damping,omitempty"`
	Elasticity     Optional[string]                                                                            `json:"elasticity,omitempty"`
	Enable         Optional[string]                                                                            `json:"enable,omitempty"`
	ForwardElastK  Optional[string]                                                                            `json:"forwardElastK,omitempty"`
	Gravity        Optional[string]                                                                            `json:"gravity,omitempty"`
	LinkBones      Optional[string]                                                                            `json:"linkBones,omitempty"`
	MaxAmplitude   Optional[string]                                                                            `json:"maxAmplitude,omitempty"`
	MaxOffset      Optional[string]                                                                            `json:"maxOffset,omitempty"`
	Node           Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesTrackNodesEntryNodeEntry]] `json:"node,omitempty"`
	Offset         Optional[string]                                                                            `json:"offset,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesTracks struct {
	LeftMaterial  Optional[string] `json:"leftMaterial,omitempty"`
	LodDist       Optional[string] `json:"lodDist,omitempty"`
	RightMaterial Optional[string] `json:"rightMaterial,omitempty"`
	TextureScale  Optional[string] `json:"textureScale,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocksChassisItemsEntry struct {
	Name       Optional[string] `json:"name,omitempty"`
	Properties Optional[string] `json:"properties,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocksChassis struct {
	Items Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesUnlocksChassisItemsEntry]] `json:"#items,omitempty"`
	Text  Optional[string]                                                                            `json:"#text,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocksGun struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocksTurret struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocksVehicle struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocks struct {
	Chassis Optional[StrictTankConfigurationChassisEntryPropertiesUnlocksChassis] `json:"chassis,omitempty"`
	Gun     Optional[StrictTankConfigurationChassisEntryPropertiesUnlocksGun]     `json:"gun,omitempty"`
	Turret  Optional[StrictTankConfigurationChassisEntryPropertiesUnlocksTurret]  `json:"turret,omitempty"`
	Vehicle Optional[StrictTankConfigurationChassisEntryPropertiesUnlocksVehicle] `json:"vehicle,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesWheelsGroup struct {
	Count      Optional[string] `json:"count,omitempty"`
	IsLeft     Optional[string] `json:"isLeft,omitempty"`
	Radius     Optional[string] `json:"radius,omitempty"`
	StartIndex Optional[string] `json:"startIndex,omitempty"`
	Template   Optional[string] `json:"template,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesWheelsGroupEntry struct {
	Count      Optional[string] `json:"count,omitempty"`
	IsLeft     Optional[string] `json:"isLeft,omitempty"`
	Radius     Optional[string] `json:"radius,omitempty"`
	StartIndex Optional[string] `json:"startIndex,omitempty"`
	Template   Optional[string] `json:"template,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesWheelsWheelEntry struct {
	IsLeading Optional[string] `json:"isLeading,omitempty"`
	IsLeft    Optional[string] `json:"isLeft,omitempty"`
	Name      Optional[string] `json:"name,omitempty"`
	Radius    Optional[string] `json:"radius,omitempty"`
	SyncAngle Optional[string] `json:"syncAngle,omitempty"`
}

type StrictTankConfigurationChassisEntryPropertiesWheels struct {
	Group                 Optional[XMLObjectList[StrictTankConfigurationChassisEntryPropertiesWheelsGroup, StrictTankConfigurationChassisEntryPropertiesWheelsGroupEntry]] `json:"group,omitempty"`
	LeadingWheelSyncAngle Optional[string]                                                                                                                                 `json:"leadingWheelSyncAngle,omitempty"`
	LodDist               Optional[string]                                                                                                                                 `json:"lodDist,omitempty"`
	Wheel                 Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesWheelsWheelEntry]]                                                              `json:"wheel,omitempty"`
}

type StrictTankConfigurationChassisEntryProperties struct {
	AODecals               Optional[StrictTankConfigurationChassisEntryPropertiesAODecals]                                                                                `json:"AODecals,omitempty"`
	Armor                  Optional[map[string]Optional[XMLScalarObject[StrictTankConfigurationChassisEntryPropertiesArmorArmorValue]]]                                   `json:"armor,omitempty"`
	BrakeForce             Optional[string]                                                                                                                               `json:"brakeForce,omitempty"`
	BulkHealthFactor       Optional[string]                                                                                                                               `json:"bulkHealthFactor,omitempty"`
	Description            Optional[string]                                                                                                                               `json:"description,omitempty"`
	DrivingWheels          Optional[XMLScalarList[string]]                                                                                                                `json:"drivingWheels,omitempty"`
	Effects                Optional[StrictTankConfigurationChassisEntryPropertiesEffects]                                                                                 `json:"effects,omitempty"`
	FitsRatingBattle       Optional[string]                                                                                                                               `json:"fitsRatingBattle,omitempty"`
	GroundNodes            Optional[StrictTankConfigurationChassisEntryPropertiesGroundNodes]                                                                             `json:"groundNodes,omitempty"`
	HitTester              Optional[StrictTankConfigurationChassisEntryPropertiesHitTester]                                                                               `json:"hitTester,omitempty"`
	HullPosition           Optional[string]                                                                                                                               `json:"hullPosition,omitempty"`
	Icon                   Optional[string]                                                                                                                               `json:"icon,omitempty"`
	InternalTrack          Optional[StrictTankConfigurationChassisEntryPropertiesInternalTrack]                                                                           `json:"internalTrack,omitempty"`
	Level                  Optional[string]                                                                                                                               `json:"level,omitempty"`
	MaxClimbAngle          Optional[string]                                                                                                                               `json:"maxClimbAngle,omitempty"`
	MaxHealth              Optional[XMLScalarList[string]]                                                                                                                `json:"maxHealth,omitempty"`
	MaxLoad                Optional[string]                                                                                                                               `json:"maxLoad,omitempty"`
	MaxRegenHealth         Optional[XMLScalarList[string]]                                                                                                                `json:"maxRegenHealth,omitempty"`
	Models                 Optional[StrictTankConfigurationChassisEntryPropertiesModels]                                                                                  `json:"models,omitempty"`
	NavmeshGirth           Optional[string]                                                                                                                               `json:"navmeshGirth,omitempty"`
	NotInShop              Optional[string]                                                                                                                               `json:"notInShop,omitempty"`
	Price                  Optional[string]                                                                                                                               `json:"price,omitempty"`
	RepairCost             Optional[string]                                                                                                                               `json:"repairCost,omitempty"`
	RotationIsAroundCenter Optional[string]                                                                                                                               `json:"rotationIsAroundCenter,omitempty"`
	RotationSpeed          Optional[string]                                                                                                                               `json:"rotationSpeed,omitempty"`
	ShortUserString        Optional[string]                                                                                                                               `json:"shortUserString,omitempty"`
	ShotDispersionFactors  Optional[StrictTankConfigurationChassisEntryPropertiesShotDispersionFactors]                                                                   `json:"shotDispersionFactors,omitempty"`
	Sound                  Optional[string]                                                                                                                               `json:"sound,omitempty"`
	SoundNPC               Optional[string]                                                                                                                               `json:"soundNPC,omitempty"`
	SoundPC                Optional[string]                                                                                                                               `json:"soundPC,omitempty"`
	SplineDesc             Optional[XMLObjectList[StrictTankConfigurationChassisEntryPropertiesSplineDesc, StrictTankConfigurationChassisEntryPropertiesSplineDescEntry]] `json:"splineDesc,omitempty"`
	Tags                   Optional[string]                                                                                                                               `json:"tags,omitempty"`
	TerrainResistance      Optional[string]                                                                                                                               `json:"terrainResistance,omitempty"`
	TopRightCarryingPoint  Optional[XMLScalarList[string]]                                                                                                                `json:"topRightCarryingPoint,omitempty"`
	Traces                 Optional[StrictTankConfigurationChassisEntryPropertiesTraces]                                                                                  `json:"traces,omitempty"`
	TrackNodes             Optional[XMLObjectList[StrictTankConfigurationChassisEntryPropertiesTrackNodes, StrictTankConfigurationChassisEntryPropertiesTrackNodesEntry]] `json:"trackNodes,omitempty"`
	TrackThickness         Optional[XMLScalarList[string]]                                                                                                                `json:"trackThickness,omitempty"`
	Tracks                 Optional[StrictTankConfigurationChassisEntryPropertiesTracks]                                                                                  `json:"tracks,omitempty"`
	Unlocks                Optional[StrictTankConfigurationChassisEntryPropertiesUnlocks]                                                                                 `json:"unlocks,omitempty"`
	UserString             Optional[string]                                                                                                                               `json:"userString,omitempty"`
	Weight                 Optional[string]                                                                                                                               `json:"weight,omitempty"`
	Wheels                 Optional[StrictTankConfigurationChassisEntryPropertiesWheels]                                                                                  `json:"wheels,omitempty"`
	WwsoundNPC             Optional[string]                                                                                                                               `json:"wwsoundNPC,omitempty"`
	WwsoundPC              Optional[string]                                                                                                                               `json:"wwsoundPC,omitempty"`
}

type StrictTankConfigurationChassisEntry struct {
	Name       Optional[string]                                        `json:"name,omitempty"`
	Properties Optional[StrictTankConfigurationChassisEntryProperties] `json:"properties,omitempty"`
}

type StrictTankConfigurationCrew struct {
	Commander Optional[string]                `json:"commander,omitempty"`
	Driver    Optional[string]                `json:"driver,omitempty"`
	Gunner    Optional[XMLScalarList[string]] `json:"gunner,omitempty"`
	Loader    Optional[XMLScalarList[string]] `json:"loader,omitempty"`
	Radioman  Optional[[]Optional[string]]    `json:"radioman,omitempty"`
}

type StrictTankConfigurationEffects struct {
	Collision         Optional[string] `json:"collision,omitempty"`
	DamagedStateGroup Optional[string] `json:"damagedStateGroup,omitempty"`
	Destruction       Optional[string] `json:"destruction,omitempty"`
	Explosion         Optional[string] `json:"explosion,omitempty"`
	Flaming           Optional[string] `json:"flaming,omitempty"`
}

type StrictTankConfigurationEmblems struct {
	Alpha   Optional[string] `json:"alpha,omitempty"`
	Default Optional[string] `json:"default,omitempty"`
	LodDist Optional[string] `json:"lodDist,omitempty"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocksChassisItemsEntry struct {
	Name       Optional[string] `json:"name,omitempty"`
	Properties Optional[string] `json:"properties,omitempty"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocksChassis struct {
	Items Optional[[]Optional[StrictTankConfigurationEnginesEntryPropertiesUnlocksChassisItemsEntry]] `json:"#items,omitempty"`
	Text  Optional[string]                                                                            `json:"#text,omitempty"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocksEngine struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocksVehicle struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocksVehicleEntry struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocks struct {
	Chassis Optional[StrictTankConfigurationEnginesEntryPropertiesUnlocksChassis]                                                                                  `json:"chassis,omitempty"`
	Engine  Optional[StrictTankConfigurationEnginesEntryPropertiesUnlocksEngine]                                                                                   `json:"engine,omitempty"`
	Vehicle Optional[XMLObjectList[StrictTankConfigurationEnginesEntryPropertiesUnlocksVehicle, StrictTankConfigurationEnginesEntryPropertiesUnlocksVehicleEntry]] `json:"vehicle,omitempty"`
}

type StrictTankConfigurationEnginesEntryProperties struct {
	Text             Optional[string]                                               `json:"#text,omitempty"`
	FitsRatingBattle Optional[string]                                               `json:"fitsRatingBattle,omitempty"`
	Unlocks          Optional[StrictTankConfigurationEnginesEntryPropertiesUnlocks] `json:"unlocks,omitempty"`
}

type StrictTankConfigurationEnginesEntry struct {
	Name       Optional[string]                                                         `json:"name,omitempty"`
	Properties Optional[XMLScalarObject[StrictTankConfigurationEnginesEntryProperties]] `json:"properties,omitempty"`
}

type StrictTankConfigurationExtrasAlteredSpotDuration struct {
	SpotDurationTimeChange Optional[string] `json:"spotDurationTimeChange,omitempty"`
	WithSpotEyeEffect      Optional[string] `json:"withSpotEyeEffect,omitempty"`
}

type StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesControllerStateEntryFactorsModifiers struct {
	EnginePowerFactor         Optional[string] `json:"enginePowerFactor,omitempty"`
	SpeedLimitBkwdFactor      Optional[string] `json:"speedLimitBkwdFactor,omitempty"`
	SpeedLimitFwdFactor       Optional[string] `json:"speedLimitFwdFactor,omitempty"`
	TurretRotationSpeedFactor Optional[string] `json:"turretRotationSpeedFactor,omitempty"`
}

type StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesControllerStateEntry struct {
	Armors           Optional[string]                                                                                          `json:"armors,omitempty"`
	Default          Optional[string]                                                                                          `json:"default,omitempty"`
	Enabled          Optional[string]                                                                                          `json:"enabled,omitempty"`
	FactorsModifiers Optional[StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesControllerStateEntryFactorsModifiers] `json:"factorsModifiers,omitempty"`
	Id               Optional[string]                                                                                          `json:"id,omitempty"`
	Type             Optional[string]                                                                                          `json:"type,omitempty"`
}

type StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesController struct {
	State Optional[[]Optional[StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesControllerStateEntry]] `json:"state,omitempty"`
}

type StrictTankConfigurationExtrasArmorMovingAbility struct {
	ArmorsStatesController Optional[StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesController] `json:"armorsStatesController,omitempty"`
	WrappedAbilityName     Optional[string]                                                                `json:"wrappedAbilityName,omitempty"`
}

type StrictTankConfigurationExtrasArmorsStatesControllerStateEntryActivateConditionHealthActivateCondition struct {
	Text   Optional[string] `json:"#text,omitempty"`
	AttrOp Optional[string] `json:"@op,omitempty"`
}

type StrictTankConfigurationExtrasArmorsStatesControllerStateEntryActivateCondition struct {
	HealthActivateCondition Optional[StrictTankConfigurationExtrasArmorsStatesControllerStateEntryActivateConditionHealthActivateCondition] `json:"healthActivateCondition,omitempty"`
}

type StrictTankConfigurationExtrasArmorsStatesControllerStateEntryFactorsModifiers struct {
	ReceiveDamageFactor Optional[string] `json:"receiveDamageFactor,omitempty"`
}

type StrictTankConfigurationExtrasArmorsStatesControllerStateEntry struct {
	ActivateCondition Optional[StrictTankConfigurationExtrasArmorsStatesControllerStateEntryActivateCondition] `json:"activateCondition,omitempty"`
	Armors            Optional[string]                                                                         `json:"armors,omitempty"`
	Default           Optional[string]                                                                         `json:"default,omitempty"`
	Enabled           Optional[string]                                                                         `json:"enabled,omitempty"`
	FactorsModifiers  Optional[StrictTankConfigurationExtrasArmorsStatesControllerStateEntryFactorsModifiers]  `json:"factorsModifiers,omitempty"`
	Id                Optional[string]                                                                         `json:"id,omitempty"`
	Type              Optional[string]                                                                         `json:"type,omitempty"`
}

type StrictTankConfigurationExtrasArmorsStatesController struct {
	State Optional[[]Optional[StrictTankConfigurationExtrasArmorsStatesControllerStateEntry]] `json:"state,omitempty"`
}

type StrictTankConfigurationExtrasDynamicArmorWrapperOverrides struct {
	Cooldown   Optional[string] `json:"cooldown,omitempty"`
	Duration   Optional[string] `json:"duration,omitempty"`
	ShieldCoef Optional[string] `json:"shieldCoef,omitempty"`
}

type StrictTankConfigurationExtrasDynamicArmorWrapper struct {
	Overrides          Optional[StrictTankConfigurationExtrasDynamicArmorWrapperOverrides] `json:"overrides,omitempty"`
	WrappedAbilityName Optional[string]                                                    `json:"wrappedAbilityName,omitempty"`
}

type StrictTankConfigurationExtrasDynamicGunSwitcherWrapperOverrides struct {
	InnerCooldown Optional[string] `json:"innerCooldown,omitempty"`
}

type StrictTankConfigurationExtrasDynamicGunSwitcherWrapper struct {
	Overrides          Optional[StrictTankConfigurationExtrasDynamicGunSwitcherWrapperOverrides] `json:"overrides,omitempty"`
	WrappedAbilityName Optional[string]                                                          `json:"wrappedAbilityName,omitempty"`
}

type StrictTankConfigurationExtrasGunCooling struct {
	CoolingTime        Optional[string]                                       `json:"coolingTime,omitempty"`
	DamageMul          Optional[XMLScalarObject[map[string]Optional[string]]] `json:"damageMul,omitempty"`
	ReloadBooster      Optional[XMLScalarObject[map[string]Optional[string]]] `json:"reloadBooster,omitempty"`
	UncompletedPenalty Optional[string]                                       `json:"uncompletedPenalty,omitempty"`
}

type StrictTankConfigurationExtrasGunSwitcherWrapperOverrides struct {
	Cooldown      Optional[string] `json:"cooldown,omitempty"`
	Duration      Optional[string] `json:"duration,omitempty"`
	InnerCooldown Optional[string] `json:"innerCooldown,omitempty"`
}

type StrictTankConfigurationExtrasGunSwitcherWrapper struct {
	Overrides          Optional[StrictTankConfigurationExtrasGunSwitcherWrapperOverrides] `json:"overrides,omitempty"`
	WrappedAbilityName Optional[string]                                                   `json:"wrappedAbilityName,omitempty"`
}

type StrictTankConfigurationExtrasRocketBoosterWrapperOverridesEqtParams struct {
	Cooldown Optional[string] `json:"cooldown,omitempty"`
	Duration Optional[string] `json:"duration,omitempty"`
}

type StrictTankConfigurationExtrasRocketBoosterWrapperOverridesMultipliers struct {
	AngularSpeedMultiplier          Optional[string] `json:"angularSpeedMultiplier,omitempty"`
	BackwardSpeedMultiplier         Optional[string] `json:"backwardSpeedMultiplier,omitempty"`
	EnginePowerMultiplier           Optional[string] `json:"enginePowerMultiplier,omitempty"`
	ForwardSpeedMultiplier          Optional[string] `json:"forwardSpeedMultiplier,omitempty"`
	IncomingRammingDamageMultiplier Optional[string] `json:"incomingRammingDamageMultiplier,omitempty"`
	OutgoingRammingDamageMultiplier Optional[string] `json:"outgoingRammingDamageMultiplier,omitempty"`
}

type StrictTankConfigurationExtrasRocketBoosterWrapperOverridesStatics struct {
	FlatteningCoefficient                  Optional[string] `json:"flatteningCoefficient,omitempty"`
	IncomingDefaultRammingDamageMultiplier Optional[string] `json:"incomingDefaultRammingDamageMultiplier,omitempty"`
	OutgoingDefaultRammingDamageMultiplier Optional[string] `json:"outgoingDefaultRammingDamageMultiplier,omitempty"`
	StaticForce                            Optional[string] `json:"staticForce,omitempty"`
	StaticRestForce                        Optional[string] `json:"staticRestForce,omitempty"`
	WideningCoefficient                    Optional[string] `json:"wideningCoefficient,omitempty"`
}

type StrictTankConfigurationExtrasRocketBoosterWrapperOverrides struct {
	EqtParams   Optional[StrictTankConfigurationExtrasRocketBoosterWrapperOverridesEqtParams]   `json:"eqtParams,omitempty"`
	Multipliers Optional[StrictTankConfigurationExtrasRocketBoosterWrapperOverridesMultipliers] `json:"multipliers,omitempty"`
	Statics     Optional[StrictTankConfigurationExtrasRocketBoosterWrapperOverridesStatics]     `json:"statics,omitempty"`
}

type StrictTankConfigurationExtrasRocketBoosterWrapper struct {
	Overrides          Optional[StrictTankConfigurationExtrasRocketBoosterWrapperOverrides] `json:"overrides,omitempty"`
	WrappedAbilityName Optional[string]                                                     `json:"wrappedAbilityName,omitempty"`
}

type StrictTankConfigurationExtrasShotDispersionStabilizatorBackward struct {
	SpeedIncreaseFactor Optional[string] `json:"speedIncreaseFactor,omitempty"`
	SpeedLimitFactor    Optional[string] `json:"speedLimitFactor,omitempty"`
}

type StrictTankConfigurationExtrasShotDispersionStabilizatorForward struct {
	SpeedIncreaseFactor Optional[string] `json:"speedIncreaseFactor,omitempty"`
	SpeedLimitFactor    Optional[string] `json:"speedLimitFactor,omitempty"`
}

type StrictTankConfigurationExtrasShotDispersionStabilizatorRotation struct {
	SpeedIncreaseFactor Optional[string] `json:"speedIncreaseFactor,omitempty"`
	SpeedLimitFactor    Optional[string] `json:"speedLimitFactor,omitempty"`
}

type StrictTankConfigurationExtrasShotDispersionStabilizator struct {
	Backward Optional[StrictTankConfigurationExtrasShotDispersionStabilizatorBackward] `json:"backward,omitempty"`
	Forward  Optional[StrictTankConfigurationExtrasShotDispersionStabilizatorForward]  `json:"forward,omitempty"`
	Rotation Optional[StrictTankConfigurationExtrasShotDispersionStabilizatorRotation] `json:"rotation,omitempty"`
}

type StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectDesert struct {
	EnginePowerFactor          Optional[string] `json:"enginePowerFactor,omitempty"`
	SpeedLimitsBackwardFactor  Optional[string] `json:"speedLimitsBackwardFactor,omitempty"`
	SpeedLimitsForwardFactor   Optional[string] `json:"speedLimitsForwardFactor,omitempty"`
	VehicleRotationSpeedFactor Optional[string] `json:"vehicleRotationSpeedFactor,omitempty"`
}

type StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectSummer struct {
	ChassisMovementShotDispersionFactor Optional[string] `json:"chassisMovementShotDispersionFactor,omitempty"`
	ChassisRotationShotDispersionFactor Optional[string] `json:"chassisRotationShotDispersionFactor,omitempty"`
	GunAimingFactor                     Optional[string] `json:"gunAimingFactor,omitempty"`
	GunReloadFactor                     Optional[string] `json:"gunReloadFactor,omitempty"`
	GunTurretRotationDispersionFactor   Optional[string] `json:"gunTurretRotationDispersionFactor,omitempty"`
	ShotDispersionAngleFactor           Optional[string] `json:"shotDispersionAngleFactor,omitempty"`
}

type StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectWinter struct {
	ChassisHealthFactor           Optional[string] `json:"chassisHealthFactor,omitempty"`
	ChassisHysteresisHealthFactor Optional[string] `json:"chassisHysteresisHealthFactor,omitempty"`
	ChassisMaxRegenHealthFactor   Optional[string] `json:"chassisMaxRegenHealthFactor,omitempty"`
	ChassisRegenFactor            Optional[string] `json:"chassisRegenFactor,omitempty"`
	EngineHealthFactor            Optional[string] `json:"engineHealthFactor,omitempty"`
	EngineHysteresisHealthFactor  Optional[string] `json:"engineHysteresisHealthFactor,omitempty"`
	EngineMaxRegenHealthFactor    Optional[string] `json:"engineMaxRegenHealthFactor,omitempty"`
	EngineRegenFactor             Optional[string] `json:"engineRegenFactor,omitempty"`
	VehicleMaxHealthFactor        Optional[string] `json:"vehicleMaxHealthFactor,omitempty"`
}

type StrictTankConfigurationExtrasYearSeasonsEffectOverrides struct {
	YearSeasonsEffectDesert Optional[StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectDesert] `json:"yearSeasonsEffectDesert,omitempty"`
	YearSeasonsEffectSummer Optional[StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectSummer] `json:"yearSeasonsEffectSummer,omitempty"`
	YearSeasonsEffectWinter Optional[StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectWinter] `json:"yearSeasonsEffectWinter,omitempty"`
}

type StrictTankConfigurationExtrasYearSeasonsEffect struct {
	Overrides Optional[StrictTankConfigurationExtrasYearSeasonsEffectOverrides] `json:"overrides,omitempty"`
}

type StrictTankConfigurationExtras struct {
	AlteredSpotDuration        Optional[StrictTankConfigurationExtrasAlteredSpotDuration]        `json:"alteredSpotDuration,omitempty"`
	ArmorMovingAbility         Optional[StrictTankConfigurationExtrasArmorMovingAbility]         `json:"armorMovingAbility,omitempty"`
	ArmorsStatesController     Optional[StrictTankConfigurationExtrasArmorsStatesController]     `json:"armorsStatesController,omitempty"`
	DesertPower                Optional[string]                                                  `json:"desertPower,omitempty"`
	DynamicArmorWrapper        Optional[StrictTankConfigurationExtrasDynamicArmorWrapper]        `json:"dynamicArmorWrapper,omitempty"`
	DynamicGunSwitcherWrapper  Optional[StrictTankConfigurationExtrasDynamicGunSwitcherWrapper]  `json:"dynamicGunSwitcherWrapper,omitempty"`
	GunCooling                 Optional[StrictTankConfigurationExtrasGunCooling]                 `json:"gunCooling,omitempty"`
	GunSwitcherWrapper         Optional[StrictTankConfigurationExtrasGunSwitcherWrapper]         `json:"gunSwitcherWrapper,omitempty"`
	RocketBoosterWrapper       Optional[StrictTankConfigurationExtrasRocketBoosterWrapper]       `json:"rocketBoosterWrapper,omitempty"`
	ShotDispersionStabilizator Optional[StrictTankConfigurationExtrasShotDispersionStabilizator] `json:"shotDispersionStabilizator,omitempty"`
	YearSeasonsEffect          Optional[StrictTankConfigurationExtrasYearSeasonsEffect]          `json:"yearSeasonsEffect,omitempty"`
}

type StrictTankConfigurationFuelTanksEntry struct {
	Name       Optional[string] `json:"name,omitempty"`
	Properties Optional[string] `json:"properties,omitempty"`
}

type StrictTankConfigurationHorns struct {
	DistanceFactor Optional[string] `json:"distanceFactor,omitempty"`
	PriceFactor    Optional[string] `json:"priceFactor,omitempty"`
	VolumeFactor   Optional[string] `json:"volumeFactor,omitempty"`
}

type StrictTankConfigurationHullAmmoBayHealth struct {
	MaxHealth      Optional[string] `json:"maxHealth,omitempty"`
	MaxRegenHealth Optional[string] `json:"maxRegenHealth,omitempty"`
	RepairCost     Optional[string] `json:"repairCost,omitempty"`
}

type StrictTankConfigurationHullArmorArmorValue struct {
	Text                    Optional[string] `json:"#text,omitempty"`
	ChanceToHitByExplosion  Optional[string] `json:"chanceToHitByExplosion,omitempty"`
	ChanceToHitByProjectile Optional[string] `json:"chanceToHitByProjectile,omitempty"`
	VehicleDamageFactor     Optional[string] `json:"vehicleDamageFactor,omitempty"`
}

type StrictTankConfigurationHullBlitzBbox struct {
	Max Optional[string] `json:"max,omitempty"`
	Min Optional[string] `json:"min,omitempty"`
}

type StrictTankConfigurationHullBlitzPoints struct {
	Normal Optional[string] `json:"normal,omitempty"`
}

type StrictTankConfigurationHullBlitz struct {
	ArmorScale       Optional[string]                                 `json:"armorScale,omitempty"`
	AverageThickness Optional[map[string]Optional[string]]            `json:"averageThickness,omitempty"`
	Bbox             Optional[StrictTankConfigurationHullBlitzBbox]   `json:"bbox,omitempty"`
	BlitzPartName    Optional[string]                                 `json:"blitzPartName,omitempty"`
	Points           Optional[StrictTankConfigurationHullBlitzPoints] `json:"points,omitempty"`
	PushNormalsScale Optional[string]                                 `json:"pushNormalsScale,omitempty"`
}

type StrictTankConfigurationHullCamouflage struct {
	ExclusionMask Optional[string] `json:"exclusionMask,omitempty"`
	Tiling        Optional[string] `json:"tiling,omitempty"`
}

type StrictTankConfigurationHullEmblemSlotsClan struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationHullEmblemSlotsInscription struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationHullEmblemSlotsInscriptionEntry struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationHullEmblemSlotsPlayer struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationHullEmblemSlotsPlayerEntry struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	IsMirrored    Optional[string] `json:"isMirrored,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationHullEmblemSlots struct {
	Clan        Optional[StrictTankConfigurationHullEmblemSlotsClan]                                                                               `json:"clan,omitempty"`
	Inscription Optional[XMLObjectList[StrictTankConfigurationHullEmblemSlotsInscription, StrictTankConfigurationHullEmblemSlotsInscriptionEntry]] `json:"inscription,omitempty"`
	Player      Optional[XMLObjectList[StrictTankConfigurationHullEmblemSlotsPlayer, StrictTankConfigurationHullEmblemSlotsPlayerEntry]]           `json:"player,omitempty"`
}

type StrictTankConfigurationHullExhaustTags struct {
	Diesel   Optional[string] `json:"diesel,omitempty"`
	Gasoline Optional[string] `json:"gasoline,omitempty"`
}

type StrictTankConfigurationHullExhaust struct {
	Nodes Optional[XMLScalarList[string]]                  `json:"nodes,omitempty"`
	Pixie Optional[string]                                 `json:"pixie,omitempty"`
	Rates Optional[string]                                 `json:"rates,omitempty"`
	Tags  Optional[StrictTankConfigurationHullExhaustTags] `json:"tags,omitempty"`
}

type StrictTankConfigurationHullFakeTurrets struct {
	Battle Optional[string] `json:"battle,omitempty"`
	Lobby  Optional[string] `json:"lobby,omitempty"`
}

type StrictTankConfigurationHullHitTester struct {
	CollisionModel Optional[string] `json:"collisionModel,omitempty"`
}

type StrictTankConfigurationHullModels struct {
	Destroyed Optional[string] `json:"destroyed,omitempty"`
	Exploded  Optional[string] `json:"exploded,omitempty"`
	Undamaged Optional[string] `json:"undamaged,omitempty"`
}

type StrictTankConfigurationHullSwinging struct {
	LodDist              Optional[string] `json:"lodDist,omitempty"`
	PitchParams          Optional[string] `json:"pitchParams,omitempty"`
	RollParams           Optional[string] `json:"rollParams,omitempty"`
	SensitivityToImpulse Optional[string] `json:"sensitivityToImpulse,omitempty"`
	WaterSwinging        Optional[string] `json:"waterSwinging,omitempty"`
}

type StrictTankConfigurationHullTurretInitialRotation struct {
	Pitch Optional[string] `json:"pitch,omitempty"`
	Roll  Optional[string] `json:"roll,omitempty"`
	Yaw   Optional[string] `json:"yaw,omitempty"`
}

type StrictTankConfigurationHullTurretPositions struct {
	Turret Optional[string] `json:"turret,omitempty"`
}

type StrictTankConfigurationHull struct {
	AODecals              Optional[string]                                                                                       `json:"AODecals,omitempty"`
	AmmoBayHealth         Optional[StrictTankConfigurationHullAmmoBayHealth]                                                     `json:"ammoBayHealth,omitempty"`
	Armor                 Optional[map[string]Optional[XMLScalarObjectList[StrictTankConfigurationHullArmorArmorValue, string]]] `json:"armor,omitempty"`
	Blitz                 Optional[StrictTankConfigurationHullBlitz]                                                             `json:"blitz,omitempty"`
	Camouflage            Optional[StrictTankConfigurationHullCamouflage]                                                        `json:"camouflage,omitempty"`
	EmblemSlots           Optional[StrictTankConfigurationHullEmblemSlots]                                                       `json:"emblemSlots,omitempty"`
	Exhaust               Optional[StrictTankConfigurationHullExhaust]                                                           `json:"exhaust,omitempty"`
	FakeTurrets           Optional[StrictTankConfigurationHullFakeTurrets]                                                       `json:"fakeTurrets,omitempty"`
	HangarShadowTexture   Optional[string]                                                                                       `json:"hangarShadowTexture,omitempty"`
	HitTester             Optional[StrictTankConfigurationHullHitTester]                                                         `json:"hitTester,omitempty"`
	MaxHealth             Optional[string]                                                                                       `json:"maxHealth,omitempty"`
	Models                Optional[StrictTankConfigurationHullModels]                                                            `json:"models,omitempty"`
	PrimaryArmor          Optional[string]                                                                                       `json:"primaryArmor,omitempty"`
	Swinging              Optional[StrictTankConfigurationHullSwinging]                                                          `json:"swinging,omitempty"`
	TurretInitialRotation Optional[StrictTankConfigurationHullTurretInitialRotation]                                             `json:"turretInitialRotation,omitempty"`
	TurretPositions       Optional[StrictTankConfigurationHullTurretPositions]                                                   `json:"turretPositions,omitempty"`
	Weight                Optional[string]                                                                                       `json:"weight,omitempty"`
}

type StrictTankConfigurationInvisibility struct {
	CamouflageNetBonus Optional[string] `json:"camouflageNetBonus,omitempty"`
	FirePenalty        Optional[string] `json:"firePenalty,omitempty"`
	Moving             Optional[string] `json:"moving,omitempty"`
	Still              Optional[string] `json:"still,omitempty"`
}

type StrictTankConfigurationRadiosEntryProperties struct {
	Text    Optional[string] `json:"#text,omitempty"`
	Unlocks Optional[string] `json:"unlocks,omitempty"`
}

type StrictTankConfigurationRadiosEntry struct {
	Name       Optional[string]                                                        `json:"name,omitempty"`
	Properties Optional[XMLScalarObject[StrictTankConfigurationRadiosEntryProperties]] `json:"properties,omitempty"`
}

type StrictTankConfigurationSpeedLimits struct {
	Backward Optional[string] `json:"backward,omitempty"`
	Forward  Optional[string] `json:"forward,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesArmorArmorValue struct {
	Text                    Optional[string] `json:"#text,omitempty"`
	ChanceToHitByExplosion  Optional[string] `json:"chanceToHitByExplosion,omitempty"`
	ChanceToHitByProjectile Optional[string] `json:"chanceToHitByProjectile,omitempty"`
	VehicleDamageFactor     Optional[string] `json:"vehicleDamageFactor,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesCamouflage struct {
	ExclusionMask Optional[string] `json:"exclusionMask,omitempty"`
	Tiling        Optional[string] `json:"tiling,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsClan struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsInscription struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsInscriptionEntry struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsPlayer struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsPlayerEntry struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	IsMirrored    Optional[string] `json:"isMirrored,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlots struct {
	Clan        Optional[StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsClan]                                                                                                  `json:"clan,omitempty"`
	Inscription Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsInscription, StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsInscriptionEntry]] `json:"inscription,omitempty"`
	Player      Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsPlayer, StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsPlayerEntry]]           `json:"player,omitempty"`
}

type StrictTankConfigurationNamedShell struct {
	Text  Optional[string] `json:"#text,omitempty"`
	Price Optional[string] `json:"price,omitempty"`
}

type StrictTankConfigurationNamedShot struct {
	Shell Optional[StrictTankConfigurationNamedShell] `json:"shell,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesArmorArmorValue struct {
	Text                    Optional[string] `json:"#text,omitempty"`
	ChanceToHitByExplosion  Optional[string] `json:"chanceToHitByExplosion,omitempty"`
	ChanceToHitByProjectile Optional[string] `json:"chanceToHitByProjectile,omitempty"`
	VehicleDamageFactor     Optional[string] `json:"vehicleDamageFactor,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesBurst struct {
	Count Optional[string] `json:"count,omitempty"`
	Rate  Optional[string] `json:"rate,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesCamouflage struct {
	ExclusionMask Optional[string] `json:"exclusionMask,omitempty"`
	Tiling        Optional[string] `json:"tiling,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesClip struct {
	Count Optional[string] `json:"count,omitempty"`
	Rate  Optional[string] `json:"rate,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesEmblemSlotsInsigniaOnGun struct {
	HideIfDamaged Optional[string] `json:"hideIfDamaged,omitempty"`
	RayEnd        Optional[string] `json:"rayEnd,omitempty"`
	RayStart      Optional[string] `json:"rayStart,omitempty"`
	RayUp         Optional[string] `json:"rayUp,omitempty"`
	Size          Optional[string] `json:"size,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesEmblemSlots struct {
	InsigniaOnGun Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesEmblemSlotsInsigniaOnGun] `json:"insigniaOnGun,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtraPitchLimits struct {
	Text       Optional[string] `json:"#text,omitempty"`
	Back       Optional[string] `json:"back,omitempty"`
	Front      Optional[string] `json:"front,omitempty"`
	Transition Optional[string] `json:"transition,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtrasImprovedDetectionShellEntry struct {
	Duration Optional[string] `json:"duration,omitempty"`
	Kind     Optional[string] `json:"kind,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtrasImprovedDetection struct {
	Shell Optional[[]Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtrasImprovedDetectionShellEntry]] `json:"shell,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtras struct {
	ImprovedDetection Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtrasImprovedDetection] `json:"improvedDetection,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesFakeTurrets struct {
	Battle Optional[string] `json:"battle,omitempty"`
	Lobby  Optional[string] `json:"lobby,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunBarrel struct {
	Controller Optional[string] `json:"controller,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunFeeder struct {
	Controller Optional[string] `json:"controller,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunLoader struct {
	Controller Optional[string] `json:"controller,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0Overrides struct {
	AimingTimeFactor         Optional[string] `json:"aimingTimeFactor,omitempty"`
	BurstSize                Optional[string] `json:"burstSize,omitempty"`
	Capacity                 Optional[string] `json:"capacity,omitempty"`
	LockGunTime              Optional[string] `json:"lockGunTime,omitempty"`
	MovementDispersion       Optional[string] `json:"movementDispersion,omitempty"`
	ReloadTimeFactor         Optional[string] `json:"reloadTimeFactor,omitempty"`
	ShotDispersion           Optional[string] `json:"shotDispersion,omitempty"`
	SwitchCooldown           Optional[string] `json:"switchCooldown,omitempty"`
	TurretRotationDispersion Optional[string] `json:"turretRotationDispersion,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0 struct {
	GunBarrel Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunBarrel] `json:"gunBarrel,omitempty"`
	GunFeeder Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunFeeder] `json:"gunFeeder,omitempty"`
	GunLoader Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunLoader] `json:"gunLoader,omitempty"`
	Overrides Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0Overrides] `json:"overrides,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunBarrel struct {
	Controller Optional[string] `json:"controller,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunFeeder struct {
	Controller Optional[string] `json:"controller,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunLoader struct {
	Controller Optional[string] `json:"controller,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1Overrides struct {
	AimingTimeFactor         Optional[string] `json:"aimingTimeFactor,omitempty"`
	BuckshotSize             Optional[string] `json:"buckshotSize,omitempty"`
	Capacity                 Optional[string] `json:"capacity,omitempty"`
	LockGunTime              Optional[string] `json:"lockGunTime,omitempty"`
	MovementDispersion       Optional[string] `json:"movementDispersion,omitempty"`
	ReloadTimeFactor         Optional[string] `json:"reloadTimeFactor,omitempty"`
	ShotDispersion           Optional[string] `json:"shotDispersion,omitempty"`
	SwitchCooldown           Optional[string] `json:"switchCooldown,omitempty"`
	TurretRotationDispersion Optional[string] `json:"turretRotationDispersion,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1 struct {
	GunBarrel Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunBarrel] `json:"gunBarrel,omitempty"`
	GunFeeder Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunFeeder] `json:"gunFeeder,omitempty"`
	GunLoader Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunLoader] `json:"gunLoader,omitempty"`
	Overrides Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1Overrides] `json:"overrides,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcher struct {
	Setting0 Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0] `json:"setting0,omitempty"`
	Setting1 Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1] `json:"setting1,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesHitTester struct {
	CollisionModel Optional[string] `json:"collisionModel,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesModels struct {
	Destroyed Optional[string] `json:"destroyed,omitempty"`
	Exploded  Optional[string] `json:"exploded,omitempty"`
	Undamaged Optional[string] `json:"undamaged,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesRecoil struct {
	Amplitude   Optional[string] `json:"amplitude,omitempty"`
	BackoffTime Optional[string] `json:"backoffTime,omitempty"`
	LodDist     Optional[string] `json:"lodDist,omitempty"`
	ReturnTime  Optional[string] `json:"returnTime,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotDispersionFactors struct {
	AfterShot       Optional[string] `json:"afterShot,omitempty"`
	TurretRotation  Optional[string] `json:"turretRotation,omitempty"`
	WhileGunDamaged Optional[string] `json:"whileGunDamaged,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryPropertiesShellPrice struct {
	Text Optional[string] `json:"#text,omitempty"`
	Gold Optional[string] `json:"gold,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryPropertiesShell struct {
	Text  Optional[string]                                                                                                           `json:"#text,omitempty"`
	Price Optional[XMLScalarObject[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryPropertiesShellPrice]] `json:"price,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryProperties struct {
	Shell Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryPropertiesShell] `json:"shell,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntry struct {
	Name       Optional[string]                                                                                `json:"name,omitempty"`
	Properties Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryProperties] `json:"properties,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksGun struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksGunEntry struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksTurret struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksVehicle struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksVehicleEntry struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocks struct {
	Gun     Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksGun, StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksGunEntry]]         `json:"gun,omitempty"`
	Turret  Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksTurret]                                                                                                       `json:"turret,omitempty"`
	Vehicle Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksVehicle, StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksVehicleEntry]] `json:"vehicle,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryProperties struct {
	Text                     Optional[string]                                                                                                                 `json:"#text,omitempty"`
	NamedShots               map[string]Optional[StrictTankConfigurationNamedShot]                                                                            `json:"-"`
	AimingTime               Optional[string]                                                                                                                 `json:"aimingTime,omitempty"`
	AnimateEmblemSlots       Optional[string]                                                                                                                 `json:"animateEmblemSlots,omitempty"`
	Armor                    Optional[map[string]Optional[XMLScalarObject[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesArmorArmorValue]]] `json:"armor,omitempty"`
	Burst                    Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesBurst]                                                 `json:"burst,omitempty"`
	Camouflage               Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesCamouflage]                                            `json:"camouflage,omitempty"`
	Clip                     Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesClip]                                                  `json:"clip,omitempty"`
	Effects                  Optional[string]                                                                                                                 `json:"effects,omitempty"`
	EmblemSlots              Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesEmblemSlots]                                           `json:"emblemSlots,omitempty"`
	ExtraPitchLimits         Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtraPitchLimits]                                      `json:"extraPitchLimits,omitempty"`
	Extras                   Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtras]                                                `json:"extras,omitempty"`
	FakeTurrets              Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesFakeTurrets]                                           `json:"fakeTurrets,omitempty"`
	FitsRatingBattle         Optional[string]                                                                                                                 `json:"fitsRatingBattle,omitempty"`
	GunSwitcher              Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcher]                                           `json:"gunSwitcher,omitempty"`
	HitTester                Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesHitTester]                                             `json:"hitTester,omitempty"`
	InvisibilityFactorAtShot Optional[XMLScalarList[string]]                                                                                                  `json:"invisibilityFactorAtShot,omitempty"`
	MaxAmmo                  Optional[XMLScalarList[string]]                                                                                                  `json:"maxAmmo,omitempty"`
	Models                   Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesModels]                                                `json:"models,omitempty"`
	PitchLimits              Optional[XMLScalarList[string]]                                                                                                  `json:"pitchLimits,omitempty"`
	ProjectileSpawnShift     Optional[map[string]Optional[string]]                                                                                            `json:"projectileSpawnShift,omitempty"`
	PumpGunMode              Optional[string]                                                                                                                 `json:"pumpGunMode,omitempty"`
	PumpGunReloadTimes       Optional[string]                                                                                                                 `json:"pumpGunReloadTimes,omitempty"`
	Recoil                   Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesRecoil]                                                `json:"recoil,omitempty"`
	ReloadEffect             Optional[string]                                                                                                                 `json:"reloadEffect,omitempty"`
	ReloadTime               Optional[string]                                                                                                                 `json:"reloadTime,omitempty"`
	ShotDispersionFactors    Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotDispersionFactors]                                 `json:"shotDispersionFactors,omitempty"`
	ShotDispersionRadius     Optional[string]                                                                                                                 `json:"shotDispersionRadius,omitempty"`
	Shots                    Optional[[]Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntry]]                                `json:"shots,omitempty"`
	TurretYawLimits          Optional[XMLScalarList[string]]                                                                                                  `json:"turretYawLimits,omitempty"`
	Unlocks                  Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocks]                                               `json:"unlocks,omitempty"`
	YawLimits                Optional[string]                                                                                                                 `json:"yawLimits,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntry struct {
	Name       Optional[string]                                                            `json:"name,omitempty"`
	Properties Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryProperties] `json:"properties,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesHitTester struct {
	CollisionModel Optional[string] `json:"collisionModel,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesModels struct {
	Destroyed Optional[string]                `json:"destroyed,omitempty"`
	Exploded  Optional[XMLScalarList[string]] `json:"exploded,omitempty"`
	Undamaged Optional[string]                `json:"undamaged,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesSurveyingDeviceHealth struct {
	MaxHealth      Optional[string] `json:"maxHealth,omitempty"`
	MaxRegenHealth Optional[string] `json:"maxRegenHealth,omitempty"`
	RepairCost     Optional[string] `json:"repairCost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesTurretRotatorHealth struct {
	MaxHealth      Optional[string] `json:"maxHealth,omitempty"`
	MaxRegenHealth Optional[string] `json:"maxRegenHealth,omitempty"`
	RepairCost     Optional[string] `json:"repairCost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocksEngine struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocksGun struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocksGunEntry struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocksTurret struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocksVehicle struct {
	Text Optional[string] `json:"#text,omitempty"`
	Cost Optional[string] `json:"cost,omitempty"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocks struct {
	Engine  Optional[StrictTankConfigurationTurrets0EntryPropertiesUnlocksEngine]                                                                            `json:"engine,omitempty"`
	Gun     Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesUnlocksGun, StrictTankConfigurationTurrets0EntryPropertiesUnlocksGunEntry]] `json:"gun,omitempty"`
	Turret  Optional[StrictTankConfigurationTurrets0EntryPropertiesUnlocksTurret]                                                                            `json:"turret,omitempty"`
	Vehicle Optional[StrictTankConfigurationTurrets0EntryPropertiesUnlocksVehicle]                                                                           `json:"vehicle,omitempty"`
}

type StrictTankConfigurationTurrets0EntryProperties struct {
	AODecals                   Optional[string]                                                                                                          `json:"AODecals,omitempty"`
	Armor                      Optional[map[string]Optional[XMLScalarObjectList[StrictTankConfigurationTurrets0EntryPropertiesArmorArmorValue, string]]] `json:"armor,omitempty"`
	Camouflage                 Optional[StrictTankConfigurationTurrets0EntryPropertiesCamouflage]                                                        `json:"camouflage,omitempty"`
	Ceilless                   Optional[string]                                                                                                          `json:"ceilless,omitempty"`
	CircularVisionRadius       Optional[string]                                                                                                          `json:"circularVisionRadius,omitempty"`
	Description                Optional[string]                                                                                                          `json:"description,omitempty"`
	EmblemSlots                Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesEmblemSlots, string]]                                `json:"emblemSlots,omitempty"`
	FitsRatingBattle           Optional[string]                                                                                                          `json:"fitsRatingBattle,omitempty"`
	GunPosition                Optional[XMLScalarList[string]]                                                                                           `json:"gunPosition,omitempty"`
	Guns                       Optional[[]Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntry]]                                             `json:"guns,omitempty"`
	HitTester                  Optional[StrictTankConfigurationTurrets0EntryPropertiesHitTester]                                                         `json:"hitTester,omitempty"`
	Icon                       Optional[string]                                                                                                          `json:"icon,omitempty"`
	InvisibilityFactor         Optional[string]                                                                                                          `json:"invisibilityFactor,omitempty"`
	Level                      Optional[string]                                                                                                          `json:"level,omitempty"`
	MaxHealth                  Optional[string]                                                                                                          `json:"maxHealth,omitempty"`
	Models                     Optional[StrictTankConfigurationTurrets0EntryPropertiesModels]                                                            `json:"models,omitempty"`
	NotInShop                  Optional[string]                                                                                                          `json:"notInShop,omitempty"`
	PhysicsShape               Optional[string]                                                                                                          `json:"physicsShape,omitempty"`
	Price                      Optional[string]                                                                                                          `json:"price,omitempty"`
	PrimaryArmor               Optional[string]                                                                                                          `json:"primaryArmor,omitempty"`
	RotationSpeed              Optional[XMLScalarList[string]]                                                                                           `json:"rotationSpeed,omitempty"`
	ShortUserString            Optional[string]                                                                                                          `json:"shortUserString,omitempty"`
	ShowEmblemsOnGun           Optional[string]                                                                                                          `json:"showEmblemsOnGun,omitempty"`
	SurveyingDeviceHealth      Optional[StrictTankConfigurationTurrets0EntryPropertiesSurveyingDeviceHealth]                                             `json:"surveyingDeviceHealth,omitempty"`
	Tags                       Optional[string]                                                                                                          `json:"tags,omitempty"`
	TurretRotatorHealth        Optional[StrictTankConfigurationTurrets0EntryPropertiesTurretRotatorHealth]                                               `json:"turretRotatorHealth,omitempty"`
	TurretRotatorSound         Optional[string]                                                                                                          `json:"turretRotatorSound,omitempty"`
	TurretRotatorSoundGear     Optional[string]                                                                                                          `json:"turretRotatorSoundGear,omitempty"`
	TurretRotatorSoundManual   Optional[string]                                                                                                          `json:"turretRotatorSoundManual,omitempty"`
	Unlocks                    Optional[StrictTankConfigurationTurrets0EntryPropertiesUnlocks]                                                           `json:"unlocks,omitempty"`
	UserString                 Optional[string]                                                                                                          `json:"userString,omitempty"`
	Weight                     Optional[string]                                                                                                          `json:"weight,omitempty"`
	WwturretRotatorSoundManual Optional[string]                                                                                                          `json:"wwturretRotatorSoundManual,omitempty"`
	YawLimits                  Optional[XMLScalarList[string]]                                                                                           `json:"yawLimits,omitempty"`
}

type StrictTankConfigurationTurrets0Entry struct {
	Name       Optional[string]                                         `json:"name,omitempty"`
	Properties Optional[StrictTankConfigurationTurrets0EntryProperties] `json:"properties,omitempty"`
}

type StrictTankConfiguration struct {
	AttrExcept             Optional[string]                                            `json:"@except,omitempty"`
	Camouflage             Optional[StrictTankConfigurationCamouflage]                 `json:"camouflage,omitempty"`
	Chassis                Optional[[]Optional[StrictTankConfigurationChassisEntry]]   `json:"chassis,omitempty"`
	ConsumableSlots        Optional[string]                                            `json:"consumableSlots,omitempty"`
	Crew                   Optional[StrictTankConfigurationCrew]                       `json:"crew,omitempty"`
	CrewXpFactor           Optional[string]                                            `json:"crewXpFactor,omitempty"`
	CustomizationNation    Optional[string]                                            `json:"customizationNation,omitempty"`
	Effects                Optional[StrictTankConfigurationEffects]                    `json:"effects,omitempty"`
	Emblems                Optional[StrictTankConfigurationEmblems]                    `json:"emblems,omitempty"`
	Engines                Optional[[]Optional[StrictTankConfigurationEnginesEntry]]   `json:"engines,omitempty"`
	EnrichmentBoost        Optional[string]                                            `json:"enrichmentBoost,omitempty"`
	Extras                 Optional[StrictTankConfigurationExtras]                     `json:"extras,omitempty"`
	FuelTanks              Optional[[]Optional[StrictTankConfigurationFuelTanksEntry]] `json:"fuelTanks,omitempty"`
	Horns                  Optional[StrictTankConfigurationHorns]                      `json:"horns,omitempty"`
	Hull                   Optional[StrictTankConfigurationHull]                       `json:"hull,omitempty"`
	Invisibility           Optional[StrictTankConfigurationInvisibility]               `json:"invisibility,omitempty"`
	IsUnlocked             Optional[string]                                            `json:"isUnlocked,omitempty"`
	MaxMastery             Optional[string]                                            `json:"maxMastery,omitempty"`
	OptDevicePreset        Optional[string]                                            `json:"optDevicePreset,omitempty"`
	PremiumVehicleXPFactor Optional[string]                                            `json:"premiumVehicleXPFactor,omitempty"`
	ProvisionSlots         Optional[string]                                            `json:"provisionSlots,omitempty"`
	Radios                 Optional[[]Optional[StrictTankConfigurationRadiosEntry]]    `json:"radios,omitempty"`
	RepairCost             Optional[string]                                            `json:"repairCost,omitempty"`
	SpeedLimits            Optional[StrictTankConfigurationSpeedLimits]                `json:"speedLimits,omitempty"`
	Subnation              Optional[string]                                            `json:"subnation,omitempty"`
	TurretBlowEnabled      Optional[string]                                            `json:"turretBlowEnabled,omitempty"`
	Turrets0               Optional[[]Optional[StrictTankConfigurationTurrets0Entry]]  `json:"turrets0,omitempty"`
}

// UnmarshalJSON validates the configuration's known fields and value shapes.
func (c *StrictTankConfiguration) UnmarshalJSON(data []byte) error {
	b := bytes.TrimSpace(data)
	if len(b) == 0 || b[0] != '{' {
		return fmt.Errorf("tank configuration must be an object")
	}
	if err := validateTankOrderedSelections(data); err != nil {
		return err
	}
	type fields StrictTankConfiguration
	var v fields
	if err := decodeStrictJSON(data, &v); err != nil {
		return err
	}
	*c = StrictTankConfiguration(v)
	return nil
}
