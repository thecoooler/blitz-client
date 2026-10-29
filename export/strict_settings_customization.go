package export

// SettingCamouflages describes the observed camouflages settings document.
// Names of camouflages, groups and invisibility categories are source keys.
type SettingCamouflages struct {
	Camouflages                      Optional[map[string]Optional[SettingCamouflage]]      `json:"camouflages,omitzero"`
	Groups                           Optional[map[string]Optional[SettingCamouflageGroup]] `json:"groups,omitzero"`
	Invisibility                     Optional[SettingCamouflageInvisibility]               `json:"invisibility,omitzero"`
	NextAvailableID                  Optional[XMLInteger]                                  `json:"nextAvailableId,omitzero"`
	NextAvailableIDForDynamicContent Optional[XMLInteger]                                  `json:"nextAvailableIdForDynamicContent,omitzero"`
}
type SettingCamouflage struct {
	Category           Optional[string]                        `json:"category,omitzero"`
	Description        Optional[XMLLocalizationKey]            `json:"description,omitzero"`
	Group              Optional[string]                        `json:"group,omitzero"`
	Icon               Optional[string]                        `json:"icon,omitzero"`
	ID                 Optional[XMLInteger]                    `json:"id,omitzero"`
	Kind               Optional[string]                        `json:"kind,omitzero"`
	NotInShop          Optional[XMLBool]                       `json:"notInShop,omitzero"`
	Script             Optional[XMLScalarList[string, string]] `json:"script,omitzero"`
	Tags               Optional[string]                        `json:"tags,omitzero"`
	UnlockClanLevel    Optional[XMLInteger]                    `json:"unlockClanLevel,omitzero"`
	UnlockCostCategory Optional[XMLScalarList[string, string]] `json:"unlockCostCategory,omitzero"`
	UnlockPremium      Optional[StrictNull]                    `json:"unlockPremium,omitzero"`
	UnlockQuest        Optional[StrictNull]                    `json:"unlockQuest,omitzero"`
	UnlockShopBundle   Optional[StrictNull]                    `json:"unlockShopBundle,omitzero"`
	UnlockTankRank     Optional[XMLInteger]                    `json:"unlockTankRank,omitzero"`
	UserString         Optional[XMLLocalizationKey]            `json:"userString,omitzero"`
	VehicleFilter      Optional[SettingCamouflageFilter]       `json:"vehicleFilter,omitzero"`
}
type SettingCamouflageGroup struct {
	Rarity     Optional[XMLInteger] `json:"rarity,omitzero"`
	UserString Optional[string]     `json:"userString,omitzero"`
}
type SettingCamouflageInvisibility struct {
	Categories Optional[map[string]Optional[SettingCamouflageCategory]] `json:"categories,omitzero"`
}
type SettingCamouflageCategory struct {
	ATSPG      Optional[XMLNumber] `json:"AT-SPG,omitzero"`
	HeavyTank  Optional[XMLNumber] `json:"heavyTank,omitzero"`
	LightTank  Optional[XMLNumber] `json:"lightTank,omitzero"`
	MediumTank Optional[XMLNumber] `json:"mediumTank,omitzero"`
}
type SettingCamouflageFilter struct {
	Exclude Optional[SettingCamouflageFilterPart]                                             `json:"exclude,omitzero"`
	Include Optional[XMLObjectList[SettingCamouflageFilterPart, SettingCamouflageFilterPart]] `json:"include,omitzero"`
}
type SettingCamouflageFilterPart struct {
	Nations Optional[string]                         `json:"nations,omitzero"`
	Vehicle Optional[SettingCamouflageFilterVehicle] `json:"vehicle,omitzero"`
}
type SettingCamouflageFilterVehicle struct {
	Text     Optional[string]     `json:"#text,omitzero"`
	Name     Optional[string]     `json:"name,omitzero"`
	Tags     Optional[string]     `json:"tags,omitzero"`
	MaxLevel Optional[XMLInteger] `json:"maxLevel,omitzero"`
	MinLevel Optional[XMLInteger] `json:"minLevel,omitzero"`
}

// SettingPlayerEmblems is keyed by group; emblem names remain dynamic.
type SettingPlayerEmblems map[string]Optional[SettingEmblemGroup]
type SettingEmblemGroup struct {
	Emblems     Optional[map[string]Optional[XMLObjectList[SettingEmblem, SettingEmblem]]] `json:"emblems,omitzero"`
	NotInShop   Optional[XMLBool]                                                          `json:"notInShop,omitzero"`
	PriceFactor Optional[XMLNumber]                                                        `json:"priceFactor,omitzero"`
	UserString  Optional[XMLLocalizationKey]                                               `json:"userString,omitzero"`
}
type SettingEmblem struct {
	ID         Optional[XMLInteger]         `json:"id,omitzero"`
	TexName    Optional[string]             `json:"texName,omitzero"`
	UserString Optional[XMLLocalizationKey] `json:"userString,omitzero"`
}

// SettingRoles retains the separate level settings and dynamic role names.
type SettingRoles struct {
	BossRole       Optional[SettingRole]               `json:"bossRole,omitzero"`
	GravitonRole   Optional[SettingRole]               `json:"gravitonRole,omitzero"`
	HealerRole     Optional[SettingRole]               `json:"healerRole,omitzero"`
	KamikazeRole   Optional[SettingRole]               `json:"kamikazeRole,omitzero"`
	KillerRole     Optional[SettingRole]               `json:"killerRole,omitzero"`
	LevelsSettings Optional[SettingRoleLevelsSettings] `json:"levelsSettings,omitzero"`
}
type SettingRole struct {
	Boss        Optional[StrictNull]         `json:"boss,omitzero"`
	Description Optional[XMLLocalizationKey] `json:"description,omitzero"`
	Icon        Optional[string]             `json:"icon,omitzero"`
	ID          Optional[XMLInteger]         `json:"id,omitzero"`
	Levels      Optional[SettingRoleLevels]  `json:"levels,omitzero"`
	UserString  Optional[XMLLocalizationKey] `json:"userString,omitzero"`
}
type SettingRoleLevels struct {
	Level Optional[[]Optional[SettingRoleLevel]] `json:"level,omitzero"`
}
type SettingRoleLevel struct {
	Abilities Optional[SettingRoleAbilities] `json:"abilities,omitzero"`
	Script    Optional[StrictNull]           `json:"script,omitzero"`
}
type SettingRoleAbilities struct {
	Slot Optional[[]Optional[SettingRoleSlot]] `json:"slot,omitzero"`
}
type SettingRoleSlot struct {
	Text  Optional[string]     `json:"#text,omitzero"`
	Index Optional[XMLInteger] `json:"@index,omitzero"`
}
type SettingRoleLevelsSettings struct {
	Level Optional[[]Optional[SettingRoleLevelSetting]] `json:"level,omitzero"`
}
type SettingRoleLevelSetting struct {
	LevelNum   Optional[XMLInteger] `json:"levelNum,omitzero"`
	RolePoints Optional[XMLInteger] `json:"rolePoints,omitzero"`
}
