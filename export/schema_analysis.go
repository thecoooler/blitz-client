package export

import (
	"io"
	"reflect"
	"sort"
	"strings"
)

// SchemaField identifies a struct field of the schema by Go names. Owner has
// no type arguments, e.g. "StrictTankConfigurationRadiosEntryProperties".
type SchemaField struct {
	Owner string
	Field string
}

// XMLFieldUsage summarizes the XML text values observed for one schema field,
// including values inside its Optional, union, list and dictionary layers.
type XMLFieldUsage struct {
	// Declared is the kind of the field's current Go type.
	Declared XMLScalarKind
	Count    int
	// ByKind counts values by the most specific kind each one fits.
	ByKind map[XMLScalarKind]int
	// Examples holds up to five distinct values that fit only XMLTextKind.
	Examples []string
	// Mismatches counts values rejected by the declared kind.
	Mismatches int
	fits       uint8
}

// Fits reports whether every observed value is spelled as kind.
func (u *XMLFieldUsage) Fits(kind XMLScalarKind) bool { return u.fits&(1<<kind) != 0 }

// Inferred returns the most specific kind that all observed values fit.
func (u *XMLFieldUsage) Inferred() XMLScalarKind {
	for _, kind := range xmlScalarKinds {
		if u.Fits(kind) {
			return kind
		}
	}
	return XMLTextKind
}

func (u *XMLFieldUsage) add(value string) {
	var mask uint8
	best := XMLTextKind
	for i := len(xmlScalarKinds) - 1; i >= 0; i-- {
		if kind := xmlScalarKinds[i]; kind.Matches(value) {
			mask |= 1 << kind
			best = kind
		}
	}
	if u.Count == 0 {
		u.fits = mask
	} else {
		u.fits &= mask
	}
	u.Count++
	if u.ByKind == nil {
		u.ByKind = map[XMLScalarKind]int{}
	}
	u.ByKind[best]++
	if !u.Declared.Matches(value) {
		u.Mismatches++
	}
	if best == XMLTextKind && len(u.Examples) < 5 {
		for _, e := range u.Examples {
			if e == value {
				return
			}
		}
		u.Examples = append(u.Examples, value)
	}
}

func (u *XMLFieldUsage) merge(o *XMLFieldUsage) {
	if u.Count == 0 {
		u.fits = o.fits
	} else if o.Count != 0 {
		u.fits &= o.fits
	}
	u.Count += o.Count
	u.Mismatches += o.Mismatches
	if u.ByKind == nil {
		u.ByKind = map[XMLScalarKind]int{}
	}
	for k, n := range o.ByKind {
		u.ByKind[k] += n
	}
	for _, e := range o.Examples {
		if len(u.Examples) < 5 && !containsString(u.Examples, e) {
			u.Examples = append(u.Examples, e)
		}
	}
}

func containsString(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}

// SchemaAnalysis reports how an export fits the schema: the lenient decoding
// warnings (fields and shapes the schema lacks) and the spelling of every XML
// text field. It is meant for updating the schema after a game patch.
type SchemaAnalysis struct {
	Warnings []SchemaWarning
	Fields   map[SchemaField]*XMLFieldUsage
}

// AnalyzeClientExportSchema decodes a v1 export of any --include selection
// leniently and collects a SchemaAnalysis.
func AnalyzeClientExportSchema(r io.Reader) (*SchemaAnalysis, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	analysis := &SchemaAnalysis{Fields: map[SchemaField]*XMLFieldUsage{}}
	observe := func(owner reflect.Type, field string, declared XMLScalarKind, value string) {
		if owner == reflect.TypeFor[SnapshotGame]() {
			// Version strings are not XML text; a numeric build must stay a string.
			return
		}
		key := SchemaField{Field: field}
		if owner != nil {
			key.Owner, _, _ = strings.Cut(owner.Name(), "[")
		}
		usage := analysis.Fields[key]
		if usage == nil {
			usage = &XMLFieldUsage{Declared: declared}
			analysis.Fields[key] = usage
		}
		usage.add(value)
	}
	_, warnings, err := decodePartialClientExport(data, schemaOptions{lenient: true, observe: observe})
	analysis.Warnings = warnings
	return analysis, err
}

// Merge adds another analysis, e.g. of an export from a different game
// version, so inference covers both.
func (a *SchemaAnalysis) Merge(o *SchemaAnalysis) {
	a.Warnings = append(a.Warnings, o.Warnings...)
	if a.Fields == nil {
		a.Fields = map[SchemaField]*XMLFieldUsage{}
	}
	for key, usage := range o.Fields {
		if mine := a.Fields[key]; mine != nil {
			mine.merge(usage)
		} else {
			copied := *usage
			copied.ByKind = nil
			copied.Count, copied.Mismatches, copied.Examples = 0, 0, nil
			copied.merge(usage)
			a.Fields[key] = &copied
		}
	}
}

// SortedFields returns the analysed fields ordered by owner and field name.
func (a *SchemaAnalysis) SortedFields() []SchemaField {
	keys := make([]SchemaField, 0, len(a.Fields))
	for key := range a.Fields {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Owner != keys[j].Owner {
			return keys[i].Owner < keys[j].Owner
		}
		return keys[i].Field < keys[j].Field
	})
	return keys
}
