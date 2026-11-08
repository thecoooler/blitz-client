package client

import "testing"

func TestGameModes(t *testing.T) {
	modes := GameModes()
	if len(modes) == 0 {
		t.Fatal("no game modes")
	}
	ids := map[string]bool{}
	for _, mode := range modes {
		if mode.ID == "" || mode.Name == "" {
			t.Fatalf("incomplete mode: %+v", mode)
		}
		if mode.Description != nil && *mode.Description == "" {
			t.Fatalf("empty description key: %+v", mode)
		}
		if mode.ShortDescription != nil && *mode.ShortDescription == "" {
			t.Fatalf("empty short description key: %+v", mode)
		}
		if ids[mode.ID] {
			t.Fatalf("duplicate mode %q", mode.ID)
		}
		ids[mode.ID] = true
	}
	modes[0].ID = "changed"
	*modes[0].Description = "changed"
	if GameModes()[0].ID == "changed" || *GameModes()[0].Description == "changed" {
		t.Fatal("GameModes returned shared state")
	}
}
