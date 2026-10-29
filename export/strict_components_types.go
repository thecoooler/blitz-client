package export

// StrictNationComponents describes the seven installed component documents.
type StrictNationComponents struct {
	Chassis   Optional[StrictComponentDocument[StrictComponentChassisShared]]   `json:"chassis,omitzero"`
	Engines   Optional[StrictComponentDocument[StrictComponentEnginesShared]]   `json:"engines,omitzero"`
	FuelTanks Optional[StrictComponentDocument[StrictComponentFuelTanksShared]] `json:"fuelTanks,omitzero"`
	Guns      Optional[StrictComponentDocument[StrictComponentGunsShared]]      `json:"guns,omitzero"`
	Radios    Optional[StrictComponentDocument[StrictComponentRadiosShared]]    `json:"radios,omitzero"`
	Shells    Optional[StrictShellDocument]                                     `json:"shells,omitzero"`
	Turrets   Optional[StrictComponentDocument[StrictComponentTurretsShared]]   `json:"turrets,omitzero"`
}

// StrictComponentDocument retains the shared definitions and their ID table.
type StrictComponentDocument[T any] struct {
	Ids             Optional[map[string]Optional[XMLInteger]] `json:"ids,omitzero"`
	NextAvailableId Optional[XMLInteger]                      `json:"nextAvailableId,omitzero"`
	Shared          Optional[map[string]Optional[T]]          `json:"shared,omitzero"`
}

type StrictComponentChassisShared struct {
	Armor                  Optional[StrictComponentChassisSharedArmor]                 `json:"armor,omitzero"`
	Blitz                  Optional[StrictComponentChassisSharedBlitz]                 `json:"blitz,omitzero"`
	BrakeForce             Optional[XMLInteger]                                        `json:"brakeForce,omitzero"`
	BulkHealthFactor       Optional[XMLNumber]                                         `json:"bulkHealthFactor,omitzero"`
	Deprecated             Optional[string]                                            `json:"deprecated,omitzero"`
	Description            Optional[XMLLocalizationKey]                                `json:"description,omitzero"`
	DrivingWheels          Optional[string]                                            `json:"drivingWheels,omitzero"`
	Effects                Optional[StrictComponentChassisSharedEffects]               `json:"effects,omitzero"`
	HitTester              Optional[StrictComponentChassisSharedHitTester]             `json:"hitTester,omitzero"`
	HullPosition           Optional[XMLNumberVector]                                   `json:"hullPosition,omitzero"`
	Level                  Optional[XMLInteger]                                        `json:"level,omitzero"`
	MaxClimbAngle          Optional[XMLInteger]                                        `json:"maxClimbAngle,omitzero"`
	MaxHealth              Optional[XMLInteger]                                        `json:"maxHealth,omitzero"`
	MaxLoad                Optional[XMLInteger]                                        `json:"maxLoad,omitzero"`
	MaxRegenHealth         Optional[XMLInteger]                                        `json:"maxRegenHealth,omitzero"`
	Models                 Optional[StrictComponentChassisSharedModels]                `json:"models,omitzero"`
	NavmeshGirth           Optional[XMLNumber]                                         `json:"navmeshGirth,omitzero"`
	Price                  Optional[XMLInteger]                                        `json:"price,omitzero"`
	RepairCost             Optional[XMLNumber]                                         `json:"repairCost,omitzero"`
	RotationIsAroundCenter Optional[XMLBool]                                           `json:"rotationIsAroundCenter,omitzero"`
	RotationSpeed          Optional[XMLInteger]                                        `json:"rotationSpeed,omitzero"`
	ShotDispersionFactors  Optional[StrictComponentChassisSharedShotDispersionFactors] `json:"shotDispersionFactors,omitzero"`
	Sound                  Optional[string]                                            `json:"sound,omitzero"`
	Tags                   Optional[string]                                            `json:"tags,omitzero"`
	TerrainResistance      Optional[XMLNumberVector]                                   `json:"terrainResistance,omitzero"`
	TopRightCarryingPoint  Optional[XMLNumberVector]                                   `json:"topRightCarryingPoint,omitzero"`
	Traces                 Optional[StrictComponentChassisSharedTraces]                `json:"traces,omitzero"`
	Tracks                 Optional[StrictComponentChassisSharedTracks]                `json:"tracks,omitzero"`
	Unlocks                Optional[StrictComponentChassisSharedUnlocks]               `json:"unlocks,omitzero"`
	UserString             Optional[XMLLocalizationKey]                                `json:"userString,omitzero"`
	Weight                 Optional[XMLInteger]                                        `json:"weight,omitzero"`
	Wheels                 Optional[StrictComponentChassisSharedWheels]                `json:"wheels,omitzero"`
}

type StrictComponentChassisSharedArmor struct {
	LeftTrack  Optional[XMLInteger] `json:"leftTrack,omitzero"`
	RightTrack Optional[XMLInteger] `json:"rightTrack,omitzero"`
}

type StrictComponentChassisSharedBlitz struct {
	Bbox          Optional[StrictComponentChassisSharedBlitzBbox] `json:"bbox,omitzero"`
	BlitzPartName Optional[string]                                `json:"blitzPartName,omitzero"`
	SoundNPC      Optional[string]                                `json:"soundNPC,omitzero"`
	SoundPC       Optional[string]                                `json:"soundPC,omitzero"`
}

type StrictComponentChassisSharedBlitzBbox struct {
	Max Optional[XMLNumberVector] `json:"max,omitzero"`
	Min Optional[XMLNumberVector] `json:"min,omitzero"`
}

type StrictComponentChassisSharedEffects struct {
	Dust    Optional[string] `json:"dust,omitzero"`
	LodDist Optional[string] `json:"lodDist,omitzero"`
	Mud     Optional[string] `json:"mud,omitzero"`
}

type StrictComponentChassisSharedHitTester struct {
	CollisionModel Optional[string] `json:"collisionModel,omitzero"`
}

type StrictComponentChassisSharedModels struct {
	Destroyed Optional[string] `json:"destroyed,omitzero"`
	Exploded  Optional[string] `json:"exploded,omitzero"`
	Undamaged Optional[string] `json:"undamaged,omitzero"`
}

type StrictComponentChassisSharedShotDispersionFactors struct {
	VehicleMovement Optional[XMLNumber] `json:"vehicleMovement,omitzero"`
	VehicleRotation Optional[XMLNumber] `json:"vehicleRotation,omitzero"`
}

type StrictComponentChassisSharedTraces struct {
	BufferPrefs Optional[string]          `json:"bufferPrefs,omitzero"`
	LodDist     Optional[string]          `json:"lodDist,omitzero"`
	Size        Optional[XMLNumberVector] `json:"size,omitzero"`
	TextureSet  Optional[string]          `json:"textureSet,omitzero"`
}

type StrictComponentChassisSharedTracks struct {
	LeftMaterial  Optional[string] `json:"leftMaterial,omitzero"`
	LodDist       Optional[string] `json:"lodDist,omitzero"`
	RightMaterial Optional[string] `json:"rightMaterial,omitzero"`
}

type StrictComponentChassisSharedUnlocks struct {
	Chassis Optional[StrictComponentChassisSharedUnlocksChassis] `json:"chassis,omitzero"`
}

type StrictComponentChassisSharedUnlocksChassis struct {
	Items Optional[[]Optional[StrictComponentChassisSharedUnlocksChassisItemsEntry]] `json:"#items,omitzero"`
	Text  Optional[string]                                                           `json:"#text,omitzero"`
}

type StrictComponentChassisSharedUnlocksChassisItemsEntry struct {
	Name       Optional[string]     `json:"name,omitzero"`
	Properties Optional[XMLInteger] `json:"properties,omitzero"`
}

type StrictComponentChassisSharedWheels struct {
	Group   Optional[[]Optional[StrictComponentChassisSharedWheelsGroupEntry]] `json:"group,omitzero"`
	LodDist Optional[string]                                                   `json:"lodDist,omitzero"`
	Wheel   Optional[[]Optional[StrictComponentChassisSharedWheelsWheelEntry]] `json:"wheel,omitzero"`
}

type StrictComponentChassisSharedWheelsGroupEntry struct {
	Count      Optional[XMLInteger] `json:"count,omitzero"`
	IsLeft     Optional[XMLBool]    `json:"isLeft,omitzero"`
	Radius     Optional[XMLNumber]  `json:"radius,omitzero"`
	StartIndex Optional[XMLInteger] `json:"startIndex,omitzero"`
	Template   Optional[string]     `json:"template,omitzero"`
}

type StrictComponentChassisSharedWheelsWheelEntry struct {
	IsLeft Optional[XMLBool]   `json:"isLeft,omitzero"`
	Name   Optional[string]    `json:"name,omitzero"`
	Radius Optional[XMLNumber] `json:"radius,omitzero"`
}

type StrictComponentEnginesShared struct {
	Deprecated         Optional[string]                          `json:"deprecated,omitzero"`
	Description        Optional[XMLLocalizationKey]              `json:"description,omitzero"`
	FireStartingChance Optional[XMLNumber]                       `json:"fireStartingChance,omitzero"`
	Level              Optional[XMLInteger]                      `json:"level,omitzero"`
	MaxHealth          Optional[XMLInteger]                      `json:"maxHealth,omitzero"`
	MaxRegenHealth     Optional[XMLInteger]                      `json:"maxRegenHealth,omitzero"`
	NotInShop          Optional[XMLScalarList[XMLBool, XMLBool]] `json:"notInShop,omitzero"`
	Power              Optional[XMLInteger]                      `json:"power,omitzero"`
	Price              Optional[XMLInteger]                      `json:"price,omitzero"`
	RepairCost         Optional[XMLInteger]                      `json:"repairCost,omitzero"`
	RpmMax             Optional[XMLInteger]                      `json:"rpm_max,omitzero"`
	RpmMin             Optional[XMLInteger]                      `json:"rpm_min,omitzero"`
	ShortUserString    Optional[XMLLocalizationKey]              `json:"shortUserString,omitzero"`
	Sound              Optional[string]                          `json:"sound,omitzero"`
	SoundNPC           Optional[string]                          `json:"soundNPC,omitzero"`
	SoundPC            Optional[string]                          `json:"soundPC,omitzero"`
	Tags               Optional[XMLScalarList[string, string]]   `json:"tags,omitzero"`
	UserString         Optional[XMLLocalizationKey]              `json:"userString,omitzero"`
	Weight             Optional[XMLInteger]                      `json:"weight,omitzero"`
	WwsoundNPC         Optional[XMLScalarList[string, string]]   `json:"wwsoundNPC,omitzero"`
	WwsoundPC          Optional[XMLScalarList[string, string]]   `json:"wwsoundPC,omitzero"`
}

type StrictComponentFuelTanksShared struct {
	Description    Optional[XMLLocalizationKey] `json:"description,omitzero"`
	MaxHealth      Optional[XMLInteger]         `json:"maxHealth,omitzero"`
	MaxRegenHealth Optional[XMLInteger]         `json:"maxRegenHealth,omitzero"`
	Price          Optional[XMLInteger]         `json:"price,omitzero"`
	RepairCost     Optional[XMLInteger]         `json:"repairCost,omitzero"`
	Tags           Optional[string]             `json:"tags,omitzero"`
	UserString     Optional[XMLLocalizationKey] `json:"userString,omitzero"`
	Weight         Optional[XMLInteger]         `json:"weight,omitzero"`
}

type StrictComponentGunsShared struct {
	AimingTime               Optional[XMLNumber]                                       `json:"aimingTime,omitzero"`
	Burst                    Optional[StrictComponentGunsSharedBurst]                  `json:"burst,omitzero"`
	Clip                     Optional[StrictComponentGunsSharedClip]                   `json:"clip,omitzero"`
	Description              Optional[XMLLocalizationKey]                              `json:"description,omitzero"`
	Effects                  Optional[string]                                          `json:"effects,omitzero"`
	ExtraPitchLimits         Optional[StrictComponentGunsSharedExtraPitchLimits]       `json:"extraPitchLimits,omitzero"`
	Impulse                  Optional[XMLNumber]                                       `json:"impulse,omitzero"`
	InvisibilityFactorAtShot Optional[XMLNumber]                                       `json:"invisibilityFactorAtShot,omitzero"`
	Level                    Optional[XMLInteger]                                      `json:"level,omitzero"`
	MaxAmmo                  Optional[XMLInteger]                                      `json:"maxAmmo,omitzero"`
	MaxHealth                Optional[XMLInteger]                                      `json:"maxHealth,omitzero"`
	MaxRegenHealth           Optional[XMLInteger]                                      `json:"maxRegenHealth,omitzero"`
	NotInShop                Optional[XMLBool]                                         `json:"notInShop,omitzero"`
	PitchLimits              Optional[XMLNumberVector]                                 `json:"pitchLimits,omitzero"`
	Price                    Optional[XMLInteger]                                      `json:"price,omitzero"`
	Recoil                   Optional[StrictComponentGunsSharedRecoil]                 `json:"recoil,omitzero"`
	ReloadEffect             Optional[string]                                          `json:"reloadEffect,omitzero"`
	ReloadTime               Optional[XMLNumber]                                       `json:"reloadTime,omitzero"`
	RepairCost               Optional[XMLNumber]                                       `json:"repairCost,omitzero"`
	RotationSpeed            Optional[XMLNumber]                                       `json:"rotationSpeed,omitzero"`
	ShortUserString          Optional[XMLLocalizationKey]                              `json:"shortUserString,omitzero"`
	ShotDispersionFactors    Optional[StrictComponentGunsSharedShotDispersionFactors]  `json:"shotDispersionFactors,omitzero"`
	ShotDispersionRadius     Optional[XMLNumber]                                       `json:"shotDispersionRadius,omitzero"`
	Shots                    Optional[[]Optional[StrictComponentGunsSharedShotsEntry]] `json:"shots,omitzero"`
	Tags                     Optional[XMLScalarList[string, string]]                   `json:"tags,omitzero"`
	TurretYawLimits          Optional[XMLScalarList[XMLNumberVector, XMLNumberVector]] `json:"turretYawLimits,omitzero"`
	UserString               Optional[XMLLocalizationKey]                              `json:"userString,omitzero"`
	Weight                   Optional[XMLInteger]                                      `json:"weight,omitzero"`
	YawLimits                Optional[XMLNumberVector]                                 `json:"yawLimits,omitzero"`
}

type StrictComponentGunsSharedBurst struct {
	Count Optional[XMLInteger] `json:"count,omitzero"`
	Rate  Optional[XMLInteger] `json:"rate,omitzero"`
}

type StrictComponentGunsSharedClip struct {
	Count Optional[XMLInteger] `json:"count,omitzero"`
	Rate  Optional[XMLInteger] `json:"rate,omitzero"`
}

type StrictComponentGunsSharedExtraPitchLimits struct {
	Back Optional[XMLNumberVector] `json:"back,omitzero"`
}

type StrictComponentGunsSharedRecoil struct {
	Amplitude    Optional[XMLNumber] `json:"amplitude,omitzero"`
	BackoffTime  Optional[XMLNumber] `json:"backoffTime,omitzero"`
	LodDist      Optional[string]    `json:"lodDist,omitzero"`
	RecoilEffect Optional[string]    `json:"recoilEffect,omitzero"`
	ReturnTime   Optional[XMLNumber] `json:"returnTime,omitzero"`
}

type StrictComponentGunsSharedShotDispersionFactors struct {
	AfterShot       Optional[XMLNumber] `json:"afterShot,omitzero"`
	TurretRotation  Optional[XMLNumber] `json:"turretRotation,omitzero"`
	WhileGunDamaged Optional[XMLNumber] `json:"whileGunDamaged,omitzero"`
}

type StrictComponentGunsSharedShotsEntry struct {
	Name       Optional[string]                                        `json:"name,omitzero"`
	Properties Optional[StrictComponentGunsSharedShotsEntryProperties] `json:"properties,omitzero"`
}

type StrictComponentGunsSharedShotsEntryProperties struct {
	Acceleration           Optional[XMLNumber]       `json:"acceleration,omitzero"`
	DefaultPortion         Optional[XMLNumber]       `json:"defaultPortion,omitzero"`
	Gravity                Optional[XMLNumber]       `json:"gravity,omitzero"`
	MaxDistance            Optional[XMLInteger]      `json:"maxDistance,omitzero"`
	MaxSpeed               Optional[XMLInteger]      `json:"maxSpeed,omitzero"`
	PiercingPower          Optional[XMLNumberVector] `json:"piercingPower,omitzero"`
	RotationalDeceleration Optional[XMLInteger]      `json:"rotationalDeceleration,omitzero"`
	Speed                  Optional[XMLInteger]      `json:"speed,omitzero"`
	TurnSpeedHorizontal    Optional[XMLNumber]       `json:"turnSpeedHorizontal,omitzero"`
	TurnSpeedVertical      Optional[XMLNumber]       `json:"turnSpeedVertical,omitzero"`
}

type StrictComponentRadiosShared struct {
	Description    Optional[XMLLocalizationKey] `json:"description,omitzero"`
	Distance       Optional[XMLInteger]         `json:"distance,omitzero"`
	Level          Optional[XMLInteger]         `json:"level,omitzero"`
	MaxHealth      Optional[XMLInteger]         `json:"maxHealth,omitzero"`
	MaxRegenHealth Optional[XMLInteger]         `json:"maxRegenHealth,omitzero"`
	NotInShop      Optional[XMLBool]            `json:"notInShop,omitzero"`
	Price          Optional[XMLInteger]         `json:"price,omitzero"`
	RepairCost     Optional[XMLNumber]          `json:"repairCost,omitzero"`
	Tags           Optional[string]             `json:"tags,omitzero"`
	UserString     Optional[XMLLocalizationKey] `json:"userString,omitzero"`
	Weight         Optional[XMLInteger]         `json:"weight,omitzero"`
}

type StrictComponentTurretsShared struct {
}

type StrictShell struct {
	Blitz                             Optional[StrictShellBlitz]                                    `json:"blitz,omitzero"`
	Caliber                           Optional[XMLNumber]                                           `json:"caliber,omitzero"`
	Damage                            Optional[StrictShellDamage]                                   `json:"damage,omitzero"`
	Description                       Optional[XMLLocalizationKey]                                  `json:"description,omitzero"`
	Effects                           Optional[string]                                              `json:"effects,omitzero"`
	ExplosionRadius                   Optional[XMLNumber]                                           `json:"explosionRadius,omitzero"`
	Icon                              Optional[string]                                              `json:"icon,omitzero"`
	Id                                Optional[XMLInteger]                                          `json:"id,omitzero"`
	Improved                          Optional[XMLBool]                                             `json:"improved,omitzero"`
	IsTracer                          Optional[XMLBool]                                             `json:"isTracer,omitzero"`
	Kind                              Optional[string]                                              `json:"kind,omitzero"`
	NormalizationAngle                Optional[XMLInteger]                                          `json:"normalizationAngle,omitzero"`
	NotInShop                         Optional[XMLBool]                                             `json:"notInShop,omitzero"`
	PiercingPowerLossFactorByDistance Optional[XMLNumber]                                           `json:"piercingPowerLossFactorByDistance,omitzero"`
	Price                             Optional[XMLScalarObject[XMLInteger, StrictShellPriceObject]] `json:"price,omitzero"`
	ProjectileType                    Optional[string]                                              `json:"projectileType,omitzero"`
	RicochetAngle                     Optional[XMLInteger]                                          `json:"ricochetAngle,omitzero"`
	UserString                        Optional[XMLLocalizationKey]                                  `json:"userString,omitzero"`
}

type StrictShellBlitz struct {
	BlitzIconFrame Optional[XMLInteger] `json:"blitzIconFrame,omitzero"`
}

type StrictShellDamage struct {
	Armor   Optional[XMLInteger] `json:"armor,omitzero"`
	Devices Optional[XMLInteger] `json:"devices,omitzero"`
}

type StrictShellPriceObject struct {
	Text Optional[XMLInteger] `json:"#text,omitzero"`
	Gold Optional[string]     `json:"gold,omitzero"`
}
