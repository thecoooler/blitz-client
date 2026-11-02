package export

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/resources/sprite"
	_ "golang.org/x/image/webp"
)

// iconSet is one output folder of icons collected from client directories.
// Every sprite descriptor and standalone image in a source directory becomes
// an icon named after its file; atlases referenced by descriptors do not.
// Prefixes optionally restrict the icon IDs taken from the sources, while
// excludes drop matching IDs or prefixes.
type iconSet struct {
	id       string
	sources  []string
	prefixes []string
	excludes []string
}

// includes reports whether an icon ID belongs to this set.
func (s iconSet) includes(id string) bool {
	for _, exclude := range s.excludes {
		if strings.HasPrefix(id, exclude) {
			return false
		}
	}
	if len(s.prefixes) == 0 {
		return true
	}
	for _, prefix := range s.prefixes {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}

// iconSets lists the exported groups. Event-specific folders are not included.
// The class icons are spread over three client directories and share a prefix
// that excludes the tier numbers and other lobby icons stored beside them.
// The unused SPG icons from the legacy PC engine are explicitly excluded.
var iconSets = []iconSet{
	{"shells", []string{"Gfx/Shared/tank-supply/ammunition/big"}, nil, nil},
	{"consumables", []string{"Gfx/UI/InventoryIcons/Big/Consumables"}, nil, nil},
	{"provisions", []string{"Gfx/UI/InventoryIcons/Big/Provisions"}, nil, nil},
	{"equipment", []string{"Gfx/UI/InventoryIcons/Big/OptionalDevices", "Gfx/Lobby/tank-supply/optional-devices"}, nil, nil},
	{"crew-skills", []string{"Gfx/UI/CrewSkillIcons"}, nil, nil},
	{"flags", []string{"Gfx/Lobby/flags"}, nil, nil},
	{"tank-classes", []string{
		"Gfx/Lobby/icons",
		"Gfx/Lobby/icons/tank-tiers",
		"Gfx/Lobby/icons/tank-tiers/common",
	}, []string{"icon_tank-class_", "heavy-tank_xl", "light-tank_xl", "middle-tank_xl", "pt-tank_xl"}, []string{"icon_tank-class_spg"}},
}

// existingIconSets drops source directories missing from the client and sets
// left without any sources, so uninstalled optional icon folders are not fatal.
func existingIconSets(resourceDir string, sets []iconSet) []iconSet {
	out := make([]iconSet, 0, len(sets))
	for _, set := range sets {
		var sources []string
		for _, source := range set.sources {
			if isDir(filepath.Join(resourceDir, filepath.FromSlash(source))) {
				sources = append(sources, source)
			}
		}
		if len(sources) > 0 {
			out = append(out, iconSet{id: set.id, sources: sources, prefixes: set.prefixes, excludes: set.excludes})
		}
	}
	return out
}

const iconCatalogFormatVersion = 1

// IconCatalog is written to assets/icons/catalog.json.
type IconCatalog struct {
	FormatVersion int            `json:"formatVersion"`
	Count         int            `json:"count"`
	Sets          []IconSetEntry `json:"sets"`
}

// IconSetEntry describes one icon folder.
type IconSetEntry struct {
	ID string `json:"id"`
	// Sources are client directories relative to Data.
	Sources []string    `json:"sources"`
	Count   int         `json:"count"`
	Icons   []IconEntry `json:"icons"`
}

// IconEntry describes one exported PNG.
type IconEntry struct {
	// ID is the client file name without resolution suffix and extension.
	ID string `json:"id"`
	// File is relative to the catalog directory.
	File   string `json:"file"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	// Scale is the client resolution variant: 2 for @2x resources, otherwise 1.
	Scale  int        `json:"scale"`
	Source IconSource `json:"source"`
}

// IconSource locates the client resource of an icon. Paths are relative to
// Data and omit the .dvpl suffix.
type IconSource struct {
	// Kind is "sprite" for an atlas frame or "image" for a standalone image.
	// A packed image with an embedded descriptor is a sprite whose texture is
	// the image itself.
	Kind    string     `json:"kind"`
	Path    string     `json:"path"`
	Texture string     `json:"texture,omitempty"`
	Frame   *IconFrame `json:"frame,omitempty"`
}

// IconFrame is the atlas rectangle drawn at an offset on the icon canvas.
type IconFrame struct {
	X       int `json:"x"`
	Y       int `json:"y"`
	Width   int `json:"width"`
	Height  int `json:"height"`
	OffsetX int `json:"offsetX"`
	OffsetY int `json:"offsetY"`
}

// IconExportReport summarizes the icon part of a client export.
type IconExportReport struct {
	Catalog        string `json:"catalog"`
	IconCount      int    `json:"iconCount"`
	WrittenCount   int    `json:"writtenCount"`
	UnchangedCount int    `json:"unchangedCount"`
	// RemovedCount counts icons of the previous catalog that no longer exist.
	RemovedCount int `json:"removedCount"`
}

// iconSource is a discovered icon before rendering.
type iconSource struct {
	id       string
	relative string // unpacked path relative to Data
	file     string // file on disk, possibly .dvpl
	scale    int
	sprite   *sprite.Sprite
}

// iconIndex maps a set ID and a sprite reference to the icon file relative
// to the catalog directory.
type iconIndex map[string]map[string]string

// exportIcons writes every icon set under outputDir/assets/icons as PNG with
// the descriptor's full canvas, then the catalog. References name additional
// sprites per set, relative to Data with an optional extension; each one that
// exists joins its set and is listed in the returned index. Icons listed by a
// previous catalog and absent from the new one are removed; other files are
// kept.
func exportIcons(resourceDir, outputDir string, sets []iconSet, references map[string][]string) (*IconExportReport, iconIndex, error) {
	catalogPath, err := iconPath(outputDir, "assets/icons/catalog.json")
	if err != nil {
		return nil, nil, err
	}
	previous := previousIconFiles(catalogPath)
	report := &IconExportReport{Catalog: catalogPath}
	catalog := IconCatalog{FormatVersion: iconCatalogFormatVersion, Sets: []IconSetEntry{}}
	directories := iconDirectories{resourceDir: resourceDir, cache: map[string]map[string]iconSource{}, textures: map[string]bool{}}
	// Discover every directory before selecting icons: an atlas can be named
	// by a descriptor in another directory or set.
	for _, set := range sets {
		for _, directory := range set.sources {
			if _, err := directories.directory(directory); err != nil {
				return nil, nil, err
			}
		}
		for _, reference := range references[set.id] {
			clean, err := iconRelativePath(reference)
			if err != nil {
				return nil, nil, err
			}
			directory := path.Dir(clean)
			absolute, err := iconPath(resourceDir, directory)
			if err != nil {
				return nil, nil, err
			}
			if isDir(absolute) {
				if _, err := directories.directory(directory); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	atlases := map[string]image.Image{}
	written := map[string]bool{}
	index := iconIndex{}
	for _, set := range sets {
		sources, referenced, err := directories.set(set, references[set.id])
		if err != nil {
			return nil, nil, err
		}
		index[set.id] = map[string]string{}
		for reference, id := range referenced {
			index[set.id][reference] = set.id + "/" + id + ".png"
		}
		entry := IconSetEntry{ID: set.id, Sources: set.sources, Icons: []IconEntry{}}
		for _, source := range sources {
			frames := 1
			if source.sprite != nil {
				frames = len(source.sprite.Frames)
			}
			for frame := 0; frame < frames; frame++ {
				icon, encoded, err := renderIcon(resourceDir, source, frame, atlases)
				if err != nil {
					return nil, nil, err
				}
				// The first frame keeps the file name; additional sprite frames
				// are separate variants with a stable suffix.
				if frame > 0 {
					icon.ID = source.id + "-frame" + strconv.Itoa(frame)
				}
				icon.File = set.id + "/" + icon.ID + ".png"
				file, err := iconPath(outputDir, "assets/icons/"+icon.File)
				if err != nil {
					return nil, nil, err
				}
				changed, err := writeIfChanged(file, encoded)
				if err != nil {
					return nil, nil, err
				}
				if changed {
					report.WrittenCount++
				} else {
					report.UnchangedCount++
				}
				written[icon.File] = true
				entry.Icons = append(entry.Icons, icon)
				entry.Count++
			}
		}
		catalog.Count += entry.Count
		catalog.Sets = append(catalog.Sets, entry)
	}
	for _, file := range previous {
		if !written[file] {
			absolute, err := iconPath(outputDir, "assets/icons/"+file)
			if err != nil {
				return nil, nil, err
			}
			if err := os.Remove(absolute); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return nil, nil, blitz.IOError("could not remove stale icon", file, err)
			}
			report.RemovedCount++
		}
	}
	if err := writeJSON(catalogPath, catalog, false); err != nil {
		return nil, nil, err
	}
	report.IconCount = catalog.Count
	return report, index, nil
}

// iconDirectories discovers the icons of client directories once each.
type iconDirectories struct {
	resourceDir string
	cache       map[string]map[string]iconSource
	textures    map[string]bool
}

// set returns a set's icons sorted by ID and the IDs of the references that
// resolved. Source directories must exist; a reference to a missing
// directory or file is left out.
func (d *iconDirectories) set(set iconSet, references []string) ([]iconSource, map[string]string, error) {
	byID := map[string]iconSource{}
	add := func(source iconSource) error {
		if existing, ok := byID[source.id]; ok && existing.relative != source.relative {
			return blitz.ErrorAt("ICON_EXPORT_DUPLICATE", fmt.Sprintf(
				"icon set %s has two icons named %s: %s and %s", set.id, source.id, existing.relative, source.relative), source.file)
		}
		byID[source.id] = source
		return nil
	}
	for _, directory := range set.sources {
		icons, err := d.icons(directory)
		if err != nil {
			return nil, nil, err
		}
		for _, id := range slices.Sorted(maps.Keys(icons)) {
			if !set.includes(id) {
				continue
			}
			if err := add(icons[id]); err != nil {
				return nil, nil, err
			}
		}
	}
	referenced := map[string]string{}
	for _, reference := range references {
		clean, err := iconRelativePath(reference)
		if err != nil {
			return nil, nil, err
		}
		directory, name := path.Dir(clean), path.Base(clean)
		absolute, err := iconPath(d.resourceDir, directory)
		if err != nil {
			return nil, nil, err
		}
		if !isDir(absolute) {
			continue
		}
		icons, err := d.icons(directory)
		if err != nil {
			return nil, nil, err
		}
		source, ok := icons[spriteReferenceStem(name)]
		if !ok {
			continue
		}
		if err := add(source); err != nil {
			return nil, nil, err
		}
		referenced[reference] = source.id
	}
	out := make([]iconSource, 0, len(byID))
	for _, id := range slices.Sorted(maps.Keys(byID)) {
		out = append(out, byID[id])
	}
	return out, referenced, nil
}

// directory discovers candidates by ID and kind, preferring @2x variants.
// Atlas filtering waits until all descriptors have been discovered.
func (d *iconDirectories) directory(directory string) (map[string]iconSource, error) {
	directory, err := iconRelativePath(directory)
	if err != nil {
		return nil, err
	}
	if icons, ok := d.cache[directory]; ok {
		return icons, nil
	}
	absolute, err := iconPath(d.resourceDir, directory)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(absolute)
	if err != nil {
		return nil, blitz.IOError("could not read icon directory", absolute, err)
	}
	files := map[string]string{}
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			name := unpackedPath(entry.Name())
			// A plain file takes precedence over its .dvpl sibling.
			if _, ok := files[name]; !ok || !isPacked(entry.Name()) {
				files[name] = entry.Name()
			}
		}
	}
	if d.textures == nil {
		d.textures = map[string]bool{}
	}
	local := map[string]iconSource{}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		stem, kind, ok := iconFileStem(name)
		if !ok {
			continue
		}
		if strings.ContainsAny(name, `\:`) {
			return nil, blitz.ErrorAt("ICON_EXPORT_INVALID", "icon file name contains a path separator or drive prefix", filepath.Join(absolute, files[name]))
		}
		id, double := strings.CutSuffix(stem, "@2x")
		source := iconSource{id: id, relative: path.Join(directory, name), file: filepath.Join(absolute, files[name]), scale: 1}
		if double {
			source.scale = 2
		}
		if kind == "sprite" {
			data, err := client.ReadClientAsset(source.file)
			if err != nil {
				return nil, err
			}
			if source.sprite, err = sprite.Parse(data); err != nil {
				return nil, atPath(err, source.file)
			}
			for _, texture := range source.sprite.Textures {
				if texture == "" {
					continue
				}
				relative, err := iconTexturePath(source.relative, texture)
				if err != nil {
					return nil, err
				}
				if _, err := iconPath(d.resourceDir, relative); err != nil {
					return nil, err
				}
				d.textures[iconTextureKey(relative)] = true
			}
		} else {
			// Packed images may carry their own canvas and frame.
			data, err := client.ReadClientAsset(source.file)
			if err != nil {
				return nil, err
			}
			if embedded, ok, err := sprite.Embedded(data); err != nil {
				return nil, atPath(err, source.file)
			} else if ok {
				source.sprite = embedded
			}
		}
		// Only equal formats can be resolution variants. Keep other formats
		// until atlas exclusion, then reject any remaining ID collisions.
		key := source.id + "\x00" + kind + "\x00" + strings.TrimPrefix(name, stem)
		if current, ok := local[key]; !ok || source.scale > current.scale {
			local[key] = source
		}
	}
	d.cache[directory] = local
	return local, nil
}

// icons resolves candidates after all atlas paths are known.
func (d *iconDirectories) icons(directory string) (map[string]iconSource, error) {
	candidates, err := d.directory(directory)
	if err != nil {
		return nil, err
	}
	icons := map[string]iconSource{}
	for _, key := range slices.Sorted(maps.Keys(candidates)) {
		source := candidates[key]
		if d.isAtlas(source) {
			continue
		}
		if existing, ok := icons[source.id]; ok {
			return nil, blitz.ErrorAt("ICON_EXPORT_DUPLICATE", fmt.Sprintf(
				"%s has two icons named %s: %s and %s", directory, source.id, existing.relative, source.relative), source.file)
		}
		icons[source.id] = source
	}
	return icons, nil
}

// spriteReferenceStem drops the extension a UI sprite reference may carry:
// the descriptor's, an image format, or the .psd the sprite was made from.
func spriteReferenceStem(name string) string {
	for _, extension := range []string{".txt", ".psd", ".packed.webp", ".webp", ".png"} {
		if stem, ok := strings.CutSuffix(name, extension); ok {
			return strings.TrimSuffix(stem, "@2x")
		}
	}
	return strings.TrimSuffix(name, "@2x")
}

// iconFileStem splits an unpacked file name into its stem and icon kind.
func iconFileStem(name string) (string, string, bool) {
	if stem, ok := strings.CutSuffix(name, ".txt"); ok {
		return stem, "sprite", true
	}
	for _, extension := range []string{".packed.webp", ".webp", ".png"} {
		if stem, ok := strings.CutSuffix(name, extension); ok {
			return stem, "image", true
		}
	}
	return "", "", false
}

func renderIcon(resourceDir string, source iconSource, index int, atlases map[string]image.Image) (IconEntry, []byte, error) {
	icon := IconEntry{ID: source.id, Scale: source.scale, Source: IconSource{Kind: "image", Path: source.relative}}
	var canvas *image.NRGBA
	if source.sprite == nil {
		img, err := decodeIconImage(source.file)
		if err != nil {
			return IconEntry{}, nil, err
		}
		canvas = image.NewNRGBA(image.Rectangle{Max: img.Bounds().Size()})
		draw.Draw(canvas, canvas.Bounds(), img, img.Bounds().Min, draw.Src)
	} else {
		descriptor := source.sprite
		if index >= len(descriptor.Frames) {
			return IconEntry{}, nil, blitz.ErrorAt("ICON_EXPORT_INVALID",
				fmt.Sprintf("icon sprite has no frame %d", index), source.file)
		}
		frame := descriptor.Frames[index]
		texture := source.relative
		if name := descriptor.Textures[frame.Texture]; name != "" {
			var err error
			texture, err = iconTexturePath(source.relative, name)
			if err != nil {
				return IconEntry{}, nil, err
			}
		}
		atlas, ok := atlases[texture]
		if !ok {
			if _, err := iconPath(resourceDir, texture); err != nil {
				return IconEntry{}, nil, err
			}
			if _, err := iconPath(resourceDir, texture+".dvpl"); err != nil {
				return IconEntry{}, nil, err
			}
			file := client.ResolveClientAsset(resourceDir, filepath.FromSlash(texture))
			if file == "" {
				return IconEntry{}, nil, blitz.ErrorAt("ICON_EXPORT_INVALID", "sprite atlas does not exist",
					filepath.Join(resourceDir, filepath.FromSlash(texture)))
			}
			var err error
			if atlas, err = decodeIconImage(file); err != nil {
				return IconEntry{}, nil, err
			}
			atlases[texture] = atlas
		}
		if frame.X > atlas.Bounds().Dx() || frame.Y > atlas.Bounds().Dy() ||
			frame.Width > atlas.Bounds().Dx()-frame.X || frame.Height > atlas.Bounds().Dy()-frame.Y {
			return IconEntry{}, nil, blitz.ErrorAt("ICON_EXPORT_INVALID", "sprite frame lies outside its atlas", source.file)
		}
		rect := image.Rect(frame.X, frame.Y, frame.X+frame.Width, frame.Y+frame.Height).Add(atlas.Bounds().Min)
		canvas = image.NewNRGBA(image.Rect(0, 0, descriptor.Width, descriptor.Height))
		target := image.Rect(frame.OffsetX, frame.OffsetY, frame.OffsetX+frame.Width, frame.OffsetY+frame.Height)
		draw.Draw(canvas, target, atlas, rect.Min, draw.Src)
		icon.Source = IconSource{Kind: "sprite", Path: source.relative, Texture: texture, Frame: &IconFrame{
			X: frame.X, Y: frame.Y, Width: frame.Width, Height: frame.Height, OffsetX: frame.OffsetX, OffsetY: frame.OffsetY}}
	}
	icon.Width, icon.Height = canvas.Bounds().Dx(), canvas.Bounds().Dy()
	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&encoded, canvas); err != nil {
		return IconEntry{}, nil, blitz.ErrorAt("ICON_EXPORT_INVALID", "could not encode icon: "+err.Error(), source.file)
	}
	return icon, encoded.Bytes(), nil
}

func decodeIconImage(file string) (image.Image, error) {
	data, err := client.ReadClientAsset(file)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, blitz.ErrorAt("ICON_EXPORT_INVALID", "could not decode image: "+err.Error(), file)
	}
	return img, nil
}

// previousIconFiles returns the icon files of an existing catalog. A missing
// or unreadable catalog yields none, so nothing is removed.
func previousIconFiles(catalogPath string) []string {
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil
	}
	var catalog IconCatalog
	if json.Unmarshal(data, &catalog) != nil || catalog.FormatVersion != iconCatalogFormatVersion {
		return nil
	}
	var files []string
	for _, set := range catalog.Sets {
		for _, icon := range set.Icons {
			// Only PNG paths inside the icon folder are trusted for removal.
			if clean, err := iconRelativePath(icon.File); err == nil && strings.HasSuffix(clean, ".png") {
				files = append(files, clean)
			}
		}
	}
	return files
}

// writeIfChanged writes data atomically unless the file already holds it.
func writeIfChanged(file string, data []byte) (bool, error) {
	existing, err := os.ReadFile(file)
	switch {
	case err == nil && bytes.Equal(existing, data):
		return false, nil
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return false, blitz.IOError("could not read existing icon", file, err)
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o777); err != nil {
		return false, blitz.IOError("could not create icon directory", file, err)
	}
	return true, atomicWrite(file, data)
}

func atPath(err error, file string) error {
	typed := *blitz.AsError(err)
	typed.Path = file
	return &typed
}

// iconRelativePath uses a portable spelling before applying native path rules.
func iconRelativePath(relative string) (string, error) {
	relative = strings.ReplaceAll(relative, `\`, "/")
	clean := path.Clean(relative)
	if strings.ContainsAny(clean, ":\x00") || !filepath.IsLocal(filepath.FromSlash(clean)) || path.IsAbs(clean) {
		return "", blitz.ErrorAt("ICON_EXPORT_INVALID", "icon path must stay inside its root", relative)
	}
	return clean, nil
}

// iconPath rejects symlinks below the selected root, including the final file.
// Like the other file exporter, it assumes the tree is not concurrently mutated.
func iconPath(root, relative string) (string, error) {
	clean, err := iconRelativePath(relative)
	if err != nil {
		return "", err
	}
	absolute := filepath.Join(root, filepath.FromSlash(clean))
	for current := absolute; current != filepath.Clean(root); current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&fs.ModeSymlink != 0 {
			return "", blitz.ErrorAt("ICON_EXPORT_INVALID", "icon path contains a symlink", current)
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", blitz.IOError("could not inspect icon path", current, err)
		}
	}
	return absolute, nil
}

func iconTexturePath(descriptor, texture string) (string, error) {
	texture = strings.ReplaceAll(texture, `\`, "/")
	if path.IsAbs(texture) || strings.Contains(texture, ":") {
		return "", blitz.ErrorAt("ICON_EXPORT_INVALID", "atlas path must be relative to its descriptor", texture)
	}
	return iconRelativePath(path.Join(path.Dir(descriptor), texture))
}

func iconTextureKey(relative string) string {
	stem, _, ok := iconFileStem(path.Base(relative))
	if !ok {
		return relative
	}
	return path.Join(path.Dir(relative), strings.TrimSuffix(stem, "@2x")+strings.TrimPrefix(path.Base(relative), stem))
}

func (d *iconDirectories) isAtlas(source iconSource) bool {
	_, kind, _ := iconFileStem(path.Base(source.relative))
	return kind == "image" && d.textures[iconTextureKey(source.relative)]
}
