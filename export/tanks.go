package export

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/lossy"
	"github.com/thecoooler/blitz-client/resources"
)

// TextureSize is a maximum texture width and height.
type TextureSize struct {
	Width, Height uint32
}

// TankExportOptions controls model encoding within a client snapshot.
type TankExportOptions struct {
	// MaxTextureSize nil preserves source dimensions.
	MaxTextureSize *TextureSize
	TextureFormat  string
	WebPQuality    uint8
	IncludePBR     bool
	Compress       bool
}

func NewTankExportOptions() TankExportOptions {
	return TankExportOptions{MaxTextureSize: &TextureSize{1024, 1024}, TextureFormat: "webp", WebPQuality: 90, IncludePBR: true, Compress: true}
}

// fatalModelError reports failures that would repeat for every tank (missing
// tools, unwritable output); they abort the export instead of becoming issues.
func fatalModelError(err error) bool {
	switch blitz.AsError(err).Code {
	case "BROTLI_NOT_FOUND", "FFMPEG_NOT_FOUND", "CWEBP_NOT_FOUND", "IO_ERROR":
		return true
	}
	return false
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
	return nil
}

// assetResolver maps a logical client path (slash-separated, with or without
// .dvpl) to the stored file, honoring the cache overlay. It returns "" if absent.
type assetResolver func(rel string) string

func filesResolver(files map[string]sourceFile) assetResolver {
	return func(rel string) string {
		rel = strings.TrimSuffix(path.Clean(strings.ReplaceAll(rel, `\`, "/")), ".dvpl")
		if !safeDataPath(rel) {
			return ""
		}
		return files[filepath.FromSlash(rel)].path
	}
}

// exportOne converts one tank. tank.ModelFiles hold logical client paths.
// convertedTextures carries the first produced file per source texture across
// tanks; repeated sources are linked instead of converted again.
func exportOne(tank *client.ClientTank, resolve assetResolver, outputRoot string, options *TankExportOptions, convertedTextures map[string]string) (*SnapshotModel, error) {
	bundle := filepath.Join(outputRoot, tank.Nation, tank.ID)
	if err := os.MkdirAll(bundle, 0o777); err != nil {
		return nil, blitz.IOError("could not create model bundle", bundle, err)
	}
	sc2Data, err := readResolved(resolve, tank.ModelFiles.Scene)
	if err != nil {
		return nil, err
	}
	scgData, err := readResolved(resolve, tank.ModelFiles.Geometry)
	if err != nil {
		return nil, err
	}
	model, err := exportModel(sc2Data, scgData, tank.ModelFiles.Scene, tank.ModelFiles.Geometry, tank.ID,
		options.IncludePBR, options.TextureFormat, options.WebPQuality, options.Compress)
	if err != nil {
		return nil, err
	}

	modelJSON, err := jsonx.Marshal(model.metadata)
	if err != nil {
		return nil, err
	}
	_, err = writeCompressedAsset(filepath.Join(bundle, "model.json"), modelJSON, options.Compress)
	if err != nil {
		return nil, err
	}
	_, err = writeCompressedAsset(filepath.Join(bundle, "model.bin"), model.binary, options.Compress)
	if err != nil {
		return nil, err
	}
	if model.pbrBinary != nil {
		if _, err = writeCompressedAsset(filepath.Join(bundle, "model.pbr.bin"), model.pbrBinary, options.Compress); err != nil {
			return nil, err
		}
	} else {
		removeAsset(filepath.Join(bundle, "model.pbr.bin"))
	}

	armorPath := filepath.Join(bundle, "armor.json")
	armorMeshPath := filepath.Join(bundle, "armor-mesh.json")
	removeAsset(armorPath)
	removeAsset(armorMeshPath)
	collisionSC2 := fmt.Sprintf("3d/Tanks/CollisionMeshes/%s-%s.sc2", tank.Nation, tank.ID)
	collisionSCG := fmt.Sprintf("3d/Tanks/CollisionMeshes/%s-%s.scg", tank.Nation, tank.ID)
	vehicleXML := fmt.Sprintf("XML/item_defs/vehicles/%s/%s.xml", tank.Nation, tank.ID)
	hasArmor := false
	var armorNote *string
	if resolve(collisionSC2) != "" && resolve(collisionSCG) != "" && resolve(vehicleXML) != "" {
		_, _, err := exportArmorFiles(resolve, collisionSC2, collisionSCG, vehicleXML, tank, armorPath, armorMeshPath, options.Compress)
		if err != nil {
			if fatalModelError(err) {
				return nil, err
			}
			note := err.Error()
			armorNote = &note
		} else {
			hasArmor = true
		}
	}

	modelDirectory := path.Dir(tank.ModelFiles.Scene)
	seenTextures := map[string]bool{}
	for _, texture := range model.textures {
		if seenTextures[texture.file] {
			continue
		}
		seenTextures[texture.file] = true
		target := filepath.Join(bundle, texture.file)
		if texture.source == "" {
			if err := writeFallbackPNG(target, texture.role); err != nil {
				return nil, err
			}
			continue
		}
		// Many vehicles share track, wheel and turret textures; the cache maps
		// each source+role to its first converted file for the whole run.
		cacheKey := texture.source + "\x00" + texture.role
		if first, ok := convertedTextures[cacheKey]; ok {
			if err := linkTexture(first, target); err != nil {
				return nil, err
			}
			continue
		}
		// Material paths are relative to the scene; the resolver also finds
		// unpacked cache files that override packed game resources.
		source := resolve(path.Join(modelDirectory, filepath.ToSlash(texture.source)))
		if source == "" {
			// Some client maps ship only as PVRTC variants, which ffmpeg cannot
			// decode; emit the placeholder color in the requested format so the
			// file still matches the extension the model declares.
			temp, err := os.MkdirTemp("", "blitz-tank-texture-")
			if err != nil {
				return nil, blitz.IOError("could not create temporary directory", os.TempDir(), err)
			}
			placeholder := filepath.Join(temp, "placeholder.png")
			err = writeFallbackPNG(placeholder, texture.role)
			if err == nil {
				err = convertTexture(placeholder, target, options.MaxTextureSize, options.TextureFormat, options.WebPQuality, texture.lossless)
			}
			os.RemoveAll(temp)
			if err != nil {
				return nil, err
			}
		} else {
			if err := convertTexture(source, target, options.MaxTextureSize, options.TextureFormat, options.WebPQuality, texture.lossless); err != nil {
				return nil, err
			}
		}
		convertedTextures[cacheKey] = target
	}

	bundleURL := fmt.Sprintf("./assets/tanks/%s/%s/", tank.Nation, tank.ID)
	suffix := ""
	if options.Compress {
		suffix = ".br"
	}
	result := &SnapshotModel{Bundle: bundleURL, Model: bundleURL + "model.json" + suffix, ArmorNote: armorNote}
	if hasArmor {
		armorURL := bundleURL + "armor.json" + suffix
		armorMeshURL := bundleURL + "armor-mesh.json" + suffix
		result.Armor = &armorURL
		result.ArmorMesh = &armorMeshURL
	}
	return result, nil
}

func exportArmorFiles(resolve assetResolver, sc2Path, scgPath, xmlPath string, tank *client.ClientTank, armorPath, meshPath string, compress bool) (assetStats, assetStats, error) {
	var inputs [3][]byte
	for index, rel := range []string{sc2Path, scgPath, xmlPath} {
		data, err := readResolved(resolve, rel)
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

func readResolved(resolve assetResolver, rel string) ([]byte, error) {
	source := resolve(rel)
	if source == "" {
		return nil, blitz.ErrorAt("CLIENT_DATA_MISSING", "required model resource is missing", rel)
	}
	return readDecoded(source)
}

func readDecoded(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, blitz.IOError("could not read resource", path, err)
	}
	plain, _, err := resources.DecodeDVPL(data)
	if err != nil {
		return nil, fmt.Errorf("could not decode %s: %v", path, err)
	}
	return plain, nil
}

// webpEncoderName probes ffmpeg once for a built-in WebP encoder. When ffmpeg
// can encode WebP directly, textures skip the intermediate PNG and cwebp.
// Empty means ffmpeg has no WebP encoder and cwebp is used instead.
var webpEncoderName = sync.OnceValue(func() string {
	data, err := exec.Command("ffmpeg", "-hide_banner", "-encoders").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		for _, field := range strings.Fields(line) {
			if field == "libwebp" || field == "libwebp_anim" {
				return field
			}
		}
	}
	return ""
})

func convertTexture(source, destination string, maxSize *TextureSize, format string, quality uint8, lossless bool) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o777); err != nil {
		return blitz.IOError("could not create texture directory", filepath.Dir(destination), err)
	}
	sourceBytes, err := readDecoded(source)
	if err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "blitz-tank-texture-")
	if err != nil {
		return blitz.IOError("could not create temporary directory", os.TempDir(), err)
	}
	defer os.RemoveAll(temp)
	dds := filepath.Join(temp, "source.dds")
	if err := os.WriteFile(dds, sourceBytes, 0o666); err != nil {
		return blitz.IOError("could not stage DDS", dds, err)
	}
	scale := []string{}
	if maxSize != nil {
		scale = []string{"-vf", fmt.Sprintf(
			"scale=w='min(iw,%d)':h='min(ih,%d)':force_original_aspect_ratio=decrease", maxSize.Width, maxSize.Height)}
	}
	if format == "webp" {
		if encoder := webpEncoderName(); encoder != "" {
			args := []string{"-hide_banner", "-v", "error", "-y", "-i", dds}
			args = append(args, scale...)
			args = append(args, "-frames:v", "1", "-update", "1", "-c:v", encoder)
			if lossless {
				args = append(args, "-lossless", "1", "-compression_level", "6")
			} else {
				args = append(args, "-quality", strconv.Itoa(int(quality)), "-compression_level", "4")
			}
			args = append(args, destination)
			if stderr, err := run("ffmpeg", args...); err != nil {
				if stderr == nil {
					return blitz.NewError("FFMPEG_NOT_FOUND", "ffmpeg is required to decode DDS textures: "+blitz.IOMessage(err))
				}
				return fmt.Errorf("ffmpeg failed to encode %s: %s", source, strings.TrimSpace(lossy.String(stderr)))
			}
			return nil
		}
	}

	png := filepath.Join(temp, "decoded.png")
	args := []string{"-hide_banner", "-v", "error", "-y", "-i", dds}
	args = append(args, scale...)
	// The PNG is a lossless intermediate, so its compression level only costs
	// time; the final texture is identical regardless of the level used.
	args = append(args, "-frames:v", "1", "-update", "1", "-compression_level", "1", png)
	if stderr, err := run("ffmpeg", args...); err != nil {
		if stderr == nil {
			return blitz.NewError("FFMPEG_NOT_FOUND", "ffmpeg is required to decode DDS textures: "+blitz.IOMessage(err))
		}
		return fmt.Errorf("ffmpeg failed to decode %s: %s", source, strings.TrimSpace(lossy.String(stderr)))
	}
	if format == "png" {
		data, err := os.ReadFile(png)
		if err == nil {
			err = os.WriteFile(destination, data, 0o666)
		}
		if err != nil {
			return blitz.IOError("could not write texture", destination, err)
		}
		return nil
	}
	args = []string{"-m", "4", "-alpha_q", "100", "-exact", "-metadata", "none"}
	if lossless {
		args = append(args, "-lossless", "-z", "9")
	} else {
		args = append(args, "-q", strconv.Itoa(int(quality)))
	}
	args = append(args, png, "-o", destination)
	if stderr, err := run("cwebp", args...); err != nil {
		if stderr == nil {
			return blitz.NewError("CWEBP_NOT_FOUND", "cwebp is required to export WebP textures: "+blitz.IOMessage(err))
		}
		return fmt.Errorf("cwebp failed to encode %s: %s", source, strings.TrimSpace(lossy.String(stderr)))
	}
	return nil
}

// linkTexture places the previously converted texture at target. A hard link
// shares the first file's bytes; a plain copy covers filesystems that reject links.
func linkTexture(from, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o777); err != nil {
		return blitz.IOError("could not create texture directory", filepath.Dir(to), err)
	}
	if err := os.Link(from, to); err == nil {
		return nil
	}
	data, err := os.ReadFile(from)
	if err != nil {
		return blitz.IOError("could not read converted texture", from, err)
	}
	if err := os.WriteFile(to, data, 0o666); err != nil {
		return blitz.IOError("could not write texture", to, err)
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
		return blitz.IOError("could not create texture directory", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, png, 0o666); err != nil {
		return blitz.IOError("could not write fallback texture", path, err)
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
