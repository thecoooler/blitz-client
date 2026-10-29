package export

// Effect settings retain the original XML scalar spelling and repeated fields.

type StrictChassisEffects struct {
	Decals    Optional[StrictChassisEffectsDecals]                              `json:"decals,omitzero"`
	Particles Optional[map[string]Optional[StrictChassisEffectsParticlesEntry]] `json:"particles,omitzero"`
}

type StrictChassisEffectsDecals struct {
	BufferPrefs Optional[map[string]Optional[StrictChassisEffectsDecalsBufferPrefsEntry]] `json:"bufferPrefs,omitzero"`
	TextureSets Optional[map[string]Optional[StrictChassisEffectsDecalsTextureSetsEntry]] `json:"textureSets,omitzero"`
}

type StrictChassisEffectsDecalsBufferPrefsEntry struct {
	LifeTime       Optional[XMLInteger] `json:"lifeTime,omitzero"`
	TrianglesCount Optional[XMLInteger] `json:"trianglesCount,omitzero"`
}

type StrictChassisEffectsDecalsTextureSetsEntry struct {
	Bump    Optional[string] `json:"bump,omitzero"`
	Default Optional[string] `json:"default,omitzero"`
	Sand    Optional[string] `json:"sand,omitzero"`
	Snow    Optional[string] `json:"snow,omitzero"`
	Stone   Optional[string] `json:"stone,omitzero"`
}

type StrictChassisEffectsParticlesEntry struct {
	Default Optional[string] `json:"default,omitzero"`
	Ground  Optional[string] `json:"ground,omitzero"`
	Sand    Optional[string] `json:"sand,omitzero"`
	Snow    Optional[string] `json:"snow,omitzero"`
	Stone   Optional[string] `json:"stone,omitzero"`
	Water   Optional[string] `json:"water,omitzero"`
}

type StrictGunEffect struct {
	Blitz   Optional[StrictGunEffectBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictGunEffectEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictGunEffectStages]  `json:"stages,omitzero"`
}

type StrictGunEffectBlitz struct {
	BulletModel    Optional[string]  `json:"bulletModel,omitzero"`
	Effect         Optional[string]  `json:"effect,omitzero"`
	IsTorpedo      Optional[XMLBool] `json:"isTorpedo,omitzero"`
	SoundCannonade Optional[string]  `json:"soundCannonade,omitzero"`
	SoundNPC       Optional[string]  `json:"soundNPC,omitzero"`
	SoundPC        Optional[string]  `json:"soundPC,omitzero"`
}

type StrictGunEffectEffects struct {
	Pixie Optional[StrictGunEffectEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictGunEffectEffectsSound] `json:"sound,omitzero"`
}

type StrictGunEffectEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	Position Optional[string]     `json:"position,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictGunEffectEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictGunEffectStages struct {
	Shooting Optional[XMLNumber] `json:"shooting,omitzero"`
}

type StrictShotEffect struct {
	AirHit           Optional[StrictShotEffectAirHit]           `json:"airHit,omitzero"`
	ArmorCriticalHit Optional[StrictShotEffectArmorCriticalHit] `json:"armorCriticalHit,omitzero"`
	ArmorCriticalMdl Optional[StrictShotEffectArmorCriticalMdl] `json:"armorCriticalMdl,omitzero"`
	ArmorHeavyHit    Optional[StrictShotEffectArmorHeavyHit]    `json:"armorHeavyHit,omitzero"`
	ArmorHit         Optional[StrictShotEffectArmorHit]         `json:"armorHit,omitzero"`
	ArmorResisted    Optional[StrictShotEffectArmorResisted]    `json:"armorResisted,omitzero"`
	ArmorRicochet    Optional[StrictShotEffectArmorRicochet]    `json:"armorRicochet,omitzero"`
	DeepWaterHit     Optional[StrictShotEffectDeepWaterHit]     `json:"deepWaterHit,omitzero"`
	GroundHit        Optional[StrictShotEffectGroundHit]        `json:"groundHit,omitzero"`
	MetalHit         Optional[StrictShotEffectMetalHit]         `json:"metalHit,omitzero"`
	Projectile       Optional[StrictShotEffectProjectile]       `json:"projectile,omitzero"`
	SandHit          Optional[StrictShotEffectSandHit]          `json:"sandHit,omitzero"`
	ShallowWaterHit  Optional[StrictShotEffectShallowWaterHit]  `json:"shallowWaterHit,omitzero"`
	SnowHit          Optional[StrictShotEffectSnowHit]          `json:"snowHit,omitzero"`
	StoneHit         Optional[StrictShotEffectStoneHit]         `json:"stoneHit,omitzero"`
	TargetHitMarks   Optional[StrictShotEffectTargetHitMarks]   `json:"targetHitMarks,omitzero"`
	TargetImpulse    Optional[XMLNumber]                        `json:"targetImpulse,omitzero"`
	TracerType       Optional[string]                           `json:"tracerType,omitzero"`
	Unimplemented    Optional[string]                           `json:"unimplemented,omitzero"`
	WaterParams      Optional[StrictShotEffectWaterParams]      `json:"waterParams,omitzero"`
	WoodHit          Optional[StrictShotEffectWoodHit]          `json:"woodHit,omitzero"`
}

type StrictShotEffectAirHit struct {
	Blitz   Optional[StrictShotEffectAirHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectAirHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectAirHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectAirHitBlitz struct {
	Effect Optional[string] `json:"effect,omitzero"`
	Sound  Optional[string] `json:"sound,omitzero"`
}

type StrictShotEffectAirHitEffects struct {
	Decal Optional[StrictShotEffectAirHitEffectsDecal] `json:"decal,omitzero"`
	Pixie Optional[StrictShotEffectAirHitEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictShotEffectAirHitEffectsSound] `json:"sound,omitzero"`
}

type StrictShotEffectAirHitEffectsDecal struct {
	BumpTexName Optional[string]          `json:"bumpTexName,omitzero"`
	GroupName   Optional[string]          `json:"groupName,omitzero"`
	RandomYaw   Optional[XMLBool]         `json:"randomYaw,omitzero"`
	Size        Optional[XMLNumberVector] `json:"size,omitzero"`
	StartKey    Optional[string]          `json:"startKey,omitzero"`
	TexName     Optional[string]          `json:"texName,omitzero"`
	Variation   Optional[XMLNumber]       `json:"variation,omitzero"`
}

type StrictShotEffectAirHitEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectAirHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectAirHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectArmorCriticalHit struct {
	Blitz   Optional[StrictShotEffectArmorCriticalHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectArmorCriticalHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectArmorCriticalHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectArmorCriticalHitBlitz struct {
	Effect        Optional[string] `json:"effect,omitzero"`
	SoundNPC      Optional[string] `json:"soundNPC,omitzero"`
	SoundPC       Optional[string] `json:"soundPC,omitzero"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitzero"`
}

type StrictShotEffectArmorCriticalHitEffects struct {
	FlashBang Optional[StrictShotEffectArmorCriticalHitEffectsFlashBang] `json:"flashBang,omitzero"`
	Pixie     Optional[StrictShotEffectArmorCriticalHitEffectsPixie]     `json:"pixie,omitzero"`
	ShockWave Optional[StrictShotEffectArmorCriticalHitEffectsShockWave] `json:"shockWave,omitzero"`
	Sound     Optional[StrictShotEffectArmorCriticalHitEffectsSound]     `json:"sound,omitzero"`
}

type StrictShotEffectArmorCriticalHitEffectsFlashBang struct {
	Stages   Optional[StrictShotEffectArmorCriticalHitEffectsFlashBangStages] `json:"stages,omitzero"`
	StartKey Optional[string]                                                 `json:"startKey,omitzero"`
}

type StrictShotEffectArmorCriticalHitEffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictShotEffectArmorCriticalHitEffectsFlashBangStagesStageEntry]] `json:"stage,omitzero"`
}

type StrictShotEffectArmorCriticalHitEffectsFlashBangStagesStageEntry struct {
	Text  Optional[XMLNumber]       `json:"#text,omitzero"`
	Color Optional[XMLNumberVector] `json:"color,omitzero"`
}

type StrictShotEffectArmorCriticalHitEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectArmorCriticalHitEffectsShockWave struct {
	Duration Optional[XMLNumber] `json:"duration,omitzero"`
	StartKey Optional[string]    `json:"startKey,omitzero"`
}

type StrictShotEffectArmorCriticalHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectArmorCriticalHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectArmorCriticalMdl struct {
	Blitz   Optional[StrictShotEffectArmorCriticalMdlBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectArmorCriticalMdlEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectArmorCriticalMdlStages]  `json:"stages,omitzero"`
}

type StrictShotEffectArmorCriticalMdlBlitz struct {
	Effect        Optional[string] `json:"effect,omitzero"`
	SoundNPC      Optional[string] `json:"soundNPC,omitzero"`
	SoundPC       Optional[string] `json:"soundPC,omitzero"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitzero"`
}

type StrictShotEffectArmorCriticalMdlEffects struct {
	FlashBang Optional[StrictShotEffectArmorCriticalMdlEffectsFlashBang] `json:"flashBang,omitzero"`
	Pixie     Optional[StrictShotEffectArmorCriticalMdlEffectsPixie]     `json:"pixie,omitzero"`
	ShockWave Optional[StrictShotEffectArmorCriticalMdlEffectsShockWave] `json:"shockWave,omitzero"`
	Sound     Optional[StrictShotEffectArmorCriticalMdlEffectsSound]     `json:"sound,omitzero"`
}

type StrictShotEffectArmorCriticalMdlEffectsFlashBang struct {
	Stages   Optional[StrictShotEffectArmorCriticalMdlEffectsFlashBangStages] `json:"stages,omitzero"`
	StartKey Optional[string]                                                 `json:"startKey,omitzero"`
}

type StrictShotEffectArmorCriticalMdlEffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictShotEffectArmorCriticalMdlEffectsFlashBangStagesStageEntry]] `json:"stage,omitzero"`
}

type StrictShotEffectArmorCriticalMdlEffectsFlashBangStagesStageEntry struct {
	Text  Optional[XMLNumber]       `json:"#text,omitzero"`
	Color Optional[XMLNumberVector] `json:"color,omitzero"`
}

type StrictShotEffectArmorCriticalMdlEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectArmorCriticalMdlEffectsShockWave struct {
	Duration Optional[XMLNumber] `json:"duration,omitzero"`
	StartKey Optional[string]    `json:"startKey,omitzero"`
}

type StrictShotEffectArmorCriticalMdlEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectArmorCriticalMdlStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectArmorHeavyHit struct {
	Blitz   Optional[StrictShotEffectArmorHeavyHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectArmorHeavyHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectArmorHeavyHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectArmorHeavyHitBlitz struct {
	Effect        Optional[string] `json:"effect,omitzero"`
	SoundNPC      Optional[string] `json:"soundNPC,omitzero"`
	SoundPC       Optional[string] `json:"soundPC,omitzero"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitzero"`
}

type StrictShotEffectArmorHeavyHitEffects struct {
	FlashBang Optional[StrictShotEffectArmorHeavyHitEffectsFlashBang] `json:"flashBang,omitzero"`
	Pixie     Optional[StrictShotEffectArmorHeavyHitEffectsPixie]     `json:"pixie,omitzero"`
	ShockWave Optional[StrictShotEffectArmorHeavyHitEffectsShockWave] `json:"shockWave,omitzero"`
	Sound     Optional[StrictShotEffectArmorHeavyHitEffectsSound]     `json:"sound,omitzero"`
}

type StrictShotEffectArmorHeavyHitEffectsFlashBang struct {
	Stages   Optional[StrictShotEffectArmorHeavyHitEffectsFlashBangStages] `json:"stages,omitzero"`
	StartKey Optional[string]                                              `json:"startKey,omitzero"`
}

type StrictShotEffectArmorHeavyHitEffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictShotEffectArmorHeavyHitEffectsFlashBangStagesStageEntry]] `json:"stage,omitzero"`
}

type StrictShotEffectArmorHeavyHitEffectsFlashBangStagesStageEntry struct {
	Text  Optional[XMLNumber]       `json:"#text,omitzero"`
	Color Optional[XMLNumberVector] `json:"color,omitzero"`
}

type StrictShotEffectArmorHeavyHitEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectArmorHeavyHitEffectsShockWave struct {
	Duration Optional[XMLNumber] `json:"duration,omitzero"`
	StartKey Optional[string]    `json:"startKey,omitzero"`
}

type StrictShotEffectArmorHeavyHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectArmorHeavyHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectArmorHit struct {
	Blitz   Optional[StrictShotEffectArmorHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectArmorHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectArmorHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectArmorHitBlitz struct {
	Effect        Optional[string] `json:"effect,omitzero"`
	SoundNPC      Optional[string] `json:"soundNPC,omitzero"`
	SoundPC       Optional[string] `json:"soundPC,omitzero"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitzero"`
}

type StrictShotEffectArmorHitEffects struct {
	FlashBang Optional[StrictShotEffectArmorHitEffectsFlashBang] `json:"flashBang,omitzero"`
	Pixie     Optional[StrictShotEffectArmorHitEffectsPixie]     `json:"pixie,omitzero"`
	ShockWave Optional[StrictShotEffectArmorHitEffectsShockWave] `json:"shockWave,omitzero"`
	Sound     Optional[StrictShotEffectArmorHitEffectsSound]     `json:"sound,omitzero"`
}

type StrictShotEffectArmorHitEffectsFlashBang struct {
	Stages   Optional[StrictShotEffectArmorHitEffectsFlashBangStages] `json:"stages,omitzero"`
	StartKey Optional[string]                                         `json:"startKey,omitzero"`
}

type StrictShotEffectArmorHitEffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictShotEffectArmorHitEffectsFlashBangStagesStageEntry]] `json:"stage,omitzero"`
}

type StrictShotEffectArmorHitEffectsFlashBangStagesStageEntry struct {
	Text  Optional[XMLNumber]       `json:"#text,omitzero"`
	Color Optional[XMLNumberVector] `json:"color,omitzero"`
}

type StrictShotEffectArmorHitEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectArmorHitEffectsShockWave struct {
	Duration Optional[XMLNumber] `json:"duration,omitzero"`
	StartKey Optional[string]    `json:"startKey,omitzero"`
}

type StrictShotEffectArmorHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectArmorHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectArmorResisted struct {
	Blitz   Optional[StrictShotEffectArmorResistedBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectArmorResistedEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectArmorResistedStages]  `json:"stages,omitzero"`
}

type StrictShotEffectArmorResistedBlitz struct {
	Effect        Optional[string] `json:"effect,omitzero"`
	SoundNPC      Optional[string] `json:"soundNPC,omitzero"`
	SoundPC       Optional[string] `json:"soundPC,omitzero"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitzero"`
}

type StrictShotEffectArmorResistedEffects struct {
	Pixie     Optional[StrictShotEffectArmorResistedEffectsPixie]     `json:"pixie,omitzero"`
	ShockWave Optional[StrictShotEffectArmorResistedEffectsShockWave] `json:"shockWave,omitzero"`
	Sound     Optional[StrictShotEffectArmorResistedEffectsSound]     `json:"sound,omitzero"`
}

type StrictShotEffectArmorResistedEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectArmorResistedEffectsShockWave struct {
	Duration Optional[XMLNumber] `json:"duration,omitzero"`
	StartKey Optional[string]    `json:"startKey,omitzero"`
}

type StrictShotEffectArmorResistedEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectArmorResistedStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectArmorRicochet struct {
	Blitz   Optional[StrictShotEffectArmorRicochetBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectArmorRicochetEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectArmorRicochetStages]  `json:"stages,omitzero"`
}

type StrictShotEffectArmorRicochetBlitz struct {
	Effect        Optional[string] `json:"effect,omitzero"`
	SoundNPC      Optional[string] `json:"soundNPC,omitzero"`
	SoundPC       Optional[string] `json:"soundPC,omitzero"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitzero"`
}

type StrictShotEffectArmorRicochetEffects struct {
	Pixie     Optional[StrictShotEffectArmorRicochetEffectsPixie]     `json:"pixie,omitzero"`
	ShockWave Optional[StrictShotEffectArmorRicochetEffectsShockWave] `json:"shockWave,omitzero"`
	Sound     Optional[StrictShotEffectArmorRicochetEffectsSound]     `json:"sound,omitzero"`
}

type StrictShotEffectArmorRicochetEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectArmorRicochetEffectsShockWave struct {
	Duration Optional[XMLNumber] `json:"duration,omitzero"`
	StartKey Optional[string]    `json:"startKey,omitzero"`
}

type StrictShotEffectArmorRicochetEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectArmorRicochetStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectDeepWaterHit struct {
	Blitz   Optional[StrictShotEffectDeepWaterHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectDeepWaterHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectDeepWaterHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectDeepWaterHitBlitz struct {
	Effect Optional[string] `json:"effect,omitzero"`
	Sound  Optional[string] `json:"sound,omitzero"`
}

type StrictShotEffectDeepWaterHitEffects struct {
	Pixie Optional[StrictShotEffectDeepWaterHitEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictShotEffectDeepWaterHitEffectsSound] `json:"sound,omitzero"`
}

type StrictShotEffectDeepWaterHitEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectDeepWaterHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectDeepWaterHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectGroundHit struct {
	Blitz   Optional[StrictShotEffectGroundHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectGroundHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectGroundHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectGroundHitBlitz struct {
	Effect Optional[string] `json:"effect,omitzero"`
	Sound  Optional[string] `json:"sound,omitzero"`
}

type StrictShotEffectGroundHitEffects struct {
	Decal Optional[StrictShotEffectGroundHitEffectsDecal] `json:"decal,omitzero"`
	Pixie Optional[StrictShotEffectGroundHitEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictShotEffectGroundHitEffectsSound] `json:"sound,omitzero"`
}

type StrictShotEffectGroundHitEffectsDecal struct {
	BumpTexName Optional[XMLScalarList[string, string]] `json:"bumpTexName,omitzero"`
	GroupName   Optional[string]                        `json:"groupName,omitzero"`
	RandomYaw   Optional[XMLBool]                       `json:"randomYaw,omitzero"`
	Size        Optional[XMLNumberVector]               `json:"size,omitzero"`
	StartKey    Optional[string]                        `json:"startKey,omitzero"`
	TexName     Optional[XMLScalarList[string, string]] `json:"texName,omitzero"`
	Variation   Optional[XMLNumber]                     `json:"variation,omitzero"`
}

type StrictShotEffectGroundHitEffectsPixie struct {
	AlwaysUpdateModel Optional[XMLBool]                       `json:"alwaysUpdateModel,omitzero"`
	File              Optional[XMLScalarList[string, string]] `json:"file,omitzero"`
	Force             Optional[XMLInteger]                    `json:"force,omitzero"`
	StartKey          Optional[string]                        `json:"startKey,omitzero"`
}

type StrictShotEffectGroundHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectGroundHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectMetalHit struct {
	Blitz   Optional[StrictShotEffectMetalHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectMetalHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectMetalHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectMetalHitBlitz struct {
	Effect Optional[string] `json:"effect,omitzero"`
	Sound  Optional[string] `json:"sound,omitzero"`
}

type StrictShotEffectMetalHitEffects struct {
	Decal Optional[StrictShotEffectMetalHitEffectsDecal] `json:"decal,omitzero"`
	Pixie Optional[StrictShotEffectMetalHitEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictShotEffectMetalHitEffectsSound] `json:"sound,omitzero"`
}

type StrictShotEffectMetalHitEffectsDecal struct {
	GroupName Optional[string]          `json:"groupName,omitzero"`
	RandomYaw Optional[XMLBool]         `json:"randomYaw,omitzero"`
	Size      Optional[XMLNumberVector] `json:"size,omitzero"`
	StartKey  Optional[string]          `json:"startKey,omitzero"`
	TexName   Optional[string]          `json:"texName,omitzero"`
	Variation Optional[XMLNumber]       `json:"variation,omitzero"`
}

type StrictShotEffectMetalHitEffectsPixie struct {
	File                  Optional[string]     `json:"file,omitzero"`
	Force                 Optional[XMLInteger] `json:"force,omitzero"`
	OrientBySurfaceNormal Optional[XMLBool]    `json:"orientBySurfaceNormal,omitzero"`
	StartKey              Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectMetalHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectMetalHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectProjectile struct {
	Effects      Optional[StrictShotEffectProjectileEffects] `json:"effects,omitzero"`
	Model        Optional[string]                            `json:"model,omitzero"`
	ModelOwnShot Optional[string]                            `json:"modelOwnShot,omitzero"`
}

type StrictShotEffectProjectileEffects struct {
	Pixie Optional[StrictShotEffectProjectileEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictShotEffectProjectileEffectsSound] `json:"sound,omitzero"`
}

type StrictShotEffectProjectileEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	Position Optional[string]     `json:"position,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectProjectileEffectsSound struct {
	EndKey   Optional[string] `json:"endKey,omitzero"`
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectSandHit struct {
	Blitz   Optional[StrictShotEffectSandHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectSandHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectSandHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectSandHitBlitz struct {
	Effect Optional[string] `json:"effect,omitzero"`
	Sound  Optional[string] `json:"sound,omitzero"`
}

type StrictShotEffectSandHitEffects struct {
	Decal Optional[StrictShotEffectSandHitEffectsDecal] `json:"decal,omitzero"`
	Pixie Optional[StrictShotEffectSandHitEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictShotEffectSandHitEffectsSound] `json:"sound,omitzero"`
}

type StrictShotEffectSandHitEffectsDecal struct {
	BumpTexName Optional[string]          `json:"bumpTexName,omitzero"`
	GroupName   Optional[string]          `json:"groupName,omitzero"`
	RandomYaw   Optional[XMLBool]         `json:"randomYaw,omitzero"`
	Size        Optional[XMLNumberVector] `json:"size,omitzero"`
	StartKey    Optional[string]          `json:"startKey,omitzero"`
	TexName     Optional[string]          `json:"texName,omitzero"`
	Variation   Optional[XMLNumber]       `json:"variation,omitzero"`
}

type StrictShotEffectSandHitEffectsPixie struct {
	AlwaysUpdateModel Optional[XMLBool]    `json:"alwaysUpdateModel,omitzero"`
	File              Optional[string]     `json:"file,omitzero"`
	Force             Optional[XMLInteger] `json:"force,omitzero"`
	StartKey          Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectSandHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectSandHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectShallowWaterHit struct {
	Blitz   Optional[StrictShotEffectShallowWaterHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectShallowWaterHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectShallowWaterHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectShallowWaterHitBlitz struct {
	Effect Optional[string] `json:"effect,omitzero"`
	Sound  Optional[string] `json:"sound,omitzero"`
}

type StrictShotEffectShallowWaterHitEffects struct {
	Pixie Optional[StrictShotEffectShallowWaterHitEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictShotEffectShallowWaterHitEffectsSound] `json:"sound,omitzero"`
}

type StrictShotEffectShallowWaterHitEffectsPixie struct {
	File     Optional[string]     `json:"file,omitzero"`
	Force    Optional[XMLInteger] `json:"force,omitzero"`
	StartKey Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectShallowWaterHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectShallowWaterHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectSnowHit struct {
	Blitz   Optional[StrictShotEffectSnowHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectSnowHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectSnowHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectSnowHitBlitz struct {
	Effect Optional[string] `json:"effect,omitzero"`
	Sound  Optional[string] `json:"sound,omitzero"`
}

type StrictShotEffectSnowHitEffects struct {
	Decal Optional[StrictShotEffectSnowHitEffectsDecal] `json:"decal,omitzero"`
	Pixie Optional[StrictShotEffectSnowHitEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictShotEffectSnowHitEffectsSound] `json:"sound,omitzero"`
}

type StrictShotEffectSnowHitEffectsDecal struct {
	BumpTexName Optional[string]          `json:"bumpTexName,omitzero"`
	GroupName   Optional[string]          `json:"groupName,omitzero"`
	RandomYaw   Optional[XMLBool]         `json:"randomYaw,omitzero"`
	Size        Optional[XMLNumberVector] `json:"size,omitzero"`
	StartKey    Optional[string]          `json:"startKey,omitzero"`
	TexName     Optional[string]          `json:"texName,omitzero"`
	Variation   Optional[XMLNumber]       `json:"variation,omitzero"`
}

type StrictShotEffectSnowHitEffectsPixie struct {
	AlwaysUpdateModel Optional[XMLBool]    `json:"alwaysUpdateModel,omitzero"`
	File              Optional[string]     `json:"file,omitzero"`
	Force             Optional[XMLInteger] `json:"force,omitzero"`
	StartKey          Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectSnowHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectSnowHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectStoneHit struct {
	Blitz   Optional[StrictShotEffectStoneHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectStoneHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectStoneHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectStoneHitBlitz struct {
	Effect       Optional[string] `json:"effect,omitzero"`
	ObjectEffect Optional[string] `json:"objectEffect,omitzero"`
	Sound        Optional[string] `json:"sound,omitzero"`
}

type StrictShotEffectStoneHitEffects struct {
	Decal Optional[StrictShotEffectStoneHitEffectsDecal] `json:"decal,omitzero"`
	Pixie Optional[StrictShotEffectStoneHitEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictShotEffectStoneHitEffectsSound] `json:"sound,omitzero"`
}

type StrictShotEffectStoneHitEffectsDecal struct {
	BumpTexName Optional[string]          `json:"bumpTexName,omitzero"`
	GroupName   Optional[string]          `json:"groupName,omitzero"`
	RandomYaw   Optional[XMLBool]         `json:"randomYaw,omitzero"`
	Size        Optional[XMLNumberVector] `json:"size,omitzero"`
	StartKey    Optional[string]          `json:"startKey,omitzero"`
	TexName     Optional[string]          `json:"texName,omitzero"`
	Variation   Optional[XMLNumber]       `json:"variation,omitzero"`
}

type StrictShotEffectStoneHitEffectsPixie struct {
	File                  Optional[string]     `json:"file,omitzero"`
	Force                 Optional[XMLInteger] `json:"force,omitzero"`
	OrientBySurfaceNormal Optional[XMLBool]    `json:"orientBySurfaceNormal,omitzero"`
	StartKey              Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectStoneHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectStoneHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictShotEffectTargetHitMarks struct {
	ArmorHit      Optional[string] `json:"armorHit,omitzero"`
	ArmorResisted Optional[string] `json:"armorResisted,omitzero"`
}

type StrictShotEffectWaterParams struct {
	RippleDiameter    Optional[XMLNumber]           `json:"rippleDiameter,omitzero"`
	ShallowWaterDepth Optional[XMLNumber]           `json:"shallowWaterDepth,omitzero"`
	ShowWaterEffect   Optional[[]Optional[XMLBool]] `json:"showWaterEffect,omitzero"`
}

type StrictShotEffectWoodHit struct {
	Blitz   Optional[StrictShotEffectWoodHitBlitz]   `json:"blitz,omitzero"`
	Effects Optional[StrictShotEffectWoodHitEffects] `json:"effects,omitzero"`
	Stages  Optional[StrictShotEffectWoodHitStages]  `json:"stages,omitzero"`
}

type StrictShotEffectWoodHitBlitz struct {
	Effect Optional[string] `json:"effect,omitzero"`
	Sound  Optional[string] `json:"sound,omitzero"`
}

type StrictShotEffectWoodHitEffects struct {
	Decal Optional[StrictShotEffectWoodHitEffectsDecal] `json:"decal,omitzero"`
	Pixie Optional[StrictShotEffectWoodHitEffectsPixie] `json:"pixie,omitzero"`
	Sound Optional[StrictShotEffectWoodHitEffectsSound] `json:"sound,omitzero"`
}

type StrictShotEffectWoodHitEffectsDecal struct {
	BumpTexName Optional[string]          `json:"bumpTexName,omitzero"`
	GroupName   Optional[string]          `json:"groupName,omitzero"`
	RandomYaw   Optional[XMLBool]         `json:"randomYaw,omitzero"`
	Size        Optional[XMLNumberVector] `json:"size,omitzero"`
	StartKey    Optional[string]          `json:"startKey,omitzero"`
	TexName     Optional[string]          `json:"texName,omitzero"`
	Variation   Optional[XMLNumber]       `json:"variation,omitzero"`
}

type StrictShotEffectWoodHitEffectsPixie struct {
	File                  Optional[string]     `json:"file,omitzero"`
	Force                 Optional[XMLInteger] `json:"force,omitzero"`
	OrientBySurfaceNormal Optional[XMLBool]    `json:"orientBySurfaceNormal,omitzero"`
	StartKey              Optional[string]     `json:"startKey,omitzero"`
}

type StrictShotEffectWoodHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictShotEffectWoodHitStages struct {
	Impact Optional[XMLNumber] `json:"impact,omitzero"`
}

type StrictVehicleEffect struct {
	Variant1 Optional[StrictVehicleEffectVariant1] `json:"variant1,omitzero"`
	Variant2 Optional[StrictVehicleEffectVariant2] `json:"variant2,omitzero"`
	Variant3 Optional[StrictVehicleEffectVariant3] `json:"variant3,omitzero"`
}

type StrictVehicleEffectVariant1 struct {
	Effects Optional[StrictVehicleEffectVariant1Effects] `json:"effects,omitzero"`
	Stages  Optional[StrictVehicleEffectVariant1Stages]  `json:"stages,omitzero"`
}

type StrictVehicleEffectVariant1Effects struct {
	Decal        Optional[StrictVehicleEffectVariant1EffectsDecal]                                                                     `json:"decal,omitzero"`
	FlashBang    Optional[StrictVehicleEffectVariant1EffectsFlashBang]                                                                 `json:"flashBang,omitzero"`
	Pixie        Optional[XMLObjectList[StrictVehicleEffectVariant1EffectsPixieObject, StrictVehicleEffectVariant1EffectsPixieObject]] `json:"pixie,omitzero"`
	ShockWave    Optional[StrictVehicleEffectVariant1EffectsShockWave]                                                                 `json:"shockWave,omitzero"`
	Sound        Optional[XMLObjectList[StrictVehicleEffectVariant1EffectsSoundObject, StrictVehicleEffectVariant1EffectsSoundObject]] `json:"sound,omitzero"`
	StopEmission Optional[StrictVehicleEffectVariant1EffectsStopEmission]                                                              `json:"stopEmission,omitzero"`
}

type StrictVehicleEffectVariant1EffectsDecal struct {
	GroupName Optional[string]          `json:"groupName,omitzero"`
	RandomYaw Optional[XMLBool]         `json:"randomYaw,omitzero"`
	Size      Optional[XMLNumberVector] `json:"size,omitzero"`
	StartKey  Optional[string]          `json:"startKey,omitzero"`
	TexName   Optional[string]          `json:"texName,omitzero"`
}

type StrictVehicleEffectVariant1EffectsFlashBang struct {
	Stages   Optional[StrictVehicleEffectVariant1EffectsFlashBangStages] `json:"stages,omitzero"`
	StartKey Optional[string]                                            `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant1EffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictVehicleEffectVariant1EffectsFlashBangStagesStageEntry]] `json:"stage,omitzero"`
}

type StrictVehicleEffectVariant1EffectsFlashBangStagesStageEntry struct {
	Text  Optional[XMLNumber]       `json:"#text,omitzero"`
	Color Optional[XMLNumberVector] `json:"color,omitzero"`
}

type StrictVehicleEffectVariant1EffectsPixieObject struct {
	AlwaysUpdateModel     Optional[XMLBool]    `json:"alwaysUpdateModel,omitzero"`
	EndKey                Optional[string]     `json:"endKey,omitzero"`
	File                  Optional[string]     `json:"file,omitzero"`
	Force                 Optional[XMLInteger] `json:"force,omitzero"`
	OrientBySurfaceNormal Optional[XMLBool]    `json:"orientBySurfaceNormal,omitzero"`
	Position              Optional[string]     `json:"position,omitzero"`
	StartKey              Optional[string]     `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant1EffectsShockWave struct {
	Duration Optional[XMLNumber] `json:"duration,omitzero"`
	StartKey Optional[string]    `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant1EffectsSoundObject struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant1EffectsStopEmission struct {
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant1Stages struct {
	Death         Optional[XMLNumber]  `json:"death,omitzero"`
	Explosion     Optional[XMLNumber]  `json:"explosion,omitzero"`
	Fire          Optional[XMLNumber]  `json:"fire,omitzero"`
	Impact        Optional[XMLNumber]  `json:"impact,omitzero"`
	NoEmission    Optional[XMLNumber]  `json:"noEmission,omitzero"`
	Static        Optional[XMLNumber]  `json:"static,omitzero"`
	StopDeath     Optional[XMLInteger] `json:"stopDeath,omitzero"`
	StopExplosion Optional[XMLInteger] `json:"stopExplosion,omitzero"`
}

type StrictVehicleEffectVariant2 struct {
	Effects Optional[StrictVehicleEffectVariant2Effects] `json:"effects,omitzero"`
	Stages  Optional[StrictVehicleEffectVariant2Stages]  `json:"stages,omitzero"`
}

type StrictVehicleEffectVariant2Effects struct {
	Decal        Optional[StrictVehicleEffectVariant2EffectsDecal]                                                                     `json:"decal,omitzero"`
	FlashBang    Optional[StrictVehicleEffectVariant2EffectsFlashBang]                                                                 `json:"flashBang,omitzero"`
	Pixie        Optional[XMLObjectList[StrictVehicleEffectVariant2EffectsPixieObject, StrictVehicleEffectVariant2EffectsPixieObject]] `json:"pixie,omitzero"`
	ShockWave    Optional[StrictVehicleEffectVariant2EffectsShockWave]                                                                 `json:"shockWave,omitzero"`
	Sound        Optional[XMLObjectList[StrictVehicleEffectVariant2EffectsSoundObject, StrictVehicleEffectVariant2EffectsSoundObject]] `json:"sound,omitzero"`
	StopEmission Optional[StrictVehicleEffectVariant2EffectsStopEmission]                                                              `json:"stopEmission,omitzero"`
}

type StrictVehicleEffectVariant2EffectsDecal struct {
	GroupName Optional[string]          `json:"groupName,omitzero"`
	RandomYaw Optional[XMLBool]         `json:"randomYaw,omitzero"`
	Size      Optional[XMLNumberVector] `json:"size,omitzero"`
	StartKey  Optional[string]          `json:"startKey,omitzero"`
	TexName   Optional[string]          `json:"texName,omitzero"`
}

type StrictVehicleEffectVariant2EffectsFlashBang struct {
	Stages   Optional[StrictVehicleEffectVariant2EffectsFlashBangStages] `json:"stages,omitzero"`
	StartKey Optional[string]                                            `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant2EffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictVehicleEffectVariant2EffectsFlashBangStagesStageEntry]] `json:"stage,omitzero"`
}

type StrictVehicleEffectVariant2EffectsFlashBangStagesStageEntry struct {
	Text  Optional[XMLNumber]       `json:"#text,omitzero"`
	Color Optional[XMLNumberVector] `json:"color,omitzero"`
}

type StrictVehicleEffectVariant2EffectsPixieObject struct {
	AlwaysUpdateModel Optional[XMLBool]    `json:"alwaysUpdateModel,omitzero"`
	EndKey            Optional[string]     `json:"endKey,omitzero"`
	File              Optional[string]     `json:"file,omitzero"`
	Force             Optional[XMLInteger] `json:"force,omitzero"`
	Position          Optional[string]     `json:"position,omitzero"`
	StartKey          Optional[string]     `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant2EffectsShockWave struct {
	Duration Optional[XMLNumber] `json:"duration,omitzero"`
	StartKey Optional[string]    `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant2EffectsSoundObject struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant2EffectsStopEmission struct {
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant2Stages struct {
	Death         Optional[XMLNumber]  `json:"death,omitzero"`
	Explosion     Optional[XMLNumber]  `json:"explosion,omitzero"`
	Fire          Optional[XMLNumber]  `json:"fire,omitzero"`
	Impact        Optional[XMLNumber]  `json:"impact,omitzero"`
	NoEmission    Optional[XMLNumber]  `json:"noEmission,omitzero"`
	Static        Optional[XMLNumber]  `json:"static,omitzero"`
	StopDeath     Optional[XMLInteger] `json:"stopDeath,omitzero"`
	StopExplosion Optional[XMLInteger] `json:"stopExplosion,omitzero"`
}

type StrictVehicleEffectVariant3 struct {
	Effects Optional[StrictVehicleEffectVariant3Effects] `json:"effects,omitzero"`
	Stages  Optional[StrictVehicleEffectVariant3Stages]  `json:"stages,omitzero"`
}

type StrictVehicleEffectVariant3Effects struct {
	Decal     Optional[StrictVehicleEffectVariant3EffectsDecal]                  `json:"decal,omitzero"`
	FlashBang Optional[StrictVehicleEffectVariant3EffectsFlashBang]              `json:"flashBang,omitzero"`
	Pixie     Optional[[]Optional[StrictVehicleEffectVariant3EffectsPixieEntry]] `json:"pixie,omitzero"`
	ShockWave Optional[StrictVehicleEffectVariant3EffectsShockWave]              `json:"shockWave,omitzero"`
	Sound     Optional[[]Optional[StrictVehicleEffectVariant3EffectsSoundEntry]] `json:"sound,omitzero"`
}

type StrictVehicleEffectVariant3EffectsDecal struct {
	GroupName Optional[string]          `json:"groupName,omitzero"`
	RandomYaw Optional[XMLBool]         `json:"randomYaw,omitzero"`
	Size      Optional[XMLNumberVector] `json:"size,omitzero"`
	StartKey  Optional[string]          `json:"startKey,omitzero"`
	TexName   Optional[string]          `json:"texName,omitzero"`
}

type StrictVehicleEffectVariant3EffectsFlashBang struct {
	Stages   Optional[StrictVehicleEffectVariant3EffectsFlashBangStages] `json:"stages,omitzero"`
	StartKey Optional[string]                                            `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant3EffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictVehicleEffectVariant3EffectsFlashBangStagesStageEntry]] `json:"stage,omitzero"`
}

type StrictVehicleEffectVariant3EffectsFlashBangStagesStageEntry struct {
	Text  Optional[XMLNumber]       `json:"#text,omitzero"`
	Color Optional[XMLNumberVector] `json:"color,omitzero"`
}

type StrictVehicleEffectVariant3EffectsPixieEntry struct {
	AlwaysUpdateModel Optional[XMLBool]    `json:"alwaysUpdateModel,omitzero"`
	EndKey            Optional[string]     `json:"endKey,omitzero"`
	File              Optional[string]     `json:"file,omitzero"`
	Force             Optional[XMLInteger] `json:"force,omitzero"`
	Position          Optional[string]     `json:"position,omitzero"`
	StartKey          Optional[string]     `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant3EffectsShockWave struct {
	Duration Optional[XMLNumber] `json:"duration,omitzero"`
	StartKey Optional[string]    `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant3EffectsSoundEntry struct {
	Name     Optional[string] `json:"name,omitzero"`
	StartKey Optional[string] `json:"startKey,omitzero"`
}

type StrictVehicleEffectVariant3Stages struct {
	Death     Optional[XMLNumber]  `json:"death,omitzero"`
	Static    Optional[XMLNumber]  `json:"static,omitzero"`
	StopDeath Optional[XMLInteger] `json:"stopDeath,omitzero"`
}
