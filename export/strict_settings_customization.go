package export

// SettingCamouflages describes the observed camouflages settings document.
// Names of camouflages, groups and invisibility categories are source keys.
type SettingCamouflages struct {
	Camouflages                      Optional[map[string]Optional[SettingCamouflage]]      `json:"camouflages,omitempty"`
	Groups                           Optional[map[string]Optional[SettingCamouflageGroup]] `json:"groups,omitempty"`
	Invisibility                     Optional[SettingCamouflageInvisibility]               `json:"invisibility,omitempty"`
	NextAvailableID                  Optional[string]                                      `json:"nextAvailableId,omitempty"`
	NextAvailableIDForDynamicContent Optional[string]                                      `json:"nextAvailableIdForDynamicContent,omitempty"`
}
type SettingCamouflage struct {
	Category           Optional[string]                  `json:"category,omitempty"`
	Description        Optional[string]                  `json:"description,omitempty"`
	Group              Optional[string]                  `json:"group,omitempty"`
	Icon               Optional[string]                  `json:"icon,omitempty"`
	ID                 Optional[string]                  `json:"id,omitempty"`
	Kind               Optional[string]                  `json:"kind,omitempty"`
	NotInShop          Optional[string]                  `json:"notInShop,omitempty"`
	Script             Optional[XMLScalarList[string]]   `json:"script,omitempty"`
	Tags               Optional[string]                  `json:"tags,omitempty"`
	UnlockClanLevel    Optional[string]                  `json:"unlockClanLevel,omitempty"`
	UnlockCostCategory Optional[XMLScalarList[string]]   `json:"unlockCostCategory,omitempty"`
	UnlockPremium      Optional[StrictNull]              `json:"unlockPremium,omitempty"`
	UnlockQuest        Optional[StrictNull]              `json:"unlockQuest,omitempty"`
	UnlockShopBundle   Optional[StrictNull]              `json:"unlockShopBundle,omitempty"`
	UnlockTankRank     Optional[string]                  `json:"unlockTankRank,omitempty"`
	UserString         Optional[string]                  `json:"userString,omitempty"`
	VehicleFilter      Optional[SettingCamouflageFilter] `json:"vehicleFilter,omitempty"`
}
type SettingCamouflageGroup struct {
	Rarity     Optional[string] `json:"rarity,omitempty"`
	UserString Optional[string] `json:"userString,omitempty"`
}
type SettingCamouflageInvisibility struct {
	Categories Optional[map[string]Optional[SettingCamouflageCategory]] `json:"categories,omitempty"`
}
type SettingCamouflageCategory struct {
	ATSPG      Optional[string] `json:"AT-SPG,omitempty"`
	HeavyTank  Optional[string] `json:"heavyTank,omitempty"`
	LightTank  Optional[string] `json:"lightTank,omitempty"`
	MediumTank Optional[string] `json:"mediumTank,omitempty"`
}
type SettingCamouflageFilter struct {
	Exclude Optional[SettingCamouflageFilterPart]                                             `json:"exclude,omitempty"`
	Include Optional[XMLObjectList[SettingCamouflageFilterPart, SettingCamouflageFilterPart]] `json:"include,omitempty"`
}
type SettingCamouflageFilterPart struct {
	Nations Optional[string]                         `json:"nations,omitempty"`
	Vehicle Optional[SettingCamouflageFilterVehicle] `json:"vehicle,omitempty"`
}
type SettingCamouflageFilterVehicle struct {
	Text     Optional[string] `json:"#text,omitempty"`
	Name     Optional[string] `json:"name,omitempty"`
	Tags     Optional[string] `json:"tags,omitempty"`
	MaxLevel Optional[string] `json:"maxLevel,omitempty"`
	MinLevel Optional[string] `json:"minLevel,omitempty"`
}

// SettingPlayerEmblems is keyed by group; emblem names remain dynamic.
type SettingPlayerEmblems map[string]Optional[SettingEmblemGroup]
type SettingEmblemGroup struct {
	Emblems     Optional[map[string]Optional[XMLObjectList[SettingEmblem, SettingEmblem]]] `json:"emblems,omitempty"`
	NotInShop   Optional[string]                                                           `json:"notInShop,omitempty"`
	PriceFactor Optional[string]                                                           `json:"priceFactor,omitempty"`
	UserString  Optional[string]                                                           `json:"userString,omitempty"`
}
type SettingEmblem struct {
	ID         Optional[string] `json:"id,omitempty"`
	TexName    Optional[string] `json:"texName,omitempty"`
	UserString Optional[string] `json:"userString,omitempty"`
}

// SettingRoles retains the separate level settings and dynamic role names.
type SettingRoles struct {
	BossRole       Optional[SettingRole]               `json:"bossRole,omitempty"`
	GravitonRole   Optional[SettingRole]               `json:"gravitonRole,omitempty"`
	HealerRole     Optional[SettingRole]               `json:"healerRole,omitempty"`
	KamikazeRole   Optional[SettingRole]               `json:"kamikazeRole,omitempty"`
	KillerRole     Optional[SettingRole]               `json:"killerRole,omitempty"`
	LevelsSettings Optional[SettingRoleLevelsSettings] `json:"levelsSettings,omitempty"`
}
type SettingRole struct {
	Boss        Optional[StrictNull]        `json:"boss,omitempty"`
	Description Optional[string]            `json:"description,omitempty"`
	Icon        Optional[string]            `json:"icon,omitempty"`
	ID          Optional[string]            `json:"id,omitempty"`
	Levels      Optional[SettingRoleLevels] `json:"levels,omitempty"`
	UserString  Optional[string]            `json:"userString,omitempty"`
}
type SettingRoleLevels struct {
	Level Optional[[]Optional[SettingRoleLevel]] `json:"level,omitempty"`
}
type SettingRoleLevel struct {
	Abilities Optional[SettingRoleAbilities] `json:"abilities,omitempty"`
	Script    Optional[StrictNull]           `json:"script,omitempty"`
}
type SettingRoleAbilities struct {
	Slot Optional[[]Optional[SettingRoleSlot]] `json:"slot,omitempty"`
}
type SettingRoleSlot struct {
	Text  Optional[string] `json:"#text,omitempty"`
	Index Optional[string] `json:"@index,omitempty"`
}
type SettingRoleLevelsSettings struct {
	Level Optional[[]Optional[SettingRoleLevelSetting]] `json:"level,omitempty"`
}
type SettingRoleLevelSetting struct {
	LevelNum   Optional[string] `json:"levelNum,omitempty"`
	RolePoints Optional[string] `json:"rolePoints,omitempty"`
}
