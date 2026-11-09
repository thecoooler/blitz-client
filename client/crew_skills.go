package client

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/pickle"
	"github.com/thecoooler/blitz-client/resources/xml"
)

// ClientCrewSkill is a crew skill from XML/item_defs/tankmen/avatar.xml.
type ClientCrewSkill struct {
	// ID is the XML element name, such as lucky_shot; the UI keys the skill's
	// icon by it.
	ID string `json:"id"`
	// UserString, Description, Tip, and Features retain localization keys from
	// <userString>, <effectDescription>, <tipDescription>, and
	// <featuresDescription>; empty when absent.
	UserString  string `json:"userString"`
	Description string `json:"description"`
	Tip         string `json:"tip"`
	Features    string `json:"features"`
	// TankClass is the <skillsByClasses> entry listing the skill, such as
	// lightTank; empty when none does.
	TankClass string `json:"tankClass"`
	// Type is the skill's <type>, such as trigger or continuous.
	Type string `json:"type"`
	// Parameters contains effect coefficients and activation/availability conditions
	// with original XML names. Values are numbers, booleans, strings or nested objects.
	Parameters map[string]any `json:"parameters"`
	// Levels is null until a server training configuration is supplied.
	Levels []CrewSkillLevel `json:"levels"`
}

// CrewSkillLevel contains the XP price and the linear coefficients at a level.
// Coefficients keep their XML names; they are not final vehicle stat multipliers.
type CrewSkillLevel struct {
	Level      int                `json:"level"`
	XP         int64              `json:"xp"`
	Parameters map[string]float64 `json:"parameters"`
}

// CrewSkillTraining is the shared training configuration from the server cache.
type CrewSkillTraining struct {
	MaxLevel         int     `json:"maxLevel"`
	GradeUpSkillCost []int64 `json:"gradeUpSkillCost"`
	// Economics preserves avatarSkillsEconomics, keyed by its numeric mode ID.
	Economics map[string]any `json:"avatarSkillsEconomics"`
}

// ReadCrewSkillTraining reads a zlib-compressed Python pickle server_config_*.dat
// cache. The decoder never executes Python globals. No account progress is read.
func ReadCrewSkillTraining(path string) (*CrewSkillTraining, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, blitz.IOError("could not read server configuration", path, err)
	}
	invalid := func(err error) (*CrewSkillTraining, error) {
		return nil, blitz.ErrorAt("CLIENT_CONFIG_INVALID", err.Error(), path)
	}
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return invalid(err)
	}
	// Bound decoded input, including malformed or unexpectedly large caches.
	data, err = io.ReadAll(io.LimitReader(reader, 64*1024*1024+1))
	reader.Close()
	if err != nil {
		return invalid(err)
	}
	if len(data) > 64*1024*1024 {
		return invalid(fmt.Errorf("server configuration exceeds 64 MiB"))
	}
	value, err := pickle.Decode(data)
	if err != nil {
		return invalid(err)
	}
	dict, ok := value.(*pickle.Dict)
	if !ok {
		return invalid(fmt.Errorf("server configuration must be a dictionary"))
	}
	var costs, economics pickle.Value
	for _, entry := range dict.Entries {
		switch pickleKey(entry.Key) {
		case "gradeUpSkillCost":
			costs = entry.Value
		case "avatarSkillsEconomics":
			economics = entry.Value
		}
	}
	list, ok := costs.(*pickle.List)
	if !ok || len(list.Items) == 0 {
		return invalid(fmt.Errorf("gradeUpSkillCost must be a nonempty list"))
	}
	training := &CrewSkillTraining{MaxLevel: len(list.Items), GradeUpSkillCost: make([]int64, 0, len(list.Items))}
	for _, value := range list.Items {
		cost, ok := value.(int64)
		if !ok || cost < 0 {
			return invalid(fmt.Errorf("gradeUpSkillCost must contain nonnegative integers"))
		}
		training.GradeUpSkillCost = append(training.GradeUpSkillCost, cost)
	}
	if economics != nil {
		budget := crewConfigBudget{nodes: 100_000, stringBytes: 64 * 1024 * 1024}
		converted, err := crewConfigJSON(economics, 0, &budget)
		if err != nil {
			return invalid(err)
		}
		training.Economics, ok = converted.(map[string]any)
		if !ok {
			return invalid(fmt.Errorf("avatarSkillsEconomics must be a dictionary"))
		}
	}
	return training, nil
}

func pickleKey(value pickle.Value) string {
	switch value := value.(type) {
	case string:
		return value
	case pickle.Bytes:
		return string(value)
	case int64:
		return strconv.FormatInt(value, 10)
	}
	return ""
}

// Count each occurrence, including memoized aliases, since JSON expands them.
// Bound strings separately: a few aliases of a large string can exceed the
// output budget even when the number of values is small.
type crewConfigBudget struct {
	nodes       int
	stringBytes int
}

func (budget *crewConfigBudget) takeString(size int) error {
	if size > budget.stringBytes {
		return fmt.Errorf("economics strings exceed 64 MiB")
	}
	budget.stringBytes -= size
	return nil
}

func crewConfigJSON(value pickle.Value, depth int, budget *crewConfigBudget) (any, error) {
	if depth > 64 {
		return nil, fmt.Errorf("economics nesting exceeds 64 levels")
	}
	if budget.nodes == 0 {
		return nil, fmt.Errorf("economics exceeds 100000 expanded values")
	}
	budget.nodes--
	switch value := value.(type) {
	case string:
		if err := budget.takeString(len(value)); err != nil {
			return nil, err
		}
		return value, nil
	case nil, bool, int64:
		return value, nil
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			break
		}
		return value, nil
	case pickle.Bytes:
		if err := budget.takeString(len(value)); err != nil {
			return nil, err
		}
		return string(value), nil
	case *pickle.Dict:
		if len(value.Entries) > budget.nodes {
			return nil, fmt.Errorf("economics exceeds 100000 expanded values")
		}
		out := map[string]any{}
		for _, entry := range value.Entries {
			key := pickleKey(entry.Key)
			if key == "" {
				return nil, fmt.Errorf("unsupported economics dictionary key")
			}
			if err := budget.takeString(len(key)); err != nil {
				return nil, err
			}
			converted, err := crewConfigJSON(entry.Value, depth+1, budget)
			if err != nil {
				return nil, err
			}
			out[key] = converted
		}
		return out, nil
	case *pickle.List:
		return crewConfigSequence(value.Items, depth+1, budget)
	case pickle.Tuple:
		return crewConfigSequence(value, depth+1, budget)
	}
	return nil, fmt.Errorf("unsupported economics value %T", value)
}

func crewConfigSequence(values []pickle.Value, depth int, budget *crewConfigBudget) ([]any, error) {
	// Check before allocating the output slice, including for wide containers.
	if len(values) > budget.nodes {
		return nil, fmt.Errorf("economics exceeds 100000 expanded values")
	}
	out := make([]any, 0, len(values))
	for _, value := range values {
		converted, err := crewConfigJSON(value, depth, budget)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

// ApplyCrewSkillTraining builds level tables using
// only explicit *PerLevel coefficients. Conditions and fixed values stay in Parameters.
func ApplyCrewSkillTraining(skills []ClientCrewSkill, training *CrewSkillTraining) {
	for i := range skills {
		skill := &skills[i]
		skill.Levels = nil
		if training == nil {
			continue
		}
		skill.Levels = make([]CrewSkillLevel, 0, len(training.GradeUpSkillCost))
		for index, xp := range training.GradeUpSkillCost {
			level := CrewSkillLevel{Level: index + 1, XP: xp, Parameters: map[string]float64{}}
			for name, value := range skill.Parameters {
				if strings.HasSuffix(name, "PerLevel") {
					if number, ok := value.(float64); ok {
						level.Parameters[name] = number * float64(level.Level)
					}
				}
			}
			skill.Levels = append(skill.Levels, level)
		}
	}
}

// crewSkillParameters keeps source field names and preserves unknown fields,
// nested values and repeated elements when the client adds new parameters.
func crewSkillParameters(node *xml.Node) map[string]any {
	var parameters map[string]any
	for _, child := range node.Children {
		switch child.Name {
		case "userString", "effectDescription", "tipDescription", "featuresDescription", "icon", "type":
			continue
		}
		if parameters == nil {
			parameters = map[string]any{}
		}
		addCrewXMLValue(parameters, child)
	}
	return parameters
}

func addCrewXMLValue(out map[string]any, node *xml.Node) {
	text := strings.TrimSpace(node.Text)
	var value any = text
	if len(node.Children) > 0 {
		nested := map[string]any{}
		if text != "" {
			nested["#text"] = text
		}
		for _, child := range node.Children {
			addCrewXMLValue(nested, child)
		}
		value = nested
	} else if number, err := strconv.ParseFloat(text, 64); err == nil && !math.IsNaN(number) && !math.IsInf(number, 0) {
		value = number
	} else if text == "true" || text == "false" {
		value = text == "true"
	}
	if previous, exists := out[node.Name]; exists {
		if repeated, ok := previous.([]any); ok {
			out[node.Name] = append(repeated, value)
		} else {
			out[node.Name] = []any{previous, value}
		}
	} else {
		out[node.Name] = value
	}
}

// crewSkillDefinition is one skill parsed from the shared avatar.xml source.
// Localized summaries and raw parameter-bearing definitions are projections of
// this one parse.
type crewSkillDefinition struct {
	ID          string
	UserString  string
	Description string
	Tip         string
	Features    string
	TankClass   string
	Type        string
	Parameters  map[string]any
}

// parseCrewSkillDefinitions reads skills in XML order and resolves the class
// table once. Duplicate class membership is invalid.
func parseCrewSkillDefinitions(root *xml.Node) ([]crewSkillDefinition, error) {
	definitions := []crewSkillDefinition{}
	if root == nil || root.Child("skills") == nil {
		return definitions, nil
	}
	classes := map[string]string{}
	if table := root.Child("skillsByClasses"); table != nil {
		for _, class := range table.Children {
			for _, skill := range strings.Fields(class.Text) {
				if previous, ok := classes[skill]; ok {
					return nil, fmt.Errorf(
						"crew skill %s is listed for both %s and %s", skill, previous, class.Name)
				}
				classes[skill] = class.Name
			}
		}
	}
	for _, node := range root.Child("skills").Children {
		text := func(name string) string {
			value, _ := xml.ChildText(node, name)
			return value
		}
		kind, _ := xml.ChildText(node, "type")
		definitions = append(definitions, crewSkillDefinition{
			ID:          node.Name,
			UserString:  text("userString"),
			Description: text("effectDescription"),
			Tip:         text("tipDescription"),
			Features:    text("featuresDescription"),
			TankClass:   classes[node.Name],
			Type:        kind,
			Parameters:  crewSkillParameters(node),
		})
	}
	return definitions, nil
}

// ParseCrewSkills reads skills in XML order without resolving localization keys.
// The caller supplies the parsed resource so Data/packs overlay rules remain shared.
func ParseCrewSkills(root *xml.Node) ([]ClientCrewSkill, error) {
	definitions, err := parseCrewSkillDefinitions(root)
	if err != nil {
		return nil, blitz.NewError("CLIENT_XML_INVALID", err.Error())
	}
	skills := make([]ClientCrewSkill, 0, len(definitions))
	for _, definition := range definitions {
		skills = append(skills, ClientCrewSkill{
			ID:          definition.ID,
			UserString:  definition.UserString,
			Description: definition.Description,
			Tip:         definition.Tip,
			Features:    definition.Features,
			TankClass:   definition.TankClass,
			Type:        definition.Type,
			Parameters:  definition.Parameters,
		})
	}
	return skills, nil
}
