package export

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// SchemaWarningKind classifies a lenient decoding warning.
type SchemaWarningKind string

const (
	// SchemaUnknownField is an object key that the schema does not describe.
	// Its value is skipped.
	SchemaUnknownField SchemaWarningKind = "unknown-field"
	// SchemaUnexpectedValue is a value whose shape or scalar spelling does not
	// match the schema. The enclosing field, dictionary entry or array element
	// is dropped.
	SchemaUnexpectedValue SchemaWarningKind = "unexpected-value"
)

// SchemaWarning reports data that lenient decoding could not represent.
type SchemaWarning struct {
	Kind SchemaWarningKind
	// Path addresses the offending value, e.g. $.tanks[3].parameters.foo.
	Path string
	// SchemaPath is Path with array indices written as [] and dictionary keys
	// as {}, so repeated occurrences of one schema change share a SchemaPath.
	SchemaPath string
	// Dropped is the path of the value removed from the result; it is empty
	// for unknown fields and may enclose Path for unexpected values.
	Dropped string
	Message string
}

func (w SchemaWarning) String() string {
	s := fmt.Sprintf("%s: %s: %s", w.Kind, w.Path, w.Message)
	if w.Dropped != "" && w.Dropped != w.Path {
		s += " (dropped " + w.Dropped + ")"
	}
	return s
}

// SchemaError reports data rejected by the schema.
type SchemaError struct {
	Path    string
	Message string
}

func (e *SchemaError) Error() string { return e.Path + ": " + e.Message }

// schemaValueError is a schema mismatch that lenient decoding can recover from
// by dropping the enclosing member. Other errors (duplicate keys) are fatal.
type schemaValueError struct {
	path, schemaPath, message string
}

func (e *schemaValueError) Error() string { return e.path + ": " + e.message }

type pathKind uint8

const (
	pathField pathKind = iota
	pathKey
	pathIndex
)

type pathElem struct {
	kind  pathKind
	key   string
	index int
}

// scalarObserver receives every JSON string decoded into a string-kinded
// value before validation, with the struct type and Go field that contain it
// (through Optional, unions, slices and dictionaries) and the declared kind.
type scalarObserver func(owner reflect.Type, field string, declared XMLScalarKind, value string)

type schemaDecoder struct {
	data     []byte
	pos      int
	lenient  bool
	warnings []SchemaWarning
	path     []pathElem
	seen     []bool
	owner    reflect.Type
	field    string
	observe  scalarObserver
}

// schemaHook decodes types whose JSON shape is not an ordinary struct, map,
// slice or scalar: Optional, unions and validated scalars.
type schemaHook interface {
	decodeSchema(d *schemaDecoder) error
}

// schemaExtraKeys lets a struct accept object keys outside its JSON fields,
// such as dictionaries stored beside fixed fields. It reports whether it
// consumed the value for key.
type schemaExtraKeys interface {
	decodeExtraKey(d *schemaDecoder, key string) (bool, error)
}

// schemaValidator checks constraints across fields after a struct is decoded.
type schemaValidator interface {
	validateSchema() error
}

type schemaOptions struct {
	lenient bool
	observe scalarObserver
}

// decodeStrictJSON decodes one JSON document into target with the strict
// schema: unknown fields, wrong shapes, duplicate keys, missing required
// fields and trailing data are errors.
func decodeStrictJSON(data []byte, target any) error {
	_, err := decodeSchemaJSON(data, target, schemaOptions{})
	return err
}

func decodeSchemaJSON(data []byte, target any, options schemaOptions) ([]SchemaWarning, error) {
	if !json.Valid(data) {
		decoder := json.NewDecoder(bytes.NewReader(data))
		var raw json.RawMessage
		if decoder.Decode(&raw) == nil && decoder.Decode(&raw) == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("invalid JSON")
	}
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return nil, fmt.Errorf("decode target must be a non-nil pointer")
	}
	d := &schemaDecoder{data: data, lenient: options.lenient, observe: options.observe}
	d.skipWS()
	if err := d.decodeValue(v.Elem()); err != nil {
		var ve *schemaValueError
		if errors.As(err, &ve) {
			return d.warnings, &SchemaError{Path: ve.path, Message: ve.message}
		}
		return d.warnings, err
	}
	return d.warnings, nil
}

// Scanner. decodeSchemaJSON validates the document first, so the scanner
// only has to distinguish tokens, not diagnose malformed input.

func (d *schemaDecoder) skipWS() {
	for d.pos < len(d.data) {
		switch d.data[d.pos] {
		case ' ', '\t', '\n', '\r':
			d.pos++
		default:
			return
		}
	}
}

func (d *schemaDecoder) peek() byte {
	d.skipWS()
	if d.pos >= len(d.data) {
		return 0
	}
	return d.data[d.pos]
}

func (d *schemaDecoder) peekNull() bool { return d.peek() == 'n' }

func (d *schemaDecoder) skipLiteral() {
	d.skipWS()
	for d.pos < len(d.data) && d.data[d.pos] >= 'a' && d.data[d.pos] <= 'z' {
		d.pos++
	}
}

// stringBytes returns the raw contents of the string at pos and whether they
// need unescaping.
func (d *schemaDecoder) stringBytes() ([]byte, bool) {
	d.skipWS()
	start := d.pos + 1
	escaped := false
	i := start
	for ; d.data[i] != '"'; i++ {
		if d.data[i] == '\\' {
			escaped = true
			i++
		} else if d.data[i] >= utf8.RuneSelf {
			escaped = true
		}
	}
	d.pos = i + 1
	return d.data[start:i], escaped
}

func (d *schemaDecoder) readString() string {
	start := d.pos
	raw, slow := d.stringBytes()
	if !slow {
		return string(raw)
	}
	return unquoteJSONString(d.data[start:d.pos])
}

func unquoteJSONString(quoted []byte) string {
	inner := quoted[1 : len(quoted)-1]
	if !strings.ContainsRune(string(inner), '\\') && utf8.Valid(inner) {
		return string(inner)
	}
	var s string
	if err := json.Unmarshal(quoted, &s); err != nil {
		panic(fmt.Sprintf("validated JSON string failed to decode: %v", err))
	}
	return s
}

func (d *schemaDecoder) readNumber() string {
	d.skipWS()
	start := d.pos
	for d.pos < len(d.data) && isNumberByte(d.data[d.pos]) {
		d.pos++
	}
	return string(d.data[start:d.pos])
}

func isNumberByte(c byte) bool {
	return c >= '0' && c <= '9' || c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E'
}

func (d *schemaDecoder) skipValue() {
	switch d.peek() {
	case '"':
		d.stringBytes()
	case '{', '[':
		depth := 0
		for {
			switch d.data[d.pos] {
			case '"':
				d.stringBytes()
				continue
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
			d.pos++
			if depth == 0 {
				return
			}
		}
	case 't', 'f', 'n':
		d.skipLiteral()
	default:
		d.readNumber()
	}
}

func (d *schemaDecoder) consume(c byte) bool {
	if d.peek() == c {
		d.pos++
		return true
	}
	return false
}

// Paths and errors.

func (d *schemaDecoder) pathString(schema bool) string {
	var b strings.Builder
	b.WriteByte('$')
	for _, e := range d.path {
		switch e.kind {
		case pathIndex:
			if schema {
				b.WriteString("[]")
			} else {
				b.WriteString("[" + strconv.Itoa(e.index) + "]")
			}
		case pathKey:
			if schema {
				b.WriteString("{}")
				continue
			}
			fallthrough
		default:
			if isPathIdentifier(e.key) {
				b.WriteString("." + e.key)
			} else {
				b.WriteString("[" + strconv.Quote(e.key) + "]")
			}
		}
	}
	return b.String()
}

func isPathIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func (d *schemaDecoder) valueErr(format string, args ...any) error {
	return &schemaValueError{path: d.pathString(false), schemaPath: d.pathString(true), message: fmt.Sprintf(format, args...)}
}

func (d *schemaDecoder) fatalErr(format string, args ...any) error {
	return &SchemaError{Path: d.pathString(false), Message: fmt.Sprintf(format, args...)}
}

func (d *schemaDecoder) push(e pathElem) { d.path = append(d.path, e) }
func (d *schemaDecoder) pop()            { d.path = d.path[:len(d.path)-1] }

// markDynamicKey reports the innermost object key as a dictionary key in
// schema paths.
func (d *schemaDecoder) markDynamicKey() { d.path[len(d.path)-1].kind = pathKey }

func (d *schemaDecoder) duplicateKey(key string) error {
	return d.fatalErr("duplicate object key %q", key)
}

// unknownField handles the value of an unrecognised key at the current path
// (the key is already pushed): an error in strict mode, a warning otherwise.
func (d *schemaDecoder) unknownField(key string) error {
	if !d.lenient {
		d.pop()
		err := d.valueErr("unknown field %q", key)
		d.push(pathElem{kind: pathField, key: key})
		return err
	}
	d.warnings = append(d.warnings, SchemaWarning{Kind: SchemaUnknownField, Path: d.pathString(false), SchemaPath: d.pathString(true), Message: fmt.Sprintf("unknown field %q", key)})
	d.skipValue()
	return nil
}

// member decodes one object member or array element into v. In lenient mode a
// value mismatch is replaced by the zero value (absent) and a warning; the
// result reports whether the value was kept.
func (d *schemaDecoder) member(v reflect.Value, nonNull bool) (bool, error) {
	start, mark := d.pos, len(d.warnings)
	var err error
	if nonNull && d.peekNull() {
		err = d.valueErr("must not be null")
	} else {
		err = d.decodeValue(v)
	}
	if err == nil {
		return true, nil
	}
	var ve *schemaValueError
	if !d.lenient || !errors.As(err, &ve) {
		return false, err
	}
	d.pos = start
	d.skipValue()
	v.SetZero()
	d.warnings = append(d.warnings[:mark], SchemaWarning{Kind: SchemaUnexpectedValue, Path: ve.path, SchemaPath: ve.schemaPath, Dropped: d.pathString(false), Message: ve.message})
	return false, nil
}

// decodeAny decodes the next value into the value ptr points to.
func (d *schemaDecoder) decodeAny(ptr any) error {
	return d.decodeValue(reflect.ValueOf(ptr).Elem())
}

// memberAny is member for a pointer target.
func (d *schemaDecoder) memberAny(ptr any) (bool, error) {
	return d.member(reflect.ValueOf(ptr).Elem(), false)
}

// object iterates the members of a JSON object, with each key pushed on the
// path while fn decodes its value. fn must consume the value.
func (d *schemaDecoder) object(fn func(key string) error) error {
	if !d.consume('{') {
		return d.valueErr("expected object, got %s", d.tokenName())
	}
	var seen map[string]struct{}
	for !d.consume('}') {
		d.consume(',')
		key := d.readString()
		d.consume(':')
		if seen == nil {
			seen = make(map[string]struct{})
		}
		if _, ok := seen[key]; ok {
			return d.duplicateKey(key)
		}
		seen[key] = struct{}{}
		d.push(pathElem{kind: pathField, key: key})
		err := fn(key)
		d.pop()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *schemaDecoder) tokenName() string {
	switch d.peek() {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	}
	return "number"
}

// Type plans.

type planKind uint8

const (
	planHook planKind = iota
	planRaw
	planString
	planBool
	planInt
	planStruct
	planMap
	planSlice
	planArray
	planPointer
	planAny
	planJSONNumber
)

type fieldPlan struct {
	name     string
	goName   string
	index    int
	required bool
	nonNull  bool
	nullable bool
}

type typePlan struct {
	kind     planKind
	fields   []fieldPlan
	byName   map[string]int
	extra    bool
	validate bool
}

var (
	schemaPlans         sync.Map
	schemaHookType      = reflect.TypeOf((*schemaHook)(nil)).Elem()
	schemaExtraType     = reflect.TypeOf((*schemaExtraKeys)(nil)).Elem()
	schemaValidatorType = reflect.TypeOf((*schemaValidator)(nil)).Elem()
	rawMessageType      = reflect.TypeOf(json.RawMessage(nil))
	jsonNumberType      = reflect.TypeOf(json.Number(""))
)

func planFor(t reflect.Type) *typePlan {
	if p, ok := schemaPlans.Load(t); ok {
		return p.(*typePlan)
	}
	p := buildPlan(t)
	actual, _ := schemaPlans.LoadOrStore(t, p)
	return actual.(*typePlan)
}

func isOptionalType(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && strings.HasPrefix(t.Name(), "Optional[") && t.PkgPath() == schemaHookType.PkgPath()
}

func buildPlan(t reflect.Type) *typePlan {
	if reflect.PointerTo(t).Implements(schemaHookType) {
		return &typePlan{kind: planHook}
	}
	switch t {
	case rawMessageType:
		return &typePlan{kind: planRaw}
	case jsonNumberType:
		return &typePlan{kind: planJSONNumber}
	}
	if t.Kind() == reflect.Interface && t.NumMethod() == 0 {
		return &typePlan{kind: planAny}
	}
	switch t.Kind() {
	case reflect.String:
		return &typePlan{kind: planString}
	case reflect.Bool:
		return &typePlan{kind: planBool}
	case reflect.Int, reflect.Int64:
		return &typePlan{kind: planInt}
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			panic("schema map keys must be strings: " + t.String())
		}
		return &typePlan{kind: planMap}
	case reflect.Slice:
		return &typePlan{kind: planSlice}
	case reflect.Array:
		return &typePlan{kind: planArray}
	case reflect.Pointer:
		return &typePlan{kind: planPointer}
	case reflect.Struct:
		p := &typePlan{kind: planStruct, byName: map[string]int{}}
		p.extra = reflect.PointerTo(t).Implements(schemaExtraType)
		p.validate = reflect.PointerTo(t).Implements(schemaValidatorType)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			tag := strings.Split(f.Tag.Get("json"), ",")
			if tag[0] == "-" {
				continue
			}
			if f.Anonymous {
				panic("schema structs cannot embed fields: " + t.String())
			}
			name := tag[0]
			if name == "" {
				name = f.Name
			}
			optional := isOptionalType(f.Type)
			fp := fieldPlan{name: name, goName: f.Name, index: i, required: !optional, nonNull: !optional && f.Type.Kind() != reflect.Pointer}
			for _, option := range strings.Split(f.Tag.Get("schema"), ",") {
				switch option {
				case "required":
					fp.required = true
				case "nonnull":
					fp.nonNull = true
				case "nullable":
					fp.nonNull, fp.nullable = false, true
				case "":
				default:
					panic("unknown schema tag option " + option + " in " + t.String())
				}
			}
			p.byName[name] = len(p.fields)
			p.fields = append(p.fields, fp)
		}
		return p
	}
	panic("unsupported schema type " + t.String())
}

func (d *schemaDecoder) decodeValue(v reflect.Value) error {
	p := planFor(v.Type())
	switch p.kind {
	case planHook:
		return v.Addr().Interface().(schemaHook).decodeSchema(d)
	case planRaw:
		d.skipWS()
		start := d.pos
		d.skipValue()
		v.SetBytes(append([]byte(nil), d.data[start:d.pos]...))
		return nil
	case planString:
		if d.peek() != '"' {
			return d.valueErr("expected string, got %s", d.tokenName())
		}
		s := d.readString()
		if d.observe != nil {
			d.observe(d.owner, d.field, XMLTextKind, s)
		}
		v.SetString(s)
		return nil
	case planBool:
		switch d.peek() {
		case 't':
			v.SetBool(true)
		case 'f':
			v.SetBool(false)
		default:
			return d.valueErr("expected boolean, got %s", d.tokenName())
		}
		d.skipLiteral()
		return nil
	case planInt:
		if c := d.peek(); c != '-' && (c < '0' || c > '9') {
			return d.valueErr("expected integer, got %s", d.tokenName())
		}
		spelling := d.readNumber()
		n, err := strconv.ParseInt(spelling, 10, 64)
		if err != nil {
			return d.valueErr("expected 64-bit integer, got %s", spelling)
		}
		v.SetInt(n)
		return nil
	case planStruct:
		return d.decodeStruct(v, p)
	case planMap:
		return d.decodeMap(v)
	case planSlice:
		return d.decodeSlice(v)
	case planArray:
		return d.decodeArray(v)
	case planJSONNumber:
		if c := d.peek(); c != '-' && (c < '0' || c > '9') {
			return d.valueErr("expected number, got %s", d.tokenName())
		}
		v.SetString(d.readNumber())
		return nil
	case planAny:
		value, err := d.decodeGeneric()
		if err != nil {
			return err
		}
		if value == nil {
			v.SetZero()
		} else {
			v.Set(reflect.ValueOf(value))
		}
		return nil
	case planPointer:
		if d.peekNull() {
			d.skipLiteral()
			v.SetZero()
			return nil
		}
		elem := reflect.New(v.Type().Elem())
		if err := d.decodeValue(elem.Elem()); err != nil {
			return err
		}
		v.Set(elem)
		return nil
	}
	panic("unreachable")
}

// decodeGeneric decodes like encoding/json into any with UseNumber, but
// rejects duplicate keys.
func (d *schemaDecoder) decodeGeneric() (any, error) {
	switch d.peek() {
	case '{':
		m := map[string]any{}
		err := d.object(func(key string) error {
			value, err := d.decodeGeneric()
			m[key] = value
			return err
		})
		return m, err
	case '[':
		d.pos++
		a := []any{}
		for !d.consume(']') {
			d.consume(',')
			value, err := d.decodeGeneric()
			if err != nil {
				return nil, err
			}
			a = append(a, value)
		}
		return a, nil
	case '"':
		return d.readString(), nil
	case 't':
		d.skipLiteral()
		return true, nil
	case 'f':
		d.skipLiteral()
		return false, nil
	case 'n':
		d.skipLiteral()
		return nil, nil
	}
	return json.Number(d.readNumber()), nil
}

func (d *schemaDecoder) decodeStruct(v reflect.Value, p *typePlan) error {
	if !d.consume('{') {
		return d.valueErr("expected object, got %s", d.tokenName())
	}
	v.SetZero()
	base := len(d.seen)
	for range p.fields {
		d.seen = append(d.seen, false)
	}
	defer func() { d.seen = d.seen[:base] }()
	owner, field := d.owner, d.field
	defer func() { d.owner, d.field = owner, field }()
	var unknown map[string]struct{}
	for !d.consume('}') {
		d.consume(',')
		raw, slow := d.stringBytes()
		var key string
		var index int
		var known bool
		if !slow {
			if index, known = p.byName[string(raw)]; !known {
				key = string(raw)
			}
		} else {
			key = unquoteJSONString(d.data[d.pos-len(raw)-2 : d.pos])
			index, known = p.byName[key]
		}
		d.consume(':')
		if known {
			fp := &p.fields[index]
			if d.seen[base+index] {
				return d.duplicateKey(fp.name)
			}
			d.seen[base+index] = true
			d.push(pathElem{kind: pathField, key: fp.name})
			if fp.nullable && d.peekNull() {
				d.skipLiteral()
				v.Field(fp.index).SetZero()
				d.pop()
				continue
			}
			d.owner, d.field = v.Type(), fp.goName
			kept, err := d.member(v.Field(fp.index), fp.nonNull)
			d.owner, d.field = owner, field
			if err == nil && !kept && fp.required {
				err = d.valueErr("required field was dropped")
			}
			d.pop()
			if err != nil {
				return err
			}
			continue
		}
		if unknown == nil {
			unknown = make(map[string]struct{})
		}
		if _, ok := unknown[key]; ok {
			return d.duplicateKey(key)
		}
		unknown[key] = struct{}{}
		d.push(pathElem{kind: pathField, key: key})
		var err error
		handled := false
		if p.extra {
			d.owner, d.field = v.Type(), ""
			handled, err = v.Addr().Interface().(schemaExtraKeys).decodeExtraKey(d, key)
			d.owner, d.field = owner, field
		}
		if err == nil && !handled {
			err = d.unknownField(key)
		}
		d.pop()
		if err != nil {
			return err
		}
	}
	for i := range p.fields {
		fp := &p.fields[i]
		if fp.required && !d.seen[base+i] {
			return d.valueErr("missing required field %q", fp.name)
		}
	}
	if p.validate {
		if err := v.Addr().Interface().(schemaValidator).validateSchema(); err != nil {
			return d.valueErr("%v", err)
		}
	}
	return nil
}

func (d *schemaDecoder) decodeMap(v reflect.Value) error {
	if d.peek() != '{' {
		return d.valueErr("expected object, got %s", d.tokenName())
	}
	v.Set(reflect.MakeMap(v.Type()))
	elemType := v.Type().Elem()
	return d.object(func(key string) error {
		d.markDynamicKey()
		elem := reflect.New(elemType).Elem()
		kept, err := d.member(elem, false)
		if kept {
			v.SetMapIndex(reflect.ValueOf(key).Convert(v.Type().Key()), elem)
		}
		return err
	})
}

func (d *schemaDecoder) decodeSlice(v reflect.Value) error {
	if !d.consume('[') {
		return d.valueErr("expected array, got %s", d.tokenName())
	}
	v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	for index := 0; !d.consume(']'); index++ {
		d.consume(',')
		n := v.Len()
		if n == v.Cap() {
			v.Grow(max(n, 4))
		}
		v.SetLen(n + 1)
		d.push(pathElem{kind: pathIndex, index: index})
		kept, err := d.member(v.Index(n), false)
		d.pop()
		if err != nil {
			return err
		}
		if !kept {
			v.SetLen(n)
		}
	}
	return nil
}

func (d *schemaDecoder) decodeArray(v reflect.Value) error {
	if !d.consume('[') {
		return d.valueErr("expected array, got %s", d.tokenName())
	}
	index := 0
	for ; !d.consume(']'); index++ {
		d.consume(',')
		if index >= v.Len() {
			return d.valueErr("expected %d elements", v.Len())
		}
		d.push(pathElem{kind: pathIndex, index: index})
		err := d.decodeValue(v.Index(index))
		d.pop()
		if err != nil {
			return err
		}
	}
	if index != v.Len() {
		return d.valueErr("expected %d elements, got %d", v.Len(), index)
	}
	return nil
}
