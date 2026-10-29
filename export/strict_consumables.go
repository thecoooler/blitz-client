// Types describe XML shapes observed in the current consumables settings documents.
package export

type StrictConsumable struct {
	AbilityCost            Optional[string]                                      `json:"abilityCost,omitempty"`
	Category               Optional[string]                                      `json:"category,omitempty"`
	DescrParam             Optional[[]Optional[StrictConsumableDescrParamEntry]] `json:"descrParam,omitempty"`
	Description            Optional[string]                                      `json:"description,omitempty"`
	DisplayParams          Optional[StrictConsumableDisplayParams]               `json:"display_params,omitempty"`
	DynamicParams          Optional[StrictConsumableDynamicParams]               `json:"dynamic_params,omitempty"`
	GameModeFilter         Optional[StrictConsumableGameModeFilter]              `json:"gameModeFilter,omitempty"`
	Grade                  Optional[string]                                      `json:"grade,omitempty"`
	HideIconOnAbilityPanel Optional[StrictNull]                                  `json:"hideIconOnAbilityPanel,omitempty"`
	Icon                   Optional[string]                                      `json:"icon,omitempty"`
	Id                     Optional[string]                                      `json:"id,omitempty"`
	IsAbility              Optional[StrictNull]                                  `json:"isAbility,omitempty"`
	NextImprovement        Optional[string]                                      `json:"nextImprovement,omitempty"`
	Script                 Optional[StrictConsumableScript]                      `json:"script,omitempty"`
	Tags                   Optional[string]                                      `json:"tags,omitempty"`
	UserString             Optional[string]                                      `json:"userString,omitempty"`
	VehicleFilter          Optional[StrictConsumableVehicleFilter]               `json:"vehicleFilter,omitempty"`
}

type StrictConsumableDescrParamEntry struct {
	Name  Optional[string] `json:"name,omitempty"`
	Value Optional[string] `json:"value,omitempty"`
}

type StrictConsumableDisplayParams struct {
	AimingAccuracy    Optional[string] `json:"aimingAccuracy,omitempty"`
	AimingTime        Optional[string] `json:"aimingTime,omitempty"`
	AvgDamage         Optional[string] `json:"avgDamage,omitempty"`
	ClipReloadTime    Optional[string] `json:"clipReloadTime,omitempty"`
	Damage            Optional[string] `json:"damage,omitempty"`
	DamageFactor      Optional[string] `json:"damageFactor,omitempty"`
	DamageFactor2     Optional[string] `json:"damageFactor2,omitempty"`
	DamageFactor3     Optional[string] `json:"damageFactor3,omitempty"`
	EnginePower       Optional[string] `json:"enginePower,omitempty"`
	EnginePowerFactor Optional[string] `json:"enginePowerFactor,omitempty"`
	HealthFactor      Optional[string] `json:"healthFactor,omitempty"`
	MaxDamageFactor   Optional[string] `json:"maxDamageFactor,omitempty"`
	MaxVelocityFactor Optional[string] `json:"maxVelocityFactor,omitempty"`
	MinDamageFactor   Optional[string] `json:"minDamageFactor,omitempty"`
	MoveSpeed         Optional[string] `json:"moveSpeed,omitempty"`
	RotationSpeed     Optional[string] `json:"rotationSpeed,omitempty"`
}

type StrictConsumableDynamicParams struct {
	Percents Optional[string] `json:"percents,omitempty"`
}

type StrictConsumableGameModeFilter struct {
	Lunar                      Optional[StrictNull] `json:"Lunar,omitempty"`
	Arcade                     Optional[StrictNull] `json:"arcade,omitempty"`
	Battleroyale               Optional[StrictNull] `json:"battleroyale,omitempty"`
	Bossmode                   Optional[StrictNull] `json:"bossmode,omitempty"`
	Gravitizing                Optional[StrictNull] `json:"gravitizing,omitempty"`
	Hellgames                  Optional[StrictNull] `json:"hellgames,omitempty"`
	QuickTournamentArcade      Optional[StrictNull] `json:"quick_tournament_arcade,omitempty"`
	QuickTournamentBossmode    Optional[StrictNull] `json:"quick_tournament_bossmode,omitempty"`
	QuickTournamentGravitizing Optional[StrictNull] `json:"quick_tournament_gravitizing,omitempty"`
	QuickTournamentHellgames   Optional[StrictNull] `json:"quick_tournament_hellgames,omitempty"`
	QuickTournamentLunar       Optional[StrictNull] `json:"quick_tournament_lunar,omitempty"`
	TournamentArcade           Optional[StrictNull] `json:"tournament_arcade,omitempty"`
	TournamentBossmode         Optional[StrictNull] `json:"tournament_bossmode,omitempty"`
	TournamentGravitizing      Optional[StrictNull] `json:"tournament_gravitizing,omitempty"`
	TournamentHellgames        Optional[StrictNull] `json:"tournament_hellgames,omitempty"`
	TournamentLunar            Optional[StrictNull] `json:"tournament_lunar,omitempty"`
}

type StrictConsumableScript struct {
	Text                         Optional[string]                            `json:"#text,omitempty"`
	AbortVelocity                Optional[string]                            `json:"abortVelocity,omitempty"`
	Acceleration                 Optional[string]                            `json:"acceleration,omitempty"`
	ActivateStates               Optional[string]                            `json:"activateStates,omitempty"`
	ActivationDamageFactor       Optional[string]                            `json:"activationDamageFactor,omitempty"`
	AimingTimeFactor             Optional[string]                            `json:"aimingTimeFactor,omitempty"`
	AllyHealFactorByAllyHP       Optional[string]                            `json:"allyHealFactorByAllyHP,omitempty"`
	AttractorMass                Optional[string]                            `json:"attractorMass,omitempty"`
	Attributes                   Optional[StrictConsumableScriptAttributes]  `json:"attributes,omitempty"`
	Automatic                    Optional[string]                            `json:"automatic,omitempty"`
	BkwdSpeedLimitBias           Optional[string]                            `json:"bkwdSpeedLimitBias,omitempty"`
	BonusValues                  Optional[StrictConsumableScriptBonusValues] `json:"bonusValues,omitempty"`
	BreakTracks                  Optional[string]                            `json:"breakTracks,omitempty"`
	BurstDamagePercent           Optional[string]                            `json:"burstDamagePercent,omitempty"`
	BurstForce                   Optional[string]                            `json:"burstForce,omitempty"`
	BurstRange                   Optional[string]                            `json:"burstRange,omitempty"`
	ClipReloadTimeFactor         Optional[string]                            `json:"clipReloadTimeFactor,omitempty"`
	ConflictExtras               Optional[string]                            `json:"conflictExtras,omitempty"`
	ContusionChances             Optional[string]                            `json:"contusionChances,omitempty"`
	Cooldown                     Optional[string]                            `json:"cooldown,omitempty"`
	CooldownReducePercent        Optional[string]                            `json:"cooldownReducePercent,omitempty"`
	DamageFactor                 Optional[string]                            `json:"damageFactor,omitempty"`
	DamageFactors                Optional[string]                            `json:"damageFactors,omitempty"`
	DeactivateStates             Optional[string]                            `json:"deactivateStates,omitempty"`
	DisableTracksCollisionDamage Optional[string]                            `json:"disableTracksCollisionDamage,omitempty"`
	Distance                     Optional[string]                            `json:"distance,omitempty"`
	DistanceOffset               Optional[string]                            `json:"distanceOffset,omitempty"`
	Duration                     Optional[string]                            `json:"duration,omitempty"`
	EffectDuration               Optional[string]                            `json:"effectDuration,omitempty"`
	EffectExtraName              Optional[string]                            `json:"effectExtraName,omitempty"`
	EnginePowerIncrease          Optional[string]                            `json:"enginePowerIncrease,omitempty"`
	ExtrasMustBeStopped          Optional[string]                            `json:"extrasMustBeStopped,omitempty"`
	FlyTime                      Optional[string]                            `json:"flyTime,omitempty"`
	FwdSpeedLimitBias            Optional[string]                            `json:"fwdSpeedLimitBias,omitempty"`
	GunReloadTimeFactor          Optional[string]                            `json:"gunReloadTimeFactor,omitempty"`
	HealFactorPercent            Optional[string]                            `json:"healFactorPercent,omitempty"`
	HealWaveDelay                Optional[string]                            `json:"healWaveDelay,omitempty"`
	HealWavesCount               Optional[string]                            `json:"healWavesCount,omitempty"`
	HealthFactor                 Optional[string]                            `json:"healthFactor,omitempty"`
	InnerCooldown                Optional[string]                            `json:"innerCooldown,omitempty"`
	KeepSpoted                   Optional[string]                            `json:"keepSpoted,omitempty"`
	MaxAngleDegrees              Optional[string]                            `json:"maxAngleDegrees,omitempty"`
	MaxDamageFactor              Optional[string]                            `json:"maxDamageFactor,omitempty"`
	MaxHealthDamageFactor        Optional[string]                            `json:"maxHealthDamageFactor,omitempty"`
	MaxHealthDamagePercent       Optional[string]                            `json:"maxHealthDamagePercent,omitempty"`
	MaxHealthSelfDamageFactor    Optional[string]                            `json:"maxHealthSelfDamageFactor,omitempty"`
	MinAngleDegrees              Optional[string]                            `json:"minAngleDegrees,omitempty"`
	MinDamageFactor              Optional[string]                            `json:"minDamageFactor,omitempty"`
	Multipliers                  Optional[StrictConsumableScriptMultipliers] `json:"multipliers,omitempty"`
	Overrides                    Optional[StrictConsumableScriptOverrides]   `json:"overrides,omitempty"`
	PoisonTickDelay              Optional[string]                            `json:"poisonTickDelay,omitempty"`
	PoisonTicks                  Optional[string]                            `json:"poisonTicks,omitempty"`
	ProjectileSpeed              Optional[string]                            `json:"projectileSpeed,omitempty"`
	QuantityLimit                Optional[string]                            `json:"quantityLimit,omitempty"`
	QuantityPerUse               Optional[string]                            `json:"quantityPerUse,omitempty"`
	Radius                       Optional[string]                            `json:"radius,omitempty"`
	Range                        Optional[string]                            `json:"range,omitempty"`
	RotationSpeedFactor          Optional[string]                            `json:"rotationSpeedFactor,omitempty"`
	SelfHealReducePercent        Optional[string]                            `json:"selfHealReducePercent,omitempty"`
	SelfInstantRestorePercent    Optional[string]                            `json:"selfInstantRestorePercent,omitempty"`
	ShieldCoef                   Optional[string]                            `json:"shieldCoef,omitempty"`
	ShieldDuration               Optional[string]                            `json:"shieldDuration,omitempty"`
	ShotDispersionAngleFactor    Optional[string]                            `json:"shotDispersionAngleFactor,omitempty"`
	ShotEffect                   Optional[string]                            `json:"shotEffect,omitempty"`
	StartCooldown                Optional[string]                            `json:"startCooldown,omitempty"`
	Statics                      Optional[StrictConsumableScriptStatics]     `json:"statics,omitempty"`
	TargetExtrasMustBeStopped    Optional[string]                            `json:"targetExtrasMustBeStopped,omitempty"`
	TimeSeconds                  Optional[string]                            `json:"timeSeconds,omitempty"`
}

type StrictConsumableScriptAttributes struct {
	ClipReloadFactor          Optional[string] `json:"clipReloadFactor,omitempty"`
	CommonDamageFactor        Optional[string] `json:"commonDamageFactor,omitempty"`
	GunReloadFactor           Optional[string] `json:"gunReloadFactor,omitempty"`
	PumpGunReloadFactor       Optional[string] `json:"pumpGunReloadFactor,omitempty"`
	ReceiveDamageFactor       Optional[string] `json:"receiveDamageFactor,omitempty"`
	ShotDispersionAngleFactor Optional[string] `json:"shotDispersionAngleFactor,omitempty"`
}

type StrictConsumableScriptBonusValues struct {
	ChassisHealthFactor      Optional[string] `json:"chassisHealthFactor,omitempty"`
	DamagePercentBonus       Optional[string] `json:"damagePercentBonus,omitempty"`
	DamageReflectionPercent  Optional[string] `json:"damageReflectionPercent,omitempty"`
	EnginePowerAbilityFactor Optional[string] `json:"enginePowerAbilityFactor,omitempty"`
	EnginePowerFactor        Optional[string] `json:"enginePowerFactor,omitempty"`
	EnginePowerIncrease      Optional[string] `json:"enginePowerIncrease,omitempty"`
	GunReloadSpeedIncrease   Optional[string] `json:"gunReloadSpeedIncrease,omitempty"`
	MaxVelocityFactor        Optional[string] `json:"maxVelocityFactor,omitempty"`
	RammingDamageFactor      Optional[string] `json:"rammingDamageFactor,omitempty"`
	RotationSpeedFactor      Optional[string] `json:"rotationSpeedFactor,omitempty"`
	SacrificeLoss            Optional[string] `json:"sacrificeLoss,omitempty"`
	SpeedLimitAbilityBonus   Optional[string] `json:"speedLimitAbilityBonus,omitempty"`
	TerrainResistanceFactor  Optional[string] `json:"terrainResistanceFactor,omitempty"`
}

type StrictConsumableScriptMultipliers struct {
	AngularSpeedMultiplier          Optional[string] `json:"angularSpeedMultiplier,omitempty"`
	BackwardSpeedMultiplier         Optional[string] `json:"backwardSpeedMultiplier,omitempty"`
	EnginePowerMultiplier           Optional[string] `json:"enginePowerMultiplier,omitempty"`
	ForwardSpeedMultiplier          Optional[string] `json:"forwardSpeedMultiplier,omitempty"`
	IncomingRammingDamageMultiplier Optional[string] `json:"incomingRammingDamageMultiplier,omitempty"`
	OutgoingRammingDamageMultiplier Optional[string] `json:"outgoingRammingDamageMultiplier,omitempty"`
}

type StrictConsumableScriptOverrides struct {
	Override Optional[[]Optional[StrictConsumableScriptOverridesOverrideEntry]] `json:"override,omitempty"`
}

type StrictConsumableScriptOverridesOverrideEntry struct {
	AttractorMass      Optional[string] `json:"attractorMass,omitempty"`
	BurstDamagePercent Optional[string] `json:"burstDamagePercent,omitempty"`
	BurstForce         Optional[string] `json:"burstForce,omitempty"`
	BurstRange         Optional[string] `json:"burstRange,omitempty"`
	Cooldown           Optional[string] `json:"cooldown,omitempty"`
	EffectDuration     Optional[string] `json:"effectDuration,omitempty"`
	MaxVehLevel        Optional[string] `json:"maxVehLevel,omitempty"`
	MinVehLevel        Optional[string] `json:"minVehLevel,omitempty"`
	Radius             Optional[string] `json:"radius,omitempty"`
}

type StrictConsumableScriptStatics struct {
	FlatteningCoefficient                  Optional[string] `json:"flatteningCoefficient,omitempty"`
	IncomingDefaultRammingDamageMultiplier Optional[string] `json:"incomingDefaultRammingDamageMultiplier,omitempty"`
	OutgoingDefaultRammingDamageMultiplier Optional[string] `json:"outgoingDefaultRammingDamageMultiplier,omitempty"`
	StaticForce                            Optional[string] `json:"staticForce,omitempty"`
	StaticRestForce                        Optional[string] `json:"staticRestForce,omitempty"`
	WideningCoefficient                    Optional[string] `json:"wideningCoefficient,omitempty"`
}

type StrictConsumableVehicleFilter struct {
	Exclude Optional[StrictConsumableVehicleFilterExclude] `json:"exclude,omitempty"`
	Include Optional[StrictConsumableVehicleFilterInclude] `json:"include,omitempty"`
}

type StrictConsumableVehicleFilterExclude struct {
	Vehicle Optional[StrictConsumableVehicleFilterExcludeVehicle] `json:"vehicle,omitempty"`
}

type StrictConsumableVehicleFilterExcludeVehicle struct {
	ExtendedTags Optional[string] `json:"extendedTags,omitempty"`
	Name         Optional[string] `json:"name,omitempty"`
}

type StrictConsumableVehicleFilterInclude struct {
	Vehicle Optional[StrictConsumableVehicleFilterIncludeVehicle] `json:"vehicle,omitempty"`
}

type StrictConsumableVehicleFilterIncludeVehicle struct {
	MaxLevel Optional[string] `json:"maxLevel,omitempty"`
	MinLevel Optional[string] `json:"minLevel,omitempty"`
	Name     Optional[string] `json:"name,omitempty"`
}
