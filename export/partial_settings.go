package export

import (
	"encoding/json"
	"fmt"
	"sort"
)

// PartialSettings types the listed settings documents and keeps all remaining
// documents as raw JSON. Other keys are source document names, not validated
// fields. UnvalidatedDocuments lists them deterministically.
type PartialSettings struct {
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
	Other                   map[string]json.RawMessage                         `json:"-"`
}

type SettingCombatRole struct {
	DefaultAbilities Optional[[]Optional[string]] `json:"default_abilities,omitzero"`
	ID               Optional[StrictNumber]       `json:"id,omitzero"`
}
type SettingConsumablesList struct {
	Items           Optional[[]Optional[SettingInclude]] `json:"items,omitzero"`
	NextAvailableID Optional[XMLInteger]                 `json:"nextAvailableId,omitzero"`
}
type SettingInclude struct {
	Path Optional[string] `json:"@path,omitzero"`
}
type SettingHitMark struct {
	ModelSizes  Optional[XMLNumberVector] `json:"modelSizes,omitzero"`
	Priority    Optional[XMLInteger]      `json:"priority,omitzero"`
	TextureName Optional[string]          `json:"texName,omitzero"`
}
type SettingHorn struct {
	Distance    Optional[XMLNumber]          `json:"distance,omitzero"`
	Gold        Optional[XMLInteger]         `json:"gold,omitzero"`
	ID          Optional[XMLInteger]         `json:"id,omitzero"`
	MaxDuration Optional[XMLInteger]         `json:"maxDuration,omitzero"`
	Mode        Optional[string]             `json:"mode,omitzero"`
	Sound       Optional[string]             `json:"sound,omitzero"`
	UserString  Optional[XMLLocalizationKey] `json:"userString,omitzero"`
	VehicleTags Optional[string]             `json:"vehicleTags,omitzero"`
}
type SettingNation struct {
	ID Optional[XMLInteger] `json:"id,omitzero"`
}
type SettingStuffPackage struct {
	Content     Optional[string]     `json:"content,omitzero"`
	Description Optional[string]     `json:"description,omitzero"`
	Effect      Optional[string]     `json:"effect,omitzero"`
	ID          Optional[XMLInteger] `json:"id,omitzero"`
	Image       Optional[string]     `json:"image,omitzero"`
	Title       Optional[string]     `json:"title,omitzero"`
}

func (s PartialSettings) MarshalJSON() ([]byte, error) {
	entries := make(map[string]json.RawMessage, len(s.Other)+7)
	for k, v := range s.Other {
		entries[k] = v
	}
	known := []struct {
		name    string
		value   any
		present bool
	}{
		{"vehicle", s.Vehicle, s.Vehicle.Present()},
		{"combat_roles", s.CombatRoles, s.CombatRoles.Present()},
		{"camouflages", s.Camouflages, s.Camouflages.Present()},
		{"player_emblems", s.PlayerEmblems, s.PlayerEmblems.Present()},
		{"roles", s.Roles, s.Roles.Present()},

		{"chassis_effects", s.ChassisEffects, s.ChassisEffects.Present()},
		{"gun_effects", s.GunEffects, s.GunEffects.Present()},
		{"shot_effects", s.ShotEffects, s.ShotEffects.Present()},
		{"vehicle_effects", s.VehicleEffects, s.VehicleEffects.Present()},

		{"consumables/battleroyale", s.ConsumablesBattleroyale, s.ConsumablesBattleroyale.Present()},
		{"consumables/bossmode", s.ConsumablesBossmode, s.ConsumablesBossmode.Present()},
		{"consumables/common", s.ConsumablesCommon, s.ConsumablesCommon.Present()},
		{"consumables/gravitizing", s.ConsumablesGravitizing, s.ConsumablesGravitizing.Present()},
		{"consumables/hellgames", s.ConsumablesHellgames, s.ConsumablesHellgames.Present()},
		{"consumables/lunar", s.ConsumablesLunar, s.ConsumablesLunar.Present()},
		{"consumables/madgames", s.ConsumablesMadgames, s.ConsumablesMadgames.Present()},
		{"consumables/prototypes", s.ConsumablesPrototypes, s.ConsumablesPrototypes.Present()},

		{"consumables/list", s.ConsumablesList, s.ConsumablesList.Present()},
		{"consumablesDefault", s.ConsumablesDefault, s.ConsumablesDefault.Present()},
		{"hit_marks", s.HitMarks, s.HitMarks.Present()},
		{"horns", s.Horns, s.Horns.Present()},
		{"nations", s.Nations, s.Nations.Present()},
		{"stuff_packages", s.StuffPackages, s.StuffPackages.Present()},
	}
	for _, item := range known {
		if _, exists := entries[item.name]; exists {
			return nil, fmt.Errorf("settings document %q is in Other but has a typed field", item.name)
		}
		if item.present {
			raw, err := json.Marshal(item.value)
			if err != nil {
				return nil, err
			}
			entries[item.name] = raw
		}
	}
	return json.Marshal(entries)
}

// UnvalidatedDocuments returns names whose values remain raw JSON.
func (s PartialSettings) UnvalidatedDocuments() []string {
	names := make([]string, 0, len(s.Other))
	for name := range s.Other {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
