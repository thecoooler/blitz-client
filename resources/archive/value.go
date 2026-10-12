// Package archive decodes DAVA keyed archives used by SC2 scenes and SCG
// polygon-group files.
package archive

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Kind identifies the variant held by a Value.
type Kind int

const (
	Null Kind = iota
	Bool
	Signed
	Unsigned
	Float
	String
	Bytes
	Array
	Map
)

// Value is a decoded keyed-archive value.
type Value struct {
	Kind     Kind
	Bool     bool
	Signed   int64
	Unsigned uint64
	Float    float64
	String   string
	Bytes    []byte
	Array    []Value
	Map      map[string]Value
}

// Get returns a map entry; it is nil for missing keys and non-map values.
func (v *Value) Get(key string) *Value {
	if v == nil || v.Kind != Map {
		return nil
	}
	value, ok := v.Map[key]
	if !ok {
		return nil
	}
	return &value
}

// AsString returns the value of a String.
func (v *Value) AsString() (string, bool) {
	if v == nil || v.Kind != String {
		return "", false
	}
	return v.String, true
}

// AsBytes returns the value of a Bytes value.
func (v *Value) AsBytes() ([]byte, bool) {
	if v == nil || v.Kind != Bytes {
		return nil, false
	}
	return v.Bytes, true
}

// AsF64 converts numeric values to float64.
func (v *Value) AsF64() (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch v.Kind {
	case Float:
		return v.Float, true
	case Signed:
		return float64(v.Signed), true
	case Unsigned:
		return float64(v.Unsigned), true
	}
	return 0, false
}

// AsU64 converts non-negative integers to uint64.
func (v *Value) AsU64() (uint64, bool) {
	if v == nil {
		return 0, false
	}
	switch v.Kind {
	case Unsigned:
		return v.Unsigned, true
	case Signed:
		if v.Signed >= 0 {
			return uint64(v.Signed), true
		}
	}
	return 0, false
}

// AsBool treats nonzero integers as true and other non-boolean values as false.
func (v *Value) AsBool() bool {
	if v == nil {
		return false
	}
	switch v.Kind {
	case Bool:
		return v.Bool
	case Signed:
		return v.Signed != 0
	case Unsigned:
		return v.Unsigned != 0
	}
	return false
}

// AsArray returns the items of an Array.
func (v *Value) AsArray() ([]Value, bool) {
	if v == nil || v.Kind != Array {
		return nil, false
	}
	return v.Array, true
}

// AsMap returns the entries of a Map.
func (v *Value) AsMap() (map[string]Value, bool) {
	if v == nil || v.Kind != Map {
		return nil, false
	}
	return v.Map, true
}

// SortedKeys returns map keys in byte order, matching the original BTreeMap.
func SortedKeys(entries map[string]Value) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Debug formats the value like the Rust implementation's Debug output, which
// appears in some diagnostics.
func (v Value) Debug() string {
	switch v.Kind {
	case Null:
		return "Null"
	case Bool:
		return fmt.Sprintf("Bool(%t)", v.Bool)
	case Signed:
		return fmt.Sprintf("Signed(%d)", v.Signed)
	case Unsigned:
		return fmt.Sprintf("Unsigned(%d)", v.Unsigned)
	case Float:
		return "Float(" + DebugFloat(v.Float) + ")"
	case String:
		return "String(" + strconv.Quote(v.String) + ")"
	case Bytes:
		parts := make([]string, len(v.Bytes))
		for index, b := range v.Bytes {
			parts[index] = strconv.Itoa(int(b))
		}
		return "Bytes([" + strings.Join(parts, ", ") + "])"
	case Array:
		parts := make([]string, len(v.Array))
		for index, item := range v.Array {
			parts[index] = item.Debug()
		}
		return "Array([" + strings.Join(parts, ", ") + "])"
	case Map:
		keys := SortedKeys(v.Map)
		parts := make([]string, len(keys))
		for index, key := range keys {
			parts[index] = strconv.Quote(key) + ": " + v.Map[key].Debug()
		}
		return "Map({" + strings.Join(parts, ", ") + "})"
	}
	return "?"
}

// DebugFloat formats a float the way Rust's Debug does for f64.
func DebugFloat(value float64) string {
	text := strconv.FormatFloat(value, 'f', -1, 64)
	switch text {
	case "NaN", "+Inf", "-Inf":
		return map[string]string{"NaN": "NaN", "+Inf": "inf", "-Inf": "-inf"}[text]
	}
	if !strings.Contains(text, ".") {
		text += ".0"
	}
	return text
}
