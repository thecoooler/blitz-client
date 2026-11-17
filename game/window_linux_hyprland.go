//go:build linux && (amd64 || arm64)

package game

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// Hyprland reports its windows over its own Unix sockets instead of the EWMH
// properties of an X11 window manager: the request socket answers `j/clients`
// and `j/activewindow`, and the event socket announces changes. Wine and Proton
// games run on XWayland, where _NET_ACTIVE_WINDOW is empty or holds a window
// that closed long ago, so the compositor itself is asked when it is Hyprland.
// Positions and sizes are logical coordinates, as the compositor lays windows
// out, which is what the overlay places itself by.

const hyprlandTimeout = 2 * time.Second

// hyprlandClient is one window of `j/clients`, of the fields this package uses.
type hyprlandClient struct {
	Address    string         `json:"address"`
	Mapped     bool           `json:"mapped"`
	Hidden     bool           `json:"hidden"`
	At         hyprlandVector `json:"at"`
	Size       hyprlandVector `json:"size"`
	PID        int            `json:"pid"`
	Class      string         `json:"class"`
	Title      string         `json:"title"`
	XWayland   bool           `json:"xwayland"`
	Floating   bool           `json:"floating"`
	Fullscreen int            `json:"fullscreen"`
}

// hyprlandVector is a position or a size. Hyprland reports both as an array in
// JSON, and as a comma-separated string in the releases before it did.
type hyprlandVector struct{ X, Y int }

func (v *hyprlandVector) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		return nil
	}
	if strings.HasPrefix(text, `"`) {
		var pair string
		if err := json.Unmarshal(data, &pair); err != nil {
			return err
		}
		text = "[" + pair + "]"
	}
	var values []float64
	if err := json.Unmarshal([]byte(text), &values); err != nil {
		return err
	}
	if len(values) > 0 {
		v.X = int(values[0])
	}
	if len(values) > 1 {
		v.Y = int(values[1])
	}
	return nil
}

var errHyprlandUnavailable = errors.New("game: the Hyprland IPC sockets are not available")

// hyprlandSocket returns the request socket of the running Hyprland instance.
func hyprlandSocket() (string, bool) {
	signature, ok := os.LookupEnv("HYPRLAND_INSTANCE_SIGNATURE")
	if !ok || signature == "" {
		return "", false
	}
	if dir, ok := os.LookupEnv("XDG_RUNTIME_DIR"); ok && dir != "" {
		return filepath.Join(dir, "hypr", signature, ".socket.sock"), true
	}
	// Releases before 0.40 kept the sockets under /tmp.
	return filepath.Join("/tmp", "hypr", signature, ".socket.sock"), true
}

// hyprlandRequest sends one command to the compositor and returns its answer.
func hyprlandRequest(command string) ([]byte, error) {
	path, ok := hyprlandSocket()
	if !ok {
		return nil, errHyprlandUnavailable
	}
	conn, err := net.DialTimeout("unix", path, hyprlandTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(hyprlandTimeout)); err != nil {
		return nil, err
	}
	if _, err = io.WriteString(conn, command); err != nil {
		return nil, err
	}
	answer, err := io.ReadAll(conn)
	if err != nil {
		return nil, err
	}
	return answer, nil
}

// hyprlandClients lists the windows the compositor manages.
func hyprlandClients() ([]hyprlandClient, error) {
	answer, err := hyprlandRequest("j/clients")
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(answer))) == 0 {
		return nil, nil
	}
	var clients []hyprlandClient
	if err = json.Unmarshal(answer, &clients); err != nil {
		return nil, err
	}
	return clients, nil
}

// hyprlandActive returns the window that holds the focus, or the zero value
// when none does.
func hyprlandActive() (hyprlandClient, error) {
	answer, err := hyprlandRequest("j/activewindow")
	if err != nil {
		return hyprlandClient{}, err
	}
	if len(strings.TrimSpace(string(answer))) == 0 {
		return hyprlandClient{}, nil
	}
	var active hyprlandClient
	if err = json.Unmarshal(answer, &active); err != nil {
		return hyprlandClient{}, err
	}
	return active, nil
}

var hyprlandOnce = sync.OnceValue(func() bool {
	_, err := hyprlandClients()
	return err == nil
})

// hyprlandSupported reports whether this session is Hyprland and answers.
func hyprlandSupported() bool { return hyprlandOnce() }

// inspectHyprlandWindows reports the client window from the compositor, and
// whether the compositor answered at all: without Hyprland the caller falls
// back to the EWMH properties of X11.
func inspectHyprlandWindows(pids, ignore []int) (windowReport, bool, error) {
	clients, err := hyprlandClients()
	if err != nil {
		return windowReport{}, false, nil
	}
	active, err := hyprlandActive()
	if err != nil {
		active = hyprlandClient{}
	}
	report := windowReport{known: true}
	if active.PID != 0 && slices.Contains(pids, active.PID) && !slices.Contains(ignore, active.PID) {
		report.focused = true
	}
	game, ok := hyprlandGameWindow(clients, pids, active.Address)
	if !ok {
		return report, true, nil
	}
	report.pid = game.PID
	window := &Window{
		Bounds:     Rect{X: game.At.X, Y: game.At.Y, Width: game.Size.X, Height: game.Size.Y},
		Minimized:  game.Hidden,
		Fullscreen: game.Fullscreen != 0,
	}
	window.Visible = game.Mapped && !game.Hidden
	report.window = window
	return report, true, nil
}

// hyprlandGameWindow picks the window of the client: the focused one when it
// belongs to the client, otherwise the largest one shown, so that helper
// windows such as launchers do not stand for the game.
func hyprlandGameWindow(clients []hyprlandClient, pids []int, activeAddress string) (hyprlandClient, bool) {
	var chosen hyprlandClient
	found := false
	for _, client := range clients {
		if !slices.Contains(pids, client.PID) || !client.Mapped || client.Hidden {
			continue
		}
		if client.Size.X < minimumWindowSide || client.Size.Y < minimumWindowSide {
			continue
		}
		if client.Address != "" && client.Address == activeAddress {
			return client, true
		}
		if !found || client.Size.X*client.Size.Y > chosen.Size.X*chosen.Size.Y {
			chosen, found = client, true
		}
	}
	return chosen, found
}
