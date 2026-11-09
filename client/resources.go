package client

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/pathx"
	"github.com/thecoooler/blitz-client/resources"
)

// Resources is the logical view of an installed client's Data directory with
// an optional cache overlay. Cache files win over installation files, a plain
// file wins over its .dvpl sibling, and DVPL payloads are returned decoded.
type Resources struct {
	dataDir  string
	cacheDir string
	caseFold bool
	entries  map[string]resourceEntry
}

type resourceEntry struct {
	// logical is the path relative to the resource root, using forward slashes
	// and without a .dvpl suffix.
	logical string
	// path is the physical file selected for the entry.
	path string
	// packed reports whether path is a DVPL container.
	packed bool
}

// OpenResources opens a client resource tree. dataDir is required; cacheDir is
// an optional overlay of downloaded packs.
func OpenResources(dataDir, cacheDir string) (*Resources, error) {
	data, err := resourceRoot(dataDir, "Data directory")
	if err != nil {
		return nil, err
	}
	cache := ""
	if cacheDir != "" {
		if cache, err = resourceRoot(cacheDir, "cache directory"); err != nil {
			return nil, err
		}
	}
	return openResources(data, cache, CaseInsensitiveFS())
}

func resourceRoot(directory, label string) (string, error) {
	info, err := os.Stat(directory)
	if err != nil {
		return "", blitz.IOError("could not resolve client resource "+label, directory, err)
	}
	if !info.IsDir() {
		return "", blitz.ErrorAt("CLIENT_RESOURCE_ROOT_INVALID", "client resource "+label+" is not a directory", directory)
	}
	return directory, nil
}

// openResources builds the resource index with an explicit case rule. Tests use
// it to exercise the Windows and macOS rule on any host.
func openResources(dataDir, cacheDir string, caseFold bool) (*Resources, error) {
	r := &Resources{
		dataDir:  dataDir,
		cacheDir: cacheDir,
		caseFold: caseFold,
		entries:  map[string]resourceEntry{},
	}
	if err := r.index(dataDir); err != nil {
		return nil, err
	}
	if cacheDir != "" {
		if err := r.index(cacheDir); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// index merges one physical root into the logical index. Later roots override
// earlier ones, which is how the cache overlay wins over Data.
func (r *Resources) index(root string) error {
	local := map[string]resourceEntry{}
	plainSpellings := map[string]string{}
	packedSpellings := map[string]string{}
	var collision error
	err := filepath.WalkDir(root, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if file == root {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			// Symlinks could escape the root; the existing file adapters skip
			// them, so the logical view does too.
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		logical, ok := normalizeLogical(filepath.ToSlash(relative))
		if !ok {
			return nil
		}
		key := foldLogical(logical, r.caseFold)
		next := resourceEntry{logical: logical, path: file, packed: isPacked(relative)}
		// Two plain files or two .dvpl containers that differ only in case
		// collide on a case-insensitive view. A plain file and its .dvpl
		// sibling are the same resource.
		spellings := plainSpellings
		if next.packed {
			spellings = packedSpellings
		}
		if previous, exists := spellings[key]; exists && previous != logical {
			collision = blitz.ErrorAt("CLIENT_RESOURCE_COLLISION",
				"client resources differ only in case: "+previous+" and "+logical, file)
			return fs.SkipAll
		}
		spellings[key] = logical
		if previous, exists := local[key]; exists {
			// A plain file wins over its .dvpl sibling within one root.
			if previous.packed && !next.packed {
				local[key] = next
			}
			return nil
		}
		local[key] = next
		return nil
	})
	if collision != nil {
		return collision
	}
	if err != nil {
		return blitz.IOError("could not scan client resources", root, err)
	}
	for key, entry := range local {
		r.entries[key] = entry
	}
	return nil
}

// Open resolves a logical path and returns its plain payload, decoding DVPL.
func (r *Resources) Open(relative string) ([]byte, error) {
	entry, ok := r.lookup(relative)
	if !ok {
		if _, valid := normalizeLogical(relative); !valid {
			return nil, blitz.ErrorAt("CLIENT_RESOURCE_INVALID_PATH",
				"client resource path must stay inside the resource root", relative)
		}
		return nil, blitz.ErrorAt("CLIENT_DATA_MISSING", "required client resource is missing", relative)
	}
	return decodeResourceFile(entry.path)
}

// Exists reports whether a logical resource exists in Data or the cache.
func (r *Resources) Exists(relative string) bool {
	_, ok := r.lookup(relative)
	return ok
}

// Path returns the physical file selected for a logical resource.
func (r *Resources) Path(relative string) (string, bool) {
	entry, ok := r.lookup(relative)
	if !ok {
		return "", false
	}
	return entry.path, true
}

// IsDir reports whether a logical directory exists in Data or the cache.
func (r *Resources) IsDir(relative string) bool {
	logical, ok := normalizeLogical(relative)
	if !ok {
		return false
	}
	for _, root := range []string{r.dataDir, r.cacheDir} {
		if root == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(logical)))
		if err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// List returns the sorted logical file paths below prefix. Paths are relative
// to Data, use forward slashes, and omit the .dvpl suffix. The cache overlay is
// included, with cache files replacing their Data counterparts.
func (r *Resources) List(prefix string) ([]string, error) {
	normalized, ok := normalizePrefix(prefix)
	if !ok {
		return nil, blitz.ErrorAt("CLIENT_RESOURCE_INVALID_PATH",
			"client resource path must stay inside the resource root", prefix)
	}
	result := []string{}
	for _, entry := range r.entries {
		if !pathHasPrefix(entry.logical, normalized) {
			continue
		}
		result = append(result, entry.logical)
	}
	slices.SortFunc(result, pathx.Compare)
	return result, nil
}

func (r *Resources) lookup(relative string) (resourceEntry, bool) {
	key, ok := LogicalResourceKey(relative, r.caseFold)
	if !ok {
		return resourceEntry{}, false
	}
	entry, found := r.entries[key]
	return entry, found
}

// CaseInsensitiveFS reports whether the host filesystem matches resource paths
// without regard to case.
func CaseInsensitiveFS() bool {
	return runtime.GOOS == "windows" || runtime.GOOS == "darwin"
}

// LogicalResourceKey normalizes a logical client path for lookup. It returns a
// cleaned forward-slash path without a final .dvpl suffix, folded to lower case
// when fold is true. It reports false for paths that escape the resource root.
func LogicalResourceKey(relative string, fold bool) (string, bool) {
	logical, ok := normalizeLogical(relative)
	if !ok {
		return "", false
	}
	return foldLogical(logical, fold), true
}

func foldLogical(logical string, fold bool) string {
	if fold {
		return strings.ToLower(logical)
	}
	return logical
}

func normalizeLogical(relative string) (string, bool) {
	relative = strings.ReplaceAll(relative, `\`, "/")
	if relative == "" || strings.HasPrefix(relative, "/") || strings.ContainsAny(relative, ":\x00") {
		return "", false
	}
	logical := path.Clean(relative)
	if logical == "." || logical == ".." || strings.HasPrefix(logical, "../") {
		return "", false
	}
	if strings.EqualFold(path.Ext(logical), ".dvpl") {
		logical = logical[:len(logical)-len(".dvpl")]
	}
	if logical == "" || logical == "." {
		return "", false
	}
	return logical, true
}

func normalizePrefix(prefix string) (string, bool) {
	prefix = strings.ReplaceAll(prefix, `\`, "/")
	if prefix == "" || prefix == "." || prefix == "/" {
		return "", true
	}
	if strings.HasPrefix(prefix, "/") || strings.ContainsAny(prefix, ":\x00") {
		return "", false
	}
	clean := path.Clean(prefix)
	if clean == "." {
		return "", true
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

func pathHasPrefix(logical, prefix string) bool {
	return prefix == "" || logical == prefix || strings.HasPrefix(logical, prefix+"/")
}

func isPacked(relative string) bool {
	return strings.EqualFold(path.Ext(filepath.ToSlash(relative)), ".dvpl")
}

func decodeResourceFile(file string) ([]byte, error) {
	data, _, err := readClientAssetContainer(file)
	return data, err
}

// readClientAssetContainer reads a stored resource and reports whether it was
// plain or a DVPL container before returning its decoded payload.
func readClientAssetContainer(file string) ([]byte, resources.ContainerKind, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, resources.Plain, blitz.IOError("could not read client resource", file, err)
	}
	return DecodeClientAsset(data, file)
}

// DecodeClientAsset decodes already-read resource bytes and reports whether
// they were a DVPL container.
func DecodeClientAsset(data []byte, file string) ([]byte, resources.ContainerKind, error) {
	plain, kind, err := resources.DecodeDVPL(data)
	if err != nil {
		typed := *blitz.AsError(err)
		typed.Path = file
		return nil, resources.Plain, &typed
	}
	return plain, kind, nil
}

// resolveResourceFile applies the plain-over-DVPL rule to a resource tree
// without a cache overlay. It is the legacy path-based entry point used by
// ResolveClientAsset.
func resolveResourceFile(root, relative string) (string, bool) {
	if _, ok := normalizeLogical(relative); !ok {
		return "", false
	}
	clean := path.Clean(strings.ReplaceAll(relative, `\`, "/"))
	candidates := []string{clean, clean + ".dvpl"}
	if strings.EqualFold(path.Ext(clean), ".dvpl") {
		candidates = []string{clean[:len(clean)-len(".dvpl")], clean}
	}
	for _, candidate := range candidates {
		file := filepath.Join(root, filepath.FromSlash(candidate))
		if isFile(file) {
			return file, true
		}
	}
	return "", false
}
