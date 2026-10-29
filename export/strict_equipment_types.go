package export

// Equipment fields retain XML scalar spelling; empty XML elements are null.
type StrictEquipment struct {
	Devices Optional[StrictEquipmentDevices] `json:"devices,omitempty"`
	Slots   Optional[StrictEquipmentSlots]   `json:"slots,omitempty"`
}
type StrictEquipmentSlots struct {
	Presets Optional[map[string]StrictEquipmentPreset] `json:"presets,omitempty"`
}

type StrictEquipmentDevice struct {
	Description   Optional[string]                             `json:"description,omitempty"`
	DisplayParams Optional[StrictEquipmentDeviceDisplayParams] `json:"display_params,omitempty"`
	Icon          Optional[string]                             `json:"icon,omitempty"`
	ID            Optional[string]                             `json:"id,omitempty"`
	Script        Optional[StrictEquipmentDeviceScript]        `json:"script,omitempty"`
	UserString    Optional[string]                             `json:"userString,omitempty"`
}

type StrictEquipmentDeviceDisplayParams struct {
	ARMORPIERCING                   Optional[string]                      `json:"ARMOR_PIERCING,omitempty"`
	ARMORPIERCINGCR                 Optional[string]                      `json:"ARMOR_PIERCING_CR,omitempty"`
	HIGHEXPLOSIVE                   Optional[string]                      `json:"HIGH_EXPLOSIVE,omitempty"`
	HOLLOWCHARGE                    Optional[string]                      `json:"HOLLOW_CHARGE,omitempty"`
	ActivateWhenStillSec            Optional[string]                      `json:"activateWhenStillSec,omitempty"`
	AdditiveShotDispersionFactor    Optional[string]                      `json:"additiveShotDispersionFactor,omitempty"`
	AmmoEvasionPercentageBonus      Optional[string]                      `json:"ammoEvasionPercentageBonus,omitempty"`
	ArmorHomogenizationFactor       Optional[string]                      `json:"armorHomogenizationFactor,omitempty"`
	CircularVisionRadiusFactor      Optional[StrictEquipmentClassDisplay] `json:"circularVisionRadiusFactor,omitempty"`
	CircularVisionRadiusIncrease    Optional[string]                      `json:"circularVisionRadiusIncrease,omitempty"`
	CrewEvasionPercentageBonus      Optional[string]                      `json:"crewEvasionPercentageBonus,omitempty"`
	CrewLevelIncrease               Optional[string]                      `json:"crewLevelIncrease,omitempty"`
	DetectabilityFactor             Optional[StrictEquipmentClassDisplay] `json:"detectabilityFactor,omitempty"`
	EngineEvasionPercentageBonus    Optional[string]                      `json:"engineEvasionPercentageBonus,omitempty"`
	EnginePower                     Optional[StrictEquipmentClassDisplay] `json:"enginePower,omitempty"`
	EquipmentDurationFactor         Optional[string]                      `json:"equipmentDurationFactor,omitempty"`
	EquipmentReloadBoost            Optional[string]                      `json:"equipmentReloadBoost,omitempty"`
	FirmGroundPassabilityIncrease   Optional[string]                      `json:"firmGroundPassabilityIncrease,omitempty"`
	GunAimingTimeFactor             Optional[string]                      `json:"gunAimingTimeFactor,omitempty"`
	GunReloadTimeFactor             Optional[string]                      `json:"gunReloadTimeFactor,omitempty"`
	InvisibilityFactorAtShot        Optional[string]                      `json:"invisibilityFactorAtShot,omitempty"`
	LowerPitchLimitIncrease         Optional[string]                      `json:"lowerPitchLimitIncrease,omitempty"`
	MaxHealthPercentageBonus        Optional[string]                      `json:"maxHealthPercentageBonus,omitempty"`
	MaxHpBonusPercent               Optional[string]                      `json:"maxHpBonusPercent,omitempty"`
	MediumGroundPassabilityIncrease Optional[string]                      `json:"mediumGroundPassabilityIncrease,omitempty"`
	PiercingPenaltyFactor500m       Optional[string]                      `json:"piercingPenaltyFactor500m,omitempty"`
	ProjectileAccelerationFactor    Optional[string]                      `json:"projectileAccelerationFactor,omitempty"`
	ProjectileMaxSpeedFactor        Optional[string]                      `json:"projectileMaxSpeedFactor,omitempty"`
	ProjectileSpeedFactor           Optional[string]                      `json:"projectileSpeedFactor,omitempty"`
	RammingAbsorptionPercent        Optional[string]                      `json:"rammingAbsorptionPercent,omitempty"`
	RepairSpeedFactor               Optional[string]                      `json:"repairSpeedFactor,omitempty"`
	RotationSpeedLimit              Optional[string]                      `json:"rotationSpeedLimit,omitempty"`
	ShotDispersionAngleFactor       Optional[string]                      `json:"shotDispersionAngleFactor,omitempty"`
	SoftGroundPassabilityIncrease   Optional[string]                      `json:"softGroundPassabilityIncrease,omitempty"`
	UpperPitchLimitIncrease         Optional[string]                      `json:"upperPitchLimitIncrease,omitempty"`
}

type StrictEquipmentClassDisplay struct {
	ATSPG  Optional[string] `json:"AT_SPG,omitempty"`
	Heavy  Optional[string] `json:"Heavy,omitempty"`
	Light  Optional[string] `json:"Light,omitempty"`
	Medium Optional[string] `json:"Medium,omitempty"`
}

type StrictEquipmentDeviceScript struct {
	Text                                    Optional[string]                                          `json:"#text,omitempty"`
	ActivateBonusFactor                     Optional[string]                                          `json:"activateBonusFactor,omitempty"`
	ActivateWhenStillSec                    Optional[string]                                          `json:"activateWhenStillSec,omitempty"`
	Attribute                               Optional[string]                                          `json:"attribute,omitempty"`
	Bonus                                   Optional[StrictEquipmentClassBonus]                       `json:"bonus,omitempty"`
	BonusPercentage                         Optional[StrictEquipmentDeviceScriptBonusPercentage]      `json:"bonusPercentage,omitempty"`
	BonusValues                             Optional[StrictEquipmentDeviceScriptBonusValues]          `json:"bonusValues,omitempty"`
	ChancesToHit                            Optional[StrictEquipmentDeviceScriptChancesToHit]         `json:"chancesToHit,omitempty"`
	EnginePowerPercentageBonus              Optional[StrictEquipmentClassBonus]                       `json:"enginePowerPercentageBonus,omitempty"`
	LargeCaliberHighExplosiveDamageFactor   Optional[string]                                          `json:"largeCaliberHighExplosiveDamageFactor,omitempty"`
	LargeCaliberHighExplosivePiercingFactor Optional[string]                                          `json:"largeCaliberHighExplosivePiercingFactor,omitempty"`
	MaxHpBonusPercent                       Optional[string]                                          `json:"maxHpBonusPercent,omitempty"`
	Modules                                 Optional[string]                                          `json:"modules,omitempty"`
	ModulesHealthFactors                    Optional[StrictEquipmentDeviceScriptModulesHealthFactors] `json:"modulesHealthFactors,omitempty"`
	PiercingFactors                         Optional[StrictEquipmentDeviceScriptPiercingFactors]      `json:"piercingFactors,omitempty"`
	RammingAbsorptionPercent                Optional[string]                                          `json:"rammingAbsorptionPercent,omitempty"`
	VisionRadiusPercentageBonus             Optional[StrictEquipmentClassBonus]                       `json:"visionRadiusPercentageBonus,omitempty"`
	Weight                                  Optional[string]                                          `json:"weight,omitempty"`
}

type StrictEquipmentClassBonus struct {
	ATSPG      Optional[string] `json:"AT-SPG,omitempty"`
	SPG        Optional[string] `json:"SPG,omitempty"`
	HeavyTank  Optional[string] `json:"heavyTank,omitempty"`
	LightTank  Optional[string] `json:"lightTank,omitempty"`
	MediumTank Optional[string] `json:"mediumTank,omitempty"`
}

type StrictEquipmentDeviceScriptBonusPercentage struct {
	FirmGroundPassabilityIncrease   Optional[string] `json:"firmGroundPassabilityIncrease,omitempty"`
	InvisibilityFactorAtShot        Optional[string] `json:"invisibilityFactorAtShot,omitempty"`
	MediumGroundPassabilityIncrease Optional[string] `json:"mediumGroundPassabilityIncrease,omitempty"`
	PiercingPenaltyFactor500m       Optional[string] `json:"piercingPenaltyFactor500m,omitempty"`
	ProjectileAccelerationFactor    Optional[string] `json:"projectileAccelerationFactor,omitempty"`
	ProjectileMaxSpeedFactor        Optional[string] `json:"projectileMaxSpeedFactor,omitempty"`
	ProjectileSpeedFactor           Optional[string] `json:"projectileSpeedFactor,omitempty"`
	RotationSpeedFactor             Optional[string] `json:"rotationSpeedFactor,omitempty"`
	SoftGroundPassabilityIncrease   Optional[string] `json:"softGroundPassabilityIncrease,omitempty"`
}

type StrictEquipmentDeviceScriptBonusValues struct {
	CircularVisionRadiusIncrease Optional[string] `json:"circularVisionRadiusIncrease,omitempty"`
	LowerPitchLimitIncrease      Optional[string] `json:"lowerPitchLimitIncrease,omitempty"`
	Percent                      Optional[string] `json:"percent,omitempty"`
	UpperPitchLimitIncrease      Optional[string] `json:"upperPitchLimitIncrease,omitempty"`
	Value                        Optional[string] `json:"value,omitempty"`
}

type StrictEquipmentDeviceScriptChancesToHit struct {
	Crew    Optional[string]                                         `json:"crew,omitempty"`
	Modules Optional[StrictEquipmentDeviceScriptChancesToHitModules] `json:"modules,omitempty"`
}

type StrictEquipmentDeviceScriptChancesToHitModules struct {
	AmmoBayHealth Optional[string] `json:"ammoBayHealth,omitempty"`
	EngineHealth  Optional[string] `json:"engineHealth,omitempty"`
}

type StrictEquipmentDeviceScriptModulesHealthFactors struct {
	Modules Optional[StrictEquipmentDeviceScriptModulesHealthFactorsModules] `json:"modules,omitempty"`
}

type StrictEquipmentDeviceScriptModulesHealthFactorsModules struct {
	AmmoBayHealth    Optional[string] `json:"ammoBayHealth,omitempty"`
	FuelTankHealth   Optional[string] `json:"fuelTankHealth,omitempty"`
	LeftTrackHealth  Optional[string] `json:"leftTrackHealth,omitempty"`
	RightTrackHealth Optional[string] `json:"rightTrackHealth,omitempty"`
}

type StrictEquipmentDeviceScriptPiercingFactors struct {
	ARMORPIERCING   Optional[string] `json:"ARMOR_PIERCING,omitempty"`
	ARMORPIERCINGCR Optional[string] `json:"ARMOR_PIERCING_CR,omitempty"`
	ARMORPIERCINGHE Optional[string] `json:"ARMOR_PIERCING_HE,omitempty"`
	HIGHEXPLOSIVE   Optional[string] `json:"HIGH_EXPLOSIVE,omitempty"`
	HOLLOWCHARGE    Optional[string] `json:"HOLLOW_CHARGE,omitempty"`
}

type StrictEquipmentPreset struct {
	Level0 Optional[StrictEquipmentPresetLevel] `json:"level0,omitempty"`
	Level1 Optional[StrictEquipmentPresetLevel] `json:"level1,omitempty"`
	Level2 Optional[StrictEquipmentPresetLevel] `json:"level2,omitempty"`
}

type StrictEquipmentPresetLevel struct {
	Attack         Optional[StrictEquipmentSlot] `json:"attack,omitempty"`
	Defence        Optional[StrictEquipmentSlot] `json:"defence,omitempty"`
	Specialization Optional[StrictEquipmentSlot] `json:"specialization,omitempty"`
}

type StrictEquipmentSlot struct {
	Device0 Optional[string] `json:"device0,omitempty"`
	Device1 Optional[string] `json:"device1,omitempty"`
}
