package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// decodeStrictJSON rejects duplicate object keys, unknown struct fields, and
// trailing JSON values. UseNumber preserves the spelling of numeric values
// decoded into interfaces. Types implementing json.Unmarshaler must enforce
// their own field and value constraints.
func decodeStrictJSON(data []byte, target any) error {
	if err := checkJSONKeys(data, reflect.TypeOf(target)); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

// checkJSONKeys walks the original token stream before unmarshalling, since
// encoding/json otherwise silently accepts duplicate keys (including in maps
// and values handled by custom unmarshalers).
func checkJSONKeys(data []byte, target reflect.Type) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := checkJSONValue(decoder, "$", target); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func checkJSONValue(decoder *json.Decoder, path string, target reflect.Type) error {
	target = strictJSONType(target)
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		fields := strictJSONFields(target)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("%s: expected object key", path)
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("%s: duplicate object key %q", path, key)
			}
			seen[key] = struct{}{}
			var child reflect.Type
			if fields != nil {
				var exists bool
				child, exists = fields[key]
				if !exists {
					return fmt.Errorf("%s: unknown field %q", path, key)
				}
			} else if target != nil && target.Kind() == reflect.Map {
				child = target.Elem()
			}
			if err := checkJSONValue(decoder, path+"["+strconv.Quote(key)+"]", child); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return fmt.Errorf("%s: expected end of object", path)
		}
	case '[':
		var child reflect.Type
		if target != nil && (target.Kind() == reflect.Slice || target.Kind() == reflect.Array) {
			child = target.Elem()
		}
		for index := 0; decoder.More(); index++ {
			if err := checkJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index), child); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return fmt.Errorf("%s: expected end of array", path)
		}
	default:
		return fmt.Errorf("%s: unexpected delimiter %q", path, delim)
	}
	return nil
}

// optionalJSONType exposes Optional's value type for the preliminary key walk.
type optionalJSONType interface{ strictOptionalValueType() reflect.Type }

var jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
var optionalJSONTypeType = reflect.TypeOf((*optionalJSONType)(nil)).Elem()

// Custom unmarshalers control their own schema; the key walk still checks
// duplicate keys, but cannot infer the custom type's accepted fields.
func strictJSONType(t reflect.Type) reflect.Type {
	for t != nil {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
			continue
		}
		if t.Implements(optionalJSONTypeType) {
			t = reflect.Zero(t).Interface().(optionalJSONType).strictOptionalValueType()
			continue
		}
		if reflect.PointerTo(t).Implements(jsonUnmarshalerType) || t.Implements(jsonUnmarshalerType) {
			return nil
		}
		return t
	}
	return nil
}

// strictJSONFields returns exact JSON field names for ordinary structs.
// Anonymous embedded structs without a JSON name are flattened as in encoding/json.
func strictJSONFields(t reflect.Type) map[string]reflect.Type {
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	fields := make(map[string]reflect.Type)
	seen := make(map[reflect.Type]bool)
	var add func(reflect.Type)
	add = func(v reflect.Type) {
		if seen[v] {
			return
		}
		seen[v] = true
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.PkgPath != "" {
				continue
			}
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			ft := f.Type
			if f.Anonymous && name == "" {
				for ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				if ft.Kind() == reflect.Struct {
					add(ft)
					continue
				}
			}
			if name == "" {
				name = f.Name
			}
			fields[name] = f.Type
		}
	}
	add(t)
	return fields
}
