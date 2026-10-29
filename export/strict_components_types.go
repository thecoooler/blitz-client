package export

// StrictNationComponents describes the seven installed component documents.
type StrictNationComponents struct {
	Chassis   Optional[StrictComponentDocument[StrictComponentChassisShared]]   `json:"chassis,omitempty"`
	Engines   Optional[StrictComponentDocument[StrictComponentEnginesShared]]   `json:"engines,omitempty"`
	FuelTanks Optional[StrictComponentDocument[StrictComponentFuelTanksShared]] `json:"fuelTanks,omitempty"`
	Guns      Optional[StrictComponentDocument[StrictComponentGunsShared]]      `json:"guns,omitempty"`
	Radios    Optional[StrictComponentDocument[StrictComponentRadiosShared]]    `json:"radios,omitempty"`
	Shells    Optional[StrictShellDocument]                                     `json:"shells,omitempty"`
	Turrets   Optional[StrictComponentDocument[StrictComponentTurretsShared]]   `json:"turrets,omitempty"`
}

// StrictComponentDocument retains the shared definitions and their ID table.
type StrictComponentDocument[T any] struct {
	Ids             Optional[map[string]Optional[string]] `json:"ids,omitempty"`
	NextAvailableId Optional[string]                      `json:"nextAvailableId,omitempty"`
	Shared          Optional[map[string]Optional[T]]      `json:"shared,omitempty"`
}

type StrictComponentChassisShared struct {
	Armor                  Optional[StrictComponentChassisSharedArmor]                 `json:"armor,omitempty"`
	Blitz                  Optional[StrictComponentChassisSharedBlitz]                 `json:"blitz,omitempty"`
	BrakeForce             Optional[string]                                            `json:"brakeForce,omitempty"`
	BulkHealthFactor       Optional[string]                                            `json:"bulkHealthFactor,omitempty"`
	Deprecated             Optional[string]                                            `json:"deprecated,omitempty"`
	Description            Optional[string]                                            `json:"description,omitempty"`
	DrivingWheels          Optional[string]                                            `json:"drivingWheels,omitempty"`
	Effects                Optional[StrictComponentChassisSharedEffects]               `json:"effects,omitempty"`
	HitTester              Optional[StrictComponentChassisSharedHitTester]             `json:"hitTester,omitempty"`
	HullPosition           Optional[string]                                            `json:"hullPosition,omitempty"`
	Level                  Optional[string]                                            `json:"level,omitempty"`
	MaxClimbAngle          Optional[string]                                            `json:"maxClimbAngle,omitempty"`
	MaxHealth              Optional[string]                                            `json:"maxHealth,omitempty"`
	MaxLoad                Optional[string]                                            `json:"maxLoad,omitempty"`
	MaxRegenHealth         Optional[string]                                            `json:"maxRegenHealth,omitempty"`
	Models                 Optional[StrictComponentChassisSharedModels]                `json:"models,omitempty"`
	NavmeshGirth           Optional[string]                                            `json:"navmeshGirth,omitempty"`
	Price                  Optional[string]                                            `json:"price,omitempty"`
	RepairCost             Optional[string]                                            `json:"repairCost,omitempty"`
	RotationIsAroundCenter Optional[string]                                            `json:"rotationIsAroundCenter,omitempty"`
	RotationSpeed          Optional[string]                                            `json:"rotationSpeed,omitempty"`
	ShotDispersionFactors  Optional[StrictComponentChassisSharedShotDispersionFactors] `json:"shotDispersionFactors,omitempty"`
	Sound                  Optional[string]                                            `json:"sound,omitempty"`
	Tags                   Optional[string]                                            `json:"tags,omitempty"`
	TerrainResistance      Optional[string]                                            `json:"terrainResistance,omitempty"`
	TopRightCarryingPoint  Optional[string]                                            `json:"topRightCarryingPoint,omitempty"`
	Traces                 Optional[StrictComponentChassisSharedTraces]                `json:"traces,omitempty"`
	Tracks                 Optional[StrictComponentChassisSharedTracks]                `json:"tracks,omitempty"`
	Unlocks                Optional[StrictComponentChassisSharedUnlocks]               `json:"unlocks,omitempty"`
	UserString             Optional[string]                                            `json:"userString,omitempty"`
	Weight                 Optional[string]                                            `json:"weight,omitempty"`
	Wheels                 Optional[StrictComponentChassisSharedWheels]                `json:"wheels,omitempty"`
}

type StrictComponentChassisSharedArmor struct {
	LeftTrack  Optional[string] `json:"leftTrack,omitempty"`
	RightTrack Optional[string] `json:"rightTrack,omitempty"`
}

type StrictComponentChassisSharedBlitz struct {
	Bbox          Optional[StrictComponentChassisSharedBlitzBbox] `json:"bbox,omitempty"`
	BlitzPartName Optional[string]                                `json:"blitzPartName,omitempty"`
	SoundNPC      Optional[string]                                `json:"soundNPC,omitempty"`
	SoundPC       Optional[string]                                `json:"soundPC,omitempty"`
}

type StrictComponentChassisSharedBlitzBbox struct {
	Max Optional[string] `json:"max,omitempty"`
	Min Optional[string] `json:"min,omitempty"`
}

type StrictComponentChassisSharedEffects struct {
	Dust    Optional[string] `json:"dust,omitempty"`
	LodDist Optional[string] `json:"lodDist,omitempty"`
	Mud     Optional[string] `json:"mud,omitempty"`
}

type StrictComponentChassisSharedHitTester struct {
	CollisionModel Optional[string] `json:"collisionModel,omitempty"`
}

type StrictComponentChassisSharedModels struct {
	Destroyed Optional[string] `json:"destroyed,omitempty"`
	Exploded  Optional[string] `json:"exploded,omitempty"`
	Undamaged Optional[string] `json:"undamaged,omitempty"`
}

type StrictComponentChassisSharedShotDispersionFactors struct {
	VehicleMovement Optional[string] `json:"vehicleMovement,omitempty"`
	VehicleRotation Optional[string] `json:"vehicleRotation,omitempty"`
}

type StrictComponentChassisSharedTraces struct {
	BufferPrefs Optional[string] `json:"bufferPrefs,omitempty"`
	LodDist     Optional[string] `json:"lodDist,omitempty"`
	Size        Optional[string] `json:"size,omitempty"`
	TextureSet  Optional[string] `json:"textureSet,omitempty"`
}

type StrictComponentChassisSharedTracks struct {
	LeftMaterial  Optional[string] `json:"leftMaterial,omitempty"`
	LodDist       Optional[string] `json:"lodDist,omitempty"`
	RightMaterial Optional[string] `json:"rightMaterial,omitempty"`
}

type StrictComponentChassisSharedUnlocks struct {
	Chassis Optional[StrictComponentChassisSharedUnlocksChassis] `json:"chassis,omitempty"`
}

type StrictComponentChassisSharedUnlocksChassis struct {
	Items Optional[[]Optional[StrictComponentChassisSharedUnlocksChassisItemsEntry]] `json:"#items,omitempty"`
	Text  Optional[string]                                                           `json:"#text,omitempty"`
}

type StrictComponentChassisSharedUnlocksChassisItemsEntry struct {
	Name       Optional[string] `json:"name,omitempty"`
	Properties Optional[string] `json:"properties,omitempty"`
}

type StrictComponentChassisSharedWheels struct {
	Group   Optional[[]Optional[StrictComponentChassisSharedWheelsGroupEntry]] `json:"group,omitempty"`
	LodDist Optional[string]                                                   `json:"lodDist,omitempty"`
	Wheel   Optional[[]Optional[StrictComponentChassisSharedWheelsWheelEntry]] `json:"wheel,omitempty"`
}

type StrictComponentChassisSharedWheelsGroupEntry struct {
	Count      Optional[string] `json:"count,omitempty"`
	IsLeft     Optional[string] `json:"isLeft,omitempty"`
	Radius     Optional[string] `json:"radius,omitempty"`
	StartIndex Optional[string] `json:"startIndex,omitempty"`
	Template   Optional[string] `json:"template,omitempty"`
}

type StrictComponentChassisSharedWheelsWheelEntry struct {
	IsLeft Optional[string] `json:"isLeft,omitempty"`
	Name   Optional[string] `json:"name,omitempty"`
	Radius Optional[string] `json:"radius,omitempty"`
}

type StrictComponentEnginesShared struct {
	Deprecated         Optional[string]                `json:"deprecated,omitempty"`
	Description        Optional[string]                `json:"description,omitempty"`
	FireStartingChance Optional[string]                `json:"fireStartingChance,omitempty"`
	Level              Optional[string]                `json:"level,omitempty"`
	MaxHealth          Optional[string]                `json:"maxHealth,omitempty"`
	MaxRegenHealth     Optional[string]                `json:"maxRegenHealth,omitempty"`
	NotInShop          Optional[XMLScalarList[string]] `json:"notInShop,omitempty"`
	Power              Optional[string]                `json:"power,omitempty"`
	Price              Optional[string]                `json:"price,omitempty"`
	RepairCost         Optional[string]                `json:"repairCost,omitempty"`
	RpmMax             Optional[string]                `json:"rpm_max,omitempty"`
	RpmMin             Optional[string]                `json:"rpm_min,omitempty"`
	ShortUserString    Optional[string]                `json:"shortUserString,omitempty"`
	Sound              Optional[string]                `json:"sound,omitempty"`
	SoundNPC           Optional[string]                `json:"soundNPC,omitempty"`
	SoundPC            Optional[string]                `json:"soundPC,omitempty"`
	Tags               Optional[XMLScalarList[string]] `json:"tags,omitempty"`
	UserString         Optional[string]                `json:"userString,omitempty"`
	Weight             Optional[string]                `json:"weight,omitempty"`
	WwsoundNPC         Optional[XMLScalarList[string]] `json:"wwsoundNPC,omitempty"`
	WwsoundPC          Optional[XMLScalarList[string]] `json:"wwsoundPC,omitempty"`
}

type StrictComponentFuelTanksShared struct {
	Description    Optional[string] `json:"description,omitempty"`
	MaxHealth      Optional[string] `json:"maxHealth,omitempty"`
	MaxRegenHealth Optional[string] `json:"maxRegenHealth,omitempty"`
	Price          Optional[string] `json:"price,omitempty"`
	RepairCost     Optional[string] `json:"repairCost,omitempty"`
	Tags           Optional[string] `json:"tags,omitempty"`
	UserString     Optional[string] `json:"userString,omitempty"`
	Weight         Optional[string] `json:"weight,omitempty"`
}

type StrictComponentGunsShared struct {
	AimingTime               Optional[string]                                          `json:"aimingTime,omitempty"`
	Burst                    Optional[StrictComponentGunsSharedBurst]                  `json:"burst,omitempty"`
	Clip                     Optional[StrictComponentGunsSharedClip]                   `json:"clip,omitempty"`
	Description              Optional[string]                                          `json:"description,omitempty"`
	Effects                  Optional[string]                                          `json:"effects,omitempty"`
	ExtraPitchLimits         Optional[StrictComponentGunsSharedExtraPitchLimits]       `json:"extraPitchLimits,omitempty"`
	Impulse                  Optional[string]                                          `json:"impulse,omitempty"`
	InvisibilityFactorAtShot Optional[string]                                          `json:"invisibilityFactorAtShot,omitempty"`
	Level                    Optional[string]                                          `json:"level,omitempty"`
	MaxAmmo                  Optional[string]                                          `json:"maxAmmo,omitempty"`
	MaxHealth                Optional[string]                                          `json:"maxHealth,omitempty"`
	MaxRegenHealth           Optional[string]                                          `json:"maxRegenHealth,omitempty"`
	NotInShop                Optional[string]                                          `json:"notInShop,omitempty"`
	PitchLimits              Optional[string]                                          `json:"pitchLimits,omitempty"`
	Price                    Optional[string]                                          `json:"price,omitempty"`
	Recoil                   Optional[StrictComponentGunsSharedRecoil]                 `json:"recoil,omitempty"`
	ReloadEffect             Optional[string]                                          `json:"reloadEffect,omitempty"`
	ReloadTime               Optional[string]                                          `json:"reloadTime,omitempty"`
	RepairCost               Optional[string]                                          `json:"repairCost,omitempty"`
	RotationSpeed            Optional[string]                                          `json:"rotationSpeed,omitempty"`
	ShortUserString          Optional[string]                                          `json:"shortUserString,omitempty"`
	ShotDispersionFactors    Optional[StrictComponentGunsSharedShotDispersionFactors]  `json:"shotDispersionFactors,omitempty"`
	ShotDispersionRadius     Optional[string]                                          `json:"shotDispersionRadius,omitempty"`
	Shots                    Optional[[]Optional[StrictComponentGunsSharedShotsEntry]] `json:"shots,omitempty"`
	Tags                     Optional[XMLScalarList[string]]                           `json:"tags,omitempty"`
	TurretYawLimits          Optional[XMLScalarList[string]]                           `json:"turretYawLimits,omitempty"`
	UserString               Optional[string]                                          `json:"userString,omitempty"`
	Weight                   Optional[string]                                          `json:"weight,omitempty"`
	YawLimits                Optional[string]                                          `json:"yawLimits,omitempty"`
}

type StrictComponentGunsSharedBurst struct {
	Count Optional[string] `json:"count,omitempty"`
	Rate  Optional[string] `json:"rate,omitempty"`
}

type StrictComponentGunsSharedClip struct {
	Count Optional[string] `json:"count,omitempty"`
	Rate  Optional[string] `json:"rate,omitempty"`
}

type StrictComponentGunsSharedExtraPitchLimits struct {
	Back Optional[string] `json:"back,omitempty"`
}

type StrictComponentGunsSharedRecoil struct {
	Amplitude    Optional[string] `json:"amplitude,omitempty"`
	BackoffTime  Optional[string] `json:"backoffTime,omitempty"`
	LodDist      Optional[string] `json:"lodDist,omitempty"`
	RecoilEffect Optional[string] `json:"recoilEffect,omitempty"`
	ReturnTime   Optional[string] `json:"returnTime,omitempty"`
}

type StrictComponentGunsSharedShotDispersionFactors struct {
	AfterShot       Optional[string] `json:"afterShot,omitempty"`
	TurretRotation  Optional[string] `json:"turretRotation,omitempty"`
	WhileGunDamaged Optional[string] `json:"whileGunDamaged,omitempty"`
}

type StrictComponentGunsSharedShotsEntry struct {
	Name       Optional[string]                                        `json:"name,omitempty"`
	Properties Optional[StrictComponentGunsSharedShotsEntryProperties] `json:"properties,omitempty"`
}

type StrictComponentGunsSharedShotsEntryProperties struct {
	Acceleration           Optional[string] `json:"acceleration,omitempty"`
	DefaultPortion         Optional[string] `json:"defaultPortion,omitempty"`
	Gravity                Optional[string] `json:"gravity,omitempty"`
	MaxDistance            Optional[string] `json:"maxDistance,omitempty"`
	MaxSpeed               Optional[string] `json:"maxSpeed,omitempty"`
	PiercingPower          Optional[string] `json:"piercingPower,omitempty"`
	RotationalDeceleration Optional[string] `json:"rotationalDeceleration,omitempty"`
	Speed                  Optional[string] `json:"speed,omitempty"`
	TurnSpeedHorizontal    Optional[string] `json:"turnSpeedHorizontal,omitempty"`
	TurnSpeedVertical      Optional[string] `json:"turnSpeedVertical,omitempty"`
}

type StrictComponentRadiosShared struct {
	Description    Optional[string] `json:"description,omitempty"`
	Distance       Optional[string] `json:"distance,omitempty"`
	Level          Optional[string] `json:"level,omitempty"`
	MaxHealth      Optional[string] `json:"maxHealth,omitempty"`
	MaxRegenHealth Optional[string] `json:"maxRegenHealth,omitempty"`
	NotInShop      Optional[string] `json:"notInShop,omitempty"`
	Price          Optional[string] `json:"price,omitempty"`
	RepairCost     Optional[string] `json:"repairCost,omitempty"`
	Tags           Optional[string] `json:"tags,omitempty"`
	UserString     Optional[string] `json:"userString,omitempty"`
	Weight         Optional[string] `json:"weight,omitempty"`
}

type StrictComponentTurretsShared struct {
}

type StrictShell struct {
	Blitz                             Optional[StrictShellBlitz]                        `json:"blitz,omitempty"`
	Caliber                           Optional[string]                                  `json:"caliber,omitempty"`
	Damage                            Optional[StrictShellDamage]                       `json:"damage,omitempty"`
	Description                       Optional[string]                                  `json:"description,omitempty"`
	Effects                           Optional[string]                                  `json:"effects,omitempty"`
	ExplosionRadius                   Optional[string]                                  `json:"explosionRadius,omitempty"`
	Icon                              Optional[string]                                  `json:"icon,omitempty"`
	Id                                Optional[string]                                  `json:"id,omitempty"`
	Improved                          Optional[string]                                  `json:"improved,omitempty"`
	IsTracer                          Optional[string]                                  `json:"isTracer,omitempty"`
	Kind                              Optional[string]                                  `json:"kind,omitempty"`
	NormalizationAngle                Optional[string]                                  `json:"normalizationAngle,omitempty"`
	NotInShop                         Optional[string]                                  `json:"notInShop,omitempty"`
	PiercingPowerLossFactorByDistance Optional[string]                                  `json:"piercingPowerLossFactorByDistance,omitempty"`
	Price                             Optional[XMLScalarObject[StrictShellPriceObject]] `json:"price,omitempty"`
	ProjectileType                    Optional[string]                                  `json:"projectileType,omitempty"`
	RicochetAngle                     Optional[string]                                  `json:"ricochetAngle,omitempty"`
	UserString                        Optional[string]                                  `json:"userString,omitempty"`
}

type StrictShellBlitz struct {
	BlitzIconFrame Optional[string] `json:"blitzIconFrame,omitempty"`
}

type StrictShellDamage struct {
	Armor   Optional[string] `json:"armor,omitempty"`
	Devices Optional[string] `json:"devices,omitempty"`
}

type StrictShellPriceObject struct {
	Text Optional[string] `json:"#text,omitempty"`
	Gold Optional[string] `json:"gold,omitempty"`
}
