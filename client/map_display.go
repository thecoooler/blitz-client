package client

import (
	"math"
	"strings"
)

// LoadMapDisplayCatalog loads Russian localized map names keyed by their map
// IDs. Entries without a translation or with ambiguous IDs are omitted.
func LoadMapDisplayCatalog(gameDir string) (map[uint32]string, error) {
	resolved, err := ResolveGameDir(gameDir)
	if err != nil {
		return nil, err
	}
	localized, err := readLocalizedNames(resolved.ResourceDir)
	if err != nil {
		return nil, err
	}
	// Preserve the localization key through map parsing so a raw localName
	// cannot be mistaken for a translated display name.
	keys := make(map[string]string, len(localized))
	for key := range localized {
		if strings.HasPrefix(key, "#maps:") {
			keys[key] = "\x00" + key
		}
	}
	maps, err := readMaps(resolved.ResourceDir, keys)
	if err != nil {
		return nil, err
	}
	names := make(map[uint32]string)
	seen := make(map[uint32]bool)
	ambiguous := make(map[uint32]bool)
	for _, m := range maps {
		if m.ID <= 0 || m.ID > math.MaxUint32 {
			continue
		}
		id := uint32(m.ID)
		if seen[id] {
			ambiguous[id] = true
		}
		seen[id] = true
		if key, ok := strings.CutPrefix(m.LocalName, "\x00"); ok && localized[key] != "" {
			names[id] = localized[key]
		}
	}
	for id := range ambiguous {
		delete(names, id)
	}
	return names, nil
}
