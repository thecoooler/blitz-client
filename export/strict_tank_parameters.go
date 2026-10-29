package export

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StrictTankParameters describes parameter YAML shapes observed in the current
// export. Numbers retain their exact JSON spelling; model-part, skin, effect and
// track identifiers are dynamic dictionary keys. Unknown fixed fields fail.
type StrictTankParameters struct {
	ArmorScale       Optional[map[string]Optional[StrictTankArmorScale]]       `json:"armorScale,omitzero"`
	Chassis          Optional[StrictTankParameterChassis]                      `json:"chassis,omitzero"`
	Collision        Optional[map[string]Optional[StrictTankCollision]]        `json:"collision,omitzero"`
	DetachedTurret   Optional[StrictTankDetachedTurret]                        `json:"detachedTurret,omitzero"`
	GunMarks         Optional[map[string]Optional[StrictTankGunMarkValue]]     `json:"gunMarks,omitzero"`
	GunRecoil        Optional[map[string]Optional[StrictTankGunRecoil]]        `json:"gunRecoil,omitzero"`
	HitMarks         Optional[map[string]Optional[StrictTankHitMark]]          `json:"hitMarks,omitzero"`
	MarkerPosition   Optional[StrictTankMarkerPosition]                        `json:"markerPosition,omitzero"`
	MaskSlice        Optional[map[string]Optional[StrictTankMaskSlice]]        `json:"maskSlice,omitzero"`
	Mechanics        Optional[[]Optional[StrictTankMechanic]]                  `json:"mechanics,omitzero"`
	ResourcesPath    Optional[StrictTankResourcesPath]                         `json:"resourcesPath,omitzero"`
	ShadowDecal      Optional[StrictTankShadowDecal]                           `json:"shadowDecal,omitzero"`
	Suspension       Optional[StrictTankSuspension]                            `json:"suspension,omitzero"`
	TextureTransform Optional[map[string]Optional[StrictTankTextureTransform]] `json:"textureTransform,omitzero"`
	TreadParameters  Optional[StrictTankTreadParameters]                       `json:"treadParameters,omitzero"`
}

type StrictTankArmorScale struct {
	ArmorOffset      Optional[[]StrictNumber] `json:"armorOffset,omitzero"`
	ArmorScale       Optional[[]StrictNumber] `json:"armorScale,omitzero"`
	PushNormalsScale Optional[bool]           `json:"pushNormalsScale,omitzero"`
}
type StrictTankParameterChassis struct {
	TextureScale Optional[StrictNumber] `json:"textureScale,omitzero"`
}
type StrictTankCollision struct {
	AverageThickness Optional[map[string]Optional[StrictNumber]] `json:"averageThickness,omitzero"`
	BBox             Optional[StrictTankBoundingBox]             `json:"bbox,omitzero"`
	Points           Optional[[]StrictNumber]                    `json:"points,omitzero"`
}
type StrictTankBoundingBox struct {
	Max Optional[[]StrictNumber] `json:"max,omitzero"`
	Min Optional[[]StrictNumber] `json:"min,omitzero"`
}
type StrictTankGunRecoil struct {
	Distance         Optional[StrictNumber] `json:"distance,omitzero"`
	DurationBackward Optional[StrictNumber] `json:"durationBackward,omitzero"`
	DurationForward  Optional[StrictNumber] `json:"durationForward,omitzero"`
	Enabled          Optional[bool]         `json:"enabled,omitzero"`
}
type StrictTankHitMark struct {
	PlacementRestrictedZoneLength Optional[StrictNumber] `json:"placementRestrictedZoneLength,omitzero"`
}
type StrictTankMarkerPosition struct {
	MarkerHeight Optional[StrictNumber] `json:"markerHeight,omitzero"`
}
type StrictTankMaskSlice struct {
	Enabled                 Optional[bool]           `json:"enabled,omitzero"`
	FrustumConeAngleDegrees Optional[StrictNumber]   `json:"frustumConeAngleDegrees,omitzero"`
	FrustumConeBaseRadius   Optional[StrictNumber]   `json:"frustumConeBaseRadius,omitzero"`
	FrustumConeSliceMode    Optional[StrictNumber]   `json:"frustumConeSliceMode,omitzero"`
	PlaneAxis               Optional[[]StrictNumber] `json:"planeAxis,omitzero"`
	PlaneNormal             Optional[[]StrictNumber] `json:"planeNormal,omitzero"`
	PlanePosition           Optional[[]StrictNumber] `json:"planePosition,omitzero"`
	PlaneSliceMode          Optional[StrictNumber]   `json:"planeSliceMode,omitzero"`
}
type StrictTankMechanic struct {
	Name Optional[string] `json:"name,omitzero"`
}
type StrictTankResourcesPath struct {
	BigIconPath          Optional[string]                                           `json:"bigIconPath,omitzero"`
	BlitzModelPath       Optional[string]                                           `json:"blitzModelPath,omitzero"`
	ChassisSoundNPC      Optional[string]                                           `json:"chassisSoundNPC,omitzero"`
	ChassisSoundPC       Optional[string]                                           `json:"chassisSoundPC,omitzero"`
	CollisionMesh        Optional[string]                                           `json:"collisionMesh,omitzero"`
	CustomEffects        Optional[map[string]Optional[map[string]Optional[string]]] `json:"customEffects,omitzero"`
	EnableArmorHighlight Optional[bool]                                             `json:"enableArmorHighlight,omitzero"`
	EnableReflection     Optional[bool]                                             `json:"enableReflection,omitzero"`
	EngineSoundNPC       Optional[string]                                           `json:"engineSoundNPC,omitzero"`
	EngineSoundPC        Optional[string]                                           `json:"engineSoundPC,omitzero"`
	ExclusionMask        Optional[string]                                           `json:"exclusionMask,omitzero"`
	FireEffectPathAlive  Optional[string]                                           `json:"fireEffectPathAlive,omitzero"`
	FireEffectPathDead   Optional[string]                                           `json:"fireEffectPathDead,omitzero"`
	ShopIconPath         Optional[string]                                           `json:"shopIconPath,omitzero"`
	SmallIconPath        Optional[string]                                           `json:"smallIconPath,omitzero"`
	SmokeEffectPath      Optional[string]                                           `json:"smokeEffectPath,omitzero"`
}
type StrictTankShadowDecal struct {
	BlendingType Optional[string]         `json:"blendingType,omitzero"`
	FogBehaviour Optional[string]         `json:"fogBehaviour,omitzero"`
	Offset       Optional[[]StrictNumber] `json:"offset,omitzero"`
	Scale        Optional[[]StrictNumber] `json:"scale,omitzero"`
	TexturePath  Optional[string]         `json:"texturePath,omitzero"`
}
type StrictTankSuspension struct {
	ChunkPrototypeExtending Optional[StrictNumber]                `json:"chunkPrototypeExtending,omitzero"`
	Enabled                 Optional[bool]                        `json:"enabled,omitzero"`
	LeftTrackChain          Optional[string]                      `json:"leftTrackChain,omitzero"`
	LeftTrackChains         Optional[map[string]Optional[string]] `json:"leftTrackChains,omitzero"`
	RightTrackChain         Optional[string]                      `json:"rightTrackChain,omitzero"`
	RightTrackChains        Optional[map[string]Optional[string]] `json:"rightTrackChains,omitzero"`
	TrackBendingInfo        Optional[StrictTankTrackBendingInfo]  `json:"trackBendingInfo,omitzero"`
	TrackLayingInfo         Optional[StrictTankTrackLayingInfo]   `json:"trackLayingInfo,omitzero"`
	Wheels                  Optional[string]                      `json:"wheels,omitzero"`
	WheelsReactionSpeed     Optional[StrictNumber]                `json:"wheelsReactionSpeed,omitzero"`
}
type StrictTankTrackBendingInfo struct {
	BackFactor      Optional[StrictNumber] `json:"backFactor,omitzero"`
	FrontDriveWheel Optional[bool]         `json:"frontDriveWheel,omitzero"`
	FrontFactor     Optional[StrictNumber] `json:"frontFactor,omitzero"`
	LengthPower     Optional[StrictNumber] `json:"lengthPower,omitzero"`
	Speed           Optional[StrictNumber] `json:"speed,omitzero"`
	UpperFactor     Optional[StrictNumber] `json:"upperFactor,omitzero"`
	UpperMin        Optional[StrictNumber] `json:"upperMin,omitzero"`
}
type StrictTankTrackLayingInfo struct {
	BendingFactor   Optional[StrictNumber] `json:"bendingFactor,omitzero"`
	LengthPower     Optional[StrictNumber] `json:"lengthPower,omitzero"`
	PointCountPower Optional[StrictNumber] `json:"pointCountPower,omitzero"`
	PressurePower   Optional[StrictNumber] `json:"pressurePower,omitzero"`
	PrimaryPower    Optional[StrictNumber] `json:"primaryPower,omitzero"`
}
type StrictTankTextureTransform struct {
	Enabled  Optional[bool]           `json:"enabled,omitzero"`
	Offset   Optional[[]StrictNumber] `json:"offset,omitzero"`
	Ratio    Optional[StrictNumber]   `json:"ratio,omitzero"`
	Rotation Optional[StrictNumber]   `json:"rotation,omitzero"`
	Scale    Optional[[]StrictNumber] `json:"scale,omitzero"`
	Shear    Optional[[]StrictNumber] `json:"shear,omitzero"`
}
type StrictTankTreadParameters struct {
	ForceManualSpace Optional[bool]         `json:"forceManualSpace,omitzero"`
	ForceManualWidth Optional[bool]         `json:"forceManualWidth,omitzero"`
	ManualSpace      Optional[StrictNumber] `json:"manualSpace,omitzero"`
	ManualWidth      Optional[StrictNumber] `json:"manualWidth,omitzero"`
	TexturePath      Optional[string]       `json:"texturePath,omitzero"`
}
type StrictTankGunMark struct {
	Diameter       Optional[StrictNumber]   `json:"diameter,omitzero"`
	GradeOnePath   Optional[string]         `json:"gradeOnePath,omitzero"`
	GradeThreePath Optional[string]         `json:"gradeThreePath,omitzero"`
	GradeTwoPath   Optional[string]         `json:"gradeTwoPath,omitzero"`
	Offset         Optional[[]StrictNumber] `json:"offset,omitzero"`
	Rotation       Optional[StrictNumber]   `json:"rotation,omitzero"`
	Scale          Optional[[]StrictNumber] `json:"scale,omitzero"`
}

// StrictTankGunMarkValue represents either a path string or a skin dictionary.
type StrictTankGunMarkValue struct {
	Path  *string
	Skins map[string]Optional[StrictTankGunMark]
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
	CylinderPos     Optional[[]StrictNumber]                          `json:"cylinderPos,omitzero"`
	CylinderXYScale Optional[StrictNumber]                            `json:"cylinderXYScale,omitzero"`
	CylinderZScale  Optional[StrictNumber]                            `json:"cylinderZScale,omitzero"`
	Turrets         map[string]Optional[StrictTankDetachedTurretPart] `json:"-"`
}
type StrictTankDetachedTurretPart struct {
	ClippingPlaneZPos  Optional[StrictNumber] `json:"clippingPlaneZPos,omitzero"`
	ShouldClipGun      Optional[bool]         `json:"shouldClipGun,omitzero"`
	TextureScaleFactor Optional[StrictNumber] `json:"textureScaleFactor,omitzero"`
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
