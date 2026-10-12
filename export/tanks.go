package export

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/lossy"
	"github.com/thecoooler/blitz-client/internal/pathx"
	"github.com/thecoooler/blitz-client/internal/rs"
	"github.com/thecoooler/blitz-client/resources"
)

const catalogSource = "Installed client XML/item_defs/vehicles/*/list.xml and paired 3d/Tanks SC2/SCG assets"

// TextureSize is a maximum texture width and height.
type TextureSize struct {
	Width, Height uint32
}

// TankExportOptions configures ExportTankCatalog.
type TankExportOptions struct {
	// GameDir is the install directory or its Data directory; empty searches defaults.
	GameDir string
	// ViewerDir is the viewer project root; assets are written under assets/tanks.
	ViewerDir string
	// Limit of 0 means all tanks. Ignored when specific tank IDs are supplied.
	Limit int
	// Tanks holds tank IDs or nation/ID keys.
	Tanks []string
	// Nation restricts automatic discovery to one nation.
	Nation string
	// MaxTextureSize nil preserves source dimensions.
	MaxTextureSize *TextureSize
	TextureFormat  string
	WebPQuality    uint8
	IncludePBR     bool
	Compress       bool
	ForceTextures  bool
	// Prune removes old tank bundles not exported by this invocation.
	Prune bool
}

// NewTankExportOptions returns the default model export options.
func NewTankExportOptions(viewerDir string) TankExportOptions {
	return TankExportOptions{
		ViewerDir:      viewerDir,
		MaxTextureSize: &TextureSize{1024, 1024},
		TextureFormat:  "webp",
		WebPQuality:    90,
		Compress:       true,
	}
}

// TankExportReport summarizes a model export.
type TankExportReport struct {
	Catalog                   string              `json:"catalog"`
	ExportReport              string              `json:"exportReport"`
	DiscoveredClientTankCount int                 `json:"discoveredClientTankCount"`
	ExportedCount             int                 `json:"exportedCount"`
	FailedCount               int                 `json:"failedCount"`
	Failures                  []TankExportFailure `json:"failures"`
	Entries                   []map[string]any    `json:"entries"`
	ElapsedSeconds            jsonx.F64           `json:"elapsedSeconds"`
}

// TankExportFailure describes one tank that could not be exported.
type TankExportFailure struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

type exportSummary struct {
	StartedAt                 string              `json:"startedAt"`
	FinishedAt                string              `json:"finishedAt"`
	DataRoot                  string              `json:"dataRoot"`
	DiscoveredClientTankCount int                 `json:"discoveredClientTankCount"`
	DiscoverySeconds          jsonx.F64           `json:"discoverySeconds"`
	RequestedLimit            any                 `json:"requestedLimit"`
	MaxTextureSize            string              `json:"maxTextureSize"`
	TextureFormat             string              `json:"textureFormat"`
	WebPQuality               *uint8              `json:"webpQuality"`
	PBREnabled                bool                `json:"pbrEnabled"`
	Compression               string              `json:"compression"`
	ForceTextures             bool                `json:"forceTextures"`
	ExportedCount             int                 `json:"exportedCount"`
	FailedCount               int                 `json:"failedCount"`
	TotalElapsedSeconds       jsonx.F64           `json:"totalElapsedSeconds"`
	AverageTankSeconds        *jsonx.F64          `json:"averageTankSeconds"`
	Catalog                   string              `json:"catalog"`
	PerTank                   []map[string]any    `json:"perTank"`
	Failures                  []TankExportFailure `json:"failures"`
}

// ExportTankCatalog exports client tank models into the viewer's
// asset/catalog format. Geometry decoding, transforms, and binary bundles are
// implemented here; image and Brotli codecs use the platform's ffmpeg, cwebp,
// and brotli executables.
func ExportTankCatalog(options TankExportOptions) (*TankExportReport, error) {
	if err := validateTankOptions(&options); err != nil {
		return nil, err
	}
	totalStarted := time.Now()
	resolved, err := client.ResolveGameDir(options.GameDir)
	if err != nil {
		return nil, err
	}
	discoveryStarted := time.Now()
	discovered, err := client.DiscoverClientTanks(resolved.ResourceDir)
	if err != nil {
		return nil, err
	}
	return exportTankCatalogFromDiscovered(options, resolved.ResourceDir, discovered,
		time.Since(discoveryStarted).Seconds(), totalStarted)
}

func exportTankCatalogFromDiscovered(options TankExportOptions, resourceDir string, discovered []client.ClientTank,
	discoverySeconds float64, totalStarted time.Time) (*TankExportReport, error) {
	if err := validateTankOptions(&options); err != nil {
		return nil, err
	}
	ordered, err := selectTanks(discovered, &options)
	if err != nil {
		return nil, err
	}
	requested := -1
	if len(options.Tanks) > 0 {
		requested = len(options.Tanks)
	} else if options.Limit != 0 {
		requested = options.Limit
	}

	viewerRoot, err := canonicalOrAbsolute(options.ViewerDir)
	if err != nil {
		return nil, err
	}
	outputRoot := filepath.Join(viewerRoot, "assets/tanks")
	if err := os.MkdirAll(outputRoot, 0o777); err != nil {
		return nil, blitz.IOError("could not create viewer asset directory", outputRoot, err)
	}

	startedAt := isoTimestamp(time.Now())
	entries := []map[string]any{}
	perTank := []map[string]any{}
	failures := []TankExportFailure{}
	attempted := map[[2]string]bool{}
	for _, tank := range ordered {
		if requested >= 0 && len(entries) >= requested {
			break
		}
		key := [2]string{tank.Nation, tank.ID}
		if attempted[key] {
			continue
		}
		attempted[key] = true
		entry, measurement, err := exportOne(&tank, resourceDir, outputRoot, &options)
		if err != nil {
			failures = append(failures, TankExportFailure{ID: tank.Nation + "/" + tank.ID, Error: err.Error()})
			continue
		}
		entries = append(entries, entry)
		perTank = append(perTank, measurement)
	}

	if options.Prune {
		if err := pruneOldBundles(outputRoot, entries); err != nil {
			return nil, err
		}
	}
	catalogPath := filepath.Join(outputRoot, "catalog.json")
	reportPath := filepath.Join(outputRoot, "export-report.json")
	catalog := object{
		"formatVersion": 1,
		"source":        catalogSource,
		"generatedAt":   startedAt,
		"count":         len(entries),
		"entries":       entries,
	}
	if err := writeJSON(catalogPath, catalog, false); err != nil {
		return nil, err
	}

	elapsed := time.Since(totalStarted).Seconds()
	var requestedLimit any = "all"
	if requested >= 0 {
		requestedLimit = requested
	}
	catalogRelative := catalogPath
	if relative, err := filepath.Rel(viewerRoot, catalogPath); err == nil && !strings.HasPrefix(relative, "..") {
		catalogRelative = relative
	}
	catalogRelative = strings.ReplaceAll(catalogRelative, `\`, "/")
	summary := exportSummary{
		StartedAt:                 startedAt,
		FinishedAt:                isoTimestamp(time.Now()),
		DataRoot:                  resourceDir,
		DiscoveredClientTankCount: len(discovered),
		DiscoverySeconds:          jsonx.F64(discoverySeconds),
		RequestedLimit:            requestedLimit,
		MaxTextureSize:            textureSizeLabel(options.MaxTextureSize),
		TextureFormat:             options.TextureFormat,
		PBREnabled:                options.IncludePBR,
		Compression:               compressionLabel(options.Compress),
		ForceTextures:             options.ForceTextures,
		ExportedCount:             len(perTank),
		FailedCount:               len(failures),
		TotalElapsedSeconds:       jsonx.F64(elapsed),
		Catalog:                   catalogRelative,
		PerTank:                   perTank,
		Failures:                  failures,
	}
	if options.TextureFormat == "webp" {
		quality := options.WebPQuality
		summary.WebPQuality = &quality
	}
	if len(perTank) > 0 {
		var sum float64
		for _, item := range perTank {
			if value, ok := item["elapsedSeconds"].(jsonx.F64); ok {
				sum += float64(value)
			}
		}
		average := jsonx.F64(sum / float64(len(perTank)))
		summary.AverageTankSeconds = &average
	}
	if err := writeJSON(reportPath, summary, false); err != nil {
		return nil, err
	}

	report := &TankExportReport{
		Catalog:                   catalogPath,
		ExportReport:              reportPath,
		DiscoveredClientTankCount: len(discovered),
		ExportedCount:             len(entries),
		FailedCount:               len(failures),
		Failures:                  failures,
		Entries:                   entries,
		ElapsedSeconds:            jsonx.F64(elapsed),
	}
	if requested >= 0 && report.ExportedCount < requested {
		return nil, blitz.NewError("TANK_EXPORT_INCOMPLETE",
			fmt.Sprintf("exported %d of %d requested tanks; see %s", report.ExportedCount, requested, report.ExportReport))
	}
	return report, nil
}

func compressionLabel(compress bool) string {
	if compress {
		return "brotli-9"
	}
	return "none"
}

func validateTankOptions(options *TankExportOptions) error {
	if options.TextureFormat != "png" && options.TextureFormat != "webp" {
		return blitz.NewError("USAGE", "texture format must be webp or png")
	}
	if options.WebPQuality > 100 {
		return blitz.NewError("USAGE", "WebP quality must be from 0 to 100")
	}
	if size := options.MaxTextureSize; size != nil && (size.Width == 0 || size.Height == 0) {
		return blitz.NewError("USAGE", "texture dimensions must be positive")
	}
	if options.Nation != "" && !client.IsKnownNation(options.Nation) {
		return blitz.NewError("USAGE", "unknown nation "+rs.Quote(options.Nation))
	}
	return nil
}

func selectTanks(discovered []client.ClientTank, options *TankExportOptions) ([]client.ClientTank, error) {
	if len(options.Tanks) > 0 {
		out := make([]client.ClientTank, 0, len(options.Tanks))
		for _, requested := range options.Tanks {
			tank, err := resolveTank(discovered, requested)
			if err != nil {
				return nil, err
			}
			out = append(out, tank)
		}
		return out, nil
	}
	var candidates []client.ClientTank
	for _, tank := range discovered {
		if options.Nation == "" || options.Nation == tank.Nation {
			candidates = append(candidates, tank)
		}
	}
	return benchmarkOrder(candidates), nil
}

func resolveTank(discovered []client.ClientTank, requested string) (client.ClientTank, error) {
	nation, id, qualified := strings.Cut(requested, "/")
	if !qualified {
		nation, id, qualified = strings.Cut(requested, ":")
	}
	if qualified {
		nation = asciiLower(nation)
		if !client.IsKnownNation(nation) {
			return client.ClientTank{}, blitz.NewError("TANK_NOT_FOUND", "unknown nation "+rs.Quote(nation))
		}
	} else {
		id = requested
	}
	var matches []client.ClientTank
	for _, tank := range discovered {
		if tank.ID == id && (!qualified || tank.Nation == nation) {
			matches = append(matches, tank)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return client.ClientTank{}, blitz.NewError("TANK_NOT_FOUND",
			fmt.Sprintf("tank %s is not present with a model in this client", rs.Quote(requested)))
	}
	nations := make([]string, len(matches))
	for index, tank := range matches {
		nations[index] = tank.Nation
	}
	return client.ClientTank{}, blitz.NewError("TANK_ID_AMBIGUOUS",
		fmt.Sprintf("tank ID %s is ambiguous (%s); use nation/%s", rs.Quote(id), strings.Join(nations, ", "), id))
}

func levelNumber(tank *client.ClientTank) int64 {
	text := "0"
	if tank.Level != nil {
		text = *tank.Level
	}
	value, err := rs.ParseInt(text, 32)
	if err != nil {
		return 0
	}
	return value
}

func benchmarkOrder(candidates []client.ClientTank) []client.ClientTank {
	var ordered []client.ClientTank
	byNation := map[string][]client.ClientTank{}
	foundBZ := false
	for _, tank := range candidates {
		if !foundBZ && tank.Nation == "china" && tank.ID == "Ch48_BZ_75" {
			foundBZ = true
			ordered = append(ordered, tank)
			continue
		}
		byNation[tank.Nation] = append(byNation[tank.Nation], tank)
	}
	for _, tanks := range byNation {
		sort.SliceStable(tanks, func(i, j int) bool {
			left, right := levelNumber(&tanks[i]), levelNumber(&tanks[j])
			if left != right {
				return left > right
			}
			return asciiLower(tanks[i].ID) < asciiLower(tanks[j].ID)
		})
	}
	for {
		moved := false
		for _, nation := range client.Nations() {
			if group := byNation[nation.ID]; len(group) > 0 {
				ordered = append(ordered, group[0])
				byNation[nation.ID] = group[1:]
				moved = true
			}
		}
		if !moved {
			return ordered
		}
	}
}

func exportOne(tank *client.ClientTank, dataRoot, outputRoot string, options *TankExportOptions) (map[string]any, map[string]any, error) {
	started := time.Now()
	bundle := filepath.Join(outputRoot, tank.Nation, tank.ID)
	if err := os.MkdirAll(bundle, 0o777); err != nil {
		return nil, nil, fmt.Errorf("could not create %s: %s", bundle, blitz.IOMessage(err))
	}
	readStarted := time.Now()
	sc2Data, err := readDecoded(tank.ModelFiles.Scene)
	if err != nil {
		return nil, nil, err
	}
	scgData, err := readDecoded(tank.ModelFiles.Geometry)
	if err != nil {
		return nil, nil, err
	}
	readSeconds := time.Since(readStarted).Seconds()
	geometryStarted := time.Now()
	model, err := exportModel(sc2Data, scgData, tank.ModelFiles.Scene, tank.ModelFiles.Geometry, tank.ID,
		options.IncludePBR, options.TextureFormat, options.WebPQuality, options.Compress)
	if err != nil {
		return nil, nil, err
	}
	geometrySeconds := time.Since(geometryStarted).Seconds()

	modelJSON, err := jsonx.Marshal(model.metadata)
	if err != nil {
		return nil, nil, err
	}
	modelJSONStats, err := writeCompressedAsset(filepath.Join(bundle, "model.json"), modelJSON, options.Compress)
	if err != nil {
		return nil, nil, err
	}
	modelBinaryStats, err := writeCompressedAsset(filepath.Join(bundle, "model.bin"), model.binary, options.Compress)
	if err != nil {
		return nil, nil, err
	}
	var pbrBinaryStats assetStats
	if model.pbrBinary != nil {
		if pbrBinaryStats, err = writeCompressedAsset(filepath.Join(bundle, "model.pbr.bin"), model.pbrBinary, options.Compress); err != nil {
			return nil, nil, err
		}
	} else {
		removeAsset(filepath.Join(bundle, "model.pbr.bin"))
	}

	armorStarted := time.Now()
	armorPath := filepath.Join(bundle, "armor.json")
	armorMeshPath := filepath.Join(bundle, "armor-mesh.json")
	removeAsset(armorPath)
	removeAsset(armorMeshPath)
	collisionSC2 := client.ResolveClientAsset(dataRoot, fmt.Sprintf("3d/Tanks/CollisionMeshes/%s-%s.sc2.dvpl", tank.Nation, tank.ID))
	collisionSCG := client.ResolveClientAsset(dataRoot, fmt.Sprintf("3d/Tanks/CollisionMeshes/%s-%s.scg.dvpl", tank.Nation, tank.ID))
	vehicleXML := client.ResolveClientAsset(dataRoot, fmt.Sprintf("XML/item_defs/vehicles/%s/%s.xml.dvpl", tank.Nation, tank.ID))
	hasArmor := false
	var armorNote any
	var armorStats, armorMeshStats assetStats
	if collisionSC2 != "" && collisionSCG != "" && vehicleXML != "" {
		values, mesh, err := exportArmorFiles(collisionSC2, collisionSCG, vehicleXML, tank, armorPath, armorMeshPath, options.Compress)
		if err != nil {
			armorNote = err.Error()
		} else {
			hasArmor = true
			armorStats, armorMeshStats = values, mesh
		}
	}
	armorSeconds := time.Since(armorStarted).Seconds()

	textureStarted := time.Now()
	modelDirectory := filepath.Dir(tank.ModelFiles.Scene)
	seenTextures := map[string]bool{}
	textureConversions, placeholderTextures := 0, 0
	for _, texture := range model.textures {
		if seenTextures[texture.file] {
			continue
		}
		seenTextures[texture.file] = true
		target := filepath.Join(bundle, texture.file)
		if texture.source == "" {
			if err := writeFallbackPNG(target, texture.role); err != nil {
				return nil, nil, err
			}
			continue
		}
		source := texture.source
		if !filepath.IsAbs(source) {
			source = filepath.Join(modelDirectory, source)
		}
		if info, err := os.Stat(source); err != nil || !info.Mode().IsRegular() {
			// Some client maps ship only as PVRTC variants, which ffmpeg cannot
			// decode; emit the placeholder color in the requested format so the
			// file still matches the extension the model declares.
			placeholderTextures++
			temp, err := os.MkdirTemp("", "blitz-tank-texture-")
			if err != nil {
				return nil, nil, fmt.Errorf("could not create temporary directory: %s", blitz.IOMessage(err))
			}
			placeholder := filepath.Join(temp, "placeholder.png")
			err = writeFallbackPNG(placeholder, texture.role)
			if err == nil {
				err = convertTexture(placeholder, target, options.MaxTextureSize, options.TextureFormat, options.WebPQuality, texture.lossless)
			}
			os.RemoveAll(temp)
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		if err := convertTexture(source, target, options.MaxTextureSize, options.TextureFormat, options.WebPQuality, texture.lossless); err != nil {
			return nil, nil, err
		}
		textureConversions++
	}
	textureSeconds := time.Since(textureStarted).Seconds()

	elapsed := time.Since(started).Seconds()
	bundleURL := fmt.Sprintf("./assets/tanks/%s/%s/", tank.Nation, tank.ID)
	suffix := ""
	if options.Compress {
		suffix = ".br"
	}
	fallbackMaps := model.fallbackMaps + placeholderTextures
	var armorURL, armorMeshURL, webp any
	if hasArmor {
		armorURL = bundleURL + "armor.json" + suffix
		armorMeshURL = bundleURL + "armor-mesh.json" + suffix
	}
	if options.TextureFormat == "webp" {
		webp = options.WebPQuality
	}
	common := map[string]any{
		"elapsedSeconds":                       roundMillis(elapsed),
		"geometrySeconds":                      roundMillis(geometrySeconds),
		"texturesSeconds":                      roundMillis(textureSeconds),
		"readModelSeconds":                     roundMillis(readSeconds),
		"armorSeconds":                         roundMillis(armorSeconds),
		"triangles":                            model.triangleCount,
		"vertices":                             model.vertexCount,
		"materials":                            model.materialCount,
		"fallbackMaps":                         fallbackMaps,
		"generatedUvGroups":                    model.generatedUVGroups,
		"modelJsonBytes":                       modelJSONStats.storedBytes,
		"modelJsonUncompressedBytes":           modelJSONStats.uncompressedBytes,
		"binaryGeometryBytes":                  modelBinaryStats.storedBytes,
		"binaryGeometryUncompressedBytes":      modelBinaryStats.uncompressedBytes,
		"pbrBinaryGeometryBytes":               pbrBinaryStats.storedBytes,
		"pbrBinaryGeometryUncompressedBytes":   pbrBinaryStats.uncompressedBytes,
		"totalBinaryGeometryBytes":             modelBinaryStats.storedBytes + pbrBinaryStats.storedBytes,
		"totalBinaryGeometryUncompressedBytes": modelBinaryStats.uncompressedBytes + pbrBinaryStats.uncompressedBytes,
		"textureCacheHits":                     0,
		"textureConversions":                   textureConversions,
		"textureFormat":                        options.TextureFormat,
		"webpQuality":                          webp,
		"hasArmor":                             hasArmor,
		"pbrEnabled":                           options.IncludePBR,
		"compression":                          compressionLabel(options.Compress),
		"armorJsonBytes":                       armorStats.storedBytes,
		"armorJsonUncompressedBytes":           armorStats.uncompressedBytes,
		"armorMeshBytes":                       armorMeshStats.storedBytes,
		"armorMeshUncompressedBytes":           armorMeshStats.uncompressedBytes,
	}
	entry := map[string]any{
		"id":         tank.Nation + "/" + tank.ID,
		"name":       tank.Name,
		"nation":     tank.NationLabel,
		"level":      tank.Level,
		"userString": tank.UserString,
		"bundle":     bundleURL,
		"model":      bundleURL + "model.json" + suffix,
		"armor":      armorURL,
		"armorMesh":  armorMeshURL,
		"armorNote":  armorNote,
	}
	measurement := map[string]any{"id": entry["id"], "armorError": armorNote}
	for key, value := range common {
		entry[key] = value
		measurement[key] = value
	}
	return entry, measurement, nil
}

func exportArmorFiles(sc2Path, scgPath, xmlPath string, tank *client.ClientTank, armorPath, meshPath string, compress bool) (assetStats, assetStats, error) {
	var inputs [3][]byte
	for index, path := range []string{sc2Path, scgPath, xmlPath} {
		data, err := readDecoded(path)
		if err != nil {
			return assetStats{}, assetStats{}, err
		}
		inputs[index] = data
	}
	result, err := exportArmor(inputs[0], inputs[1], inputs[2], tank.Nation, tank.ID, sc2Path, scgPath, xmlPath)
	if err != nil {
		return assetStats{}, assetStats{}, err
	}
	valuesJSON, err := jsonx.MarshalIndent(result.values)
	if err != nil {
		return assetStats{}, assetStats{}, err
	}
	meshJSON, err := jsonx.MarshalIndent(result.mesh)
	if err != nil {
		return assetStats{}, assetStats{}, err
	}
	valuesStats, err := writeCompressedAsset(armorPath, valuesJSON, compress)
	if err != nil {
		return assetStats{}, assetStats{}, err
	}
	meshStats, err := writeCompressedAsset(meshPath, meshJSON, compress)
	if err != nil {
		return assetStats{}, assetStats{}, err
	}
	return valuesStats, meshStats, nil
}

func readDecoded(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %s", path, blitz.IOMessage(err))
	}
	plain, _, err := resources.DecodeDVPL(data)
	if err != nil {
		return nil, fmt.Errorf("could not decode %s: %v", path, err)
	}
	return plain, nil
}

func convertTexture(source, destination string, maxSize *TextureSize, format string, quality uint8, lossless bool) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o777); err != nil {
		return fmt.Errorf("could not create %s: %s", filepath.Dir(destination), blitz.IOMessage(err))
	}
	sourceBytes, err := readDecoded(source)
	if err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "blitz-tank-texture-")
	if err != nil {
		return fmt.Errorf("could not create temporary directory: %s", blitz.IOMessage(err))
	}
	defer os.RemoveAll(temp)
	dds := filepath.Join(temp, "source.dds")
	png := filepath.Join(temp, "decoded.png")
	if err := os.WriteFile(dds, sourceBytes, 0o666); err != nil {
		return fmt.Errorf("could not stage DDS %s: %s", source, blitz.IOMessage(err))
	}
	args := []string{"-hide_banner", "-v", "error", "-y", "-i", dds}
	if maxSize != nil {
		args = append(args, "-vf", fmt.Sprintf(
			"scale=w='min(iw,%d)':h='min(ih,%d)':force_original_aspect_ratio=decrease", maxSize.Width, maxSize.Height))
	}
	args = append(args, "-frames:v", "1", "-update", "1", "-compression_level", "9", png)
	if stderr, err := run("ffmpeg", args...); err != nil {
		if stderr == nil {
			return fmt.Errorf("ffmpeg is required to decode DDS textures: %s", blitz.IOMessage(err))
		}
		return fmt.Errorf("ffmpeg failed to decode %s: %s", source, strings.TrimSpace(lossy.String(stderr)))
	}
	if format == "png" {
		data, err := os.ReadFile(png)
		if err == nil {
			err = os.WriteFile(destination, data, 0o666)
		}
		if err != nil {
			return fmt.Errorf("could not write %s: %s", destination, blitz.IOMessage(err))
		}
		return nil
	}
	args = []string{"-m", "6", "-alpha_q", "100", "-exact", "-metadata", "none"}
	if lossless {
		args = append(args, "-lossless", "-z", "9")
	} else {
		args = append(args, "-q", strconv.Itoa(int(quality)))
	}
	args = append(args, png, "-o", destination)
	if stderr, err := run("cwebp", args...); err != nil {
		if stderr == nil {
			return fmt.Errorf("cwebp is required to export WebP textures: %s", blitz.IOMessage(err))
		}
		return fmt.Errorf("cwebp failed to encode %s: %s", source, strings.TrimSpace(lossy.String(stderr)))
	}
	return nil
}

// run executes a command. It returns nil stderr when the command could not
// be started, and the captured stderr when it exited unsuccessfully.
func run(name string, args ...string) ([]byte, error) {
	command := exec.Command(name, args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return nil, err
	}
	if err := command.Wait(); err != nil {
		return stderr.Bytes(), err
	}
	return nil, nil
}

// fallbackIDAT holds the zlib streams the Rust exporter produced with
// miniz_oxide at level 9, so fallback PNGs stay byte-identical.
var fallbackTextures = map[string][]byte{
	"baseColor":         {120, 218, 99, 152, 62, 167, 247, 63, 0, 6, 78, 2, 192},
	"normal":            {120, 218, 99, 104, 0, 2, 0, 5, 5, 2, 1},
	"roughnessMetallic": {120, 218, 99, 96, 96, 96, 56, 3, 0, 0, 209, 0, 205},
	"misc":              {120, 218, 99, 96, 96, 96, 248, 15, 0, 1, 4, 1, 0},
}

func writeFallbackPNG(path, role string) error {
	idat, ok := fallbackTextures[role]
	if !ok {
		return fmt.Errorf("unknown fallback texture role %s", role)
	}
	png := []byte("\x89PNG\r\n\x1a\n")
	header := binary.BigEndian.AppendUint32(nil, 1)
	header = binary.BigEndian.AppendUint32(header, 1)
	header = append(header, 8, 6, 0, 0, 0)
	png = pngChunk(png, "IHDR", header)
	png = pngChunk(png, "IDAT", idat)
	png = pngChunk(png, "IEND", nil)
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return fmt.Errorf("could not create %s: %s", filepath.Dir(path), blitz.IOMessage(err))
	}
	if err := os.WriteFile(path, png, 0o666); err != nil {
		return fmt.Errorf("could not write fallback texture %s: %s", path, blitz.IOMessage(err))
	}
	return nil
}

func pngChunk(output []byte, kind string, data []byte) []byte {
	output = binary.BigEndian.AppendUint32(output, uint32(len(data)))
	output = append(output, kind...)
	output = append(output, data...)
	crcInput := append([]byte(kind), data...)
	return binary.BigEndian.AppendUint32(output, resources.CRC32(crcInput))
}

func removeAsset(path string) {
	os.Remove(path)
	os.Remove(path + ".br")
}

func pruneOldBundles(outputRoot string, entries []map[string]any) error {
	included := map[string]bool{}
	for _, entry := range entries {
		if id, ok := entry["id"].(string); ok {
			included[id] = true
		}
	}
	for _, nation := range client.Nations() {
		nationDir := filepath.Join(outputRoot, nation.ID)
		children, err := readDirUnsorted(nationDir)
		if err != nil {
			continue
		}
		for _, child := range children {
			if !child.IsDir() || included[nation.ID+"/"+child.Name()] {
				continue
			}
			path := filepath.Join(nationDir, child.Name())
			if err := os.RemoveAll(path); err != nil {
				return blitz.IOError("could not prune stale tank bundle", path, err)
			}
		}
	}
	return nil
}

func canonicalOrAbsolute(path string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		canonical, err := pathx.Canonicalize(path)
		if err != nil {
			return "", blitz.IOError("could not resolve viewer directory", path, err)
		}
		return canonical, nil
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	directory, err := os.Getwd()
	if err != nil {
		return "", blitz.NewError("IO_ERROR", err.Error())
	}
	return filepath.Join(directory, path), nil
}

func textureSizeLabel(size *TextureSize) string {
	if size == nil {
		return "original"
	}
	return fmt.Sprintf("%dx%d", size.Width, size.Height)
}

func roundMillis(value float64) jsonx.F64 {
	return jsonx.F64(math.Round(value*1000) / 1000)
}

func isoTimestamp(moment time.Time) string {
	return moment.UTC().Truncate(time.Second).Format("2006-01-02T15:04:05") + "+00:00"
}
