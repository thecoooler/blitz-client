// Package game reports whether the Tanks Blitz client is running and whether
// the player is currently in it: the game window is the foreground window and
// is visible on screen. An overlay polls Detect, or uses Watch, to decide when
// to show itself.
//
// Windows and macOS are fully supported. Linux finds native and Wine/Proton
// processes; window state requires an X11 or XWayland display and libX11. The
// package does not use cgo.
package game

import (
	"context"
	"os"
	"reflect"
	"slices"
	"strings"
	"time"
)

// DefaultProcessNames are the executable names of the known clients: the
// macOS app, and the Lesta and Wargaming Windows builds. Matching ignores
// letter case and a trailing ".exe".
var DefaultProcessNames = []string{"Tanks Blitz", "tanksblitz", "wotblitz", "World of Tanks Blitz"}

// Options configures detection. The zero value uses DefaultProcessNames.
type Options struct {
	// ProcessNames replaces DefaultProcessNames when it is not empty.
	ProcessNames []string
	// IgnorePIDs lists processes whose windows are transparent to focus
	// detection, such as an overlay that takes focus. The current process is
	// always ignored: when it owns the foreground window, the topmost window
	// of another process decides focus.
	IgnorePIDs []int
}

// State is a snapshot of the client and its window.
type State struct {
	// Running reports that a client process exists.
	Running bool `json:"running"`
	// PID and Executable identify the client process, preferring the one that
	// owns Window. Executable is empty when the path cannot be read.
	PID        int    `json:"pid,omitempty"`
	Executable string `json:"executable,omitempty"`
	// Window is the main client window, or nil when none was found.
	Window *Window `json:"window"`
	// FocusKnown is false when this platform or session cannot report
	// windows (for example, a Wayland-only Linux session). Focused and
	// InGame are then always false.
	FocusKnown bool `json:"focus_known"`
	// Focused reports that the client is the foreground application, so its
	// window is above other windows and receives keyboard input.
	Focused bool `json:"focused"`
	// InGame reports that the player is in the client: it is focused and its
	// window is visible on screen.
	InGame bool `json:"in_game"`
}

// Window describes the client window. Unknown cursor states are nil.
type Window struct {
	Bounds Rect `json:"bounds"`
	// Visible reports that the window is shown on the current desktop: not
	// minimized, hidden, or cloaked on another virtual desktop.
	Visible bool `json:"visible"`
	// Minimized reports an iconified window. On macOS it also covers hidden
	// applications and windows on another Space.
	Minimized bool `json:"minimized"`
	// Maximized reports a zoomed window. macOS has no separate maximized
	// state, so it matches Fullscreen there.
	Maximized bool `json:"maximized"`
	// Fullscreen reports that the window covers a whole display.
	Fullscreen bool `json:"fullscreen"`
	// CursorHidden reports that the system cursor is hidden, as the client
	// does while the mouse controls the camera (Windows and macOS).
	CursorHidden *bool `json:"cursor_hidden"`
	// CursorCaptured reports that the cursor is confined to a screen region
	// smaller than the desktop (Windows only).
	CursorCaptured *bool `json:"cursor_captured"`
}

// Rect is a rectangle in screen coordinates: pixels on Windows and Linux,
// points on macOS.
type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// process is a running client process found by the platform scanner.
type process struct {
	pid        int
	executable string
}

// windowReport is what the platform window inspector returns.
type windowReport struct {
	known   bool
	pid     int // owner of window, 0 when no window was found
	window  *Window
	focused bool
}

// Detect takes one snapshot of the client and its window.
func Detect(options Options) (State, error) {
	names := options.ProcessNames
	if len(names) == 0 {
		names = DefaultProcessNames
	}
	processes, err := findProcesses(newMatcher(names))
	if err != nil {
		return State{}, err
	}
	if len(processes) == 0 {
		return State{FocusKnown: windowsSupported()}, nil
	}
	pids := make([]int, len(processes))
	for i, proc := range processes {
		pids[i] = proc.pid
	}
	ignore := append([]int{os.Getpid()}, options.IgnorePIDs...)
	report, err := inspectWindows(pids, ignore)
	if err != nil {
		return State{}, err
	}

	chosen := processes[0]
	for _, proc := range processes {
		if proc.pid == report.pid {
			chosen = proc
		}
	}
	state := State{
		Running:    true,
		PID:        chosen.pid,
		Executable: chosen.executable,
		Window:     report.window,
		FocusKnown: report.known,
		Focused:    report.known && report.focused,
	}
	state.InGame = state.Focused && state.Window != nil && state.Window.Visible
	return state, nil
}

// Watch polls Detect every interval until ctx is done. It calls emit with
// the first result and then whenever the state or error changes. A
// non-positive interval defaults to 500 ms.
func Watch(ctx context.Context, interval time.Duration, options Options, emit func(State, error)) {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var last State
	var lastErr string
	first := true
	for {
		state, err := Detect(options)
		errText := ""
		if err != nil {
			errText = err.Error()
		}
		if first || errText != lastErr || !reflect.DeepEqual(state, last) {
			emit(state, err)
			first, last, lastErr = false, state, errText
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// matcher compares executable names case-insensitively without ".exe".
type matcher struct{ names []string }

func newMatcher(names []string) matcher {
	m := matcher{}
	for _, name := range names {
		if name = normalizeName(name); name != "" {
			m.names = append(m.names, name)
		}
	}
	return m
}

// normalizeName returns the lowercase base name of a Unix or Windows path
// without a trailing ".exe".
func normalizeName(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.TrimSuffix(name, ".exe")
}

func (m matcher) match(name string) bool {
	return slices.Contains(m.names, normalizeName(name))
}

// matchTruncated accepts a kernel process name cut to limit bytes, which a
// longer configured name may start with.
func (m matcher) matchTruncated(name string, limit int) bool {
	if m.match(name) {
		return true
	}
	if len(name) < limit {
		return false
	}
	prefix := normalizeName(name)
	for _, candidate := range m.names {
		if strings.HasPrefix(candidate, prefix) {
			return true
		}
	}
	return false
}

// coversDisplay reports whether a window rectangle covers a display.
func coversDisplay(window, display Rect) bool {
	return display.Width > 0 && display.Height > 0 &&
		window.X <= display.X && window.Y <= display.Y &&
		window.X+window.Width >= display.X+display.Width &&
		window.Y+window.Height >= display.Y+display.Height
}

func boolPtr(value bool) *bool { return &value }
