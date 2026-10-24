package export

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/internal/pathx"
)

// ClientExportOptions configures the single client snapshot format.
type ClientExportOptions struct {
	GameDir   string
	OutputDir string
	CacheDir  string
	NoCache   bool
	// ServerConfig optionally supplies crew skill training costs from a local cache.
	ServerConfig string
	// Include is data, icons, models, or all. Empty means all.
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
			if !slices.Contains([]string{"all", "data", "icons", "models"}, part) {
				return nil, fmt.Errorf("unknown export section %q; expected all, data, icons, models", part)
			}
			selected[part] = true
		}
	}
	if selected["all"] && len(selected) > 1 {
		return nil, fmt.Errorf("all cannot be combined with individual export sections")
	}
	result := []string{}
	for _, part := range []string{"data", "icons", "models"} {
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
	Tanks    int           `json:"tanks"`
	Models   int           `json:"models"`
	Issues   []ExportIssue `json:"issues"`
	// IconExport, ItemsFile and MissingItemIcons are set when icons are included.
	IconExport       *IconExportReport `json:"iconExport,omitempty"`
	ItemsFile        string            `json:"itemsFile,omitempty"`
	MissingItemIcons []string          `json:"missingItemIcons,omitempty"`
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
	if o.NoCache && o.CacheDir != "" {
		return nil, blitz.NewError("USAGE", "--cache-dir and --no-cache cannot be combined")
	}
	if o.ServerConfig != "" && !slices.Contains(included, "data") {
		return nil, blitz.NewError("USAGE", "--server-config requires the data section")
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
	game, err := client.ResolveGameDir(o.GameDir)
	if err != nil {
		return nil, err
	}
	// Canonical roots keep the overlap check below correct for symlinked Data.
	resourceDir, err := canonicalDirectory(game.ResourceDir)
	if err != nil {
		return nil, err
	}
	roots := []string{resourceDir}
	cache := o.CacheDir
	if cache == "" && !o.NoCache && game.CacheDir != nil && isDir(*game.CacheDir) {
		cache = *game.CacheDir
	}
	if cache != "" {
		cache, err = canonicalDirectory(cache)
		if err != nil {
			return nil, err
		}
		roots = append(roots, cache)
	}
	out, err := resolveOutputPath(o.OutputDir)
	if err != nil {
		return nil, err
	}
	for _, root := range roots {
		if pathx.HasPrefix(out, root) || pathx.HasPrefix(root, out) {
			return nil, fmt.Errorf("output and source directories must not overlap")
		}
	}
	entries, err := os.ReadDir(out)
	existed := err == nil
	if existed && len(entries) != 0 {
		return nil, fmt.Errorf("export requires a new or empty output directory: %s", out)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	defer func() {
		if err != nil {
			removeExportOutput(out, existed)
		}
	}()
	files := map[string]sourceFile{}
	skipped := 0
	types := []ClientFileType{FileTypeConfigs}
	if slices.Contains(included, "icons") || slices.Contains(included, "models") {
		types = append(types, FileTypeImages)
	}
	if slices.Contains(included, "models") {
		types = append(types, FileTypeModels)
	}
	for i, root := range roots {
		if err := collectFiles(root, ClientFilesExportOptions{Types: types}, i > 0, files, &skipped); err != nil {
			return nil, err
		}
	}
	snapshot, err := extractSnapshot(files, included, out)
	if err != nil {
		return nil, err
	}
	var iconReport *IconExportReport
	var itemsFile string
	var missingItemIcons []string
	if slices.Contains(included, "icons") {
		if iconReport, itemsFile, missingItemIcons, err = exportItemIcons(resourceDir, out); err != nil {
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
		if err := exportModels(snapshot, filesResolver(files), filepath.Join(out, "assets/tanks"), &o.ModelOptions); err != nil {
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
	return &ClientExportReport{File: path, Included: included, Tanks: len(snapshot.Tanks), Models: count, Issues: snapshot.Issues, IconExport: iconReport, ItemsFile: itemsFile, MissingItemIcons: missingItemIcons}, nil
}

// exportItemIcons writes the item icon sets, their catalog and items.json.
// The item definitions and UI sprite mappings are read from the client's Data
// directory, like the icon resources themselves.
func exportItemIcons(resourceDir, out string) (*IconExportReport, string, []string, error) {
	items, err := client.ReadClientItems(resourceDir)
	if err != nil {
		return nil, "", nil, err
	}
	sprites, err := client.ReadItemIconSprites(resourceDir)
	if err != nil {
		return nil, "", nil, err
	}
	report, index, err := exportIcons(resourceDir, out, existingIconSets(resourceDir, iconSets), itemIconReferences(items, sprites))
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
func exportModels(snapshot *Snapshot, resolve assetResolver, outputRoot string, options *TankExportOptions) error {
	folders := map[string]string{}
	for _, nation := range client.Nations() {
		folders[nation.ID] = nation.ModelFolder
	}
	attempted, exported := 0, 0
	var first error
	for i := range snapshot.Tanks {
		tank := &snapshot.Tanks[i]
		folder := folders[tank.Nation]
		base := "3d/Tanks/" + folder + "/" + tank.TechName
		if folder == "" || resolve(base+".sc2") == "" || resolve(base+".scg") == "" {
			snapshot.Issues = append(snapshot.Issues, ExportIssue{Tank: tank.ID, Section: "models", Message: "showroom model is not installed"})
			continue
		}
		model := client.ClientTank{ID: tank.TechName, Nation: tank.Nation, Name: tank.TechName, ModelFiles: client.TankModelFiles{Scene: base + ".sc2", Geometry: base + ".scg"}}
		attempted++
		entry, err := exportOne(&model, resolve, outputRoot, options)
		if err != nil {
			if fatalModelError(err) {
				return err
			}
			if first == nil {
				first = err
			}
			snapshot.Issues = append(snapshot.Issues, ExportIssue{Tank: tank.ID, Section: "models", Message: err.Error()})
			continue
		}
		exported++
		tank.Model = entry
	}
	if attempted > 0 && exported == 0 {
		return blitz.NewError("MODEL_EXPORT_FAILED", fmt.Sprintf("all %d model exports failed; first error: %v", attempted, first))
	}
	return nil
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
