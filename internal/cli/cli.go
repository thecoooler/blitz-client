// Package cli implements the blitz-client command line. Commands, options,
// output, and exit codes match the original tool: 0 on success, 4 when a
// validation fails, and 2 for errors.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/alecthomas/kong"
	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
)

// app carries the output streams, the global --json flag, and the result of
// commands that report success or failure on their own (mod validate).
type app struct {
	stdout, stderr io.Writer
	json           bool
	valid          bool
}

func (a *app) printf(format string, args ...any)  { fmt.Fprintf(a.stdout, format, args...) }
func (a *app) println(args ...any)                { fmt.Fprintln(a.stdout, args...) }
func (a *app) eprintf(format string, args ...any) { fmt.Fprintf(a.stderr, format, args...) }

// printJSON writes pretty JSON followed by a newline.
func (a *app) printJSON(value any) {
	data, err := jsonx.MarshalIndent(value)
	if err != nil {
		a.eprintf("could not serialize JSON output: %v\n", err)
		return
	}
	a.stdout.Write(append(data, '\n'))
}

func usageError(message string) error { return blitz.NewError("USAGE", message) }

// exitRequest is how kong reports that it wants the process to stop (--help,
// completion scripts); the panic is caught by Run.
type exitRequest struct{ code int }

// Run executes the command line and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) (code int) {
	a := &app{stdout: stdout, stderr: stderr, json: slices.Contains(args, "--json"), valid: true}
	defer func() {
		if recovered := recover(); recovered != nil {
			request, ok := recovered.(exitRequest)
			if !ok {
				panic(recovered)
			}
			code = request.code
		}
	}()

	err := a.execute(args)
	switch {
	case err == nil && a.valid:
		return 0
	case err == nil:
		return 4
	}
	typed := blitz.AsError(err)
	follow := len(args) > 1 && args[0] == "replay" && args[1] == "follow"
	follow = follow || len(args) > 1 && args[0] == "game" && slices.Contains(args, "--watch")
	switch {
	case a.json && !follow:
		a.printJSON(jsonx.Object{"ok": false, "error": jsonx.ToValue(typed)})
	case len(args) > 1 && args[0] == "mod" && args[1] == "validate":
		a.printValidationError(typed)
	default:
		fmt.Fprintln(stderr, typed.Error())
	}
	return 2
}

// execute parses the command line through the kong grammar in grammar.go and
// runs the selected command. Parse errors keep the original code of errors
// raised by custom mappers and Validate hooks.
func (a *app) execute(args []string) error {
	cli := &CLI{}
	parser, err := kong.New(cli,
		kong.Name("blitz-client"),
		kong.Description("Inspect the installed Tanks Blitz client, export tank models, install resource mods, and analyze replays."),
		kong.Writers(a.stdout, a.stderr),
		kong.Exit(func(code int) { panic(exitRequest{code}) }),
	)
	if err != nil {
		return usageError(err.Error())
	}
	ctx, err := parser.Parse(args)
	if err != nil {
		var typed *blitz.Error
		if errors.As(err, &typed) {
			return typed
		}
		return usageError(err.Error())
	}
	return ctx.Run(a)
}

// Main runs the command line with the process streams.
func Main() {
	// Report writes to a closed pipe as errors instead of dying silently.
	signal.Ignore(syscall.SIGPIPE)
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
