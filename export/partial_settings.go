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
	Vehicle                 Optional[StrictVehicleSettings]
	Camouflages             Optional[SettingCamouflages]
	PlayerEmblems           Optional[SettingPlayerEmblems]
	Roles                   Optional[SettingRoles]
	ChassisEffects          Optional[StrictChassisEffects]
	GunEffects              Optional[map[string]Optional[StrictGunEffect]]
	ShotEffects             Optional[map[string]Optional[StrictShotEffect]]
	VehicleEffects          Optional[map[string]Optional[StrictVehicleEffect]]
	CombatRoles             Optional[map[string]Optional[SettingCombatRole]]
	ConsumablesBattleroyale Optional[map[string]Optional[StrictConsumable]]
	ConsumablesBossmode     Optional[map[string]Optional[StrictConsumable]]
	ConsumablesCommon       Optional[map[string]Optional[StrictConsumable]]
	ConsumablesGravitizing  Optional[map[string]Optional[StrictConsumable]]
	ConsumablesHellgames    Optional[map[string]Optional[StrictConsumable]]
	ConsumablesLunar        Optional[map[string]Optional[StrictConsumable]]
	ConsumablesMadgames     Optional[map[string]Optional[StrictConsumable]]
	ConsumablesPrototypes   Optional[map[string]Optional[StrictConsumable]]
	ConsumablesList         Optional[SettingConsumablesList]
	ConsumablesDefault      Optional[map[string]Optional[[]Optional[string]]]
	HitMarks                Optional[map[string]Optional[SettingHitMark]]
	Horns                   Optional[map[string]Optional[SettingHorn]]
	Nations                 Optional[map[string]Optional[SettingNation]]
	StuffPackages           Optional[map[string]Optional[SettingStuffPackage]]
	Other                   map[string]json.RawMessage
}

type SettingCombatRole struct {
	DefaultAbilities Optional[[]Optional[string]] `json:"default_abilities,omitempty"`
	ID               Optional[StrictNumber]       `json:"id,omitempty"`
}
type SettingConsumablesList struct {
	Items           Optional[[]Optional[SettingInclude]] `json:"items,omitempty"`
	NextAvailableID Optional[string]                     `json:"nextAvailableId,omitempty"`
}
type SettingInclude struct {
	Path Optional[string] `json:"@path,omitempty"`
}
type SettingHitMark struct {
	ModelSizes  Optional[string] `json:"modelSizes,omitempty"`
	Priority    Optional[string] `json:"priority,omitempty"`
	TextureName Optional[string] `json:"texName,omitempty"`
}
type SettingHorn struct {
	Distance    Optional[string] `json:"distance,omitempty"`
	Gold        Optional[string] `json:"gold,omitempty"`
	ID          Optional[string] `json:"id,omitempty"`
	MaxDuration Optional[string] `json:"maxDuration,omitempty"`
	Mode        Optional[string] `json:"mode,omitempty"`
	Sound       Optional[string] `json:"sound,omitempty"`
	UserString  Optional[string] `json:"userString,omitempty"`
	VehicleTags Optional[string] `json:"vehicleTags,omitempty"`
}
type SettingNation struct {
	ID Optional[string] `json:"id,omitempty"`
}
type SettingStuffPackage struct {
	Content     Optional[string] `json:"content,omitempty"`
	Description Optional[string] `json:"description,omitempty"`
	Effect      Optional[string] `json:"effect,omitempty"`
	ID          Optional[string] `json:"id,omitempty"`
	Image       Optional[string] `json:"image,omitempty"`
	Title       Optional[string] `json:"title,omitempty"`
}

func (s *PartialSettings) UnmarshalJSON(data []byte) error {
	var entries map[string]json.RawMessage
	if err := decodeStrictJSON(data, &entries); err != nil {
		return err
	}
	if entries == nil {
		return fmt.Errorf("settings must be an object")
	}
	var v PartialSettings
	known := map[string]any{
		"vehicle":                  &v.Vehicle,
		"camouflages":              &v.Camouflages,
		"player_emblems":           &v.PlayerEmblems,
		"roles":                    &v.Roles,
		"chassis_effects":          &v.ChassisEffects,
		"gun_effects":              &v.GunEffects,
		"shot_effects":             &v.ShotEffects,
		"vehicle_effects":          &v.VehicleEffects,
		"consumables/battleroyale": &v.ConsumablesBattleroyale,
		"consumables/bossmode":     &v.ConsumablesBossmode,
		"consumables/common":       &v.ConsumablesCommon,
		"consumables/gravitizing":  &v.ConsumablesGravitizing,
		"consumables/hellgames":    &v.ConsumablesHellgames,
		"consumables/lunar":        &v.ConsumablesLunar,
		"consumables/madgames":     &v.ConsumablesMadgames,
		"consumables/prototypes":   &v.ConsumablesPrototypes,
		"combat_roles":             &v.CombatRoles, "consumables/list": &v.ConsumablesList,
		"consumablesDefault": &v.ConsumablesDefault, "hit_marks": &v.HitMarks,
		"horns": &v.Horns, "nations": &v.Nations, "stuff_packages": &v.StuffPackages,
	}
	for name, target := range known {
		if raw, ok := entries[name]; ok {
			if err := decodeStrictJSON(raw, target); err != nil {
				return fmt.Errorf("settings.%s: %w", name, err)
			}
			delete(entries, name)
		}
	}
	v.Other = entries
	*s = v
	return nil
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
