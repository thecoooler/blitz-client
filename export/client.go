package export

import (
	"os"
	"path/filepath"
	"time"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/pathx"
)

// ClientExportOptions configures ExportClient.
type ClientExportOptions struct {
	// GameDir is the install directory or its Data directory; empty uses platform defaults.
	GameDir string
	// OutputDir receives maps.json, tanks.json, items.json, assets/icons and assets/tanks.
	OutputDir string
	// DataOnly skips the model export; metadata and icons are still written.
	DataOnly bool
	// ModelOptions configures the model export. Its ViewerDir is set to OutputDir.
	ModelOptions TankExportOptions
}

// NewClientExportOptions returns the default combined export options.
func NewClientExportOptions(outputDir string) ClientExportOptions {
	return ClientExportOptions{OutputDir: outputDir, ModelOptions: NewTankExportOptions(outputDir)}
}

// ClientExportReport summarizes a combined export. ItemCount counts
// consumables, provisions, and equipment; MissingItemIcons lists the
// "kind/key" icon keys that got no icon.
type ClientExportReport struct {
	OutputDir        string            `json:"outputDir"`
	MapsFile         string            `json:"mapsFile"`
	TanksFile        string            `json:"tanksFile"`
	ItemsFile        string            `json:"itemsFile"`
	MapCount         int               `json:"mapCount"`
	TankCount        int               `json:"tankCount"`
	ShellTypeCount   int               `json:"shellTypeCount"`
	ItemCount        int               `json:"itemCount"`
	MissingItemIcons []string          `json:"missingItemIcons"`
	DataOnly         bool              `json:"dataOnly"`
	IconExport       *IconExportReport `json:"iconExport"`
	ModelExport      *TankExportReport `json:"modelExport,omitempty"`
	ElapsedSeconds   jsonx.F64         `json:"elapsedSeconds"`
}

// ExportClient exports client maps, full tank metadata, items, icons, and
// optionally the model bundles in one run. Resource lists and localization
// are shared with model discovery, and the metadata files use the legacy
// exporter's schema.
func ExportClient(options ClientExportOptions) (*ClientExportReport, error) {
	started := time.Now()
	resolved, err := client.ResolveGameDir(options.GameDir)
	if err != nil {
		return nil, err
	}
	catalogStarted := time.Now()
	catalog, err := client.DiscoverClientCatalog(resolved.ResourceDir)
	if err != nil {
		return nil, err
	}
	discoverySeconds := time.Since(catalogStarted).Seconds()

	if err := os.MkdirAll(options.OutputDir, 0o777); err != nil {
		return nil, blitz.IOError("could not create export directory", options.OutputDir, err)
	}
	outputDir, err := pathx.Canonicalize(options.OutputDir)
	if err != nil {
		return nil, blitz.IOError("could not resolve export directory", options.OutputDir, err)
	}
	mapsFile := filepath.Join(outputDir, "maps.json")
	tanksFile := filepath.Join(outputDir, "tanks.json")
	maps := catalog.Maps
	if maps == nil {
		maps = []client.ClientMap{}
	}
	if err := writeJSON(mapsFile, maps, false); err != nil {
		return nil, err
	}
	if err := writeJSON(tanksFile, catalog.Tanks, false); err != nil {
		return nil, err
	}

	report := &ClientExportReport{
		OutputDir: outputDir,
		MapsFile:  mapsFile,
		TanksFile: tanksFile,
		MapCount:  len(catalog.Maps),
		TankCount: len(catalog.Tanks),
		DataOnly:  options.DataOnly,
	}
	sprites, err := client.ReadItemIconSprites(resolved.ResourceDir)
	if err != nil {
		return nil, err
	}
	icons, index, err := exportIcons(resolved.ResourceDir, outputDir, iconSets, itemIconReferences(catalog.Items, sprites))
	if err != nil {
		return nil, err
	}
	report.IconExport = icons
	items, missing := linkItemIcons(catalog.Items, sprites, index)
	report.ItemsFile = filepath.Join(outputDir, "items.json")
	if err := writeJSON(report.ItemsFile, items, false); err != nil {
		return nil, err
	}
	report.ShellTypeCount = len(items.Shells)
	report.ItemCount = len(items.Consumables) + len(items.Provisions) + len(items.Equipment)
	report.MissingItemIcons = missing
	if !options.DataOnly {
		modelOptions := options.ModelOptions
		modelOptions.GameDir = resolved.GameDir
		modelOptions.ViewerDir = outputDir
		if report.ModelExport, err = exportTankCatalogFromDiscovered(modelOptions, resolved.ResourceDir,
			catalog.ModelTanks, discoverySeconds, started); err != nil {
			return nil, err
		}
	}
	report.ElapsedSeconds = jsonx.F64(time.Since(started).Seconds())
	return report, nil
}
