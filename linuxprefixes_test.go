package blitzclient

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// prefixTree creates <root>/<name>/drive_c/users/<user>/Documents/<relative>.
func prefixTree(t *testing.T, prefix, user, relative string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(prefix, "drive_c", "users", user, "Documents", relative), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestLinuxPrefixesFindsPortProtonAndSteamPrefixes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	portproton := filepath.Join(home, "PortProton", "data", "prefixes", "LGC")
	steam := filepath.Join(home, ".local", "share", "Steam", "steamapps", "compatdata", "444200", "pfx")
	prefixTree(t, portproton, "steamuser", "TanksBlitz")
	prefixTree(t, steam, "steamuser", "TanksBlitz")
	// PortProton keeps folders of its own beside the prefixes.
	if err := os.MkdirAll(filepath.Join(home, "PortProton", "data", "prefixes", "dist"), 0o755); err != nil {
		t.Fatal(err)
	}

	prefixes := LinuxPrefixes()
	if len(prefixes) == 0 || prefixes[0] != portproton {
		t.Fatalf("PortProton prefixes come first, got %v", prefixes)
	}
	if !slices.Contains(prefixes, steam) {
		t.Errorf("the Steam prefix is missing from %v", prefixes)
	}
	if slices.Contains(prefixes, filepath.Join(home, "PortProton", "data", "prefixes", "dist")) {
		t.Errorf("a folder without drive_c is not a prefix: %v", prefixes)
	}
}

func TestLinuxDocumentsPathPrefersAnExistingFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	empty := filepath.Join(home, "PortProton", "data", "prefixes", "OLD")
	full := filepath.Join(home, "PortProton", "data", "prefixes", "LGC")
	if err := os.MkdirAll(filepath.Join(empty, "drive_c", "users", "steamuser", "Documents"), 0o755); err != nil {
		t.Fatal(err)
	}
	prefixTree(t, full, "steamuser", "TanksBlitz/replays")

	path, ok := linuxDocumentsPath("TanksBlitz/replays")
	want := filepath.Join(full, "drive_c", "users", "steamuser", "Documents", "TanksBlitz", "replays")
	if !ok || path != want {
		t.Fatalf("got %q (exists %t), want %q", path, ok, want)
	}
}

func TestLinuxDocumentsPathReportsTheFolderItWouldUse(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	prefix := filepath.Join(home, "PortProton", "data", "prefixes", "LGC")
	prefixTree(t, prefix, "steamuser", "Other")

	path, ok := linuxDocumentsPath("TanksBlitz/replays")
	want := filepath.Join(prefix, "drive_c", "users", "steamuser", "Documents", "TanksBlitz", "replays")
	if ok || path != want {
		t.Fatalf("got %q (exists %t), want %q", path, ok, want)
	}
}

func TestLinuxDocumentsPathIgnoresSharedWineProfiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	prefix := filepath.Join(home, "PortProton", "data", "prefixes", "LGC")
	prefixTree(t, prefix, "Public", "TanksBlitz")

	if path, ok := linuxDocumentsPath("TanksBlitz"); ok || path != "" {
		t.Fatalf("the Public profile is not a user profile, got %q", path)
	}
}
