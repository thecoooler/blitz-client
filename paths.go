package blitzclient

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultCacheDir returns the standard Tanks Blitz resource-cache path without
// checking whether it exists. Linux has no standard path: it searches the
// user's Proton and PortProton prefixes for the folder the client wrote.
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
	case "linux":
		if dir, ok := linuxDocumentsPath("TanksBlitz/packs"); ok {
			return dir, nil
		}
		return "", NewError("CACHE_DIR_NOT_FOUND", linuxPrefixesNotFound())
	default:
		return "", NewError("CACHE_DIR_NOT_FOUND", "no standard resource-cache path is configured for this OS")
	}
}

// DefaultLocalDataDir returns the standard directory holding client-local
// state, such as device settings and downloaded configuration, without
// checking whether it exists.
func DefaultLocalDataDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, ok := os.LookupEnv("HOME")
		if !ok {
			return "", NewError("LOCAL_DATA_DIR_NOT_FOUND", "HOME is not set")
		}
		return filepath.Join(home, "Library/Containers/com.tanksblitz/Data/Documents/tb-project/cache"), nil
	case "windows":
		profile, ok := os.LookupEnv("USERPROFILE")
		if !ok {
			return "", NewError("LOCAL_DATA_DIR_NOT_FOUND", "USERPROFILE is not set")
		}
		return filepath.Join(profile, "Documents/TanksBlitz/cache"), nil
	case "linux":
		if dir, ok := linuxDocumentsPath("TanksBlitz/cache"); ok {
			return dir, nil
		}
		return "", NewError("LOCAL_DATA_DIR_NOT_FOUND", linuxPrefixesNotFound())
	default:
		return "", NewError("LOCAL_DATA_DIR_NOT_FOUND", "no standard local-data path is configured for this OS")
	}
}

// DefaultMapsConfigDir returns the standard directory holding the maps server
// cache (maps_config_*.dat) without checking whether it exists.
func DefaultMapsConfigDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, ok := os.LookupEnv("HOME")
		if !ok {
			return "", NewError("MAPS_CONFIG_NOT_FOUND", "HOME is not set")
		}
		return filepath.Join(home, "Library/Containers/com.tanksblitz/Data/Documents/tb-project/cache"), nil
	case "windows":
		profile, ok := os.LookupEnv("USERPROFILE")
		if !ok {
			return "", NewError("MAPS_CONFIG_NOT_FOUND", "USERPROFILE is not set")
		}
		return filepath.Join(profile, "Documents/TanksBlitz/cache"), nil
	case "linux":
		if dir, ok := linuxDocumentsPath("TanksBlitz/cache"); ok {
			return dir, nil
		}
		return "", NewError("MAPS_CONFIG_NOT_FOUND", linuxPrefixesNotFound())
	default:
		return "", NewError("MAPS_CONFIG_NOT_FOUND", "no standard maps-server-cache path is configured for this OS")
	}
}

// DefaultLogDir returns the standard client journal path for this OS without
// checking whether it exists. The client writes one blitz-logs file per launch
// into this directory.
func DefaultLogDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, ok := os.LookupEnv("HOME")
		if !ok {
			return "", NewError("LOG_FOLDER_NOT_FOUND", "HOME is not set")
		}
		return filepath.Join(home, "Library/Containers/com.tanksblitz/Data/Documents/tb-project"), nil
	case "windows":
		profile, ok := os.LookupEnv("USERPROFILE")
		if !ok {
			return "", NewError("LOG_FOLDER_NOT_FOUND", "USERPROFILE is not set")
		}
		return filepath.Join(profile, "Documents/TanksBlitz"), nil
	case "linux":
		if dir, ok := linuxDocumentsPath("TanksBlitz"); ok {
			return dir, nil
		}
		return "", NewError("LOG_FOLDER_NOT_FOUND", linuxPrefixesNotFound())
	default:
		return "", NewError("LOG_FOLDER_NOT_FOUND", "no standard journal path is configured for this OS")
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
	case "linux":
		if dir, ok := linuxDocumentsPath("TanksBlitz/replays"); ok {
			return dir, nil
		}
		return "", NewError("REPLAY_FOLDER_NOT_FOUND", linuxPrefixesNotFound()+"; pass --dir")
	default:
		return "", NewError("REPLAY_FOLDER_NOT_FOUND", "no standard replay path is configured for this OS; pass --dir")
	}
}

// linuxPrefixesNotFound explains a Linux default that no prefix answered.
func linuxPrefixesNotFound() string {
	if len(LinuxPrefixes()) == 0 {
		return "no Proton or PortProton prefix was found under $HOME"
	}
	return "no prefix holds a Documents/TanksBlitz folder"
}
