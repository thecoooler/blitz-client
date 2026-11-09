package cli

import (
	"slices"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/export"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/rs"
)

func (c LocateCmd) Run(a *app) error {
	gameDir := derefFlag(c.GameDir)
	resolved, err := client.ResolveGameDir(gameDir)
	if err != nil {
		return err
	}
	if a.json {
		a.printJSON(resolved)
		return nil
	}
	a.println(resolved.GameDir)
	a.println("resource directory: " + resolved.ResourceDir)
	if resolved.CacheDir != nil {
		a.println("cache directory: " + *resolved.CacheDir)
	}
	if resolved.ReplayDir != nil {
		a.println("replay directory: " + *resolved.ReplayDir)
	}
	return nil
}

func (c FilesUnpackCmd) Run(a *app) error {
	options := export.NewClientFilesExportOptions(c.OutputDir)
	options.GameDir, options.CacheDir, options.NoCache = clientSourceFlags(c.GameDir, c.CacheDir, c.NoCache)
	if len(c.Exclude) > 0 {
		options.Exclude = c.Exclude
	}
	if len(c.Types) > 0 {
		options.Types = c.Types
	}
	options.DryRun = c.DryRun
	if c.CacheDir != nil && *c.CacheDir == "" {
		return blitz.ErrorAt("FILE_EXPORT_SOURCE_NOT_FOUND", "source directory does not exist", "")
	}
	report, err := export.ExportClientFilesWithProgress(options, func(report *export.ClientFilesExportReport) error {
		processed := report.WrittenCount + report.UnchangedCount
		if !a.json && !report.DryRun && (processed == 0 || processed%1000 == 0 || processed == report.FileCount) {
			a.eprintf("Files: %d/%d; written: %d, unchanged: %d\n", processed, report.FileCount, report.WrittenCount, report.UnchangedCount)
		}
		return nil
	})
	if err != nil {
		return err
	}
	switch {
	case a.json:
		a.printJSON(report)
	case report.DryRun:
		a.printf("Dry run: %d file(s), %d bytes (%.2f MiB) on disk\n", report.FileCount, report.SourceBytes, float64(report.SourceBytes)/1_048_576)
		a.printf("Selected %d DVPL file(s); %d file(s) from cache\n", report.DVPLCount, report.CacheFileCount)
	default:
		a.printf("Exported %d file(s): %d written, %d unchanged\n", report.FileCount, report.WrittenCount, report.UnchangedCount)
		a.printf("Unpacked %d DVPL file(s); %d file(s) from cache\n", report.DVPLCount, report.CacheFileCount)
		a.printf("Output: %s\n", report.OutputDir)
	}
	if !a.json && len(report.Exclude) > 0 {
		a.printf("Excluded %d file(s) by path pattern\n", report.ExcludedFileCount)
	}
	return nil
}

func (c TanksListCmd) Run(a *app) error {
	nation := ""
	if c.Nation != "" {
		nation = asciiLower(c.Nation)
		if !client.IsKnownNation(nation) {
			return usageError("unknown nation " + rs.Quote(nation))
		}
	}
	tanks, err := client.ListClientTanks(derefFlag(c.GameDir))
	if err != nil {
		return err
	}
	if nation != "" {
		tanks = slices.DeleteFunc(tanks, func(tank client.ClientTank) bool { return tank.Nation != nation })
	}
	if a.json {
		a.printJSON(jsonx.Object{"count": len(tanks), "tanks": jsonx.ToValue(tanks)})
		return nil
	}
	for _, tank := range tanks {
		level := "?"
		if tank.Level != nil {
			level = *tank.Level
		}
		a.printf("%s\t%s\t%s\t%s\n", tank.Nation, tank.ID, level, tank.Name)
	}
	a.printf("%d tanks\n", len(tanks))
	return nil
}

func (c ClientExportCmd) Run(a *app) error {
	o := export.NewClientExportOptions(c.OutputDir)
	o.GameDir, o.CacheDir, o.NoCache = clientSourceFlags(c.GameDir, c.CacheDir, c.NoCache)
	if len(c.Include) > 0 {
		o.Include = c.Include
	}
	o.ServerConfig = c.ServerConfig
	o.MapsConfig = c.MapsConfig
	c.MaxTextureSize.apply(&o.ModelOptions)
	if c.TextureFormat != nil {
		o.ModelOptions.TextureFormat = *c.TextureFormat
	}
	if c.WebPQuality != nil {
		if *c.WebPQuality > 100 {
			return usageError("--webp-quality must be from 0 to 100")
		}
		o.ModelOptions.WebPQuality = *c.WebPQuality
	}
	if c.NoCompress {
		o.ModelOptions.Compress = false
	}
	if c.NoPBR {
		o.ModelOptions.IncludePBR = false
	}
	report, err := export.ExportClient(o)
	if err != nil {
		return err
	}
	if a.json {
		a.printJSON(report)
	} else {
		a.printf("Exported %d tanks, %d models (%s)\n", report.Tanks, report.Models, strings.Join(report.Included, ","))
		if report.MechanicExport != nil {
			a.printf("Mechanics: %d\n", report.MechanicExport.Mechanics)
		}
		if game := report.Game; game != nil && game.VersionText != nil {
			a.printf("Game: %s\n", *game.VersionText)
		}
		a.printf("Snapshot: %s\n", report.File)
		if len(report.Issues) > 0 {
			a.printf("Media issues: %d (see data.json)\n", len(report.Issues))
		}
	}
	return nil
}

func (t textureSize) apply(options *export.TankExportOptions) {
	switch {
	case t.original:
		options.MaxTextureSize = nil
	case t.width != 0:
		options.MaxTextureSize = &export.TextureSize{Width: t.width, Height: t.height}
	}
}

// clientSourceFlags reads the game and cache flags shared by the client export
// commands; nil means the flag was not given.
func clientSourceFlags(gameDir, cacheDir *string, noCache bool) (string, string, bool) {
	return derefFlag(gameDir), derefFlag(cacheDir), noCache
}

// derefFlag reads an optional flag value; nil means the flag was not given.
func derefFlag(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func parseTextureSize(value string) (*export.TextureSize, error) {
	normalized := strings.ReplaceAll(asciiLower(strings.TrimSpace(value)), "×", "x")
	if normalized == "original" {
		return nil, nil
	}
	parse := func(text, message string) (uint32, error) {
		number, err := rs.ParseUint(text, 32)
		if err != nil {
			return 0, usageError(message)
		}
		return uint32(number), nil
	}
	var width, height uint32
	var err error
	if w, h, ok := strings.Cut(normalized, "x"); ok {
		if width, err = parse(w, "texture width must be a positive integer"); err != nil {
			return nil, err
		}
		if height, err = parse(h, "texture height must be a positive integer"); err != nil {
			return nil, err
		}
	} else {
		if width, err = parse(normalized, "texture size must be a positive integer"); err != nil {
			return nil, err
		}
		height = width
	}
	if width == 0 || height == 0 {
		return nil, usageError("texture size must be N or WIDTHxHEIGHT with positive dimensions")
	}
	return &export.TextureSize{Width: width, Height: height}, nil
}

func asciiLower(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 'a' - 'A'
		}
		return r
	}, value)
}
