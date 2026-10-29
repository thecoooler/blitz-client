package export

// StrictCrewSkill describes the crew skill records observed in the current
// client export. Parameters retain their source names and exact JSON numbers.
type StrictCrewSkill struct {
	ID          string                     `json:"id"`
	UserString  XMLLocalizationKey         `json:"userString"`
	Description XMLLocalizationKey         `json:"description"`
	Tip         string                     `json:"tip"`
	Features    string                     `json:"features"`
	TankClass   string                     `json:"tankClass"`
	Type        string                     `json:"type"`
	Parameters  *StrictCrewSkillParameters `json:"parameters"`
	Levels      []StrictCrewSkillLevel     `json:"levels" schema:"nullable"`
}

// StrictCrewSkillLevel contains the exporter's individual XP cost and
// coefficients; the coefficients are not final vehicle multipliers.
type StrictCrewSkillLevel struct {
	Level      int64                      `json:"level"`
	XP         int64                      `json:"xp"`
	Parameters *StrictCrewSkillParameters `json:"parameters"`
}

// StrictCrewSkillParameters describes the named numeric fields observed in
// the current export. Optional fields retain absence and an explicit null.
type StrictCrewSkillParameters struct {
	ActivationChancePerLevel                Optional[StrictNumber] `json:"activationChancePerLevel,omitzero"`
	AimingFactorPerLevel                    Optional[StrictNumber] `json:"aimingFactorPerLevel,omitzero"`
	ArmourPiercingFactor                    Optional[StrictNumber] `json:"armourPiercingFactor,omitzero"`
	ArmourPiercingPowerRandomization        Optional[StrictNumber] `json:"armourPiercingPowerRandomization,omitzero"`
	CamouflageFactorPerLevel                Optional[StrictNumber] `json:"camouflageFactorPerLevel,omitzero"`
	CaptureSpeedBonusForAllyPresence        Optional[StrictNumber] `json:"captureSpeedBonusForAllyPresence,omitzero"`
	CaptureSpeedBonusForSkilledAllyPresence Optional[StrictNumber] `json:"captureSpeedBonusForSkilledAllyPresence,omitzero"`
	CaptureSpeedBonusPerAlly                Optional[StrictNumber] `json:"captureSpeedBonusPerAlly,omitzero"`
	CaptureSpeedBonusPerSkilledAlly         Optional[StrictNumber] `json:"captureSpeedBonusPerSkilledAlly,omitzero"`
	CaptureSpeedBonusPerSkilledAllyLevel    Optional[StrictNumber] `json:"captureSpeedBonusPerSkilledAllyLevel,omitzero"`
	CaptureSpeedFactorPerLevel              Optional[StrictNumber] `json:"captureSpeedFactorPerLevel,omitzero"`
	ChancePerLevel                          Optional[StrictNumber] `json:"chancePerLevel,omitzero"`
	DeviceChanceToHitBoost                  Optional[StrictNumber] `json:"deviceChanceToHitBoost,omitzero"`
	EnemyCount                              Optional[StrictNumber] `json:"enemyCount,omitzero"`
	EnemyInRadius                           Optional[StrictNumber] `json:"enemyInRadius,omitzero"`
	EnginePowerFactorPerLevel               Optional[StrictNumber] `json:"enginePowerFactorPerLevel,omitzero"`
	GunReloadTimeFactorPerLevel             Optional[StrictNumber] `json:"gunReloadTimeFactorPerLevel,omitzero"`
	HealthBurnPerSecLossFactorPerLevel      Optional[StrictNumber] `json:"healthBurnPerSecLossFactorPerLevel,omitzero"`
	MinLoggedStateValue                     Optional[StrictNumber] `json:"minLoggedStateValue,omitzero"`
	PenaltyFactorPerLevel                   Optional[StrictNumber] `json:"penaltyFactorPerLevel,omitzero"`
	RangedHitDamageCoefficient              Optional[StrictNumber] `json:"rangedHitDamageCoefficient,omitzero"`
	RangingShotsCount                       Optional[StrictNumber] `json:"rangingShotsCount,omitzero"`
	ReloadFactorPerLevel                    Optional[StrictNumber] `json:"reloadFactorPerLevel,omitzero"`
	RepairSpeedFactorPerLevel               Optional[StrictNumber] `json:"repairSpeedFactorPerLevel,omitzero"`
	RotationSpeedFactorPerLevel             Optional[StrictNumber] `json:"rotationSpeedFactorPerLevel,omitzero"`
	ShotDispersionFactor                    Optional[StrictNumber] `json:"shotDispersionFactor,omitzero"`
	ShotDispersionFactorPerLevel            Optional[StrictNumber] `json:"shotDispersionFactorPerLevel,omitzero"`
	ShotInBurstDispersionFactorPerLevel     Optional[StrictNumber] `json:"shotInBurstDispersionFactorPerLevel,omitzero"`
	SkillStateModifierWithAllies            Optional[StrictNumber] `json:"skillStateModifierWithAllies,omitzero"`
	SkillStateModifierWithSkilledAllies     Optional[StrictNumber] `json:"skillStateModifierWithSkilledAllies,omitzero"`
	ThresholdEnable                         Optional[StrictNumber] `json:"thresholdEnable,omitzero"`
	TurretRotationSpeedFactorPerLevel       Optional[StrictNumber] `json:"turretRotationSpeedFactorPerLevel,omitzero"`
	VehicleHealthFraction                   Optional[StrictNumber] `json:"vehicleHealthFraction,omitzero"`
	VisionRadiusFactorPerLevel              Optional[StrictNumber] `json:"visionRadiusFactorPerLevel,omitzero"`
	XpBonusFactorPerLevel                   Optional[StrictNumber] `json:"xpBonusFactorPerLevel,omitzero"`
}
