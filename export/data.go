package export

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/resources/xml"
	"go.yaml.in/yaml/v3"
)

// StrictLocalization describes the localization included by the v1 exporter.
// A dictionary entry may be null; an absent key is not the same as null.
type Localization struct {
	RU map[string]*string `json:"ru"`
}

// Snapshot is the one offline interchange format for data and media consumers
// and the untyped projection returned by ReadClientExport and
// DecodeClientExport. Omitted sections were not requested; included empty
// sections have empty containers.
// Game, ExportedAt (UTC, RFC 3339) and ExporterVersion are written for every
// selection; packs can change data between exports of the same game version.
type Snapshot struct {
	Format            string                    `json:"format"`
	Version           int                       `json:"version"`
	Included          []string                  `json:"included"`
	Game              *SnapshotGame             `json:"game,omitempty"`
	ExportedAt        string                    `json:"exportedAt,omitempty"`
	ExporterVersion   Optional[string]          `json:"exporterVersion,omitzero"`
	Nations           []string                  `json:"nations"`
	Tanks             []SnapshotTank            `json:"tanks"`
	Localization      *Localization             `json:"localization,omitempty"`
	Maps              *RawMaps                  `json:"maps,omitempty"`
	Modes             []client.ClientGameMode   `json:"modes,omitempty"`
	Components        map[string]map[string]any `json:"components,omitempty"`
	Equipment         any                       `json:"equipment,omitempty"`
	Provisions        map[string]any            `json:"provisions,omitempty"`
	Settings          map[string]any            `json:"settings,omitempty"`
	HangarParameters  any                       `json:"hangarParameters,omitempty"`
	CrewSkills        []client.ClientCrewSkill  `json:"crewSkills,omitempty"`
	CrewSkillTraining *client.CrewSkillTraining `json:"crewSkillTraining,omitempty"`
	Issues            []ExportIssue             `json:"issues"`
}

// SnapshotTank keeps all properties and media references of one vehicle together.
type SnapshotTank struct {
	ID            string             `json:"id"`
	Nation        string             `json:"nation"`
	ClientID      int64              `json:"clientId"`
	TechName      string             `json:"techName"`
	Metadata      any                `json:"metadata,omitempty"`
	Parameters    any                `json:"parameters,omitempty"`
	Configuration any                `json:"configuration,omitempty"`
	Icons         map[string]*string `json:"icons,omitempty"`
	Model         *SnapshotModel     `json:"model,omitempty"`
}

// xmlValue uses familiar field names and scalar/null/list shapes. Only component
// selection containers use named arrays, so configuration order is explicit.
func xmlValue(n *xml.Node) any {
	turrets := strings.TrimPrefix(n.Name, "turrets")
	isTurrets := turrets != n.Name && strings.Trim(turrets, "0123456789") == ""
	ordered := slices.Contains([]string{"guns", "shots", "chassis", "engines", "fuelTanks", "radios"}, n.Name) || isTurrets
	if ordered {
		entries := []any{}
		for _, child := range n.Children {
			entries = append(entries, map[string]any{"name": child.Name, "properties": xmlValue(child)})
		}
		if strings.TrimSpace(n.Text) == "" && len(n.Attributes) == 0 {
			return entries
		}
		value := map[string]any{"#items": entries}
		if text := strings.TrimSpace(n.Text); text != "" {
			value["#text"] = text
		}
		for key, v := range n.Attributes {
			value["@"+key] = v
		}
		return value
	}
	text := strings.TrimSpace(n.Text)
	if len(n.Children) == 0 && len(n.Attributes) == 0 {
		if text == "" {
			return nil
		}
		return text
	}
	value := map[string]any{}
	if text != "" {
		value["#text"] = text
	}
	for key, v := range n.Attributes {
		value["@"+key] = v
	}
	groups := map[string][]any{}
	for _, child := range n.Children {
		groups[child.Name] = append(groups[child.Name], xmlValue(child))
	}
	for key, group := range groups {
		if len(group) == 1 {
			value[key] = group[0]
		} else {
			value[key] = group
		}
	}
	return value
}

type snapshotReader struct {
	resources *client.Resources
	files     map[string]sourceFile
}

func (r snapshotReader) read(rel string) ([]byte, error) {
	if !r.has(rel) {
		return nil, blitz.ErrorAt("CLIENT_DATA_MISSING", "required export resource is missing", rel)
	}
	if r.resources != nil {
		return r.resources.Open(rel)
	}
	return client.ReadClientAsset(r.files[filepath.FromSlash(rel)].path)
}
func (r snapshotReader) has(rel string) bool {
	if r.resources != nil {
		return r.resources.Exists(rel)
	}
	_, ok := r.files[filepath.FromSlash(rel)]
	return ok
}

// Exists, Open and Location implement client.CatalogSource. The overlay rules
// remain here; the catalog owns document parsing and validation.
func (r snapshotReader) Exists(rel string) bool { return r.has(rel) }

func (r snapshotReader) Open(rel string) ([]byte, error) { return r.read(rel) }

func (r snapshotReader) Location(rel string) string { return rel }

func (r snapshotReader) xml(rel string) (*xml.Node, error) {
	b, err := r.read(rel)
	if err != nil {
		return nil, err
	}
	root, err := xml.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	return root, nil
}
func (r snapshotReader) document(rel string) (any, error) {
	if strings.HasSuffix(rel, ".xml") {
		root, err := r.xml(rel)
		if err != nil {
			return nil, err
		}
		return xmlValue(root), nil
	}
	b, err := r.read(rel)
	if err != nil {
		return nil, err
	}
	var v any
	if err := yaml.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	v, err = dataJSONValue(v)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	return v, nil
}

func extractSnapshot(resources *client.Resources, files map[string]sourceFile, included []string, out string) (*Snapshot, error) {
	r := snapshotReader{resources: resources, files: files}
	data, icons := slices.Contains(included, "data"), slices.Contains(included, "icons")
	nations, err := client.ReadAvailableNations(r)
	if err != nil {
		return nil, err
	}
	game, err := extractGame(r)
	if err != nil {
		return nil, err
	}
	s := &Snapshot{Format: "blitz-client", Version: 1, Included: included, Game: game,
		ExportedAt: time.Now().UTC().Format(time.RFC3339), ExporterVersion: exporterVersion(),
		Nations: []string{}, Tanks: []SnapshotTank{}, Issues: []ExportIssue{}}
	if data {
		s.Components = map[string]map[string]any{}
	}
	for _, nation := range nations {
		if !safeDataName(nation) {
			return nil, fmt.Errorf("invalid nation %q", nation)
		}
		base := "XML/item_defs/vehicles/" + nation + "/"
		list, err := client.ReadVehicleList(r, nation)
		if err != nil {
			return nil, err
		}
		if list == nil {
			// Like the tank catalog, a listed nation without vehicles is skipped.
			continue
		}
		s.Nations = append(s.Nations, nation)
		for _, entry := range list.Entries {
			item := entry.Node
			if !safeDataName(item.Name) {
				return nil, fmt.Errorf("invalid technical name %q", item.Name)
			}
			raw := entry.ID
			if raw == "" {
				return nil, fmt.Errorf("%s:%s missing client id", nation, item.Name)
			}
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || id < 0 {
				return nil, fmt.Errorf("%s:%s invalid client id %q", nation, item.Name, raw)
			}
			key := nation + ":" + strconv.FormatInt(id, 10)
			tank := SnapshotTank{ID: key, Nation: nation, ClientID: id, TechName: item.Name}
			if data {
				// The numeric ID has one authoritative representation on the entity.
				metadata := &xml.Node{Name: item.Name, Attributes: item.Attributes}
				for _, child := range item.Children {
					if child.Name != "id" {
						metadata.Children = append(metadata.Children, child)
					}
				}
				tank.Metadata = xmlValue(metadata)
				tank.Configuration, err = r.document(base + item.Name + ".xml")
				if err != nil {
					return nil, err
				}
			}
			parametersPath := "3d/Tanks/Parameters/" + nation + "/" + item.Name + ".yaml"
			if icons && !data && !r.has(parametersPath) {
				// Icons are optional media; only the data section requires parameters.
				tank.Icons = map[string]*string{"big": nil, "small": nil}
				s.Issues = append(s.Issues, ExportIssue{Tank: tank.ID, Section: "icons", Message: "missing tank parameters with icon paths"})
			} else if data || icons {
				parameters, err := r.document(parametersPath)
				if err != nil {
					return nil, err
				}
				if data {
					tank.Parameters = parameters
				}
				if icons {
					if err := extractIcons(r, out, &tank, parameters, &s.Issues); err != nil {
						return nil, err
					}
				}
			}
			s.Tanks = append(s.Tanks, tank)
		}
		if data {
			components := map[string]any{}
			for _, kind := range []string{"guns", "shells"} {
				value, err := r.document(base + "components/" + kind + ".xml")
				if err != nil {
					return nil, err
				}
				components[kind] = value
			}
			for _, rel := range sortedResources(files, base+"components/") {
				name := strings.TrimPrefix(rel, base+"components/")
				name = strings.TrimSuffix(name, filepath.Ext(name))
				if name == "guns" || name == "shells" {
					continue
				}
				v, err := r.document(rel)
				if err != nil {
					return nil, err
				}
				if _, exists := components[name]; exists {
					return nil, fmt.Errorf("duplicate component document %s", name)
				}
				components[name] = v
			}
			s.Components[nation] = components
		}
	}
	if !data {
		return s, nil
	}
	s.Modes = client.GameModes()
	ru, err := r.read("Strings/ru.yaml")
	if err != nil {
		return nil, err
	}
	var names client.Localization
	if names, err = client.DecodeLocalization(ru); err != nil {
		return nil, fmt.Errorf("Strings/ru.yaml: %w", err)
	}
	s.Localization = &Localization{RU: names}
	if r.has("hangarParamsConfig.yaml") {
		s.HangarParameters, err = r.document("hangarParamsConfig.yaml")
		if err != nil {
			return nil, err
		}
	}
	if r.has("XML/item_defs/tankmen/avatar.xml") {
		root, err := r.xml("XML/item_defs/tankmen/avatar.xml")
		if err != nil {
			return nil, err
		}
		s.CrewSkills, err = client.ParseCrewSkills(root)
		if err != nil {
			return nil, err
		}
	}
	if document, err := client.ReadMapsDocument(r); err != nil {
		return nil, err
	} else if document != nil {
		s.Maps = rawMapsFromDocument(document)
	}
	common := "XML/item_defs/vehicles/common/"
	devices, err := r.document(common + "optional_devices.xml")
	if err != nil {
		return nil, err
	}
	slots, err := r.document(common + "optional_device_slots.xml")
	if err != nil {
		return nil, err
	}
	s.Equipment = map[string]any{"devices": devices, "slots": slots}
	listing, err := r.xml(common + "provisions/list.xml")
	if err != nil {
		return nil, err
	}
	s.Provisions = map[string]any{}
	for _, item := range listing.Children {
		if item.Name != "items" {
			continue
		}
		rel := item.Attributes["path"]
		if !safeDataPath(rel) {
			return nil, fmt.Errorf("invalid provision include %q", rel)
		}
		mode := strings.TrimSuffix(rel, filepath.Ext(rel))
		if _, exists := s.Provisions[mode]; exists {
			return nil, fmt.Errorf("duplicate provision mode %s", mode)
		}
		s.Provisions[mode], err = r.document(common + "provisions/" + rel)
		if err != nil {
			return nil, err
		}
	}
	s.Settings = map[string]any{}
	for _, rel := range sortedResources(files, common) {
		name := strings.TrimPrefix(rel, common)
		if name == "optional_devices.xml" || name == "optional_device_slots.xml" || strings.HasPrefix(name, "provisions/") {
			continue
		}
		name = strings.TrimSuffix(name, filepath.Ext(name))
		if _, exists := s.Settings[name]; exists {
			return nil, fmt.Errorf("duplicate common settings %s", name)
		}
		s.Settings[name], err = r.document(rel)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func extractIcons(r snapshotReader, out string, tank *SnapshotTank, parameters any, issues *[]ExportIssue) error {
	tank.Icons = map[string]*string{"big": nil, "small": nil}
	p, ok := parameters.(map[string]any)
	if !ok {
		return fmt.Errorf("%s parameters must be an object", tank.ID)
	}
	resources, _ := p["resourcesPath"].(map[string]any)
	for _, size := range []string{"big", "small"} {
		resource, _ := resources[size+"IconPath"].(string)
		rel := strings.TrimPrefix(strings.TrimPrefix(resource, "~res:"), "/")
		if resource != "" && !safeDataPath(rel) {
			return fmt.Errorf("unsafe icon path %q", resource)
		}
		if resource != "" {
			for _, ext := range []string{"", ".packed.webp", "@2x.packed.webp", ".webp", ".png"} {
				candidate := rel + ext
				if !r.has(candidate) || FileTypeOf(candidate) != FileTypeImages {
					continue
				}
				b, err := r.read(candidate)
				if err != nil {
					return err
				}
				dest := "images/tanks/" + tank.Nation + "/" + tank.TechName + "/" + size + filepath.Ext(rel+ext)
				if err := writeDataFile(out, dest, b); err != nil {
					return err
				}
				tank.Icons[size] = &dest
				break
			}
		}
		if tank.Icons[size] == nil {
			*issues = append(*issues, ExportIssue{Tank: tank.ID, Section: "icons", Message: "missing " + size + " icon"})
		}
	}
	return nil
}

func sortedResources(files map[string]sourceFile, prefix string) []string {
	result := []string{}
	for rel := range files {
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, prefix) && isDataDocument(rel) {
			result = append(result, rel)
		}
	}
	sort.Strings(result)
	return result
}
func safeDataName(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, "/\\:")
}
func safeDataPath(s string) bool {
	if s == "" || strings.ContainsAny(s, "\\:") || strings.HasPrefix(s, "/") {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == ".." || p == "." || p == "" {
			return false
		}
	}
	return true
}
func isDataDocument(s string) bool {
	return strings.HasSuffix(s, ".xml") || strings.HasSuffix(s, ".yaml") || strings.HasSuffix(s, ".yml")
}
func writeDataFile(root, rel string, data []byte) error {
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	return atomicWrite(p, data)
}

// YAML permits non-string mapping keys (for example equipment levels) and
// non-finite floats. JSON uses their YAML scalar spelling.
func dataJSONValue(v any) (any, error) {
	switch x := v.(type) {
	case map[any]any:
		out := map[string]any{}
		for key, value := range x {
			name, err := yamlKeyName(key)
			if err != nil {
				return nil, err
			}
			if _, ok := out[name]; ok {
				return nil, fmt.Errorf("ambiguous YAML mapping key %q", name)
			}
			converted, err := dataJSONValue(value)
			if err != nil {
				return nil, err
			}
			out[name] = converted
		}
		return out, nil
	case map[string]any:
		for k, v := range x {
			converted, err := dataJSONValue(v)
			if err != nil {
				return nil, err
			}
			x[k] = converted
		}
		return x, nil
	case []any:
		for i, v := range x {
			converted, err := dataJSONValue(v)
			if err != nil {
				return nil, err
			}
			x[i] = converted
		}
		return x, nil
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return yamlFloat(x), nil
		}
		return x, nil
	default:
		return v, nil
	}
}

func yamlKeyName(key any) (string, error) {
	switch k := key.(type) {
	case string:
		return k, nil
	case int:
		return strconv.Itoa(k), nil
	case int64:
		return strconv.FormatInt(k, 10), nil
	case uint64:
		return strconv.FormatUint(k, 10), nil
	case float64:
		return yamlFloat(k), nil
	case bool:
		return strconv.FormatBool(k), nil
	case nil:
		return "null", nil
	case time.Time:
		return k.Format(time.RFC3339Nano), nil
	default:
		return "", fmt.Errorf("unsupported YAML mapping key %v", key)
	}
}

func yamlFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return ".nan"
	case math.IsInf(f, 1):
		return ".inf"
	case math.IsInf(f, -1):
		return "-.inf"
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// SnapshotModel contains only the media attachment; tank identity stays on SnapshotTank.
type SnapshotModel struct {
	Bundle    string  `json:"bundle"`
	Model     string  `json:"model"`
	Armor     *string `json:"armor"`
	ArmorMesh *string `json:"armorMesh"`
	ArmorNote *string `json:"armorNote"`
}
