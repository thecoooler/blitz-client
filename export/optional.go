package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

type optionalState uint8

const (
	optionalNull optionalState = iota
	optionalValue
)

// Optional represents an absent, null, or present JSON object field. The zero
// value means absent. Use it with the omitempty JSON tag to omit absent fields.
// Null and present values are retained across JSON decoding and encoding.
//
// An Optional is a map so encoding/json's omitempty can omit its zero value;
// construct non-zero values with Some or NullValue instead of editing the map.
type Optional[T any] map[optionalState]T

// Some returns a present, non-null value. A value whose own JSON encoding is
// null should be represented with NullValue instead.
func Some[T any](value T) Optional[T] { return Optional[T]{optionalValue: value} }

// NullValue returns a present field whose JSON value is null.
func NullValue[T any]() Optional[T] {
	var zero T
	return Optional[T]{optionalNull: zero}
}

// Present reports whether the field was present, including an explicit null.
func (o Optional[T]) Present() bool { return len(o) != 0 }

// Null reports whether the field was present with a null value.
func (o Optional[T]) Null() bool { _, ok := o[optionalNull]; return ok }

// Get returns the contained value and true only for a present non-null field.
func (o Optional[T]) Get() (T, bool) { value, ok := o[optionalValue]; return value, ok }

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*o = NullValue[T]()
		return nil
	}
	var value T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	*o = Some(value)
	return nil
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if len(o) == 0 || (len(o) == 1 && o.Null()) {
		return []byte("null"), nil
	}
	if value, ok := o.Get(); ok && len(o) == 1 {
		return json.Marshal(value)
	}
	return nil, fmt.Errorf("invalid Optional state")
}

func (o Optional[T]) strictOptionalValueType() reflect.Type { return reflect.TypeOf((*T)(nil)).Elem() }
