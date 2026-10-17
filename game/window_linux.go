//go:build amd64 || arm64

package game

import (
	"slices"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Window state comes from the EWMH properties of an X11 window manager. Wine
// and Proton windows live on XWayland in Wayland sessions, so they are still
// visible there; without libX11 or a display, focus is reported as unknown.

const (
	anyPropertyType   = 0
	minimumWindowSide = 64 // smaller windows are helpers, not the game
)

var x11 struct {
	once sync.Once
	err  error
	mu   sync.Mutex

	openDisplay      func(name uintptr) uintptr
	closeDisplay     func(display uintptr) int32
	defaultRoot      func(display uintptr) uint
	internAtom       func(display uintptr, name string, onlyIfExists int32) uint
	getProperty      func(display uintptr, window, property uint, offset, length int64, remove int32, kind uint, actualType *uint, actualFormat *int32, items, bytesAfter *uint, data *unsafe.Pointer) int32
	free             func(data unsafe.Pointer) int32
	getGeometry      func(display uintptr, window uint, root *uint, x, y *int32, width, height, border, depth *uint32) int32
	translate        func(display uintptr, source, destination uint, x, y int32, outX, outY *int32, child *uint) int32
	setErrorHandler  func(handler uintptr) uintptr
	ignoreXErrorFunc uintptr
}

func loadX11() error {
	x11.once.Do(func() {
		var lib uintptr
		for _, name := range []string{"libX11.so.6", "libX11.so"} {
			if lib, x11.err = purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL); x11.err == nil {
				break
			}
		}
		if x11.err != nil {
			return
		}
		purego.RegisterLibFunc(&x11.openDisplay, lib, "XOpenDisplay")
		purego.RegisterLibFunc(&x11.closeDisplay, lib, "XCloseDisplay")
		purego.RegisterLibFunc(&x11.defaultRoot, lib, "XDefaultRootWindow")
		purego.RegisterLibFunc(&x11.internAtom, lib, "XInternAtom")
		purego.RegisterLibFunc(&x11.getProperty, lib, "XGetWindowProperty")
		purego.RegisterLibFunc(&x11.free, lib, "XFree")
		purego.RegisterLibFunc(&x11.getGeometry, lib, "XGetGeometry")
		purego.RegisterLibFunc(&x11.translate, lib, "XTranslateCoordinates")
		purego.RegisterLibFunc(&x11.setErrorHandler, lib, "XSetErrorHandler")
		// Windows can disappear between calls; the default handler would exit
		// the process on the resulting BadWindow error.
		x11.ignoreXErrorFunc = purego.NewCallback(func(display, event uintptr) uintptr { return 0 })
		x11.setErrorHandler(x11.ignoreXErrorFunc)
	})
	return x11.err
}

// xSession is one display connection with interned EWMH atoms.
type xSession struct {
	display uintptr
	root    uint
	atoms   map[string]uint
}

func openX11() (*xSession, bool) {
	if loadX11() != nil {
		return nil, false
	}
	display := x11.openDisplay(0)
	if display == 0 {
		return nil, false
	}
	session := &xSession{display: display, root: x11.defaultRoot(display), atoms: map[string]uint{}}
	for _, name := range []string{
		"_NET_ACTIVE_WINDOW", "_NET_CLIENT_LIST_STACKING", "_NET_WM_PID", "_NET_WM_STATE",
		"_NET_WM_STATE_HIDDEN", "_NET_WM_STATE_FULLSCREEN",
		"_NET_WM_STATE_MAXIMIZED_VERT", "_NET_WM_STATE_MAXIMIZED_HORZ",
	} {
		session.atoms[name] = x11.internAtom(display, name, 0)
	}
	return session, true
}

func (s *xSession) close() { x11.closeDisplay(s.display) }

// longs reads a format-32 property, which Xlib returns as C longs.
func (s *xSession) longs(window uint, property string) []uint {
	var actualType, items, bytesAfter uint
	var format int32
	var data unsafe.Pointer
	if x11.getProperty(s.display, window, s.atoms[property], 0, 1<<16, 0, anyPropertyType,
		&actualType, &format, &items, &bytesAfter, &data) != 0 || data == nil {
		return nil
	}
	defer x11.free(data)
	if format != 32 || items == 0 {
		return nil
	}
	return slices.Clone(unsafe.Slice((*uint)(data), items))
}

func (s *xSession) pid(window uint) int {
	if values := s.longs(window, "_NET_WM_PID"); len(values) > 0 {
		return int(values[0])
	}
	return 0
}

func (s *xSession) states(window uint) []uint { return s.longs(window, "_NET_WM_STATE") }

func (s *xSession) bounds(window uint) (Rect, bool) {
	var root, child uint
	var x, y int32
	var width, height, border, depth uint32
	if x11.getGeometry(s.display, window, &root, &x, &y, &width, &height, &border, &depth) == 0 {
		return Rect{}, false
	}
	x11.translate(s.display, window, root, 0, 0, &x, &y, &child)
	return Rect{X: int(x), Y: int(y), Width: int(width), Height: int(height)}, true
}

func (s *xSession) hidden(window uint) bool {
	return slices.Contains(s.states(window), s.atoms["_NET_WM_STATE_HIDDEN"])
}

func (s *xSession) large(window uint) bool {
	bounds, ok := s.bounds(window)
	return ok && bounds.Width >= minimumWindowSide && bounds.Height >= minimumWindowSide
}

var displayOnce = sync.OnceValue(func() bool {
	x11.mu.Lock()
	defer x11.mu.Unlock()
	session, ok := openX11()
	if ok {
		session.close()
	}
	return ok
})

// windowsSupported checks the display once; it does not change during a
// session.
func windowsSupported() bool { return displayOnce() }

func inspectWindows(pids, ignore []int) (windowReport, error) {
	x11.mu.Lock()
	defer x11.mu.Unlock()
	session, ok := openX11()
	if !ok {
		return windowReport{}, nil
	}
	defer session.close()
	report := windowReport{known: true}

	// Managed windows from the top of the stack down.
	stacking := session.longs(session.root, "_NET_CLIENT_LIST_STACKING")
	slices.Reverse(stacking)

	var active uint
	if values := session.longs(session.root, "_NET_ACTIVE_WINDOW"); len(values) > 0 {
		active = values[0]
	}
	if active != 0 && slices.Contains(ignore, session.pid(active)) {
		// An ignored process, such as the overlay, is active: the topmost
		// shown window of another process decides.
		active = 0
		for _, window := range stacking {
			if !slices.Contains(ignore, session.pid(window)) && !session.hidden(window) && session.large(window) {
				active = window
				break
			}
		}
	}

	var game uint
	if active != 0 && slices.Contains(pids, session.pid(active)) {
		game = active
		report.focused = true
	} else {
		for _, window := range stacking {
			if slices.Contains(pids, session.pid(window)) && session.large(window) {
				game = window
				break
			}
		}
	}
	if game == 0 {
		return report, nil
	}

	report.pid = session.pid(game)
	bounds, _ := session.bounds(game)
	states := session.states(game)
	has := func(name string) bool { return slices.Contains(states, session.atoms[name]) }
	window := &Window{
		Bounds:     bounds,
		Minimized:  has("_NET_WM_STATE_HIDDEN"),
		Maximized:  has("_NET_WM_STATE_MAXIMIZED_VERT") && has("_NET_WM_STATE_MAXIMIZED_HORZ"),
		Fullscreen: has("_NET_WM_STATE_FULLSCREEN"),
	}
	window.Visible = !window.Minimized
	report.window = window
	return report, nil
}
