package game

import (
	"slices"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	gwOwner               = 4
	gaRootOwner           = 3
	gwHwndNext            = 2
	monitorDefaultNearest = 2
	cursorShowing         = 1
	dwmwaCloaked          = 14
	smXVirtualScreen      = 76
	smYVirtualScreen      = 77
	smCXVirtualScreen     = 78
	smCYVirtualScreen     = 79
	minimumWindowSide     = 64 // smaller top-level windows are helpers, not the game
)

var (
	user32              = windows.NewLazySystemDLL("user32.dll")
	procGetWindow       = user32.NewProc("GetWindow")
	procGetAncestor     = user32.NewProc("GetAncestor")
	procIsIconic        = user32.NewProc("IsIconic")
	procIsZoomed        = user32.NewProc("IsZoomed")
	procGetWindowRect   = user32.NewProc("GetWindowRect")
	procMonitorFromWnd  = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo  = user32.NewProc("GetMonitorInfoW")
	procGetCursorInfo   = user32.NewProc("GetCursorInfo")
	procGetClipCursor   = user32.NewProc("GetClipCursor")
	procGetSystemMetric = user32.NewProc("GetSystemMetrics")

	enumOnce     sync.Once
	enumCallback uintptr
	enumMu       sync.Mutex
	enumTarget   []windows.HWND
)

type monitorInfo struct {
	Size    uint32
	Monitor windows.Rect
	Work    windows.Rect
	Flags   uint32
}

type cursorInfo struct {
	Size   uint32
	Flags  uint32
	Cursor windows.Handle
	X, Y   int32
}

func windowsSupported() bool { return true }

func windowPID(hwnd windows.HWND) int {
	var pid uint32
	windows.GetWindowThreadProcessId(hwnd, &pid)
	return int(pid)
}

func call(proc *windows.LazyProc, args ...uintptr) uintptr {
	result, _, _ := proc.Call(args...)
	return result
}

func windowRect(hwnd windows.HWND) windows.Rect {
	var rect windows.Rect
	call(procGetWindowRect, uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	return rect
}

func toRect(rect windows.Rect) Rect {
	return Rect{X: int(rect.Left), Y: int(rect.Top), Width: int(rect.Right - rect.Left), Height: int(rect.Bottom - rect.Top)}
}

// topLevelWindows lists visible unowned top-level windows from the top of
// the z-order down.
func topLevelWindows() []windows.HWND {
	enumOnce.Do(func() {
		enumCallback = windows.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
			if windows.IsWindowVisible(hwnd) && call(procGetWindow, uintptr(hwnd), gwOwner) == 0 {
				enumTarget = append(enumTarget, hwnd)
			}
			return 1
		})
	})
	enumMu.Lock()
	defer enumMu.Unlock()
	enumTarget = nil
	windows.EnumWindows(enumCallback, nil)
	return enumTarget
}

func cloaked(hwnd windows.HWND) bool {
	var value uint32
	err := windows.DwmGetWindowAttribute(hwnd, dwmwaCloaked, unsafe.Pointer(&value), uint32(unsafe.Sizeof(value)))
	return err == nil && value != 0
}

func large(hwnd windows.HWND) bool {
	if call(procIsIconic, uintptr(hwnd)) != 0 {
		return true // a minimized window reports a tiny placeholder rectangle
	}
	rect := toRect(windowRect(hwnd))
	return rect.Width >= minimumWindowSide && rect.Height >= minimumWindowSide
}

func inspectWindows(pids, ignore []int) (windowReport, error) {
	report := windowReport{known: true}
	foreground := windows.GetForegroundWindow()
	if foreground != 0 && slices.Contains(ignore, windowPID(foreground)) {
		// An ignored process, such as the overlay, is active: the next visible
		// window of another process below it decides.
		next := foreground
		for {
			next = windows.HWND(call(procGetWindow, uintptr(next), gwHwndNext))
			if next == 0 {
				break
			}
			if windows.IsWindowVisible(next) && call(procIsIconic, uintptr(next)) == 0 && !cloaked(next) &&
				!slices.Contains(ignore, windowPID(next)) && large(next) {
				break
			}
		}
		foreground = next
	}

	var game windows.HWND
	if foreground != 0 && slices.Contains(pids, windowPID(foreground)) {
		// A focused dialog of the game reports its main window.
		game = foreground
		if root := windows.HWND(call(procGetAncestor, uintptr(foreground), gaRootOwner)); root != 0 {
			game = root
		}
		report.focused = true
	} else {
		for _, hwnd := range topLevelWindows() {
			if slices.Contains(pids, windowPID(hwnd)) && large(hwnd) {
				game = hwnd
				break
			}
		}
	}
	if game == 0 {
		return report, nil
	}

	report.pid = windowPID(game)
	window := &Window{
		Bounds:    toRect(windowRect(game)),
		Minimized: call(procIsIconic, uintptr(game)) != 0,
		Maximized: call(procIsZoomed, uintptr(game)) != 0,
	}
	window.Visible = windows.IsWindowVisible(game) && !window.Minimized && !cloaked(game)
	if monitor := call(procMonitorFromWnd, uintptr(game), monitorDefaultNearest); monitor != 0 {
		info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
		if call(procGetMonitorInfo, monitor, uintptr(unsafe.Pointer(&info))) != 0 {
			window.Fullscreen = !window.Minimized && coversDisplay(window.Bounds, toRect(info.Monitor))
		}
	}
	cursor := cursorInfo{Size: uint32(unsafe.Sizeof(cursorInfo{}))}
	if call(procGetCursorInfo, uintptr(unsafe.Pointer(&cursor))) != 0 {
		window.CursorHidden = boolPtr(cursor.Flags&cursorShowing == 0)
	}
	var clip windows.Rect
	if call(procGetClipCursor, uintptr(unsafe.Pointer(&clip))) != 0 {
		desktop := Rect{
			X:      int(int32(call(procGetSystemMetric, smXVirtualScreen))),
			Y:      int(int32(call(procGetSystemMetric, smYVirtualScreen))),
			Width:  int(int32(call(procGetSystemMetric, smCXVirtualScreen))),
			Height: int(int32(call(procGetSystemMetric, smCYVirtualScreen))),
		}
		window.CursorCaptured = boolPtr(!coversDisplay(toRect(clip), desktop))
	}
	report.window = window
	return report, nil
}
