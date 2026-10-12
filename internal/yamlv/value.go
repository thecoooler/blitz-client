// Package yamlv reads and writes YAML the way serde_yaml_ng does: plain
// scalars resolve with its YAML 1.2 rules, diagnostics use serde's wording,
// and re-serialized documents use the same scalar styles and layout.
package yamlv

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/rs"
	"go.yaml.in/yaml/v3"
)

// Kind identifies a resolved YAML value.
type Kind int

const (
	Null Kind = iota
	Bool
	Int
	Float
	String
	Seq
	Map
)

// Value is a resolved YAML value with its source position.
type Value struct {
	Kind  Kind
	Bool  bool
	Int   int64
	Uint  uint64
	Neg   bool // Int holds a negative value; otherwise Uint holds it
	Float float64
	Str   string
	Items []*Value
	Keys  []*Value
	Vals  []*Value
	// Plain reports an unquoted scalar; Raw is its source text.
	Plain bool
	Raw   string
	Node  *yaml.Node
}

// Get returns the value for a string key.
func (v *Value) Get(key string) *Value {
	if v == nil || v.Kind != Map {
		return nil
	}
	for index, candidate := range v.Keys {
		if candidate.Kind == String && candidate.Str == key {
			return v.Vals[index]
		}
	}
	return nil
}

// Set replaces a string key's value or appends the key.
func (v *Value) Set(key string, value *Value) {
	for index, candidate := range v.Keys {
		if candidate.Kind == String && candidate.Str == key {
			v.Vals[index] = value
			return
		}
	}
	v.Keys = append(v.Keys, &Value{Kind: String, Str: key})
	v.Vals = append(v.Vals, value)
}

// NewString returns a string value.
func NewString(value string) *Value { return &Value{Kind: String, Str: value} }

// NewSeq returns a sequence of strings.
func NewSeq(values []string) *Value {
	seq := &Value{Kind: Seq}
	for _, value := range values {
		seq.Items = append(seq.Items, NewString(value))
	}
	return seq
}

// Unexpected describes the value like serde::de::Unexpected.
func (v *Value) Unexpected() string {
	switch v.Kind {
	case Null:
		return "unit value"
	case Bool:
		return fmt.Sprintf("boolean `%t`", v.Bool)
	case Int:
		if v.Neg {
			return fmt.Sprintf("integer `%d`", v.Int)
		}
		return fmt.Sprintf("integer `%d`", v.Uint)
	case Float:
		return "floating point `" + serdeFloat(v.Float) + "`"
	case String:
		return "string " + rs.Quote(v.Str)
	case Seq:
		return "sequence"
	default:
		return "map"
	}
}

func serdeFloat(value float64) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "inf"
	case math.IsInf(value, -1):
		return "-inf"
	}
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.ContainsAny(text, ".e") {
		text += ".0"
	}
	return text
}

// Position formats a node position like serde_yaml_ng's Location.
func Position(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return fmt.Sprintf(" at line %d column %d", node.Line, node.Column)
}

// ParseDocument parses exactly one YAML document. An empty document yields a
// nil node.
func ParseDocument(text []byte) (*yaml.Node, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(text))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, parseError(err)
	}
	var next yaml.Node
	if err := decoder.Decode(&next); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, parseError(err)
		}
		return nil, errors.New("deserializing from YAML containing more than one document is not supported")
	}
	if document.Kind == yaml.DocumentNode && len(document.Content) == 1 {
		return resolveAlias(document.Content[0]), nil
	}
	return nil, nil
}

func parseError(err error) error {
	message := strings.TrimPrefix(err.Error(), "yaml: ")
	if rest, ok := strings.CutPrefix(message, "line "); ok {
		if _, after, found := strings.Cut(rest, ": "); found {
			message = after
		}
	}
	return errors.New(message)
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}

// ScalarText returns the content of a scalar node.
func ScalarText(node *yaml.Node) string { return node.Value }

// IsPlain reports whether a scalar is unquoted and untagged.
func IsPlain(node *yaml.Node) bool {
	return node.Style&(yaml.SingleQuotedStyle|yaml.DoubleQuotedStyle|yaml.LiteralStyle|yaml.FoldedStyle) == 0 &&
		(node.Tag == "" || node.Style&yaml.TaggedStyle == 0)
}

// Build resolves a node into a Value. path names the node in diagnostics.
func Build(node *yaml.Node, path string) (*Value, error) {
	node = resolveAlias(node)
	if node == nil {
		return &Value{Kind: Null}, nil
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return scalar(node), nil
	case yaml.SequenceNode:
		seq := &Value{Kind: Seq, Node: node}
		for index, child := range node.Content {
			item, err := Build(child, fmt.Sprintf("%s[%d]", path, index))
			if err != nil {
				return nil, err
			}
			seq.Items = append(seq.Items, item)
		}
		return seq, nil
	case yaml.MappingNode:
		mapping := &Value{Kind: Map, Node: node}
		for index := 0; index+1 < len(node.Content); index += 2 {
			keyNode := node.Content[index]
			key, err := Build(keyNode, path)
			if err != nil {
				return nil, err
			}
			for _, existing := range mapping.Keys {
				if Equal(existing, key) {
					prefix := ""
					if path != "" {
						prefix = path + ": "
					}
					return nil, fmt.Errorf("%sduplicate entry with key %s%s", prefix, key.debugKey(), Position(keyNode))
				}
			}
			childPath := key.pathSegment(path)
			value, err := Build(node.Content[index+1], childPath)
			if err != nil {
				return nil, err
			}
			mapping.Keys = append(mapping.Keys, key)
			mapping.Vals = append(mapping.Vals, value)
		}
		return mapping, nil
	}
	return &Value{Kind: Null}, nil
}

func (v *Value) pathSegment(parent string) string {
	name := v.Raw
	if v.Kind == String {
		name = v.Str
	}
	if parent == "" {
		return name
	}
	return parent + "." + name
}

func (v *Value) debugKey() string {
	switch v.Kind {
	case String:
		return rs.Quote(v.Str)
	case Null:
		return "null"
	default:
		return v.Raw
	}
}

// Equal compares resolved values.
func Equal(a, b *Value) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case Null:
		return true
	case Bool:
		return a.Bool == b.Bool
	case Int:
		return a.Neg == b.Neg && a.Int == b.Int && a.Uint == b.Uint
	case Float:
		return a.Float == b.Float
	case String:
		return a.Str == b.Str
	case Seq:
		if len(a.Items) != len(b.Items) {
			return false
		}
		for index := range a.Items {
			if !Equal(a.Items[index], b.Items[index]) {
				return false
			}
		}
		return true
	default:
		if len(a.Keys) != len(b.Keys) {
			return false
		}
		for index := range a.Keys {
			if !Equal(a.Keys[index], b.Keys[index]) || !Equal(a.Vals[index], b.Vals[index]) {
				return false
			}
		}
		return true
	}
}

func scalar(node *yaml.Node) *Value {
	text := node.Value
	value := &Value{Kind: String, Str: text, Raw: text, Node: node}
	if !IsPlain(node) {
		return value
	}
	value.Plain = true
	resolved := ResolvePlain(text)
	resolved.Raw, resolved.Node, resolved.Plain = text, node, true
	return resolved
}

// ResolvePlain types an untagged plain scalar like visit_untagged_scalar.
func ResolvePlain(text string) *Value {
	if text == "" || text == "~" || text == "null" || text == "Null" || text == "NULL" {
		return &Value{Kind: Null}
	}
	switch text {
	case "true", "True", "TRUE":
		return &Value{Kind: Bool, Bool: true}
	case "false", "False", "FALSE":
		return &Value{Kind: Bool, Bool: false}
	}
	if value, ok := parseUnsigned(text, 64); ok {
		return &Value{Kind: Int, Uint: value}
	}
	if value, ok := parseNegative(text, 64); ok {
		return &Value{Kind: Int, Neg: true, Int: value}
	}
	if _, ok := parseUnsigned(text, 128); ok {
		return &Value{Kind: Float, Float: mustFloat(text)}
	}
	if _, ok := parseNegative(text, 128); ok {
		return &Value{Kind: Float, Float: mustFloat(text)}
	}
	if !DigitsButNotNumber(text) {
		if value, ok := parseF64(text); ok {
			return &Value{Kind: Float, Float: value}
		}
	}
	return &Value{Kind: String, Str: text}
}

func mustFloat(text string) float64 {
	value, _ := strconv.ParseFloat(strings.TrimPrefix(text, "+"), 64)
	return value
}

// IsNonString reports whether serde_yaml_ng would read the plain text as a
// null, boolean, or number, and therefore quotes it when serializing a string.
func IsNonString(text string) bool {
	return ResolvePlain(text).Kind != String || DigitsButNotNumber(text)
}

// DigitsButNotNumber matches leading zeros followed by digits, which YAML 1.2
// reads as a string.
func DigitsButNotNumber(text string) bool {
	text = strings.TrimLeft(text[:min(len(text), 1)], "+-") + text[min(len(text), 1):]
	if len(text) <= 1 || text[0] != '0' {
		return false
	}
	for _, b := range []byte(text[1:]) {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

// radixDigits parses digits with Rust's from_str_radix rules, which accept one
// leading sign.
func radixDigits(text string, radix int, bits int) (uint64, bool, bool) {
	negative := false
	if strings.HasPrefix(text, "-") {
		negative, text = true, text[1:]
	} else {
		text = strings.TrimPrefix(text, "+")
	}
	if text == "" {
		return 0, false, false
	}
	for _, c := range text {
		if !isDigit(c, radix) {
			return 0, false, false
		}
	}
	if bits > 64 {
		return 0, negative, true
	}
	value, err := strconv.ParseUint(text, radix, 64)
	return value, negative, err == nil
}

func isDigit(c rune, radix int) bool {
	var digit int
	switch {
	case c >= '0' && c <= '9':
		digit = int(c - '0')
	case c >= 'a' && c <= 'z':
		digit = int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		digit = int(c-'A') + 10
	default:
		return false
	}
	return digit < radix
}

func parseUnsigned(text string, bits int) (uint64, bool) {
	unpositive := strings.TrimPrefix(text, "+")
	for _, prefix := range []struct {
		text  string
		radix int
	}{{"0x", 16}, {"0o", 8}, {"0b", 2}} {
		if rest, ok := strings.CutPrefix(unpositive, prefix.text); ok {
			if strings.HasPrefix(rest, "+") || strings.HasPrefix(rest, "-") {
				return 0, false
			}
			if value, negative, ok := radixDigits(rest, prefix.radix, bits); ok && !negative {
				return value, true
			}
		}
	}
	if strings.HasPrefix(unpositive, "+") || strings.HasPrefix(unpositive, "-") || DigitsButNotNumber(text) {
		return 0, false
	}
	value, negative, ok := radixDigits(unpositive, 10, bits)
	return value, ok && !negative
}

func parseNegative(text string, bits int) (int64, bool) {
	for _, prefix := range []struct {
		text  string
		radix int
	}{{"-0x", 16}, {"-0o", 8}, {"-0b", 2}} {
		if rest, ok := strings.CutPrefix(text, prefix.text); ok {
			if strings.HasPrefix(rest, "+") || strings.HasPrefix(rest, "-") {
				continue
			}
			if value, ok := negativeValue(rest, prefix.radix, bits); ok {
				return value, true
			}
		}
	}
	if DigitsButNotNumber(text) {
		return 0, false
	}
	rest, ok := strings.CutPrefix(text, "-")
	if !ok || strings.HasPrefix(rest, "+") || strings.HasPrefix(rest, "-") {
		return 0, false
	}
	return negativeValue(rest, 10, bits)
}

func negativeValue(digits string, radix, bits int) (int64, bool) {
	value, negative, ok := radixDigits(digits, radix, bits)
	if !ok || negative {
		return 0, false
	}
	if bits > 64 {
		return 0, true
	}
	if value > 1<<63 {
		return 0, false
	}
	return -int64(value), true
}

func parseF64(text string) (float64, bool) {
	unpositive := text
	if rest, ok := strings.CutPrefix(text, "+"); ok {
		if strings.HasPrefix(rest, "+") || strings.HasPrefix(rest, "-") {
			return 0, false
		}
		unpositive = rest
	}
	switch {
	case unpositive == ".inf" || unpositive == ".Inf" || unpositive == ".INF":
		return math.Inf(1), true
	case text == "-.inf" || text == "-.Inf" || text == "-.INF":
		return math.Inf(-1), true
	case text == ".nan" || text == ".NaN" || text == ".NAN":
		return math.NaN(), true
	}
	if strings.ContainsAny(unpositive, "xX_") || strings.HasPrefix(unpositive, "+") {
		return 0, false
	}
	value, err := strconv.ParseFloat(unpositive, 64)
	if err != nil || math.IsInf(value, 0) || math.IsNaN(value) {
		return 0, false
	}
	return value, true
}

// FormatNumber formats an Int or Float value like serde_yaml_ng.
func FormatNumber(v *Value) string {
	switch {
	case v.Kind == Int && v.Neg:
		return strconv.FormatInt(v.Int, 10)
	case v.Kind == Int:
		return strconv.FormatUint(v.Uint, 10)
	case math.IsInf(v.Float, 1):
		return ".inf"
	case math.IsInf(v.Float, -1):
		return "-.inf"
	case math.IsNaN(v.Float):
		return ".nan"
	}
	return jsonx.FormatRyu(v.Float, 64)
}
