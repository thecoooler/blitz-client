// Types below describe the XML shapes observed in the current provisions export.
package export

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StrictProvisions is keyed by source mode, then provision name.
type StrictProvisions map[string]map[string]StrictProvision

type StrictProvision struct {
	AbilityCost            Optional[string]                         `json:"abilityCost,omitempty"`
	Category               Optional[string]                         `json:"category,omitempty"`
	DescrParam             Optional[StrictProvisionDescrParamValue] `json:"descrParam,omitempty"`
	Description            Optional[string]                         `json:"description,omitempty"`
	DisplayParams          Optional[StrictProvisionDisplayParams]   `json:"display_params,omitempty"`
	DynamicParams          Optional[StrictProvisionDynamicParams]   `json:"dynamic_params,omitempty"`
	ExtraEffectDuration    Optional[string]                         `json:"extraEffectDuration,omitempty"`
	GameModeFilter         Optional[StrictProvisionGameModeFilter]  `json:"gameModeFilter,omitempty"`
	GeometryFilter         Optional[StrictProvisionGeometryFilter]  `json:"geometryFilter,omitempty"`
	Grade                  Optional[string]                         `json:"grade,omitempty"`
	HideIconOnAbilityPanel Optional[StrictNull]                     `json:"hideIconOnAbilityPanel,omitempty"`
	Icon                   Optional[string]                         `json:"icon,omitempty"`
	Id                     Optional[string]                         `json:"id,omitempty"`
	IsAbility              Optional[StrictNull]                     `json:"isAbility,omitempty"`
	NextImprovement        Optional[string]                         `json:"nextImprovement,omitempty"`
	Price                  Optional[StrictProvisionPrice]           `json:"price,omitempty"`
	Script                 Optional[StrictProvisionScriptValue]     `json:"script,omitempty"`
	Tags                   Optional[string]                         `json:"tags,omitempty"`
	UserString             Optional[string]                         `json:"userString,omitempty"`
	VehicleFilter          Optional[StrictProvisionVehicleFilter]   `json:"vehicleFilter,omitempty"`
}

func (v *StrictProvision) UnmarshalJSON(data []byte) error {
	type plain StrictProvision
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionDescrParam struct {
	Name  Optional[string] `json:"name,omitempty"`
	Value Optional[string] `json:"value,omitempty"`
}

func (v *StrictProvisionDescrParam) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionDescrParam
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionDisplayParams struct {
	AimingTimeFactor                Optional[string] `json:"aimingTimeFactor,omitempty"`
	AntiHighExplosiveFactor         Optional[string] `json:"antiHighExplosiveFactor,omitempty"`
	BkwdSpeedLimit                  Optional[string] `json:"bkwdSpeedLimit,omitempty"`
	CommonDamageFactor              Optional[string] `json:"commonDamageFactor,omitempty"`
	CrewProtectionIncrease          Optional[string] `json:"crewProtectionIncrease,omitempty"`
	DealingRammingDamageFactor      Optional[string] `json:"dealingRammingDamageFactor,omitempty"`
	EnginePowerIncrease             Optional[string] `json:"enginePowerIncrease,omitempty"`
	EnginePower                     Optional[string] `json:"engine__power,omitempty"`
	ExplosivesHealthIncrease        Optional[string] `json:"explosivesHealthIncrease,omitempty"`
	FireProtectionIncrease          Optional[string] `json:"fireProtectionIncrease,omitempty"`
	FirmGroundPassabilityIncrease   Optional[string] `json:"firmGroundPassabilityIncrease,omitempty"`
	FwdSpeedLimit                   Optional[string] `json:"fwdSpeedLimit,omitempty"`
	GunReloadTime                   Optional[string] `json:"gun__reloadTime,omitempty"`
	IncreasedDamageChance           Optional[string] `json:"increasedDamageChance,omitempty"`
	MaxHpBonusPercent               Optional[string] `json:"maxHpBonusPercent,omitempty"`
	MediumGroundPassabilityIncrease Optional[string] `json:"mediumGroundPassabilityIncrease,omitempty"`
	ProjectileSpeed                 Optional[string] `json:"projectileSpeed,omitempty"`
	RammingDamageFactor             Optional[string] `json:"rammingDamageFactor,omitempty"`
	RepairSpeedFactor               Optional[string] `json:"repairSpeedFactor,omitempty"`
	RepairSpeedIncrease             Optional[string] `json:"repairSpeedIncrease,omitempty"`
	ShotDispersionAngleFactor       Optional[string] `json:"shotDispersionAngleFactor,omitempty"`
}

func (v *StrictProvisionDisplayParams) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionDisplayParams
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionDynamicParams struct {
	Percents Optional[string] `json:"percents,omitempty"`
}

func (v *StrictProvisionDynamicParams) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionDynamicParams
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionGameModeFilter struct {
	Arcade                     Optional[StrictNull] `json:"arcade,omitempty"`
	Battleroyale               Optional[StrictNull] `json:"battleroyale,omitempty"`
	Bossmode                   Optional[StrictNull] `json:"bossmode,omitempty"`
	Gravitizing                Optional[StrictNull] `json:"gravitizing,omitempty"`
	Hellgames                  Optional[StrictNull] `json:"hellgames,omitempty"`
	QuickTournamentArcade      Optional[StrictNull] `json:"quick_tournament_arcade,omitempty"`
	QuickTournamentBossmode    Optional[StrictNull] `json:"quick_tournament_bossmode,omitempty"`
	QuickTournamentGravitizing Optional[StrictNull] `json:"quick_tournament_gravitizing,omitempty"`
	QuickTournamentHellgames   Optional[StrictNull] `json:"quick_tournament_hellgames,omitempty"`
	QuickTournamentVampiric    Optional[StrictNull] `json:"quick_tournament_vampiric,omitempty"`
	TournamentArcade           Optional[StrictNull] `json:"tournament_arcade,omitempty"`
	TournamentBossmode         Optional[StrictNull] `json:"tournament_bossmode,omitempty"`
	TournamentGravitizing      Optional[StrictNull] `json:"tournament_gravitizing,omitempty"`
	TournamentHellgames        Optional[StrictNull] `json:"tournament_hellgames,omitempty"`
	TournamentVampiric         Optional[StrictNull] `json:"tournament_vampiric,omitempty"`
	Vampiric                   Optional[StrictNull] `json:"vampiric,omitempty"`
}

func (v *StrictProvisionGameModeFilter) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionGameModeFilter
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionGeometryFilter struct {
	Include Optional[string] `json:"include,omitempty"`
}

func (v *StrictProvisionGeometryFilter) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionGeometryFilter
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionPrice struct {
	Level1  Optional[string] `json:"level1,omitempty"`
	Level10 Optional[string] `json:"level10,omitempty"`
	Level2  Optional[string] `json:"level2,omitempty"`
	Level3  Optional[string] `json:"level3,omitempty"`
	Level4  Optional[string] `json:"level4,omitempty"`
	Level5  Optional[string] `json:"level5,omitempty"`
	Level6  Optional[string] `json:"level6,omitempty"`
	Level7  Optional[string] `json:"level7,omitempty"`
	Level8  Optional[string] `json:"level8,omitempty"`
	Level9  Optional[string] `json:"level9,omitempty"`
}

func (v *StrictProvisionPrice) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionPrice
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScript struct {
	Text                       Optional[string]                           `json:"#text,omitempty"`
	ARMORPIERCING              Optional[string]                           `json:"ARMOR_PIERCING,omitempty"`
	ARMORPIERCINGCR            Optional[string]                           `json:"ARMOR_PIERCING_CR,omitempty"`
	HIGHEXPLOSIVE              Optional[string]                           `json:"HIGH_EXPLOSIVE,omitempty"`
	HOLLOWCHARGE               Optional[string]                           `json:"HOLLOW_CHARGE,omitempty"`
	Attributes                 Optional[StrictProvisionScriptAttributes]  `json:"attributes,omitempty"`
	BonusValues                Optional[StrictProvisionScriptBonusValues] `json:"bonusValues,omitempty"`
	CurseHeal                  Optional[string]                           `json:"curseHeal,omitempty"`
	CurseReduction             Optional[string]                           `json:"curseReduction,omitempty"`
	CurseStrength              Optional[string]                           `json:"curseStrength,omitempty"`
	DamageReductionPercent     Optional[string]                           `json:"damageReductionPercent,omitempty"`
	EffectDuration             Optional[string]                           `json:"effectDuration,omitempty"`
	ExplodeDamagePercent       Optional[string]                           `json:"explodeDamagePercent,omitempty"`
	HpStockPercent             Optional[string]                           `json:"hpStockPercent,omitempty"`
	InvisibilityInMovingFactor Optional[string]                           `json:"invisibilityInMovingFactor,omitempty"`
	MaxBonusSources            Optional[string]                           `json:"maxBonusSources,omitempty"`
	ProjectileSpeedFactor      Optional[string]                           `json:"projectileSpeedFactor,omitempty"`
	Range                      Optional[string]                           `json:"range,omitempty"`
	Respawns                   Optional[StrictProvisionScriptRespawns]    `json:"respawns,omitempty"`
	RevengeDamagePercentage    Optional[string]                           `json:"revengeDamagePercentage,omitempty"`
	SimpleMode                 Optional[string]                           `json:"simpleMode,omitempty"`
	SixthSenseDelay            Optional[string]                           `json:"sixthSenseDelay,omitempty"`
	StopObservingDelayFactor   Optional[string]                           `json:"stopObservingDelayFactor,omitempty"`
	TimeForComeBack            Optional[string]                           `json:"timeForComeBack,omitempty"`
}

func (v *StrictProvisionScript) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScript
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScriptAttributes struct {
	ChassisHealthFactor                 Optional[string] `json:"chassisHealthFactor,omitempty"`
	ChassisHysteresisHealthFactor       Optional[string] `json:"chassisHysteresisHealthFactor,omitempty"`
	ChassisMaxRegenHealthFactor         Optional[string] `json:"chassisMaxRegenHealthFactor,omitempty"`
	ChassisMovementShotDispersionFactor Optional[string] `json:"chassisMovementShotDispersionFactor,omitempty"`
	ChassisRegenFactor                  Optional[string] `json:"chassisRegenFactor,omitempty"`
	ChassisRotationShotDispersionFactor Optional[string] `json:"chassisRotationShotDispersionFactor,omitempty"`
	CircularVisionRadiusFactor          Optional[string] `json:"circularVisionRadiusFactor,omitempty"`
	CommonDamageFactor                  Optional[string] `json:"commonDamageFactor,omitempty"`
	EngineHealthFactor                  Optional[string] `json:"engineHealthFactor,omitempty"`
	EngineHysteresisHealthFactor        Optional[string] `json:"engineHysteresisHealthFactor,omitempty"`
	EngineMaxRegenHealthFactor          Optional[string] `json:"engineMaxRegenHealthFactor,omitempty"`
	EnginePowerFactor                   Optional[string] `json:"enginePowerFactor,omitempty"`
	EngineRegenFactor                   Optional[string] `json:"engineRegenFactor,omitempty"`
	EquipmentDurationFactor             Optional[string] `json:"equipmentDurationFactor,omitempty"`
	EquipmentReloadBoost                Optional[string] `json:"equipmentReloadBoost,omitempty"`
	FirmGroundPassabilityIncrease       Optional[string] `json:"firmGroundPassabilityIncrease,omitempty"`
	GunAimingFactor                     Optional[string] `json:"gunAimingFactor,omitempty"`
	GunReloadFactor                     Optional[string] `json:"gunReloadFactor,omitempty"`
	GunTurretRotationDispersionFactor   Optional[string] `json:"gunTurretRotationDispersionFactor,omitempty"`
	HighExplosiveDamageFactor           Optional[string] `json:"highExplosiveDamageFactor,omitempty"`
	InvisibilityFactor                  Optional[string] `json:"invisibilityFactor,omitempty"`
	MediumGroundPassabilityIncrease     Optional[string] `json:"mediumGroundPassabilityIncrease,omitempty"`
	PumpGunReloadFactor                 Optional[string] `json:"pumpGunReloadFactor,omitempty"`
	ReceiveDamageFactor                 Optional[string] `json:"receiveDamageFactor,omitempty"`
	RepairSpeedFactor                   Optional[string] `json:"repairSpeedFactor,omitempty"`
	ShotDispersionAngleFactor           Optional[string] `json:"shotDispersionAngleFactor,omitempty"`
	SpeedLimitsBackwardFactor           Optional[string] `json:"speedLimitsBackwardFactor,omitempty"`
	SpeedLimitsFactor                   Optional[string] `json:"speedLimitsFactor,omitempty"`
	SpeedLimitsForwardFactor            Optional[string] `json:"speedLimitsForwardFactor,omitempty"`
	TurretRotationSpeedFactor           Optional[string] `json:"turretRotationSpeedFactor,omitempty"`
	VehicleMaxHealthFactor              Optional[string] `json:"vehicleMaxHealthFactor,omitempty"`
	VehicleRotationSpeedFactor          Optional[string] `json:"vehicleRotationSpeedFactor,omitempty"`
}

func (v *StrictProvisionScriptAttributes) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScriptAttributes
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScriptBonusValues struct {
	AimingTimeFactor            Optional[string]                                                   `json:"aimingTimeFactor,omitempty"`
	BkwdSpeedLimitBias          Optional[string]                                                   `json:"bkwdSpeedLimitBias,omitempty"`
	ChancesToHit                Optional[StrictProvisionScriptBonusValuesChancesToHit]             `json:"chancesToHit,omitempty"`
	CommonDamageFactor          Optional[string]                                                   `json:"commonDamageFactor,omitempty"`
	CrewChanceToHitFactor       Optional[string]                                                   `json:"crewChanceToHitFactor,omitempty"`
	CrewLevelIncrease           Optional[string]                                                   `json:"crewLevelIncrease,omitempty"`
	DealingRammingDamageFactor  Optional[string]                                                   `json:"dealingRammingDamageFactor,omitempty"`
	EnginePowerAbilityFactor    Optional[string]                                                   `json:"enginePowerAbilityFactor,omitempty"`
	EnginePowerFactor           Optional[string]                                                   `json:"enginePowerFactor,omitempty"`
	EnginePowerIncrease         Optional[string]                                                   `json:"enginePowerIncrease,omitempty"`
	ExplosivesHealthIncrease    Optional[string]                                                   `json:"explosivesHealthIncrease,omitempty"`
	Factor                      Optional[string]                                                   `json:"factor,omitempty"`
	FireProtectionIncrease      Optional[string]                                                   `json:"fireProtectionIncrease,omitempty"`
	FwdSpeedLimitBias           Optional[string]                                                   `json:"fwdSpeedLimitBias,omitempty"`
	GunAimingTimeBonus          Optional[string]                                                   `json:"gunAimingTimeBonus,omitempty"`
	GunReloadFactor             Optional[string]                                                   `json:"gunReloadFactor,omitempty"`
	GunReloadingBonus           Optional[string]                                                   `json:"gunReloadingBonus,omitempty"`
	HealFactor                  Optional[string]                                                   `json:"healFactor,omitempty"`
	HealRandomFactor            Optional[string]                                                   `json:"healRandomFactor,omitempty"`
	IncreaseDamageFactor        Optional[string]                                                   `json:"increaseDamageFactor,omitempty"`
	IncreasedDamageChance       Optional[string]                                                   `json:"increasedDamageChance,omitempty"`
	KillHealFactor              Optional[string]                                                   `json:"killHealFactor,omitempty"`
	LifestealFactor             Optional[string]                                                   `json:"lifestealFactor,omitempty"`
	LifestealRandomFactor       Optional[string]                                                   `json:"lifestealRandomFactor,omitempty"`
	MaxHealthPercentBonus       Optional[string]                                                   `json:"maxHealthPercentBonus,omitempty"`
	MaxStaticsDamagePercent     Optional[string]                                                   `json:"maxStaticsDamagePercent,omitempty"`
	ModulesMaxHpFactors         Optional[StrictProvisionScriptBonusValuesModulesMaxHpFactors]      `json:"modulesMaxHpFactors,omitempty"`
	ModulesMaxRegenHpFactors    Optional[StrictProvisionScriptBonusValuesModulesMaxRegenHpFactors] `json:"modulesMaxRegenHpFactors,omitempty"`
	RammingDamageFactor         Optional[string]                                                   `json:"rammingDamageFactor,omitempty"`
	RepairSpeedIncrease         Optional[string]                                                   `json:"repairSpeedIncrease,omitempty"`
	ShotDispersionAngleFactor   Optional[string]                                                   `json:"shotDispersionAngleFactor,omitempty"`
	SpeedLimitAbilityBonus      Optional[string]                                                   `json:"speedLimitAbilityBonus,omitempty"`
	SpeedLimitsFactor           Optional[string]                                                   `json:"speedLimitsFactor,omitempty"`
	TimeBetweenShots            Optional[string]                                                   `json:"timeBetweenShots,omitempty"`
	TurretRotationSpeedIncrease Optional[string]                                                   `json:"turretRotationSpeedIncrease,omitempty"`
	VisionRadiusBonus           Optional[string]                                                   `json:"visionRadiusBonus,omitempty"`
}

func (v *StrictProvisionScriptBonusValues) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScriptBonusValues
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScriptBonusValuesChancesToHit struct {
	Modules Optional[StrictProvisionScriptBonusValuesChancesToHitModules] `json:"modules,omitempty"`
}

func (v *StrictProvisionScriptBonusValuesChancesToHit) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScriptBonusValuesChancesToHit
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScriptBonusValuesChancesToHitModules struct {
	AmmoBayHealth         Optional[string]             `json:"ammoBayHealth,omitempty"`
	EngineHealth          Optional[string]             `json:"engineHealth,omitempty"`
	FuelTankHealth        Optional[string]             `json:"fuelTankHealth,omitempty"`
	SurveyingDeviceHealth Optional[[]Optional[string]] `json:"surveyingDeviceHealth,omitempty"`
	TurretRotatorHealth   Optional[string]             `json:"turretRotatorHealth,omitempty"`
}

func (v *StrictProvisionScriptBonusValuesChancesToHitModules) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScriptBonusValuesChancesToHitModules
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScriptBonusValuesModulesMaxHpFactors struct {
	Modules Optional[StrictProvisionScriptBonusValuesModulesMaxHpFactorsModules] `json:"modules,omitempty"`
}

func (v *StrictProvisionScriptBonusValuesModulesMaxHpFactors) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScriptBonusValuesModulesMaxHpFactors
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScriptBonusValuesModulesMaxHpFactorsModules struct {
	AmmoBayHealth Optional[string] `json:"ammoBayHealth,omitempty"`
}

func (v *StrictProvisionScriptBonusValuesModulesMaxHpFactorsModules) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScriptBonusValuesModulesMaxHpFactorsModules
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScriptBonusValuesModulesMaxRegenHpFactors struct {
	Modules Optional[StrictProvisionScriptBonusValuesModulesMaxRegenHpFactorsModules] `json:"modules,omitempty"`
}

func (v *StrictProvisionScriptBonusValuesModulesMaxRegenHpFactors) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScriptBonusValuesModulesMaxRegenHpFactors
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScriptBonusValuesModulesMaxRegenHpFactorsModules struct {
	AmmoBayHealth Optional[string] `json:"ammoBayHealth,omitempty"`
}

func (v *StrictProvisionScriptBonusValuesModulesMaxRegenHpFactorsModules) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScriptBonusValuesModulesMaxRegenHpFactorsModules
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScriptRespawns struct {
	Respawn Optional[[]Optional[StrictProvisionScriptRespawnsRespawnItem]] `json:"respawn,omitempty"`
}

func (v *StrictProvisionScriptRespawns) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScriptRespawns
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionScriptRespawnsRespawnItem struct {
	BonusDamage      Optional[string] `json:"bonusDamage,omitempty"`
	BonusEnginePower Optional[string] `json:"bonusEnginePower,omitempty"`
	BonusMaxSpeed    Optional[string] `json:"bonusMaxSpeed,omitempty"`
	BonusReloadSpeed Optional[string] `json:"bonusReloadSpeed,omitempty"`
	RespawnIndex     Optional[string] `json:"respawnIndex,omitempty"`
}

func (v *StrictProvisionScriptRespawnsRespawnItem) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionScriptRespawnsRespawnItem
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionVehicleFilter struct {
	Exclude Optional[StrictProvisionVehicleFilterExclude] `json:"exclude,omitempty"`
	Include Optional[StrictProvisionVehicleFilterInclude] `json:"include,omitempty"`
}

func (v *StrictProvisionVehicleFilter) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionVehicleFilter
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionVehicleFilterExclude struct {
	Vehicle Optional[StrictProvisionVehicleFilterExcludeVehicle] `json:"vehicle,omitempty"`
}

func (v *StrictProvisionVehicleFilterExclude) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionVehicleFilterExclude
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionVehicleFilterExcludeVehicle struct {
	Name Optional[string] `json:"name,omitempty"`
}

func (v *StrictProvisionVehicleFilterExcludeVehicle) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionVehicleFilterExcludeVehicle
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionVehicleFilterInclude struct {
	Nations Optional[string]                                     `json:"nations,omitempty"`
	Vehicle Optional[StrictProvisionVehicleFilterIncludeVehicle] `json:"vehicle,omitempty"`
}

func (v *StrictProvisionVehicleFilterInclude) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionVehicleFilterInclude
	return decodeNonNullObject(data, (*plain)(v))
}

type StrictProvisionVehicleFilterIncludeVehicle struct {
	MaxLevel Optional[string] `json:"maxLevel,omitempty"`
	MinLevel Optional[string] `json:"minLevel,omitempty"`
	Name     Optional[string] `json:"name,omitempty"`
}

func (v *StrictProvisionVehicleFilterIncludeVehicle) UnmarshalJSON(data []byte) error {
	type plain StrictProvisionVehicleFilterIncludeVehicle
	return decodeNonNullObject(data, (*plain)(v))
}

// StrictProvisionDescrParamValue preserves the single and repeated XML forms.
type StrictProvisionDescrParamValue struct {
	Single   *StrictProvisionDescrParam
	Repeated []Optional[StrictProvisionDescrParam]
}

func (v *StrictProvisionDescrParamValue) UnmarshalJSON(data []byte) error {
	d := bytes.TrimSpace(data)
	if len(d) == 0 {
		return fmt.Errorf("empty description parameter")
	}
	switch d[0] {
	case '{':
		var x StrictProvisionDescrParam
		if err := decodeStrictJSON(d, &x); err != nil {
			return err
		}
		*v = StrictProvisionDescrParamValue{Single: &x}
	case '[':
		var x []Optional[StrictProvisionDescrParam]
		if err := decodeStrictJSON(d, &x); err != nil {
			return err
		}
		*v = StrictProvisionDescrParamValue{Repeated: x}
	default:
		return fmt.Errorf("description parameter must be an object or array")
	}
	return nil
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

func (v *StrictProvisionScriptValue) UnmarshalJSON(data []byte) error {
	d := bytes.TrimSpace(data)
	if len(d) == 0 {
		return fmt.Errorf("empty provision script")
	}
	switch d[0] {
	case '"':
		var x string
		if err := decodeStrictJSON(d, &x); err != nil {
			return err
		}
		*v = StrictProvisionScriptValue{Scalar: &x}
	case '{':
		var x StrictProvisionScript
		if err := decodeStrictJSON(d, &x); err != nil {
			return err
		}
		*v = StrictProvisionScriptValue{Object: &x}
	default:
		return fmt.Errorf("provision script must be a string or object")
	}
	return nil
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
