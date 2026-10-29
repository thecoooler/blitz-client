package export

// Effect settings retain the original XML scalar spelling and repeated fields.

type StrictChassisEffects struct {
	Decals    Optional[StrictChassisEffectsDecals]                              `json:"decals,omitempty"`
	Particles Optional[map[string]Optional[StrictChassisEffectsParticlesEntry]] `json:"particles,omitempty"`
}

type StrictChassisEffectsDecals struct {
	BufferPrefs Optional[map[string]Optional[StrictChassisEffectsDecalsBufferPrefsEntry]] `json:"bufferPrefs,omitempty"`
	TextureSets Optional[map[string]Optional[StrictChassisEffectsDecalsTextureSetsEntry]] `json:"textureSets,omitempty"`
}

type StrictChassisEffectsDecalsBufferPrefsEntry struct {
	LifeTime       Optional[string] `json:"lifeTime,omitempty"`
	TrianglesCount Optional[string] `json:"trianglesCount,omitempty"`
}

type StrictChassisEffectsDecalsTextureSetsEntry struct {
	Bump    Optional[string] `json:"bump,omitempty"`
	Default Optional[string] `json:"default,omitempty"`
	Sand    Optional[string] `json:"sand,omitempty"`
	Snow    Optional[string] `json:"snow,omitempty"`
	Stone   Optional[string] `json:"stone,omitempty"`
}

type StrictChassisEffectsParticlesEntry struct {
	Default Optional[string] `json:"default,omitempty"`
	Ground  Optional[string] `json:"ground,omitempty"`
	Sand    Optional[string] `json:"sand,omitempty"`
	Snow    Optional[string] `json:"snow,omitempty"`
	Stone   Optional[string] `json:"stone,omitempty"`
	Water   Optional[string] `json:"water,omitempty"`
}

type StrictGunEffect struct {
	Blitz   Optional[StrictGunEffectBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictGunEffectEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictGunEffectStages]  `json:"stages,omitempty"`
}

type StrictGunEffectBlitz struct {
	BulletModel    Optional[string] `json:"bulletModel,omitempty"`
	Effect         Optional[string] `json:"effect,omitempty"`
	IsTorpedo      Optional[string] `json:"isTorpedo,omitempty"`
	SoundCannonade Optional[string] `json:"soundCannonade,omitempty"`
	SoundNPC       Optional[string] `json:"soundNPC,omitempty"`
	SoundPC        Optional[string] `json:"soundPC,omitempty"`
}

type StrictGunEffectEffects struct {
	Pixie Optional[StrictGunEffectEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictGunEffectEffectsSound] `json:"sound,omitempty"`
}

type StrictGunEffectEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	Position Optional[string] `json:"position,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictGunEffectEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictGunEffectStages struct {
	Shooting Optional[string] `json:"shooting,omitempty"`
}

type StrictShotEffect struct {
	AirHit           Optional[StrictShotEffectAirHit]           `json:"airHit,omitempty"`
	ArmorCriticalHit Optional[StrictShotEffectArmorCriticalHit] `json:"armorCriticalHit,omitempty"`
	ArmorCriticalMdl Optional[StrictShotEffectArmorCriticalMdl] `json:"armorCriticalMdl,omitempty"`
	ArmorHeavyHit    Optional[StrictShotEffectArmorHeavyHit]    `json:"armorHeavyHit,omitempty"`
	ArmorHit         Optional[StrictShotEffectArmorHit]         `json:"armorHit,omitempty"`
	ArmorResisted    Optional[StrictShotEffectArmorResisted]    `json:"armorResisted,omitempty"`
	ArmorRicochet    Optional[StrictShotEffectArmorRicochet]    `json:"armorRicochet,omitempty"`
	DeepWaterHit     Optional[StrictShotEffectDeepWaterHit]     `json:"deepWaterHit,omitempty"`
	GroundHit        Optional[StrictShotEffectGroundHit]        `json:"groundHit,omitempty"`
	MetalHit         Optional[StrictShotEffectMetalHit]         `json:"metalHit,omitempty"`
	Projectile       Optional[StrictShotEffectProjectile]       `json:"projectile,omitempty"`
	SandHit          Optional[StrictShotEffectSandHit]          `json:"sandHit,omitempty"`
	ShallowWaterHit  Optional[StrictShotEffectShallowWaterHit]  `json:"shallowWaterHit,omitempty"`
	SnowHit          Optional[StrictShotEffectSnowHit]          `json:"snowHit,omitempty"`
	StoneHit         Optional[StrictShotEffectStoneHit]         `json:"stoneHit,omitempty"`
	TargetHitMarks   Optional[StrictShotEffectTargetHitMarks]   `json:"targetHitMarks,omitempty"`
	TargetImpulse    Optional[string]                           `json:"targetImpulse,omitempty"`
	TracerType       Optional[string]                           `json:"tracerType,omitempty"`
	Unimplemented    Optional[string]                           `json:"unimplemented,omitempty"`
	WaterParams      Optional[StrictShotEffectWaterParams]      `json:"waterParams,omitempty"`
	WoodHit          Optional[StrictShotEffectWoodHit]          `json:"woodHit,omitempty"`
}

type StrictShotEffectAirHit struct {
	Blitz   Optional[StrictShotEffectAirHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectAirHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectAirHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectAirHitBlitz struct {
	Effect Optional[string] `json:"effect,omitempty"`
	Sound  Optional[string] `json:"sound,omitempty"`
}

type StrictShotEffectAirHitEffects struct {
	Decal Optional[StrictShotEffectAirHitEffectsDecal] `json:"decal,omitempty"`
	Pixie Optional[StrictShotEffectAirHitEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictShotEffectAirHitEffectsSound] `json:"sound,omitempty"`
}

type StrictShotEffectAirHitEffectsDecal struct {
	BumpTexName Optional[string] `json:"bumpTexName,omitempty"`
	GroupName   Optional[string] `json:"groupName,omitempty"`
	RandomYaw   Optional[string] `json:"randomYaw,omitempty"`
	Size        Optional[string] `json:"size,omitempty"`
	StartKey    Optional[string] `json:"startKey,omitempty"`
	TexName     Optional[string] `json:"texName,omitempty"`
	Variation   Optional[string] `json:"variation,omitempty"`
}

type StrictShotEffectAirHitEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectAirHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectAirHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectArmorCriticalHit struct {
	Blitz   Optional[StrictShotEffectArmorCriticalHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectArmorCriticalHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectArmorCriticalHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectArmorCriticalHitBlitz struct {
	Effect        Optional[string] `json:"effect,omitempty"`
	SoundNPC      Optional[string] `json:"soundNPC,omitempty"`
	SoundPC       Optional[string] `json:"soundPC,omitempty"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitempty"`
}

type StrictShotEffectArmorCriticalHitEffects struct {
	FlashBang Optional[StrictShotEffectArmorCriticalHitEffectsFlashBang] `json:"flashBang,omitempty"`
	Pixie     Optional[StrictShotEffectArmorCriticalHitEffectsPixie]     `json:"pixie,omitempty"`
	ShockWave Optional[StrictShotEffectArmorCriticalHitEffectsShockWave] `json:"shockWave,omitempty"`
	Sound     Optional[StrictShotEffectArmorCriticalHitEffectsSound]     `json:"sound,omitempty"`
}

type StrictShotEffectArmorCriticalHitEffectsFlashBang struct {
	Stages   Optional[StrictShotEffectArmorCriticalHitEffectsFlashBangStages] `json:"stages,omitempty"`
	StartKey Optional[string]                                                 `json:"startKey,omitempty"`
}

type StrictShotEffectArmorCriticalHitEffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictShotEffectArmorCriticalHitEffectsFlashBangStagesStageEntry]] `json:"stage,omitempty"`
}

type StrictShotEffectArmorCriticalHitEffectsFlashBangStagesStageEntry struct {
	Text  Optional[string] `json:"#text,omitempty"`
	Color Optional[string] `json:"color,omitempty"`
}

type StrictShotEffectArmorCriticalHitEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorCriticalHitEffectsShockWave struct {
	Duration Optional[string] `json:"duration,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorCriticalHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorCriticalHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectArmorCriticalMdl struct {
	Blitz   Optional[StrictShotEffectArmorCriticalMdlBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectArmorCriticalMdlEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectArmorCriticalMdlStages]  `json:"stages,omitempty"`
}

type StrictShotEffectArmorCriticalMdlBlitz struct {
	Effect        Optional[string] `json:"effect,omitempty"`
	SoundNPC      Optional[string] `json:"soundNPC,omitempty"`
	SoundPC       Optional[string] `json:"soundPC,omitempty"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitempty"`
}

type StrictShotEffectArmorCriticalMdlEffects struct {
	FlashBang Optional[StrictShotEffectArmorCriticalMdlEffectsFlashBang] `json:"flashBang,omitempty"`
	Pixie     Optional[StrictShotEffectArmorCriticalMdlEffectsPixie]     `json:"pixie,omitempty"`
	ShockWave Optional[StrictShotEffectArmorCriticalMdlEffectsShockWave] `json:"shockWave,omitempty"`
	Sound     Optional[StrictShotEffectArmorCriticalMdlEffectsSound]     `json:"sound,omitempty"`
}

type StrictShotEffectArmorCriticalMdlEffectsFlashBang struct {
	Stages   Optional[StrictShotEffectArmorCriticalMdlEffectsFlashBangStages] `json:"stages,omitempty"`
	StartKey Optional[string]                                                 `json:"startKey,omitempty"`
}

type StrictShotEffectArmorCriticalMdlEffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictShotEffectArmorCriticalMdlEffectsFlashBangStagesStageEntry]] `json:"stage,omitempty"`
}

type StrictShotEffectArmorCriticalMdlEffectsFlashBangStagesStageEntry struct {
	Text  Optional[string] `json:"#text,omitempty"`
	Color Optional[string] `json:"color,omitempty"`
}

type StrictShotEffectArmorCriticalMdlEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorCriticalMdlEffectsShockWave struct {
	Duration Optional[string] `json:"duration,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorCriticalMdlEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorCriticalMdlStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectArmorHeavyHit struct {
	Blitz   Optional[StrictShotEffectArmorHeavyHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectArmorHeavyHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectArmorHeavyHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectArmorHeavyHitBlitz struct {
	Effect        Optional[string] `json:"effect,omitempty"`
	SoundNPC      Optional[string] `json:"soundNPC,omitempty"`
	SoundPC       Optional[string] `json:"soundPC,omitempty"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitempty"`
}

type StrictShotEffectArmorHeavyHitEffects struct {
	FlashBang Optional[StrictShotEffectArmorHeavyHitEffectsFlashBang] `json:"flashBang,omitempty"`
	Pixie     Optional[StrictShotEffectArmorHeavyHitEffectsPixie]     `json:"pixie,omitempty"`
	ShockWave Optional[StrictShotEffectArmorHeavyHitEffectsShockWave] `json:"shockWave,omitempty"`
	Sound     Optional[StrictShotEffectArmorHeavyHitEffectsSound]     `json:"sound,omitempty"`
}

type StrictShotEffectArmorHeavyHitEffectsFlashBang struct {
	Stages   Optional[StrictShotEffectArmorHeavyHitEffectsFlashBangStages] `json:"stages,omitempty"`
	StartKey Optional[string]                                              `json:"startKey,omitempty"`
}

type StrictShotEffectArmorHeavyHitEffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictShotEffectArmorHeavyHitEffectsFlashBangStagesStageEntry]] `json:"stage,omitempty"`
}

type StrictShotEffectArmorHeavyHitEffectsFlashBangStagesStageEntry struct {
	Text  Optional[string] `json:"#text,omitempty"`
	Color Optional[string] `json:"color,omitempty"`
}

type StrictShotEffectArmorHeavyHitEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorHeavyHitEffectsShockWave struct {
	Duration Optional[string] `json:"duration,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorHeavyHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorHeavyHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectArmorHit struct {
	Blitz   Optional[StrictShotEffectArmorHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectArmorHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectArmorHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectArmorHitBlitz struct {
	Effect        Optional[string] `json:"effect,omitempty"`
	SoundNPC      Optional[string] `json:"soundNPC,omitempty"`
	SoundPC       Optional[string] `json:"soundPC,omitempty"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitempty"`
}

type StrictShotEffectArmorHitEffects struct {
	FlashBang Optional[StrictShotEffectArmorHitEffectsFlashBang] `json:"flashBang,omitempty"`
	Pixie     Optional[StrictShotEffectArmorHitEffectsPixie]     `json:"pixie,omitempty"`
	ShockWave Optional[StrictShotEffectArmorHitEffectsShockWave] `json:"shockWave,omitempty"`
	Sound     Optional[StrictShotEffectArmorHitEffectsSound]     `json:"sound,omitempty"`
}

type StrictShotEffectArmorHitEffectsFlashBang struct {
	Stages   Optional[StrictShotEffectArmorHitEffectsFlashBangStages] `json:"stages,omitempty"`
	StartKey Optional[string]                                         `json:"startKey,omitempty"`
}

type StrictShotEffectArmorHitEffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictShotEffectArmorHitEffectsFlashBangStagesStageEntry]] `json:"stage,omitempty"`
}

type StrictShotEffectArmorHitEffectsFlashBangStagesStageEntry struct {
	Text  Optional[string] `json:"#text,omitempty"`
	Color Optional[string] `json:"color,omitempty"`
}

type StrictShotEffectArmorHitEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorHitEffectsShockWave struct {
	Duration Optional[string] `json:"duration,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectArmorResisted struct {
	Blitz   Optional[StrictShotEffectArmorResistedBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectArmorResistedEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectArmorResistedStages]  `json:"stages,omitempty"`
}

type StrictShotEffectArmorResistedBlitz struct {
	Effect        Optional[string] `json:"effect,omitempty"`
	SoundNPC      Optional[string] `json:"soundNPC,omitempty"`
	SoundPC       Optional[string] `json:"soundPC,omitempty"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitempty"`
}

type StrictShotEffectArmorResistedEffects struct {
	Pixie     Optional[StrictShotEffectArmorResistedEffectsPixie]     `json:"pixie,omitempty"`
	ShockWave Optional[StrictShotEffectArmorResistedEffectsShockWave] `json:"shockWave,omitempty"`
	Sound     Optional[StrictShotEffectArmorResistedEffectsSound]     `json:"sound,omitempty"`
}

type StrictShotEffectArmorResistedEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorResistedEffectsShockWave struct {
	Duration Optional[string] `json:"duration,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorResistedEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorResistedStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectArmorRicochet struct {
	Blitz   Optional[StrictShotEffectArmorRicochetBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectArmorRicochetEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectArmorRicochetStages]  `json:"stages,omitempty"`
}

type StrictShotEffectArmorRicochetBlitz struct {
	Effect        Optional[string] `json:"effect,omitempty"`
	SoundNPC      Optional[string] `json:"soundNPC,omitempty"`
	SoundPC       Optional[string] `json:"soundPC,omitempty"`
	SoundSniperPC Optional[string] `json:"soundSniperPC,omitempty"`
}

type StrictShotEffectArmorRicochetEffects struct {
	Pixie     Optional[StrictShotEffectArmorRicochetEffectsPixie]     `json:"pixie,omitempty"`
	ShockWave Optional[StrictShotEffectArmorRicochetEffectsShockWave] `json:"shockWave,omitempty"`
	Sound     Optional[StrictShotEffectArmorRicochetEffectsSound]     `json:"sound,omitempty"`
}

type StrictShotEffectArmorRicochetEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorRicochetEffectsShockWave struct {
	Duration Optional[string] `json:"duration,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorRicochetEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectArmorRicochetStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectDeepWaterHit struct {
	Blitz   Optional[StrictShotEffectDeepWaterHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectDeepWaterHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectDeepWaterHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectDeepWaterHitBlitz struct {
	Effect Optional[string] `json:"effect,omitempty"`
	Sound  Optional[string] `json:"sound,omitempty"`
}

type StrictShotEffectDeepWaterHitEffects struct {
	Pixie Optional[StrictShotEffectDeepWaterHitEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictShotEffectDeepWaterHitEffectsSound] `json:"sound,omitempty"`
}

type StrictShotEffectDeepWaterHitEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectDeepWaterHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectDeepWaterHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectGroundHit struct {
	Blitz   Optional[StrictShotEffectGroundHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectGroundHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectGroundHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectGroundHitBlitz struct {
	Effect Optional[string] `json:"effect,omitempty"`
	Sound  Optional[string] `json:"sound,omitempty"`
}

type StrictShotEffectGroundHitEffects struct {
	Decal Optional[StrictShotEffectGroundHitEffectsDecal] `json:"decal,omitempty"`
	Pixie Optional[StrictShotEffectGroundHitEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictShotEffectGroundHitEffectsSound] `json:"sound,omitempty"`
}

type StrictShotEffectGroundHitEffectsDecal struct {
	BumpTexName Optional[XMLScalarList[string]] `json:"bumpTexName,omitempty"`
	GroupName   Optional[string]                `json:"groupName,omitempty"`
	RandomYaw   Optional[string]                `json:"randomYaw,omitempty"`
	Size        Optional[string]                `json:"size,omitempty"`
	StartKey    Optional[string]                `json:"startKey,omitempty"`
	TexName     Optional[XMLScalarList[string]] `json:"texName,omitempty"`
	Variation   Optional[string]                `json:"variation,omitempty"`
}

type StrictShotEffectGroundHitEffectsPixie struct {
	AlwaysUpdateModel Optional[string]                `json:"alwaysUpdateModel,omitempty"`
	File              Optional[XMLScalarList[string]] `json:"file,omitempty"`
	Force             Optional[string]                `json:"force,omitempty"`
	StartKey          Optional[string]                `json:"startKey,omitempty"`
}

type StrictShotEffectGroundHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectGroundHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectMetalHit struct {
	Blitz   Optional[StrictShotEffectMetalHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectMetalHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectMetalHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectMetalHitBlitz struct {
	Effect Optional[string] `json:"effect,omitempty"`
	Sound  Optional[string] `json:"sound,omitempty"`
}

type StrictShotEffectMetalHitEffects struct {
	Decal Optional[StrictShotEffectMetalHitEffectsDecal] `json:"decal,omitempty"`
	Pixie Optional[StrictShotEffectMetalHitEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictShotEffectMetalHitEffectsSound] `json:"sound,omitempty"`
}

type StrictShotEffectMetalHitEffectsDecal struct {
	GroupName Optional[string] `json:"groupName,omitempty"`
	RandomYaw Optional[string] `json:"randomYaw,omitempty"`
	Size      Optional[string] `json:"size,omitempty"`
	StartKey  Optional[string] `json:"startKey,omitempty"`
	TexName   Optional[string] `json:"texName,omitempty"`
	Variation Optional[string] `json:"variation,omitempty"`
}

type StrictShotEffectMetalHitEffectsPixie struct {
	File                  Optional[string] `json:"file,omitempty"`
	Force                 Optional[string] `json:"force,omitempty"`
	OrientBySurfaceNormal Optional[string] `json:"orientBySurfaceNormal,omitempty"`
	StartKey              Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectMetalHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectMetalHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectProjectile struct {
	Effects      Optional[StrictShotEffectProjectileEffects] `json:"effects,omitempty"`
	Model        Optional[string]                            `json:"model,omitempty"`
	ModelOwnShot Optional[string]                            `json:"modelOwnShot,omitempty"`
}

type StrictShotEffectProjectileEffects struct {
	Pixie Optional[StrictShotEffectProjectileEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictShotEffectProjectileEffectsSound] `json:"sound,omitempty"`
}

type StrictShotEffectProjectileEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	Position Optional[string] `json:"position,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectProjectileEffectsSound struct {
	EndKey   Optional[string] `json:"endKey,omitempty"`
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectSandHit struct {
	Blitz   Optional[StrictShotEffectSandHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectSandHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectSandHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectSandHitBlitz struct {
	Effect Optional[string] `json:"effect,omitempty"`
	Sound  Optional[string] `json:"sound,omitempty"`
}

type StrictShotEffectSandHitEffects struct {
	Decal Optional[StrictShotEffectSandHitEffectsDecal] `json:"decal,omitempty"`
	Pixie Optional[StrictShotEffectSandHitEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictShotEffectSandHitEffectsSound] `json:"sound,omitempty"`
}

type StrictShotEffectSandHitEffectsDecal struct {
	BumpTexName Optional[string] `json:"bumpTexName,omitempty"`
	GroupName   Optional[string] `json:"groupName,omitempty"`
	RandomYaw   Optional[string] `json:"randomYaw,omitempty"`
	Size        Optional[string] `json:"size,omitempty"`
	StartKey    Optional[string] `json:"startKey,omitempty"`
	TexName     Optional[string] `json:"texName,omitempty"`
	Variation   Optional[string] `json:"variation,omitempty"`
}

type StrictShotEffectSandHitEffectsPixie struct {
	AlwaysUpdateModel Optional[string] `json:"alwaysUpdateModel,omitempty"`
	File              Optional[string] `json:"file,omitempty"`
	Force             Optional[string] `json:"force,omitempty"`
	StartKey          Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectSandHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectSandHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectShallowWaterHit struct {
	Blitz   Optional[StrictShotEffectShallowWaterHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectShallowWaterHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectShallowWaterHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectShallowWaterHitBlitz struct {
	Effect Optional[string] `json:"effect,omitempty"`
	Sound  Optional[string] `json:"sound,omitempty"`
}

type StrictShotEffectShallowWaterHitEffects struct {
	Pixie Optional[StrictShotEffectShallowWaterHitEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictShotEffectShallowWaterHitEffectsSound] `json:"sound,omitempty"`
}

type StrictShotEffectShallowWaterHitEffectsPixie struct {
	File     Optional[string] `json:"file,omitempty"`
	Force    Optional[string] `json:"force,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectShallowWaterHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectShallowWaterHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectSnowHit struct {
	Blitz   Optional[StrictShotEffectSnowHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectSnowHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectSnowHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectSnowHitBlitz struct {
	Effect Optional[string] `json:"effect,omitempty"`
	Sound  Optional[string] `json:"sound,omitempty"`
}

type StrictShotEffectSnowHitEffects struct {
	Decal Optional[StrictShotEffectSnowHitEffectsDecal] `json:"decal,omitempty"`
	Pixie Optional[StrictShotEffectSnowHitEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictShotEffectSnowHitEffectsSound] `json:"sound,omitempty"`
}

type StrictShotEffectSnowHitEffectsDecal struct {
	BumpTexName Optional[string] `json:"bumpTexName,omitempty"`
	GroupName   Optional[string] `json:"groupName,omitempty"`
	RandomYaw   Optional[string] `json:"randomYaw,omitempty"`
	Size        Optional[string] `json:"size,omitempty"`
	StartKey    Optional[string] `json:"startKey,omitempty"`
	TexName     Optional[string] `json:"texName,omitempty"`
	Variation   Optional[string] `json:"variation,omitempty"`
}

type StrictShotEffectSnowHitEffectsPixie struct {
	AlwaysUpdateModel Optional[string] `json:"alwaysUpdateModel,omitempty"`
	File              Optional[string] `json:"file,omitempty"`
	Force             Optional[string] `json:"force,omitempty"`
	StartKey          Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectSnowHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectSnowHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectStoneHit struct {
	Blitz   Optional[StrictShotEffectStoneHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectStoneHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectStoneHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectStoneHitBlitz struct {
	Effect       Optional[string] `json:"effect,omitempty"`
	ObjectEffect Optional[string] `json:"objectEffect,omitempty"`
	Sound        Optional[string] `json:"sound,omitempty"`
}

type StrictShotEffectStoneHitEffects struct {
	Decal Optional[StrictShotEffectStoneHitEffectsDecal] `json:"decal,omitempty"`
	Pixie Optional[StrictShotEffectStoneHitEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictShotEffectStoneHitEffectsSound] `json:"sound,omitempty"`
}

type StrictShotEffectStoneHitEffectsDecal struct {
	BumpTexName Optional[string] `json:"bumpTexName,omitempty"`
	GroupName   Optional[string] `json:"groupName,omitempty"`
	RandomYaw   Optional[string] `json:"randomYaw,omitempty"`
	Size        Optional[string] `json:"size,omitempty"`
	StartKey    Optional[string] `json:"startKey,omitempty"`
	TexName     Optional[string] `json:"texName,omitempty"`
	Variation   Optional[string] `json:"variation,omitempty"`
}

type StrictShotEffectStoneHitEffectsPixie struct {
	File                  Optional[string] `json:"file,omitempty"`
	Force                 Optional[string] `json:"force,omitempty"`
	OrientBySurfaceNormal Optional[string] `json:"orientBySurfaceNormal,omitempty"`
	StartKey              Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectStoneHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectStoneHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictShotEffectTargetHitMarks struct {
	ArmorHit      Optional[string] `json:"armorHit,omitempty"`
	ArmorResisted Optional[string] `json:"armorResisted,omitempty"`
}

type StrictShotEffectWaterParams struct {
	RippleDiameter    Optional[string]             `json:"rippleDiameter,omitempty"`
	ShallowWaterDepth Optional[string]             `json:"shallowWaterDepth,omitempty"`
	ShowWaterEffect   Optional[[]Optional[string]] `json:"showWaterEffect,omitempty"`
}

type StrictShotEffectWoodHit struct {
	Blitz   Optional[StrictShotEffectWoodHitBlitz]   `json:"blitz,omitempty"`
	Effects Optional[StrictShotEffectWoodHitEffects] `json:"effects,omitempty"`
	Stages  Optional[StrictShotEffectWoodHitStages]  `json:"stages,omitempty"`
}

type StrictShotEffectWoodHitBlitz struct {
	Effect Optional[string] `json:"effect,omitempty"`
	Sound  Optional[string] `json:"sound,omitempty"`
}

type StrictShotEffectWoodHitEffects struct {
	Decal Optional[StrictShotEffectWoodHitEffectsDecal] `json:"decal,omitempty"`
	Pixie Optional[StrictShotEffectWoodHitEffectsPixie] `json:"pixie,omitempty"`
	Sound Optional[StrictShotEffectWoodHitEffectsSound] `json:"sound,omitempty"`
}

type StrictShotEffectWoodHitEffectsDecal struct {
	BumpTexName Optional[string] `json:"bumpTexName,omitempty"`
	GroupName   Optional[string] `json:"groupName,omitempty"`
	RandomYaw   Optional[string] `json:"randomYaw,omitempty"`
	Size        Optional[string] `json:"size,omitempty"`
	StartKey    Optional[string] `json:"startKey,omitempty"`
	TexName     Optional[string] `json:"texName,omitempty"`
	Variation   Optional[string] `json:"variation,omitempty"`
}

type StrictShotEffectWoodHitEffectsPixie struct {
	File                  Optional[string] `json:"file,omitempty"`
	Force                 Optional[string] `json:"force,omitempty"`
	OrientBySurfaceNormal Optional[string] `json:"orientBySurfaceNormal,omitempty"`
	StartKey              Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectWoodHitEffectsSound struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictShotEffectWoodHitStages struct {
	Impact Optional[string] `json:"impact,omitempty"`
}

type StrictVehicleEffect struct {
	Variant1 Optional[StrictVehicleEffectVariant1] `json:"variant1,omitempty"`
	Variant2 Optional[StrictVehicleEffectVariant2] `json:"variant2,omitempty"`
	Variant3 Optional[StrictVehicleEffectVariant3] `json:"variant3,omitempty"`
}

type StrictVehicleEffectVariant1 struct {
	Effects Optional[StrictVehicleEffectVariant1Effects] `json:"effects,omitempty"`
	Stages  Optional[StrictVehicleEffectVariant1Stages]  `json:"stages,omitempty"`
}

type StrictVehicleEffectVariant1Effects struct {
	Decal        Optional[StrictVehicleEffectVariant1EffectsDecal]                                                                     `json:"decal,omitempty"`
	FlashBang    Optional[StrictVehicleEffectVariant1EffectsFlashBang]                                                                 `json:"flashBang,omitempty"`
	Pixie        Optional[XMLObjectList[StrictVehicleEffectVariant1EffectsPixieObject, StrictVehicleEffectVariant1EffectsPixieObject]] `json:"pixie,omitempty"`
	ShockWave    Optional[StrictVehicleEffectVariant1EffectsShockWave]                                                                 `json:"shockWave,omitempty"`
	Sound        Optional[XMLObjectList[StrictVehicleEffectVariant1EffectsSoundObject, StrictVehicleEffectVariant1EffectsSoundObject]] `json:"sound,omitempty"`
	StopEmission Optional[StrictVehicleEffectVariant1EffectsStopEmission]                                                              `json:"stopEmission,omitempty"`
}

type StrictVehicleEffectVariant1EffectsDecal struct {
	GroupName Optional[string] `json:"groupName,omitempty"`
	RandomYaw Optional[string] `json:"randomYaw,omitempty"`
	Size      Optional[string] `json:"size,omitempty"`
	StartKey  Optional[string] `json:"startKey,omitempty"`
	TexName   Optional[string] `json:"texName,omitempty"`
}

type StrictVehicleEffectVariant1EffectsFlashBang struct {
	Stages   Optional[StrictVehicleEffectVariant1EffectsFlashBangStages] `json:"stages,omitempty"`
	StartKey Optional[string]                                            `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant1EffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictVehicleEffectVariant1EffectsFlashBangStagesStageEntry]] `json:"stage,omitempty"`
}

type StrictVehicleEffectVariant1EffectsFlashBangStagesStageEntry struct {
	Text  Optional[string] `json:"#text,omitempty"`
	Color Optional[string] `json:"color,omitempty"`
}

type StrictVehicleEffectVariant1EffectsPixieObject struct {
	AlwaysUpdateModel     Optional[string] `json:"alwaysUpdateModel,omitempty"`
	EndKey                Optional[string] `json:"endKey,omitempty"`
	File                  Optional[string] `json:"file,omitempty"`
	Force                 Optional[string] `json:"force,omitempty"`
	OrientBySurfaceNormal Optional[string] `json:"orientBySurfaceNormal,omitempty"`
	Position              Optional[string] `json:"position,omitempty"`
	StartKey              Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant1EffectsShockWave struct {
	Duration Optional[string] `json:"duration,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant1EffectsSoundObject struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant1EffectsStopEmission struct {
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant1Stages struct {
	Death         Optional[string] `json:"death,omitempty"`
	Explosion     Optional[string] `json:"explosion,omitempty"`
	Fire          Optional[string] `json:"fire,omitempty"`
	Impact        Optional[string] `json:"impact,omitempty"`
	NoEmission    Optional[string] `json:"noEmission,omitempty"`
	Static        Optional[string] `json:"static,omitempty"`
	StopDeath     Optional[string] `json:"stopDeath,omitempty"`
	StopExplosion Optional[string] `json:"stopExplosion,omitempty"`
}

type StrictVehicleEffectVariant2 struct {
	Effects Optional[StrictVehicleEffectVariant2Effects] `json:"effects,omitempty"`
	Stages  Optional[StrictVehicleEffectVariant2Stages]  `json:"stages,omitempty"`
}

type StrictVehicleEffectVariant2Effects struct {
	Decal        Optional[StrictVehicleEffectVariant2EffectsDecal]                                                                     `json:"decal,omitempty"`
	FlashBang    Optional[StrictVehicleEffectVariant2EffectsFlashBang]                                                                 `json:"flashBang,omitempty"`
	Pixie        Optional[XMLObjectList[StrictVehicleEffectVariant2EffectsPixieObject, StrictVehicleEffectVariant2EffectsPixieObject]] `json:"pixie,omitempty"`
	ShockWave    Optional[StrictVehicleEffectVariant2EffectsShockWave]                                                                 `json:"shockWave,omitempty"`
	Sound        Optional[XMLObjectList[StrictVehicleEffectVariant2EffectsSoundObject, StrictVehicleEffectVariant2EffectsSoundObject]] `json:"sound,omitempty"`
	StopEmission Optional[StrictVehicleEffectVariant2EffectsStopEmission]                                                              `json:"stopEmission,omitempty"`
}

type StrictVehicleEffectVariant2EffectsDecal struct {
	GroupName Optional[string] `json:"groupName,omitempty"`
	RandomYaw Optional[string] `json:"randomYaw,omitempty"`
	Size      Optional[string] `json:"size,omitempty"`
	StartKey  Optional[string] `json:"startKey,omitempty"`
	TexName   Optional[string] `json:"texName,omitempty"`
}

type StrictVehicleEffectVariant2EffectsFlashBang struct {
	Stages   Optional[StrictVehicleEffectVariant2EffectsFlashBangStages] `json:"stages,omitempty"`
	StartKey Optional[string]                                            `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant2EffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictVehicleEffectVariant2EffectsFlashBangStagesStageEntry]] `json:"stage,omitempty"`
}

type StrictVehicleEffectVariant2EffectsFlashBangStagesStageEntry struct {
	Text  Optional[string] `json:"#text,omitempty"`
	Color Optional[string] `json:"color,omitempty"`
}

type StrictVehicleEffectVariant2EffectsPixieObject struct {
	AlwaysUpdateModel Optional[string] `json:"alwaysUpdateModel,omitempty"`
	EndKey            Optional[string] `json:"endKey,omitempty"`
	File              Optional[string] `json:"file,omitempty"`
	Force             Optional[string] `json:"force,omitempty"`
	Position          Optional[string] `json:"position,omitempty"`
	StartKey          Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant2EffectsShockWave struct {
	Duration Optional[string] `json:"duration,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant2EffectsSoundObject struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant2EffectsStopEmission struct {
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant2Stages struct {
	Death         Optional[string] `json:"death,omitempty"`
	Explosion     Optional[string] `json:"explosion,omitempty"`
	Fire          Optional[string] `json:"fire,omitempty"`
	Impact        Optional[string] `json:"impact,omitempty"`
	NoEmission    Optional[string] `json:"noEmission,omitempty"`
	Static        Optional[string] `json:"static,omitempty"`
	StopDeath     Optional[string] `json:"stopDeath,omitempty"`
	StopExplosion Optional[string] `json:"stopExplosion,omitempty"`
}

type StrictVehicleEffectVariant3 struct {
	Effects Optional[StrictVehicleEffectVariant3Effects] `json:"effects,omitempty"`
	Stages  Optional[StrictVehicleEffectVariant3Stages]  `json:"stages,omitempty"`
}

type StrictVehicleEffectVariant3Effects struct {
	Decal     Optional[StrictVehicleEffectVariant3EffectsDecal]                  `json:"decal,omitempty"`
	FlashBang Optional[StrictVehicleEffectVariant3EffectsFlashBang]              `json:"flashBang,omitempty"`
	Pixie     Optional[[]Optional[StrictVehicleEffectVariant3EffectsPixieEntry]] `json:"pixie,omitempty"`
	ShockWave Optional[StrictVehicleEffectVariant3EffectsShockWave]              `json:"shockWave,omitempty"`
	Sound     Optional[[]Optional[StrictVehicleEffectVariant3EffectsSoundEntry]] `json:"sound,omitempty"`
}

type StrictVehicleEffectVariant3EffectsDecal struct {
	GroupName Optional[string] `json:"groupName,omitempty"`
	RandomYaw Optional[string] `json:"randomYaw,omitempty"`
	Size      Optional[string] `json:"size,omitempty"`
	StartKey  Optional[string] `json:"startKey,omitempty"`
	TexName   Optional[string] `json:"texName,omitempty"`
}

type StrictVehicleEffectVariant3EffectsFlashBang struct {
	Stages   Optional[StrictVehicleEffectVariant3EffectsFlashBangStages] `json:"stages,omitempty"`
	StartKey Optional[string]                                            `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant3EffectsFlashBangStages struct {
	Stage Optional[[]Optional[StrictVehicleEffectVariant3EffectsFlashBangStagesStageEntry]] `json:"stage,omitempty"`
}

type StrictVehicleEffectVariant3EffectsFlashBangStagesStageEntry struct {
	Text  Optional[string] `json:"#text,omitempty"`
	Color Optional[string] `json:"color,omitempty"`
}

type StrictVehicleEffectVariant3EffectsPixieEntry struct {
	AlwaysUpdateModel Optional[string] `json:"alwaysUpdateModel,omitempty"`
	EndKey            Optional[string] `json:"endKey,omitempty"`
	File              Optional[string] `json:"file,omitempty"`
	Force             Optional[string] `json:"force,omitempty"`
	Position          Optional[string] `json:"position,omitempty"`
	StartKey          Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant3EffectsShockWave struct {
	Duration Optional[string] `json:"duration,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant3EffectsSoundEntry struct {
	Name     Optional[string] `json:"name,omitempty"`
	StartKey Optional[string] `json:"startKey,omitempty"`
}

type StrictVehicleEffectVariant3Stages struct {
	Death     Optional[string] `json:"death,omitempty"`
	Static    Optional[string] `json:"static,omitempty"`
	StopDeath Optional[string] `json:"stopDeath,omitempty"`
}
