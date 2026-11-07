package export

import (
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"image"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/resources/sprite"
	"go.yaml.in/yaml/v3"
	_ "golang.org/x/image/webp"
)

// Map image resources relative to the client Data directory. The preview
// atlas holds one banner frame per map, indexed by the maps.yaml spriteFrame;
// minimaps live in per-space folders whose names are the maps.yaml keys.
const (
	mapPreviewAtlas  = "Gfx/UI/MapsListDialog/back_maps@2x.packed.webp"
	mapMinimapPrefix = "Gfx/UI/BattleScreenHUD/minimap/"
)

// MapExport is one entry of maps.json. Preview and minimap are relative to
// the export root and null when the client resource is not installed. The
// source fields name the client resource each image was rendered from, with
// the atlas rectangle for cropped sprite frames.
type MapExport struct {
	ID                      int64       `json:"id"`
	Name                    string      `json:"name"`
	Scene                   string      `json:"scene"`
	AvailableInTrainingRoom bool        `json:"availableInTrainingRoom"`
	AvailableModes          []int64     `json:"availableModes"`
	SpriteFrame             int64       `json:"spriteFrame"`
	Preview                 *string     `json:"preview"`
	Minimap                 *string     `json:"minimap"`
	PreviewSource           *IconSource `json:"previewSource,omitempty"`
	MinimapSource           *IconSource `json:"minimapSource,omitempty"`
	Effects                 []string    `json:"effects,omitempty"`
}

// MapExportReport summarizes the map part of a client export.
type MapExportReport struct {
	File     string `json:"file"`
	Maps     int    `json:"maps"`
	Previews int    `json:"previews"`
	Minimaps int    `json:"minimaps"`
}

// mapExportResult keeps one map's rendering outcome until results are
// assembled in maps.yaml order.
type mapExportResult struct {
	entry         MapExport
	preview       []byte
	minimap       []byte
	previewSource *IconSource
	minimapSource *IconSource
	issues        []string
}

// exportMaps writes maps.json and the map images under assets/maps. Missing
// or broken per-map resources become issues; only a failure to read maps.yaml
// or to write the output fails the export.
func exportMaps(resourceDir, outputDir string, snapshot *Snapshot) (*MapExportReport, error) {
	// Like the data section, a client without maps.yaml exports nothing.
	if client.ResolveClientAsset(resourceDir, "maps.yml") == "" && client.ResolveClientAsset(resourceDir, "maps.yaml") == "" {
		snapshot.Issues = append(snapshot.Issues, ExportIssue{Section: "maps", Message: "maps.yml or maps.yaml was not found in the client resource directory"})
		if err := writeJSON(filepath.Join(outputDir, "maps.json"), []MapExport{}, false); err != nil {
			return nil, err
		}
		return &MapExportReport{File: filepath.Join(outputDir, "maps.json")}, nil
	}
	maps, err := client.ReadMapsCatalog(resourceDir)
	if err != nil {
		return nil, err
	}
	// One atlas serves every preview: resolve and decode it before the
	// workers start, so the shared cache is read-only while they run.
	previewSource, previewErr := mapSprite(resourceDir, mapPreviewAtlas)
	if previewErr != nil {
		snapshot.Issues = append(snapshot.Issues, ExportIssue{Section: "maps", Message: previewErr.Error()})
	}
	atlases := map[string]image.Image{}
	if previewSource != nil {
		atlas, err := decodeIconImage(previewSource.file)
		if err != nil {
			snapshot.Issues = append(snapshot.Issues, ExportIssue{Section: "maps", Message: err.Error()})
			previewSource = nil
		} else {
			atlases[previewSource.relative] = atlas
		}
	}
	results := make([]mapExportResult, len(maps))
	workers := min(max(runtime.GOMAXPROCS(0), 1), max(len(maps), 1))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				results[i] = renderMap(resourceDir, &maps[i], previewSource, atlases)
			}
		}()
	}
	for i := range maps {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	report := &MapExportReport{Maps: len(maps)}
	entries := make([]MapExport, 0, len(maps))
	for i := range maps {
		result := &results[i]
		for _, message := range result.issues {
			snapshot.Issues = append(snapshot.Issues, ExportIssue{Section: "maps", Message: message})
		}
		if result.preview != nil {
			file, err := writeMapImage(outputDir, maps[i].Name, "preview.png", result.preview)
			if err != nil {
				return nil, err
			}
			result.entry.Preview = &file
			result.entry.PreviewSource = result.previewSource
			report.Previews++
		}
		if result.minimap != nil {
			file, err := writeMapImage(outputDir, maps[i].Name, "minimap.png", result.minimap)
			if err != nil {
				return nil, err
			}
			result.entry.Minimap = &file
			result.entry.MinimapSource = result.minimapSource
			report.Minimaps++
		}
		entries = append(entries, result.entry)
	}
	file := filepath.Join(outputDir, "maps.json")
	if err := writeJSON(file, entries, false); err != nil {
		return nil, err
	}
	report.File = file
	return report, nil
}

// renderMap renders one map's preview and minimap. Missing or broken client
// resources are collected as issues and leave the images out of the entry.
func renderMap(resourceDir string, m *client.ClientMap, previewSource *iconSource, atlases map[string]image.Image) mapExportResult {
	result := mapExportResult{entry: MapExport{
		ID:                      m.ID,
		Name:                    m.Name,
		Scene:                   m.LocalName,
		AvailableInTrainingRoom: m.AvailableInTrainingRoom,
		AvailableModes:          m.AvailableModes,
		SpriteFrame:             m.SpriteFrame,
	}}
	effects, err := readMapEffects(resourceDir, m.LocalName)
	if err != nil {
		result.issues = append(result.issues, err.Error())
	}
	result.entry.Effects = effects
	if previewSource != nil {
		if int(m.SpriteFrame) < 0 || int(m.SpriteFrame) >= len(previewSource.sprite.Frames) {
			result.issues = append(result.issues, blitz.ErrorAt("MAP_EXPORT_MISSING",
				fmt.Sprintf("preview atlas has no frame %d", m.SpriteFrame), previewSource.file).Error())
		} else if icon, encoded, err := renderIcon(resourceDir, *previewSource, int(m.SpriteFrame), atlases); err != nil {
			result.issues = append(result.issues, err.Error())
		} else {
			result.preview = encoded
			result.previewSource = &icon.Source
		}
	}
	minimapSource, err := mapSprite(resourceDir, mapMinimapPrefix+m.Name+"/MiniMapSmall@2x.packed.webp")
	if err != nil {
		result.issues = append(result.issues, err.Error())
	} else if icon, encoded, err := renderIcon(resourceDir, *minimapSource, 0, atlases); err != nil {
		result.issues = append(result.issues, err.Error())
	} else {
		result.minimap = encoded
		result.minimapSource = &icon.Source
	}
	return result
}

// mapSprite resolves a packed image with an embedded sprite descriptor,
// falling back from the @2x variant to the SD one. A missing resource yields
// a MAP_EXPORT_MISSING error.
func mapSprite(resourceDir, relative string) (*iconSource, error) {
	clean, err := iconRelativePath(relative)
	if err != nil {
		return nil, err
	}
	file := client.ResolveClientAsset(resourceDir, filepath.FromSlash(clean))
	if file == "" {
		sd, err := iconRelativePath(strings.Replace(clean, "@2x.packed.webp", ".packed.webp", 1))
		if err != nil {
			return nil, err
		}
		file = client.ResolveClientAsset(resourceDir, filepath.FromSlash(sd))
		if file == "" {
			return nil, blitz.ErrorAt("MAP_EXPORT_MISSING", "map image is not installed",
				filepath.Join(resourceDir, filepath.FromSlash(clean)))
		}
		clean = sd
	}
	data, err := client.ReadClientAsset(file)
	if err != nil {
		return nil, err
	}
	descriptor, ok, err := sprite.Embedded(data)
	if err != nil {
		return nil, atPath(err, file)
	}
	if !ok {
		return nil, blitz.ErrorAt("MAP_EXPORT_INVALID", "packed map image has no sprite descriptor", file)
	}
	return &iconSource{id: path.Base(clean), relative: clean, file: file, scale: mapImageScale(clean), sprite: descriptor}, nil
}

func mapImageScale(relative string) int {
	if strings.Contains(path.Base(relative), "@2x") {
		return 2
	}
	return 1
}

// writeMapImage writes one rendered PNG under assets/maps/<space>/ and
// returns its path relative to the export root.
func writeMapImage(outputDir, space, name string, encoded []byte) (string, error) {
	file := "assets/maps/" + space + "/" + name
	absolute, err := iconPath(outputDir, file)
	if err != nil {
		return "", err
	}
	if _, err := writeIfChanged(absolute, encoded); err != nil {
		return "", err
	}
	return file, nil
}

// readMapEffects reads the surface effect names of a map scene from its
// blitz/map_effects.yaml keyed archive. A missing file yields no effects.
func readMapEffects(resourceDir, scene string) ([]string, error) {
	scene = strings.ReplaceAll(scene, `\`, "/")
	directory := path.Dir(scene)
	if directory == "." || directory == "/" || strings.Contains(scene, ":") {
		return nil, nil
	}
	file := client.ResolveClientAsset(resourceDir, filepath.FromSlash("3d/Maps/"+directory+"/blitz/map_effects.yaml"))
	if file == "" {
		return nil, nil
	}
	data, err := client.ReadClientAsset(file)
	if err != nil {
		return nil, err
	}
	var document struct {
		KeyedArchive map[string]map[string]any `yaml:"keyedArchive"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, blitz.ErrorAt("CLIENT_YAML_INVALID", "map_effects.yaml: "+err.Error(), file)
	}
	names := make(map[int]string, len(document.KeyedArchive))
	for key, value := range document.KeyedArchive {
		if text, ok := value["string"].(string); ok {
			if index, err := strconv.Atoi(strings.TrimPrefix(key, "fx")); err == nil {
				names[index] = text
			}
		}
	}
	if len(names) == 0 {
		return nil, nil
	}
	indices := make([]int, 0, len(names))
	for index := range names {
		indices = append(indices, index)
	}
	slices.Sort(indices)
	effects := make([]string, 0, len(names))
	for _, index := range indices {
		effects = append(effects, names[index])
	}
	return effects, nil
}
