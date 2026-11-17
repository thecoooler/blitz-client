//go:build linux && (amd64 || arm64)

package game

import (
	"encoding/json"
	"testing"
)

func TestHyprlandVectorReadsArrayAndString(t *testing.T) {
	for _, test := range []struct {
		in   string
		want hyprlandVector
	}{
		{`[12,38]`, hyprlandVector{X: 12, Y: 38}},
		{`"12,38"`, hyprlandVector{X: 12, Y: 38}},
		{`[1512.0,814.0]`, hyprlandVector{X: 1512, Y: 814}},
		{`null`, hyprlandVector{}},
	} {
		var client hyprlandClient
		if err := json.Unmarshal([]byte(`{"at":`+test.in+`}`), &client); err != nil {
			t.Fatalf("%s: %v", test.in, err)
		}
		if client.At != test.want {
			t.Errorf("%s: got %+v, want %+v", test.in, client.At, test.want)
		}
	}
}

func TestHyprlandGameWindowPrefersTheFocusedWindow(t *testing.T) {
	clients := []hyprlandClient{
		{Address: "0x1", PID: 7, Mapped: true, Size: hyprlandVector{X: 1512, Y: 814}},
		{Address: "0x2", PID: 7, Mapped: true, Size: hyprlandVector{X: 800, Y: 600}},
	}
	game, ok := hyprlandGameWindow(clients, []int{7}, "0x2")
	if !ok || game.Address != "0x2" {
		t.Fatalf("got %+v (found %t), want the focused window", game, ok)
	}
}

func TestHyprlandGameWindowFallsBackToTheLargestWindow(t *testing.T) {
	clients := []hyprlandClient{
		{Address: "0x1", PID: 7, Mapped: true, Size: hyprlandVector{X: 200, Y: 200}},
		{Address: "0x2", PID: 7, Mapped: true, Size: hyprlandVector{X: 1512, Y: 814}},
		// A helper window of another process, and a hidden window of the client.
		{Address: "0x3", PID: 9, Mapped: true, Size: hyprlandVector{X: 1920, Y: 1080}},
		{Address: "0x4", PID: 7, Mapped: true, Hidden: true, Size: hyprlandVector{X: 1920, Y: 1080}},
	}
	game, ok := hyprlandGameWindow(clients, []int{7}, "")
	if !ok || game.Address != "0x2" {
		t.Fatalf("got %+v (found %t), want the largest shown window", game, ok)
	}
}

func TestHyprlandGameWindowSkipsHelperWindows(t *testing.T) {
	clients := []hyprlandClient{{Address: "0x1", PID: 7, Mapped: true, Size: hyprlandVector{X: 32, Y: 32}}}
	if game, ok := hyprlandGameWindow(clients, []int{7}, ""); ok {
		t.Fatalf("got %+v, want no window", game)
	}
}
