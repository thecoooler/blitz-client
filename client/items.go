package client

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/lossy"
	"github.com/thecoooler/blitz-client/resources/xml"
)

// ItemKind is a category of vehicle items.
type ItemKind string

const (
	ItemShells      ItemKind = "shells"
	ItemConsumables ItemKind = "consumables"
	ItemProvisions  ItemKind = "provisions"
	ItemEquipment   ItemKind = "equipment"
)

// ItemKinds lists the item categories in export order.
var ItemKinds = []ItemKind{ItemShells, ItemConsumables, ItemProvisions, ItemEquipment}

// ClientItem is one consumable, provision, or equipment definition.
type ClientItem struct {
	ID int64 `json:"id"`
	// Name is the XML element name.
	Name string `json:"name"`
	// UserString and Description are localized; empty for hidden service
	// items without them.
	UserString  string `json:"userString"`
	Description string `json:"description"`
	// IconKey is the item's <icon>; the UI maps it to a sprite.
	IconKey string `json:"iconKey,omitempty"`
}

// ClientShellType is a shell type from the <icons> table of the shells
// lists. A shell's icon depends only on its type, which its <icon> names.
type ClientShellType struct {
	// ID is the table key and the shells' <icon>, such as ap_cr_premium.
	ID string `json:"id"`
	// Name is the table's image name without extension, such as
	// ARMOR_PIERCING_CR_PREMIUM.
	Name    string `json:"name"`
	Premium bool   `json:"premium"`
	// UserString is the localized <Name without _PREMIUM>_FULL string, such as
	// "Подкалиберный снаряд"; empty when the client has no translation.
	UserString string `json:"userString"`
}

// ClientItems groups item definitions by kind.
type ClientItems struct {
	Shells      []ClientShellType `json:"shells"`
	Consumables []ClientItem      `json:"consumables"`
	Provisions  []ClientItem      `json:"provisions"`
	Equipment   []ClientItem      `json:"equipment"`
}

// IconKeys returns the distinct icon keys of one kind in item order. A shell
// type's key is its ID.
func (items *ClientItems) IconKeys(kind ItemKind) []string {
	var keys []string
	add := func(key string) {
		if key != "" && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	if kind == ItemShells {
		for _, shell := range items.Shells {
			add(shell.ID)
		}
		return keys
	}
	var list []ClientItem
	switch kind {
	case ItemConsumables:
		list = items.Consumables
	case ItemProvisions:
		list = items.Provisions
	case ItemEquipment:
		list = items.Equipment
	}
	for _, item := range list {
		add(item.IconKey)
	}
	return keys
}

const commonItemDefs = "XML/item_defs/vehicles/common/"

// readItems reads regular items; event-specific item files are not included.
// A missing item list contributes no items.
func readItems(dataDir string, localized map[string]string) (*ClientItems, error) {
	items := &ClientItems{}
	var err error
	if items.Consumables, err = readItemList(dataDir, commonItemDefs+"consumables/common.xml", localized); err != nil {
		return nil, err
	}
	if items.Provisions, err = readItemList(dataDir, commonItemDefs+"provisions/common.xml", localized); err != nil {
		return nil, err
	}
	if items.Equipment, err = readItemList(dataDir, commonItemDefs+"optional_devices.xml", localized); err != nil {
		return nil, err
	}
	if items.Shells, err = readShellTypes(dataDir, localized); err != nil {
		return nil, err
	}
	return items, nil
}

// readItemList reads the elements with an <id> child; list metadata such as
// <nextAvailableId> is skipped.
func readItemList(dataDir, relative string, localized map[string]string) ([]ClientItem, error) {
	items := []ClientItem{}
	root, path, err := readOptionalXML(dataDir, relative)
	if root == nil {
		return items, err
	}
	for _, node := range root.Children {
		if node.Child("id") == nil {
			continue
		}
		id, err := requiredNumber(node, "id", path)
		if err != nil {
			return nil, err
		}
		item := ClientItem{ID: id, Name: node.Name}
		if value, ok := xml.ChildText(node, "userString"); ok {
			item.UserString = localize(value, localized)
		}
		if value, ok := xml.ChildText(node, "description"); ok {
			item.Description = localize(value, localized)
		}
		item.IconKey, _ = xml.ChildText(node, "icon")
		items = append(items, item)
	}
	return items, nil
}

// readShellTypes merges the <icons> tables of the available nations' shells
// lists in first-seen order. Nations list only the types their shells need,
// but a type must have the same image name everywhere.
func readShellTypes(dataDir string, localized map[string]string) ([]ClientShellType, error) {
	nations, err := readAvailableNations(dataDir)
	if err != nil {
		return nil, err
	}
	types := []ClientShellType{}
	for _, nation := range nations {
		root, path, err := readOptionalXML(dataDir, "XML/item_defs/vehicles/"+nation+"/components/shells.xml")
		if root == nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		table := root.Child("icons")
		if table == nil {
			continue
		}
		for _, entry := range table.Children {
			fields := strings.Fields(entry.Text)
			if len(fields) == 0 {
				return nil, blitz.ErrorAt("CLIENT_XML_INVALID", fmt.Sprintf("shell type %s has no image", entry.Name), path)
			}
			name := fields[0]
			if dot := strings.LastIndexByte(name, '.'); dot > 0 {
				name = name[:dot]
			}
			index := slices.IndexFunc(types, func(shell ClientShellType) bool { return shell.ID == entry.Name })
			if index >= 0 {
				if types[index].Name != name {
					return nil, blitz.ErrorAt("CLIENT_XML_INVALID", fmt.Sprintf(
						"shell type %s is %s here but %s in an earlier nation", entry.Name, name, types[index].Name), path)
				}
				continue
			}
			base, premium := strings.CutSuffix(name, "_PREMIUM")
			shell := ClientShellType{ID: entry.Name, Name: name, Premium: premium}
			if value, ok := localized[base+"_FULL"]; ok {
				shell.UserString = value
			}
			types = append(types, shell)
		}
	}
	return types, nil
}

// readOptionalXML parses a resource; the root is nil when it does not exist.
func readOptionalXML(dataDir, relative string) (*xml.Node, string, error) {
	path := ResolveClientAsset(dataDir, relative)
	if path == "" {
		return nil, "", nil
	}
	data, err := ReadClientAsset(path)
	if err != nil {
		return nil, path, err
	}
	root, err := xml.Parse(data)
	if err != nil {
		return nil, path, blitz.ErrorAt("CLIENT_XML_INVALID", err.Error(), path)
	}
	return root, path, nil
}

// iconStyle is a UI resource that assigns sprites to item icon keys. The
// first selector pattern of the first file that defines a key wins.
type iconStyle struct {
	path      string
	selectors []*regexp.Regexp
}

var (
	supplyIconStyles = []iconStyle{
		{"UI/Screens3/Lobby/Inventory/Equipment/EquipmentItemImage.style",
			selectorPatterns(`\.equipment_item\.KEY \* \.equipment-img`)},
		{"UI/Styles/Lobby/Inventory/InventoryBigStyles.style", selectorPatterns(`\.equipment_item\.KEY #icon`)},
		{"UI/Styles/Lobby/Inventory/InventoryNormalStyles.style",
			selectorPatterns(`\.equipment_item\.KEY #icon`, `\.equipment_item\.KEY #icon\.small`)},
	}
	equipmentIconStyles = []iconStyle{
		{"UI/Screens3/Lobby/Inventory/OptionalDevices/OptionalDevicesItemImage.style",
			selectorPatterns(`\.optional_device_item\.KEY #Img`)},
	}
	shellIconMap  = "UI/Screens3/Lobby/Inventory/Ammo/AmmoInfoOverlay.yaml"
	shellIconCase = regexp.MustCompile(`iconType\s*==\s*\\?"([^"\\]+)\\?"\s*->\s*\\?"~res:/([^"\\]+)`)
	styleBlock    = regexp.MustCompile(`([^{}]+)\{([^{}]*)\}`)
	styleSprite   = regexp.MustCompile(`bg-sprite\s*:\s*"~res:/([^"]+)"`)
)

func selectorPatterns(patterns ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(patterns))
	for index, pattern := range patterns {
		out[index] = regexp.MustCompile("^" + strings.ReplaceAll(pattern, "KEY", `([A-Za-z0-9_-]+)`) + "$")
	}
	return out
}

// ReadItemIconSprites maps each item kind's icon keys to the sprite the UI
// shows for them. Sprite paths are relative to Data, as written in the UI
// resource; their extension may be missing or name the source format. Only
// selectors without state classes are used, so state-only icons are absent.
// Missing UI resources contribute no keys.
func ReadItemIconSprites(dataDir string) (map[ItemKind]map[string]string, error) {
	supplies, err := readStyleSprites(dataDir, supplyIconStyles)
	if err != nil {
		return nil, err
	}
	equipment, err := readStyleSprites(dataDir, equipmentIconStyles)
	if err != nil {
		return nil, err
	}
	shells := map[string]string{}
	if path := ResolveClientAsset(dataDir, shellIconMap); path != "" {
		data, err := ReadClientAsset(path)
		if err != nil {
			return nil, err
		}
		for _, match := range shellIconCase.FindAllStringSubmatch(lossy.String(data), -1) {
			if _, ok := shells[match[1]]; !ok {
				shells[match[1]] = match[2]
			}
		}
	}
	return map[ItemKind]map[string]string{
		ItemShells:      shells,
		ItemConsumables: supplies,
		ItemProvisions:  supplies,
		ItemEquipment:   equipment,
	}, nil
}

func readStyleSprites(dataDir string, styles []iconStyle) (map[string]string, error) {
	sprites := map[string]string{}
	for _, style := range styles {
		path := ResolveClientAsset(dataDir, style.path)
		if path == "" {
			continue
		}
		data, err := ReadClientAsset(path)
		if err != nil {
			return nil, err
		}
		// Within a file, earlier selector patterns take precedence.
		found := make([]map[string]string, len(style.selectors))
		for rank := range found {
			found[rank] = map[string]string{}
		}
		for _, block := range styleBlock.FindAllStringSubmatch(lossy.String(data), -1) {
			sprite := styleSprite.FindStringSubmatch(block[2])
			if sprite == nil {
				continue
			}
			for _, selector := range strings.Split(block[1], ",") {
				selector = strings.Join(strings.Fields(selector), " ")
				for rank, pattern := range style.selectors {
					if match := pattern.FindStringSubmatch(selector); match != nil {
						if _, ok := found[rank][match[1]]; !ok {
							found[rank][match[1]] = sprite[1]
						}
					}
				}
			}
		}
		for _, keys := range found {
			for key, sprite := range keys {
				if _, ok := sprites[key]; !ok {
					sprites[key] = sprite
				}
			}
		}
	}
	return sprites, nil
}
