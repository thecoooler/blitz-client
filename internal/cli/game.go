package cli

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"time"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/game"
	"github.com/thecoooler/blitz-client/internal/jsonx"
)

func (c GameStatusCmd) Run(a *app) error {
	if c.PollMS == 0 {
		return usageError("--poll-ms requires milliseconds between 1 and 86400000")
	}
	options := game.Options{ProcessNames: c.ProcessNames}
	pollMS := uint64(c.PollMS)

	if !c.Watch {
		state, err := game.Detect(options)
		if err != nil {
			return blitz.NewError("GAME_DETECT_FAILED", err.Error())
		}
		if a.json {
			a.printJSON(state)
		} else {
			a.printGameState(state)
		}
		return nil
	}

	// --watch prints one NDJSON state per change until interrupted.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithCancelCause(ctx)
	game.Watch(ctx, time.Duration(pollMS)*time.Millisecond, options, func(state game.State, err error) {
		if err != nil {
			cancel(blitz.NewError("GAME_DETECT_FAILED", err.Error()))
			return
		}
		line, err := jsonx.Marshal(state)
		if err != nil {
			cancel(blitz.NewError("JSON_SERIALIZE_FAILED", err.Error()))
			return
		}
		if _, err := a.stdout.Write(append(line, '\n')); err != nil {
			cancel(blitz.NewError("GAME_OUTPUT_FAILED", blitz.IOMessage(err)))
		}
	})
	if err := context.Cause(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (a *app) printGameState(state game.State) {
	yesNo := func(value bool) string {
		if value {
			return "yes"
		}
		return "no"
	}
	if !state.Running {
		a.println("running: no")
		return
	}
	a.printf("running: yes (pid %d", state.PID)
	if state.Executable != "" {
		a.printf(", %s", state.Executable)
	}
	a.println(")")
	if window := state.Window; window != nil {
		flags := []string{}
		for _, flag := range []struct {
			set  bool
			name string
		}{
			{window.Visible, "visible"}, {window.Minimized, "minimized"},
			{window.Maximized, "maximized"}, {window.Fullscreen, "fullscreen"},
			{window.CursorHidden != nil && *window.CursorHidden, "cursor hidden"},
			{window.CursorCaptured != nil && *window.CursorCaptured, "cursor captured"},
		} {
			if flag.set {
				flags = append(flags, flag.name)
			}
		}
		bounds := window.Bounds
		a.printf("window: %dx%d at %d,%d", bounds.Width, bounds.Height, bounds.X, bounds.Y)
		if len(flags) > 0 {
			a.printf(" (%s)", strings.Join(flags, ", "))
		}
		a.println()
	} else {
		a.println("window: none")
	}
	if !state.FocusKnown {
		a.println("focused: unknown (window state is unavailable in this session)")
	} else {
		a.println("focused: " + yesNo(state.Focused))
	}
	a.println("in game: " + yesNo(state.InGame))
}
