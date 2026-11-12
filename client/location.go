// Package client discovers the installed Tanks Blitz client and reads its
// catalogs. It does not export models or install mods.
package client

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/pathx"
)

// ResolvedGameDir describes the selected client installation.
type ResolvedGameDir struct {
	// GameDir is the install directory selected by the user or the default-path search.
	GameDir string `json:"game_dir"`
	// ResourceDir is the resource root used to resolve mod paths.
	ResourceDir string `json:"resource_dir"`
	// CacheDir is the standard downloaded-resource cache (packs), independent
	// of the install path. The directory need not exist yet.
	CacheDir *string `json:"cache_dir"`
	Source   string  `json:"source"`
	// ReplayDir is the standard replay folder for this OS; it need not exist yet.
	ReplayDir *string `json:"replay_dir"`
}

// DefaultCacheDir re-exports blitzclient.DefaultCacheDir.
func DefaultCacheDir() (string, error) { return blitz.DefaultCacheDir() }

// DefaultReplayDir re-exports blitzclient.DefaultReplayDir.
func DefaultReplayDir() (string, error) { return blitz.DefaultReplayDir() }

func optionalPath(path string, err error) *string {
	if err != nil {
		return nil
	}
	return &path
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func canonicalize(path string) (string, error) { return pathx.Canonicalize(path) }

// ResolveGameDir uses an explicitly supplied directory, or searches the
// ordered platform defaults when explicit is empty. An invalid explicit path is
// an error and never falls back to auto-detection.
func ResolveGameDir(explicit string) (*ResolvedGameDir, error) {
	if explicit == "" {
		return DiscoverGameDir()
	}
	if !isDir(explicit) {
		return nil, blitz.ErrorAt("GAME_DIR_NOT_FOUND", "the explicitly supplied game directory does not exist", explicit)
	}
	canonical, err := canonicalize(explicit)
	if err != nil {
		return nil, blitz.IOError("could not resolve game directory", explicit, err)
	}
	return resolved(canonical, "explicit"), nil
}

// ResolveGameInstallation resolves an explicit or detected directory and
// requires a recognized resource root. Mutating operations use it so an
// arbitrary writable directory cannot be treated as the game installation.
func ResolveGameInstallation(explicit string) (*ResolvedGameDir, error) {
	resolvedDir, err := ResolveGameDir(explicit)
	if err != nil {
		return nil, err
	}
	if resolvedDir.ResourceDir == resolvedDir.GameDir {
		return nil, blitz.ErrorAt("GAME_DIR_INVALID", "the directory does not contain a recognized game resource root", resolvedDir.GameDir)
	}
	return resolvedDir, nil
}

func resolved(canonical, source string) *ResolvedGameDir {
	gameDir, resourceDir := normalizeExplicitDir(canonical)
	return &ResolvedGameDir{
		GameDir:     gameDir,
		ResourceDir: resourceDir,
		CacheDir:    optionalPath(blitz.DefaultCacheDir()),
		Source:      source,
		ReplayDir:   optionalPath(blitz.DefaultReplayDir()),
	}
}

// DiscoverGameDir returns the first existing candidate in the
// platform-specific search order.
func DiscoverGameDir() (*ResolvedGameDir, error) {
	candidates := defaultCandidates()
	for _, candidate := range candidates {
		if isDir(candidate) {
			canonical, err := canonicalize(candidate)
			if err != nil {
				canonical = candidate
			}
			return resolved(canonical, "default"), nil
		}
	}
	return nil, blitz.NewError("GAME_DIR_NOT_FOUND",
		"no default Tanks Blitz directory exists; checked: "+strings.Join(candidates, ", ")+"; pass --game-dir")
}

func findResourceRoot(gameDir string) string {
	for _, candidate := range []string{
		filepath.Join(gameDir, "Data"),
		filepath.Join(gameDir, "Contents", "Resources", "Data"),
		filepath.Join(gameDir, "World of Tanks Blitz.app", "Contents", "Resources", "Data"),
	} {
		if isDir(candidate) {
			return candidate
		}
	}
	return gameDir
}

func normalizeExplicitDir(path string) (string, string) {
	if strings.EqualFold(filepath.Base(path), "Data") {
		parent := filepath.Dir(path)
		if strings.EqualFold(filepath.Base(parent), "Resources") {
			contents := filepath.Dir(parent)
			if appRoot := filepath.Dir(contents); contents != parent && appRoot != contents {
				return appRoot, path
			}
		} else if parent != path {
			return parent, path
		}
	}
	return path, findResourceRoot(path)
}

func defaultCandidates() []string {
	var out []string
	switch runtime.GOOS {
	case "windows":
		out = windowsCandidates()
	case "darwin":
		out = macosCandidates()
	case "linux":
		out = linuxCandidates()
	}
	var unique []string
	for _, path := range out {
		if !slices.Contains(unique, path) {
			unique = append(unique, path)
		}
	}
	return unique
}

func steamInstall(root string) string {
	return filepath.Join(root, "steamapps", "common", "World of Tanks Blitz")
}

func windowsCandidates() []string {
	out := []string{`C:\Games\Tanks_Blitz`, `C:\Games\Tanks_Blitz\Data`}
	var steamRoots []string
	for _, key := range []string{"ProgramFiles(x86)", "ProgramFiles"} {
		if value, ok := os.LookupEnv(key); ok {
			steamRoots = append(steamRoots, filepath.Join(value, "Steam"))
		}
	}
	if local, ok := os.LookupEnv("LOCALAPPDATA"); ok {
		steamRoots = append(steamRoots, filepath.Join(local, "Programs", "Steam"))
	}
	steamRoots = append(steamRoots, `C:\Steam`, `D:\Steam`, `C:\Games\Steam`)
	for _, root := range steamRoots {
		out = append(out, steamInstall(root))
	}

	if programFiles, ok := os.LookupEnv("ProgramFiles"); ok {
		store := filepath.Join(programFiles, "WindowsApps")
		if entries, err := os.ReadDir(store); err == nil {
			sort.SliceStable(entries, func(i, j int) bool {
				return asciiLower(entries[i].Name()) > asciiLower(entries[j].Name())
			})
			for _, entry := range entries {
				if strings.HasPrefix(asciiLower(entry.Name()), "7458be2c.worldoftanksblitz_") {
					path := filepath.Join(store, entry.Name())
					out = append(out, filepath.Join(path, "Data"), path)
				}
			}
		}
	}

	if profile, ok := os.LookupEnv("USERPROFILE"); ok {
		for _, relative := range []string{
			`Games\World of Tanks Blitz\Data`,
			`Games\World of Tanks Blitz`,
			`XboxGames\World of Tanks Blitz\Content\Data`,
		} {
			out = append(out, filepath.Join(profile, relative))
		}
	}
	return out
}

func macosCandidates() []string {
	out := []string{
		"/Applications/Tanks Blitz.app/Contents/Resources/Data",
		"/Applications/Tanks Blitz.app",
	}
	if home, ok := os.LookupEnv("HOME"); ok {
		common := steamInstall(filepath.Join(home, "Library", "Application Support", "Steam"))
		out = append(out, common, filepath.Join(common, "World of Tanks Blitz.app"))
	}
	return out
}

func linuxCandidates() []string {
	var out []string
	if home, ok := os.LookupEnv("HOME"); ok {
		for _, steam := range []string{
			".local/share/Steam",
			".steam/steam",
			".var/app/com.valvesoftware.Steam/.local/share/Steam",
			"snap/steam/common/.local/share/Steam",
		} {
			out = append(out, steamInstall(filepath.Join(home, steam)))
		}
	}
	for _, steam := range []string{"/usr/share/steam", "/usr/lib/steam"} {
		out = append(out, steamInstall(steam))
	}
	return out
}

func asciiLower(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 'a' - 'A'
		}
		return r
	}, value)
}
