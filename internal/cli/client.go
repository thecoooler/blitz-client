package cli

import (
	"math"
	"slices"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/export"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/rs"
)

func (a *app) runClient(args []string) (bool, error) {
	if len(args) > 0 && args[0] == "locate" {
		positional, gameDir, err := takeGameDir(args[1:])
		if err != nil {
			return false, err
		}
		if len(positional) > 0 {
			return false, usageError("client locate accepts no positional arguments")
		}
		resolved, err := client.ResolveGameDir(gameDir)
		if err != nil {
			return false, err
		}
		if a.json {
			a.printJSON(resolved)
		} else {
			a.println(resolved.GameDir)
			a.println("resource directory: " + resolved.ResourceDir)
			if resolved.CacheDir != nil {
				a.println("cache directory: " + *resolved.CacheDir)
			}
			if resolved.ReplayDir != nil {
				a.println("replay directory: " + *resolved.ReplayDir)
			}
		}
		return true, nil
	}
	switch {
	case len(args) >= 2 && args[0] == "files" && args[1] == "export":
		return a.runFilesExport(args[2:])
	case len(args) >= 1 && args[0] == "export":
		return a.runClientExport(args[1:])
	case len(args) >= 2 && args[0] == "tanks" && args[1] == "list":
		return a.runTankList(args[2:])
	case len(args) >= 2 && args[0] == "tanks" && args[1] == "export":
		return a.runTankExport(args[2:])
	}
	return false, usageError("expected client locate, client export, client files export, or client tanks list/export")
}

func (a *app) runFilesExport(args []string) (bool, error) {
	options := export.NewClientFilesExportOptions(".")
	outputDir := ""
	hasOutput := false
	var types []export.ClientFileType
	hasEmptyCache := false
	for index := 0; index < len(args); index++ {
		switch arg := args[index]; arg {
		case "--game-dir", "--cache-dir", "--output-dir":
			value, err := optionValue(args, &index, arg+" requires a path")
			if err != nil {
				return false, err
			}
			switch arg {
			case "--game-dir":
				if options.GameDir, err = gameDirOption(value); err != nil {
					return false, err
				}
			case "--cache-dir":
				if value == "" {
					hasEmptyCache = true
				}
				options.CacheDir = value
			default:
				outputDir, hasOutput = value, true
			}
		case "--exclude":
			value, err := optionValue(args, &index, "--exclude requires a path pattern")
			if err != nil {
				return false, err
			}
			options.Exclude = append(options.Exclude, value)
		case "--types":
			value, err := optionValue(args, &index, "--types requires comma-separated file types")
			if err != nil {
				return false, err
			}
			for _, name := range strings.Split(value, ",") {
				kind, err := export.ParseClientFileType(name)
				if err != nil {
					return false, usageError(blitz.AsError(err).Message)
				}
				if !slices.Contains(types, kind) {
					types = append(types, kind)
				}
			}
		case "--no-cache":
			options.NoCache = true
		case "--dry-run":
			options.DryRun = true
		default:
			if strings.HasPrefix(arg, "-") {
				return false, usageError("unknown option: " + arg)
			}
			return false, usageError("unexpected argument: " + arg)
		}
	}
	if options.NoCache && (options.CacheDir != "" || hasEmptyCache) {
		return false, usageError("--cache-dir and --no-cache cannot be combined")
	}
	if !hasOutput {
		return false, usageError("client files export requires --output-dir <path>")
	}
	options.OutputDir = outputDir
	if len(types) > 0 {
		options.Types = types
	}
	if hasEmptyCache {
		return false, blitz.ErrorAt("FILE_EXPORT_SOURCE_NOT_FOUND", "source directory does not exist", "")
	}
	report, err := export.ExportClientFilesWithProgress(options, func(report *export.ClientFilesExportReport) error {
		processed := report.WrittenCount + report.UnchangedCount
		if !a.json && !report.DryRun && (processed == 0 || processed%1000 == 0 || processed == report.FileCount) {
			a.eprintf("Files: %d/%d; written: %d, unchanged: %d\n", processed, report.FileCount, report.WrittenCount, report.UnchangedCount)
		}
		return nil
	})
	if err != nil {
		return false, err
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
	return true, nil
}

func (a *app) runTankList(args []string) (bool, error) {
	var positional []string
	gameDir, nation := "", ""
	hasNation := false
	for index := 0; index < len(args); index++ {
		switch arg := args[index]; {
		case arg == "--game-dir":
			value, err := optionValue(args, &index, "--game-dir requires a path")
			if err != nil {
				return false, err
			}
			if gameDir, err = gameDirOption(value); err != nil {
				return false, err
			}
		case arg == "--nation":
			value, err := optionValue(args, &index, "--nation requires a nation name")
			if err != nil {
				return false, err
			}
			nation, hasNation = asciiLower(value), true
		case strings.HasPrefix(arg, "-"):
			return false, usageError("unknown option: " + arg)
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) > 0 {
		return false, usageError("client tanks list accepts no positional arguments")
	}
	if hasNation && !client.IsKnownNation(nation) {
		return false, usageError("unknown nation " + rs.Quote(nation))
	}
	tanks, err := client.ListClientTanks(gameDir)
	if err != nil {
		return false, err
	}
	if hasNation {
		tanks = slices.DeleteFunc(tanks, func(tank client.ClientTank) bool { return tank.Nation != nation })
	}
	if a.json {
		a.printJSON(jsonx.Object{"count": len(tanks), "tanks": jsonx.ToValue(tanks)})
		return true, nil
	}
	for _, tank := range tanks {
		level := "?"
		if tank.Level != nil {
			level = *tank.Level
		}
		a.printf("%s\t%s\t%s\t%s\n", tank.Nation, tank.ID, level, tank.Name)
	}
	a.printf("%d tanks\n", len(tanks))
	return true, nil
}

func (a *app) runTankExport(args []string) (bool, error) {
	options, viewerDir, _, err := parseModelExportOptions(args, false)
	if err != nil {
		return false, err
	}
	if viewerDir == nil {
		return false, usageError("client tanks export requires --viewer-dir <path>")
	}
	options.ViewerDir = *viewerDir
	report, err := export.ExportTankCatalog(options)
	if err != nil {
		return false, err
	}
	if a.json {
		a.printJSON(report)
	} else {
		a.printf("Exported %d tank model(s); %d failed\n", report.ExportedCount, report.FailedCount)
		a.printf("catalog: %s\n", report.Catalog)
		a.printf("report: %s\n", report.ExportReport)
		for _, failure := range report.Failures {
			a.eprintf("FAIL %s: %s\n", failure.ID, failure.Error)
		}
	}
	return true, nil
}

func (a *app) runClientExport(args []string) (bool, error) {
	modelOptions, outputDir, dataOnly, err := parseModelExportOptions(args, true)
	if err != nil {
		return false, err
	}
	if outputDir == nil {
		return false, usageError("client export requires --output-dir <path>")
	}
	options := export.NewClientExportOptions(*outputDir)
	options.GameDir = modelOptions.GameDir
	options.ModelOptions = modelOptions
	options.DataOnly = dataOnly
	report, err := export.ExportClient(options)
	if err != nil {
		return false, err
	}
	if a.json {
		a.printJSON(report)
		return true, nil
	}
	a.printf("Exported %d map(s) and %d tank record(s)\n", report.MapCount, report.TankCount)
	a.printf("maps: %s\n", report.MapsFile)
	a.printf("tanks: %s\n", report.TanksFile)
	a.printf("Exported %d shell type(s) and %d item(s); %d icon key(s) without an icon\n",
		report.ShellTypeCount, report.ItemCount, len(report.MissingItemIcons))
	a.printf("items: %s\n", report.ItemsFile)
	if icons := report.IconExport; icons != nil {
		a.printf("Exported %d icon(s): %d written, %d unchanged, %d removed\n",
			icons.IconCount, icons.WrittenCount, icons.UnchangedCount, icons.RemovedCount)
		a.printf("icon catalog: %s\n", icons.Catalog)
	}
	if models := report.ModelExport; models != nil {
		a.printf("Exported %d tank model(s); %d failed\n", models.ExportedCount, models.FailedCount)
		a.printf("model catalog: %s\n", models.Catalog)
		a.printf("model report: %s\n", models.ExportReport)
		for _, failure := range models.Failures {
			a.eprintf("FAIL %s: %s\n", failure.ID, failure.Error)
		}
	}
	return true, nil
}

func parseModelExportOptions(args []string, unified bool) (export.TankExportOptions, *string, bool, error) {
	options := export.NewTankExportOptions(".")
	var outputDir *string
	dataOnly := false
	fail := func(err error) (export.TankExportOptions, *string, bool, error) {
		return export.TankExportOptions{}, nil, false, err
	}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--game-dir":
			value, err := optionValue(args, &index, "--game-dir requires a path")
			if err != nil {
				return fail(err)
			}
			if options.GameDir, err = gameDirOption(value); err != nil {
				return fail(err)
			}
		case arg == "--viewer-dir" && !unified, arg == "--output-dir" && unified:
			value, err := optionValue(args, &index, arg+" requires a path")
			if err != nil {
				return fail(err)
			}
			outputDir = &value
		case arg == "--data-only" && unified:
			dataOnly = true
		case arg == "--tank":
			value, err := optionValue(args, &index, "--tank requires an ID")
			if err != nil {
				return fail(err)
			}
			options.Tanks = append(options.Tanks, value)
		case arg == "--nation":
			value, err := optionValue(args, &index, "--nation requires a name")
			if err != nil {
				return fail(err)
			}
			options.Nation = asciiLower(value)
		case arg == "--limit":
			value, err := optionValue(args, &index, "--limit requires a number")
			if err != nil {
				return fail(err)
			}
			limit, err := rs.ParseUint(value, 64)
			if err != nil {
				return fail(usageError("--limit must be a non-negative integer"))
			}
			options.Limit = int(min(limit, uint64(math.MaxInt)))
		case arg == "--max-texture-size":
			value, err := optionValue(args, &index, "--max-texture-size requires a size")
			if err != nil {
				return fail(err)
			}
			if options.MaxTextureSize, err = parseTextureSize(value); err != nil {
				return fail(err)
			}
		case arg == "--texture-format":
			value, err := optionValue(args, &index, "--texture-format requires webp or png")
			if err != nil {
				return fail(err)
			}
			options.TextureFormat = value
		case arg == "--webp-quality":
			value, err := optionValue(args, &index, "--webp-quality requires a number")
			if err != nil {
				return fail(err)
			}
			quality, err := rs.ParseUint(value, 8)
			if err != nil {
				return fail(usageError("--webp-quality must be from 0 to 100"))
			}
			options.WebPQuality = uint8(quality)
		case arg == "--pbr":
			options.IncludePBR = true
		case arg == "--no-compress":
			options.Compress = false
		case arg == "--force":
			options.ForceTextures = true
		case arg == "--prune":
			options.Prune = true
		case strings.HasPrefix(arg, "-"):
			return fail(usageError("unknown option: " + arg))
		default:
			return fail(usageError("unexpected argument: " + arg))
		}
	}
	return options, outputDir, dataOnly, nil
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
