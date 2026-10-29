package export

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StrictVehicleSettings models the observed settings.vehicle XML document.
// The extra and material names are dynamic keys; XML scalars remain strings.
type StrictVehicleSettings struct {
	Balance               Optional[StrictVehicleSettingsBalance]               `json:"balance,omitempty"`
	DefaultVehicleEffects Optional[StrictVehicleDefaultEffects]                `json:"defaultVehicleEffects,omitempty"`
	DeviceExtras          Optional[StrictVehicleDeviceExtras]                  `json:"deviceExtras,omitempty"`
	Extras                Optional[map[string]Optional[StrictVehicleExtra]]    `json:"extras,omitempty"`
	LodLevels             Optional[StrictVehicleLodLevels]                     `json:"lodLevels,omitempty"`
	Materials             Optional[map[string]Optional[StrictVehicleMaterial]] `json:"materials,omitempty"`
	MiscParams            Optional[StrictVehicleMiscParams]                    `json:"miscParams,omitempty"`
}

func (v *StrictVehicleSettings) UnmarshalJSON(data []byte) error {
	b := bytes.TrimSpace(data)
	if len(b) == 0 || b[0] != '{' {
		return fmt.Errorf("vehicle settings must be an object")
	}
	type alias StrictVehicleSettings
	var x alias
	if err := decodeStrictJSON(data, &x); err != nil {
		return err
	}
	*v = StrictVehicleSettings(x)
	return nil
}

type StrictVehicleSettingsBalance struct {
	ByComponentLevels        Optional[string]                      `json:"byComponentLevels,omitempty"`
	ByVehicleClasses         Optional[map[string]Optional[string]] `json:"byVehicleClasses,omitempty"`
	ByVehicleModule          Optional[string]                      `json:"byVehicleModule,omitempty"`
	ModulesWeightMultipliers Optional[StrictVehicleModuleWeights]  `json:"modulesWeightMultipliers,omitempty"`
}
type StrictVehicleModuleWeights struct {
	Chassis Optional[StrictVehicleModuleWeightChassis] `json:"chassis,omitempty"`
	Engine  Optional[string]                           `json:"engine,omitempty"`
	Gun     Optional[string]                           `json:"gun,omitempty"`
	Turret  Optional[string]                           `json:"turret,omitempty"`
	Vehicle Optional[string]                           `json:"vehicle,omitempty"`
}
type StrictVehicleModuleWeightChassis struct {
	Items Optional[StrictVehicleEmptyItems] `json:"#items,omitempty"`
	Text  Optional[string]                  `json:"#text,omitempty"`
}

// StrictVehicleEmptyItems describes the empty #items array observed here;
// nonempty arrays are rejected until their actual entry shapes are known.
type StrictVehicleEmptyItems []struct{}

func (v *StrictVehicleEmptyItems) UnmarshalJSON(data []byte) error {
	var entries []json.RawMessage
	if err := decodeStrictJSON(data, &entries); err != nil {
		return err
	}
	if entries == nil || len(entries) != 0 {
		return fmt.Errorf("expected empty #items array")
	}
	*v = StrictVehicleEmptyItems{}
	return nil
}
func (v StrictVehicleEmptyItems) MarshalJSON() ([]byte, error) {
	if v == nil || len(v) != 0 {
		return nil, fmt.Errorf("expected empty #items array")
	}
	return []byte("[]"), nil
}

type StrictVehicleDefaultEffects struct {
	CollisionVehicle     Optional[string] `json:"collisionVehicle,omitempty"`
	Destruction          Optional[string] `json:"destruction,omitempty"`
	Explosion            Optional[string] `json:"explosion,omitempty"`
	Flaming              Optional[string] `json:"flaming,omitempty"`
	GroundCollisionHeavy Optional[string] `json:"groundCollisionHeavy,omitempty"`
	GroundCollisionLight Optional[string] `json:"groundCollisionLight,omitempty"`
	MetalCollisionHeavy  Optional[string] `json:"metalCollisionHeavy,omitempty"`
	MetalCollisionLight  Optional[string] `json:"metalCollisionLight,omitempty"`
	SandCollisionHeavy   Optional[string] `json:"sandCollisionHeavy,omitempty"`
	SandCollisionLight   Optional[string] `json:"sandCollisionLight,omitempty"`
	SnowCollisionHeavy   Optional[string] `json:"snowCollisionHeavy,omitempty"`
	SnowCollisionLight   Optional[string] `json:"snowCollisionLight,omitempty"`
	StoneCollisionHeavy  Optional[string] `json:"stoneCollisionHeavy,omitempty"`
	StoneCollisionLight  Optional[string] `json:"stoneCollisionLight,omitempty"`
	WaterCollisionHeavy  Optional[string] `json:"waterCollisionHeavy,omitempty"`
	WaterCollisionLight  Optional[string] `json:"waterCollisionLight,omitempty"`
	WoodCollisionHeavy   Optional[string] `json:"woodCollisionHeavy,omitempty"`
	WoodCollisionLight   Optional[string] `json:"woodCollisionLight,omitempty"`
}
type StrictVehicleDeviceExtras struct {
	Devices Optional[map[string]Optional[string]] `json:"devices,omitempty"`
	Tankmen Optional[map[string]Optional[string]] `json:"tankmen,omitempty"`
}
type StrictVehicleExtra struct {
	Text             Optional[string]                      `json:"#text,omitempty"`
	Index            Optional[string]                      `json:"@index,omitempty"`
	Delay            Optional[string]                      `json:"delay,omitempty"`
	DeviceUserString Optional[string]                      `json:"deviceUserString,omitempty"`
	IndexInRole      Optional[string]                      `json:"indexInRole,omitempty"`
	IsLeft           Optional[string]                      `json:"isLeft,omitempty"`
	Sounds           Optional[map[string]Optional[string]] `json:"sounds,omitempty"`
	TankmanRole      Optional[string]                      `json:"tankmanRole,omitempty"`
}
type StrictVehicleLodLevels struct {
	Far    Optional[string] `json:"FAR,omitempty"`
	Medium Optional[string] `json:"MEDIUM,omitempty"`
	Near   Optional[string] `json:"NEAR,omitempty"`
}
type StrictVehicleMaterial struct {
	Armor                        Optional[string]                `json:"armor,omitempty"`
	ChanceToHitByExplosion       Optional[XMLScalarList[string]] `json:"chanceToHitByExplosion,omitempty"`
	ChanceToHitByProjectile      Optional[string]                `json:"chanceToHitByProjectile,omitempty"`
	CollideOnceOnly              Optional[string]                `json:"collideOnceOnly,omitempty"`
	ContinueTraceIfNoHit         Optional[string]                `json:"continueTraceIfNoHit,omitempty"`
	DamageKind                   Optional[string]                `json:"damageKind,omitempty"`
	Extra                        Optional[string]                `json:"extra,omitempty"`
	Incombustible                Optional[string]                `json:"incombustible,omitempty"`
	IncreasedVulnerabilityFactor Optional[string]                `json:"increasedVulnerabilityFactor,omitempty"`
	MayRicochet                  Optional[string]                `json:"mayRicochet,omitempty"`
	UseAntifragmentationLining   Optional[string]                `json:"useAntifragmentationLining,omitempty"`
	UseArmorHomogenization       Optional[string]                `json:"useArmorHomogenization,omitempty"`
	UseHitAngle                  Optional[string]                `json:"useHitAngle,omitempty"`
	VehicleDamageFactor          Optional[string]                `json:"vehicleDamageFactor,omitempty"`
}
type StrictVehicleMiscParams struct {
	CollisionEffectVelocities Optional[StrictVehicleCollisionEffectVelocities] `json:"collisionEffectVelocities,omitempty"`
	HighExplosiveLargeCaliber Optional[string]                                 `json:"highExplosiveLargeCaliber,omitempty"`
	HornCooldown              Optional[StrictVehicleHornCooldown]              `json:"hornCooldown,omitempty"`
	MinFireStartingDamage     Optional[string]                                 `json:"minFireStartingDamage,omitempty"`
	ProjectileSpeedFactor     Optional[string]                                 `json:"projectileSpeedFactor,omitempty"`
}
type StrictVehicleCollisionEffectVelocities struct {
	Hull         Optional[string] `json:"hull,omitempty"`
	Ramming      Optional[string] `json:"ramming,omitempty"`
	Track        Optional[string] `json:"track,omitempty"`
	WaterContact Optional[string] `json:"waterContact,omitempty"`
}
type StrictVehicleHornCooldown struct {
	ClientWindowExpansion Optional[string] `json:"clientWindowExpansion,omitempty"`
	MaxSignals            Optional[string] `json:"maxSignals,omitempty"`
	Window                Optional[string] `json:"window,omitempty"`
}
