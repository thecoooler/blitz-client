package cli

import (
	"slices"
	"strings"

	"github.com/alecthomas/kong"
	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/export"
	"github.com/thecoooler/blitz-client/internal/rs"
)

// CLI is the kong grammar of the blitz-client command line.
type CLI struct {
	// JSON is a root flag on purpose: kong accepts ancestor flags after the
	// subcommand, so both "--json client tanks list" and
	// "client tanks list --json" work.
	JSON   bool      `help:"Print output as JSON"`
	Client ClientCmd `cmd:"" help:"Inspect the installed game client"`
	Mod    ModCmd    `cmd:"" help:"Install, validate and update resource mods"`
	Replay ReplayCmd `cmd:"" help:"Inspect and follow replays"`
	Game   GameCmd   `cmd:"" help:"Detect the running game"`
}

type ClientCmd struct {
	Locate LocateCmd       `cmd:"" help:"Locate the game and resource directories"`
	Files  ClientFilesCmd  `cmd:"" help:"Unpack client files"`
	Tanks  ClientTanksCmd  `cmd:"" help:"List tanks in the client"`
	Export ClientExportCmd `cmd:"" help:"Export tank models and snapshot data"`
}

type ClientFilesCmd struct {
	Unpack FilesUnpackCmd `cmd:"" help:"Unpack client files to a directory"`
}

type ClientTanksCmd struct {
	List TanksListCmd `cmd:"" help:"List tanks available in the client"`
}

type LocateCmd struct {
	GameDir *string `placeholder:"path" help:"Game directory (default: the first existing default path for this OS)"`
}

type FilesUnpackCmd struct {
	OutputDir string    `required:"" placeholder:"path" help:"Directory to write the unpacked files to"`
	GameDir   *string   `placeholder:"path" help:"Game directory (default: the first existing default path for this OS)"`
	CacheDir  *string   `placeholder:"path" help:"Cache directory (mutually exclusive with --no-cache)"`
	NoCache   bool      `help:"Do not read or write the cache"`
	Types     fileTypes `placeholder:"types" help:"Comma-separated file types to unpack: all|configs,images,models,sounds,fonts,shaders,other"`
	Exclude   []string  `placeholder:"pattern" help:"Exclude files matching a path pattern (repeatable)"`
	DryRun    bool      `help:"List what would be unpacked without writing anything"`
}

func (c FilesUnpackCmd) Validate() error {
	if c.NoCache && c.CacheDir != nil {
		return usageError("--cache-dir and --no-cache cannot be combined")
	}
	return checkGameDirFlag(c.GameDir)
}

type TanksListCmd struct {
	GameDir *string `placeholder:"path" help:"Game directory (default: the first existing default path for this OS)"`
	Nation  string  `placeholder:"nation" help:"Only list tanks of this nation"`
}

func (c TanksListCmd) Validate() error { return checkGameDirFlag(c.GameDir) }

type ClientExportCmd struct {
	OutputDir      string      `required:"" placeholder:"path" help:"Directory to write the export to"`
	GameDir        *string     `placeholder:"path" help:"Game directory (default: the first existing default path for this OS)"`
	CacheDir       *string     `placeholder:"path" help:"Cache directory"`
	NoCache        bool        `help:"Do not read or write the cache"`
	Include        []string    `enum:"all,data,icons,models" placeholder:"section" help:"Snapshot sections to export (repeatable)"`
	ServerConfig   string      `placeholder:"path" help:"Server config used to restore crew skill data"`
	MaxTextureSize textureSize `placeholder:"original|N|WxH" help:"Model texture size"`
	TextureFormat  *string     `enum:"png,webp" placeholder:"format" help:"Model texture format"`
	WebPQuality    *uint8      `placeholder:"0-100" help:"WebP texture quality (0 to 100)"`
	NoCompress     bool        `help:"Do not compress exported textures"`
	NoPBR          bool        `help:"Exclude PBR textures"`
}

func (c ClientExportCmd) Validate() error { return checkGameDirFlag(c.GameDir) }

type ModCmd struct {
	List          ModListCmd          `cmd:"" help:"List installed mods"`
	Validate      ModValidateCmd      `cmd:"" help:"Validate a mods directory or a single mod"`
	Install       ModInstallCmd       `cmd:"" help:"Install a mod"`
	Uninstall     ModUninstallCmd     `cmd:"" help:"Remove an installed mod"`
	UpdateFromGit ModUpdateFromGitCmd `cmd:"" help:"Rebuild a mod from a git diff"`
}

type ModListCmd struct {
	GameDir *string `placeholder:"path" help:"Game directory (default: the first existing default path for this OS)"`
}

func (c ModListCmd) Validate() error { return checkGameDirFlag(c.GameDir) }

type ModValidateCmd struct {
	Path    string  `arg:"" help:"Mods directory, mod directory or mod.yaml/mod.yml path"`
	GameDir *string `placeholder:"path" help:"Game directory (default: the first existing default path for this OS)"`
}

func (c ModValidateCmd) Validate() error { return checkGameDirFlag(c.GameDir) }

type ModInstallCmd struct {
	Path    string  `arg:"" help:"Mod directory or mod.yaml/mod.yml path"`
	GameDir *string `placeholder:"path" help:"Game directory (default: the first existing default path for this OS)"`
}

func (c ModInstallCmd) Validate() error { return checkGameDirFlag(c.GameDir) }

type ModUninstallCmd struct {
	ModID   string  `arg:"" help:"Installed mod id"`
	GameDir *string `placeholder:"path" help:"Game directory (default: the first existing default path for this OS)"`
}

func (c ModUninstallCmd) Validate() error { return checkGameDirFlag(c.GameDir) }

type ModUpdateFromGitCmd struct {
	Path    string  `arg:"" help:"Mod directory or mod.yaml/mod.yml path"`
	Repo    string  `required:"" placeholder:"path" help:"Git repository containing the upstream mod"`
	Base    *string `placeholder:"rev" help:"Base revision for the diff (default: the current HEAD)"`
	GameDir *string `placeholder:"path" help:"Game directory (default: the first existing default path for this OS)"`
}

func (c ModUpdateFromGitCmd) Validate() error { return checkGameDirFlag(c.GameDir) }

type ReplayCmd struct {
	Inspect ReplayInspectCmd `cmd:"" help:"Analyze a replay file"`
	Latest  ReplayLatestCmd  `cmd:"" help:"Analyze the newest replay of the platform replay folder"`
	Follow  ReplayFollowCmd  `cmd:"" help:"Stream events from a live data.replay as NDJSON"`
}

// analysisFlags is shared by replay inspect and replay latest.
type analysisFlags struct {
	Data bool    `help:"Include decoded packet data"`
	Out  *string `placeholder:"path" help:"Write the JSON analysis to this file"`
}

type ReplayInspectCmd struct {
	analysisFlags `embed:""`
	Path          string `arg:"" help:".tbreplay or data.replay path"`
}

type ReplayLatestCmd struct {
	analysisFlags `embed:""`
	Dir           *string `placeholder:"path" help:"Replay folder (default: the platform replay folder)"`
}

type ReplayFollowCmd struct {
	Path          string       `arg:"" help:"Live data.replay path"`
	Out           *string      `placeholder:"path" help:"Write NDJSON to this new file instead of stdout"`
	PollMS        milliseconds `placeholder:"ms" default:"25" help:"Polling interval in milliseconds"`
	IdleTimeoutMS milliseconds `placeholder:"ms" default:"0" help:"Exit after this many milliseconds without new events (0: never)"`
}

type GameCmd struct {
	Status GameStatusCmd `cmd:"" help:"Show whether the game is running"`
}

type GameStatusCmd struct {
	ProcessNames []string     `name:"process-name" placeholder:"name" help:"Process names to look for (repeatable)"`
	Watch        bool         `help:"Stream every state change as NDJSON until interrupted"`
	PollMS       milliseconds `placeholder:"ms" default:"500" help:"Polling interval in milliseconds"`
}

// fileTypes parses --types values: a comma-separated list of client file
// types, validated by export.ParseClientFileType.
type fileTypes []export.ClientFileType

func (f *fileTypes) Decode(ctx *kong.DecodeContext) error {
	var text string
	if err := ctx.Scan.PopValueInto("types", &text); err != nil {
		return err
	}
	types := []export.ClientFileType{}
	for _, name := range strings.Split(text, ",") {
		kind, err := export.ParseClientFileType(name)
		if err != nil {
			return usageError(blitz.AsError(err).Message)
		}
		if !slices.Contains(types, kind) {
			types = append(types, kind)
		}
	}
	*f = types
	return nil
}

// milliseconds is a positive integer parsed as a duration in milliseconds.
// The zero value means "not set"; commands decide what that means.
type milliseconds uint64

func (m *milliseconds) Decode(ctx *kong.DecodeContext) error {
	var text string
	if err := ctx.Scan.PopValueInto("ms", &text); err != nil {
		return err
	}
	value, err := rs.ParseUint(text, 64)
	if err != nil || value > 86_400_000 {
		return usageError("requires milliseconds between 0 and 86400000")
	}
	*m = milliseconds(value)
	return nil
}

// textureSize parses --max-texture-size values: "original", N or WIDTHxHEIGHT.
// The zero value means "not set".
type textureSize struct {
	original      bool
	width, height uint32
}

func (t *textureSize) Decode(ctx *kong.DecodeContext) error {
	var text string
	if err := ctx.Scan.PopValueInto("size", &text); err != nil {
		return err
	}
	size, err := parseTextureSize(text)
	if err != nil {
		return err
	}
	if size == nil {
		t.original = true
		return nil
	}
	t.width, t.height = size.Width, size.Height
	return nil
}

// checkGameDirFlag rejects an explicitly empty --game-dir, which otherwise
// would be treated as "search the default locations".
func checkGameDirFlag(gameDir *string) error {
	if gameDir != nil {
		return checkGameDir(*gameDir)
	}
	return nil
}

func checkGameDir(value string) error {
	if value == "" {
		return blitz.NewError("GAME_DIR_NOT_FOUND", "the explicitly supplied game directory does not exist")
	}
	return nil
}
