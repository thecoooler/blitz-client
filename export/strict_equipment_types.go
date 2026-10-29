package export

// Equipment fields retain XML scalar spelling; empty XML elements are null.
type StrictEquipment struct {
	Devices Optional[StrictEquipmentDevices] `json:"devices,omitzero"`
	Slots   Optional[StrictEquipmentSlots]   `json:"slots,omitzero"`
}
type StrictEquipmentSlots struct {
	Presets Optional[map[string]StrictEquipmentPreset] `json:"presets,omitzero"`
}

type StrictEquipmentDevice struct {
	Description   Optional[string]                             `json:"description,omitzero"`
	DisplayParams Optional[StrictEquipmentDeviceDisplayParams] `json:"display_params,omitzero"`
	Icon          Optional[string]                             `json:"icon,omitzero"`
	ID            Optional[XMLInteger]                         `json:"id,omitzero"`
	Script        Optional[StrictEquipmentDeviceScript]        `json:"script,omitzero"`
	UserString    Optional[string]                             `json:"userString,omitzero"`
}

type StrictEquipmentDeviceDisplayParams struct {
	ARMORPIERCING                   Optional[XMLInteger]                  `json:"ARMOR_PIERCING,omitzero"`
	ARMORPIERCINGCR                 Optional[XMLInteger]                  `json:"ARMOR_PIERCING_CR,omitzero"`
	HIGHEXPLOSIVE                   Optional[XMLInteger]                  `json:"HIGH_EXPLOSIVE,omitzero"`
	HOLLOWCHARGE                    Optional[XMLInteger]                  `json:"HOLLOW_CHARGE,omitzero"`
	ActivateWhenStillSec            Optional[XMLNumber]                   `json:"activateWhenStillSec,omitzero"`
	AdditiveShotDispersionFactor    Optional[XMLInteger]                  `json:"additiveShotDispersionFactor,omitzero"`
	AmmoEvasionPercentageBonus      Optional[XMLInteger]                  `json:"ammoEvasionPercentageBonus,omitzero"`
	ArmorHomogenizationFactor       Optional[XMLInteger]                  `json:"armorHomogenizationFactor,omitzero"`
	CircularVisionRadiusFactor      Optional[StrictEquipmentClassDisplay] `json:"circularVisionRadiusFactor,omitzero"`
	CircularVisionRadiusIncrease    Optional[XMLInteger]                  `json:"circularVisionRadiusIncrease,omitzero"`
	CrewEvasionPercentageBonus      Optional[XMLInteger]                  `json:"crewEvasionPercentageBonus,omitzero"`
	CrewLevelIncrease               Optional[XMLInteger]                  `json:"crewLevelIncrease,omitzero"`
	DetectabilityFactor             Optional[StrictEquipmentClassDisplay] `json:"detectabilityFactor,omitzero"`
	EngineEvasionPercentageBonus    Optional[XMLInteger]                  `json:"engineEvasionPercentageBonus,omitzero"`
	EnginePower                     Optional[StrictEquipmentClassDisplay] `json:"enginePower,omitzero"`
	EquipmentDurationFactor         Optional[XMLInteger]                  `json:"equipmentDurationFactor,omitzero"`
	EquipmentReloadBoost            Optional[XMLInteger]                  `json:"equipmentReloadBoost,omitzero"`
	FirmGroundPassabilityIncrease   Optional[XMLInteger]                  `json:"firmGroundPassabilityIncrease,omitzero"`
	GunAimingTimeFactor             Optional[XMLInteger]                  `json:"gunAimingTimeFactor,omitzero"`
	GunReloadTimeFactor             Optional[XMLInteger]                  `json:"gunReloadTimeFactor,omitzero"`
	InvisibilityFactorAtShot        Optional[XMLInteger]                  `json:"invisibilityFactorAtShot,omitzero"`
	LowerPitchLimitIncrease         Optional[XMLInteger]                  `json:"lowerPitchLimitIncrease,omitzero"`
	MaxHealthPercentageBonus        Optional[XMLInteger]                  `json:"maxHealthPercentageBonus,omitzero"`
	MaxHpBonusPercent               Optional[XMLInteger]                  `json:"maxHpBonusPercent,omitzero"`
	MediumGroundPassabilityIncrease Optional[XMLInteger]                  `json:"mediumGroundPassabilityIncrease,omitzero"`
	PiercingPenaltyFactor500m       Optional[XMLInteger]                  `json:"piercingPenaltyFactor500m,omitzero"`
	ProjectileAccelerationFactor    Optional[XMLInteger]                  `json:"projectileAccelerationFactor,omitzero"`
	ProjectileMaxSpeedFactor        Optional[XMLInteger]                  `json:"projectileMaxSpeedFactor,omitzero"`
	ProjectileSpeedFactor           Optional[XMLInteger]                  `json:"projectileSpeedFactor,omitzero"`
	RammingAbsorptionPercent        Optional[XMLInteger]                  `json:"rammingAbsorptionPercent,omitzero"`
	RepairSpeedFactor               Optional[XMLInteger]                  `json:"repairSpeedFactor,omitzero"`
	RotationSpeedLimit              Optional[XMLInteger]                  `json:"rotationSpeedLimit,omitzero"`
	ShotDispersionAngleFactor       Optional[XMLInteger]                  `json:"shotDispersionAngleFactor,omitzero"`
	SoftGroundPassabilityIncrease   Optional[XMLInteger]                  `json:"softGroundPassabilityIncrease,omitzero"`
	UpperPitchLimitIncrease         Optional[XMLInteger]                  `json:"upperPitchLimitIncrease,omitzero"`
}

type StrictEquipmentClassDisplay struct {
	ATSPG  Optional[XMLInteger] `json:"AT_SPG,omitzero"`
	Heavy  Optional[XMLInteger] `json:"Heavy,omitzero"`
	Light  Optional[XMLInteger] `json:"Light,omitzero"`
	Medium Optional[XMLInteger] `json:"Medium,omitzero"`
}

type StrictEquipmentDeviceScript struct {
	Text                                    Optional[string]                                          `json:"#text,omitzero"`
	ActivateBonusFactor                     Optional[XMLNumber]                                       `json:"activateBonusFactor,omitzero"`
	ActivateWhenStillSec                    Optional[XMLNumber]                                       `json:"activateWhenStillSec,omitzero"`
	Attribute                               Optional[string]                                          `json:"attribute,omitzero"`
	Bonus                                   Optional[StrictEquipmentClassBonus]                       `json:"bonus,omitzero"`
	BonusPercentage                         Optional[StrictEquipmentDeviceScriptBonusPercentage]      `json:"bonusPercentage,omitzero"`
	BonusValues                             Optional[StrictEquipmentDeviceScriptBonusValues]          `json:"bonusValues,omitzero"`
	ChancesToHit                            Optional[StrictEquipmentDeviceScriptChancesToHit]         `json:"chancesToHit,omitzero"`
	EnginePowerPercentageBonus              Optional[StrictEquipmentClassBonus]                       `json:"enginePowerPercentageBonus,omitzero"`
	LargeCaliberHighExplosiveDamageFactor   Optional[XMLNumber]                                       `json:"largeCaliberHighExplosiveDamageFactor,omitzero"`
	LargeCaliberHighExplosivePiercingFactor Optional[XMLNumber]                                       `json:"largeCaliberHighExplosivePiercingFactor,omitzero"`
	MaxHpBonusPercent                       Optional[XMLInteger]                                      `json:"maxHpBonusPercent,omitzero"`
	Modules                                 Optional[string]                                          `json:"modules,omitzero"`
	ModulesHealthFactors                    Optional[StrictEquipmentDeviceScriptModulesHealthFactors] `json:"modulesHealthFactors,omitzero"`
	PiercingFactors                         Optional[StrictEquipmentDeviceScriptPiercingFactors]      `json:"piercingFactors,omitzero"`
	RammingAbsorptionPercent                Optional[XMLInteger]                                      `json:"rammingAbsorptionPercent,omitzero"`
	VisionRadiusPercentageBonus             Optional[StrictEquipmentClassBonus]                       `json:"visionRadiusPercentageBonus,omitzero"`
	Weight                                  Optional[XMLInteger]                                      `json:"weight,omitzero"`
}

type StrictEquipmentClassBonus struct {
	ATSPG      Optional[XMLNumber] `json:"AT-SPG,omitzero"`
	SPG        Optional[XMLNumber] `json:"SPG,omitzero"`
	HeavyTank  Optional[XMLNumber] `json:"heavyTank,omitzero"`
	LightTank  Optional[XMLNumber] `json:"lightTank,omitzero"`
	MediumTank Optional[XMLNumber] `json:"mediumTank,omitzero"`
}

type StrictEquipmentDeviceScriptBonusPercentage struct {
	FirmGroundPassabilityIncrease   Optional[XMLInteger] `json:"firmGroundPassabilityIncrease,omitzero"`
	InvisibilityFactorAtShot        Optional[XMLInteger] `json:"invisibilityFactorAtShot,omitzero"`
	MediumGroundPassabilityIncrease Optional[XMLInteger] `json:"mediumGroundPassabilityIncrease,omitzero"`
	PiercingPenaltyFactor500m       Optional[XMLInteger] `json:"piercingPenaltyFactor500m,omitzero"`
	ProjectileAccelerationFactor    Optional[XMLInteger] `json:"projectileAccelerationFactor,omitzero"`
	ProjectileMaxSpeedFactor        Optional[XMLInteger] `json:"projectileMaxSpeedFactor,omitzero"`
	ProjectileSpeedFactor           Optional[XMLInteger] `json:"projectileSpeedFactor,omitzero"`
	RotationSpeedFactor             Optional[XMLInteger] `json:"rotationSpeedFactor,omitzero"`
	SoftGroundPassabilityIncrease   Optional[XMLInteger] `json:"softGroundPassabilityIncrease,omitzero"`
}

type StrictEquipmentDeviceScriptBonusValues struct {
	CircularVisionRadiusIncrease Optional[XMLInteger] `json:"circularVisionRadiusIncrease,omitzero"`
	LowerPitchLimitIncrease      Optional[XMLInteger] `json:"lowerPitchLimitIncrease,omitzero"`
	Percent                      Optional[XMLInteger] `json:"percent,omitzero"`
	UpperPitchLimitIncrease      Optional[XMLInteger] `json:"upperPitchLimitIncrease,omitzero"`
	Value                        Optional[XMLNumber]  `json:"value,omitzero"`
}

type StrictEquipmentDeviceScriptChancesToHit struct {
	Crew    Optional[XMLNumber]                                      `json:"crew,omitzero"`
	Modules Optional[StrictEquipmentDeviceScriptChancesToHitModules] `json:"modules,omitzero"`
}

type StrictEquipmentDeviceScriptChancesToHitModules struct {
	AmmoBayHealth Optional[XMLNumber] `json:"ammoBayHealth,omitzero"`
	EngineHealth  Optional[XMLNumber] `json:"engineHealth,omitzero"`
}

type StrictEquipmentDeviceScriptModulesHealthFactors struct {
	Modules Optional[StrictEquipmentDeviceScriptModulesHealthFactorsModules] `json:"modules,omitzero"`
}

type StrictEquipmentDeviceScriptModulesHealthFactorsModules struct {
	AmmoBayHealth    Optional[XMLNumber] `json:"ammoBayHealth,omitzero"`
	FuelTankHealth   Optional[XMLNumber] `json:"fuelTankHealth,omitzero"`
	LeftTrackHealth  Optional[XMLNumber] `json:"leftTrackHealth,omitzero"`
	RightTrackHealth Optional[XMLNumber] `json:"rightTrackHealth,omitzero"`
}

type StrictEquipmentDeviceScriptPiercingFactors struct {
	ARMORPIERCING   Optional[XMLInteger] `json:"ARMOR_PIERCING,omitzero"`
	ARMORPIERCINGCR Optional[XMLInteger] `json:"ARMOR_PIERCING_CR,omitzero"`
	ARMORPIERCINGHE Optional[XMLInteger] `json:"ARMOR_PIERCING_HE,omitzero"`
	HIGHEXPLOSIVE   Optional[XMLInteger] `json:"HIGH_EXPLOSIVE,omitzero"`
	HOLLOWCHARGE    Optional[XMLInteger] `json:"HOLLOW_CHARGE,omitzero"`
}

type StrictEquipmentPreset struct {
	Level0 Optional[StrictEquipmentPresetLevel] `json:"level0,omitzero"`
	Level1 Optional[StrictEquipmentPresetLevel] `json:"level1,omitzero"`
	Level2 Optional[StrictEquipmentPresetLevel] `json:"level2,omitzero"`
}

type StrictEquipmentPresetLevel struct {
	Attack         Optional[StrictEquipmentSlot] `json:"attack,omitzero"`
	Defence        Optional[StrictEquipmentSlot] `json:"defence,omitzero"`
	Specialization Optional[StrictEquipmentSlot] `json:"specialization,omitzero"`
}

type StrictEquipmentSlot struct {
	Device0 Optional[string] `json:"device0,omitzero"`
	Device1 Optional[string] `json:"device1,omitzero"`
}
