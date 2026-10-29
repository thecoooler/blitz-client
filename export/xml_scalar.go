package export

import (
	"fmt"
	"strconv"
	"strings"
)

// XML text scalars. The exporter writes XML element text as JSON strings.
// These types validate the spelling observed for a field and keep it
// unchanged: the value is the original string, and encoding writes it back
// as the same JSON string. Field types are assigned from the values of every
// occurrence in real exports (see internal/tools/exportschema), not from
// guessed semantics.

// XMLScalarKind names the spelling classes of XML text scalars.
type XMLScalarKind uint8

const (
	XMLTextKind XMLScalarKind = iota
	XMLIntegerKind
	XMLNumberKind
	XMLBoolKind
	XMLLocalizationKeyKind
	XMLNumberVectorKind
)

// xmlScalarKinds lists kinds from most to least specific; inference picks the
// first kind every observed value fits.
var xmlScalarKinds = []XMLScalarKind{XMLIntegerKind, XMLNumberKind, XMLBoolKind, XMLLocalizationKeyKind, XMLNumberVectorKind, XMLTextKind}

func (k XMLScalarKind) String() string {
	switch k {
	case XMLIntegerKind:
		return "XMLInteger"
	case XMLNumberKind:
		return "XMLNumber"
	case XMLBoolKind:
		return "XMLBool"
	case XMLLocalizationKeyKind:
		return "XMLLocalizationKey"
	case XMLNumberVectorKind:
		return "XMLNumberVector"
	}
	return "string"
}

// Matches reports whether s is spelled as a value of kind k.
func (k XMLScalarKind) Matches(s string) bool {
	switch k {
	case XMLIntegerKind:
		return isXMLInteger(s)
	case XMLNumberKind:
		return isXMLNumber(s)
	case XMLBoolKind:
		return isXMLBool(s)
	case XMLLocalizationKeyKind:
		return isXMLLocalizationKey(s)
	case XMLNumberVectorKind:
		return isXMLNumberVector(s)
	}
	return true
}

// XMLInteger is a decimal integer: optional sign and digits, e.g. "-12".
type XMLInteger string

// XMLNumber is a decimal number: optional sign, digits with an optional
// fraction, optional exponent, e.g. "0.25", "-1", ".5", "1e-3".
type XMLNumber string

// XMLBool is "true", "false", "True" or "False".
type XMLBool string

// XMLLocalizationKey is a localization reference "#namespace:key".
type XMLLocalizationKey string

// XMLNumberVector is one or more XMLNumber values separated by XML whitespace
// (space, tab, CR, LF), e.g. "0 0.5 1". Leading and trailing whitespace and
// line breaks between rows are retained.
type XMLNumberVector string

func (v *XMLInteger) decodeSchema(d *schemaDecoder) error {
	s, err := d.xmlScalar(v, XMLIntegerKind)
	*v = XMLInteger(s)
	return err
}

func (v *XMLNumber) decodeSchema(d *schemaDecoder) error {
	s, err := d.xmlScalar(v, XMLNumberKind)
	*v = XMLNumber(s)
	return err
}

func (v *XMLBool) decodeSchema(d *schemaDecoder) error {
	s, err := d.xmlScalar(v, XMLBoolKind)
	*v = XMLBool(s)
	return err
}

func (v *XMLLocalizationKey) decodeSchema(d *schemaDecoder) error {
	s, err := d.xmlScalar(v, XMLLocalizationKeyKind)
	*v = XMLLocalizationKey(s)
	return err
}

func (v *XMLNumberVector) decodeSchema(d *schemaDecoder) error {
	s, err := d.xmlScalar(v, XMLNumberVectorKind)
	*v = XMLNumberVector(s)
	return err
}

func (v XMLInteger) String() string         { return string(v) }
func (v XMLNumber) String() string          { return string(v) }
func (v XMLBool) String() string            { return string(v) }
func (v XMLLocalizationKey) String() string { return string(v) }
func (v XMLNumberVector) String() string    { return string(v) }

// Int64 parses the integer; it fails only when the value exceeds int64.
func (v XMLInteger) Int64() (int64, error) {
	return strconv.ParseInt(strings.TrimPrefix(string(v), "+"), 10, 64)
}

// Float64 parses the number as the nearest float64.
func (v XMLNumber) Float64() (float64, error) { return strconv.ParseFloat(string(v), 64) }

// Bool returns the boolean value.
func (v XMLBool) Bool() (bool, error) {
	switch v {
	case "true", "True":
		return true, nil
	case "false", "False":
		return false, nil
	}
	return false, fmt.Errorf("invalid XML boolean %q", string(v))
}

// Namespace returns the part between '#' and the first ':'.
func (v XMLLocalizationKey) Namespace() string {
	namespace, _, _ := strings.Cut(strings.TrimPrefix(string(v), "#"), ":")
	return namespace
}

// Key returns the part after the first ':'.
func (v XMLLocalizationKey) Key() string {
	_, key, _ := strings.Cut(string(v), ":")
	return key
}

// Numbers returns the vector components in order, keeping their spelling.
func (v XMLNumberVector) Numbers() []XMLNumber {
	fields := strings.FieldsFunc(string(v), isXMLSpace)
	out := make([]XMLNumber, len(fields))
	for i, f := range fields {
		out[i] = XMLNumber(f)
	}
	return out
}

// Float64s parses the vector components.
func (v XMLNumberVector) Float64s() ([]float64, error) {
	numbers := v.Numbers()
	out := make([]float64, len(numbers))
	for i, n := range numbers {
		f, err := n.Float64()
		if err != nil {
			return nil, err
		}
		out[i] = f
	}
	return out, nil
}

// xmlScalar reads a JSON string for an XML scalar of the given kind and
// reports it to the schema observer before validating it.
func (d *schemaDecoder) xmlScalar(target any, kind XMLScalarKind) (string, error) {
	if d.peek() != '"' {
		return "", d.valueErr("expected %s string, got %s", kind, d.tokenName())
	}
	s := d.readString()
	if d.observe != nil {
		d.observe(d.owner, d.field, kind, s)
	}
	if !kind.Matches(s) {
		return "", d.valueErr("expected %s, got %q", kind, s)
	}
	return s, nil
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isXMLInteger(s string) bool {
	if s != "" && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

func isXMLNumber(s string) bool {
	i := 0
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		i++
	}
	digits := 0
	for i < len(s) && isDigit(s[i]) {
		i++
		digits++
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && isDigit(s[i]) {
			i++
			digits++
		}
	}
	if digits == 0 {
		return false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '-' || s[i] == '+') {
			i++
		}
		exponent := 0
		for i < len(s) && isDigit(s[i]) {
			i++
			exponent++
		}
		if exponent == 0 {
			return false
		}
	}
	return i == len(s)
}

func isXMLBool(s string) bool {
	return s == "true" || s == "false" || s == "True" || s == "False"
}

func isXMLLocalizationKey(s string) bool {
	if !strings.HasPrefix(s, "#") {
		return false
	}
	namespace, key, ok := strings.Cut(s[1:], ":")
	return ok && namespace != "" && key != "" && !strings.ContainsFunc(namespace, func(r rune) bool { return r < 128 && isXMLSpace(r) })
}

func isXMLSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }

func isXMLNumberVector(s string) bool {
	fields := strings.FieldsFunc(s, isXMLSpace)
	if len(fields) == 0 {
		return false
	}
	for _, f := range fields {
		if !isXMLNumber(f) {
			return false
		}
	}
	return true
}
