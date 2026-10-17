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
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/resources/sprite"
	_ "golang.org/x/image/webp"
)

// iconSet is one output folder of icons collected from client directories.
// Every sprite descriptor and standalone image in a source directory becomes
// an icon named after its file; atlases referenced by descriptors do not.
type iconSet struct {
	id      string
	sources []string
}

// iconSets lists the exported groups. Event-specific folders are not included.
var iconSets = []iconSet{
	{"shells", []string{"Gfx/Shared/tank-supply/ammunition/big"}},
	{"consumables", []string{"Gfx/UI/InventoryIcons/Big/Consumables"}},
	{"provisions", []string{"Gfx/UI/InventoryIcons/Big/Provisions"}},
	{"equipment", []string{"Gfx/UI/InventoryIcons/Big/OptionalDevices", "Gfx/Lobby/tank-supply/optional-devices"}},
	{"crew-skills", []string{"Gfx/UI/CrewSkillIcons"}},
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
	root := filepath.Join(outputDir, "assets", "icons")
	catalogPath := filepath.Join(root, "catalog.json")
	previous := previousIconFiles(catalogPath)
	report := &IconExportReport{Catalog: catalogPath}
	catalog := IconCatalog{FormatVersion: iconCatalogFormatVersion, Sets: []IconSetEntry{}}
	directories := iconDirectories{resourceDir: resourceDir, cache: map[string]map[string]iconSource{}}
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
		entry := IconSetEntry{ID: set.id, Sources: set.sources, Count: len(sources), Icons: []IconEntry{}}
		for _, source := range sources {
			icon, encoded, err := renderIcon(resourceDir, source, atlases)
			if err != nil {
				return nil, nil, err
			}
			icon.File = set.id + "/" + source.id + ".png"
			changed, err := writeIfChanged(filepath.Join(root, filepath.FromSlash(icon.File)), encoded)
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
		}
		catalog.Count += entry.Count
		catalog.Sets = append(catalog.Sets, entry)
	}
	for _, file := range previous {
		if !written[file] {
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(file))); err != nil && !errors.Is(err, fs.ErrNotExist) {
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
		icons, err := d.directory(directory)
		if err != nil {
			return nil, nil, err
		}
		for _, id := range slices.Sorted(maps.Keys(icons)) {
			if err := add(icons[id]); err != nil {
				return nil, nil, err
			}
		}
	}
	referenced := map[string]string{}
	for _, reference := range references {
		directory, name := path.Split(reference)
		directory = strings.TrimSuffix(directory, "/")
		if !isDir(filepath.Join(d.resourceDir, filepath.FromSlash(directory))) {
			continue
		}
		icons, err := d.directory(directory)
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

// directory returns the icons of one directory by ID, preferring @2x
// variants. Images used as atlases by its descriptors are not icons.
func (d *iconDirectories) directory(directory string) (map[string]iconSource, error) {
	if icons, ok := d.cache[directory]; ok {
		return icons, nil
	}
	absolute := filepath.Join(d.resourceDir, filepath.FromSlash(directory))
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
	// Atlases are matched by stem so either resolution of one is skipped.
	textures := map[string]bool{}
	local := map[string]iconSource{}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		stem, kind, ok := iconFileStem(name)
		if !ok {
			continue
		}
		id, double := strings.CutSuffix(stem, "@2x")
		source := iconSource{id: id, relative: directory + "/" + name, file: filepath.Join(absolute, files[name]), scale: 1}
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
				if stem, _, ok := iconFileStem(path.Base(texture)); ok {
					textures[strings.TrimSuffix(stem, "@2x")] = true
				}
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
		key := source.id + "\x00" + kind
		if current, ok := local[key]; !ok || source.scale > current.scale {
			local[key] = source
		}
	}
	icons := map[string]iconSource{}
	for _, key := range slices.Sorted(maps.Keys(local)) {
		source := local[key]
		if source.sprite == nil && textures[source.id] {
			continue
		}
		if existing, ok := icons[source.id]; ok {
			return nil, blitz.ErrorAt("ICON_EXPORT_DUPLICATE", fmt.Sprintf(
				"%s has a sprite and an image named %s: %s and %s", directory, source.id, existing.relative, source.relative), source.file)
		}
		icons[source.id] = source
	}
	d.cache[directory] = icons
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

func renderIcon(resourceDir string, source iconSource, atlases map[string]image.Image) (IconEntry, []byte, error) {
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
		if len(descriptor.Frames) != 1 {
			return IconEntry{}, nil, blitz.ErrorAt("ICON_EXPORT_INVALID",
				fmt.Sprintf("icon sprite must have one frame, found %d", len(descriptor.Frames)), source.file)
		}
		frame := descriptor.Frames[0]
		texture := source.relative
		if name := descriptor.Textures[frame.Texture]; name != "" {
			texture = path.Join(path.Dir(source.relative), name)
		}
		atlas, ok := atlases[texture]
		if !ok {
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
		rect := image.Rect(frame.X, frame.Y, frame.X+frame.Width, frame.Y+frame.Height).Add(atlas.Bounds().Min)
		if !rect.In(atlas.Bounds()) {
			return IconEntry{}, nil, blitz.ErrorAt("ICON_EXPORT_INVALID",
				fmt.Sprintf("sprite frame %v lies outside its %v atlas %s", rect, atlas.Bounds().Size(), texture), source.file)
		}
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
			if filepath.IsLocal(filepath.FromSlash(icon.File)) && strings.HasSuffix(icon.File, ".png") {
				files = append(files, icon.File)
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
