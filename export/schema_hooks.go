package export

import (
	"encoding/json"
	"strings"
)

// Schema hooks for wire shapes that are not plain structs: unions and objects
// that mix fixed fields with dictionary entries.

// decodeExtraEntry decodes the value for key into a dictionary stored beside
// fixed struct fields.
func decodeExtraEntry[T any](d *schemaDecoder, items *map[string]Optional[T], key string) error {
	d.markDynamicKey()
	var item Optional[T]
	kept, err := d.memberAny(&item)
	if kept {
		if *items == nil {
			*items = make(map[string]Optional[T])
		}
		(*items)[key] = item
	}
	return err
}

// Every key besides nextAvailableId is a device ID.
func (v *StrictEquipmentDevices) decodeExtraKey(d *schemaDecoder, key string) (bool, error) {
	return true, decodeExtraEntry(d, &v.Items, key)
}

// Every key besides icons is a shell ID.
func (v *StrictShellDocument) decodeExtraKey(d *schemaDecoder, key string) (bool, error) {
	return true, decodeExtraEntry(d, &v.Items, key)
}

// Detached turret parts are keyed turret_<name>.
func (v *StrictTankDetachedTurret) decodeExtraKey(d *schemaDecoder, key string) (bool, error) {
	if !strings.HasPrefix(key, "turret_") {
		return false, nil
	}
	return true, decodeExtraEntry(d, &v.Turrets, key)
}

// Some source files place name-keyed shot elements (_<name>) directly under
// a gun.
func (v *StrictTankConfigurationTurrets0EntryPropertiesGunsEntryProperties) decodeExtraKey(d *schemaDecoder, key string) (bool, error) {
	if !strings.HasPrefix(key, "_") {
		return false, nil
	}
	return true, decodeExtraEntry(d, &v.NamedShots, key)
}

// Settings documents without a schema stay raw in Other.
func (v *PartialSettings) decodeExtraKey(d *schemaDecoder, key string) (bool, error) {
	var raw json.RawMessage
	if err := d.decodeAny(&raw); err != nil {
		return true, err
	}
	if v.Other == nil {
		v.Other = make(map[string]json.RawMessage)
	}
	v.Other[key] = raw
	return true, nil
}

func (v *StrictMetadataPrice) decodeSchema(d *schemaDecoder) error {
	switch d.peek() {
	case '"':
		var x XMLInteger
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = StrictMetadataPrice{Scalar: &x}
	case '{':
		var x StrictMetadataPriceDetails
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = StrictMetadataPrice{Details: &x}
	default:
		return d.valueErr("metadata price must be a string or object, got %s", d.tokenName())
	}
	return nil
}

func (v *StrictTankGunMarkValue) decodeSchema(d *schemaDecoder) error {
	switch d.peek() {
	case '"':
		var x string
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = StrictTankGunMarkValue{Path: &x}
	case '{':
		var x map[string]Optional[StrictTankGunMark]
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = StrictTankGunMarkValue{Skins: x}
	default:
		return d.valueErr("gun mark must be a string or object, got %s", d.tokenName())
	}
	return nil
}

func (v *StrictProvisionDescrParamValue) decodeSchema(d *schemaDecoder) error {
	switch d.peek() {
	case '{':
		var x StrictProvisionDescrParam
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = StrictProvisionDescrParamValue{Single: &x}
	case '[':
		var x []Optional[StrictProvisionDescrParam]
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = StrictProvisionDescrParamValue{Repeated: x}
	default:
		return d.valueErr("description parameter must be an object or array, got %s", d.tokenName())
	}
	return nil
}

func (v *StrictProvisionScriptValue) decodeSchema(d *schemaDecoder) error {
	switch d.peek() {
	case '"':
		var x string
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = StrictProvisionScriptValue{Scalar: &x}
	case '{':
		var x StrictProvisionScript
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = StrictProvisionScriptValue{Object: &x}
	default:
		return d.valueErr("provision script must be a string or object, got %s", d.tokenName())
	}
	return nil
}

func (v *StrictVehicleEmptyItems) decodeSchema(d *schemaDecoder) error {
	if !d.consume('[') || !d.consume(']') {
		return d.valueErr("expected empty #items array")
	}
	*v = StrictVehicleEmptyItems{}
	return nil
}

// encoding/json entry points. Section types decode with the strict schema when
// passed to json.Unmarshal directly; nested values are decoded by the schema
// decoder, which does not call these methods.

func (v *StrictDataSnapshot) UnmarshalJSON(data []byte) error   { return decodeStrictJSON(data, v) }
func (v *PartialSnapshot) UnmarshalJSON(data []byte) error      { return decodeStrictJSON(data, v) }
func (v *StrictDataTank) UnmarshalJSON(data []byte) error       { return decodeStrictJSON(data, v) }
func (v *PartialSnapshotTank) UnmarshalJSON(data []byte) error  { return decodeStrictJSON(data, v) }
func (v *StrictTankMetadata) UnmarshalJSON(data []byte) error   { return decodeStrictJSON(data, v) }
func (v *StrictTankParameters) UnmarshalJSON(data []byte) error { return decodeStrictJSON(data, v) }
func (v *StrictTankConfiguration) UnmarshalJSON(data []byte) error {
	return decodeStrictJSON(data, v)
}
func (v *StrictLocalization) UnmarshalJSON(data []byte) error     { return decodeStrictJSON(data, v) }
func (v *StrictMaps) UnmarshalJSON(data []byte) error             { return decodeStrictJSON(data, v) }
func (v *StrictNationComponents) UnmarshalJSON(data []byte) error { return decodeStrictJSON(data, v) }
func (v *StrictEquipment) UnmarshalJSON(data []byte) error        { return decodeStrictJSON(data, v) }
func (v *StrictProvisions) UnmarshalJSON(data []byte) error       { return decodeStrictJSON(data, v) }
func (v *StrictSettings) UnmarshalJSON(data []byte) error         { return decodeStrictJSON(data, v) }
func (v *PartialSettings) UnmarshalJSON(data []byte) error        { return decodeStrictJSON(data, v) }
func (v *StrictVehicleSettings) UnmarshalJSON(data []byte) error  { return decodeStrictJSON(data, v) }
func (v *StrictHangarParameters) UnmarshalJSON(data []byte) error { return decodeStrictJSON(data, v) }
func (v *StrictCrewSkill) UnmarshalJSON(data []byte) error        { return decodeStrictJSON(data, v) }
func (v *StrictExportIssue) UnmarshalJSON(data []byte) error      { return decodeStrictJSON(data, v) }
