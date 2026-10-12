package mods

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/pathx"
	"github.com/thecoooler/blitz-client/internal/rs"
	"github.com/thecoooler/blitz-client/resources"
)

const (
	backupSuffix = ".tanks-blitz-client-original"
	stateDir     = ".tanks-blitz-client"
)

// FileValidation is the validation result for one logical resource.
type FileValidation struct {
	Path       string  `json:"path"`
	Target     *string `json:"target"`
	Valid      bool    `json:"valid"`
	OutputSize *int    `json:"output_size"`
	ErrorCode  *string `json:"error_code"`
	Message    *string `json:"message"`
}

// ValidationReport is the result of ValidateMod.
type ValidationReport struct {
	ModID       string           `json:"mod_id"`
	ModName     string           `json:"mod_name"`
	GameDir     string           `json:"game_dir"`
	ResourceDir string           `json:"resource_dir"`
	Valid       bool             `json:"valid"`
	Files       []FileValidation `json:"files"`
}

// InstallReport is the result of InstallMod.
type InstallReport struct {
	ModID   string `json:"mod_id"`
	ModName string `json:"mod_name"`
	// Installed is always true for a successful installation.
	Installed bool `json:"installed"`
	// Reinstalled is true when an installed mod with this id was replaced.
	Reinstalled bool     `json:"reinstalled"`
	Targets     []string `json:"targets"`
}

// InstalledModInfo is installed metadata from saved state; listing never
// reads mod source files.
type InstalledModInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Files holds unique logical resource paths, sorted by path.
	Files []string `json:"files"`
}

// InstalledModsReport is the result of ListInstalledMods.
type InstalledModsReport struct {
	GameDir     string             `json:"game_dir"`
	ResourceDir string             `json:"resource_dir"`
	Mods        []InstalledModInfo `json:"mods"`
}

// RemovalReport is the result of RemoveMod.
type RemovalReport struct {
	ModID         string   `json:"mod_id"`
	Removed       bool     `json:"removed"`
	RestoredFiles []string `json:"restored_files"`
}

type installerState struct {
	Version uint32         `json:"version"`
	Mods    []installedMod `json:"mods"`
}

type installedMod struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Files []installedFile `json:"files"`
}

type installedFile struct {
	LogicalPath    string          `json:"logical_path"`
	Target         string          `json:"target"`
	BaseSnapshot   string          `json:"base_snapshot"`
	Operation      storedOperation `json:"operation"`
	InstalledCRC32 uint32          `json:"installed_crc32"`
}

type storedOperation struct {
	Action                   Action  `json:"action"`
	Find                     *string `json:"find"`
	Replace                  *string `json:"replace"`
	PatchDiff                *string `json:"patch_diff"`
	IgnoreTrailingWhitespace bool    `json:"ignore_trailing_whitespace"`
}

type modOperation struct {
	modID     string
	operation storedOperation
}

type preparedFile struct {
	logicalPath          string
	target               string
	baseSnapshot         string
	baseBytes            []byte
	createBaseSnapshot   bool
	createAdjacentOrigin bool
	operations           []storedOperation
	output               []byte
	oldCurrent           []byte
}

// ListInstalledMods lists installed mods in id order. No installer state
// means an empty list.
func ListInstalledMods(gameDir string) (*InstalledModsReport, error) {
	game, err := client.ResolveGameDir(gameDir)
	if err != nil {
		return nil, err
	}
	state, err := loadState(game.GameDir)
	if err != nil {
		return nil, err
	}
	mods := []InstalledModInfo{}
	for _, item := range state.Mods {
		files := []string{}
		for _, file := range item.Files {
			if !slices.Contains(files, file.LogicalPath) {
				files = append(files, file.LogicalPath)
			}
		}
		sort.Strings(files)
		mods = append(mods, InstalledModInfo{ID: item.ID, Name: item.Name, Files: files})
	}
	sort.SliceStable(mods, func(i, j int) bool { return mods[i].ID < mods[j].ID })
	return &InstalledModsReport{GameDir: game.GameDir, ResourceDir: game.ResourceDir, Mods: mods}, nil
}

func lowerKey(path string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 'a' - 'A'
		}
		return r
	}, path)
}

func otherOperations(state *installerState, modID, target string) []modOperation {
	var out []modOperation
	for _, item := range state.Mods {
		if item.ID == modID {
			continue
		}
		for _, file := range item.Files {
			if samePath(file.Target, target) {
				out = append(out, modOperation{item.ID, file.Operation})
			}
		}
	}
	return out
}

func newOperations(manifest *Manifest, logicalPath string) ([]storedOperation, error) {
	var out []storedOperation
	for index := range manifest.Files {
		if manifest.Files[index].Name != logicalPath {
			continue
		}
		operation, err := operationFromEntry(&manifest.Files[index])
		if err != nil {
			return nil, err
		}
		out = append(out, operation)
	}
	return out, nil
}

func sortByMod(operations []modOperation) {
	sort.SliceStable(operations, func(i, j int) bool { return operations[i].modID < operations[j].modID })
}

// ValidateMod checks a mod against the game and installed mods without
// changing files.
func ValidateMod(modPath, gameDir string) (*ValidationReport, error) {
	manifest, err := LoadManifest(modPath)
	if err != nil {
		return nil, err
	}
	game, err := client.ResolveGameDir(gameDir)
	if err != nil {
		return nil, err
	}
	state, err := loadState(game.GameDir)
	if err != nil {
		return nil, err
	}
	var paths []string
	seen := map[string]bool{}
	for _, file := range manifest.Files {
		paths = append(paths, file.Name)
		seen[file.Name] = true
	}
	for _, item := range state.Mods {
		if item.ID != manifest.ID {
			continue
		}
		for _, file := range item.Files {
			if !seen[file.LogicalPath] {
				seen[file.LogicalPath] = true
				paths = append(paths, file.LogicalPath)
			}
		}
		break
	}

	files := []FileValidation{}
	occupied := map[string]string{}
	for _, logicalPath := range paths {
		target, size, err := validateTarget(game, state, manifest, logicalPath, occupied)
		if err != nil {
			typed := blitz.AsError(err)
			var targetPath *string
			if typed.Path != "" {
				value := typed.Path
				targetPath = &value
			}
			code, message := typed.Code, typed.Message
			files = append(files, FileValidation{Path: logicalPath, Target: targetPath, ErrorCode: &code, Message: &message})
			continue
		}
		files = append(files, FileValidation{Path: logicalPath, Target: &target, Valid: true, OutputSize: &size})
	}
	valid := true
	for _, file := range files {
		valid = valid && file.Valid
	}
	return &ValidationReport{
		ModID: manifest.ID, ModName: manifest.Name, GameDir: game.GameDir,
		ResourceDir: game.ResourceDir, Valid: valid, Files: files,
	}, nil
}

func validateTarget(game *client.ResolvedGameDir, state *installerState, manifest *Manifest, logicalPath string,
	occupied map[string]string) (string, int, error) {
	target, err := resolveLogicalTarget(game, logicalPath)
	if err != nil {
		return "", 0, err
	}
	key := lowerKey(target)
	previous, exists := occupied[key]
	occupied[key] = logicalPath
	if exists && previous != logicalPath {
		return "", 0, blitz.NewError("MANIFEST_DUPLICATE_TARGET", "multiple entries resolve to the same game file")
	}
	current, err := os.ReadFile(target)
	if err != nil {
		return "", 0, blitz.IOError("could not read game resource", target, err)
	}
	snapshot, crc, managed, err := managedResource(state, target)
	if err != nil {
		return "", 0, err
	}
	base := current
	if managed {
		if resources.CRC32(current) != crc {
			return "", 0, blitz.ErrorAt("INSTALLED_FILE_CHANGED", "managed resource changed outside the installer", target)
		}
		if base, err = readBaseSnapshot(game, target, snapshot); err != nil {
			return "", 0, err
		}
	}
	added, err := newOperations(manifest, logicalPath)
	if err != nil {
		return "", 0, err
	}
	operations := otherOperations(state, manifest.ID, target)
	for _, operation := range added {
		operations = append(operations, modOperation{manifest.ID, operation})
	}
	sortByMod(operations)
	if len(operations) == 0 {
		return target, len(base), nil
	}
	output, err := composeResource(target, base, operations)
	if err != nil {
		return "", 0, err
	}
	return target, len(output), nil
}

func checkInstalledFiles(game *client.ResolvedGameDir, installed *installedMod, readContext string, checkSnapshot bool) error {
	for _, file := range installed.Files {
		expected, err := resolveLogicalTarget(game, file.LogicalPath)
		if err != nil {
			return err
		}
		if !samePath(file.Target, expected) {
			return blitz.ErrorAt("STATE_INVALID_TARGET", "installed state target does not match its logical resource path", file.Target)
		}
		if checkSnapshot {
			expectedSnapshot, err := baseSnapshotPath(game, expected)
			if err != nil {
				return err
			}
			if !samePath(file.BaseSnapshot, expectedSnapshot) {
				return blitz.ErrorAt("STATE_INVALID_BASE", "installed state base snapshot path is invalid", file.BaseSnapshot)
			}
		}
		current, err := os.ReadFile(expected)
		if err != nil {
			return blitz.IOError(readContext, expected, err)
		}
		if resources.CRC32(current) != file.InstalledCRC32 {
			return blitz.ErrorAt("INSTALLED_FILE_CHANGED", "managed resource changed outside the installer", expected)
		}
	}
	return nil
}

// InstallMod installs or unconditionally reinstalls a mod, preparing all
// affected files before writing anything.
func InstallMod(modPath, gameDir string) (*InstallReport, error) {
	manifest, err := LoadManifest(modPath)
	if err != nil {
		return nil, err
	}
	game, err := client.ResolveGameDir(gameDir)
	if err != nil {
		return nil, err
	}
	statePath := statePathOf(game.GameDir)
	state, err := loadState(game.GameDir)
	if err != nil {
		return nil, err
	}
	var previous *installedMod
	for index := range state.Mods {
		if state.Mods[index].ID == manifest.ID {
			copied := state.Mods[index]
			previous = &copied
			break
		}
	}
	if previous != nil {
		// Old-only targets also need validation before their edits can be removed.
		if err := checkInstalledFiles(game, previous, "could not read managed game file", false); err != nil {
			return nil, err
		}
	}

	var prepared []preparedFile
	targets := map[string]bool{}
	for _, logicalPath := range affectedPaths(manifest, previous) {
		target, err := resolveLogicalTarget(game, logicalPath)
		if err != nil {
			return nil, err
		}
		key := lowerKey(target)
		if targets[key] {
			return nil, blitz.ErrorAt("MANIFEST_DUPLICATE_TARGET", "multiple entries resolve to the same game file", target)
		}
		targets[key] = true
		oldCurrent, err := os.ReadFile(target)
		if err != nil {
			return nil, blitz.IOError("could not read game resource", target, err)
		}
		snapshot, crc, managed, err := managedResource(state, target)
		if err != nil {
			return nil, err
		}
		file := preparedFile{logicalPath: logicalPath, target: target, oldCurrent: oldCurrent}
		if managed {
			if resources.CRC32(oldCurrent) != crc {
				return nil, blitz.ErrorAt("INSTALLED_FILE_CHANGED", "managed resource changed outside the installer", target)
			}
			if file.baseBytes, err = readBaseSnapshot(game, target, snapshot); err != nil {
				return nil, err
			}
			file.baseSnapshot = snapshot
		} else {
			if file.baseSnapshot, err = baseSnapshotPath(game, target); err != nil {
				return nil, err
			}
			file.baseBytes = oldCurrent
			file.createBaseSnapshot = true
		}
		if file.createAdjacentOrigin, err = inspectAdjacentOriginal(target + backupSuffix); err != nil {
			return nil, err
		}
		if file.operations, err = newOperations(manifest, logicalPath); err != nil {
			return nil, err
		}
		operations := otherOperations(state, manifest.ID, target)
		for _, operation := range file.operations {
			operations = append(operations, modOperation{manifest.ID, operation})
		}
		sortByMod(operations)
		file.output = file.baseBytes
		if len(operations) > 0 {
			if file.output, err = composeResource(target, file.baseBytes, operations); err != nil {
				return nil, err
			}
		}
		prepared = append(prepared, file)
	}

	var createdSnapshots []string
	removeCreated := func() {
		for _, path := range createdSnapshots {
			os.Remove(path)
		}
	}
	for _, file := range prepared {
		if file.createBaseSnapshot {
			if err := createBackup(file.baseSnapshot, file.baseBytes); err != nil {
				removeCreated()
				return nil, err
			}
			createdSnapshots = append(createdSnapshots, file.baseSnapshot)
		}
		if file.createAdjacentOrigin {
			original := file.target + backupSuffix
			if err := createBackup(original, file.baseBytes); err != nil {
				removeCreated()
				return nil, err
			}
			createdSnapshots = append(createdSnapshots, original)
		}
	}

	var committed []committedFile
	for _, file := range prepared {
		if err := replaceFile(file.target, file.output); err != nil {
			rollbackTargets(committed)
			removeCreated()
			return nil, err
		}
		committed = append(committed, committedFile{file.target, file.oldCurrent})
	}

	remaining := state.Mods[:0]
	for _, item := range state.Mods {
		if item.ID != manifest.ID {
			remaining = append(remaining, item)
		}
	}
	state.Mods = remaining
	for modIndex := range state.Mods {
		for fileIndex := range state.Mods[modIndex].Files {
			file := &state.Mods[modIndex].Files[fileIndex]
			for _, item := range prepared {
				if samePath(item.target, file.Target) {
					file.InstalledCRC32 = resources.CRC32(item.output)
					break
				}
			}
		}
	}
	installedFiles := []installedFile{}
	for _, file := range prepared {
		for _, operation := range file.operations {
			installedFiles = append(installedFiles, installedFile{
				LogicalPath:    file.logicalPath,
				Target:         file.target,
				BaseSnapshot:   file.baseSnapshot,
				Operation:      operation,
				InstalledCRC32: resources.CRC32(file.output),
			})
		}
	}
	state.Version = 2
	state.Mods = append(state.Mods, installedMod{ID: manifest.ID, Name: manifest.Name, Files: installedFiles})
	if err := saveState(statePath, state); err != nil {
		rollbackTargets(committed)
		removeCreated()
		return nil, err
	}

	// Release snapshots for targets dropped by this version and no longer managed.
	for _, file := range prepared {
		if !snapshotInUse(state, file.baseSnapshot) {
			os.Remove(file.baseSnapshot)
		}
	}
	report := &InstallReport{ModID: manifest.ID, ModName: manifest.Name, Installed: true, Reinstalled: previous != nil, Targets: []string{}}
	for _, file := range prepared {
		report.Targets = append(report.Targets, file.target)
	}
	return report, nil
}

func snapshotInUse(state *installerState, snapshot string) bool {
	for _, item := range state.Mods {
		for _, stored := range item.Files {
			if samePath(stored.BaseSnapshot, snapshot) {
				return true
			}
		}
	}
	return false
}

type restoreItem struct {
	target, snapshot string
	output, current  []byte
}

// RemoveMod uninstalls a mod, recomposing its resources from saved originals
// and the remaining mods.
func RemoveMod(modID, gameDir string) (*RemovalReport, error) {
	game, err := client.ResolveGameDir(gameDir)
	if err != nil {
		return nil, err
	}
	statePath := statePathOf(game.GameDir)
	state, err := loadState(game.GameDir)
	if err != nil {
		return nil, err
	}
	index := slices.IndexFunc(state.Mods, func(item installedMod) bool { return item.ID == modID })
	if index < 0 {
		return nil, blitz.NewError("MOD_NOT_INSTALLED", fmt.Sprintf("mod %s is not installed", rs.Quote(modID)))
	}
	installed := state.Mods[index]
	if err := checkInstalledFiles(game, &installed, "could not read installed game file", true); err != nil {
		return nil, err
	}

	state.Mods = slices.Delete(slices.Clone(state.Mods), index, index+1)
	var restore []restoreItem
	restored := map[string]bool{}
	for _, file := range installed.Files {
		if restored[file.Target] {
			continue
		}
		restored[file.Target] = true
		base, err := readBaseSnapshot(game, file.Target, file.BaseSnapshot)
		if err != nil {
			return nil, err
		}
		operations := operationsForTarget(state, file.Target)
		output := base
		if len(operations) > 0 {
			if output, err = composeResource(file.Target, base, operations); err != nil {
				return nil, err
			}
		}
		current, err := os.ReadFile(file.Target)
		if err != nil {
			return nil, blitz.IOError("could not read game file before restore", file.Target, err)
		}
		if resources.CRC32(current) != file.InstalledCRC32 {
			return nil, blitz.ErrorAt("INSTALLED_FILE_CHANGED", "managed resource changed outside the installer", file.Target)
		}
		restore = append(restore, restoreItem{file.Target, file.BaseSnapshot, output, current})
	}

	for _, item := range restore {
		for modIndex := range state.Mods {
			for fileIndex := range state.Mods[modIndex].Files {
				file := &state.Mods[modIndex].Files[fileIndex]
				if samePath(file.Target, item.target) {
					file.InstalledCRC32 = resources.CRC32(item.output)
				}
			}
		}
	}
	var committed []committedFile
	for _, item := range restore {
		if err := replaceFile(item.target, item.output); err != nil {
			rollbackTargets(committed)
			return nil, err
		}
		committed = append(committed, committedFile{item.target, item.current})
	}
	state.Version = 2
	if err := saveState(statePath, state); err != nil {
		rollbackTargets(committed)
		return nil, err
	}
	for _, item := range restore {
		if !snapshotInUse(state, item.snapshot) {
			os.Remove(item.snapshot)
		}
	}
	report := &RemovalReport{ModID: installed.ID, Removed: true, RestoredFiles: []string{}}
	for _, item := range restore {
		report.RestoredFiles = append(report.RestoredFiles, item.target)
	}
	return report, nil
}

func affectedPaths(manifest *Manifest, previous *installedMod) []string {
	var out []string
	seen := map[string]bool{}
	add := func(path string) {
		if !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
	}
	for _, file := range manifest.Files {
		add(file.Name)
	}
	if previous != nil {
		for _, file := range previous.Files {
			add(file.LogicalPath)
		}
	}
	return out
}

func resolveLogicalTarget(game *client.ResolvedGameDir, name string) (string, error) {
	relative, err := safeRelativePath(name)
	if err != nil {
		return "", blitz.NewError("MANIFEST_INVALID_PATH", err.Error())
	}
	logical := filepath.Join(game.ResourceDir, filepath.FromSlash(relative))
	explicitDVPL := strings.HasSuffix(lowerKey(name), ".dvpl")
	selected := logical
	if !explicitDVPL {
		dvplPath := logical + ".dvpl"
		compressed, plain := isFile(dvplPath), isFile(logical)
		switch {
		case compressed && plain:
			return "", blitz.ErrorAt("TARGET_AMBIGUOUS", "both plain and .dvpl versions of this resource exist", logical)
		case compressed:
			selected = dvplPath
		case !plain:
			return "", blitz.ErrorAt("TARGET_NOT_FOUND", "resource file was not found", logical)
		}
	}
	if !isFile(selected) {
		return "", blitz.ErrorAt("TARGET_NOT_FOUND", "resource file was not found", selected)
	}
	root, err := pathx.Canonicalize(game.ResourceDir)
	if err != nil {
		return "", blitz.IOError("could not resolve resource directory", game.ResourceDir, err)
	}
	resolved, err := pathx.Canonicalize(selected)
	if err != nil {
		return "", blitz.IOError("could not resolve resource file", selected, err)
	}
	if !pathx.HasPrefix(resolved, root) {
		return "", blitz.ErrorAt("TARGET_OUTSIDE_GAME_DIR", "resolved resource escapes the game directory", resolved)
	}
	return resolved, nil
}

func operationFromEntry(entry *ModFile) (storedOperation, error) {
	operation := storedOperation{
		Action:                   entry.Action,
		Find:                     entry.Find,
		Replace:                  entry.Replace,
		IgnoreTrailingWhitespace: entry.IgnoreTrailingWhitespace,
	}
	if entry.Action == ActionPatch {
		if entry.PatchDiff == nil {
			return storedOperation{}, blitz.NewError("MANIFEST_INVALID_ACTION", "patch operation has no resolved diff")
		}
		operation.PatchDiff = entry.PatchDiff
	}
	return operation, nil
}

func composeResource(target string, base []byte, operations []modOperation) ([]byte, error) {
	isDVPL := strings.HasSuffix(lowerKey(target), ".dvpl")
	plain, kind, err := resources.DecodeDVPL(base)
	if err != nil {
		return nil, err
	}
	if isDVPL && !kind.DVPL {
		return nil, blitz.ErrorAt("DVPL_MAGIC_INVALID", "file has a .dvpl suffix but no valid DVPL footer", target)
	}
	output := plain
	for _, item := range operations {
		var result []byte
		var err error
		if item.operation.Action == ActionClear {
			result = []byte{}
		} else if !utf8.Valid(output) {
			err = blitz.ErrorAt("RESOURCE_NOT_UTF8", "resource is not UTF-8 text: "+rs.UTF8Error(output), target)
		} else {
			var text string
			text, err = applyOperation(string(output), &item.operation, target)
			result = []byte(text)
		}
		if err != nil {
			if len(operations) > 1 {
				return nil, blitz.ErrorAt("MOD_CONFLICT",
					fmt.Sprintf("could not compose mod %s: %s", item.modID, blitz.AsError(err).Message), target)
			}
			return nil, err
		}
		output = result
	}
	return resources.EncodeDVPL(output, kind)
}

func applyOperation(source string, operation *storedOperation, target string) (string, error) {
	switch operation.Action {
	case ActionReplace, ActionBefore, ActionAfter:
		find, replacement := "", ""
		if operation.Find != nil {
			find = *operation.Find
		}
		if operation.Replace != nil {
			replacement = *operation.Replace
		}
		if find == "" {
			return "", blitz.ErrorAt("REPLACE_EMPTY_FIND", "find must not be empty", target)
		}
		matches := findMatches(source, find, operation.IgnoreTrailingWhitespace)
		if len(matches) != 1 {
			kind := "exact "
			if operation.IgnoreTrailingWhitespace {
				kind = "normalized "
			}
			return "", blitz.ErrorAt("REPLACE_MATCH_COUNT", fmt.Sprintf("expected one %smatch, found %d", kind, len(matches)), target)
		}
		match := matches[0]
		matched := source[match.start:match.end]
		switch operation.Action {
		case ActionBefore:
			replacement += matched
		case ActionAfter:
			replacement = matched + replacement
		}
		return source[:match.start] + replacement + source[match.end:], nil
	case ActionClear:
		return "", nil
	case ActionPatch:
		if operation.PatchDiff == nil {
			return "", blitz.ErrorAt("MANIFEST_INVALID_ACTION", "patch action has no patch content", target)
		}
		output, err := applyUnifiedDiff(source, *operation.PatchDiff)
		if err != nil {
			typed := *blitz.AsError(err)
			if typed.Path == "" {
				typed.Path = target
			}
			return "", &typed
		}
		return output, nil
	}
	return "", blitz.ErrorAt("MANIFEST_INVALID_ACTION", "unsupported action "+string(operation.Action), target)
}

func operationsForTarget(state *installerState, target string) []modOperation {
	var out []modOperation
	for _, item := range state.Mods {
		for _, file := range item.Files {
			if samePath(file.Target, target) {
				out = append(out, modOperation{item.ID, file.Operation})
			}
		}
	}
	sortByMod(out)
	return out
}

func managedResource(state *installerState, target string) (string, uint32, bool, error) {
	var snapshot string
	var crc uint32
	found := false
	for _, item := range state.Mods {
		for _, file := range item.Files {
			if !samePath(file.Target, target) {
				continue
			}
			if !found {
				snapshot, crc, found = file.BaseSnapshot, file.InstalledCRC32, true
			} else if !samePath(file.BaseSnapshot, snapshot) || file.InstalledCRC32 != crc {
				return "", 0, false, blitz.ErrorAt("STATE_INVALID", "mods disagree about the managed resource state", target)
			}
		}
	}
	return snapshot, crc, found, nil
}

func baseSnapshotPath(game *client.ResolvedGameDir, target string) (string, error) {
	root, err := pathx.Canonicalize(game.ResourceDir)
	if err != nil {
		return "", blitz.IOError("could not resolve resource directory", game.ResourceDir, err)
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || !pathx.HasPrefix(target, root) {
		return "", blitz.ErrorAt("TARGET_OUTSIDE_GAME_DIR", "resource is outside the game resource directory", target)
	}
	if relative == "." {
		return filepath.Join(game.GameDir, stateDir, "bases"), nil
	}
	return filepath.Join(game.GameDir, stateDir, "bases", relative), nil
}

func readBaseSnapshot(game *client.ResolvedGameDir, target, snapshot string) ([]byte, error) {
	expected, err := baseSnapshotPath(game, target)
	if err != nil {
		return nil, err
	}
	if !samePath(snapshot, expected) {
		return nil, blitz.ErrorAt("STATE_INVALID_BASE", "installed state base snapshot path is invalid", snapshot)
	}
	info, err := os.Lstat(snapshot)
	if err != nil {
		return nil, blitz.IOError("could not inspect base snapshot", snapshot, err)
	}
	if !info.Mode().IsRegular() {
		return nil, blitz.ErrorAt("BACKUP_INVALID", "base snapshot must be a regular file, not a symlink", snapshot)
	}
	data, err := os.ReadFile(snapshot)
	if err != nil {
		return nil, blitz.IOError("could not read base snapshot", snapshot, err)
	}
	return data, nil
}

// inspectAdjacentOriginal returns true when the adjacent original should be created.
func inspectAdjacentOriginal(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		return true, nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		return false, blitz.IOError("could not inspect adjacent original", path, err)
	}
	if !info.Mode().IsRegular() {
		return false, blitz.ErrorAt("BACKUP_INVALID", "adjacent original must be a regular file, not a symlink", path)
	}
	return false, nil
}

func samePath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return lowerKey(left) == lowerKey(right)
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func statePathOf(gameDir string) string {
	return filepath.Join(gameDir, stateDir, "state.json")
}

func loadState(gameDir string) (*installerState, error) {
	path := statePathOf(gameDir)
	if _, err := os.Stat(path); err != nil {
		return &installerState{Mods: []installedMod{}}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, blitz.IOError("could not read installer state", path, err)
	}
	var object map[string]json.RawMessage
	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		return nil, blitz.ErrorAt("STATE_INVALID", err.Error(), path)
	}
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return nil, blitz.ErrorAt("STATE_INVALID", "installer state must be an object", path)
	}
	version := uint64(0)
	if raw, ok := object["version"]; ok {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var number any
		decoder.Decode(&number)
		text, isNumber := number.(json.Number)
		parsed, err := rs.ParseUint(string(text), 64)
		if !isNumber || err != nil {
			return nil, blitz.ErrorAt("STATE_INVALID", "state version must be a non-negative integer", path)
		}
		version = parsed
	}
	hasMods := false
	if raw, ok := object["mods"]; ok {
		var mods []json.RawMessage
		if err := json.Unmarshal(raw, &mods); err != nil || mods == nil {
			return nil, blitz.ErrorAt("STATE_INVALID", "state mods must be an array", path)
		}
		hasMods = len(mods) > 0
	}
	if version != 2 {
		if !hasMods && version <= 1 {
			return &installerState{Mods: []installedMod{}}, nil
		}
		return nil, blitz.ErrorAt("STATE_VERSION_UNSUPPORTED", fmt.Sprintf(
			"installer state version %d cannot be recomposed; uninstall its mods with the previous installer version first", version), path)
	}
	state, err := decodeState(data)
	if err != nil {
		return nil, blitz.ErrorAt("STATE_INVALID", err.Error(), path)
	}
	return state, nil
}

// decodeState decodes a version-2 state, enforcing serde's required fields
// and enum variants.
func decodeState(data []byte) (*installerState, error) {
	var raw struct {
		Version *uint32 `json:"version"`
		Mods    []struct {
			ID    *string `json:"id"`
			Name  *string `json:"name"`
			Files *[]struct {
				LogicalPath  *string `json:"logical_path"`
				Target       *string `json:"target"`
				BaseSnapshot *string `json:"base_snapshot"`
				Operation    *struct {
					Action                   *string `json:"action"`
					Find                     *string `json:"find"`
					Replace                  *string `json:"replace"`
					PatchDiff                *string `json:"patch_diff"`
					IgnoreTrailingWhitespace bool    `json:"ignore_trailing_whitespace"`
				} `json:"operation"`
				InstalledCRC32 *uint32 `json:"installed_crc32"`
			} `json:"files"`
		} `json:"mods"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	missing := func(field string) error { return fmt.Errorf("missing field `%s`", field) }
	state := &installerState{Version: 2, Mods: []installedMod{}}
	for _, item := range raw.Mods {
		switch {
		case item.ID == nil:
			return nil, missing("id")
		case item.Name == nil:
			return nil, missing("name")
		case item.Files == nil:
			return nil, missing("files")
		}
		mod := installedMod{ID: *item.ID, Name: *item.Name, Files: []installedFile{}}
		for _, file := range *item.Files {
			switch {
			case file.LogicalPath == nil:
				return nil, missing("logical_path")
			case file.Target == nil:
				return nil, missing("target")
			case file.BaseSnapshot == nil:
				return nil, missing("base_snapshot")
			case file.Operation == nil:
				return nil, missing("operation")
			case file.InstalledCRC32 == nil:
				return nil, missing("installed_crc32")
			case file.Operation.Action == nil:
				return nil, missing("action")
			}
			action := Action(*file.Operation.Action)
			switch action {
			case ActionReplace, ActionBefore, ActionAfter, ActionClear, ActionPatch:
			default:
				return nil, fmt.Errorf("unknown variant `%s`, expected one of `replace`, `before`, `after`, `clear`, `patch`", action)
			}
			mod.Files = append(mod.Files, installedFile{
				LogicalPath:  *file.LogicalPath,
				Target:       *file.Target,
				BaseSnapshot: *file.BaseSnapshot,
				Operation: storedOperation{
					Action: action, Find: file.Operation.Find, Replace: file.Operation.Replace,
					PatchDiff: file.Operation.PatchDiff, IgnoreTrailingWhitespace: file.Operation.IgnoreTrailingWhitespace,
				},
				InstalledCRC32: *file.InstalledCRC32,
			})
		}
		state.Mods = append(state.Mods, mod)
	}
	return state, nil
}

func saveState(path string, state *installerState) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o777); err != nil {
		return blitz.IOError("could not create installer state directory", parent, err)
	}
	if state.Mods == nil {
		state.Mods = []installedMod{}
	}
	data, err := jsonx.MarshalIndent(state)
	if err != nil {
		return blitz.NewError("STATE_WRITE_FAILED", err.Error())
	}
	return replaceFile(path, data)
}

func createBackup(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return blitz.IOError("could not create backup directory", filepath.Dir(path), err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return blitz.IOError("could not create original backup without overwriting an existing file", path, err)
	}
	_, err = file.Write(data)
	if err == nil {
		err = file.Sync()
	}
	file.Close()
	if err != nil {
		os.Remove(path)
		return blitz.IOError("could not write original backup", path, err)
	}
	return nil
}

func replaceFile(path string, data []byte) error {
	parent := filepath.Dir(path)
	temp := filepath.Join(parent, fmt.Sprintf(".%s.bc-%d-%d.tmp", filepath.Base(path), os.Getpid(), time.Now().UnixNano()))
	file, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return blitz.IOError("could not create temporary file", temp, err)
	}
	_, err = file.Write(data)
	if err == nil {
		err = file.Sync()
	}
	file.Close()
	if err != nil {
		os.Remove(temp)
		return blitz.IOError("could not write temporary file", temp, err)
	}
	if err := os.Rename(temp, path); err != nil {
		os.Remove(temp)
		return blitz.IOError("could not move temporary file into place", path, err)
	}
	return nil
}

type committedFile struct {
	path string
	data []byte
}

func rollbackTargets(committed []committedFile) {
	for index := len(committed) - 1; index >= 0; index-- {
		replaceFile(committed[index].path, committed[index].data)
	}
}
