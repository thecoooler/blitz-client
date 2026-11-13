package export

import (
	"image"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/thecoooler/blitz-client/client"
)

const gameModeMediaRoot = "Gfx/Lobby/backgrounds/GameMode"

// GameModeExport is one entry of modes.json. It defines a battle mode with its
// localization and the client's logo, card banner and parallax banner layers.
// Name, Description and ShortDescription are the ru values behind the keys;
// both are empty when the client has no such key. Missing artwork is null.
// Position is the 1-based position of the mode in the client's selector and
// matches the order of the `modes` array in data.json.
type GameModeExport struct {
	ID                  string          `json:"id"`
	Position            int             `json:"position"`
	NameKey             string          `json:"nameKey"`
	Name                string          `json:"name"`
	DescriptionKey      *string         `json:"descriptionKey"`
	Description         string          `json:"description"`
	ShortDescriptionKey *string         `json:"shortDescriptionKey"`
	ShortDescription    string          `json:"shortDescription"`
	Logo                *string         `json:"logo"`
	Banner              *string         `json:"banner"`
	Layers              []GameModeLayer `json:"layers,omitempty"`
	LogoSource          *IconSource     `json:"logoSource,omitempty"`
	BannerSource        *IconSource     `json:"bannerSource,omitempty"`
}

// GameModeLayer is one parallax banner layer of a mode card. The client draws
// the layers in order behind the banner; consumers may compose or ignore them.
type GameModeLayer struct {
	Image  string     `json:"image"`
	Source IconSource `json:"source"`
}

// ModeExportReport summarizes the modes part of a client export.
type ModeExportReport struct {
	File    string `json:"file"`
	Modes   int    `json:"modes"`
	Logos   int    `json:"logos"`
	Banners int    `json:"banners"`
	Layers  int    `json:"layers"`
}

// exportGameModes writes modes.json and the mode artwork under assets/modes.
// The client identifies mode artwork by the lower-case mode ID, so the folder
// is not derived from the localized name. Missing optional artwork is null,
// not an error.
func exportGameModes(files map[string]sourceFile, resourceDir, outputDir string, snapshot *Snapshot) (*ModeExportReport, error) {
	localized, err := readRuLocalization(files)
	if err != nil {
		return nil, err
	}
	images, err := artworkImages(files, gameModeMediaRoot)
	if err != nil {
		return nil, err
	}
	report := &ModeExportReport{}
	entries := []GameModeExport{}
	atlases := map[string]image.Image{}
	for index, mode := range client.GameModes() {
		entry := GameModeExport{
			ID:       mode.ID,
			Position: index + 1,
			NameKey:  mode.Name,
			Name:     localizedValue(localized, mode.Name),
		}
		if mode.Description != nil {
			entry.DescriptionKey = mode.Description
			entry.Description = localizedValue(localized, *mode.Description)
		}
		if mode.ShortDescription != nil {
			entry.ShortDescriptionKey = mode.ShortDescription
			entry.ShortDescription = localizedValue(localized, *mode.ShortDescription)
		}
		art := strings.ToLower(mode.ID)
		folder := "assets/modes/" + art
		if source := images["icon_"+art]; source != nil {
			if link, origin, err := writeArtworkImage(resourceDir, outputDir, folder, "logo", source, atlases); err != nil {
				snapshot.Issues = append(snapshot.Issues, ExportIssue{Section: "modes", Message: err.Error()})
			} else {
				entry.Logo, entry.LogoSource = link, origin
				report.Logos++
			}
		}
		if source := images["bg_"+art+"_card"]; source != nil {
			if link, origin, err := writeArtworkImage(resourceDir, outputDir, folder, "banner", source, atlases); err != nil {
				snapshot.Issues = append(snapshot.Issues, ExportIssue{Section: "modes", Message: err.Error()})
			} else {
				entry.Banner, entry.BannerSource = link, origin
				report.Banners++
			}
		}
		for index := 1; index <= 3; index++ {
			source := images["bg_"+art+"_layer"+strconv.Itoa(index)]
			if source == nil {
				continue
			}
			name := "layer" + strconv.Itoa(index)
			link, origin, err := writeArtworkImage(resourceDir, outputDir, folder, name, source, atlases)
			if err != nil {
				snapshot.Issues = append(snapshot.Issues, ExportIssue{Section: "modes", Message: err.Error()})
				continue
			}
			entry.Layers = append(entry.Layers, GameModeLayer{Image: *link, Source: *origin})
			report.Layers++
		}
		entries = append(entries, entry)
	}
	file := filepath.Join(outputDir, "modes.json")
	if err := writeJSON(file, entries, false); err != nil {
		return nil, err
	}
	report.File = file
	report.Modes = len(entries)
	return report, nil
}
