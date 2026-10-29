package export

import (
	"fmt"
)

// StrictVehicleSettings models the observed settings.vehicle XML document.
// The extra and material names are dynamic keys; XML scalars remain strings.
type StrictVehicleSettings struct {
	Balance               Optional[StrictVehicleSettingsBalance]               `json:"balance,omitzero"`
	DefaultVehicleEffects Optional[StrictVehicleDefaultEffects]                `json:"defaultVehicleEffects,omitzero"`
	DeviceExtras          Optional[StrictVehicleDeviceExtras]                  `json:"deviceExtras,omitzero"`
	Extras                Optional[map[string]Optional[StrictVehicleExtra]]    `json:"extras,omitzero"`
	LodLevels             Optional[StrictVehicleLodLevels]                     `json:"lodLevels,omitzero"`
	Materials             Optional[map[string]Optional[StrictVehicleMaterial]] `json:"materials,omitzero"`
	MiscParams            Optional[StrictVehicleMiscParams]                    `json:"miscParams,omitzero"`
}

type StrictVehicleSettingsBalance struct {
	ByComponentLevels        Optional[XMLNumberVector]                `json:"byComponentLevels,omitzero"`
	ByVehicleClasses         Optional[map[string]Optional[XMLNumber]] `json:"byVehicleClasses,omitzero"`
	ByVehicleModule          Optional[string]                         `json:"byVehicleModule,omitzero"`
	ModulesWeightMultipliers Optional[StrictVehicleModuleWeights]     `json:"modulesWeightMultipliers,omitzero"`
}
type StrictVehicleModuleWeights struct {
	Chassis Optional[StrictVehicleModuleWeightChassis] `json:"chassis,omitzero"`
	Engine  Optional[XMLNumber]                        `json:"engine,omitzero"`
	Gun     Optional[XMLNumber]                        `json:"gun,omitzero"`
	Turret  Optional[XMLNumber]                        `json:"turret,omitzero"`
	Vehicle Optional[XMLNumber]                        `json:"vehicle,omitzero"`
}
type StrictVehicleModuleWeightChassis struct {
	Items Optional[StrictVehicleEmptyItems] `json:"#items,omitzero"`
	Text  Optional[XMLNumber]               `json:"#text,omitzero"`
}

// StrictVehicleEmptyItems describes the empty #items array observed here;
// nonempty arrays are rejected until their actual entry shapes are known.
type StrictVehicleEmptyItems []struct{}

func (v StrictVehicleEmptyItems) MarshalJSON() ([]byte, error) {
	if v == nil || len(v) != 0 {
		return nil, fmt.Errorf("expected empty #items array")
	}
	return []byte("[]"), nil
}

type StrictVehicleDefaultEffects struct {
	CollisionVehicle     Optional[string] `json:"collisionVehicle,omitzero"`
	Destruction          Optional[string] `json:"destruction,omitzero"`
	Explosion            Optional[string] `json:"explosion,omitzero"`
	Flaming              Optional[string] `json:"flaming,omitzero"`
	GroundCollisionHeavy Optional[string] `json:"groundCollisionHeavy,omitzero"`
	GroundCollisionLight Optional[string] `json:"groundCollisionLight,omitzero"`
	MetalCollisionHeavy  Optional[string] `json:"metalCollisionHeavy,omitzero"`
	MetalCollisionLight  Optional[string] `json:"metalCollisionLight,omitzero"`
	SandCollisionHeavy   Optional[string] `json:"sandCollisionHeavy,omitzero"`
	SandCollisionLight   Optional[string] `json:"sandCollisionLight,omitzero"`
	SnowCollisionHeavy   Optional[string] `json:"snowCollisionHeavy,omitzero"`
	SnowCollisionLight   Optional[string] `json:"snowCollisionLight,omitzero"`
	StoneCollisionHeavy  Optional[string] `json:"stoneCollisionHeavy,omitzero"`
	StoneCollisionLight  Optional[string] `json:"stoneCollisionLight,omitzero"`
	WaterCollisionHeavy  Optional[string] `json:"waterCollisionHeavy,omitzero"`
	WaterCollisionLight  Optional[string] `json:"waterCollisionLight,omitzero"`
	WoodCollisionHeavy   Optional[string] `json:"woodCollisionHeavy,omitzero"`
	WoodCollisionLight   Optional[string] `json:"woodCollisionLight,omitzero"`
}
type StrictVehicleDeviceExtras struct {
	Devices Optional[map[string]Optional[string]] `json:"devices,omitzero"`
	Tankmen Optional[map[string]Optional[string]] `json:"tankmen,omitzero"`
}
type StrictVehicleExtra struct {
	Text             Optional[string]                      `json:"#text,omitzero"`
	Index            Optional[XMLInteger]                  `json:"@index,omitzero"`
	Delay            Optional[XMLNumber]                   `json:"delay,omitzero"`
	DeviceUserString Optional[XMLLocalizationKey]          `json:"deviceUserString,omitzero"`
	IndexInRole      Optional[XMLInteger]                  `json:"indexInRole,omitzero"`
	IsLeft           Optional[XMLBool]                     `json:"isLeft,omitzero"`
	Sounds           Optional[map[string]Optional[string]] `json:"sounds,omitzero"`
	TankmanRole      Optional[string]                      `json:"tankmanRole,omitzero"`
}
type StrictVehicleLodLevels struct {
	Far    Optional[XMLInteger] `json:"FAR,omitzero"`
	Medium Optional[XMLInteger] `json:"MEDIUM,omitzero"`
	Near   Optional[XMLInteger] `json:"NEAR,omitzero"`
}
type StrictVehicleMaterial struct {
	Armor                        Optional[XMLInteger]                          `json:"armor,omitzero"`
	ChanceToHitByExplosion       Optional[XMLScalarList[XMLNumber, XMLNumber]] `json:"chanceToHitByExplosion,omitzero"`
	ChanceToHitByProjectile      Optional[XMLNumber]                           `json:"chanceToHitByProjectile,omitzero"`
	CollideOnceOnly              Optional[XMLBool]                             `json:"collideOnceOnly,omitzero"`
	ContinueTraceIfNoHit         Optional[XMLBool]                             `json:"continueTraceIfNoHit,omitzero"`
	DamageKind                   Optional[string]                              `json:"damageKind,omitzero"`
	Extra                        Optional[string]                              `json:"extra,omitzero"`
	Incombustible                Optional[string]                              `json:"incombustible,omitzero"`
	IncreasedVulnerabilityFactor Optional[XMLNumber]                           `json:"increasedVulnerabilityFactor,omitzero"`
	MayRicochet                  Optional[XMLBool]                             `json:"mayRicochet,omitzero"`
	UseAntifragmentationLining   Optional[XMLBool]                             `json:"useAntifragmentationLining,omitzero"`
	UseArmorHomogenization       Optional[XMLBool]                             `json:"useArmorHomogenization,omitzero"`
	UseHitAngle                  Optional[XMLBool]                             `json:"useHitAngle,omitzero"`
	VehicleDamageFactor          Optional[XMLNumber]                           `json:"vehicleDamageFactor,omitzero"`
}
type StrictVehicleMiscParams struct {
	CollisionEffectVelocities Optional[StrictVehicleCollisionEffectVelocities] `json:"collisionEffectVelocities,omitzero"`
	HighExplosiveLargeCaliber Optional[XMLNumber]                              `json:"highExplosiveLargeCaliber,omitzero"`
	HornCooldown              Optional[StrictVehicleHornCooldown]              `json:"hornCooldown,omitzero"`
	MinFireStartingDamage     Optional[XMLInteger]                             `json:"minFireStartingDamage,omitzero"`
	ProjectileSpeedFactor     Optional[XMLNumber]                              `json:"projectileSpeedFactor,omitzero"`
}
type StrictVehicleCollisionEffectVelocities struct {
	Hull         Optional[XMLNumberVector] `json:"hull,omitzero"`
	Ramming      Optional[XMLNumber]       `json:"ramming,omitzero"`
	Track        Optional[XMLNumberVector] `json:"track,omitzero"`
	WaterContact Optional[XMLNumberVector] `json:"waterContact,omitzero"`
}
type StrictVehicleHornCooldown struct {
	ClientWindowExpansion Optional[XMLInteger] `json:"clientWindowExpansion,omitzero"`
	MaxSignals            Optional[XMLInteger] `json:"maxSignals,omitzero"`
	Window                Optional[XMLInteger] `json:"window,omitzero"`
}
