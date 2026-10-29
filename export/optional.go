package export

import (
	"encoding/json"
	"fmt"
)

type optionalState uint8

const (
	optionalAbsent optionalState = iota
	optionalNull
	optionalValue
)

// Optional represents an absent, null, or present JSON object field. The zero
// value means absent; tag fields with omitzero so encoding omits that state.
// Null and present values are retained across JSON decoding and encoding.
// Construct non-zero values with Some or NullValue.
type Optional[T any] struct {
	state optionalState
	value T
}

// Some returns a present, non-null value. A value whose own JSON encoding is
// null should be represented with NullValue instead.
func Some[T any](value T) Optional[T] { return Optional[T]{state: optionalValue, value: value} }

// NullValue returns a present field whose JSON value is null.
func NullValue[T any]() Optional[T] { return Optional[T]{state: optionalNull} }

// Present reports whether the field was present, including an explicit null.
func (o Optional[T]) Present() bool { return o.state != optionalAbsent }

// Null reports whether the field was present with a null value.
func (o Optional[T]) Null() bool { return o.state == optionalNull }

// Get returns the contained value and true only for a present non-null field.
func (o Optional[T]) Get() (T, bool) { return o.value, o.state == optionalValue }

// IsZero reports an absent field; encoding/json's omitzero uses it.
func (o Optional[T]) IsZero() bool { return o.state == optionalAbsent }

// UnmarshalJSON applies the strict schema decoder to the contained value.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	return decodeStrictJSON(data, o)
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	switch o.state {
	case optionalAbsent, optionalNull:
		return []byte("null"), nil
	case optionalValue:
		return json.Marshal(o.value)
	}
	return nil, fmt.Errorf("invalid Optional state")
}

func (o *Optional[T]) decodeSchema(d *schemaDecoder) error {
	if d.peekNull() {
		d.skipLiteral()
		*o = Optional[T]{state: optionalNull}
		return nil
	}
	if err := d.decodeAny(&o.value); err != nil {
		return err
	}
	o.state = optionalValue
	return nil
}
