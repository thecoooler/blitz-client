package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"golang.org/x/term"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/pathx"
	"github.com/thecoooler/blitz-client/internal/rs"
	"github.com/thecoooler/blitz-client/mods"
)

type validationStyle struct{ color bool }

func newValidationStyle(terminal bool) validationStyle {
	nonzero := func(name string) bool {
		value, ok := os.LookupEnv(name)
		return ok && value != "" && value != "0"
	}
	noColor := false
	if value, ok := os.LookupEnv("NO_COLOR"); ok && value != "" {
		noColor = true
	}
	termValue, hasTerm := os.LookupEnv("TERM")
	clicolor, hasCLIColor := os.LookupEnv("CLICOLOR")
	return validationStyle{color: !noColor && (nonzero("CLICOLOR_FORCE") ||
		(terminal && (!hasTerm || termValue != "dumb") && (!hasCLIColor || clicolor != "0")))}
}

func (s validationStyle) paint(code int, value string) string {
	if s.color {
		return fmt.Sprintf("\x1b[%dm%s\x1b[0m", code, value)
	}
	return value
}

func isTerminal(file any) bool {
	if f, ok := file.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (c ModValidateCmd) Run(a *app) error {
	style := newValidationStyle(isTerminal(a.stdout))
	path := c.Path
	gameDir := derefFlag(c.GameDir)
	single := func(manifest string) error {
		report, err := mods.ValidateMod(manifest, gameDir)
		if err != nil {
			return err
		}
		if a.json {
			a.printJSON(report)
		} else {
			a.printValidation(report, style)
		}
		a.valid = report.Valid
		return nil
	}
	if !isDir(path) {
		return single(path)
	}
	manifests, err := mods.DiscoverMods(path)
	if err != nil {
		return err
	}
	if len(manifests) == 0 {
		return blitz.ErrorAt("MANIFEST_NOT_FOUND", "no mod.yaml/mod.yml found in the directory or its subdirectories", path)
	}
	// Keep the existing report shape for a single mod directory.
	if len(manifests) == 1 && pathx.Equal(pathx.Parent(manifests[0]), path) {
		return single(manifests[0])
	}

	type failure struct {
		manifest string
		report   *mods.ValidationReport
		err      *blitz.Error
	}
	results := []any{}
	passed := 0
	var failures []failure
	for _, manifest := range manifests {
		report, err := mods.ValidateMod(manifest, gameDir)
		if err != nil {
			typed := blitz.AsError(err)
			if a.json {
				results = append(results, jsonx.Object{"manifest": manifest, "error": jsonx.ToValue(typed)})
			} else {
				failures = append(failures, failure{manifest: manifest, err: typed})
			}
			continue
		}
		if report.Valid {
			passed++
		}
		switch {
		case a.json:
			results = append(results, jsonx.Object{"manifest": manifest, "report": jsonx.ToValue(report)})
		case report.Valid:
			a.printValidation(report, style)
		default:
			failures = append(failures, failure{manifest: manifest, report: report})
		}
	}
	valid := passed == len(manifests)
	a.valid = valid
	if a.json {
		a.printJSON(jsonx.Object{"valid": valid, "mods": results})
		return nil
	}
	if len(failures) > 0 {
		if passed > 0 {
			a.println()
		}
		a.println(style.paint(31, fmt.Sprintf("Failures (%d)", len(failures))))
		for _, item := range failures {
			if item.report != nil {
				a.printValidation(item.report, style)
			} else {
				a.printManifestFailure(path, item.manifest, item.err, style)
			}
		}
	}
	failedCode := 2
	if !valid {
		failedCode = 31
	}
	a.printf("\n%s · %s · %s\n",
		style.paint(2, fmt.Sprintf("%d mods", len(manifests))),
		style.paint(32, fmt.Sprintf("%d passed", passed)),
		style.paint(failedCode, fmt.Sprintf("%d failed", len(manifests)-passed)))
	return nil
}

func (c ModListCmd) Run(a *app) error {
	report, err := mods.ListInstalledMods(derefFlag(c.GameDir))
	if err != nil {
		return err
	}
	switch {
	case a.json:
		a.printJSON(report)
	case len(report.Mods) == 0:
		a.println("No mods installed")
	default:
		for _, item := range report.Mods {
			noun := "files"
			if len(item.Files) == 1 {
				noun = "file"
			}
			a.printf("%s — %s (%d %s)\n", item.ID, item.Name, len(item.Files), noun)
		}
	}
	return nil
}

type diagnostic struct{ path, code, message string }

func (a *app) printValidation(report *mods.ValidationReport, style validationStyle) {
	seen := map[diagnostic]bool{}
	var errors []mods.FileValidation
	for _, file := range report.Files {
		if file.Valid {
			continue
		}
		key := diagnostic{file.Path, deref(file.ErrorCode, "\x00"), deref(file.Message, "\x00")}
		if !seen[key] {
			seen[key] = true
			errors = append(errors, file)
		}
	}
	var symbol, count string
	if report.Valid {
		var paths []string
		for _, file := range report.Files {
			if !slices.Contains(paths, file.Path) {
				paths = append(paths, file.Path)
			}
		}
		noun := "files"
		if len(paths) == 1 {
			noun = "file"
		}
		symbol, count = style.paint(32, "✓"), fmt.Sprintf("%d %s", len(paths), noun)
	} else {
		noun := "errors"
		if len(errors) == 1 {
			noun = "error"
		}
		symbol, count = style.paint(31, "✗"), fmt.Sprintf("%d %s", len(errors), noun)
	}
	a.printf("%s %s %s\n", symbol, report.ModID, style.paint(2, "("+count+")"))
	for _, file := range errors {
		a.printf("  %s\n", file.Path)
		a.printDiagnostic(style, deref(file.ErrorCode, "ERROR"), deref(file.Message, ""))
	}
}

func deref(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

func (a *app) printManifestFailure(root, manifest string, err *blitz.Error, style validationStyle) {
	relative := manifest
	if rest, ok := stripPathPrefix(manifest, root); ok {
		relative = rest
	}
	label := pathx.Parent(relative)
	if label == "" {
		label = relative
	}
	a.printf("%s %s\n", style.paint(31, "✗"), label)
	a.printDiagnostic(style, err.Code, err.Message)
	if err.Path != "" && !pathx.Equal(err.Path, manifest) {
		a.printf("  %s\n", style.paint(2, err.Path))
	}
}

// stripPathPrefix mirrors Path::strip_prefix for paths built by joining.
func stripPathPrefix(path, prefix string) (string, bool) {
	prefix = strings.TrimRight(prefix, "/"+string(os.PathSeparator))
	if prefix == "" {
		return path, true
	}
	rest, ok := strings.CutPrefix(path, prefix)
	if !ok || (rest != "" && rest[0] != '/' && rest[0] != os.PathSeparator) {
		return "", false
	}
	return strings.TrimLeft(rest, "/"+string(os.PathSeparator)), true
}

func (a *app) printDiagnostic(style validationStyle, code, message string) {
	lines := rs.Lines(message)
	first := ""
	if len(lines) > 0 {
		first, lines = lines[0], lines[1:]
	}
	a.printf("  %s: %s\n", style.paint(31, code), first)
	for _, line := range lines {
		a.printf("  %s\n", line)
	}
}

func (a *app) printValidationError(err *blitz.Error) {
	style := newValidationStyle(isTerminal(a.stderr))
	a.eprintf("%s %s: %s\n", style.paint(31, "✗"), style.paint(31, err.Code), err.Message)
	if err.Path != "" {
		a.eprintf("  %s\n", style.paint(2, err.Path))
	}
}

func (c ModInstallCmd) Run(a *app) error {
	report, err := mods.InstallMod(c.Path, derefFlag(c.GameDir))
	if err != nil {
		return err
	}
	if a.json {
		a.printJSON(report)
		return nil
	}
	verb := "Installed"
	if report.Reinstalled {
		verb = "Reinstalled"
	}
	a.printf("%s %s — %s\n", verb, report.ModID, report.ModName)
	for _, target := range report.Targets {
		a.printf("  %s\n", target)
	}
	return nil
}

func (c ModUninstallCmd) Run(a *app) error {
	report, err := mods.RemoveMod(c.ModID, derefFlag(c.GameDir))
	if err != nil {
		return err
	}
	if a.json {
		a.printJSON(report)
		return nil
	}
	a.printf("Removed %s\n", report.ModID)
	for _, target := range report.RestoredFiles {
		a.printf("  restored %s\n", target)
	}
	return nil
}

func (c ModUpdateFromGitCmd) Run(a *app) error {
	manifest, err := mods.UpdateFromGitDiff(c.Path, c.Repo, c.Base)
	if err != nil {
		return err
	}
	if a.json {
		a.printJSON(manifest)
		return nil
	}
	var generated []string
	for _, file := range manifest.Files {
		if file.Patch != nil && *file.Patch == "mod.patch" && !slices.Contains(generated, file.Name) {
			generated = append(generated, file.Name)
		}
	}
	a.printf("Updated %s: mod.patch (%d resources)\n", manifest.ID, len(generated))
	return nil
}
