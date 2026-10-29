// Types describe XML shapes observed in the current consumables settings documents.
package export

type StrictConsumable struct {
	AbilityCost            Optional[XMLInteger]                                  `json:"abilityCost,omitzero"`
	Category               Optional[string]                                      `json:"category,omitzero"`
	DescrParam             Optional[[]Optional[StrictConsumableDescrParamEntry]] `json:"descrParam,omitzero"`
	Description            Optional[string]                                      `json:"description,omitzero"`
	DisplayParams          Optional[StrictConsumableDisplayParams]               `json:"display_params,omitzero"`
	DynamicParams          Optional[StrictConsumableDynamicParams]               `json:"dynamic_params,omitzero"`
	GameModeFilter         Optional[StrictConsumableGameModeFilter]              `json:"gameModeFilter,omitzero"`
	Grade                  Optional[XMLInteger]                                  `json:"grade,omitzero"`
	HideIconOnAbilityPanel Optional[StrictNull]                                  `json:"hideIconOnAbilityPanel,omitzero"`
	Icon                   Optional[string]                                      `json:"icon,omitzero"`
	Id                     Optional[XMLInteger]                                  `json:"id,omitzero"`
	IsAbility              Optional[StrictNull]                                  `json:"isAbility,omitzero"`
	NextImprovement        Optional[string]                                      `json:"nextImprovement,omitzero"`
	Script                 Optional[StrictConsumableScript]                      `json:"script,omitzero"`
	Tags                   Optional[string]                                      `json:"tags,omitzero"`
	UserString             Optional[string]                                      `json:"userString,omitzero"`
	VehicleFilter          Optional[StrictConsumableVehicleFilter]               `json:"vehicleFilter,omitzero"`
}

type StrictConsumableDescrParamEntry struct {
	Name  Optional[string]     `json:"name,omitzero"`
	Value Optional[XMLInteger] `json:"value,omitzero"`
}

type StrictConsumableDisplayParams struct {
	AimingAccuracy    Optional[XMLInteger] `json:"aimingAccuracy,omitzero"`
	AimingTime        Optional[XMLInteger] `json:"aimingTime,omitzero"`
	AvgDamage         Optional[XMLInteger] `json:"avgDamage,omitzero"`
	ClipReloadTime    Optional[XMLInteger] `json:"clipReloadTime,omitzero"`
	Damage            Optional[XMLInteger] `json:"damage,omitzero"`
	DamageFactor      Optional[XMLInteger] `json:"damageFactor,omitzero"`
	DamageFactor2     Optional[XMLInteger] `json:"damageFactor2,omitzero"`
	DamageFactor3     Optional[XMLInteger] `json:"damageFactor3,omitzero"`
	EnginePower       Optional[XMLInteger] `json:"enginePower,omitzero"`
	EnginePowerFactor Optional[XMLInteger] `json:"enginePowerFactor,omitzero"`
	HealthFactor      Optional[XMLInteger] `json:"healthFactor,omitzero"`
	MaxDamageFactor   Optional[XMLInteger] `json:"maxDamageFactor,omitzero"`
	MaxVelocityFactor Optional[XMLInteger] `json:"maxVelocityFactor,omitzero"`
	MinDamageFactor   Optional[XMLInteger] `json:"minDamageFactor,omitzero"`
	MoveSpeed         Optional[XMLInteger] `json:"moveSpeed,omitzero"`
	RotationSpeed     Optional[XMLInteger] `json:"rotationSpeed,omitzero"`
}

type StrictConsumableDynamicParams struct {
	Percents Optional[XMLInteger] `json:"percents,omitzero"`
}

type StrictConsumableGameModeFilter struct {
	Lunar                      Optional[StrictNull] `json:"Lunar,omitzero"`
	Arcade                     Optional[StrictNull] `json:"arcade,omitzero"`
	Battleroyale               Optional[StrictNull] `json:"battleroyale,omitzero"`
	Bossmode                   Optional[StrictNull] `json:"bossmode,omitzero"`
	Gravitizing                Optional[StrictNull] `json:"gravitizing,omitzero"`
	Hellgames                  Optional[StrictNull] `json:"hellgames,omitzero"`
	QuickTournamentArcade      Optional[StrictNull] `json:"quick_tournament_arcade,omitzero"`
	QuickTournamentBossmode    Optional[StrictNull] `json:"quick_tournament_bossmode,omitzero"`
	QuickTournamentGravitizing Optional[StrictNull] `json:"quick_tournament_gravitizing,omitzero"`
	QuickTournamentHellgames   Optional[StrictNull] `json:"quick_tournament_hellgames,omitzero"`
	QuickTournamentLunar       Optional[StrictNull] `json:"quick_tournament_lunar,omitzero"`
	TournamentArcade           Optional[StrictNull] `json:"tournament_arcade,omitzero"`
	TournamentBossmode         Optional[StrictNull] `json:"tournament_bossmode,omitzero"`
	TournamentGravitizing      Optional[StrictNull] `json:"tournament_gravitizing,omitzero"`
	TournamentHellgames        Optional[StrictNull] `json:"tournament_hellgames,omitzero"`
	TournamentLunar            Optional[StrictNull] `json:"tournament_lunar,omitzero"`
}

type StrictConsumableScript struct {
	Text                         Optional[string]                            `json:"#text,omitzero"`
	AbortVelocity                Optional[XMLInteger]                        `json:"abortVelocity,omitzero"`
	Acceleration                 Optional[XMLInteger]                        `json:"acceleration,omitzero"`
	ActivateStates               Optional[XMLInteger]                        `json:"activateStates,omitzero"`
	ActivationDamageFactor       Optional[XMLNumber]                         `json:"activationDamageFactor,omitzero"`
	AimingTimeFactor             Optional[XMLNumber]                         `json:"aimingTimeFactor,omitzero"`
	AllyHealFactorByAllyHP       Optional[XMLInteger]                        `json:"allyHealFactorByAllyHP,omitzero"`
	AttractorMass                Optional[XMLNumber]                         `json:"attractorMass,omitzero"`
	Attributes                   Optional[StrictConsumableScriptAttributes]  `json:"attributes,omitzero"`
	Automatic                    Optional[XMLBool]                           `json:"automatic,omitzero"`
	BkwdSpeedLimitBias           Optional[XMLInteger]                        `json:"bkwdSpeedLimitBias,omitzero"`
	BonusValues                  Optional[StrictConsumableScriptBonusValues] `json:"bonusValues,omitzero"`
	BreakTracks                  Optional[XMLBool]                           `json:"breakTracks,omitzero"`
	BurstDamagePercent           Optional[XMLInteger]                        `json:"burstDamagePercent,omitzero"`
	BurstForce                   Optional[XMLNumber]                         `json:"burstForce,omitzero"`
	BurstRange                   Optional[XMLInteger]                        `json:"burstRange,omitzero"`
	ClipReloadTimeFactor         Optional[XMLNumber]                         `json:"clipReloadTimeFactor,omitzero"`
	ConflictExtras               Optional[string]                            `json:"conflictExtras,omitzero"`
	ContusionChances             Optional[XMLNumberVector]                   `json:"contusionChances,omitzero"`
	Cooldown                     Optional[XMLInteger]                        `json:"cooldown,omitzero"`
	CooldownReducePercent        Optional[XMLInteger]                        `json:"cooldownReducePercent,omitzero"`
	DamageFactor                 Optional[XMLNumber]                         `json:"damageFactor,omitzero"`
	DamageFactors                Optional[XMLNumberVector]                   `json:"damageFactors,omitzero"`
	DeactivateStates             Optional[XMLInteger]                        `json:"deactivateStates,omitzero"`
	DisableTracksCollisionDamage Optional[XMLBool]                           `json:"disableTracksCollisionDamage,omitzero"`
	Distance                     Optional[XMLInteger]                        `json:"distance,omitzero"`
	DistanceOffset               Optional[XMLInteger]                        `json:"distanceOffset,omitzero"`
	Duration                     Optional[XMLNumber]                         `json:"duration,omitzero"`
	EffectDuration               Optional[XMLInteger]                        `json:"effectDuration,omitzero"`
	EffectExtraName              Optional[string]                            `json:"effectExtraName,omitzero"`
	EnginePowerIncrease          Optional[XMLNumber]                         `json:"enginePowerIncrease,omitzero"`
	ExtrasMustBeStopped          Optional[string]                            `json:"extrasMustBeStopped,omitzero"`
	FlyTime                      Optional[XMLNumber]                         `json:"flyTime,omitzero"`
	FwdSpeedLimitBias            Optional[XMLInteger]                        `json:"fwdSpeedLimitBias,omitzero"`
	GunReloadTimeFactor          Optional[XMLInteger]                        `json:"gunReloadTimeFactor,omitzero"`
	HealFactorPercent            Optional[XMLInteger]                        `json:"healFactorPercent,omitzero"`
	HealWaveDelay                Optional[XMLNumber]                         `json:"healWaveDelay,omitzero"`
	HealWavesCount               Optional[XMLInteger]                        `json:"healWavesCount,omitzero"`
	HealthFactor                 Optional[XMLNumber]                         `json:"healthFactor,omitzero"`
	InnerCooldown                Optional[XMLInteger]                        `json:"innerCooldown,omitzero"`
	KeepSpoted                   Optional[XMLBool]                           `json:"keepSpoted,omitzero"`
	MaxAngleDegrees              Optional[XMLInteger]                        `json:"maxAngleDegrees,omitzero"`
	MaxDamageFactor              Optional[XMLNumber]                         `json:"maxDamageFactor,omitzero"`
	MaxHealthDamageFactor        Optional[XMLNumber]                         `json:"maxHealthDamageFactor,omitzero"`
	MaxHealthDamagePercent       Optional[XMLInteger]                        `json:"maxHealthDamagePercent,omitzero"`
	MaxHealthSelfDamageFactor    Optional[XMLInteger]                        `json:"maxHealthSelfDamageFactor,omitzero"`
	MinAngleDegrees              Optional[XMLInteger]                        `json:"minAngleDegrees,omitzero"`
	MinDamageFactor              Optional[XMLNumber]                         `json:"minDamageFactor,omitzero"`
	Multipliers                  Optional[StrictConsumableScriptMultipliers] `json:"multipliers,omitzero"`
	Overrides                    Optional[StrictConsumableScriptOverrides]   `json:"overrides,omitzero"`
	PoisonTickDelay              Optional[XMLInteger]                        `json:"poisonTickDelay,omitzero"`
	PoisonTicks                  Optional[XMLInteger]                        `json:"poisonTicks,omitzero"`
	ProjectileSpeed              Optional[XMLInteger]                        `json:"projectileSpeed,omitzero"`
	QuantityLimit                Optional[XMLInteger]                        `json:"quantityLimit,omitzero"`
	QuantityPerUse               Optional[XMLInteger]                        `json:"quantityPerUse,omitzero"`
	Radius                       Optional[XMLInteger]                        `json:"radius,omitzero"`
	Range                        Optional[XMLNumber]                         `json:"range,omitzero"`
	RotationSpeedFactor          Optional[XMLNumber]                         `json:"rotationSpeedFactor,omitzero"`
	SelfHealReducePercent        Optional[XMLNumber]                         `json:"selfHealReducePercent,omitzero"`
	SelfInstantRestorePercent    Optional[XMLInteger]                        `json:"selfInstantRestorePercent,omitzero"`
	ShieldCoef                   Optional[XMLNumber]                         `json:"shieldCoef,omitzero"`
	ShieldDuration               Optional[XMLInteger]                        `json:"shieldDuration,omitzero"`
	ShotDispersionAngleFactor    Optional[XMLNumber]                         `json:"shotDispersionAngleFactor,omitzero"`
	ShotEffect                   Optional[string]                            `json:"shotEffect,omitzero"`
	StartCooldown                Optional[XMLInteger]                        `json:"startCooldown,omitzero"`
	Statics                      Optional[StrictConsumableScriptStatics]     `json:"statics,omitzero"`
	TargetExtrasMustBeStopped    Optional[string]                            `json:"targetExtrasMustBeStopped,omitzero"`
	TimeSeconds                  Optional[XMLNumber]                         `json:"timeSeconds,omitzero"`
}

type StrictConsumableScriptAttributes struct {
	ClipReloadFactor          Optional[XMLNumber] `json:"clipReloadFactor,omitzero"`
	CommonDamageFactor        Optional[XMLNumber] `json:"commonDamageFactor,omitzero"`
	GunReloadFactor           Optional[XMLNumber] `json:"gunReloadFactor,omitzero"`
	PumpGunReloadFactor       Optional[XMLNumber] `json:"pumpGunReloadFactor,omitzero"`
	ReceiveDamageFactor       Optional[XMLNumber] `json:"receiveDamageFactor,omitzero"`
	ShotDispersionAngleFactor Optional[XMLNumber] `json:"shotDispersionAngleFactor,omitzero"`
}

type StrictConsumableScriptBonusValues struct {
	ChassisHealthFactor      Optional[XMLInteger] `json:"chassisHealthFactor,omitzero"`
	DamagePercentBonus       Optional[XMLInteger] `json:"damagePercentBonus,omitzero"`
	DamageReflectionPercent  Optional[XMLInteger] `json:"damageReflectionPercent,omitzero"`
	EnginePowerAbilityFactor Optional[XMLNumber]  `json:"enginePowerAbilityFactor,omitzero"`
	EnginePowerFactor        Optional[XMLInteger] `json:"enginePowerFactor,omitzero"`
	EnginePowerIncrease      Optional[XMLInteger] `json:"enginePowerIncrease,omitzero"`
	GunReloadSpeedIncrease   Optional[XMLInteger] `json:"gunReloadSpeedIncrease,omitzero"`
	MaxVelocityFactor        Optional[XMLNumber]  `json:"maxVelocityFactor,omitzero"`
	RammingDamageFactor      Optional[XMLNumber]  `json:"rammingDamageFactor,omitzero"`
	RotationSpeedFactor      Optional[XMLNumber]  `json:"rotationSpeedFactor,omitzero"`
	SacrificeLoss            Optional[XMLInteger] `json:"sacrificeLoss,omitzero"`
	SpeedLimitAbilityBonus   Optional[XMLInteger] `json:"speedLimitAbilityBonus,omitzero"`
	TerrainResistanceFactor  Optional[XMLNumber]  `json:"terrainResistanceFactor,omitzero"`
}

type StrictConsumableScriptMultipliers struct {
	AngularSpeedMultiplier          Optional[XMLNumber]  `json:"angularSpeedMultiplier,omitzero"`
	BackwardSpeedMultiplier         Optional[XMLInteger] `json:"backwardSpeedMultiplier,omitzero"`
	EnginePowerMultiplier           Optional[XMLNumber]  `json:"enginePowerMultiplier,omitzero"`
	ForwardSpeedMultiplier          Optional[XMLNumber]  `json:"forwardSpeedMultiplier,omitzero"`
	IncomingRammingDamageMultiplier Optional[XMLNumber]  `json:"incomingRammingDamageMultiplier,omitzero"`
	OutgoingRammingDamageMultiplier Optional[XMLNumber]  `json:"outgoingRammingDamageMultiplier,omitzero"`
}

type StrictConsumableScriptOverrides struct {
	Override Optional[[]Optional[StrictConsumableScriptOverridesOverrideEntry]] `json:"override,omitzero"`
}

type StrictConsumableScriptOverridesOverrideEntry struct {
	AttractorMass      Optional[XMLNumber]  `json:"attractorMass,omitzero"`
	BurstDamagePercent Optional[XMLInteger] `json:"burstDamagePercent,omitzero"`
	BurstForce         Optional[XMLInteger] `json:"burstForce,omitzero"`
	BurstRange         Optional[XMLInteger] `json:"burstRange,omitzero"`
	Cooldown           Optional[XMLInteger] `json:"cooldown,omitzero"`
	EffectDuration     Optional[XMLInteger] `json:"effectDuration,omitzero"`
	MaxVehLevel        Optional[XMLInteger] `json:"maxVehLevel,omitzero"`
	MinVehLevel        Optional[XMLInteger] `json:"minVehLevel,omitzero"`
	Radius             Optional[XMLInteger] `json:"radius,omitzero"`
}

type StrictConsumableScriptStatics struct {
	FlatteningCoefficient                  Optional[XMLInteger] `json:"flatteningCoefficient,omitzero"`
	IncomingDefaultRammingDamageMultiplier Optional[XMLNumber]  `json:"incomingDefaultRammingDamageMultiplier,omitzero"`
	OutgoingDefaultRammingDamageMultiplier Optional[XMLNumber]  `json:"outgoingDefaultRammingDamageMultiplier,omitzero"`
	StaticForce                            Optional[XMLInteger] `json:"staticForce,omitzero"`
	StaticRestForce                        Optional[XMLInteger] `json:"staticRestForce,omitzero"`
	WideningCoefficient                    Optional[XMLInteger] `json:"wideningCoefficient,omitzero"`
}

type StrictConsumableVehicleFilter struct {
	Exclude Optional[StrictConsumableVehicleFilterExclude] `json:"exclude,omitzero"`
	Include Optional[StrictConsumableVehicleFilterInclude] `json:"include,omitzero"`
}

type StrictConsumableVehicleFilterExclude struct {
	Vehicle Optional[StrictConsumableVehicleFilterExcludeVehicle] `json:"vehicle,omitzero"`
}

type StrictConsumableVehicleFilterExcludeVehicle struct {
	ExtendedTags Optional[string] `json:"extendedTags,omitzero"`
	Name         Optional[string] `json:"name,omitzero"`
}

type StrictConsumableVehicleFilterInclude struct {
	Vehicle Optional[StrictConsumableVehicleFilterIncludeVehicle] `json:"vehicle,omitzero"`
}

type StrictConsumableVehicleFilterIncludeVehicle struct {
	MaxLevel Optional[XMLInteger] `json:"maxLevel,omitzero"`
	MinLevel Optional[XMLInteger] `json:"minLevel,omitzero"`
	Name     Optional[string]     `json:"name,omitzero"`
}
