// Types below describe the XML shapes observed in the current provisions export.
package export

import (
	"encoding/json"
	"fmt"
)

// StrictProvisions is keyed by source mode, then provision name.
type StrictProvisions map[string]map[string]StrictProvision

type StrictProvision struct {
	AbilityCost            Optional[XMLInteger]                     `json:"abilityCost,omitzero"`
	Category               Optional[string]                         `json:"category,omitzero"`
	DescrParam             Optional[StrictProvisionDescrParamValue] `json:"descrParam,omitzero"`
	Description            Optional[string]                         `json:"description,omitzero"`
	DisplayParams          Optional[StrictProvisionDisplayParams]   `json:"display_params,omitzero"`
	DynamicParams          Optional[StrictProvisionDynamicParams]   `json:"dynamic_params,omitzero"`
	ExtraEffectDuration    Optional[XMLNumber]                      `json:"extraEffectDuration,omitzero"`
	GameModeFilter         Optional[StrictProvisionGameModeFilter]  `json:"gameModeFilter,omitzero"`
	GeometryFilter         Optional[StrictProvisionGeometryFilter]  `json:"geometryFilter,omitzero"`
	Grade                  Optional[XMLInteger]                     `json:"grade,omitzero"`
	HideIconOnAbilityPanel Optional[StrictNull]                     `json:"hideIconOnAbilityPanel,omitzero"`
	Icon                   Optional[string]                         `json:"icon,omitzero"`
	Id                     Optional[XMLInteger]                     `json:"id,omitzero"`
	IsAbility              Optional[StrictNull]                     `json:"isAbility,omitzero"`
	NextImprovement        Optional[string]                         `json:"nextImprovement,omitzero"`
	Price                  Optional[StrictProvisionPrice]           `json:"price,omitzero"`
	Script                 Optional[StrictProvisionScriptValue]     `json:"script,omitzero"`
	Tags                   Optional[string]                         `json:"tags,omitzero"`
	UserString             Optional[string]                         `json:"userString,omitzero"`
	VehicleFilter          Optional[StrictProvisionVehicleFilter]   `json:"vehicleFilter,omitzero"`
}

type StrictProvisionDescrParam struct {
	Name  Optional[string]     `json:"name,omitzero"`
	Value Optional[XMLInteger] `json:"value,omitzero"`
}

type StrictProvisionDisplayParams struct {
	AimingTimeFactor                Optional[XMLInteger] `json:"aimingTimeFactor,omitzero"`
	AntiHighExplosiveFactor         Optional[XMLInteger] `json:"antiHighExplosiveFactor,omitzero"`
	BkwdSpeedLimit                  Optional[XMLInteger] `json:"bkwdSpeedLimit,omitzero"`
	CommonDamageFactor              Optional[XMLInteger] `json:"commonDamageFactor,omitzero"`
	CrewProtectionIncrease          Optional[XMLInteger] `json:"crewProtectionIncrease,omitzero"`
	DealingRammingDamageFactor      Optional[XMLInteger] `json:"dealingRammingDamageFactor,omitzero"`
	EnginePowerIncrease             Optional[XMLInteger] `json:"enginePowerIncrease,omitzero"`
	EnginePower                     Optional[XMLInteger] `json:"engine__power,omitzero"`
	ExplosivesHealthIncrease        Optional[XMLInteger] `json:"explosivesHealthIncrease,omitzero"`
	FireProtectionIncrease          Optional[XMLInteger] `json:"fireProtectionIncrease,omitzero"`
	FirmGroundPassabilityIncrease   Optional[XMLInteger] `json:"firmGroundPassabilityIncrease,omitzero"`
	FwdSpeedLimit                   Optional[XMLInteger] `json:"fwdSpeedLimit,omitzero"`
	GunReloadTime                   Optional[XMLInteger] `json:"gun__reloadTime,omitzero"`
	IncreasedDamageChance           Optional[XMLInteger] `json:"increasedDamageChance,omitzero"`
	MaxHpBonusPercent               Optional[XMLInteger] `json:"maxHpBonusPercent,omitzero"`
	MediumGroundPassabilityIncrease Optional[XMLInteger] `json:"mediumGroundPassabilityIncrease,omitzero"`
	ProjectileSpeed                 Optional[XMLInteger] `json:"projectileSpeed,omitzero"`
	RammingDamageFactor             Optional[XMLInteger] `json:"rammingDamageFactor,omitzero"`
	RepairSpeedFactor               Optional[XMLInteger] `json:"repairSpeedFactor,omitzero"`
	RepairSpeedIncrease             Optional[XMLInteger] `json:"repairSpeedIncrease,omitzero"`
	ShotDispersionAngleFactor       Optional[XMLInteger] `json:"shotDispersionAngleFactor,omitzero"`
}

type StrictProvisionDynamicParams struct {
	Percents Optional[XMLInteger] `json:"percents,omitzero"`
}

type StrictProvisionGameModeFilter struct {
	Arcade                     Optional[StrictNull] `json:"arcade,omitzero"`
	Battleroyale               Optional[StrictNull] `json:"battleroyale,omitzero"`
	Bossmode                   Optional[StrictNull] `json:"bossmode,omitzero"`
	Gravitizing                Optional[StrictNull] `json:"gravitizing,omitzero"`
	Hellgames                  Optional[StrictNull] `json:"hellgames,omitzero"`
	QuickTournamentArcade      Optional[StrictNull] `json:"quick_tournament_arcade,omitzero"`
	QuickTournamentBossmode    Optional[StrictNull] `json:"quick_tournament_bossmode,omitzero"`
	QuickTournamentGravitizing Optional[StrictNull] `json:"quick_tournament_gravitizing,omitzero"`
	QuickTournamentHellgames   Optional[StrictNull] `json:"quick_tournament_hellgames,omitzero"`
	QuickTournamentVampiric    Optional[StrictNull] `json:"quick_tournament_vampiric,omitzero"`
	TournamentArcade           Optional[StrictNull] `json:"tournament_arcade,omitzero"`
	TournamentBossmode         Optional[StrictNull] `json:"tournament_bossmode,omitzero"`
	TournamentGravitizing      Optional[StrictNull] `json:"tournament_gravitizing,omitzero"`
	TournamentHellgames        Optional[StrictNull] `json:"tournament_hellgames,omitzero"`
	TournamentVampiric         Optional[StrictNull] `json:"tournament_vampiric,omitzero"`
	Vampiric                   Optional[StrictNull] `json:"vampiric,omitzero"`
}

type StrictProvisionGeometryFilter struct {
	Include Optional[string] `json:"include,omitzero"`
}

type StrictProvisionPrice struct {
	Level1  Optional[XMLInteger] `json:"level1,omitzero"`
	Level10 Optional[XMLInteger] `json:"level10,omitzero"`
	Level2  Optional[XMLInteger] `json:"level2,omitzero"`
	Level3  Optional[XMLInteger] `json:"level3,omitzero"`
	Level4  Optional[XMLInteger] `json:"level4,omitzero"`
	Level5  Optional[XMLInteger] `json:"level5,omitzero"`
	Level6  Optional[XMLInteger] `json:"level6,omitzero"`
	Level7  Optional[XMLInteger] `json:"level7,omitzero"`
	Level8  Optional[XMLInteger] `json:"level8,omitzero"`
	Level9  Optional[XMLInteger] `json:"level9,omitzero"`
}

type StrictProvisionScript struct {
	Text                       Optional[string]                           `json:"#text,omitzero"`
	ARMORPIERCING              Optional[XMLInteger]                       `json:"ARMOR_PIERCING,omitzero"`
	ARMORPIERCINGCR            Optional[XMLInteger]                       `json:"ARMOR_PIERCING_CR,omitzero"`
	HIGHEXPLOSIVE              Optional[XMLInteger]                       `json:"HIGH_EXPLOSIVE,omitzero"`
	HOLLOWCHARGE               Optional[XMLInteger]                       `json:"HOLLOW_CHARGE,omitzero"`
	Attributes                 Optional[StrictProvisionScriptAttributes]  `json:"attributes,omitzero"`
	BonusValues                Optional[StrictProvisionScriptBonusValues] `json:"bonusValues,omitzero"`
	CurseHeal                  Optional[XMLNumber]                        `json:"curseHeal,omitzero"`
	CurseReduction             Optional[XMLNumber]                        `json:"curseReduction,omitzero"`
	CurseStrength              Optional[XMLNumber]                        `json:"curseStrength,omitzero"`
	DamageReductionPercent     Optional[XMLInteger]                       `json:"damageReductionPercent,omitzero"`
	EffectDuration             Optional[XMLInteger]                       `json:"effectDuration,omitzero"`
	ExplodeDamagePercent       Optional[XMLInteger]                       `json:"explodeDamagePercent,omitzero"`
	HpStockPercent             Optional[XMLInteger]                       `json:"hpStockPercent,omitzero"`
	InvisibilityInMovingFactor Optional[XMLNumber]                        `json:"invisibilityInMovingFactor,omitzero"`
	MaxBonusSources            Optional[XMLInteger]                       `json:"maxBonusSources,omitzero"`
	ProjectileSpeedFactor      Optional[XMLInteger]                       `json:"projectileSpeedFactor,omitzero"`
	Range                      Optional[XMLInteger]                       `json:"range,omitzero"`
	Respawns                   Optional[StrictProvisionScriptRespawns]    `json:"respawns,omitzero"`
	RevengeDamagePercentage    Optional[XMLInteger]                       `json:"revengeDamagePercentage,omitzero"`
	SimpleMode                 Optional[XMLBool]                          `json:"simpleMode,omitzero"`
	SixthSenseDelay            Optional[XMLNumber]                        `json:"sixthSenseDelay,omitzero"`
	StopObservingDelayFactor   Optional[XMLNumber]                        `json:"stopObservingDelayFactor,omitzero"`
	TimeForComeBack            Optional[XMLNumber]                        `json:"timeForComeBack,omitzero"`
}

type StrictProvisionScriptAttributes struct {
	ChassisHealthFactor                 Optional[XMLNumber]  `json:"chassisHealthFactor,omitzero"`
	ChassisHysteresisHealthFactor       Optional[XMLNumber]  `json:"chassisHysteresisHealthFactor,omitzero"`
	ChassisMaxRegenHealthFactor         Optional[XMLNumber]  `json:"chassisMaxRegenHealthFactor,omitzero"`
	ChassisMovementShotDispersionFactor Optional[XMLNumber]  `json:"chassisMovementShotDispersionFactor,omitzero"`
	ChassisRegenFactor                  Optional[XMLNumber]  `json:"chassisRegenFactor,omitzero"`
	ChassisRotationShotDispersionFactor Optional[XMLNumber]  `json:"chassisRotationShotDispersionFactor,omitzero"`
	CircularVisionRadiusFactor          Optional[XMLNumber]  `json:"circularVisionRadiusFactor,omitzero"`
	CommonDamageFactor                  Optional[XMLNumber]  `json:"commonDamageFactor,omitzero"`
	EngineHealthFactor                  Optional[XMLNumber]  `json:"engineHealthFactor,omitzero"`
	EngineHysteresisHealthFactor        Optional[XMLNumber]  `json:"engineHysteresisHealthFactor,omitzero"`
	EngineMaxRegenHealthFactor          Optional[XMLNumber]  `json:"engineMaxRegenHealthFactor,omitzero"`
	EnginePowerFactor                   Optional[XMLNumber]  `json:"enginePowerFactor,omitzero"`
	EngineRegenFactor                   Optional[XMLNumber]  `json:"engineRegenFactor,omitzero"`
	EquipmentDurationFactor             Optional[XMLNumber]  `json:"equipmentDurationFactor,omitzero"`
	EquipmentReloadBoost                Optional[XMLNumber]  `json:"equipmentReloadBoost,omitzero"`
	FirmGroundPassabilityIncrease       Optional[XMLInteger] `json:"firmGroundPassabilityIncrease,omitzero"`
	GunAimingFactor                     Optional[XMLNumber]  `json:"gunAimingFactor,omitzero"`
	GunReloadFactor                     Optional[XMLNumber]  `json:"gunReloadFactor,omitzero"`
	GunTurretRotationDispersionFactor   Optional[XMLNumber]  `json:"gunTurretRotationDispersionFactor,omitzero"`
	HighExplosiveDamageFactor           Optional[XMLNumber]  `json:"highExplosiveDamageFactor,omitzero"`
	InvisibilityFactor                  Optional[XMLNumber]  `json:"invisibilityFactor,omitzero"`
	MediumGroundPassabilityIncrease     Optional[XMLInteger] `json:"mediumGroundPassabilityIncrease,omitzero"`
	PumpGunReloadFactor                 Optional[XMLNumber]  `json:"pumpGunReloadFactor,omitzero"`
	ReceiveDamageFactor                 Optional[XMLNumber]  `json:"receiveDamageFactor,omitzero"`
	RepairSpeedFactor                   Optional[XMLNumber]  `json:"repairSpeedFactor,omitzero"`
	ShotDispersionAngleFactor           Optional[XMLNumber]  `json:"shotDispersionAngleFactor,omitzero"`
	SpeedLimitsBackwardFactor           Optional[XMLNumber]  `json:"speedLimitsBackwardFactor,omitzero"`
	SpeedLimitsFactor                   Optional[XMLNumber]  `json:"speedLimitsFactor,omitzero"`
	SpeedLimitsForwardFactor            Optional[XMLNumber]  `json:"speedLimitsForwardFactor,omitzero"`
	TurretRotationSpeedFactor           Optional[XMLNumber]  `json:"turretRotationSpeedFactor,omitzero"`
	VehicleMaxHealthFactor              Optional[XMLNumber]  `json:"vehicleMaxHealthFactor,omitzero"`
	VehicleRotationSpeedFactor          Optional[XMLNumber]  `json:"vehicleRotationSpeedFactor,omitzero"`
}

type StrictProvisionScriptBonusValues struct {
	AimingTimeFactor            Optional[XMLNumber]                                                `json:"aimingTimeFactor,omitzero"`
	BkwdSpeedLimitBias          Optional[XMLInteger]                                               `json:"bkwdSpeedLimitBias,omitzero"`
	ChancesToHit                Optional[StrictProvisionScriptBonusValuesChancesToHit]             `json:"chancesToHit,omitzero"`
	CommonDamageFactor          Optional[XMLNumber]                                                `json:"commonDamageFactor,omitzero"`
	CrewChanceToHitFactor       Optional[XMLNumber]                                                `json:"crewChanceToHitFactor,omitzero"`
	CrewLevelIncrease           Optional[XMLInteger]                                               `json:"crewLevelIncrease,omitzero"`
	DealingRammingDamageFactor  Optional[XMLNumber]                                                `json:"dealingRammingDamageFactor,omitzero"`
	EnginePowerAbilityFactor    Optional[XMLNumber]                                                `json:"enginePowerAbilityFactor,omitzero"`
	EnginePowerFactor           Optional[XMLNumber]                                                `json:"enginePowerFactor,omitzero"`
	EnginePowerIncrease         Optional[XMLInteger]                                               `json:"enginePowerIncrease,omitzero"`
	ExplosivesHealthIncrease    Optional[XMLInteger]                                               `json:"explosivesHealthIncrease,omitzero"`
	Factor                      Optional[XMLNumber]                                                `json:"factor,omitzero"`
	FireProtectionIncrease      Optional[XMLInteger]                                               `json:"fireProtectionIncrease,omitzero"`
	FwdSpeedLimitBias           Optional[XMLInteger]                                               `json:"fwdSpeedLimitBias,omitzero"`
	GunAimingTimeBonus          Optional[XMLInteger]                                               `json:"gunAimingTimeBonus,omitzero"`
	GunReloadFactor             Optional[XMLNumber]                                                `json:"gunReloadFactor,omitzero"`
	GunReloadingBonus           Optional[XMLInteger]                                               `json:"gunReloadingBonus,omitzero"`
	HealFactor                  Optional[XMLNumber]                                                `json:"healFactor,omitzero"`
	HealRandomFactor            Optional[XMLNumber]                                                `json:"healRandomFactor,omitzero"`
	IncreaseDamageFactor        Optional[XMLNumber]                                                `json:"increaseDamageFactor,omitzero"`
	IncreasedDamageChance       Optional[XMLInteger]                                               `json:"increasedDamageChance,omitzero"`
	KillHealFactor              Optional[XMLNumber]                                                `json:"killHealFactor,omitzero"`
	LifestealFactor             Optional[XMLNumber]                                                `json:"lifestealFactor,omitzero"`
	LifestealRandomFactor       Optional[XMLNumber]                                                `json:"lifestealRandomFactor,omitzero"`
	MaxHealthPercentBonus       Optional[XMLInteger]                                               `json:"maxHealthPercentBonus,omitzero"`
	MaxStaticsDamagePercent     Optional[XMLInteger]                                               `json:"maxStaticsDamagePercent,omitzero"`
	ModulesMaxHpFactors         Optional[StrictProvisionScriptBonusValuesModulesMaxHpFactors]      `json:"modulesMaxHpFactors,omitzero"`
	ModulesMaxRegenHpFactors    Optional[StrictProvisionScriptBonusValuesModulesMaxRegenHpFactors] `json:"modulesMaxRegenHpFactors,omitzero"`
	RammingDamageFactor         Optional[XMLNumber]                                                `json:"rammingDamageFactor,omitzero"`
	RepairSpeedIncrease         Optional[XMLInteger]                                               `json:"repairSpeedIncrease,omitzero"`
	ShotDispersionAngleFactor   Optional[XMLNumber]                                                `json:"shotDispersionAngleFactor,omitzero"`
	SpeedLimitAbilityBonus      Optional[XMLInteger]                                               `json:"speedLimitAbilityBonus,omitzero"`
	SpeedLimitsFactor           Optional[XMLNumber]                                                `json:"speedLimitsFactor,omitzero"`
	TimeBetweenShots            Optional[XMLNumber]                                                `json:"timeBetweenShots,omitzero"`
	TurretRotationSpeedIncrease Optional[XMLInteger]                                               `json:"turretRotationSpeedIncrease,omitzero"`
	VisionRadiusBonus           Optional[XMLInteger]                                               `json:"visionRadiusBonus,omitzero"`
}

type StrictProvisionScriptBonusValuesChancesToHit struct {
	Modules Optional[StrictProvisionScriptBonusValuesChancesToHitModules] `json:"modules,omitzero"`
}

type StrictProvisionScriptBonusValuesChancesToHitModules struct {
	AmmoBayHealth         Optional[XMLNumber]             `json:"ammoBayHealth,omitzero"`
	EngineHealth          Optional[XMLNumber]             `json:"engineHealth,omitzero"`
	FuelTankHealth        Optional[XMLNumber]             `json:"fuelTankHealth,omitzero"`
	SurveyingDeviceHealth Optional[[]Optional[XMLNumber]] `json:"surveyingDeviceHealth,omitzero"`
	TurretRotatorHealth   Optional[XMLNumber]             `json:"turretRotatorHealth,omitzero"`
}

type StrictProvisionScriptBonusValuesModulesMaxHpFactors struct {
	Modules Optional[StrictProvisionScriptBonusValuesModulesMaxHpFactorsModules] `json:"modules,omitzero"`
}

type StrictProvisionScriptBonusValuesModulesMaxHpFactorsModules struct {
	AmmoBayHealth Optional[XMLNumber] `json:"ammoBayHealth,omitzero"`
}

type StrictProvisionScriptBonusValuesModulesMaxRegenHpFactors struct {
	Modules Optional[StrictProvisionScriptBonusValuesModulesMaxRegenHpFactorsModules] `json:"modules,omitzero"`
}

type StrictProvisionScriptBonusValuesModulesMaxRegenHpFactorsModules struct {
	AmmoBayHealth Optional[XMLNumber] `json:"ammoBayHealth,omitzero"`
}

type StrictProvisionScriptRespawns struct {
	Respawn Optional[[]Optional[StrictProvisionScriptRespawnsRespawnItem]] `json:"respawn,omitzero"`
}

type StrictProvisionScriptRespawnsRespawnItem struct {
	BonusDamage      Optional[XMLNumber]  `json:"bonusDamage,omitzero"`
	BonusEnginePower Optional[XMLNumber]  `json:"bonusEnginePower,omitzero"`
	BonusMaxSpeed    Optional[XMLNumber]  `json:"bonusMaxSpeed,omitzero"`
	BonusReloadSpeed Optional[XMLNumber]  `json:"bonusReloadSpeed,omitzero"`
	RespawnIndex     Optional[XMLInteger] `json:"respawnIndex,omitzero"`
}

type StrictProvisionVehicleFilter struct {
	Exclude Optional[StrictProvisionVehicleFilterExclude] `json:"exclude,omitzero"`
	Include Optional[StrictProvisionVehicleFilterInclude] `json:"include,omitzero"`
}

type StrictProvisionVehicleFilterExclude struct {
	Vehicle Optional[StrictProvisionVehicleFilterExcludeVehicle] `json:"vehicle,omitzero"`
}

type StrictProvisionVehicleFilterExcludeVehicle struct {
	Name Optional[string] `json:"name,omitzero"`
}

type StrictProvisionVehicleFilterInclude struct {
	Nations Optional[string]                                     `json:"nations,omitzero"`
	Vehicle Optional[StrictProvisionVehicleFilterIncludeVehicle] `json:"vehicle,omitzero"`
}

type StrictProvisionVehicleFilterIncludeVehicle struct {
	MaxLevel Optional[XMLInteger] `json:"maxLevel,omitzero"`
	MinLevel Optional[XMLInteger] `json:"minLevel,omitzero"`
	Name     Optional[string]     `json:"name,omitzero"`
}

// StrictProvisionDescrParamValue preserves the single and repeated XML forms.
type StrictProvisionDescrParamValue struct {
	Single   *StrictProvisionDescrParam
	Repeated []Optional[StrictProvisionDescrParam]
}

func (v StrictProvisionDescrParamValue) MarshalJSON() ([]byte, error) {
	if (v.Single == nil) == (v.Repeated == nil) {
		return nil, fmt.Errorf("description parameter must have one variant")
	}
	if v.Single != nil {
		return json.Marshal(v.Single)
	}
	return json.Marshal(v.Repeated)
}

// StrictProvisionScriptValue preserves a scalar XML script or its object form.
type StrictProvisionScriptValue struct {
	Scalar *string
	Object *StrictProvisionScript
}

func (v StrictProvisionScriptValue) MarshalJSON() ([]byte, error) {
	if (v.Scalar == nil) == (v.Object == nil) {
		return nil, fmt.Errorf("provision script must have one variant")
	}
	if v.Scalar != nil {
		return json.Marshal(*v.Scalar)
	}
	return json.Marshal(v.Object)
}
