package export

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
)

// ClientExportOptions configures the single client snapshot format.
type ClientExportOptions struct {
	GameDir   string
	OutputDir string
	CacheDir  string
	NoCache   bool
	// ServerConfig optionally supplies crew skill training costs from a local cache.
	ServerConfig string
	// MapsConfig optionally supplies per-map battle modes from a local cache
	// (maps_config_*.dat, or the directory holding it).
	MapsConfig string
	// Include is data, icons, models, maps, mechanics, modes, or all. Empty means all.
	Include      []string
	ModelOptions TankExportOptions
}

func NewClientExportOptions(outputDir string) ClientExportOptions {
	models := NewTankExportOptions()
	return ClientExportOptions{OutputDir: outputDir, Include: []string{"all"}, ModelOptions: models}
}

// ParseExportIncludes validates and canonicalizes the requested snapshot sections.
func ParseExportIncludes(values []string) ([]string, error) {
	if len(values) == 0 {
		values = []string{"all"}
	}
	selected := map[string]bool{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if !slices.Contains([]string{"all", "data", "icons", "models", "maps", "mechanics", "modes"}, part) {
				return nil, fmt.Errorf("unknown export section %q; expected all, data, icons, models, maps, mechanics, modes", part)
			}
			selected[part] = true
		}
	}
	if selected["all"] && len(selected) > 1 {
		return nil, fmt.Errorf("all cannot be combined with individual export sections")
	}
	result := []string{}
	for _, part := range []string{"data", "icons", "models", "maps", "mechanics", "modes"} {
		if selected["all"] || selected[part] {
			result = append(result, part)
		}
	}
	return result, nil
}

// ClientExportReport describes the written snapshot. Nonfatal missing media are in Issues.
type ClientExportReport struct {
	File     string        `json:"file"`
	Included []string      `json:"included"`
	Game     *SnapshotGame `json:"game"`
	Tanks    int           `json:"tanks"`
	Models   int           `json:"models"`
	Issues   []ExportIssue `json:"issues"`
	// IconExport, ItemsFile and MissingItemIcons are set when icons are included.
	IconExport       *IconExportReport `json:"iconExport,omitempty"`
	ItemsFile        string            `json:"itemsFile,omitempty"`
	MissingItemIcons []string          `json:"missingItemIcons,omitempty"`
	// MapExport is set when maps are included.
	MapExport *MapExportReport `json:"mapExport,omitempty"`
	// MechanicExport is set when mechanics are included.
	MechanicExport *MechanicExportReport `json:"mechanicExport,omitempty"`
	// ModeExport is set when modes are included.
	ModeExport *ModeExportReport `json:"modeExport,omitempty"`
}

type ExportIssue struct {
	Tank    string `json:"tank"`
	Section string `json:"section"`
	Message string `json:"message"`
}

// ExportClient writes one data.json and its referenced binary media to a new directory.
// It never writes alternate metadata or model catalogs. A failed export removes
// everything it wrote, so the same directory can be reused for a retry.
func ExportClient(o ClientExportOptions) (report *ClientExportReport, err error) {
	included, err := ParseExportIncludes(o.Include)
	if err != nil {
		return nil, blitz.NewError("USAGE", err.Error())
	}
	if exportCacheConflict(o.CacheDir, o.NoCache) {
		return nil, blitz.NewError("USAGE", "--cache-dir and --no-cache cannot be combined")
	}
	if o.ServerConfig != "" && !slices.Contains(included, "data") {
		return nil, blitz.NewError("USAGE", "--server-config requires the data section")
	}
	if o.MapsConfig != "" && !slices.Contains(included, "maps") {
		return nil, blitz.NewError("USAGE", "--maps-config requires the maps section")
	}
	if o.OutputDir == "" {
		return nil, blitz.NewError("USAGE", "output directory is required")
	}
	// Validate conversion settings before reading/writing the snapshot.
	if slices.Contains(included, "models") {
		if err := validateTankOptions(&o.ModelOptions); err != nil {
			return nil, err
		}
	}
	roots, err := openExportRoots(exportSessionOptions{
		GameDir:            o.GameDir,
		CacheDir:           o.CacheDir,
		NoCache:            o.NoCache,
		OutputDir:          o.OutputDir,
		RequireEmptyOutput: true,
	})
	if err != nil {
		return nil, err
	}
	out := roots.OutputDir
	defer func() {
		if err != nil {
			removeExportOutput(out, roots.OutputExisted)
		}
	}()
	clientResources, err := client.OpenResources(roots.ResourceDir, roots.CacheDir)
	if err != nil {
		return nil, err
	}
	files := map[string]sourceFile{}
	skipped := 0
	types := []ClientFileType{FileTypeConfigs}
	if slices.Contains(included, "icons") || slices.Contains(included, "models") || slices.Contains(included, "maps") || slices.Contains(included, "mechanics") || slices.Contains(included, "modes") {
		types = append(types, FileTypeImages)
	}
	if slices.Contains(included, "models") {
		types = append(types, FileTypeModels)
	}
	for i, root := range roots.sources() {
		if err := collectFiles(root, ClientFilesExportOptions{Types: types}, i > 0, files, &skipped); err != nil {
			return nil, err
		}
	}
	snapshot, err := extractSnapshot(clientResources, files, included, out)
	if err != nil {
		return nil, err
	}
	var iconReport *IconExportReport
	var itemsFile string
	var missingItemIcons []string
	if slices.Contains(included, "icons") {
		if iconReport, itemsFile, missingItemIcons, err = exportItemIcons(clientResources, roots.ResourceDir, out); err != nil {
			return nil, err
		}
	}
	if o.ServerConfig != "" {
		training, err := client.ReadCrewSkillTraining(o.ServerConfig)
		if err != nil {
			return nil, err
		}
		snapshot.CrewSkillTraining = training
		client.ApplyCrewSkillTraining(snapshot.CrewSkills, training)
	}
	if slices.Contains(included, "models") {
		if err := exportModels(snapshot, resourcesResolver(clientResources), filepath.Join(out, "assets/tanks"), &o.ModelOptions); err != nil {
			return nil, err
		}
	}
	var mapReport *MapExportReport
	if slices.Contains(included, "maps") {
		if mapReport, err = exportMapsWithConfig(roots.ResourceDir, out, o.MapsConfig, snapshot); err != nil {
			return nil, err
		}
	}
	var mechanicReport *MechanicExportReport
	if slices.Contains(included, "mechanics") {
		if mechanicReport, err = exportMechanics(files, roots.ResourceDir, out, snapshot); err != nil {
			return nil, err
		}
	}
	var modeReport *ModeExportReport
	if slices.Contains(included, "modes") {
		if modeReport, err = exportGameModes(files, roots.ResourceDir, out, snapshot); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(out, "data.json")
	if err := writeJSON(path, snapshot, false); err != nil {
		return nil, err
	}
	count := 0
	for _, tank := range snapshot.Tanks {
		if tank.Model != nil {
			count++
		}
	}
	return &ClientExportReport{File: path, Included: included, Game: snapshot.Game, Tanks: len(snapshot.Tanks), Models: count, Issues: snapshot.Issues, IconExport: iconReport, ItemsFile: itemsFile, MissingItemIcons: missingItemIcons, MapExport: mapReport, MechanicExport: mechanicReport, ModeExport: modeReport}, nil
}

// exportItemIcons writes the item icon sets, their catalog and items.json.
// The item definitions and UI sprite mappings are read from the client's Data
// directory, like the icon resources themselves.
func exportItemIcons(resources *client.Resources, resourceDir, out string) (*IconExportReport, string, []string, error) {
	items, err := client.ReadClientItems(resourceDir)
	if err != nil {
		return nil, "", nil, err
	}
	sprites, err := client.ReadItemIconSprites(resourceDir)
	if err != nil {
		return nil, "", nil, err
	}
	report, index, err := exportIconsWithResources(resources, resourceDir, out, existingIconSets(resources, iconSets), itemIconReferences(items, sprites))
	if err != nil {
		return nil, "", nil, err
	}
	linked, missing := linkItemIcons(items, sprites, index)
	itemsFile := filepath.Join(out, "items.json")
	if err := writeJSON(itemsFile, linked, false); err != nil {
		return nil, "", nil, err
	}
	return report, itemsFile, missing, nil
}

// exportModels attaches converted showroom models. Per-tank failures become
// issues; failures that affect every tank, or a run where no model converts, fail.
// Tanks convert in parallel; results, issues and the chosen error are assembled
// in tank order, so the snapshot matches a sequential run.
func exportModels(snapshot *Snapshot, resolve assetResolver, outputRoot string, options *TankExportOptions) error {
	folders := map[string]string{}
	for _, nation := range client.Nations() {
		folders[nation.ID] = nation.ModelFolder
	}
	type conversion struct {
		index int
		model client.ClientTank
	}
	var conversions []conversion
	for i := range snapshot.Tanks {
		tank := &snapshot.Tanks[i]
		folder := folders[tank.Nation]
		base := "3d/Tanks/" + folder + "/" + tank.TechName
		if folder == "" || resolve(base+".sc2") == "" || resolve(base+".scg") == "" {
			snapshot.Issues = append(snapshot.Issues, ExportIssue{Tank: tank.ID, Section: "models", Message: "showroom model is not installed"})
			continue
		}
		conversions = append(conversions, conversion{index: i, model: client.ClientTank{
			ID: tank.TechName, Nation: tank.Nation, Name: tank.TechName,
			ModelFiles: client.TankModelFiles{Scene: base + ".sc2", Geometry: base + ".scg"},
		}})
	}

	textures := newTextureDeduper()
	results := make([]modelExportResult, len(snapshot.Tanks))
	var failed atomic.Bool
	workers := min(max(runtime.GOMAXPROCS(0), 1), max(len(conversions), 1))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if failed.Load() {
					return
				}
				c := &conversions[j]
				entry, err := exportOne(&c.model, resolve, outputRoot, options, textures)
				results[c.index] = modelExportResult{entry: entry, err: err}
				if err != nil && fatalModelError(err) {
					failed.Store(true)
					return
				}
			}
		}()
	}
	for j := range conversions {
		jobs <- j
	}
	close(jobs)
	wg.Wait()

	attempted, exported := 0, 0
	var first, fatal error
	for _, c := range conversions {
		attempted++
		result := results[c.index]
		if result.err != nil {
			if fatalModelError(result.err) {
				if fatal == nil {
					fatal = result.err
				}
				continue
			}
			if first == nil {
				first = result.err
			}
			tank := &snapshot.Tanks[c.index]
			snapshot.Issues = append(snapshot.Issues, ExportIssue{Tank: tank.ID, Section: "models", Message: result.err.Error()})
			continue
		}
		exported++
		snapshot.Tanks[c.index].Model = result.entry
	}
	if fatal != nil {
		return fatal
	}
	if attempted > 0 && exported == 0 {
		return blitz.NewError("MODEL_EXPORT_FAILED", fmt.Sprintf("all %d model exports failed; first error: %v", attempted, first))
	}
	return nil
}

// modelExportResult keeps one tank's conversion outcome until results are
// assembled in tank order.
type modelExportResult struct {
	entry *SnapshotModel
	err   error
}

// removeExportOutput deletes a failed export. The directory was new or empty
// before the export, so everything inside it was written by this run.
func removeExportOutput(out string, existed bool) {
	if !existed {
		os.RemoveAll(out)
		return
	}
	entries, _ := os.ReadDir(out)
	for _, entry := range entries {
		os.RemoveAll(filepath.Join(out, entry.Name()))
	}
}
