package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// StrictTankParameters describes parameter YAML shapes observed in the current
// export. Numbers retain their exact JSON spelling; model-part, skin, effect and
// track identifiers are dynamic dictionary keys. Unknown fixed fields fail.
type StrictTankParameters struct {
	ArmorScale       Optional[map[string]Optional[StrictTankArmorScale]]       `json:"armorScale,omitempty"`
	Chassis          Optional[StrictTankParameterChassis]                      `json:"chassis,omitempty"`
	Collision        Optional[map[string]Optional[StrictTankCollision]]        `json:"collision,omitempty"`
	DetachedTurret   Optional[StrictTankDetachedTurret]                        `json:"detachedTurret,omitempty"`
	GunMarks         Optional[map[string]Optional[StrictTankGunMarkValue]]     `json:"gunMarks,omitempty"`
	GunRecoil        Optional[map[string]Optional[StrictTankGunRecoil]]        `json:"gunRecoil,omitempty"`
	HitMarks         Optional[map[string]Optional[StrictTankHitMark]]          `json:"hitMarks,omitempty"`
	MarkerPosition   Optional[StrictTankMarkerPosition]                        `json:"markerPosition,omitempty"`
	MaskSlice        Optional[map[string]Optional[StrictTankMaskSlice]]        `json:"maskSlice,omitempty"`
	Mechanics        Optional[[]Optional[StrictTankMechanic]]                  `json:"mechanics,omitempty"`
	ResourcesPath    Optional[StrictTankResourcesPath]                         `json:"resourcesPath,omitempty"`
	ShadowDecal      Optional[StrictTankShadowDecal]                           `json:"shadowDecal,omitempty"`
	Suspension       Optional[StrictTankSuspension]                            `json:"suspension,omitempty"`
	TextureTransform Optional[map[string]Optional[StrictTankTextureTransform]] `json:"textureTransform,omitempty"`
	TreadParameters  Optional[StrictTankTreadParameters]                       `json:"treadParameters,omitempty"`
}

func (p *StrictTankParameters) UnmarshalJSON(data []byte) error {
	if d := bytes.TrimSpace(data); len(d) == 0 || d[0] != '{' {
		return fmt.Errorf("tank parameters must be an object")
	}
	type fields StrictTankParameters
	var v fields
	if err := decodeStrictJSON(data, &v); err != nil {
		return err
	}
	*p = StrictTankParameters(v)
	return nil
}

type StrictTankArmorScale struct {
	ArmorOffset      Optional[[]StrictNumber] `json:"armorOffset,omitempty"`
	ArmorScale       Optional[[]StrictNumber] `json:"armorScale,omitempty"`
	PushNormalsScale Optional[bool]           `json:"pushNormalsScale,omitempty"`
}
type StrictTankParameterChassis struct {
	TextureScale Optional[StrictNumber] `json:"textureScale,omitempty"`
}
type StrictTankCollision struct {
	AverageThickness Optional[map[string]Optional[StrictNumber]] `json:"averageThickness,omitempty"`
	BBox             Optional[StrictTankBoundingBox]             `json:"bbox,omitempty"`
	Points           Optional[[]StrictNumber]                    `json:"points,omitempty"`
}
type StrictTankBoundingBox struct {
	Max Optional[[]StrictNumber] `json:"max,omitempty"`
	Min Optional[[]StrictNumber] `json:"min,omitempty"`
}
type StrictTankGunRecoil struct {
	Distance         Optional[StrictNumber] `json:"distance,omitempty"`
	DurationBackward Optional[StrictNumber] `json:"durationBackward,omitempty"`
	DurationForward  Optional[StrictNumber] `json:"durationForward,omitempty"`
	Enabled          Optional[bool]         `json:"enabled,omitempty"`
}
type StrictTankHitMark struct {
	PlacementRestrictedZoneLength Optional[StrictNumber] `json:"placementRestrictedZoneLength,omitempty"`
}
type StrictTankMarkerPosition struct {
	MarkerHeight Optional[StrictNumber] `json:"markerHeight,omitempty"`
}
type StrictTankMaskSlice struct {
	Enabled                 Optional[bool]           `json:"enabled,omitempty"`
	FrustumConeAngleDegrees Optional[StrictNumber]   `json:"frustumConeAngleDegrees,omitempty"`
	FrustumConeBaseRadius   Optional[StrictNumber]   `json:"frustumConeBaseRadius,omitempty"`
	FrustumConeSliceMode    Optional[StrictNumber]   `json:"frustumConeSliceMode,omitempty"`
	PlaneAxis               Optional[[]StrictNumber] `json:"planeAxis,omitempty"`
	PlaneNormal             Optional[[]StrictNumber] `json:"planeNormal,omitempty"`
	PlanePosition           Optional[[]StrictNumber] `json:"planePosition,omitempty"`
	PlaneSliceMode          Optional[StrictNumber]   `json:"planeSliceMode,omitempty"`
}
type StrictTankMechanic struct {
	Name Optional[string] `json:"name,omitempty"`
}
type StrictTankResourcesPath struct {
	BigIconPath          Optional[string]                                           `json:"bigIconPath,omitempty"`
	BlitzModelPath       Optional[string]                                           `json:"blitzModelPath,omitempty"`
	ChassisSoundNPC      Optional[string]                                           `json:"chassisSoundNPC,omitempty"`
	ChassisSoundPC       Optional[string]                                           `json:"chassisSoundPC,omitempty"`
	CollisionMesh        Optional[string]                                           `json:"collisionMesh,omitempty"`
	CustomEffects        Optional[map[string]Optional[map[string]Optional[string]]] `json:"customEffects,omitempty"`
	EnableArmorHighlight Optional[bool]                                             `json:"enableArmorHighlight,omitempty"`
	EnableReflection     Optional[bool]                                             `json:"enableReflection,omitempty"`
	EngineSoundNPC       Optional[string]                                           `json:"engineSoundNPC,omitempty"`
	EngineSoundPC        Optional[string]                                           `json:"engineSoundPC,omitempty"`
	ExclusionMask        Optional[string]                                           `json:"exclusionMask,omitempty"`
	FireEffectPathAlive  Optional[string]                                           `json:"fireEffectPathAlive,omitempty"`
	FireEffectPathDead   Optional[string]                                           `json:"fireEffectPathDead,omitempty"`
	ShopIconPath         Optional[string]                                           `json:"shopIconPath,omitempty"`
	SmallIconPath        Optional[string]                                           `json:"smallIconPath,omitempty"`
	SmokeEffectPath      Optional[string]                                           `json:"smokeEffectPath,omitempty"`
}
type StrictTankShadowDecal struct {
	BlendingType Optional[string]         `json:"blendingType,omitempty"`
	FogBehaviour Optional[string]         `json:"fogBehaviour,omitempty"`
	Offset       Optional[[]StrictNumber] `json:"offset,omitempty"`
	Scale        Optional[[]StrictNumber] `json:"scale,omitempty"`
	TexturePath  Optional[string]         `json:"texturePath,omitempty"`
}
type StrictTankSuspension struct {
	ChunkPrototypeExtending Optional[StrictNumber]                `json:"chunkPrototypeExtending,omitempty"`
	Enabled                 Optional[bool]                        `json:"enabled,omitempty"`
	LeftTrackChain          Optional[string]                      `json:"leftTrackChain,omitempty"`
	LeftTrackChains         Optional[map[string]Optional[string]] `json:"leftTrackChains,omitempty"`
	RightTrackChain         Optional[string]                      `json:"rightTrackChain,omitempty"`
	RightTrackChains        Optional[map[string]Optional[string]] `json:"rightTrackChains,omitempty"`
	TrackBendingInfo        Optional[StrictTankTrackBendingInfo]  `json:"trackBendingInfo,omitempty"`
	TrackLayingInfo         Optional[StrictTankTrackLayingInfo]   `json:"trackLayingInfo,omitempty"`
	Wheels                  Optional[string]                      `json:"wheels,omitempty"`
	WheelsReactionSpeed     Optional[StrictNumber]                `json:"wheelsReactionSpeed,omitempty"`
}
type StrictTankTrackBendingInfo struct {
	BackFactor      Optional[StrictNumber] `json:"backFactor,omitempty"`
	FrontDriveWheel Optional[bool]         `json:"frontDriveWheel,omitempty"`
	FrontFactor     Optional[StrictNumber] `json:"frontFactor,omitempty"`
	LengthPower     Optional[StrictNumber] `json:"lengthPower,omitempty"`
	Speed           Optional[StrictNumber] `json:"speed,omitempty"`
	UpperFactor     Optional[StrictNumber] `json:"upperFactor,omitempty"`
	UpperMin        Optional[StrictNumber] `json:"upperMin,omitempty"`
}
type StrictTankTrackLayingInfo struct {
	BendingFactor   Optional[StrictNumber] `json:"bendingFactor,omitempty"`
	LengthPower     Optional[StrictNumber] `json:"lengthPower,omitempty"`
	PointCountPower Optional[StrictNumber] `json:"pointCountPower,omitempty"`
	PressurePower   Optional[StrictNumber] `json:"pressurePower,omitempty"`
	PrimaryPower    Optional[StrictNumber] `json:"primaryPower,omitempty"`
}
type StrictTankTextureTransform struct {
	Enabled  Optional[bool]           `json:"enabled,omitempty"`
	Offset   Optional[[]StrictNumber] `json:"offset,omitempty"`
	Ratio    Optional[StrictNumber]   `json:"ratio,omitempty"`
	Rotation Optional[StrictNumber]   `json:"rotation,omitempty"`
	Scale    Optional[[]StrictNumber] `json:"scale,omitempty"`
	Shear    Optional[[]StrictNumber] `json:"shear,omitempty"`
}
type StrictTankTreadParameters struct {
	ForceManualSpace Optional[bool]         `json:"forceManualSpace,omitempty"`
	ForceManualWidth Optional[bool]         `json:"forceManualWidth,omitempty"`
	ManualSpace      Optional[StrictNumber] `json:"manualSpace,omitempty"`
	ManualWidth      Optional[StrictNumber] `json:"manualWidth,omitempty"`
	TexturePath      Optional[string]       `json:"texturePath,omitempty"`
}
type StrictTankGunMark struct {
	Diameter       Optional[StrictNumber]   `json:"diameter,omitempty"`
	GradeOnePath   Optional[string]         `json:"gradeOnePath,omitempty"`
	GradeThreePath Optional[string]         `json:"gradeThreePath,omitempty"`
	GradeTwoPath   Optional[string]         `json:"gradeTwoPath,omitempty"`
	Offset         Optional[[]StrictNumber] `json:"offset,omitempty"`
	Rotation       Optional[StrictNumber]   `json:"rotation,omitempty"`
	Scale          Optional[[]StrictNumber] `json:"scale,omitempty"`
}

// StrictTankGunMarkValue represents either a path string or a skin dictionary.
type StrictTankGunMarkValue struct {
	Path  *string
	Skins map[string]Optional[StrictTankGunMark]
}

func (v *StrictTankGunMarkValue) UnmarshalJSON(data []byte) error {
	d := bytes.TrimSpace(data)
	if len(d) == 0 {
		return fmt.Errorf("empty gun mark")
	}
	switch d[0] {
	case '"':
		var p string
		if err := decodeStrictJSON(d, &p); err != nil {
			return err
		}
		*v = StrictTankGunMarkValue{Path: &p}
	case '{':
		var m map[string]Optional[StrictTankGunMark]
		if err := decodeStrictJSON(d, &m); err != nil {
			return err
		}
		*v = StrictTankGunMarkValue{Skins: m}
	default:
		return fmt.Errorf("gun mark must be a string or object")
	}
	return nil
}
func (v StrictTankGunMarkValue) MarshalJSON() ([]byte, error) {
	if (v.Path == nil) == (v.Skins == nil) {
		return nil, fmt.Errorf("gun mark must have exactly one variant")
	}
	if v.Path != nil {
		return json.Marshal(*v.Path)
	}
	return json.Marshal(v.Skins)
}

// StrictTankDetachedTurret has fixed cylinder fields and dynamic turret names.
type StrictTankDetachedTurret struct {
	CylinderPos     Optional[[]StrictNumber]
	CylinderXYScale Optional[StrictNumber]
	CylinderZScale  Optional[StrictNumber]
	Turrets         map[string]Optional[StrictTankDetachedTurretPart]
}
type StrictTankDetachedTurretPart struct {
	ClippingPlaneZPos  Optional[StrictNumber] `json:"clippingPlaneZPos,omitempty"`
	ShouldClipGun      Optional[bool]         `json:"shouldClipGun,omitempty"`
	TextureScaleFactor Optional[StrictNumber] `json:"textureScaleFactor,omitempty"`
}

func (v *StrictTankDetachedTurret) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := decodeStrictJSON(data, &raw); err != nil {
		return err
	}
	if raw == nil {
		return fmt.Errorf("detached turret must be an object")
	}
	var result StrictTankDetachedTurret
	for key, value := range raw {
		var err error
		switch key {
		case "cylinderPos":
			err = decodeStrictJSON(value, &result.CylinderPos)
		case "cylinderXYScale":
			err = decodeStrictJSON(value, &result.CylinderXYScale)
		case "cylinderZScale":
			err = decodeStrictJSON(value, &result.CylinderZScale)
		default:
			if !strings.HasPrefix(key, "turret_") {
				return fmt.Errorf("unknown detached turret field %q", key)
			}
			var item Optional[StrictTankDetachedTurretPart]
			err = decodeStrictJSON(value, &item)
			if result.Turrets == nil {
				result.Turrets = make(map[string]Optional[StrictTankDetachedTurretPart])
			}
			result.Turrets[key] = item
		}
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	*v = result
	return nil
}
func (v StrictTankDetachedTurret) MarshalJSON() ([]byte, error) {
	out := make(map[string]json.RawMessage)
	put := func(key string, present bool, value interface{}) error {
		if !present {
			return nil
		}
		data, err := json.Marshal(value)
		if err == nil {
			out[key] = data
		}
		return err
	}
	if err := put("cylinderPos", v.CylinderPos.Present(), v.CylinderPos); err != nil {
		return nil, err
	}
	if err := put("cylinderXYScale", v.CylinderXYScale.Present(), v.CylinderXYScale); err != nil {
		return nil, err
	}
	if err := put("cylinderZScale", v.CylinderZScale.Present(), v.CylinderZScale); err != nil {
		return nil, err
	}
	for key, item := range v.Turrets {
		if !strings.HasPrefix(key, "turret_") || !item.Present() {
			return nil, fmt.Errorf("invalid turret key or value %q", key)
		}
		if err := put(key, true, item); err != nil {
			return nil, err
		}
	}
	return json.Marshal(out)
}
