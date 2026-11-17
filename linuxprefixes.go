package blitzclient

import (
	"os"
	"path/filepath"
	"strings"
)

// LinuxPrefixes lists the Wine and Proton prefixes of the current user that
// hold a drive_c, in the order they are searched: PortProton keeps every
// prefix in its own data directory, and Steam keeps one pfx per Proton game
// under compatdata. Other operating systems have none.
func LinuxPrefixes() []string {
	home, ok := os.LookupEnv("HOME")
	if !ok {
		return nil
	}
	var out []string
	for _, root := range []string{
		filepath.Join(home, "PortProton", "data", "prefixes"),
		filepath.Join(home, ".local", "share", "PortProton", "prefixes"),
	} {
		out = append(out, prefixesUnder(root, "")...)
	}
	var steamRoots []string
	for _, steam := range []string{
		".local/share/Steam",
		".steam/steam",
		".var/app/com.valvesoftware.Steam/.local/share/Steam",
		"snap/steam/common/.local/share/Steam",
	} {
		steamRoots = append(steamRoots, filepath.Join(home, steam))
	}
	steamRoots = append(steamRoots, "/usr/share/steam", "/usr/lib/steam")
	for _, steam := range steamRoots {
		out = append(out, prefixesUnder(filepath.Join(steam, "steamapps", "compatdata"), "pfx")...)
	}
	return out
}

// prefixesUnder lists the subdirectories of root, or the folder named inner
// inside each of them, that hold a drive_c.
func prefixesUnder(root, inner string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		prefix := filepath.Join(root, entry.Name())
		if inner != "" {
			prefix = filepath.Join(prefix, inner)
		}
		if info, err := os.Stat(filepath.Join(prefix, "drive_c")); err != nil || !info.IsDir() {
			continue
		}
		out = append(out, prefix)
	}
	return out
}

// linuxDocumentsPath returns the first <prefix>/drive_c/users/<user>/Documents
// path holding relative, and whether it exists. When no prefix holds it, the
// path under the first prefix with a user profile is returned, so that a caller
// reports the folder it would use rather than none.
func linuxDocumentsPath(relative string) (string, bool) {
	var candidates []string
	for _, prefix := range LinuxPrefixes() {
		for _, user := range wineUsers(prefix) {
			candidates = append(candidates, filepath.Join(user, "Documents", relative))
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}
	}
	if len(candidates) > 0 {
		return candidates[0], false
	}
	return "", false
}

// wineUsers lists the user profiles of a prefix, without the profiles Wine
// shares between users and the ones it keeps for its own use.
func wineUsers(prefix string) []string {
	root := filepath.Join(prefix, "drive_c", "users")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		switch strings.ToLower(entry.Name()) {
		case "public", "all users", "default", "default user", "defaultuser0":
			continue
		}
		out = append(out, filepath.Join(root, entry.Name()))
	}
	return out
}
