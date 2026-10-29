package export

// StrictSettings describes the 23 observed settings documents. New document names
// are rejected by the strict data reader.
type StrictSettings struct {
	Vehicle                 Optional[StrictVehicleSettings]                    `json:"vehicle,omitzero"`
	Camouflages             Optional[SettingCamouflages]                       `json:"camouflages,omitzero"`
	PlayerEmblems           Optional[SettingPlayerEmblems]                     `json:"player_emblems,omitzero"`
	Roles                   Optional[SettingRoles]                             `json:"roles,omitzero"`
	ChassisEffects          Optional[StrictChassisEffects]                     `json:"chassis_effects,omitzero"`
	GunEffects              Optional[map[string]Optional[StrictGunEffect]]     `json:"gun_effects,omitzero"`
	ShotEffects             Optional[map[string]Optional[StrictShotEffect]]    `json:"shot_effects,omitzero"`
	VehicleEffects          Optional[map[string]Optional[StrictVehicleEffect]] `json:"vehicle_effects,omitzero"`
	CombatRoles             Optional[map[string]Optional[SettingCombatRole]]   `json:"combat_roles,omitzero"`
	ConsumablesBattleroyale Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/battleroyale,omitzero"`
	ConsumablesBossmode     Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/bossmode,omitzero"`
	ConsumablesCommon       Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/common,omitzero"`
	ConsumablesGravitizing  Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/gravitizing,omitzero"`
	ConsumablesHellgames    Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/hellgames,omitzero"`
	ConsumablesLunar        Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/lunar,omitzero"`
	ConsumablesMadgames     Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/madgames,omitzero"`
	ConsumablesPrototypes   Optional[map[string]Optional[StrictConsumable]]    `json:"consumables/prototypes,omitzero"`
	ConsumablesList         Optional[SettingConsumablesList]                   `json:"consumables/list,omitzero"`
	ConsumablesDefault      Optional[map[string]Optional[[]Optional[string]]]  `json:"consumablesDefault,omitzero"`
	HitMarks                Optional[map[string]Optional[SettingHitMark]]      `json:"hit_marks,omitzero"`
	Horns                   Optional[map[string]Optional[SettingHorn]]         `json:"horns,omitzero"`
	Nations                 Optional[map[string]Optional[SettingNation]]       `json:"nations,omitzero"`
	StuffPackages           Optional[map[string]Optional[SettingStuffPackage]] `json:"stuff_packages,omitzero"`
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
