package client

import (
	"compress/zlib"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/thecoooler/blitz-client/internal/pickle"
	"github.com/thecoooler/blitz-client/resources/xml"
)

func writeCrewConfig(t *testing.T, payload string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server_config.dat")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zlib.NewWriter(file)
	if _, err = writer.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCrewSkillParametersAndTraining(t *testing.T) {
	source := `<root>
 <skillsByClasses><heavyTank>desperado</heavyTank></skillsByClasses>
 <skills><desperado><userString>name</userString><type>continuous</type>
 <vehicleHealthFraction>0.15</vehicleHealthFraction><gunReloadTimeFactorPerLevel>0.01</gunReloadTimeFactorPerLevel>
 <futureCondition><enabled>false</enabled><mode>a</mode><mode>b</mode></futureCondition>
 <zero>0.00</zero><icon><name>ignored</name></icon></desperado></skills></root>`
	root, err := xml.Parse([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	skills, err := ParseCrewSkills(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("skills: %+v", skills)
	}
	skill := skills[0]
	want := map[string]any{"vehicleHealthFraction": 0.15, "gunReloadTimeFactorPerLevel": 0.01, "zero": float64(0), "futureCondition": map[string]any{"enabled": false, "mode": []any{"a", "b"}}}
	if skill.TankClass != "heavyTank" || skill.UserString != "name" || !reflect.DeepEqual(skill.Parameters, want) || skill.Levels != nil {
		t.Fatalf("skill: %+v", skill)
	}
	// Protocol 0 fixture includes byte-string keys, numeric mode keys and tuples.
	path := writeCrewConfig(t, "(dS'gradeUpSkillCost'\n(lI25000\naI50000\nasS'avatarSkillsEconomics'\n(dI0\n(dS'rate'\nI1\nsS'limit'\nI5\nsS'cost'\n(I0\nI0\ntsS'enabled'\nI01\nsss.")
	training, err := ReadCrewSkillTraining(path)
	if err != nil {
		t.Fatal(err)
	}
	if training.MaxLevel != 2 || !reflect.DeepEqual(training.GradeUpSkillCost, []int64{25000, 50000}) {
		t.Fatalf("training: %+v", training)
	}
	mode := training.Economics["0"].(map[string]any)
	if mode["limit"] != int64(5) || mode["enabled"] != true || !reflect.DeepEqual(mode["cost"], []any{int64(0), int64(0)}) {
		t.Fatalf("economics: %v", mode)
	}
	ApplyCrewSkillTraining(skills, training)
	wantLevels := []CrewSkillLevel{{Level: 1, XP: 25000, Parameters: map[string]float64{"gunReloadTimeFactorPerLevel": 0.01}}, {Level: 2, XP: 50000, Parameters: map[string]float64{"gunReloadTimeFactorPerLevel": 0.02}}}
	if !reflect.DeepEqual(skills[0].Levels, wantLevels) {
		t.Fatalf("levels: %+v", skills[0].Levels)
	}
	if _, err = json.Marshal(skills); err != nil {
		t.Fatal(err)
	}
	ApplyCrewSkillTraining(skills, nil)
	if skills[0].Levels != nil {
		t.Fatal("training was not cleared")
	}
}

func TestCrewSkillTrainingRejectsInvalidCache(t *testing.T) {
	for _, payload := range []string{
		"not a pickle", "N.", "(d.", "(dS'gradeUpSkillCost'\n(ls.",
		"(dS'gradeUpSkillCost'\n(lI-1\nas.",
		"(dS'gradeUpSkillCost'\n(lI1\nasS'avatarSkillsEconomics'\nI1\ns.",
		// Self-reference in the economics must fail without recursing indefinitely.
		"(dp0\nS'gradeUpSkillCost'\n(lI1\nasS'avatarSkillsEconomics'\ng0\ns.",
	} {
		path := writeCrewConfig(t, payload)
		_, err := ReadCrewSkillTraining(path)
		requireError(t, err, "CLIENT_CONFIG_INVALID", path)
	}
	path := filepath.Join(t.TempDir(), "broken.dat")
	if err := os.WriteFile(path, []byte("invalid zlib"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := ReadCrewSkillTraining(path)
	requireError(t, err, "CLIENT_CONFIG_INVALID", path)
}

// A tiny acyclic pickle can describe an exponential JSON tree using memo aliases.
func TestCrewSkillTrainingRejectsMemoExpansion(t *testing.T) {
	var payload strings.Builder
	payload.WriteString("(lI0\nap0\n0")
	for level := 1; level <= 30; level++ {
		fmt.Fprintf(&payload, "(lg%d\nag%d\nap%d\n0", level-1, level-1, level)
	}
	payload.WriteString("(dS'gradeUpSkillCost'\n(lI1\nasS'avatarSkillsEconomics'\n(dI0\ng30\nss.")
	path := writeCrewConfig(t, payload.String())
	_, err := ReadCrewSkillTraining(path)
	requireError(t, err, "CLIENT_CONFIG_INVALID", path)
	if !strings.Contains(err.Error(), "expanded values") {
		t.Fatalf("unexpected rejection: %v", err)
	}
}

func TestCrewConfigExpansionBudget(t *testing.T) {
	shared := &pickle.List{Items: []pickle.Value{int64(0), int64(1)}}
	tree := &pickle.List{Items: []pickle.Value{shared, shared}}
	budget := crewConfigBudget{nodes: 7, stringBytes: 10}
	got, err := crewConfigJSON(tree, 0, &budget)
	want := []any{[]any{int64(0), int64(1)}, []any{int64(0), int64(1)}}
	if err != nil || !reflect.DeepEqual(got, want) || budget.nodes != 0 {
		t.Fatalf("got=%v budget=%+v err=%v", got, budget, err)
	}
	budget = crewConfigBudget{nodes: 6, stringBytes: 10}
	if _, err := crewConfigJSON(tree, 0, &budget); err == nil {
		t.Fatal("accepted too many expanded values")
	}

	// Repeated text must consume the string budget on every occurrence.
	text := pickle.Bytes("abcdef")
	budget = crewConfigBudget{nodes: 10, stringBytes: 11}
	if _, err := crewConfigJSON(pickle.Tuple{text, text}, 0, &budget); err == nil || !strings.Contains(err.Error(), "strings exceed") {
		t.Fatalf("unexpected result for repeated text: %v", err)
	}
	// Dictionary keys also contribute to the expanded string budget.
	budget = crewConfigBudget{nodes: 10, stringBytes: 5}
	dict := &pickle.Dict{Entries: []pickle.Entry{{Key: text, Value: nil}}}
	if _, err := crewConfigJSON(dict, 0, &budget); err == nil || !strings.Contains(err.Error(), "strings exceed") {
		t.Fatalf("unexpected result for dictionary key: %v", err)
	}
}
