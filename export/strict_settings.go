package export

// StrictSettings describes the 23 observed settings documents. New document names
// are rejected by the strict data reader.
type StrictSettings struct {
	Vehicle                 Optional[StrictVehicleSettings]                    `json:"vehicle,omitempty"`
	Camouflages             Optional[SettingCamouflages]                       `json:"camouflages,omitempty"`
	PlayerEmblems           Optional[SettingPlayerEmblems]                     `json:"player_emblems,omitempty"`
	Roles                   Optional[SettingRoles]                             `json:"roles,omitempty"`
	ChassisEffects          Optional[StrictChassisEffects]                     `json:"chassis_effects,omitempty"`
	GunEffects              Optional[map[string]Optional[StrictGunEffect]]     `json:"gun_effects,omitempty"`
	ShotEffects             Optional[map[string]Optional[StrictShotEffect]]    `json:"shot_effects,omitempty"`
	VehicleEffects          Optional[map[string]Optional[StrictVehicleEffect]] `json:"vehicle_effects,omitempty"`
	CombatRoles             Optional[map[string]Optional[SettingCombatRole]]   `json:"combat_roles,omitempty"`
	ConsumablesBattleroyale Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/battleroyale,omitempty"`
	ConsumablesBossmode     Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/bossmode,omitempty"`
	ConsumablesCommon       Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/common,omitempty"`
	ConsumablesGravitizing  Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/gravitizing,omitempty"`
	ConsumablesHellgames    Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/hellgames,omitempty"`
	ConsumablesLunar        Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/lunar,omitempty"`
	ConsumablesMadgames     Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/madgames,omitempty"`
	ConsumablesPrototypes   Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/prototypes,omitempty"`
	ConsumablesList         Optional[SettingConsumablesList]                   `json:"consumables/list,omitempty"`
	ConsumablesDefault      Optional[map[string]Optional[[]Optional[string]]]  `json:"consumablesDefault,omitempty"`
	HitMarks                Optional[map[string]Optional[SettingHitMark]]      `json:"hit_marks,omitempty"`
	Horns                   Optional[map[string]Optional[SettingHorn]]         `json:"horns,omitempty"`
	Nations                 Optional[map[string]Optional[SettingNation]]       `json:"nations,omitempty"`
	StuffPackages           Optional[map[string]Optional[SettingStuffPackage]] `json:"stuff_packages,omitempty"`
}

func strictSettingsFromPartial(v PartialSettings) StrictSettings {
	return StrictSettings{
		Vehicle:                 v.Vehicle,
		Camouflages:             v.Camouflages,
		PlayerEmblems:           v.PlayerEmblems,
		Roles:                   v.Roles,
		ChassisEffects:          v.ChassisEffects,
		GunEffects:              v.GunEffects,
		ShotEffects:             v.ShotEffects,
		VehicleEffects:          v.VehicleEffects,
		CombatRoles:             v.CombatRoles,
		ConsumablesBattleroyale: v.ConsumablesBattleroyale,
		ConsumablesBossmode:     v.ConsumablesBossmode,
		ConsumablesCommon:       v.ConsumablesCommon,
		ConsumablesGravitizing:  v.ConsumablesGravitizing,
		ConsumablesHellgames:    v.ConsumablesHellgames,
		ConsumablesLunar:        v.ConsumablesLunar,
		ConsumablesMadgames:     v.ConsumablesMadgames,
		ConsumablesPrototypes:   v.ConsumablesPrototypes,
		ConsumablesList:         v.ConsumablesList,
		ConsumablesDefault:      v.ConsumablesDefault,
		HitMarks:                v.HitMarks,
		Horns:                   v.Horns,
		Nations:                 v.Nations,
		StuffPackages:           v.StuffPackages,
	}
}
