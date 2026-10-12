package blitzclient

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultCacheDir returns the standard Tanks Blitz resource-cache path without
// checking whether it exists.
func DefaultCacheDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, ok := os.LookupEnv("HOME")
		if !ok {
			return "", NewError("CACHE_DIR_NOT_FOUND", "HOME is not set")
		}
		return filepath.Join(home, "Library/Containers/com.tanksblitz/Data/Library/Caches/packs"), nil
	case "windows":
		profile, ok := os.LookupEnv("USERPROFILE")
		if !ok {
			return "", NewError("CACHE_DIR_NOT_FOUND", "USERPROFILE is not set")
		}
		return filepath.Join(profile, "Documents/TanksBlitz/packs"), nil
	default:
		return "", NewError("CACHE_DIR_NOT_FOUND", "no standard resource-cache path is configured for this OS")
	}
}

// DefaultReplayDir returns the standard replay path for this OS without
// checking whether it exists.
func DefaultReplayDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, ok := os.LookupEnv("HOME")
		if !ok {
			return "", NewError("REPLAY_FOLDER_NOT_FOUND", "HOME is not set; pass --dir")
		}
		return filepath.Join(home, "Library/Containers/com.tanksblitz/Data/Documents/tb-project/replays"), nil
	case "windows":
		profile, ok := os.LookupEnv("USERPROFILE")
		if !ok {
			return "", NewError("REPLAY_FOLDER_NOT_FOUND", "USERPROFILE is not set; pass --dir")
		}
		return filepath.Join(profile, "Documents/TanksBlitz/replays"), nil
	default:
		return "", NewError("REPLAY_FOLDER_NOT_FOUND", "no standard replay path is configured for this OS; pass --dir")
	}
}
