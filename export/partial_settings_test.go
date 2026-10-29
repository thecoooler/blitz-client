package export

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestPartialSettings(t *testing.T) {
	input := `{"combat_roles":{"fighter":{"id":9007199254740993,"default_abilities":["b",null,"a"]},"empty":null},"consumablesDefault":null,"nations":{},"other":{"unknown":[1,null]}}`
	var settings PartialSettings
	if err := decodeStrictJSON([]byte(input), &settings); err != nil {
		t.Fatal(err)
	}
	if !settings.ConsumablesDefault.Null() || !reflect.DeepEqual(settings.UnvalidatedDocuments(), []string{"other"}) {
		t.Fatalf("state: %#v", settings)
	}
	roles, ok := settings.CombatRoles.Get()
	if !ok || !roles["empty"].Null() {
		t.Fatal("null role lost")
	}
	fighter, ok := roles["fighter"].Get()
	if !ok {
		t.Fatal("missing role")
	}
	number, ok := fighter.ID.Get()
	if !ok || number != "9007199254740993" {
		t.Fatalf("id: %s", number)
	}
	abilities, ok := fighter.DefaultAbilities.Get()
	if !ok || len(abilities) != 3 || !abilities[1].Null() {
		t.Fatalf("abilities: %#v", abilities)
	}
	got, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	var a, b map[string]json.RawMessage
	if err := json.Unmarshal([]byte(input), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &b); err != nil {
		t.Fatal(err)
	}
	var av, bv any
	if err := decodeStrictJSON([]byte(input), &av); err != nil {
		t.Fatal(err)
	}
	if err := decodeStrictJSON(got, &bv); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(av, bv) {
		t.Fatalf("round trip: %s", got)
	}
	for _, bad := range []string{`{"combat_roles":{"x":{"id":"1"}}}`, `{"nations":{"x":{"ID":"1"}}}`, `{"nations":{"x":{"new":1}}}`, `{"nations":{"x":{"id":"1","id":"2"}}}`} {
		if err := decodeStrictJSON([]byte(bad), &settings); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	if !strings.Contains(string(got), `9007199254740993`) {
		t.Fatal("number rounded")
	}
}
