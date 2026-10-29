package export

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StrictCrewSkill describes the crew skill records observed in the current
// client export. Parameters retain their source names and exact JSON numbers.
type StrictCrewSkill struct {
	ID          string                     `json:"id"`
	UserString  string                     `json:"userString"`
	Description string                     `json:"description"`
	Tip         string                     `json:"tip"`
	Features    string                     `json:"features"`
	TankClass   string                     `json:"tankClass"`
	Type        string                     `json:"type"`
	Parameters  *StrictCrewSkillParameters `json:"parameters"`
	Levels      []StrictCrewSkillLevel     `json:"levels"`
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
	ActivationChancePerLevel                Optional[StrictNumber] `json:"activationChancePerLevel,omitempty"`
	AimingFactorPerLevel                    Optional[StrictNumber] `json:"aimingFactorPerLevel,omitempty"`
	ArmourPiercingFactor                    Optional[StrictNumber] `json:"armourPiercingFactor,omitempty"`
	ArmourPiercingPowerRandomization        Optional[StrictNumber] `json:"armourPiercingPowerRandomization,omitempty"`
	CamouflageFactorPerLevel                Optional[StrictNumber] `json:"camouflageFactorPerLevel,omitempty"`
	CaptureSpeedBonusForAllyPresence        Optional[StrictNumber] `json:"captureSpeedBonusForAllyPresence,omitempty"`
	CaptureSpeedBonusForSkilledAllyPresence Optional[StrictNumber] `json:"captureSpeedBonusForSkilledAllyPresence,omitempty"`
	CaptureSpeedBonusPerAlly                Optional[StrictNumber] `json:"captureSpeedBonusPerAlly,omitempty"`
	CaptureSpeedBonusPerSkilledAlly         Optional[StrictNumber] `json:"captureSpeedBonusPerSkilledAlly,omitempty"`
	CaptureSpeedBonusPerSkilledAllyLevel    Optional[StrictNumber] `json:"captureSpeedBonusPerSkilledAllyLevel,omitempty"`
	CaptureSpeedFactorPerLevel              Optional[StrictNumber] `json:"captureSpeedFactorPerLevel,omitempty"`
	ChancePerLevel                          Optional[StrictNumber] `json:"chancePerLevel,omitempty"`
	DeviceChanceToHitBoost                  Optional[StrictNumber] `json:"deviceChanceToHitBoost,omitempty"`
	EnemyCount                              Optional[StrictNumber] `json:"enemyCount,omitempty"`
	EnemyInRadius                           Optional[StrictNumber] `json:"enemyInRadius,omitempty"`
	EnginePowerFactorPerLevel               Optional[StrictNumber] `json:"enginePowerFactorPerLevel,omitempty"`
	GunReloadTimeFactorPerLevel             Optional[StrictNumber] `json:"gunReloadTimeFactorPerLevel,omitempty"`
	HealthBurnPerSecLossFactorPerLevel      Optional[StrictNumber] `json:"healthBurnPerSecLossFactorPerLevel,omitempty"`
	MinLoggedStateValue                     Optional[StrictNumber] `json:"minLoggedStateValue,omitempty"`
	PenaltyFactorPerLevel                   Optional[StrictNumber] `json:"penaltyFactorPerLevel,omitempty"`
	RangedHitDamageCoefficient              Optional[StrictNumber] `json:"rangedHitDamageCoefficient,omitempty"`
	RangingShotsCount                       Optional[StrictNumber] `json:"rangingShotsCount,omitempty"`
	ReloadFactorPerLevel                    Optional[StrictNumber] `json:"reloadFactorPerLevel,omitempty"`
	RepairSpeedFactorPerLevel               Optional[StrictNumber] `json:"repairSpeedFactorPerLevel,omitempty"`
	RotationSpeedFactorPerLevel             Optional[StrictNumber] `json:"rotationSpeedFactorPerLevel,omitempty"`
	ShotDispersionFactor                    Optional[StrictNumber] `json:"shotDispersionFactor,omitempty"`
	ShotDispersionFactorPerLevel            Optional[StrictNumber] `json:"shotDispersionFactorPerLevel,omitempty"`
	ShotInBurstDispersionFactorPerLevel     Optional[StrictNumber] `json:"shotInBurstDispersionFactorPerLevel,omitempty"`
	SkillStateModifierWithAllies            Optional[StrictNumber] `json:"skillStateModifierWithAllies,omitempty"`
	SkillStateModifierWithSkilledAllies     Optional[StrictNumber] `json:"skillStateModifierWithSkilledAllies,omitempty"`
	ThresholdEnable                         Optional[StrictNumber] `json:"thresholdEnable,omitempty"`
	TurretRotationSpeedFactorPerLevel       Optional[StrictNumber] `json:"turretRotationSpeedFactorPerLevel,omitempty"`
	VehicleHealthFraction                   Optional[StrictNumber] `json:"vehicleHealthFraction,omitempty"`
	VisionRadiusFactorPerLevel              Optional[StrictNumber] `json:"visionRadiusFactorPerLevel,omitempty"`
	XpBonusFactorPerLevel                   Optional[StrictNumber] `json:"xpBonusFactorPerLevel,omitempty"`
}

func (s *StrictCrewSkill) UnmarshalJSON(data []byte) error {
	type skill StrictCrewSkill
	var v skill
	if err := decodeStrictJSON(data, &v); err != nil {
		return err
	}
	if err := requireFields(data, "id", "userString", "description", "tip", "features", "tankClass", "type", "parameters", "levels"); err != nil {
		return err
	}
	*s = StrictCrewSkill(v)
	return nil
}

func (s *StrictCrewSkillLevel) UnmarshalJSON(data []byte) error {
	type level StrictCrewSkillLevel
	var v level
	if err := decodeStrictJSON(data, &v); err != nil {
		return err
	}
	if err := requireFields(data, "level", "xp", "parameters"); err != nil {
		return err
	}
	*s = StrictCrewSkillLevel(v)
	return nil
}

// requireFields enforces exact spelling and presence in structs with required
// fields. Only parameters and levels can be null in these records.
func requireFields(data []byte, names ...string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("expected object")
	}
	for _, name := range names {
		value, ok := fields[name]
		if !ok {
			return fmt.Errorf("missing required field %q", name)
		}
		if name != "levels" && name != "parameters" && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("field %q cannot be null", name)
		}
	}
	if len(fields) != len(names) {
		return fmt.Errorf("unexpected field in object")
	}
	return nil
}
