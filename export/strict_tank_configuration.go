// Tank configuration types based on observed v1 game resources.
// Fields reflect the current game resources, not inferred game semantics.
package export

type StrictTankConfigurationCamouflage struct {
	ExclusionMask Optional[string]          `json:"exclusionMask,omitzero"`
	PriceFactor   Optional[XMLNumber]       `json:"priceFactor,omitzero"`
	Tiling        Optional[XMLNumberVector] `json:"tiling,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesAODecalsDecalTransform struct {
	Row0 Optional[XMLNumberVector] `json:"row0,omitzero"`
	Row1 Optional[XMLNumberVector] `json:"row1,omitzero"`
	Row2 Optional[XMLNumberVector] `json:"row2,omitzero"`
	Row3 Optional[XMLNumberVector] `json:"row3,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesAODecalsDecal struct {
	Transform Optional[StrictTankConfigurationChassisEntryPropertiesAODecalsDecalTransform] `json:"transform,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesAODecals struct {
	Decal Optional[StrictTankConfigurationChassisEntryPropertiesAODecalsDecal] `json:"decal,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesArmorArmorValue struct {
	Text                    Optional[XMLInteger] `json:"#text,omitzero"`
	ChanceToHitByExplosion  Optional[XMLInteger] `json:"chanceToHitByExplosion,omitzero"`
	ChanceToHitByProjectile Optional[XMLInteger] `json:"chanceToHitByProjectile,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesEffects struct {
	Dust    Optional[string] `json:"dust,omitzero"`
	LodDist Optional[string] `json:"lodDist,omitzero"`
	Mud     Optional[string] `json:"mud,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesGroundNodesGroupEntry struct {
	Count      Optional[XMLInteger] `json:"count,omitzero"`
	IsLeft     Optional[XMLBool]    `json:"isLeft,omitzero"`
	MaxOffset  Optional[XMLNumber]  `json:"maxOffset,omitzero"`
	MinOffset  Optional[XMLNumber]  `json:"minOffset,omitzero"`
	StartIndex Optional[XMLInteger] `json:"startIndex,omitzero"`
	Template   Optional[string]     `json:"template,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesGroundNodes struct {
	Group Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesGroundNodesGroupEntry]] `json:"group,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesHitTester struct {
	CollisionModel Optional[string] `json:"collisionModel,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesInternalTrackLimitationEntry struct {
	BackwardSpeedLimit Optional[XMLNumber] `json:"backwardSpeedLimit,omitzero"`
	ForwardSpeedLimit  Optional[XMLNumber] `json:"forwardSpeedLimit,omitzero"`
	Track              Optional[string]    `json:"track,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesInternalTrack struct {
	Limitation Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesInternalTrackLimitationEntry]] `json:"limitation,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesModels struct {
	Destroyed Optional[string] `json:"destroyed,omitzero"`
	Exploded  Optional[string] `json:"exploded,omitzero"`
	Undamaged Optional[string] `json:"undamaged,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesShotDispersionFactors struct {
	VehicleMovement Optional[XMLNumber] `json:"vehicleMovement,omitzero"`
	VehicleRotation Optional[XMLNumber] `json:"vehicleRotation,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesSplineDescAtlas struct {
	UTiles Optional[XMLInteger] `json:"UTiles,omitzero"`
	VTiles Optional[XMLInteger] `json:"VTiles,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesSplineDesc struct {
	Atlas              Optional[StrictTankConfigurationChassisEntryPropertiesSplineDescAtlas] `json:"atlas,omitzero"`
	Left               Optional[string]                                                       `json:"left,omitzero"`
	LodDist            Optional[string]                                                       `json:"lodDist,omitzero"`
	Right              Optional[string]                                                       `json:"right,omitzero"`
	Segment2ModelLeft  Optional[string]                                                       `json:"segment2ModelLeft,omitzero"`
	Segment2ModelRight Optional[string]                                                       `json:"segment2ModelRight,omitzero"`
	Segment2Offset     Optional[XMLNumber]                                                    `json:"segment2Offset,omitzero"`
	SegmentLength      Optional[XMLNumber]                                                    `json:"segmentLength,omitzero"`
	SegmentModelLeft   Optional[string]                                                       `json:"segmentModelLeft,omitzero"`
	SegmentModelRight  Optional[string]                                                       `json:"segmentModelRight,omitzero"`
	SegmentOffset      Optional[XMLNumber]                                                    `json:"segmentOffset,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesSplineDescEntryAtlas struct {
	UTiles Optional[XMLInteger] `json:"UTiles,omitzero"`
	VTiles Optional[XMLInteger] `json:"VTiles,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesSplineDescEntry struct {
	Atlas             Optional[StrictTankConfigurationChassisEntryPropertiesSplineDescEntryAtlas] `json:"atlas,omitzero"`
	Left              Optional[string]                                                            `json:"left,omitzero"`
	LodDist           Optional[string]                                                            `json:"lodDist,omitzero"`
	Right             Optional[string]                                                            `json:"right,omitzero"`
	SegmentLength     Optional[XMLNumber]                                                         `json:"segmentLength,omitzero"`
	SegmentModelLeft  Optional[string]                                                            `json:"segmentModelLeft,omitzero"`
	SegmentModelRight Optional[string]                                                            `json:"segmentModelRight,omitzero"`
	SegmentOffset     Optional[XMLNumber]                                                         `json:"segmentOffset,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesTraces struct {
	BufferPrefs  Optional[string]          `json:"bufferPrefs,omitzero"`
	DecalGroup   Optional[string]          `json:"decalGroup,omitzero"`
	DecalTexture Optional[string]          `json:"decalTexture,omitzero"`
	LodDist      Optional[string]          `json:"lodDist,omitzero"`
	Size         Optional[XMLNumberVector] `json:"size,omitzero"`
	TextureSet   Optional[string]          `json:"textureSet,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesTrackNodesNodeEntry struct {
	BackwardElastK Optional[XMLNumber]  `json:"backwardElastK,omitzero"`
	Damping        Optional[XMLInteger] `json:"damping,omitzero"`
	Elasticity     Optional[XMLInteger] `json:"elasticity,omitzero"`
	ForwardElastK  Optional[XMLNumber]  `json:"forwardElastK,omitzero"`
	IsLeft         Optional[XMLBool]    `json:"isLeft,omitzero"`
	LeftSibling    Optional[string]     `json:"leftSibling,omitzero"`
	Name           Optional[string]     `json:"name,omitzero"`
	Offset         Optional[XMLNumber]  `json:"offset,omitzero"`
	RightSibling   Optional[string]     `json:"rightSibling,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesTrackNodes struct {
	BackwardElastK Optional[XMLNumber]                                                                    `json:"backwardElastK,omitzero"`
	Damping        Optional[XMLNumber]                                                                    `json:"damping,omitzero"`
	Elasticity     Optional[XMLInteger]                                                                   `json:"elasticity,omitzero"`
	Enable         Optional[XMLBool]                                                                      `json:"enable,omitzero"`
	ForwardElastK  Optional[XMLNumber]                                                                    `json:"forwardElastK,omitzero"`
	Gravity        Optional[XMLNumber]                                                                    `json:"gravity,omitzero"`
	LinkBones      Optional[XMLBool]                                                                      `json:"linkBones,omitzero"`
	MaxAmplitude   Optional[XMLNumber]                                                                    `json:"maxAmplitude,omitzero"`
	MaxOffset      Optional[XMLNumber]                                                                    `json:"maxOffset,omitzero"`
	Node           Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesTrackNodesNodeEntry]] `json:"node,omitzero"`
	Offset         Optional[XMLNumber]                                                                    `json:"offset,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesTrackNodesEntryNodeEntry struct {
	BackwardElastK Optional[XMLNumber] `json:"backwardElastK,omitzero"`
	ForwardElastK  Optional[XMLNumber] `json:"forwardElastK,omitzero"`
	IsLeft         Optional[XMLBool]   `json:"isLeft,omitzero"`
	Name           Optional[string]    `json:"name,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesTrackNodesEntry struct {
	BackwardElastK Optional[XMLInteger]                                                                        `json:"backwardElastK,omitzero"`
	Damping        Optional[XMLInteger]                                                                        `json:"damping,omitzero"`
	Elasticity     Optional[XMLInteger]                                                                        `json:"elasticity,omitzero"`
	Enable         Optional[XMLBool]                                                                           `json:"enable,omitzero"`
	ForwardElastK  Optional[XMLInteger]                                                                        `json:"forwardElastK,omitzero"`
	Gravity        Optional[XMLNumber]                                                                         `json:"gravity,omitzero"`
	LinkBones      Optional[XMLBool]                                                                           `json:"linkBones,omitzero"`
	MaxAmplitude   Optional[XMLNumber]                                                                         `json:"maxAmplitude,omitzero"`
	MaxOffset      Optional[XMLNumber]                                                                         `json:"maxOffset,omitzero"`
	Node           Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesTrackNodesEntryNodeEntry]] `json:"node,omitzero"`
	Offset         Optional[XMLNumber]                                                                         `json:"offset,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesTracks struct {
	LeftMaterial  Optional[string]    `json:"leftMaterial,omitzero"`
	LodDist       Optional[string]    `json:"lodDist,omitzero"`
	RightMaterial Optional[string]    `json:"rightMaterial,omitzero"`
	TextureScale  Optional[XMLNumber] `json:"textureScale,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocksChassisItemsEntry struct {
	Name       Optional[string]     `json:"name,omitzero" schema:"required,nonnull"`
	Properties Optional[XMLInteger] `json:"properties,omitzero" schema:"required"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocksChassis struct {
	Items Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesUnlocksChassisItemsEntry]] `json:"#items,omitzero"`
	Text  Optional[string]                                                                            `json:"#text,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocksGun struct {
	Text Optional[string] `json:"#text,omitzero"`
	Cost Optional[string] `json:"cost,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocksTurret struct {
	Text Optional[string] `json:"#text,omitzero"`
	Cost Optional[string] `json:"cost,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocksVehicle struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesUnlocks struct {
	Chassis Optional[StrictTankConfigurationChassisEntryPropertiesUnlocksChassis] `json:"chassis,omitzero"`
	Gun     Optional[StrictTankConfigurationChassisEntryPropertiesUnlocksGun]     `json:"gun,omitzero"`
	Turret  Optional[StrictTankConfigurationChassisEntryPropertiesUnlocksTurret]  `json:"turret,omitzero"`
	Vehicle Optional[StrictTankConfigurationChassisEntryPropertiesUnlocksVehicle] `json:"vehicle,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesWheelsGroup struct {
	Count      Optional[XMLInteger] `json:"count,omitzero"`
	IsLeft     Optional[XMLBool]    `json:"isLeft,omitzero"`
	Radius     Optional[XMLNumber]  `json:"radius,omitzero"`
	StartIndex Optional[XMLInteger] `json:"startIndex,omitzero"`
	Template   Optional[string]     `json:"template,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesWheelsGroupEntry struct {
	Count      Optional[XMLInteger] `json:"count,omitzero"`
	IsLeft     Optional[XMLBool]    `json:"isLeft,omitzero"`
	Radius     Optional[XMLNumber]  `json:"radius,omitzero"`
	StartIndex Optional[XMLInteger] `json:"startIndex,omitzero"`
	Template   Optional[string]     `json:"template,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesWheelsWheelEntry struct {
	IsLeading Optional[XMLBool]   `json:"isLeading,omitzero"`
	IsLeft    Optional[XMLBool]   `json:"isLeft,omitzero"`
	Name      Optional[string]    `json:"name,omitzero"`
	Radius    Optional[XMLNumber] `json:"radius,omitzero"`
	SyncAngle Optional[XMLNumber] `json:"syncAngle,omitzero"`
}

type StrictTankConfigurationChassisEntryPropertiesWheels struct {
	Group                 Optional[XMLObjectList[StrictTankConfigurationChassisEntryPropertiesWheelsGroup, StrictTankConfigurationChassisEntryPropertiesWheelsGroupEntry]] `json:"group,omitzero"`
	LeadingWheelSyncAngle Optional[XMLNumber]                                                                                                                              `json:"leadingWheelSyncAngle,omitzero"`
	LodDist               Optional[string]                                                                                                                                 `json:"lodDist,omitzero"`
	Wheel                 Optional[[]Optional[StrictTankConfigurationChassisEntryPropertiesWheelsWheelEntry]]                                                              `json:"wheel,omitzero"`
}

type StrictTankConfigurationChassisEntryProperties struct {
	AODecals               Optional[StrictTankConfigurationChassisEntryPropertiesAODecals]                                                                                `json:"AODecals,omitzero"`
	Armor                  Optional[map[string]Optional[XMLScalarObject[XMLInteger, StrictTankConfigurationChassisEntryPropertiesArmorArmorValue]]]                       `json:"armor,omitzero"`
	BrakeForce             Optional[XMLInteger]                                                                                                                           `json:"brakeForce,omitzero"`
	BulkHealthFactor       Optional[XMLNumber]                                                                                                                            `json:"bulkHealthFactor,omitzero"`
	Description            Optional[XMLLocalizationKey]                                                                                                                   `json:"description,omitzero"`
	DrivingWheels          Optional[XMLScalarList[string, string]]                                                                                                        `json:"drivingWheels,omitzero"`
	Effects                Optional[StrictTankConfigurationChassisEntryPropertiesEffects]                                                                                 `json:"effects,omitzero"`
	FitsRatingBattle       Optional[XMLBool]                                                                                                                              `json:"fitsRatingBattle,omitzero"`
	GroundNodes            Optional[StrictTankConfigurationChassisEntryPropertiesGroundNodes]                                                                             `json:"groundNodes,omitzero"`
	HitTester              Optional[StrictTankConfigurationChassisEntryPropertiesHitTester]                                                                               `json:"hitTester,omitzero"`
	HullPosition           Optional[XMLNumberVector]                                                                                                                      `json:"hullPosition,omitzero"`
	Icon                   Optional[string]                                                                                                                               `json:"icon,omitzero"`
	InternalTrack          Optional[StrictTankConfigurationChassisEntryPropertiesInternalTrack]                                                                           `json:"internalTrack,omitzero"`
	Level                  Optional[XMLInteger]                                                                                                                           `json:"level,omitzero"`
	MaxClimbAngle          Optional[XMLNumber]                                                                                                                            `json:"maxClimbAngle,omitzero"`
	MaxHealth              Optional[XMLScalarList[XMLInteger, XMLInteger]]                                                                                                `json:"maxHealth,omitzero"`
	MaxLoad                Optional[XMLInteger]                                                                                                                           `json:"maxLoad,omitzero"`
	MaxRegenHealth         Optional[XMLScalarList[XMLInteger, XMLInteger]]                                                                                                `json:"maxRegenHealth,omitzero"`
	Models                 Optional[StrictTankConfigurationChassisEntryPropertiesModels]                                                                                  `json:"models,omitzero"`
	NavmeshGirth           Optional[XMLNumber]                                                                                                                            `json:"navmeshGirth,omitzero"`
	NotInShop              Optional[XMLBool]                                                                                                                              `json:"notInShop,omitzero"`
	Price                  Optional[XMLNumber]                                                                                                                            `json:"price,omitzero"`
	RepairCost             Optional[XMLNumber]                                                                                                                            `json:"repairCost,omitzero"`
	RotationIsAroundCenter Optional[string]                                                                                                                               `json:"rotationIsAroundCenter,omitzero"`
	RotationSpeed          Optional[XMLNumber]                                                                                                                            `json:"rotationSpeed,omitzero"`
	ShortUserString        Optional[XMLLocalizationKey]                                                                                                                   `json:"shortUserString,omitzero"`
	ShotDispersionFactors  Optional[StrictTankConfigurationChassisEntryPropertiesShotDispersionFactors]                                                                   `json:"shotDispersionFactors,omitzero"`
	Sound                  Optional[string]                                                                                                                               `json:"sound,omitzero"`
	SoundNPC               Optional[string]                                                                                                                               `json:"soundNPC,omitzero"`
	SoundPC                Optional[string]                                                                                                                               `json:"soundPC,omitzero"`
	SplineDesc             Optional[XMLObjectList[StrictTankConfigurationChassisEntryPropertiesSplineDesc, StrictTankConfigurationChassisEntryPropertiesSplineDescEntry]] `json:"splineDesc,omitzero"`
	Tags                   Optional[string]                                                                                                                               `json:"tags,omitzero"`
	TerrainResistance      Optional[XMLNumberVector]                                                                                                                      `json:"terrainResistance,omitzero"`
	TopRightCarryingPoint  Optional[XMLScalarList[XMLNumberVector, XMLNumberVector]]                                                                                      `json:"topRightCarryingPoint,omitzero"`
	Traces                 Optional[StrictTankConfigurationChassisEntryPropertiesTraces]                                                                                  `json:"traces,omitzero"`
	TrackNodes             Optional[XMLObjectList[StrictTankConfigurationChassisEntryPropertiesTrackNodes, StrictTankConfigurationChassisEntryPropertiesTrackNodesEntry]] `json:"trackNodes,omitzero"`
	TrackThickness         Optional[XMLScalarList[XMLNumber, XMLNumber]]                                                                                                  `json:"trackThickness,omitzero"`
	Tracks                 Optional[StrictTankConfigurationChassisEntryPropertiesTracks]                                                                                  `json:"tracks,omitzero"`
	Unlocks                Optional[StrictTankConfigurationChassisEntryPropertiesUnlocks]                                                                                 `json:"unlocks,omitzero"`
	UserString             Optional[XMLLocalizationKey]                                                                                                                   `json:"userString,omitzero"`
	Weight                 Optional[XMLInteger]                                                                                                                           `json:"weight,omitzero"`
	Wheels                 Optional[StrictTankConfigurationChassisEntryPropertiesWheels]                                                                                  `json:"wheels,omitzero"`
	WwsoundNPC             Optional[string]                                                                                                                               `json:"wwsoundNPC,omitzero"`
	WwsoundPC              Optional[string]                                                                                                                               `json:"wwsoundPC,omitzero"`
}

type StrictTankConfigurationChassisEntry struct {
	Name       Optional[string]                                        `json:"name,omitzero" schema:"required,nonnull"`
	Properties Optional[StrictTankConfigurationChassisEntryProperties] `json:"properties,omitzero" schema:"required"`
}

type StrictTankConfigurationCrew struct {
	Commander Optional[string]                        `json:"commander,omitzero"`
	Driver    Optional[string]                        `json:"driver,omitzero"`
	Gunner    Optional[XMLScalarList[string, string]] `json:"gunner,omitzero"`
	Loader    Optional[XMLScalarList[string, string]] `json:"loader,omitzero"`
	Radioman  Optional[[]Optional[string]]            `json:"radioman,omitzero"`
}

type StrictTankConfigurationEffects struct {
	Collision         Optional[string] `json:"collision,omitzero"`
	DamagedStateGroup Optional[string] `json:"damagedStateGroup,omitzero"`
	Destruction       Optional[string] `json:"destruction,omitzero"`
	Explosion         Optional[string] `json:"explosion,omitzero"`
	Flaming           Optional[string] `json:"flaming,omitzero"`
}

type StrictTankConfigurationEmblems struct {
	Alpha   Optional[XMLNumber] `json:"alpha,omitzero"`
	Default Optional[string]    `json:"default,omitzero"`
	LodDist Optional[string]    `json:"lodDist,omitzero"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocksChassisItemsEntry struct {
	Name       Optional[string] `json:"name,omitzero" schema:"required,nonnull"`
	Properties Optional[string] `json:"properties,omitzero" schema:"required"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocksChassis struct {
	Items Optional[[]Optional[StrictTankConfigurationEnginesEntryPropertiesUnlocksChassisItemsEntry]] `json:"#items,omitzero"`
	Text  Optional[string]                                                                            `json:"#text,omitzero"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocksEngine struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocksVehicle struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocksVehicleEntry struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationEnginesEntryPropertiesUnlocks struct {
	Chassis Optional[StrictTankConfigurationEnginesEntryPropertiesUnlocksChassis]                                                                                  `json:"chassis,omitzero"`
	Engine  Optional[StrictTankConfigurationEnginesEntryPropertiesUnlocksEngine]                                                                                   `json:"engine,omitzero"`
	Vehicle Optional[XMLObjectList[StrictTankConfigurationEnginesEntryPropertiesUnlocksVehicle, StrictTankConfigurationEnginesEntryPropertiesUnlocksVehicleEntry]] `json:"vehicle,omitzero"`
}

type StrictTankConfigurationEnginesEntryProperties struct {
	Text             Optional[string]                                               `json:"#text,omitzero"`
	FitsRatingBattle Optional[XMLBool]                                              `json:"fitsRatingBattle,omitzero"`
	Unlocks          Optional[StrictTankConfigurationEnginesEntryPropertiesUnlocks] `json:"unlocks,omitzero"`
}

type StrictTankConfigurationEnginesEntry struct {
	Name       Optional[string]                                                                 `json:"name,omitzero" schema:"required,nonnull"`
	Properties Optional[XMLScalarObject[string, StrictTankConfigurationEnginesEntryProperties]] `json:"properties,omitzero" schema:"required"`
}

type StrictTankConfigurationExtrasAlteredSpotDuration struct {
	SpotDurationTimeChange Optional[XMLNumber] `json:"spotDurationTimeChange,omitzero"`
	WithSpotEyeEffect      Optional[string]    `json:"withSpotEyeEffect,omitzero"`
}

type StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesControllerStateEntryFactorsModifiers struct {
	EnginePowerFactor         Optional[XMLNumber] `json:"enginePowerFactor,omitzero"`
	SpeedLimitBkwdFactor      Optional[XMLNumber] `json:"speedLimitBkwdFactor,omitzero"`
	SpeedLimitFwdFactor       Optional[XMLNumber] `json:"speedLimitFwdFactor,omitzero"`
	TurretRotationSpeedFactor Optional[XMLNumber] `json:"turretRotationSpeedFactor,omitzero"`
}

type StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesControllerStateEntry struct {
	Armors           Optional[string]                                                                                          `json:"armors,omitzero"`
	Default          Optional[string]                                                                                          `json:"default,omitzero"`
	Enabled          Optional[XMLBool]                                                                                         `json:"enabled,omitzero"`
	FactorsModifiers Optional[StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesControllerStateEntryFactorsModifiers] `json:"factorsModifiers,omitzero"`
	Id               Optional[XMLInteger]                                                                                      `json:"id,omitzero"`
	Type             Optional[string]                                                                                          `json:"type,omitzero"`
}

type StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesController struct {
	State Optional[[]Optional[StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesControllerStateEntry]] `json:"state,omitzero"`
}

type StrictTankConfigurationExtrasArmorMovingAbility struct {
	ArmorsStatesController Optional[StrictTankConfigurationExtrasArmorMovingAbilityArmorsStatesController] `json:"armorsStatesController,omitzero"`
	WrappedAbilityName     Optional[string]                                                                `json:"wrappedAbilityName,omitzero"`
}

type StrictTankConfigurationExtrasArmorsStatesControllerStateEntryActivateConditionHealthActivateCondition struct {
	Text   Optional[XMLInteger] `json:"#text,omitzero"`
	AttrOp Optional[string]     `json:"@op,omitzero"`
}

type StrictTankConfigurationExtrasArmorsStatesControllerStateEntryActivateCondition struct {
	HealthActivateCondition Optional[StrictTankConfigurationExtrasArmorsStatesControllerStateEntryActivateConditionHealthActivateCondition] `json:"healthActivateCondition,omitzero"`
}

type StrictTankConfigurationExtrasArmorsStatesControllerStateEntryFactorsModifiers struct {
	ReceiveDamageFactor Optional[XMLNumber] `json:"receiveDamageFactor,omitzero"`
}

type StrictTankConfigurationExtrasArmorsStatesControllerStateEntry struct {
	ActivateCondition Optional[StrictTankConfigurationExtrasArmorsStatesControllerStateEntryActivateCondition] `json:"activateCondition,omitzero"`
	Armors            Optional[string]                                                                         `json:"armors,omitzero"`
	Default           Optional[string]                                                                         `json:"default,omitzero"`
	Enabled           Optional[XMLBool]                                                                        `json:"enabled,omitzero"`
	FactorsModifiers  Optional[StrictTankConfigurationExtrasArmorsStatesControllerStateEntryFactorsModifiers]  `json:"factorsModifiers,omitzero"`
	Id                Optional[XMLInteger]                                                                     `json:"id,omitzero"`
	Type              Optional[string]                                                                         `json:"type,omitzero"`
}

type StrictTankConfigurationExtrasArmorsStatesController struct {
	State Optional[[]Optional[StrictTankConfigurationExtrasArmorsStatesControllerStateEntry]] `json:"state,omitzero"`
}

type StrictTankConfigurationExtrasDynamicArmorWrapperOverrides struct {
	Cooldown   Optional[XMLInteger] `json:"cooldown,omitzero"`
	Duration   Optional[XMLInteger] `json:"duration,omitzero"`
	ShieldCoef Optional[XMLNumber]  `json:"shieldCoef,omitzero"`
}

type StrictTankConfigurationExtrasDynamicArmorWrapper struct {
	Overrides          Optional[StrictTankConfigurationExtrasDynamicArmorWrapperOverrides] `json:"overrides,omitzero"`
	WrappedAbilityName Optional[string]                                                    `json:"wrappedAbilityName,omitzero"`
}

type StrictTankConfigurationExtrasDynamicGunSwitcherWrapperOverrides struct {
	InnerCooldown Optional[XMLInteger] `json:"innerCooldown,omitzero"`
}

type StrictTankConfigurationExtrasDynamicGunSwitcherWrapper struct {
	Overrides          Optional[StrictTankConfigurationExtrasDynamicGunSwitcherWrapperOverrides] `json:"overrides,omitzero"`
	WrappedAbilityName Optional[string]                                                          `json:"wrappedAbilityName,omitzero"`
}

type StrictTankConfigurationExtrasGunCooling struct {
	CoolingTime        Optional[XMLNumber]                                                  `json:"coolingTime,omitzero"`
	DamageMul          Optional[XMLScalarObject[XMLNumber, map[string]Optional[XMLNumber]]] `json:"damageMul,omitzero"`
	ReloadBooster      Optional[XMLScalarObject[XMLNumber, map[string]Optional[XMLNumber]]] `json:"reloadBooster,omitzero"`
	UncompletedPenalty Optional[XMLNumber]                                                  `json:"uncompletedPenalty,omitzero"`
}

type StrictTankConfigurationExtrasGunSwitcherWrapperOverrides struct {
	Cooldown      Optional[XMLInteger] `json:"cooldown,omitzero"`
	Duration      Optional[XMLInteger] `json:"duration,omitzero"`
	InnerCooldown Optional[XMLInteger] `json:"innerCooldown,omitzero"`
}

type StrictTankConfigurationExtrasGunSwitcherWrapper struct {
	Overrides          Optional[StrictTankConfigurationExtrasGunSwitcherWrapperOverrides] `json:"overrides,omitzero"`
	WrappedAbilityName Optional[string]                                                   `json:"wrappedAbilityName,omitzero"`
}

type StrictTankConfigurationExtrasRocketBoosterWrapperOverridesEqtParams struct {
	Cooldown Optional[XMLInteger] `json:"cooldown,omitzero"`
	Duration Optional[XMLInteger] `json:"duration,omitzero"`
}

type StrictTankConfigurationExtrasRocketBoosterWrapperOverridesMultipliers struct {
	AngularSpeedMultiplier          Optional[XMLNumber] `json:"angularSpeedMultiplier,omitzero"`
	BackwardSpeedMultiplier         Optional[XMLNumber] `json:"backwardSpeedMultiplier,omitzero"`
	EnginePowerMultiplier           Optional[XMLNumber] `json:"enginePowerMultiplier,omitzero"`
	ForwardSpeedMultiplier          Optional[XMLNumber] `json:"forwardSpeedMultiplier,omitzero"`
	IncomingRammingDamageMultiplier Optional[XMLNumber] `json:"incomingRammingDamageMultiplier,omitzero"`
	OutgoingRammingDamageMultiplier Optional[XMLNumber] `json:"outgoingRammingDamageMultiplier,omitzero"`
}

type StrictTankConfigurationExtrasRocketBoosterWrapperOverridesStatics struct {
	FlatteningCoefficient                  Optional[XMLInteger] `json:"flatteningCoefficient,omitzero"`
	IncomingDefaultRammingDamageMultiplier Optional[XMLNumber]  `json:"incomingDefaultRammingDamageMultiplier,omitzero"`
	OutgoingDefaultRammingDamageMultiplier Optional[XMLNumber]  `json:"outgoingDefaultRammingDamageMultiplier,omitzero"`
	StaticForce                            Optional[XMLInteger] `json:"staticForce,omitzero"`
	StaticRestForce                        Optional[XMLInteger] `json:"staticRestForce,omitzero"`
	WideningCoefficient                    Optional[XMLNumber]  `json:"wideningCoefficient,omitzero"`
}

type StrictTankConfigurationExtrasRocketBoosterWrapperOverrides struct {
	EqtParams   Optional[StrictTankConfigurationExtrasRocketBoosterWrapperOverridesEqtParams]   `json:"eqtParams,omitzero"`
	Multipliers Optional[StrictTankConfigurationExtrasRocketBoosterWrapperOverridesMultipliers] `json:"multipliers,omitzero"`
	Statics     Optional[StrictTankConfigurationExtrasRocketBoosterWrapperOverridesStatics]     `json:"statics,omitzero"`
}

type StrictTankConfigurationExtrasRocketBoosterWrapper struct {
	Overrides          Optional[StrictTankConfigurationExtrasRocketBoosterWrapperOverrides] `json:"overrides,omitzero"`
	WrappedAbilityName Optional[string]                                                     `json:"wrappedAbilityName,omitzero"`
}

type StrictTankConfigurationExtrasShotDispersionStabilizatorBackward struct {
	SpeedIncreaseFactor Optional[XMLNumber]  `json:"speedIncreaseFactor,omitzero"`
	SpeedLimitFactor    Optional[XMLInteger] `json:"speedLimitFactor,omitzero"`
}

type StrictTankConfigurationExtrasShotDispersionStabilizatorForward struct {
	SpeedIncreaseFactor Optional[XMLNumber]  `json:"speedIncreaseFactor,omitzero"`
	SpeedLimitFactor    Optional[XMLInteger] `json:"speedLimitFactor,omitzero"`
}

type StrictTankConfigurationExtrasShotDispersionStabilizatorRotation struct {
	SpeedIncreaseFactor Optional[XMLNumber]  `json:"speedIncreaseFactor,omitzero"`
	SpeedLimitFactor    Optional[XMLInteger] `json:"speedLimitFactor,omitzero"`
}

type StrictTankConfigurationExtrasShotDispersionStabilizator struct {
	Backward Optional[StrictTankConfigurationExtrasShotDispersionStabilizatorBackward] `json:"backward,omitzero"`
	Forward  Optional[StrictTankConfigurationExtrasShotDispersionStabilizatorForward]  `json:"forward,omitzero"`
	Rotation Optional[StrictTankConfigurationExtrasShotDispersionStabilizatorRotation] `json:"rotation,omitzero"`
}

type StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectDesert struct {
	EnginePowerFactor          Optional[XMLNumber] `json:"enginePowerFactor,omitzero"`
	SpeedLimitsBackwardFactor  Optional[XMLNumber] `json:"speedLimitsBackwardFactor,omitzero"`
	SpeedLimitsForwardFactor   Optional[XMLNumber] `json:"speedLimitsForwardFactor,omitzero"`
	VehicleRotationSpeedFactor Optional[XMLNumber] `json:"vehicleRotationSpeedFactor,omitzero"`
}

type StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectSummer struct {
	ChassisMovementShotDispersionFactor Optional[XMLNumber] `json:"chassisMovementShotDispersionFactor,omitzero"`
	ChassisRotationShotDispersionFactor Optional[XMLNumber] `json:"chassisRotationShotDispersionFactor,omitzero"`
	GunAimingFactor                     Optional[XMLNumber] `json:"gunAimingFactor,omitzero"`
	GunReloadFactor                     Optional[XMLNumber] `json:"gunReloadFactor,omitzero"`
	GunTurretRotationDispersionFactor   Optional[XMLNumber] `json:"gunTurretRotationDispersionFactor,omitzero"`
	ShotDispersionAngleFactor           Optional[XMLNumber] `json:"shotDispersionAngleFactor,omitzero"`
}

type StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectWinter struct {
	ChassisHealthFactor           Optional[XMLNumber] `json:"chassisHealthFactor,omitzero"`
	ChassisHysteresisHealthFactor Optional[XMLNumber] `json:"chassisHysteresisHealthFactor,omitzero"`
	ChassisMaxRegenHealthFactor   Optional[XMLNumber] `json:"chassisMaxRegenHealthFactor,omitzero"`
	ChassisRegenFactor            Optional[XMLNumber] `json:"chassisRegenFactor,omitzero"`
	EngineHealthFactor            Optional[XMLNumber] `json:"engineHealthFactor,omitzero"`
	EngineHysteresisHealthFactor  Optional[XMLNumber] `json:"engineHysteresisHealthFactor,omitzero"`
	EngineMaxRegenHealthFactor    Optional[XMLNumber] `json:"engineMaxRegenHealthFactor,omitzero"`
	EngineRegenFactor             Optional[XMLNumber] `json:"engineRegenFactor,omitzero"`
	VehicleMaxHealthFactor        Optional[XMLNumber] `json:"vehicleMaxHealthFactor,omitzero"`
}

type StrictTankConfigurationExtrasYearSeasonsEffectOverrides struct {
	YearSeasonsEffectDesert Optional[StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectDesert] `json:"yearSeasonsEffectDesert,omitzero"`
	YearSeasonsEffectSummer Optional[StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectSummer] `json:"yearSeasonsEffectSummer,omitzero"`
	YearSeasonsEffectWinter Optional[StrictTankConfigurationExtrasYearSeasonsEffectOverridesYearSeasonsEffectWinter] `json:"yearSeasonsEffectWinter,omitzero"`
}

type StrictTankConfigurationExtrasYearSeasonsEffect struct {
	Overrides Optional[StrictTankConfigurationExtrasYearSeasonsEffectOverrides] `json:"overrides,omitzero"`
}

type StrictTankConfigurationExtras struct {
	AlteredSpotDuration        Optional[StrictTankConfigurationExtrasAlteredSpotDuration]        `json:"alteredSpotDuration,omitzero"`
	ArmorMovingAbility         Optional[StrictTankConfigurationExtrasArmorMovingAbility]         `json:"armorMovingAbility,omitzero"`
	ArmorsStatesController     Optional[StrictTankConfigurationExtrasArmorsStatesController]     `json:"armorsStatesController,omitzero"`
	DesertPower                Optional[string]                                                  `json:"desertPower,omitzero"`
	DynamicArmorWrapper        Optional[StrictTankConfigurationExtrasDynamicArmorWrapper]        `json:"dynamicArmorWrapper,omitzero"`
	DynamicGunSwitcherWrapper  Optional[StrictTankConfigurationExtrasDynamicGunSwitcherWrapper]  `json:"dynamicGunSwitcherWrapper,omitzero"`
	GunCooling                 Optional[StrictTankConfigurationExtrasGunCooling]                 `json:"gunCooling,omitzero"`
	GunSwitcherWrapper         Optional[StrictTankConfigurationExtrasGunSwitcherWrapper]         `json:"gunSwitcherWrapper,omitzero"`
	RocketBoosterWrapper       Optional[StrictTankConfigurationExtrasRocketBoosterWrapper]       `json:"rocketBoosterWrapper,omitzero"`
	ShotDispersionStabilizator Optional[StrictTankConfigurationExtrasShotDispersionStabilizator] `json:"shotDispersionStabilizator,omitzero"`
	YearSeasonsEffect          Optional[StrictTankConfigurationExtrasYearSeasonsEffect]          `json:"yearSeasonsEffect,omitzero"`
}

type StrictTankConfigurationFuelTanksEntry struct {
	Name       Optional[string] `json:"name,omitzero" schema:"required,nonnull"`
	Properties Optional[string] `json:"properties,omitzero" schema:"required"`
}

type StrictTankConfigurationHorns struct {
	DistanceFactor Optional[XMLInteger] `json:"distanceFactor,omitzero"`
	PriceFactor    Optional[XMLInteger] `json:"priceFactor,omitzero"`
	VolumeFactor   Optional[XMLInteger] `json:"volumeFactor,omitzero"`
}

type StrictTankConfigurationHullAmmoBayHealth struct {
	MaxHealth      Optional[XMLInteger] `json:"maxHealth,omitzero"`
	MaxRegenHealth Optional[XMLInteger] `json:"maxRegenHealth,omitzero"`
	RepairCost     Optional[XMLNumber]  `json:"repairCost,omitzero"`
}

type StrictTankConfigurationHullArmorArmorValue struct {
	Text                    Optional[XMLNumber]  `json:"#text,omitzero"`
	ChanceToHitByExplosion  Optional[XMLInteger] `json:"chanceToHitByExplosion,omitzero"`
	ChanceToHitByProjectile Optional[XMLInteger] `json:"chanceToHitByProjectile,omitzero"`
	VehicleDamageFactor     Optional[XMLNumber]  `json:"vehicleDamageFactor,omitzero"`
}

type StrictTankConfigurationHullBlitzBbox struct {
	Max Optional[XMLNumberVector] `json:"max,omitzero"`
	Min Optional[XMLNumberVector] `json:"min,omitzero"`
}

type StrictTankConfigurationHullBlitzPoints struct {
	Normal Optional[XMLNumberVector] `json:"normal,omitzero"`
}

type StrictTankConfigurationHullBlitz struct {
	ArmorScale       Optional[string]                                 `json:"armorScale,omitzero"`
	AverageThickness Optional[map[string]Optional[XMLNumber]]         `json:"averageThickness,omitzero"`
	Bbox             Optional[StrictTankConfigurationHullBlitzBbox]   `json:"bbox,omitzero"`
	BlitzPartName    Optional[string]                                 `json:"blitzPartName,omitzero"`
	Points           Optional[StrictTankConfigurationHullBlitzPoints] `json:"points,omitzero"`
	PushNormalsScale Optional[string]                                 `json:"pushNormalsScale,omitzero"`
}

type StrictTankConfigurationHullCamouflage struct {
	ExclusionMask Optional[string]          `json:"exclusionMask,omitzero"`
	Tiling        Optional[XMLNumberVector] `json:"tiling,omitzero"`
}

type StrictTankConfigurationHullEmblemSlotsClan struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationHullEmblemSlotsInscription struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationHullEmblemSlotsInscriptionEntry struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationHullEmblemSlotsPlayer struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationHullEmblemSlotsPlayerEntry struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	IsMirrored    Optional[XMLBool]         `json:"isMirrored,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationHullEmblemSlots struct {
	Clan        Optional[StrictTankConfigurationHullEmblemSlotsClan]                                                                               `json:"clan,omitzero"`
	Inscription Optional[XMLObjectList[StrictTankConfigurationHullEmblemSlotsInscription, StrictTankConfigurationHullEmblemSlotsInscriptionEntry]] `json:"inscription,omitzero"`
	Player      Optional[XMLObjectList[StrictTankConfigurationHullEmblemSlotsPlayer, StrictTankConfigurationHullEmblemSlotsPlayerEntry]]           `json:"player,omitzero"`
}

type StrictTankConfigurationHullExhaustTags struct {
	Diesel   Optional[string] `json:"diesel,omitzero"`
	Gasoline Optional[string] `json:"gasoline,omitzero"`
}

type StrictTankConfigurationHullExhaust struct {
	Nodes Optional[XMLScalarList[string, string]]          `json:"nodes,omitzero"`
	Pixie Optional[string]                                 `json:"pixie,omitzero"`
	Rates Optional[XMLNumberVector]                        `json:"rates,omitzero"`
	Tags  Optional[StrictTankConfigurationHullExhaustTags] `json:"tags,omitzero"`
}

type StrictTankConfigurationHullFakeTurrets struct {
	Battle Optional[XMLInteger] `json:"battle,omitzero"`
	Lobby  Optional[XMLInteger] `json:"lobby,omitzero"`
}

type StrictTankConfigurationHullHitTester struct {
	CollisionModel Optional[string] `json:"collisionModel,omitzero"`
}

type StrictTankConfigurationHullModels struct {
	Destroyed Optional[string] `json:"destroyed,omitzero"`
	Exploded  Optional[string] `json:"exploded,omitzero"`
	Undamaged Optional[string] `json:"undamaged,omitzero"`
}

type StrictTankConfigurationHullSwinging struct {
	LodDist              Optional[string]          `json:"lodDist,omitzero"`
	PitchParams          Optional[XMLNumberVector] `json:"pitchParams,omitzero"`
	RollParams           Optional[XMLNumberVector] `json:"rollParams,omitzero"`
	SensitivityToImpulse Optional[XMLNumber]       `json:"sensitivityToImpulse,omitzero"`
	WaterSwinging        Optional[XMLBool]         `json:"waterSwinging,omitzero"`
}

type StrictTankConfigurationHullTurretInitialRotation struct {
	Pitch Optional[XMLNumber] `json:"pitch,omitzero"`
	Roll  Optional[XMLNumber] `json:"roll,omitzero"`
	Yaw   Optional[XMLNumber] `json:"yaw,omitzero"`
}

type StrictTankConfigurationHullTurretPositions struct {
	Turret Optional[XMLNumberVector] `json:"turret,omitzero"`
}

type StrictTankConfigurationHull struct {
	AODecals              Optional[string]                                                                                                     `json:"AODecals,omitzero"`
	AmmoBayHealth         Optional[StrictTankConfigurationHullAmmoBayHealth]                                                                   `json:"ammoBayHealth,omitzero"`
	Armor                 Optional[map[string]Optional[XMLScalarObjectList[XMLNumber, StrictTankConfigurationHullArmorArmorValue, XMLNumber]]] `json:"armor,omitzero"`
	Blitz                 Optional[StrictTankConfigurationHullBlitz]                                                                           `json:"blitz,omitzero"`
	Camouflage            Optional[StrictTankConfigurationHullCamouflage]                                                                      `json:"camouflage,omitzero"`
	EmblemSlots           Optional[StrictTankConfigurationHullEmblemSlots]                                                                     `json:"emblemSlots,omitzero"`
	Exhaust               Optional[StrictTankConfigurationHullExhaust]                                                                         `json:"exhaust,omitzero"`
	FakeTurrets           Optional[StrictTankConfigurationHullFakeTurrets]                                                                     `json:"fakeTurrets,omitzero"`
	HangarShadowTexture   Optional[string]                                                                                                     `json:"hangarShadowTexture,omitzero"`
	HitTester             Optional[StrictTankConfigurationHullHitTester]                                                                       `json:"hitTester,omitzero"`
	MaxHealth             Optional[XMLInteger]                                                                                                 `json:"maxHealth,omitzero"`
	Models                Optional[StrictTankConfigurationHullModels]                                                                          `json:"models,omitzero"`
	PrimaryArmor          Optional[string]                                                                                                     `json:"primaryArmor,omitzero"`
	Swinging              Optional[StrictTankConfigurationHullSwinging]                                                                        `json:"swinging,omitzero"`
	TurretInitialRotation Optional[StrictTankConfigurationHullTurretInitialRotation]                                                           `json:"turretInitialRotation,omitzero"`
	TurretPositions       Optional[StrictTankConfigurationHullTurretPositions]                                                                 `json:"turretPositions,omitzero"`
	Weight                Optional[XMLNumber]                                                                                                  `json:"weight,omitzero"`
}

type StrictTankConfigurationInvisibility struct {
	CamouflageNetBonus Optional[XMLNumber] `json:"camouflageNetBonus,omitzero"`
	FirePenalty        Optional[XMLNumber] `json:"firePenalty,omitzero"`
	Moving             Optional[XMLNumber] `json:"moving,omitzero"`
	Still              Optional[XMLNumber] `json:"still,omitzero"`
}

type StrictTankConfigurationRadiosEntryProperties struct {
	Text    Optional[string] `json:"#text,omitzero"`
	Unlocks Optional[string] `json:"unlocks,omitzero"`
}

type StrictTankConfigurationRadiosEntry struct {
	Name       Optional[string]                                                                `json:"name,omitzero" schema:"required,nonnull"`
	Properties Optional[XMLScalarObject[string, StrictTankConfigurationRadiosEntryProperties]] `json:"properties,omitzero" schema:"required"`
}

type StrictTankConfigurationSpeedLimits struct {
	Backward Optional[XMLInteger] `json:"backward,omitzero"`
	Forward  Optional[XMLNumber]  `json:"forward,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesArmorArmorValue struct {
	Text                    Optional[XMLNumber]  `json:"#text,omitzero"`
	ChanceToHitByExplosion  Optional[XMLInteger] `json:"chanceToHitByExplosion,omitzero"`
	ChanceToHitByProjectile Optional[XMLInteger] `json:"chanceToHitByProjectile,omitzero"`
	VehicleDamageFactor     Optional[XMLNumber]  `json:"vehicleDamageFactor,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesCamouflage struct {
	ExclusionMask Optional[string]          `json:"exclusionMask,omitzero"`
	Tiling        Optional[XMLNumberVector] `json:"tiling,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsClan struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsInscription struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsInscriptionEntry struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsPlayer struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsPlayerEntry struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	IsMirrored    Optional[XMLBool]         `json:"isMirrored,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesEmblemSlots struct {
	Clan        Optional[StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsClan]                                                                                                  `json:"clan,omitzero"`
	Inscription Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsInscription, StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsInscriptionEntry]] `json:"inscription,omitzero"`
	Player      Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsPlayer, StrictTankConfigurationTurrets0EntryPropertiesEmblemSlotsPlayerEntry]]           `json:"player,omitzero"`
}

type StrictTankConfigurationNamedShell struct {
	Text  Optional[string]     `json:"#text,omitzero"`
	Price Optional[XMLInteger] `json:"price,omitzero"`
}

type StrictTankConfigurationNamedShot struct {
	Shell Optional[StrictTankConfigurationNamedShell] `json:"shell,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesArmorArmorValue struct {
	Text                    Optional[XMLNumber]  `json:"#text,omitzero"`
	ChanceToHitByExplosion  Optional[XMLInteger] `json:"chanceToHitByExplosion,omitzero"`
	ChanceToHitByProjectile Optional[XMLInteger] `json:"chanceToHitByProjectile,omitzero"`
	VehicleDamageFactor     Optional[XMLNumber]  `json:"vehicleDamageFactor,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesBurst struct {
	Count Optional[XMLInteger] `json:"count,omitzero"`
	Rate  Optional[XMLInteger] `json:"rate,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesCamouflage struct {
	ExclusionMask Optional[string]          `json:"exclusionMask,omitzero"`
	Tiling        Optional[XMLNumberVector] `json:"tiling,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesClip struct {
	Count Optional[XMLInteger] `json:"count,omitzero"`
	Rate  Optional[XMLNumber]  `json:"rate,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesEmblemSlotsInsigniaOnGun struct {
	HideIfDamaged Optional[XMLBool]         `json:"hideIfDamaged,omitzero"`
	RayEnd        Optional[XMLNumberVector] `json:"rayEnd,omitzero"`
	RayStart      Optional[XMLNumberVector] `json:"rayStart,omitzero"`
	RayUp         Optional[XMLNumberVector] `json:"rayUp,omitzero"`
	Size          Optional[XMLNumber]       `json:"size,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesEmblemSlots struct {
	InsigniaOnGun Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesEmblemSlotsInsigniaOnGun] `json:"insigniaOnGun,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtraPitchLimits struct {
	Text       Optional[string]          `json:"#text,omitzero"`
	Back       Optional[XMLNumberVector] `json:"back,omitzero"`
	Front      Optional[XMLNumberVector] `json:"front,omitzero"`
	Transition Optional[XMLInteger]      `json:"transition,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtrasImprovedDetectionShellEntry struct {
	Duration Optional[XMLInteger] `json:"duration,omitzero"`
	Kind     Optional[string]     `json:"kind,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtrasImprovedDetection struct {
	Shell Optional[[]Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtrasImprovedDetectionShellEntry]] `json:"shell,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtras struct {
	ImprovedDetection Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtrasImprovedDetection] `json:"improvedDetection,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesFakeTurrets struct {
	Battle Optional[XMLInteger] `json:"battle,omitzero"`
	Lobby  Optional[XMLInteger] `json:"lobby,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunBarrel struct {
	Controller Optional[string] `json:"controller,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunFeeder struct {
	Controller Optional[string] `json:"controller,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunLoader struct {
	Controller Optional[string] `json:"controller,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0Overrides struct {
	AimingTimeFactor         Optional[XMLInteger] `json:"aimingTimeFactor,omitzero"`
	BurstSize                Optional[XMLInteger] `json:"burstSize,omitzero"`
	Capacity                 Optional[XMLInteger] `json:"capacity,omitzero"`
	LockGunTime              Optional[XMLInteger] `json:"lockGunTime,omitzero"`
	MovementDispersion       Optional[XMLInteger] `json:"movementDispersion,omitzero"`
	ReloadTimeFactor         Optional[XMLInteger] `json:"reloadTimeFactor,omitzero"`
	ShotDispersion           Optional[XMLInteger] `json:"shotDispersion,omitzero"`
	SwitchCooldown           Optional[XMLInteger] `json:"switchCooldown,omitzero"`
	TurretRotationDispersion Optional[XMLInteger] `json:"turretRotationDispersion,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0 struct {
	GunBarrel Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunBarrel] `json:"gunBarrel,omitzero"`
	GunFeeder Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunFeeder] `json:"gunFeeder,omitzero"`
	GunLoader Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0GunLoader] `json:"gunLoader,omitzero"`
	Overrides Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0Overrides] `json:"overrides,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunBarrel struct {
	Controller Optional[string] `json:"controller,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunFeeder struct {
	Controller Optional[string] `json:"controller,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunLoader struct {
	Controller Optional[string] `json:"controller,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1Overrides struct {
	AimingTimeFactor         Optional[XMLNumber]  `json:"aimingTimeFactor,omitzero"`
	BuckshotSize             Optional[XMLInteger] `json:"buckshotSize,omitzero"`
	Capacity                 Optional[XMLInteger] `json:"capacity,omitzero"`
	LockGunTime              Optional[XMLInteger] `json:"lockGunTime,omitzero"`
	MovementDispersion       Optional[XMLNumber]  `json:"movementDispersion,omitzero"`
	ReloadTimeFactor         Optional[XMLNumber]  `json:"reloadTimeFactor,omitzero"`
	ShotDispersion           Optional[XMLNumber]  `json:"shotDispersion,omitzero"`
	SwitchCooldown           Optional[XMLInteger] `json:"switchCooldown,omitzero"`
	TurretRotationDispersion Optional[XMLNumber]  `json:"turretRotationDispersion,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1 struct {
	GunBarrel Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunBarrel] `json:"gunBarrel,omitzero"`
	GunFeeder Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunFeeder] `json:"gunFeeder,omitzero"`
	GunLoader Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1GunLoader] `json:"gunLoader,omitzero"`
	Overrides Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1Overrides] `json:"overrides,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcher struct {
	Setting0 Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting0] `json:"setting0,omitzero"`
	Setting1 Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcherSetting1] `json:"setting1,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesHitTester struct {
	CollisionModel Optional[string] `json:"collisionModel,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesModels struct {
	Destroyed Optional[string] `json:"destroyed,omitzero"`
	Exploded  Optional[string] `json:"exploded,omitzero"`
	Undamaged Optional[string] `json:"undamaged,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesRecoil struct {
	Amplitude   Optional[XMLNumber] `json:"amplitude,omitzero"`
	BackoffTime Optional[XMLNumber] `json:"backoffTime,omitzero"`
	LodDist     Optional[string]    `json:"lodDist,omitzero"`
	ReturnTime  Optional[XMLNumber] `json:"returnTime,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotDispersionFactors struct {
	AfterShot       Optional[XMLNumber] `json:"afterShot,omitzero"`
	TurretRotation  Optional[XMLNumber] `json:"turretRotation,omitzero"`
	WhileGunDamaged Optional[XMLNumber] `json:"whileGunDamaged,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryPropertiesShellPrice struct {
	Text Optional[XMLInteger] `json:"#text,omitzero"`
	Gold Optional[string]     `json:"gold,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryPropertiesShell struct {
	Text  Optional[string]                                                                                                                       `json:"#text,omitzero"`
	Price Optional[XMLScalarObject[XMLInteger, StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryPropertiesShellPrice]] `json:"price,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryProperties struct {
	Shell Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryPropertiesShell] `json:"shell,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntry struct {
	Name       Optional[string]                                                                                `json:"name,omitzero" schema:"required,nonnull"`
	Properties Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntryProperties] `json:"properties,omitzero" schema:"required"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksGun struct {
	Text Optional[string] `json:"#text,omitzero"`
	Cost Optional[string] `json:"cost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksGunEntry struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksTurret struct {
	Text Optional[string] `json:"#text,omitzero"`
	Cost Optional[string] `json:"cost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksVehicle struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksVehicleEntry struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocks struct {
	Gun     Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksGun, StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksGunEntry]]         `json:"gun,omitzero"`
	Turret  Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksTurret]                                                                                                       `json:"turret,omitzero"`
	Vehicle Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksVehicle, StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocksVehicleEntry]] `json:"vehicle,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntryProperties struct {
	Text                     Optional[string]                                                                                                                            `json:"#text,omitzero"`
	NamedShots               map[string]Optional[StrictTankConfigurationNamedShot]                                                                                       `json:"-"`
	AimingTime               Optional[XMLNumber]                                                                                                                         `json:"aimingTime,omitzero"`
	AnimateEmblemSlots       Optional[XMLBool]                                                                                                                           `json:"animateEmblemSlots,omitzero"`
	Armor                    Optional[map[string]Optional[XMLScalarObject[XMLNumber, StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesArmorArmorValue]]] `json:"armor,omitzero"`
	Burst                    Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesBurst]                                                            `json:"burst,omitzero"`
	Camouflage               Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesCamouflage]                                                       `json:"camouflage,omitzero"`
	Clip                     Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesClip]                                                             `json:"clip,omitzero"`
	Effects                  Optional[string]                                                                                                                            `json:"effects,omitzero"`
	EmblemSlots              Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesEmblemSlots]                                                      `json:"emblemSlots,omitzero"`
	ExtraPitchLimits         Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtraPitchLimits]                                                 `json:"extraPitchLimits,omitzero"`
	Extras                   Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesExtras]                                                           `json:"extras,omitzero"`
	FakeTurrets              Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesFakeTurrets]                                                      `json:"fakeTurrets,omitzero"`
	FitsRatingBattle         Optional[XMLBool]                                                                                                                           `json:"fitsRatingBattle,omitzero"`
	GunSwitcher              Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesGunSwitcher]                                                      `json:"gunSwitcher,omitzero"`
	HitTester                Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesHitTester]                                                        `json:"hitTester,omitzero"`
	InvisibilityFactorAtShot Optional[XMLScalarList[XMLNumber, XMLNumber]]                                                                                               `json:"invisibilityFactorAtShot,omitzero"`
	MaxAmmo                  Optional[XMLScalarList[XMLInteger, XMLInteger]]                                                                                             `json:"maxAmmo,omitzero"`
	Models                   Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesModels]                                                           `json:"models,omitzero"`
	PitchLimits              Optional[XMLScalarList[XMLNumberVector, XMLNumberVector]]                                                                                   `json:"pitchLimits,omitzero"`
	ProjectileSpawnShift     Optional[map[string]Optional[XMLNumberVector]]                                                                                              `json:"projectileSpawnShift,omitzero"`
	PumpGunMode              Optional[XMLBool]                                                                                                                           `json:"pumpGunMode,omitzero"`
	PumpGunReloadTimes       Optional[XMLNumberVector]                                                                                                                   `json:"pumpGunReloadTimes,omitzero"`
	Recoil                   Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesRecoil]                                                           `json:"recoil,omitzero"`
	ReloadEffect             Optional[string]                                                                                                                            `json:"reloadEffect,omitzero"`
	ReloadTime               Optional[XMLNumber]                                                                                                                         `json:"reloadTime,omitzero"`
	ShotDispersionFactors    Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotDispersionFactors]                                            `json:"shotDispersionFactors,omitzero"`
	ShotDispersionRadius     Optional[XMLNumber]                                                                                                                         `json:"shotDispersionRadius,omitzero"`
	Shots                    Optional[[]Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesShotsEntry]]                                           `json:"shots,omitzero"`
	TurretYawLimits          Optional[XMLScalarList[XMLNumberVector, XMLNumberVector]]                                                                                   `json:"turretYawLimits,omitzero"`
	Unlocks                  Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryPropertiesUnlocks]                                                          `json:"unlocks,omitzero"`
	YawLimits                Optional[XMLNumberVector]                                                                                                                   `json:"yawLimits,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesGunsEntry struct {
	Name       Optional[string]                                                            `json:"name,omitzero" schema:"required,nonnull"`
	Properties Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntryProperties] `json:"properties,omitzero" schema:"required"`
}

type StrictTankConfigurationTurrets0EntryPropertiesHitTester struct {
	CollisionModel Optional[string] `json:"collisionModel,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesModels struct {
	Destroyed Optional[string]                        `json:"destroyed,omitzero"`
	Exploded  Optional[XMLScalarList[string, string]] `json:"exploded,omitzero"`
	Undamaged Optional[string]                        `json:"undamaged,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesSurveyingDeviceHealth struct {
	MaxHealth      Optional[XMLInteger] `json:"maxHealth,omitzero"`
	MaxRegenHealth Optional[XMLInteger] `json:"maxRegenHealth,omitzero"`
	RepairCost     Optional[XMLNumber]  `json:"repairCost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesTurretRotatorHealth struct {
	MaxHealth      Optional[XMLInteger] `json:"maxHealth,omitzero"`
	MaxRegenHealth Optional[XMLInteger] `json:"maxRegenHealth,omitzero"`
	RepairCost     Optional[XMLNumber]  `json:"repairCost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocksEngine struct {
	Text Optional[string] `json:"#text,omitzero"`
	Cost Optional[string] `json:"cost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocksGun struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocksGunEntry struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocksTurret struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocksVehicle struct {
	Text Optional[string]     `json:"#text,omitzero"`
	Cost Optional[XMLInteger] `json:"cost,omitzero"`
}

type StrictTankConfigurationTurrets0EntryPropertiesUnlocks struct {
	Engine  Optional[StrictTankConfigurationTurrets0EntryPropertiesUnlocksEngine]                                                                            `json:"engine,omitzero"`
	Gun     Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesUnlocksGun, StrictTankConfigurationTurrets0EntryPropertiesUnlocksGunEntry]] `json:"gun,omitzero"`
	Turret  Optional[StrictTankConfigurationTurrets0EntryPropertiesUnlocksTurret]                                                                            `json:"turret,omitzero"`
	Vehicle Optional[StrictTankConfigurationTurrets0EntryPropertiesUnlocksVehicle]                                                                           `json:"vehicle,omitzero"`
}

type StrictTankConfigurationTurrets0EntryProperties struct {
	AODecals                   Optional[string]                                                                                                                        `json:"AODecals,omitzero"`
	Armor                      Optional[map[string]Optional[XMLScalarObjectList[XMLNumber, StrictTankConfigurationTurrets0EntryPropertiesArmorArmorValue, XMLNumber]]] `json:"armor,omitzero"`
	Camouflage                 Optional[StrictTankConfigurationTurrets0EntryPropertiesCamouflage]                                                                      `json:"camouflage,omitzero"`
	Ceilless                   Optional[XMLBool]                                                                                                                       `json:"ceilless,omitzero"`
	CircularVisionRadius       Optional[XMLInteger]                                                                                                                    `json:"circularVisionRadius,omitzero"`
	Description                Optional[XMLLocalizationKey]                                                                                                            `json:"description,omitzero"`
	EmblemSlots                Optional[XMLObjectList[StrictTankConfigurationTurrets0EntryPropertiesEmblemSlots, string]]                                              `json:"emblemSlots,omitzero"`
	FitsRatingBattle           Optional[XMLBool]                                                                                                                       `json:"fitsRatingBattle,omitzero"`
	GunPosition                Optional[XMLScalarList[XMLNumberVector, XMLNumberVector]]                                                                               `json:"gunPosition,omitzero"`
	Guns                       Optional[[]Optional[StrictTankConfigurationTurrets0EntryPropertiesGunsEntry]]                                                           `json:"guns,omitzero"`
	HitTester                  Optional[StrictTankConfigurationTurrets0EntryPropertiesHitTester]                                                                       `json:"hitTester,omitzero"`
	Icon                       Optional[string]                                                                                                                        `json:"icon,omitzero"`
	InvisibilityFactor         Optional[XMLNumber]                                                                                                                     `json:"invisibilityFactor,omitzero"`
	Level                      Optional[XMLInteger]                                                                                                                    `json:"level,omitzero"`
	MaxHealth                  Optional[XMLInteger]                                                                                                                    `json:"maxHealth,omitzero"`
	Models                     Optional[StrictTankConfigurationTurrets0EntryPropertiesModels]                                                                          `json:"models,omitzero"`
	NotInShop                  Optional[XMLBool]                                                                                                                       `json:"notInShop,omitzero"`
	PhysicsShape               Optional[XMLNumberVector]                                                                                                               `json:"physicsShape,omitzero"`
	Price                      Optional[XMLNumber]                                                                                                                     `json:"price,omitzero"`
	PrimaryArmor               Optional[string]                                                                                                                        `json:"primaryArmor,omitzero"`
	RotationSpeed              Optional[XMLScalarList[XMLNumber, XMLNumber]]                                                                                           `json:"rotationSpeed,omitzero"`
	ShortUserString            Optional[XMLLocalizationKey]                                                                                                            `json:"shortUserString,omitzero"`
	ShowEmblemsOnGun           Optional[XMLBool]                                                                                                                       `json:"showEmblemsOnGun,omitzero"`
	SurveyingDeviceHealth      Optional[StrictTankConfigurationTurrets0EntryPropertiesSurveyingDeviceHealth]                                                           `json:"surveyingDeviceHealth,omitzero"`
	Tags                       Optional[string]                                                                                                                        `json:"tags,omitzero"`
	TurretRotatorHealth        Optional[StrictTankConfigurationTurrets0EntryPropertiesTurretRotatorHealth]                                                             `json:"turretRotatorHealth,omitzero"`
	TurretRotatorSound         Optional[string]                                                                                                                        `json:"turretRotatorSound,omitzero"`
	TurretRotatorSoundGear     Optional[string]                                                                                                                        `json:"turretRotatorSoundGear,omitzero"`
	TurretRotatorSoundManual   Optional[string]                                                                                                                        `json:"turretRotatorSoundManual,omitzero"`
	Unlocks                    Optional[StrictTankConfigurationTurrets0EntryPropertiesUnlocks]                                                                         `json:"unlocks,omitzero"`
	UserString                 Optional[XMLLocalizationKey]                                                                                                            `json:"userString,omitzero"`
	Weight                     Optional[XMLNumber]                                                                                                                     `json:"weight,omitzero"`
	WwturretRotatorSoundManual Optional[string]                                                                                                                        `json:"wwturretRotatorSoundManual,omitzero"`
	YawLimits                  Optional[XMLScalarList[XMLNumberVector, XMLNumberVector]]                                                                               `json:"yawLimits,omitzero"`
}

type StrictTankConfigurationTurrets0Entry struct {
	Name       Optional[string]                                         `json:"name,omitzero" schema:"required,nonnull"`
	Properties Optional[StrictTankConfigurationTurrets0EntryProperties] `json:"properties,omitzero" schema:"required"`
}

type StrictTankConfiguration struct {
	AttrExcept             Optional[string]                                            `json:"@except,omitzero"`
	Camouflage             Optional[StrictTankConfigurationCamouflage]                 `json:"camouflage,omitzero"`
	Chassis                Optional[[]Optional[StrictTankConfigurationChassisEntry]]   `json:"chassis,omitzero"`
	ConsumableSlots        Optional[XMLInteger]                                        `json:"consumableSlots,omitzero"`
	Crew                   Optional[StrictTankConfigurationCrew]                       `json:"crew,omitzero"`
	CrewXpFactor           Optional[XMLNumber]                                         `json:"crewXpFactor,omitzero"`
	CustomizationNation    Optional[string]                                            `json:"customizationNation,omitzero"`
	Effects                Optional[StrictTankConfigurationEffects]                    `json:"effects,omitzero"`
	Emblems                Optional[StrictTankConfigurationEmblems]                    `json:"emblems,omitzero"`
	Engines                Optional[[]Optional[StrictTankConfigurationEnginesEntry]]   `json:"engines,omitzero"`
	EnrichmentBoost        Optional[XMLNumber]                                         `json:"enrichmentBoost,omitzero"`
	Extras                 Optional[StrictTankConfigurationExtras]                     `json:"extras,omitzero"`
	FuelTanks              Optional[[]Optional[StrictTankConfigurationFuelTanksEntry]] `json:"fuelTanks,omitzero"`
	Horns                  Optional[StrictTankConfigurationHorns]                      `json:"horns,omitzero"`
	Hull                   Optional[StrictTankConfigurationHull]                       `json:"hull,omitzero"`
	Invisibility           Optional[StrictTankConfigurationInvisibility]               `json:"invisibility,omitzero"`
	IsUnlocked             Optional[XMLBool]                                           `json:"isUnlocked,omitzero"`
	MaxMastery             Optional[XMLBool]                                           `json:"maxMastery,omitzero"`
	OptDevicePreset        Optional[string]                                            `json:"optDevicePreset,omitzero"`
	PremiumVehicleXPFactor Optional[XMLNumber]                                         `json:"premiumVehicleXPFactor,omitzero"`
	ProvisionSlots         Optional[XMLInteger]                                        `json:"provisionSlots,omitzero"`
	Radios                 Optional[[]Optional[StrictTankConfigurationRadiosEntry]]    `json:"radios,omitzero"`
	RepairCost             Optional[XMLNumber]                                         `json:"repairCost,omitzero"`
	SpeedLimits            Optional[StrictTankConfigurationSpeedLimits]                `json:"speedLimits,omitzero"`
	Subnation              Optional[string]                                            `json:"subnation,omitzero"`
	TurretBlowEnabled      Optional[XMLBool]                                           `json:"turretBlowEnabled,omitzero"`
	Turrets0               Optional[[]Optional[StrictTankConfigurationTurrets0Entry]]  `json:"turrets0,omitzero"`
}
