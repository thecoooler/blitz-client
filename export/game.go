package export

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// SnapshotGame identifies the client build a snapshot was read from. Every
// source resource is optional: a missing one leaves its fields null or empty,
// a malformed one fails the export like other configuration resources.
type SnapshotGame struct {
	// VersionText is the first line of version.txt, for example
	// "26.10.70_533068 master MacOS". Version, Build, Branch and Platform are
	// its fields; they are null when the line has no such part.
	VersionText *string `json:"versionText"`
	Version     *string `json:"version"`
	Build       *string `json:"build"`
	Branch      *string `json:"branch"`
	Platform    *string `json:"platform"`
	// Realms are the distinct realm IDs of regions_publish.yaml, in source order.
	Realms []string `json:"realms"`
	// Languages come from languages.yaml, in source order.
	Languages []string `json:"languages"`
	// HangarPresets are the directories of 3d/Configs/Presets, sorted. Seasonal
	// presets (for example hg_halloween_26) name the patch's hangar events.
	HangarPresets []string `json:"hangarPresets"`
	// CacheFiles counts the resources selected for this export that come from
	// the downloaded packs overlay instead of the installed Data directory, so
	// it depends on included. Zero means the export reflects the installed
	// client only.
	CacheFiles int `json:"cacheFiles"`
}

func extractGame(r snapshotReader) (*SnapshotGame, error) {
	g := &SnapshotGame{Realms: []string{}, Languages: []string{}, HangarPresets: []string{}}
	if r.has("version.txt") {
		b, err := r.read("version.txt")
		if err != nil {
			return nil, err
		}
		parseVersionText(g, string(b))
	}
	if r.has("regions_publish.yaml") {
		b, err := r.read("regions_publish.yaml")
		if err != nil {
			return nil, err
		}
		if g.Realms, err = publishRealms(b); err != nil {
			return nil, fmt.Errorf("regions_publish.yaml: %w", err)
		}
	}
	if r.has("languages.yaml") {
		b, err := r.read("languages.yaml")
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(b, &g.Languages); err != nil {
			return nil, fmt.Errorf("languages.yaml: %w", err)
		}
		if g.Languages == nil {
			g.Languages = []string{}
		}
	}
	for rel, f := range r.files {
		rel = filepath.ToSlash(rel)
		if f.fromCache {
			g.CacheFiles++
		}
		if name, ok := strings.CutPrefix(rel, "3d/Configs/Presets/"); ok && path.Base(name) == "presets.yaml" && path.Dir(name) != "." && !strings.Contains(path.Dir(name), "/") {
			g.HangarPresets = append(g.HangarPresets, path.Dir(name))
		}
	}
	slices.Sort(g.HangarPresets)
	g.HangarPresets = slices.Compact(g.HangarPresets)
	return g, nil
}

// parseVersionText splits "<version>_<build> <branch> <platform>". A first
// token without a numeric "_" suffix is the version alone; the platform is the
// rest of the line, which may contain spaces.
func parseVersionText(g *SnapshotGame, text string) {
	line, _, _ := strings.Cut(strings.TrimPrefix(text, "\ufeff"), "\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	g.VersionText = &line
	fields := strings.Fields(line)
	version := fields[0]
	if i := strings.LastIndexByte(version, '_'); i > 0 && i < len(version)-1 && strings.Trim(version[i+1:], "0123456789") == "" {
		build := version[i+1:]
		version = version[:i]
		g.Build = &build
	}
	g.Version = &version
	if len(fields) > 1 {
		g.Branch = &fields[1]
	}
	if len(fields) > 2 {
		platform := strings.Join(fields[2:], " ")
		g.Platform = &platform
	}
}

func publishRealms(b []byte) ([]string, error) {
	var doc struct {
		Regions yaml.Node `yaml:"Regions"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	realms := []string{}
	if doc.Regions.Kind == 0 {
		return realms, nil
	}
	if doc.Regions.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("Regions must be a mapping")
	}
	// Mapping nodes keep source order; values alternate with keys.
	for i := 1; i < len(doc.Regions.Content); i += 2 {
		var region struct {
			Realm string `yaml:"realm"`
		}
		if err := doc.Regions.Content[i].Decode(&region); err != nil {
			return nil, fmt.Errorf("region %s: %w", doc.Regions.Content[i-1].Value, err)
		}
		if region.Realm != "" && !slices.Contains(realms, region.Realm) {
			realms = append(realms, region.Realm)
		}
	}
	return realms, nil
}
