// Package mods loads mod manifests and validates, installs, and removes them.
// It also generates manifest patches from Git working-tree changes.
package mods

import (
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/pathx"
	"github.com/thecoooler/blitz-client/internal/rs"
	"github.com/thecoooler/blitz-client/internal/yamlv"
	"go.yaml.in/yaml/v3"
)

// Action is a manifest operation kind.
type Action string

const (
	ActionReplace Action = "replace"
	ActionBefore  Action = "before"
	ActionAfter   Action = "after"
	ActionClear   Action = "clear"
	ActionPatch   Action = "patch"
)

// debugName formats the action like the Rust enum's Debug output.
func (a Action) debugName() string {
	if a == "" {
		return ""
	}
	return strings.ToUpper(string(a[:1])) + string(a[1:])
}

// Manifest is a loaded mod with its patch sources expanded into operations.
type Manifest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Patch lists patch files and directories in their declared order.
	Patch []string  `json:"patch"`
	Files []ModFile `json:"files"`
}

// ModFile is one operation on a logical resource path.
type ModFile struct {
	// Name is relative to the game's resource root; DVPL files omit ".dvpl".
	Name    string  `json:"name"`
	Action  Action  `json:"action"`
	Find    *string `json:"find,omitempty"`
	Replace *string `json:"replace,omitempty"`
	Patch   *string `json:"patch,omitempty"`
	// PatchDiff is the resolved single-resource diff; never serialized.
	PatchDiff                *string `json:"-"`
	IgnoreTrailingWhitespace bool    `json:"ignore-trailing-whitespace"`
}

func isFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func locateModDir(p string) string {
	if isFile(p) {
		return pathx.Parent(p)
	}
	return p
}

// dirOrDot treats Rust's empty parent as the current directory for I/O.
func dirOrDot(dir string) string {
	if dir == "" {
		return "."
	}
	return dir
}

func manifestFile(p string) (string, error) {
	if isFile(p) && !isYAML(p) {
		return "", blitz.ErrorAt("MANIFEST_FORMAT_UNSUPPORTED", "only YAML mod manifests are supported", p)
	}
	dir := locateModDir(p)
	var candidates []string
	for _, name := range []string{"mod.yaml", "mod.yml"} {
		if candidate := pathx.Join(dir, name); isFile(candidate) {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) > 1 {
		return "", blitz.ErrorAt("MANIFEST_AMBIGUOUS", "multiple mod manifests in the same directory", dir)
	}
	if isFile(p) {
		return p, nil
	}
	if len(candidates) == 0 {
		return "", blitz.ErrorAt("MANIFEST_NOT_FOUND", "no mod.yaml or mod.yml in the directory", p)
	}
	return candidates[0], nil
}

func isYAML(p string) bool {
	ext, ok := pathx.Extension(p)
	return ok && (ext == "yaml" || ext == "yml")
}

var manifestFields = []string{"version", "id", "name", "patch", "files"}
var operationFields = []string{"search", "insert", "replace", "ignore-trailing-whitespace"}

func expectedOneOf(fields []string) string {
	quoted := make([]string, len(fields))
	for index, field := range fields {
		quoted[index] = "`" + field + "`"
	}
	return strings.Join(quoted, ", ")
}

type rawManifest struct {
	version  uint32
	id, name string
	patch    []string
	files    *yamlv.Value
	hasID    bool
	hasName  bool
}

// decodeManifest mirrors serde's derived deserializer for the YAML manifest,
// including its diagnostics.
func decodeManifest(text []byte) (*rawManifest, error) {
	root, err := yamlv.ParseDocument(text)
	if err != nil {
		return nil, err
	}
	raw := &rawManifest{version: 1}
	if root == nil {
		return nil, fmt.Errorf("missing field `id`")
	}
	if root.Kind != yaml.MappingNode {
		value, _ := yamlv.Build(root, "")
		return nil, fmt.Errorf("invalid type: %s, expected struct YamlManifest", value.Unexpected())
	}
	seen := map[string]bool{}
	for index := 0; index+1 < len(root.Content); index += 2 {
		keyNode, valueNode := root.Content[index], resolve(root.Content[index+1])
		key := keyNode.Value
		if !slices.Contains(manifestFields, key) {
			return nil, fmt.Errorf("unknown field `%s`, expected one of %s%s", key, expectedOneOf(manifestFields), yamlv.Position(keyNode))
		}
		if seen[key] {
			return nil, fmt.Errorf("duplicate field `%s`", key)
		}
		seen[key] = true
		fieldError := func(message string) error {
			return fmt.Errorf("%s: %s%s", key, message, yamlv.Position(valueNode))
		}
		switch key {
		case "version":
			version, err := decodeU32(valueNode)
			if err != nil {
				return nil, fieldError(err.Error())
			}
			raw.version = version
		case "id", "name":
			text, err := decodeDirectString(valueNode)
			if err != nil {
				return nil, fieldError(err.Error())
			}
			if key == "id" {
				raw.id, raw.hasID = text, true
			} else {
				raw.name, raw.hasName = text, true
			}
		case "patch":
			value, err := yamlv.Build(valueNode, key)
			if err != nil {
				return nil, err
			}
			if raw.patch, err = decodePatchSources(value); err != nil {
				return nil, err
			}
		case "files":
			value, err := yamlv.Build(valueNode, key)
			if err != nil {
				return nil, err
			}
			if value.Kind != yamlv.Map {
				return nil, fieldError(fmt.Sprintf("invalid type: %s, expected a YAML mapping", value.Unexpected()))
			}
			raw.files = value
		}
	}
	if !raw.hasID {
		return nil, fmt.Errorf("missing field `id`")
	}
	if !raw.hasName {
		return nil, fmt.Errorf("missing field `name`")
	}
	return raw, nil
}

func resolve(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}

func decodeU32(node *yaml.Node) (uint32, error) {
	value, err := yamlv.Build(node, "")
	if err != nil {
		return 0, err
	}
	if value.Kind == yamlv.Int && !value.Neg {
		if value.Uint > 1<<32-1 {
			return 0, fmt.Errorf("invalid value: %s, expected u32", value.Unexpected())
		}
		return uint32(value.Uint), nil
	}
	return 0, fmt.Errorf("invalid type: %s, expected u32", value.Unexpected())
}

// decodeDirectString accepts any scalar's text, as serde_yaml_ng does when a
// string is deserialized straight from the document.
func decodeDirectString(node *yaml.Node) (string, error) {
	if node == nil {
		return "", nil
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Value, nil
	case yaml.SequenceNode:
		return "", fmt.Errorf("invalid type: sequence, expected a string")
	default:
		return "", fmt.Errorf("invalid type: map, expected a string")
	}
}

func decodePatchSources(value *yamlv.Value) ([]string, error) {
	untagged := fmt.Errorf("data did not match any variant of untagged enum PatchSources")
	switch value.Kind {
	case yamlv.Null:
		return []string{}, nil
	case yamlv.String:
		return []string{value.Str}, nil
	case yamlv.Seq:
		out := []string{}
		for _, item := range value.Items {
			if item.Kind != yamlv.String {
				return nil, untagged
			}
			out = append(out, item.Str)
		}
		return out, nil
	}
	return nil, untagged
}

func valueString(value *yamlv.Value) (*string, error) {
	switch value.Kind {
	case yamlv.Null:
		return nil, nil
	case yamlv.String:
		text := value.Str
		return &text, nil
	}
	return nil, fmt.Errorf("invalid type: %s, expected a string", value.Unexpected())
}

type yamlOperation struct {
	search, replace *string
	ignore          bool
}

func decodeOperation(value *yamlv.Value) (*yamlOperation, error) {
	operation := &yamlOperation{}
	if value.Kind == yamlv.Null {
		return operation, nil
	}
	if value.Kind != yamlv.Map {
		return nil, fmt.Errorf("invalid type: %s, expected struct YamlOperation", value.Unexpected())
	}
	seen := map[string]bool{}
	for index, keyValue := range value.Keys {
		key := keyValue.Str
		if keyValue.Kind != yamlv.String {
			key = keyValue.Raw
		}
		field := key
		if field == "insert" {
			field = "replace"
		}
		if !slices.Contains(operationFields, key) {
			return nil, fmt.Errorf("unknown field `%s`, expected one of %s", key, expectedOneOf(operationFields))
		}
		if seen[field] {
			return nil, fmt.Errorf("duplicate field `%s`", field)
		}
		seen[field] = true
		item := value.Vals[index]
		var err error
		switch field {
		case "search":
			operation.search, err = valueString(item)
		case "replace":
			operation.replace, err = valueString(item)
		default:
			if item.Kind != yamlv.Bool {
				err = fmt.Errorf("invalid type: %s, expected a boolean", item.Unexpected())
			}
			operation.ignore = item.Bool
		}
		if err != nil {
			return nil, err
		}
	}
	return operation, nil
}

type actionEntry struct {
	keys       []string
	operations map[string]*yamlOperation
}

func decodeActions(value *yamlv.Value) ([]actionEntry, error) {
	switch value.Kind {
	case yamlv.Null:
		return nil, nil
	case yamlv.Seq:
	default:
		return nil, fmt.Errorf("invalid type: %s, expected a sequence", value.Unexpected())
	}
	var out []actionEntry
	for _, item := range value.Items {
		if item.Kind == yamlv.Null {
			out = append(out, actionEntry{operations: map[string]*yamlOperation{}})
			continue
		}
		if item.Kind != yamlv.Map {
			return nil, fmt.Errorf("invalid type: %s, expected a map", item.Unexpected())
		}
		entry := actionEntry{operations: map[string]*yamlOperation{}}
		for index, key := range item.Keys {
			if key.Kind != yamlv.String {
				return nil, fmt.Errorf("invalid type: %s, expected a string", key.Unexpected())
			}
			operation, err := decodeOperation(item.Vals[index])
			if err != nil {
				return nil, err
			}
			if _, exists := entry.operations[key.Str]; !exists {
				entry.keys = append(entry.keys, key.Str)
			}
			entry.operations[key.Str] = operation
		}
		out = append(out, entry)
	}
	return out, nil
}

func parseManifest(text []byte, manifestPath string) (*Manifest, error) {
	invalid := func(err error) error { return blitz.ErrorAt("MANIFEST_INVALID", err.Error(), manifestPath) }
	raw, err := decodeManifest(text)
	if err != nil {
		return nil, invalid(err)
	}
	if raw.version != 1 {
		return nil, invalid(fmt.Errorf("unsupported manifest version: %d", raw.version))
	}
	manifest := &Manifest{ID: raw.id, Name: raw.name, Patch: raw.patch, Files: []ModFile{}}
	if manifest.Patch == nil {
		manifest.Patch = []string{}
	}
	if raw.files == nil {
		return manifest, nil
	}
	for index, key := range raw.files.Keys {
		if key.Kind != yamlv.String {
			return nil, invalid(fmt.Errorf("file path must be a string"))
		}
		name := key.Str
		actions, err := decodeActions(raw.files.Vals[index])
		if err != nil {
			return nil, invalid(err)
		}
		if len(actions) == 0 {
			return nil, invalid(fmt.Errorf("file %s has no operations", name))
		}
		for _, entry := range actions {
			if len(entry.keys) != 1 {
				return nil, invalid(fmt.Errorf("each operation must contain exactly one action"))
			}
			actionName := entry.keys[0]
			operation := entry.operations[actionName]
			var action Action
			switch actionName {
			case "replace", "before", "after", "clear":
				action = Action(actionName)
			case "patch":
				return nil, invalid(fmt.Errorf("patch must be declared at the top level of the manifest"))
			default:
				return nil, invalid(fmt.Errorf("unknown action: %s", actionName))
			}
			manifest.Files = append(manifest.Files, ModFile{
				Name:                     name,
				Action:                   action,
				Find:                     operation.search,
				Replace:                  operation.replace,
				IgnoreTrailingWhitespace: operation.ignore,
			})
		}
	}
	return manifest, nil
}

// LoadManifest reads and validates a manifest, expanding its patch sources
// into per-resource operations. path may be a mod directory or manifest file.
func LoadManifest(p string) (*Manifest, error) {
	manifestPath, err := manifestFile(p)
	if err != nil {
		return nil, err
	}
	text, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, blitz.IOError("could not read mod manifest", manifestPath, err)
	}
	if !utf8Valid(text) {
		return nil, blitz.IOError("could not read mod manifest", manifestPath, errInvalidUTF8)
	}
	manifest, err := parseManifest(text, manifestPath)
	if err != nil {
		return nil, err
	}
	if err := validateManifest(manifest, manifestPath); err != nil {
		return nil, err
	}
	if err := expandPatches(manifest, locateModDir(manifestPath), nil); err != nil {
		return nil, err
	}
	return manifest, nil
}

func validateManifest(manifest *Manifest, manifestPath string) error {
	validID := strings.TrimSpace(manifest.ID) != ""
	for _, c := range manifest.ID {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			validID = false
		}
	}
	if !validID {
		return blitz.ErrorAt("MANIFEST_INVALID_ID", "id must contain only ASCII letters, digits, '-' or '_'", manifestPath)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return blitz.ErrorAt("MANIFEST_INVALID_NAME", "name must not be empty", manifestPath)
	}
	if len(manifest.Files) == 0 && len(manifest.Patch) == 0 {
		return blitz.ErrorAt("MANIFEST_EMPTY", "mod has no file operations", manifestPath)
	}
	for _, source := range manifest.Patch {
		if source != "." {
			if _, err := safeRelativePath(source); err != nil {
				return blitz.ErrorAt("MANIFEST_INVALID_PATCH_PATH", err.Error(), manifestPath)
			}
		}
	}
	for _, file := range manifest.Files {
		if _, err := safeRelativePath(file.Name); err != nil {
			return blitz.ErrorAt("MANIFEST_INVALID_PATH", err.Error(), manifestPath)
		}
		textual := file.Action == ActionReplace || file.Action == ActionBefore || file.Action == ActionAfter
		switch {
		case textual && (file.Find == nil || *file.Find == "" || file.Replace == nil):
			return blitz.ErrorAt("MANIFEST_INVALID_ACTION", fmt.Sprintf(
				"%s action for %s requires non-empty find and present replace", file.Action.debugName(), file.Name), manifestPath)
		case file.Action == ActionPatch && file.Patch == nil:
			return blitz.ErrorAt("MANIFEST_INVALID_ACTION",
				fmt.Sprintf("patch action for %s requires a patch path", file.Name), manifestPath)
		}
		if file.IgnoreTrailingWhitespace {
			if !textual {
				return blitz.ErrorAt("MANIFEST_INVALID_ACTION",
					"ignore-trailing-whitespace is only supported for replace/before/after", manifestPath)
			}
			find := ""
			if file.Find != nil {
				find = *file.Find
			}
			if strings.Trim(find, " \t\r\n") == "" {
				return blitz.ErrorAt("MANIFEST_INVALID_ACTION", "normalized find must contain non-whitespace text", manifestPath)
			}
		}
		if file.Patch != nil {
			if _, err := safeRelativePath(*file.Patch); err != nil {
				return blitz.ErrorAt("MANIFEST_INVALID_PATCH_PATH", err.Error(), manifestPath)
			}
		}
	}
	return nil
}

// safeRelativePath normalizes a portable relative path, rejecting absolute
// paths, traversal, drive prefixes, and NUL bytes. The result uses "/".
func safeRelativePath(p string) (string, error) {
	if p == "" || strings.ContainsAny(p, "\x00:") {
		return "", fmt.Errorf("invalid relative path: %s", rs.Quote(p))
	}
	portable := strings.ReplaceAll(p, `\`, "/")
	if strings.HasPrefix(portable, "/") {
		return "", fmt.Errorf("absolute paths are not allowed: %s", p)
	}
	var parts []string
	for _, part := range strings.Split(portable, "/") {
		switch part {
		case "", ".":
		case "..":
			return "", fmt.Errorf("path traversal is not allowed: %s", p)
		default:
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("path must not be empty")
	}
	return path.Join(parts...), nil
}

var errInvalidUTF8 = errors.New("stream did not contain valid UTF-8")

func utf8Valid(data []byte) bool { return utf8.Valid(data) }

func expandPatches(manifest *Manifest, modDir string, generated *string) error {
	operations, err := loadPatches(modDir, manifest.Patch, generated)
	if err != nil {
		return err
	}
	manifest.Files = append(operations, manifest.Files...)
	return nil
}

// UpdateFromGitDiff writes all modifications in repo since baseRevision (nil
// means HEAD) to one mod.patch and makes it the first patch source,
// preserving manual sources and operations.
func UpdateFromGitDiff(modPath, repo string, baseRevision *string) (*Manifest, error) {
	manifestPath, err := manifestFile(modPath)
	if err != nil {
		return nil, err
	}
	modDir := locateModDir(manifestPath)
	rawText, err := os.ReadFile(manifestPath)
	if err == nil && !utf8Valid(rawText) {
		err = errInvalidUTF8
	}
	if err != nil {
		return nil, blitz.IOError("could not read mod manifest", manifestPath, err)
	}
	manifest, err := parseManifest(rawText, manifestPath)
	if err != nil {
		return nil, err
	}
	sources := []string{"mod.patch"}
	for _, source := range manifest.Patch {
		if safe, err := safeRelativePath(source); source == "." || err != nil || safe != "mod.patch" {
			sources = append(sources, source)
		}
	}
	manifest.Patch = sources
	if err := validateManifest(manifest, manifestPath); err != nil {
		return nil, err
	}
	base := "HEAD"
	if baseRevision != nil {
		base = *baseRevision
	}
	changes, err := collectChanges(repo, base)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, blitz.NewError("GIT_DIFF_EMPTY", "no modified text resources to generate mod.patch")
	}
	generated := strings.Join(changes, "")
	// Resolve every source before writing, substituting the proposed generated diff.
	if err := expandPatches(manifest, modDir, &generated); err != nil {
		return nil, err
	}
	node, err := yamlv.ParseDocument(rawText)
	if err != nil {
		return nil, blitz.ErrorAt("MANIFEST_INVALID", err.Error(), manifestPath)
	}
	document, err := yamlv.Build(node, "")
	if err != nil {
		return nil, blitz.ErrorAt("MANIFEST_INVALID", err.Error(), manifestPath)
	}
	if len(manifest.Patch) == 1 {
		document.Set("patch", yamlv.NewString(manifest.Patch[0]))
	} else {
		document.Set("patch", yamlv.NewSeq(manifest.Patch))
	}
	serialized, err := yamlv.Marshal(document)
	if err != nil {
		return nil, blitz.ErrorAt("MANIFEST_INVALID", err.Error(), manifestPath)
	}
	patchPath := pathx.Join(modDir, "mod.patch")
	previous, previousErr := os.ReadFile(patchPath)
	if err := writeReplacing(patchPath, []byte(generated)); err != nil {
		return nil, err
	}
	if err := writeReplacing(manifestPath, serialized); err != nil {
		if previousErr == nil {
			writeReplacing(patchPath, previous)
		} else {
			os.Remove(patchPath)
		}
		return nil, err
	}
	return manifest, nil
}

func writeReplacing(target string, data []byte) error {
	// Like Path::with_extension("bc-tmp").
	temp := target + ".bc-tmp"
	if ext, ok := pathx.Extension(target); ok {
		temp = target[:len(target)-len(ext)] + "bc-tmp"
	}
	if err := os.WriteFile(temp, data, 0o666); err != nil {
		return blitz.IOError("could not write temporary manifest", temp, err)
	}
	if err := os.Rename(temp, target); err != nil {
		os.Remove(temp)
		return blitz.IOError("could not replace mod manifest", target, err)
	}
	return nil
}
