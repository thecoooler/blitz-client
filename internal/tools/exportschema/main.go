// Command exportschema checks the strict export schema against real client
// exports and updates its XML scalar field types.
//
//	go run ./internal/tools/exportschema [-write] [-text] [-stats] data.json...
//
// Each data.json is produced by `blitz-client client export`; give exports of
// several game versions to infer types that fit all of them. The command
// prints
//
//   - schema warnings: fields the schema lacks and values whose shape or
//     spelling it rejects, grouped by schema path;
//   - scalar drift: XML text fields whose declared type differs from the most
//     specific type every observed value fits (string, XMLInteger, XMLNumber,
//     XMLBool, XMLLocalizationKey, XMLNumberVector).
//
// It exits with status 1 when there are warnings or drift. -write rewrites the
// drifted field types in the export package source; structural changes
// reported as warnings must be added by hand. -text lists the fields that stay
// string, with value examples. -stats counts schema fields by scalar type.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/thecoooler/blitz-client/export"
)

func main() {
	write := flag.Bool("write", false, "rewrite drifted scalar field types in -dir")
	text := flag.Bool("text", false, "list fields that remain string")
	stats := flag.Bool("stats", false, "count schema fields by scalar type in -dir")
	dir := flag.String("dir", "export", "export package source directory")
	flag.Parse()
	if *stats && flag.NArg() == 0 {
		if err := printStats(*dir); err != nil {
			fail(err)
		}
		return
	}
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: exportschema [-write] [-text] [-stats] [-dir export] data.json...")
		os.Exit(2)
	}
	analysis := &export.SchemaAnalysis{}
	for _, path := range flag.Args() {
		f, err := os.Open(path)
		if err != nil {
			fail(err)
		}
		a, err := export.AnalyzeClientExportSchema(f)
		f.Close()
		if err != nil {
			fail(fmt.Errorf("%s: %w", path, err))
		}
		analysis.Merge(a)
	}
	problems := printWarnings(analysis.Warnings)
	drift := map[export.SchemaField]export.XMLScalarKind{}
	for _, key := range analysis.SortedFields() {
		usage := analysis.Fields[key]
		if inferred := usage.Inferred(); inferred != usage.Declared && key.Owner != "" {
			drift[key] = inferred
			fmt.Printf("drift: %s.%s: declared %s, values fit %s (%d values%s)\n", key.Owner, key.Field, usage.Declared, inferred, usage.Count, describeKinds(usage))
		}
	}
	problems += len(drift)
	if *text {
		for _, key := range analysis.SortedFields() {
			usage := analysis.Fields[key]
			if usage.Inferred() == export.XMLTextKind {
				fmt.Printf("string: %s.%s: %d values%s; examples %q\n", key.Owner, key.Field, usage.Count, describeKinds(usage), usage.Examples)
			}
		}
		err := eachSchemaFile(*dir, func(path string, _ *token.FileSet, file *ast.File) error {
			if !isSchemaTypesFile(path) {
				return nil
			}
			eachStructField(file, func(owner, field string, f *ast.Field) {
				hasString := false
				walkScalarIdents(f.Type, func(id *ast.Ident) { hasString = hasString || id.Name == "string" })
				if hasString && analysis.Fields[export.SchemaField{Owner: owner, Field: field}] == nil {
					fmt.Printf("string: %s.%s: no values in the given exports\n", owner, field)
				}
			})
			return nil
		})
		if err != nil {
			fail(err)
		}
	}
	if *write && len(drift) > 0 {
		if err := rewrite(*dir, drift); err != nil {
			fail(err)
		}
	}
	if *stats {
		if err := printStats(*dir); err != nil {
			fail(err)
		}
	}
	fmt.Printf("%d fields observed, %d warnings, %d drifted fields\n", len(analysis.Fields), len(analysis.Warnings), len(drift))
	if problems > 0 && !*write {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func describeKinds(u *export.XMLFieldUsage) string {
	if len(u.ByKind) < 2 {
		return ""
	}
	parts := make([]string, 0, len(u.ByKind))
	for kind, n := range u.ByKind {
		parts = append(parts, fmt.Sprintf("%s %d", kind, n))
	}
	sort.Strings(parts)
	return ": " + strings.Join(parts, ", ")
}

func printWarnings(warnings []export.SchemaWarning) int {
	type group struct {
		count   int
		example export.SchemaWarning
	}
	groups := map[string]*group{}
	for _, w := range warnings {
		key := string(w.Kind) + " " + w.SchemaPath
		if groups[key] == nil {
			groups[key] = &group{example: w}
		}
		groups[key].count++
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		g := groups[key]
		fmt.Printf("warning: %s (%d occurrences), e.g. %s\n", key, g.count, g.example)
	}
	return len(groups)
}

var scalarTypes = map[string]bool{
	"string": true, "XMLInteger": true, "XMLNumber": true, "XMLBool": true,
	"XMLLocalizationKey": true, "XMLNumberVector": true,
}

// rewrite replaces the XML scalar types in each drifted field's type
// expression, leaving dictionary key types alone.
func rewrite(dir string, drift map[export.SchemaField]export.XMLScalarKind) error {
	pending := map[export.SchemaField]bool{}
	for key := range drift {
		pending[key] = true
	}
	err := eachSchemaFile(dir, func(path string, fset *token.FileSet, file *ast.File) error {
		changed := false
		eachStructField(file, func(owner, field string, f *ast.Field) {
			kind, ok := drift[export.SchemaField{Owner: owner, Field: field}]
			if !ok {
				return
			}
			replaced := false
			walkScalarIdents(f.Type, func(id *ast.Ident) {
				if scalarTypes[id.Name] {
					id.Name = kind.String()
					replaced = true
				}
			})
			if replaced {
				changed = true
				delete(pending, export.SchemaField{Owner: owner, Field: field})
			}
		})
		if !changed {
			return nil
		}
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, file); err != nil {
			return err
		}
		return os.WriteFile(path, buf.Bytes(), 0o644)
	})
	if err != nil {
		return err
	}
	for key := range pending {
		fmt.Printf("not rewritten: %s.%s has no XML scalar in its type expression; the string is inside a named union type\n", key.Owner, key.Field)
	}
	return nil
}

func eachSchemaFile(dir string, fn func(string, *token.FileSet, *ast.File) error) error {
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		if err := fn(path, fset, file); err != nil {
			return err
		}
	}
	return nil
}

func eachStructField(file *ast.File, fn func(owner, field string, f *ast.Field)) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts := spec.(*ast.TypeSpec)
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, f := range st.Fields.List {
				for _, name := range f.Names {
					fn(ts.Name.Name, name.Name, f)
				}
			}
		}
	}
}

// walkScalarIdents visits the identifiers that can name a field's value type:
// it skips dictionary key types and package selectors.
func walkScalarIdents(expr ast.Expr, fn func(*ast.Ident)) {
	switch e := expr.(type) {
	case *ast.Ident:
		fn(e)
	case *ast.StarExpr:
		walkScalarIdents(e.X, fn)
	case *ast.ArrayType:
		walkScalarIdents(e.Elt, fn)
	case *ast.MapType:
		walkScalarIdents(e.Value, fn)
	case *ast.IndexExpr:
		walkScalarIdents(e.Index, fn)
	case *ast.IndexListExpr:
		for _, index := range e.Indices {
			walkScalarIdents(index, fn)
		}
	}
}

// isSchemaTypesFile reports files that declare the strict schema types.
func isSchemaTypesFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, "strict_") || strings.HasPrefix(base, "partial_")
}

// printStats counts struct fields of the strict schema by the scalar types in
// their type expressions. A field holding scalars of two types counts once
// for each.
func printStats(dir string) error {
	counts := map[string]int{}
	err := eachSchemaFile(dir, func(path string, _ *token.FileSet, file *ast.File) error {
		if !isSchemaTypesFile(path) {
			return nil
		}
		eachStructField(file, func(owner, field string, f *ast.Field) {
			seen := map[string]bool{}
			walkScalarIdents(f.Type, func(id *ast.Ident) {
				switch id.Name {
				case "string", "XMLInteger", "XMLNumber", "XMLBool", "XMLLocalizationKey", "XMLNumberVector", "StrictNumber", "StrictInteger", "StrictNull", "bool", "int", "int64":
					if !seen[id.Name] {
						seen[id.Name] = true
						counts[id.Name]++
					}
				}
			})
		})
		return nil
	})
	if err != nil {
		return err
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return counts[names[i]] > counts[names[j]] })
	for _, name := range names {
		fmt.Printf("stats: %-20s %d fields\n", name, counts[name])
	}
	return nil
}
