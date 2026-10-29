package export

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// XMLScalarObject is an XML text scalar or a structured XML value. Exactly one
// variant must be set. A containing Optional represents null and absence.
type XMLScalarObject[T any] struct {
	Scalar *string
	Object *T
}

func (v *XMLScalarObject[T]) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return fmt.Errorf("empty XML value")
	}
	switch b[0] {
	case '"':
		var x string
		if err := decodeStrictJSON(b, &x); err != nil {
			return err
		}
		*v = XMLScalarObject[T]{Scalar: &x}
	case '{':
		var x T
		if err := decodeStrictJSON(b, &x); err != nil {
			return err
		}
		*v = XMLScalarObject[T]{Object: &x}
	default:
		return fmt.Errorf("expected XML string or object")
	}
	return nil
}
func (v XMLScalarObject[T]) MarshalJSON() ([]byte, error) {
	if (v.Scalar == nil) == (v.Object == nil) {
		return nil, fmt.Errorf("XML value must have exactly one variant")
	}
	if v.Scalar != nil {
		return json.Marshal(*v.Scalar)
	}
	return json.Marshal(v.Object)
}

// XMLScalarList is an XML scalar or a list of repeated same-name elements.
type XMLScalarList[T any] struct {
	Scalar *string
	List   *[]Optional[T]
}

func (v *XMLScalarList[T]) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return fmt.Errorf("empty XML value")
	}
	switch b[0] {
	case '"':
		var x string
		if err := decodeStrictJSON(b, &x); err != nil {
			return err
		}
		*v = XMLScalarList[T]{Scalar: &x}
	case '[':
		var x []Optional[T]
		if err := decodeStrictJSON(b, &x); err != nil {
			return err
		}
		*v = XMLScalarList[T]{List: &x}
	default:
		return fmt.Errorf("expected XML string or list")
	}
	return nil
}
func (v XMLScalarList[T]) MarshalJSON() ([]byte, error) {
	if (v.Scalar == nil) == (v.List == nil) {
		return nil, fmt.Errorf("XML value must have exactly one variant")
	}
	if v.Scalar != nil {
		return json.Marshal(*v.Scalar)
	}
	return json.Marshal(v.List)
}

// XMLObjectList is a single XML object or a list of repeated objects.
type XMLObjectList[O, A any] struct {
	Object *O
	List   *[]Optional[A]
}

func (v *XMLObjectList[O, A]) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return fmt.Errorf("empty XML value")
	}
	switch b[0] {
	case '{':
		var x O
		if err := decodeStrictJSON(b, &x); err != nil {
			return err
		}
		*v = XMLObjectList[O, A]{Object: &x}
	case '[':
		var x []Optional[A]
		if err := decodeStrictJSON(b, &x); err != nil {
			return err
		}
		*v = XMLObjectList[O, A]{List: &x}
	default:
		return fmt.Errorf("expected XML object or list")
	}
	return nil
}
func (v XMLObjectList[O, A]) MarshalJSON() ([]byte, error) {
	if (v.Object == nil) == (v.List == nil) {
		return nil, fmt.Errorf("XML value must have exactly one variant")
	}
	if v.Object != nil {
		return json.Marshal(v.Object)
	}
	return json.Marshal(v.List)
}

// XMLScalarObjectList represents all three XML shapes when each is observed.
type XMLScalarObjectList[O, A any] struct {
	Scalar *string
	Object *O
	List   *[]Optional[A]
}

func (v *XMLScalarObjectList[O, A]) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return fmt.Errorf("empty XML value")
	}
	switch b[0] {
	case '"':
		var x string
		if err := decodeStrictJSON(b, &x); err != nil {
			return err
		}
		*v = XMLScalarObjectList[O, A]{Scalar: &x}
	case '{':
		var x O
		if err := decodeStrictJSON(b, &x); err != nil {
			return err
		}
		*v = XMLScalarObjectList[O, A]{Object: &x}
	case '[':
		var x []Optional[A]
		if err := decodeStrictJSON(b, &x); err != nil {
			return err
		}
		*v = XMLScalarObjectList[O, A]{List: &x}
	default:
		return fmt.Errorf("expected XML string, object or list")
	}
	return nil
}
func (v XMLScalarObjectList[O, A]) MarshalJSON() ([]byte, error) {
	n := 0
	if v.Scalar != nil {
		n++
	}
	if v.Object != nil {
		n++
	}
	if v.List != nil {
		n++
	}
	if n != 1 {
		return nil, fmt.Errorf("XML value must have exactly one variant")
	}
	if v.Scalar != nil {
		return json.Marshal(*v.Scalar)
	}
	if v.Object != nil {
		return json.Marshal(v.Object)
	}
	return json.Marshal(v.List)
}
