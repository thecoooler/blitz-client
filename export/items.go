package export

import (
	"slices"

	"github.com/thecoooler/blitz-client/client"
)

// ItemEntry is an item definition with its exported icon.
type ItemEntry struct {
	client.ClientItem
	// Icon is relative to the export directory; nil when the UI assigns no
	// sprite to the item's icon key or the sprite is missing.
	Icon *string `json:"icon"`
}

// ShellTypeEntry is a shell type with its exported icon.
type ShellTypeEntry struct {
	client.ClientShellType
	// Icon follows the same rules as ItemEntry.Icon.
	Icon *string `json:"icon"`
}

// ItemsFile is written to items.json.
type ItemsFile struct {
	Shells      []ShellTypeEntry `json:"shells"`
	Consumables []ItemEntry      `json:"consumables"`
	Provisions  []ItemEntry      `json:"provisions"`
	Equipment   []ItemEntry      `json:"equipment"`
}

// itemIconReferences returns the sprites of the items' icon keys per icon
// set; each item kind exports its icons into the set of the same name.
func itemIconReferences(items *client.ClientItems, sprites map[client.ItemKind]map[string]string) map[string][]string {
	references := map[string][]string{}
	for _, kind := range client.ItemKinds {
		for _, key := range items.IconKeys(kind) {
			if sprite, ok := sprites[kind][key]; ok && !slices.Contains(references[string(kind)], sprite) {
				references[string(kind)] = append(references[string(kind)], sprite)
			}
		}
	}
	return references
}

// linkItemIcons attaches exported icons to items and returns the sorted
// "kind/key" icon keys left without an icon.
func linkItemIcons(items *client.ClientItems, sprites map[client.ItemKind]map[string]string, index iconIndex) (*ItemsFile, []string) {
	missing := []string{}
	icon := func(kind client.ItemKind, key string) *string {
		if key == "" {
			return nil
		}
		if file, ok := index[string(kind)][sprites[kind][key]]; ok {
			file = "assets/icons/" + file
			return &file
		}
		if missed := string(kind) + "/" + key; !slices.Contains(missing, missed) {
			missing = append(missing, missed)
		}
		return nil
	}
	entries := func(kind client.ItemKind, list []client.ClientItem) []ItemEntry {
		out := make([]ItemEntry, 0, len(list))
		for _, item := range list {
			out = append(out, ItemEntry{ClientItem: item, Icon: icon(kind, item.IconKey)})
		}
		return out
	}
	file := &ItemsFile{Shells: make([]ShellTypeEntry, 0, len(items.Shells))}
	for _, shell := range items.Shells {
		file.Shells = append(file.Shells, ShellTypeEntry{ClientShellType: shell, Icon: icon(client.ItemShells, shell.ID)})
	}
	file.Consumables = entries(client.ItemConsumables, items.Consumables)
	file.Provisions = entries(client.ItemProvisions, items.Provisions)
	file.Equipment = entries(client.ItemEquipment, items.Equipment)
	slices.Sort(missing)
	return file, missing
}
