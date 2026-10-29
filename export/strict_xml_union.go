package export

import (
	"encoding/json"
	"fmt"
)

// XML elements observed in more than one JSON shape are unions. Exactly one
// variant is set; a containing Optional represents null and absence. S is the
// scalar type of the XML text variant, e.g. string or XMLNumber.

// XMLScalarObject is an XML text scalar or a structured XML value.
type XMLScalarObject[S ~string, T any] struct {
	Scalar *S
	Object *T
}

func (v *XMLScalarObject[S, T]) decodeSchema(d *schemaDecoder) error {
	switch d.peek() {
	case '"':
		var x S
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = XMLScalarObject[S, T]{Scalar: &x}
	case '{':
		var x T
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = XMLScalarObject[S, T]{Object: &x}
	default:
		return d.valueErr("expected XML string or object, got %s", d.tokenName())
	}
	return nil
}

func (v XMLScalarObject[S, T]) MarshalJSON() ([]byte, error) {
	if (v.Scalar == nil) == (v.Object == nil) {
		return nil, fmt.Errorf("XML value must have exactly one variant")
	}
	if v.Scalar != nil {
		return json.Marshal(*v.Scalar)
	}
	return json.Marshal(v.Object)
}

// XMLScalarList is an XML scalar or a list of repeated same-name elements.
type XMLScalarList[S ~string, T any] struct {
	Scalar *S
	List   *[]Optional[T]
}

func (v *XMLScalarList[S, T]) decodeSchema(d *schemaDecoder) error {
	switch d.peek() {
	case '"':
		var x S
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = XMLScalarList[S, T]{Scalar: &x}
	case '[':
		var x []Optional[T]
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = XMLScalarList[S, T]{List: &x}
	default:
		return d.valueErr("expected XML string or list, got %s", d.tokenName())
	}
	return nil
}

func (v XMLScalarList[S, T]) MarshalJSON() ([]byte, error) {
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

func (v *XMLObjectList[O, A]) decodeSchema(d *schemaDecoder) error {
	switch d.peek() {
	case '{':
		var x O
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = XMLObjectList[O, A]{Object: &x}
	case '[':
		var x []Optional[A]
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = XMLObjectList[O, A]{List: &x}
	default:
		return d.valueErr("expected XML object or list, got %s", d.tokenName())
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
type XMLScalarObjectList[S ~string, O, A any] struct {
	Scalar *S
	Object *O
	List   *[]Optional[A]
}

func (v *XMLScalarObjectList[S, O, A]) decodeSchema(d *schemaDecoder) error {
	switch d.peek() {
	case '"':
		var x S
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = XMLScalarObjectList[S, O, A]{Scalar: &x}
	case '{':
		var x O
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = XMLScalarObjectList[S, O, A]{Object: &x}
	case '[':
		var x []Optional[A]
		if err := d.decodeAny(&x); err != nil {
			return err
		}
		*v = XMLScalarObjectList[S, O, A]{List: &x}
	default:
		return d.valueErr("expected XML string, object or list, got %s", d.tokenName())
	}
	return nil
}

func (v XMLScalarObjectList[S, O, A]) MarshalJSON() ([]byte, error) {
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
