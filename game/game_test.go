package game

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMatcherIgnoresCaseExtensionAndDirectory(t *testing.T) {
	names := newMatcher(DefaultProcessNames)
	for _, name := range []string{
		"Tanks Blitz", "tanksblitz.exe", "TanksBlitz.EXE", "wotblitz.exe",
		`C:\Games\Tanks_Blitz\tanksblitz.exe`, "/Applications/Tanks Blitz.app/Contents/MacOS/Tanks Blitz",
	} {
		if !names.match(name) {
			t.Errorf("%q should match", name)
		}
	}
	for _, name := range []string{"", "tanks", "tanksblitz-launcher.exe", "Tanks Blitz Helper"} {
		if names.match(name) {
			t.Errorf("%q should not match", name)
		}
	}
}

func TestMatcherAcceptsTruncatedKernelNames(t *testing.T) {
	names := newMatcher(DefaultProcessNames)
	if !names.matchTruncated("World of Tanks ", 15) {
		t.Error("a name cut at the kernel limit should match its prefix")
	}
	if names.matchTruncated("World of", 15) {
		t.Error("a short name is complete and must match exactly")
	}
}

func TestCoversDisplay(t *testing.T) {
	display := Rect{X: 1440, Y: 0, Width: 1920, Height: 1080}
	if !coversDisplay(Rect{X: 1440, Y: 0, Width: 1920, Height: 1080}, display) {
		t.Error("an identical rectangle covers the display")
	}
	if !coversDisplay(Rect{X: 1432, Y: -8, Width: 1936, Height: 1096}, display) {
		t.Error("a window with borders outside the display covers it")
	}
	if coversDisplay(Rect{X: 1440, Y: 25, Width: 1920, Height: 1055}, display) {
		t.Error("a window below the menu bar does not cover the display")
	}
}

// The test binary stands in for the client: it is running but has no window,
// and its own windows never count as focused.
func TestDetectFindsProcessByName(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	state, err := Detect(Options{ProcessNames: []string{filepath.Base(executable)}})
	if err != nil {
		t.Fatal(err)
	}
	if !state.Running || state.PID == 0 {
		t.Fatalf("test process not found: %+v", state)
	}
	if state.Focused || state.InGame {
		t.Fatalf("the current process is ignored for focus: %+v", state)
	}

	state, err = Detect(Options{ProcessNames: []string{"no-such-blitz-client-process"}})
	if err != nil {
		t.Fatal(err)
	}
	if state.Running || state.Window != nil || state.InGame {
		t.Fatalf("unexpected state: %+v", state)
	}
}

func TestWatchEmitsOnlyChanges(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	calls := 0
	Watch(ctx, 10*time.Millisecond, Options{ProcessNames: []string{"no-such-blitz-client-process"}}, func(State, error) {
		calls++
	})
	if calls != 1 {
		t.Fatalf("an unchanged state is emitted once, got %d", calls)
	}
}
