package client

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/rs"
	"github.com/thecoooler/blitz-client/resources/xml"
)

// ClientCatalog is the data exported from maps.yaml,
// XML/item_defs/vehicles/*/list.xml, and the item definitions.
type ClientCatalog struct {
	Maps  []ClientMap
	Tanks []ClientTankMetadata
	Items *ClientItems
	// ModelTanks are vehicles with both showroom model resources, kept in
	// legacy model order.
	ModelTanks []ClientTank
}

// ClientMap is one entry of maps.yaml.
type ClientMap struct {
	// Name is the maps.yaml entry key, which is also the space name used by
	// client asset folders such as the minimap directory.
	Name                    string  `json:"name"`
	ID                      int64   `json:"id"`
	LocalName               string  `json:"localName"`
	AvailableInTrainingRoom bool    `json:"availableInTrainingRoom"`
	AvailableModes          []int64 `json:"availableModes"`
	SpriteFrame             int64   `json:"spriteFrame"`
	// VehicleCamouflageKind is the map's terrain kind from maps.yaml, such as
	// summer, winter or desert. It is empty when the entry omits the field,
	// which is common for alternate layouts of the same scene.
	VehicleCamouflageKind string `json:"vehicleCamouflageKind,omitempty"`
}

// ClientTankMetadata is the complete metadata of one vehicle-list entry.
type ClientTankMetadata struct {
	ID                      string            `json:"id"`
	UserString              string            `json:"userString"`
	ShortUserString         string            `json:"shortUserString"`
	Description             string            `json:"description"`
	Price                   ClientPrice       `json:"price"`
	SellPrice               *ClientPrice      `json:"sellPrice"`
	NotInShop               bool              `json:"notInShop"`
	EnrichmentPermanentCost int64             `json:"enrichmentPermanentCost"`
	Tags                    []string          `json:"tags"`
	Level                   int64             `json:"level"`
	Nation                  string            `json:"nation"`
	CombatRole              map[string]string `json:"combatRole"`
	ConfigurationModes      []string          `json:"configurationModes"`
	Leasing                 *string           `json:"leasing"`
}

// ClientPrice is a vehicle price in credits or gold.
type ClientPrice struct {
	Value  int64 `json:"value"`
	IsGold bool  `json:"is_gold"`
}

// DiscoverClientCatalog collects the complete data catalog and the subset of
// tanks with model files. XML lists and Russian localization are read once and
// shared by the metadata and model outputs.
func DiscoverClientCatalog(dataDir string) (*ClientCatalog, error) {
	discovered, err := discoverTanks(dataDir, true)
	if err != nil {
		return nil, err
	}
	maps, err := readMaps(dataDir, discovered.localized)
	if err != nil {
		return nil, err
	}
	items, err := readItems(dataDir, discovered.localized)
	if err != nil {
		return nil, err
	}
	return &ClientCatalog{Maps: maps, Tanks: discovered.metadata, Items: items, ModelTanks: discovered.models}, nil
}

func discoverModels(dataDir string) ([]ClientTank, error) {
	discovered, err := discoverTanks(dataDir, false)
	if err != nil {
		return nil, err
	}
	return discovered.models, nil
}

type discoveredTanks struct {
	metadata  []ClientTankMetadata
	models    []ClientTank
	localized map[string]string
}

// Both projections share resource loading; only full catalog discovery applies
// metadata invariants, including to entries without showroom model files.
func discoverTanks(dataDir string, fullCatalog bool) (*discoveredTanks, error) {
	if !isDir(filepath.Join(dataDir, "XML/item_defs/vehicles")) {
		return nil, blitz.ErrorAt("CLIENT_DATA_INVALID",
			"the selected directory does not contain XML/item_defs/vehicles", dataDir)
	}
	localized, err := readLocalizedNames(dataDir)
	if err != nil {
		return nil, err
	}
	source := directorySource{dir: dataDir}
	availableNations, err := ReadAvailableNations(source)
	if err != nil {
		return nil, err
	}
	metadata := []ClientTankMetadata{}
	modelsByNation := make(map[string][]ClientTank)

	for _, nation := range availableNations {
		list, err := ReadVehicleList(source, nation)
		if err != nil {
			return nil, err
		}
		if list == nil {
			continue
		}
		var info *Nation
		for index := range nations {
			if nations[index].ID == nation {
				info = &nations[index]
			}
		}
		for _, entry := range list.Entries {
			item := entry.Node
			if fullCatalog {
				tank, err := tankMetadata(item, nation, localized, list.Path)
				if err != nil {
					return nil, err
				}
				metadata = append(metadata, *tank)
			}
			if info != nil {
				if model := showroomModel(dataDir, item, nation, info, localized); model != nil {
					modelsByNation[nation] = append(modelsByNation[nation], *model)
				}
			}
		}
	}

	models := []ClientTank{}
	for _, nation := range nations {
		models = append(models, modelsByNation[nation.ID]...)
	}
	return &discoveredTanks{metadata: metadata, models: models, localized: localized}, nil
}

func tankMetadata(item *xml.Node, nation string, localized map[string]string, listPath string) (*ClientTankMetadata, error) {
	userString, err := requiredText(item, "userString", listPath)
	if err != nil {
		return nil, err
	}
	id, err := requiredText(item, "id", listPath)
	if err != nil {
		return nil, err
	}
	tank := &ClientTankMetadata{
		ID:         id,
		UserString: localize(userString, localized),
		Nation:     nation,
		CombatRole: map[string]string{},
	}
	if value, ok := xml.ChildText(item, "shortUserString"); ok {
		tank.ShortUserString = localize(value, localized)
	}
	if value, ok := xml.ChildText(item, "description"); ok {
		tank.Description = localize(value, localized)
	}
	priceNode, err := requiredNode(item, "price", listPath)
	if err != nil {
		return nil, err
	}
	if tank.Price, err = parsePrice(priceNode, listPath); err != nil {
		return nil, err
	}
	if node := item.Child("sellPrice"); node != nil {
		price, err := parsePrice(node, listPath)
		if err != nil {
			return nil, err
		}
		tank.SellPrice = &price
	}
	if value, ok := xml.ChildText(item, "notInShop"); ok {
		tank.NotInShop = strings.EqualFold(value, "true")
	}
	if tank.EnrichmentPermanentCost, err = requiredNumber(item, "enrichmentPermanentCost", listPath); err != nil {
		return nil, err
	}
	tags, err := requiredText(item, "tags", listPath)
	if err != nil {
		return nil, err
	}
	tank.Tags = strings.Fields(tags)
	if tank.Level, err = requiredNumber(item, "level", listPath); err != nil {
		return nil, err
	}
	if node := item.Child("combatRole"); node != nil {
		for _, role := range node.Children {
			tank.CombatRole[role.Name] = strings.TrimSpace(role.Text)
		}
	}
	modes, err := requiredText(item, "configurationModes", listPath)
	if err != nil {
		return nil, err
	}
	tank.ConfigurationModes = strings.Fields(modes)
	if value, ok := xml.ChildText(item, "leasing"); ok {
		leasing, err := parseLeasing(value, listPath)
		if err != nil {
			return nil, err
		}
		tank.Leasing = &leasing
	}
	return tank, nil
}

func showroomModel(dataDir string, item *xml.Node, nation string, info *Nation, localized map[string]string) *ClientTank {
	modelRoot := path.Join("3d/Tanks", info.ModelFolder)
	scene := ResolveClientAsset(dataDir, path.Join(modelRoot, item.Name+".sc2.dvpl"))
	if scene == "" {
		return nil
	}
	geometry := ResolveClientAsset(dataDir, path.Join(modelRoot, item.Name+".scg.dvpl"))
	if geometry == "" {
		return nil
	}
	tank := &ClientTank{
		ID:          item.Name,
		Nation:      nation,
		NationLabel: info.Label,
		Name:        strings.ReplaceAll(item.Name, "_", " "),
		ModelFiles:  TankModelFiles{Scene: scene, Geometry: geometry},
	}
	if level, ok := xml.ChildText(item, "level"); ok {
		tank.Level = &level
	}
	if userString, ok := xml.ChildText(item, "userString"); ok {
		tank.UserString = &userString
		if name, ok := localized[userString]; ok {
			tank.Name = name
		}
	}
	return tank
}

func requiredNode(node *xml.Node, name, path string) (*xml.Node, error) {
	if child := node.Child(name); child != nil {
		return child, nil
	}
	return nil, blitz.ErrorAt("CLIENT_XML_INVALID", fmt.Sprintf("%s is missing <%s>", node.Name, name), path)
}

func requiredText(node *xml.Node, name, path string) (string, error) {
	if text, ok := xml.ChildText(node, name); ok {
		return text, nil
	}
	return "", blitz.ErrorAt("CLIENT_XML_INVALID", fmt.Sprintf("%s has an empty or missing <%s>", node.Name, name), path)
}

func requiredNumber(node *xml.Node, name, path string) (int64, error) {
	text, err := requiredText(node, name, path)
	if err != nil {
		return 0, err
	}
	number, err := rs.ParseInt(text, 64)
	if err != nil {
		return 0, blitz.ErrorAt("CLIENT_XML_INVALID", fmt.Sprintf("%s has an invalid integer in <%s>", node.Name, name), path)
	}
	return number, nil
}

func parsePrice(node *xml.Node, path string) (ClientPrice, error) {
	value, err := rs.ParseInt(strings.TrimSpace(node.Text), 64)
	if err != nil {
		return ClientPrice{}, blitz.ErrorAt("CLIENT_XML_INVALID", node.Name+" contains an invalid price", path)
	}
	return ClientPrice{Value: value, IsGold: node.Child("gold") != nil}, nil
}

func parseLeasing(value, path string) (string, error) {
	invalid := blitz.ErrorAt("CLIENT_XML_INVALID", "invalid leasing date "+rs.Quote(value), path)
	date, clock, ok := strings.Cut(value, " ")
	if !ok {
		return "", invalid
	}
	dateParts := strings.Split(date, ".")
	timeParts := strings.Split(clock, ":")
	if len(dateParts) != 3 || len(timeParts) != 2 {
		return "", invalid
	}
	inRange := func(text string, low, high uint64) bool {
		number, err := rs.ParseUint(text, 32)
		return err == nil && number >= low && number <= high
	}
	if !inRange(dateParts[0], 1, 31) || !inRange(dateParts[1], 1, 12) ||
		!inRange(dateParts[2], 1, 1<<32-1) || !inRange(timeParts[0], 0, 23) || !inRange(timeParts[1], 0, 59) {
		return "", invalid
	}
	return dateParts[2] + "-" + dateParts[1] + "-" + dateParts[0] + "T" + clock + ":00", nil
}

func localize(value string, localized map[string]string) string {
	if translated, ok := localized[value]; ok {
		return translated
	}
	return value
}
