// Package cli implements the blitz-client command line. Commands, options,
// output, and exit codes match the original tool: 0 on success, 4 when a
// validation fails, and 2 for errors.
package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/spf13/cobra"
	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
)

const usage = `blitz-client commands:

blitz-client client locate [--game-dir <path>] [--json]
blitz-client client files unpack --output-dir <path> [--game-dir <path>] [--cache-dir <path> | --no-cache] [--types <all|configs,images,models,sounds,fonts,shaders,other>] [--exclude <pattern>] [--dry-run] [--json]
blitz-client client tanks list [--game-dir <path>] [--json]
blitz-client client export --output-dir <path> [--game-dir <path>] [--include <all|data,icons,models>]... [--cache-dir <path> | --no-cache] [--server-config <path>] [model options]
blitz-client mod list [--game-dir <path>] [--json]
blitz-client mod validate <mods-dir|mod-dir|mod.yaml|mod.yml> [--game-dir <path>] [--json]
blitz-client mod install <mod-dir|mod.yaml|mod.yml> [--game-dir <path>] [--json]
blitz-client mod uninstall <mod-id> [--game-dir <path>] [--json]
blitz-client mod update-from-git <mod-dir|mod.yaml|mod.yml> --repo <path> [--base <rev>] [--json]
blitz-client replay inspect <replay.tbreplay|data.replay> [--data] [--out <path>] [--json]
blitz-client replay latest [--dir <path>] [--data] [--out <path>] [--json]
blitz-client replay follow <data.replay> [--poll-ms <N>] [--idle-timeout-ms <N>] [--out <new-path>]
blitz-client game status [--process-name <name>]... [--watch] [--poll-ms <N>] [--json]

--game-dir is optional. Without it the first existing default path for this OS is used.
replay latest uses the newest .tbreplay of the platform replay folder; --dir overrides it.`

// app carries the output streams and the global --json flag.
type app struct {
	stdout, stderr io.Writer
	json           bool
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

// Run executes the command line and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || slices.Contains(args, "--help") || slices.Contains(args, "-h") {
		fmt.Fprintln(stdout, usage)
		return 0
	}
	a := &app{stdout: stdout, stderr: stderr, json: slices.Contains(args, "--json")}
	args = slices.DeleteFunc(slices.Clone(args), func(arg string) bool { return arg == "--json" })

	valid, err := a.execute(args)
	switch {
	case err == nil && valid:
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

// execute dispatches through the cobra command tree. Leaf commands parse
// their own arguments so options and diagnostics match the original tool.
func (a *app) execute(args []string) (bool, error) {
	var valid bool
	var runErr error
	command := func(name string, run func([]string) (bool, error)) *cobra.Command {
		return &cobra.Command{
			Use:                name,
			DisableFlagParsing: true,
			Args:               cobra.ArbitraryArgs,
			RunE: func(_ *cobra.Command, rest []string) error {
				valid, runErr = run(rest)
				return nil
			},
		}
	}
	root := &cobra.Command{Use: "blitz-client", SilenceErrors: true, SilenceUsage: true}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.AddCommand(command("client", a.runClient), command("mod", a.runMod), command("replay", a.runReplay), command("game", a.runGame))
	if len(args) == 0 {
		return false, usageError("unknown command: ")
	}
	if found, _, err := root.Find(args); err != nil || found == root {
		return false, usageError("unknown command: " + args[0])
	}
	root.SetArgs(args)
	root.SetOut(a.stdout)
	root.SetErr(a.stderr)
	if err := root.Execute(); err != nil {
		return false, usageError(err.Error())
	}
	return valid, runErr
}

// takeGameDir separates positional arguments from --game-dir.
func takeGameDir(args []string) ([]string, string, error) {
	var positional []string
	gameDir := ""
	for index := 0; index < len(args); index++ {
		switch arg := args[index]; {
		case arg == "--game-dir":
			index++
			if index >= len(args) {
				return nil, "", usageError("--game-dir requires a path")
			}
			value, err := gameDirOption(args[index])
			if err != nil {
				return nil, "", err
			}
			gameDir = value
		case len(arg) > 0 && arg[0] == '-':
			return nil, "", usageError("unknown option: " + arg)
		default:
			positional = append(positional, arg)
		}
	}
	return positional, gameDir, nil
}

// gameDirOption rejects an explicitly empty --game-dir, which the library
// would otherwise treat as "search the default locations".
func gameDirOption(value string) (string, error) {
	if value == "" {
		return "", blitz.NewError("GAME_DIR_NOT_FOUND", "the explicitly supplied game directory does not exist")
	}
	return value, nil
}

// optionValue returns the argument after a flag.
func optionValue(args []string, index *int, message string) (string, error) {
	*index++
	if *index >= len(args) {
		return "", usageError(message)
	}
	return args[*index], nil
}

// Main runs the command line with the process streams.
func Main() {
	// Report writes to a closed pipe as errors instead of dying silently.
	signal.Ignore(syscall.SIGPIPE)
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
