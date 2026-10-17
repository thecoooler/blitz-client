package client

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/lossy"
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
	ID                      int64   `json:"id"`
	LocalName               string  `json:"localName"`
	AvailableInTrainingRoom bool    `json:"availableInTrainingRoom"`
	AvailableModes          []int64 `json:"availableModes"`
	SpriteFrame             int64   `json:"spriteFrame"`
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
	availableNations, err := readAvailableNations(dataDir)
	if err != nil {
		return nil, err
	}
	metadata := []ClientTankMetadata{}
	modelsByNation := make(map[string][]ClientTank)

	for _, nation := range availableNations {
		listPath := ResolveClientAsset(dataDir, "XML/item_defs/vehicles/"+nation+"/list.xml")
		if listPath == "" {
			continue
		}
		data, err := ReadClientAsset(listPath)
		if err != nil {
			return nil, err
		}
		root, err := xml.Parse(data)
		if err != nil {
			return nil, blitz.ErrorAt("CLIENT_XML_INVALID", err.Error(), listPath)
		}
		var info *Nation
		for index := range nations {
			if nations[index].ID == nation {
				info = &nations[index]
			}
		}
		for _, item := range root.Children {
			if fullCatalog {
				tank, err := tankMetadata(item, nation, localized, listPath)
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

func readAvailableNations(dataDir string) ([]string, error) {
	path := ResolveClientAsset(dataDir, "available_nations.yaml")
	if path == "" {
		out := make([]string, len(nations))
		for index, nation := range nations {
			out[index] = nation.ID
		}
		return out, nil
	}
	data, err := ReadClientAsset(path)
	if err != nil {
		return nil, err
	}
	inList := false
	var result []string
	for _, line := range rs.Lines(lossy.String(data)) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if trimmed == "available_nations:" {
			inList = true
			continue
		}
		if !inList {
			continue
		}
		value, ok := strings.CutPrefix(trimmed, "-")
		if !ok {
			break
		}
		if nation := yamlString(strings.TrimSpace(value)); nation != "" {
			result = append(result, nation)
		}
	}
	if len(result) == 0 {
		return nil, blitz.ErrorAt("CLIENT_YAML_INVALID", "available_nations.yaml contains no nation entries", path)
	}
	return result, nil
}

func readMaps(dataDir string, localized map[string]string) ([]ClientMap, error) {
	path := ResolveClientAsset(dataDir, "maps.yml")
	if path == "" {
		path = ResolveClientAsset(dataDir, "maps.yaml")
	}
	if path == "" {
		return nil, blitz.ErrorAt("CLIENT_DATA_INVALID",
			"maps.yml or maps.yaml was not found in the client resource directory", dataDir)
	}
	data, err := ReadClientAsset(path)
	if err != nil {
		return nil, err
	}
	maps, err := parseMaps(lossy.String(data), localized)
	if err != nil {
		return nil, blitz.ErrorAt("CLIENT_YAML_INVALID", err.Error(), path)
	}
	return maps, nil
}

type mapEntry struct {
	name   string
	values map[string]string
}

func parseMaps(source string, localized map[string]string) ([]ClientMap, error) {
	inMaps := false
	var current *mapEntry
	var maps []ClientMap
	for _, line := range rs.Lines(source) {
		trimmed := strings.TrimSpace(line)
		if !inMaps {
			if trimmed == "maps:" {
				inMaps = true
			}
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "        "):
			if err := finishMap(current, localized, &maps); err != nil {
				return nil, err
			}
			key, ok := strings.CutSuffix(trimmed, ":")
			if !ok {
				return nil, fmt.Errorf("invalid map entry %s", rs.Quote(trimmed))
			}
			current = &mapEntry{name: yamlString(key), values: map[string]string{}}
		case strings.HasPrefix(line, "        "):
			if key, value, ok := strings.Cut(trimmed, ":"); ok && current != nil {
				current.values[key] = strings.TrimSpace(value)
			}
		case !strings.HasPrefix(line, " ") && strings.HasSuffix(trimmed, ":"):
			goto done
		}
	}
done:
	if err := finishMap(current, localized, &maps); err != nil {
		return nil, err
	}
	if !inMaps || len(maps) == 0 {
		return nil, fmt.Errorf("maps.yaml contains no map entries")
	}
	return maps, nil
}

func finishMap(entry *mapEntry, localized map[string]string, output *[]ClientMap) error {
	if entry == nil {
		return nil
	}
	name := rs.Quote(entry.name)
	get := func(key string) (string, bool) {
		value, ok := entry.values[key]
		return value, ok
	}
	localName := ""
	if value, ok := get("localName"); ok {
		localName = yamlString(value)
	}
	idText, ok := get("id")
	if !ok {
		return fmt.Errorf("map %s is missing id", name)
	}
	id, err := rs.ParseInt(idText, 64)
	if err != nil {
		return fmt.Errorf("map %s has invalid id", name)
	}
	frame, ok := optionalNumberField(entry.values, "frame")
	if !ok {
		frame, ok = optionalNumberField(entry.values, "spriteFrame")
	}
	if !ok {
		return fmt.Errorf("map %s has neither frame nor spriteFrame", name)
	}
	modes := []int64{}
	if value, ok := get("availableModes"); ok {
		if modes, err = yamlNumberList(value); err != nil {
			return err
		}
	}
	training := false
	for _, key := range []string{"availableInTrainingRoom", "avaliableInTrainingRoom"} {
		if value, ok := get(key); ok {
			if parsed, ok := yamlBool(value); ok && parsed {
				training = true
			}
		}
	}
	if translated, ok := localized["#maps:"+entry.name+":"+localName]; ok {
		localName = translated
	}
	*output = append(*output, ClientMap{
		ID:                      id,
		LocalName:               localName,
		AvailableInTrainingRoom: training,
		AvailableModes:          modes,
		SpriteFrame:             frame,
	})
	return nil
}

func optionalNumberField(values map[string]string, key string) (int64, bool) {
	value, ok := values[key]
	if !ok {
		return 0, false
	}
	value = strings.TrimSpace(value)
	if value == "" || value == "null" || value == "~" {
		return 0, false
	}
	number, err := rs.ParseInt(value, 64)
	return number, err == nil
}

func yamlString(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, `"`) {
		escaped := false
		for index := 1; index < len(value); index++ {
			switch {
			case value[index] == '\\' && !escaped:
				escaped = true
			case value[index] == '"' && !escaped:
				var parsed string
				if json.Unmarshal([]byte(value[:index+1]), &parsed) == nil {
					return parsed
				}
				index = len(value)
			default:
				escaped = false
			}
		}
	}
	if len(value) >= 2 && strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
	}
	return value
}

func yamlNumberList(value string) ([]int64, error) {
	value = strings.TrimSpace(value)
	body, ok := strings.CutPrefix(value, "[")
	if ok {
		body, ok = strings.CutSuffix(body, "]")
	}
	if !ok {
		return nil, fmt.Errorf("expected a numeric YAML list, got %s", rs.Quote(value))
	}
	numbers := []int64{}
	for _, part := range strings.Split(body, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		number, err := rs.ParseInt(part, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %s in YAML list", rs.Quote(part))
		}
		numbers = append(numbers, number)
	}
	return numbers, nil
}

func yamlBool(value string) (bool, bool) {
	switch strings.TrimSpace(value) {
	case "true", "True", "TRUE":
		return true, true
	case "false", "False", "FALSE":
		return false, true
	}
	return false, false
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
