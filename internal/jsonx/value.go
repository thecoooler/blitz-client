package jsonx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Valuer converts a value into the generic JSON tree produced by ToValue.
type Valuer interface {
	JSONValue() any
}

// Object is a JSON object. Keys are encoded in sorted order, like
// serde_json::Map without the preserve_order feature.
type Object = map[string]any

// ToValue converts v into a generic JSON tree like serde_json::to_value:
// objects become sorted maps, float32 values widen to their float64 digits,
// and nil slices become empty arrays.
func ToValue(v any) any {
	if v == nil {
		return nil
	}
	if valuer, ok := v.(Valuer); ok {
		return valuer.JSONValue()
	}
	switch typed := v.(type) {
	case F32:
		return F64(typed)
	case F64:
		return typed
	case float32:
		return F64(typed)
	case float64:
		return F64(typed)
	case json.Number, string, bool:
		return typed
	case Object:
		out := make(Object, len(typed))
		for key, value := range typed {
			out[key] = ToValue(value)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, value := range typed {
			out[index] = ToValue(value)
		}
		return out
	}
	return reflectValue(reflect.ValueOf(v))
}

func reflectValue(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}
	if value.CanInterface() {
		if valuer, ok := value.Interface().(Valuer); ok {
			if value.Kind() == reflect.Pointer && value.IsNil() {
				return nil
			}
			return valuer.JSONValue()
		}
		if value.Kind() != reflect.Pointer && value.CanAddr() {
			if valuer, ok := value.Addr().Interface().(Valuer); ok {
				return valuer.JSONValue()
			}
		}
	}
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		if value.IsNil() {
			return nil
		}
		return reflectValue(value.Elem())
	case reflect.Bool:
		return value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint()
	case reflect.Float32, reflect.Float64:
		return F64(value.Float())
	case reflect.String:
		return value.String()
	case reflect.Slice, reflect.Array:
		out := make([]any, value.Len())
		for index := range out {
			out[index] = reflectValue(value.Index(index))
		}
		return out
	case reflect.Map:
		out := make(Object, value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			out[mapKey(iterator.Key())] = reflectValue(iterator.Value())
		}
		return out
	case reflect.Struct:
		return structValue(value)
	}
	panic(fmt.Sprintf("jsonx: unsupported kind %s", value.Kind()))
}

func mapKey(key reflect.Value) string {
	switch key.Kind() {
	case reflect.String:
		return key.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(key.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(key.Uint(), 10)
	}
	return fmt.Sprint(key.Interface())
}

func structValue(value reflect.Value) Object {
	out := Object{}
	valueType := value.Type()
	for index := range valueType.NumField() {
		field := valueType.Field(index)
		if !field.IsExported() {
			continue
		}
		name, options, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		fieldValue := value.Field(index)
		if strings.Contains(options, "omitempty") && fieldValue.IsZero() {
			continue
		}
		out[name] = reflectValue(fieldValue)
	}
	return out
}

// DecodeValue parses JSON into a generic tree, keeping integers exact and
// formatting other numbers like serde_json.
func DecodeValue(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, fmt.Errorf("trailing characters")
	}
	return normalizeNumbers(value), nil
}

func normalizeNumbers(value any) any {
	switch typed := value.(type) {
	case json.Number:
		text := string(typed)
		if !strings.ContainsAny(text, ".eE") {
			if _, err := strconv.ParseInt(text, 10, 64); err == nil {
				return typed
			}
			if _, err := strconv.ParseUint(text, 10, 64); err == nil {
				return typed
			}
		}
		number, _ := strconv.ParseFloat(text, 64)
		return F64(number)
	case map[string]any:
		for key, item := range typed {
			typed[key] = normalizeNumbers(item)
		}
		return typed
	case []any:
		for index, item := range typed {
			typed[index] = normalizeNumbers(item)
		}
		return typed
	}
	return value
}
