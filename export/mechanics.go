package export

import (
	"image"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/resources/sprite"
	"go.yaml.in/yaml/v3"
)

const mechanicMediaRoot = "Gfx/Lobby/icons/tank-mechanics"

// mechanicMediaDirs maps a tank parameter mechanic name to the client artwork
// folder. The game keeps this mapping in its binary, not in Data, so it is
// mirrored here. Two parameter names are aliases of the enum entry that owns
// the artwork:
//
//	LVArmor              -> armor-moving (ArmorMoving)
//	AlteredSpotDuration  -> reduced-spot-duration (ReducedSpotDuration)
var mechanicMediaDirs = map[string]string{
	"ATGM":                       "atgm",
	"ATMammo":                    "atm-ammo",
	"AlteredSpotDuration":        "reduced-spot-duration",
	"ChRocketBoosters":           "rocket-boosters",
	"DesertPower":                "desert-power",
	"DoubleShot":                 "double-shot",
	"DraculaTrack":               "dracula-track",
	"DrumShot":                   "drum-reload",
	"DynamicArmor":               "dynamic-protection",
	"GunCooling":                 "gun-cooling",
	"GunSwitcher":                "gun-mode-switch",
	"ImprovedDetection":          "improved-detection",
	"InternalTrack":              "internal-track",
	"LTCamo":                     "lt-camo",
	"LVArmor":                    "armor-moving",
	"MovableArmor":               "movable-armor",
	"NotDamageModuls":            "not-damage-moduls",
	"PumpShot":                   "pump-action",
	"ReserveAmmo":                "reserve-ammo",
	"ShotDispersionStabilizator": "shot-dispersion-stabilizator",
	"TTHOnMaps":                  "tth-on-maps",
	"TripleShot":                 "triple-shot",
	"TwinGuns":                   "twin-guns",
	"WithoutCrew":                "without-crew",
}

// mechanicLocalizationAliases connect parameter names whose enum entry uses a
// different localization key. PumpShot keeps its own description key but the
// title of ArmorPumpShot.
var mechanicLocalizationAliases = map[string]mechanicLocalization{
	"AlteredSpotDuration": {
		nameKey:        "TankMechanics/ReducedSpotDuration",
		descriptionKey: "TankMechanics/ReducedSpotDurationDescription",
	},
	"LVArmor": {
		nameKey:        "TankMechanics/ArmorMoving",
		descriptionKey: "TankMechanics/ArmorMovingDescription",
	},
	"PumpShot": {
		nameKey:        "TankMechanics/ArmorPumpShot",
		descriptionKey: "TankMechanics/PumpShotDescription",
	},
}

type mechanicLocalization struct {
	nameKey        string
	descriptionKey string
}

// MechanicExport is one entry of mechanics.json. It defines the mechanic; the
// tank-to-mechanic relation remains solely in each tank's parameters.mechanics.
// Name and description are the ru localization values behind the keys; both are
// empty when the client has no such key.
type MechanicExport struct {
	ID             string      `json:"id"`
	NameKey        string      `json:"nameKey"`
	Name           string      `json:"name"`
	DescriptionKey string      `json:"descriptionKey"`
	Description    string      `json:"description"`
	Image          *string     `json:"image"`
	Icon           *string     `json:"icon"`
	ImageSource    *IconSource `json:"imageSource,omitempty"`
	IconSource     *IconSource `json:"iconSource,omitempty"`
}

// MechanicExportReport summarizes the mechanics part of a client export.
type MechanicExportReport struct {
	File      string `json:"file"`
	Mechanics int    `json:"mechanics"`
	Images    int    `json:"images"`
	Icons     int    `json:"icons"`
}

// exportMechanics writes mechanics.json and the mechanics artwork under
// assets/mechanics. It defines the mechanics referenced by tank parameters,
// with localization and the client's main artwork and large icon. The tank
// membership stays in tanks[].parameters.mechanics. Missing optional media is
// null, not an error.
func exportMechanics(files map[string]sourceFile, resourceDir, outputDir string, snapshot *Snapshot) (*MechanicExportReport, error) {
	localized, err := readRuLocalization(files)
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	for _, tank := range snapshot.Tanks {
		names, err := readTankMechanics(files, tank)
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			used[name] = true
		}
	}

	report := &MechanicExportReport{}
	entries := make([]MechanicExport, 0, len(used))
	atlases := map[string]image.Image{}
	for _, id := range slices.Sorted(maps.Keys(used)) {
		localization := mechanicLocalizationFor(id)
		entry := MechanicExport{
			ID:             id,
			NameKey:        localization.nameKey,
			Name:           localizedValue(localized, localization.nameKey),
			DescriptionKey: localization.descriptionKey,
			Description:    localizedValue(localized, localization.descriptionKey),
		}
		directory, ok := mechanicMediaDirs[id]
		if !ok {
			directory = kebabCase(id)
		}
		if directory != "" {
			images, err := artworkImages(files, mechanicMediaRoot+"/"+directory)
			if err != nil {
				return nil, err
			}
			if source := findMechanicImage(images, "bg_", nil); source != nil {
				if link, origin, err := writeArtworkImage(resourceDir, outputDir, "assets/mechanics/"+directory, "image", source, atlases); err != nil {
					snapshot.Issues = append(snapshot.Issues, ExportIssue{Section: "mechanics", Message: err.Error()})
				} else {
					entry.Image, entry.ImageSource = link, origin
					report.Images++
				}
			}
			if source := findMechanicImage(images, "icon_", []string{"_l", "_m", "_s", ""}); source != nil {
				if link, origin, err := writeArtworkImage(resourceDir, outputDir, "assets/mechanics/"+directory, "icon", source, atlases); err != nil {
					snapshot.Issues = append(snapshot.Issues, ExportIssue{Section: "mechanics", Message: err.Error()})
				} else {
					entry.Icon, entry.IconSource = link, origin
					report.Icons++
				}
			}
		}
		entries = append(entries, entry)
	}
	file := filepath.Join(outputDir, "mechanics.json")
	if err := writeJSON(file, entries, false); err != nil {
		return nil, err
	}
	report.File = file
	report.Mechanics = len(entries)
	return report, nil
}

// mechanicLocalizationFor returns the source keys used by the client for one
// parameter mechanic name.
func mechanicLocalizationFor(id string) mechanicLocalization {
	if alias, ok := mechanicLocalizationAliases[id]; ok {
		return alias
	}
	return mechanicLocalization{
		nameKey:        "TankMechanics/" + id,
		descriptionKey: "TankMechanics/" + id + "Description",
	}
}

func readTankMechanics(files map[string]sourceFile, tank SnapshotTank) ([]string, error) {
	relative := "3d/Tanks/Parameters/" + tank.Nation + "/" + tank.TechName + ".yaml"
	source, ok := files[filepath.FromSlash(relative)]
	if !ok {
		return nil, nil
	}
	data, err := client.ReadClientAsset(source.path)
	if err != nil {
		return nil, err
	}
	var document struct {
		Mechanics []struct {
			Name string `yaml:"name"`
		} `yaml:"mechanics"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, blitz.ErrorAt("CLIENT_YAML_INVALID", "tank mechanics: "+err.Error(), source.path)
	}
	names := make([]string, 0, len(document.Mechanics))
	for _, mechanic := range document.Mechanics {
		if mechanic.Name != "" {
			names = append(names, mechanic.Name)
		}
	}
	return names, nil
}

func readRuLocalization(files map[string]sourceFile) (map[string]*string, error) {
	source, ok := files[filepath.FromSlash("Strings/ru.yaml")]
	if !ok {
		return map[string]*string{}, nil
	}
	data, err := client.ReadClientAsset(source.path)
	if err != nil {
		return nil, err
	}
	var localized map[string]*string
	if err := yaml.Unmarshal(data, &localized); err != nil {
		return nil, blitz.ErrorAt("CLIENT_YAML_INVALID", "Strings/ru.yaml: "+err.Error(), source.path)
	}
	return localized, nil
}

func localizedValue(localized map[string]*string, key string) string {
	if value := localized[key]; value != nil {
		return *value
	}
	return ""
}

// artworkImages discovers the images of one client artwork folder, preferring
// the @2x variant and a packed image with its embedded sprite descriptor.
func artworkImages(files map[string]sourceFile, directory string) (map[string]*iconSource, error) {
	prefix := directory + "/"
	images := map[string]*iconSource{}
	for relative, sourceFile := range files {
		slash := filepath.ToSlash(relative)
		if !strings.HasPrefix(slash, prefix) {
			continue
		}
		rest := strings.TrimPrefix(slash, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		stem, kind, ok := iconFileStem(rest)
		if !ok {
			continue
		}
		source := &iconSource{id: strings.TrimSuffix(stem, "@2x"), relative: slash, file: sourceFile.path, scale: 1}
		if strings.HasSuffix(stem, "@2x") {
			source.scale = 2
		}
		data, err := client.ReadClientAsset(sourceFile.path)
		if err != nil {
			return nil, err
		}
		if kind == "sprite" {
			source.sprite, err = sprite.Parse(data)
			if err != nil {
				return nil, atPath(err, sourceFile.path)
			}
		} else if descriptor, ok, err := sprite.Embedded(data); err != nil {
			return nil, atPath(err, sourceFile.path)
		} else if ok {
			source.sprite = descriptor
		}
		if previous, ok := images[source.id]; !ok || artworkImageBetter(source, previous) {
			images[source.id] = source
		}
	}
	return images, nil
}

func artworkImageBetter(candidate, current *iconSource) bool {
	if candidate.scale != current.scale {
		return candidate.scale > current.scale
	}
	if (candidate.sprite != nil) != (current.sprite != nil) {
		return candidate.sprite != nil
	}
	return candidate.relative < current.relative
}

// findMechanicImage returns the first image with the given prefix and one of
// the suffixes in priority order. An empty suffix matches any remaining name.
func findMechanicImage(images map[string]*iconSource, prefix string, suffixes []string) *iconSource {
	if len(suffixes) == 0 {
		suffixes = []string{""}
	}
	ids := slices.Sorted(maps.Keys(images))
	for _, suffix := range suffixes {
		for _, id := range ids {
			if strings.HasPrefix(id, prefix) && (suffix == "" || strings.HasSuffix(id, suffix)) {
				return images[id]
			}
		}
	}
	return nil
}

func writeArtworkImage(resourceDir, outputDir, folder, name string, source *iconSource, atlases map[string]image.Image) (*string, *IconSource, error) {
	icon, encoded, err := renderIcon(resourceDir, *source, 0, atlases)
	if err != nil {
		return nil, nil, err
	}
	file := folder + "/" + name + ".png"
	absolute, err := iconPath(outputDir, file)
	if err != nil {
		return nil, nil, err
	}
	if _, err := writeIfChanged(absolute, encoded); err != nil {
		return nil, nil, err
	}
	return &file, &icon.Source, nil
}

// kebabCase is the fallback folder spelling for a mechanic added after this
// mapping was written. Known client exceptions are in mechanicMediaDirs.
func kebabCase(value string) string {
	runes := []rune(value)
	var out strings.Builder
	for index, r := range runes {
		if unicode.IsUpper(r) {
			if index > 0 {
				previous := runes[index-1]
				nextLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
				if unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && nextLower) {
					out.WriteByte('-')
				}
			}
			out.WriteRune(unicode.ToLower(r))
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
