package game

import (
	"slices"
	"sync"

	"github.com/ebitengine/purego"
)

// Quartz window lists need no Screen Recording permission as long as window
// titles are not read. The frontmost application comes from the Process
// Manager, which queries the window server on every call and therefore needs
// no run loop, unlike NSWorkspace.

const (
	cgWindowListOptionAll              = 0
	cgWindowListExcludeDesktop         = 1 << 4
	cfNumberSInt64Type                 = 4
	cfNumberFloat64Type                = 6
	cfStringEncodingUTF8               = 0x08000100
	maxDisplays                        = 32
	coreFoundationPath                 = "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation"
	coreGraphicsPath                   = "/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics"
	applicationServicesPath            = "/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices"
	minimumWindowSide          float64 = 64 // smaller layer-0 windows are helpers, not the game
)

type cgRect struct{ X, Y, Width, Height float64 }

type processSerialNumber struct{ high, low uint32 }

var quartz struct {
	once sync.Once
	err  error

	windowList      func(option, relativeTo uint32) uintptr
	arrayCount      func(array uintptr) int
	arrayValue      func(array uintptr, index int) uintptr
	dictionaryValue func(dict, key uintptr) uintptr
	numberValue     func(number uintptr, kind int, out *int64) bool
	floatValue      func(number uintptr, kind int, out *float64) bool
	booleanValue    func(boolean uintptr) bool
	release         func(object uintptr)
	rectFromDict    func(dict uintptr, rect *cgRect) bool
	activeDisplays  func(max uint32, displays *uint32, count *uint32) int32
	displayBounds   func(display uint32) cgRect
	cursorVisible   func() bool
	frontProcess    func(psn *processSerialNumber) int16
	processPID      func(psn *processSerialNumber, pid *int32) int16

	keyPID, keyLayer, keyBounds, keyOnscreen, keyAlpha uintptr
}

func loadQuartz() error {
	quartz.once.Do(func() {
		cf, err := purego.Dlopen(coreFoundationPath, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			quartz.err = err
			return
		}
		cg, err := purego.Dlopen(coreGraphicsPath, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			quartz.err = err
			return
		}
		as, err := purego.Dlopen(applicationServicesPath, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			quartz.err = err
			return
		}
		purego.RegisterLibFunc(&quartz.arrayCount, cf, "CFArrayGetCount")
		purego.RegisterLibFunc(&quartz.arrayValue, cf, "CFArrayGetValueAtIndex")
		purego.RegisterLibFunc(&quartz.dictionaryValue, cf, "CFDictionaryGetValue")
		purego.RegisterLibFunc(&quartz.numberValue, cf, "CFNumberGetValue")
		purego.RegisterLibFunc(&quartz.floatValue, cf, "CFNumberGetValue")
		purego.RegisterLibFunc(&quartz.booleanValue, cf, "CFBooleanGetValue")
		purego.RegisterLibFunc(&quartz.release, cf, "CFRelease")
		purego.RegisterLibFunc(&quartz.windowList, cg, "CGWindowListCopyWindowInfo")
		purego.RegisterLibFunc(&quartz.rectFromDict, cg, "CGRectMakeWithDictionaryRepresentation")
		purego.RegisterLibFunc(&quartz.activeDisplays, cg, "CGGetActiveDisplayList")
		purego.RegisterLibFunc(&quartz.displayBounds, cg, "CGDisplayBounds")
		if _, err := purego.Dlsym(cg, "CGCursorIsVisible"); err == nil {
			purego.RegisterLibFunc(&quartz.cursorVisible, cg, "CGCursorIsVisible")
		}
		purego.RegisterLibFunc(&quartz.frontProcess, as, "GetFrontProcess")
		purego.RegisterLibFunc(&quartz.processPID, as, "GetProcessPID")

		var createString func(allocator uintptr, text string, encoding uint32) uintptr
		purego.RegisterLibFunc(&createString, cf, "CFStringCreateWithCString")
		// The kCGWindow* key constants are CFStrings equal to their names.
		key := func(name string) uintptr { return createString(0, name, cfStringEncodingUTF8) }
		quartz.keyPID = key("kCGWindowOwnerPID")
		quartz.keyLayer = key("kCGWindowLayer")
		quartz.keyBounds = key("kCGWindowBounds")
		quartz.keyOnscreen = key("kCGWindowIsOnscreen")
		quartz.keyAlpha = key("kCGWindowAlpha")
	})
	return quartz.err
}

func windowsSupported() bool { return loadQuartz() == nil }

type quartzWindow struct {
	pid      int
	layer    int64
	bounds   Rect
	onscreen bool
	alpha    float64
}

func (w quartzWindow) normal() bool {
	return w.layer == 0 && float64(w.bounds.Width) >= minimumWindowSide && float64(w.bounds.Height) >= minimumWindowSide
}

// copyWindows lists windows front to back.
func copyWindows() []quartzWindow {
	array := quartz.windowList(cgWindowListOptionAll|cgWindowListExcludeDesktop, 0)
	if array == 0 {
		return nil
	}
	defer quartz.release(array)
	count := quartz.arrayCount(array)
	out := make([]quartzWindow, 0, count)
	for i := 0; i < count; i++ {
		dict := quartz.arrayValue(array, i)
		var window quartzWindow
		var number int64
		if value := quartz.dictionaryValue(dict, quartz.keyPID); value == 0 || !quartz.numberValue(value, cfNumberSInt64Type, &number) {
			continue
		}
		window.pid = int(number)
		if value := quartz.dictionaryValue(dict, quartz.keyLayer); value != 0 && quartz.numberValue(value, cfNumberSInt64Type, &number) {
			window.layer = number
		}
		var rect cgRect
		if value := quartz.dictionaryValue(dict, quartz.keyBounds); value != 0 && quartz.rectFromDict(value, &rect) {
			window.bounds = Rect{X: int(rect.X), Y: int(rect.Y), Width: int(rect.Width), Height: int(rect.Height)}
		}
		if value := quartz.dictionaryValue(dict, quartz.keyOnscreen); value != 0 {
			window.onscreen = quartz.booleanValue(value)
		}
		window.alpha = 1
		if value := quartz.dictionaryValue(dict, quartz.keyAlpha); value != 0 {
			quartz.floatValue(value, cfNumberFloat64Type, &window.alpha)
		}
		out = append(out, window)
	}
	return out
}

func displays() []Rect {
	ids := make([]uint32, maxDisplays)
	var count uint32
	if quartz.activeDisplays(maxDisplays, &ids[0], &count) != 0 {
		return nil
	}
	out := make([]Rect, 0, count)
	for _, id := range ids[:count] {
		bounds := quartz.displayBounds(id)
		out = append(out, Rect{X: int(bounds.X), Y: int(bounds.Y), Width: int(bounds.Width), Height: int(bounds.Height)})
	}
	return out
}

func frontPID() int {
	var psn processSerialNumber
	var pid int32
	if quartz.frontProcess(&psn) != 0 || quartz.processPID(&psn, &pid) != 0 {
		return 0
	}
	return int(pid)
}

func inspectWindows(pids, ignore []int) (windowReport, error) {
	if loadQuartz() != nil {
		return windowReport{}, nil
	}
	windows := copyWindows()

	// The first large normal window of the game, preferring on-screen ones.
	var game *quartzWindow
	for i := range windows {
		window := &windows[i]
		if !slices.Contains(pids, window.pid) || !window.normal() {
			continue
		}
		if game == nil || (window.onscreen && !game.onscreen) {
			game = window
		}
		if game.onscreen {
			break
		}
	}

	report := windowReport{known: true}
	front := frontPID()
	if slices.Contains(ignore, front) {
		// An ignored process, such as the overlay, is active: the topmost
		// visible normal window of another process decides.
		front = 0
		for _, window := range windows {
			if window.onscreen && window.alpha > 0 && window.normal() && !slices.Contains(ignore, window.pid) {
				front = window.pid
				break
			}
		}
	}
	report.focused = slices.Contains(pids, front)
	if game == nil {
		return report, nil
	}

	report.pid = game.pid
	window := &Window{
		Bounds:    game.bounds,
		Visible:   game.onscreen && game.alpha > 0,
		Minimized: !game.onscreen,
	}
	for _, display := range displays() {
		if coversDisplay(window.Bounds, display) {
			window.Fullscreen = true
		}
	}
	window.Maximized = window.Fullscreen
	if quartz.cursorVisible != nil {
		window.CursorHidden = boolPtr(!quartz.cursorVisible())
	}
	report.window = window
	return report, nil
}
