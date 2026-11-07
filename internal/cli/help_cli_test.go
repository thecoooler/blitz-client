package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpPerLevel(t *testing.T) {
	root := run(t, nil, "--help")
	if root.code != 0 {
		t.Fatalf("exit %d: %s", root.code, root.stderr)
	}
	if !strings.Contains(string(root.stdout), "Commands:") || !strings.Contains(string(root.stdout), "client files unpack") {
		t.Fatalf("root help misses commands: %s", root.stdout)
	}
	// The root help lists commands, not every option of every leaf.
	if strings.Contains(string(root.stdout), "--max-texture-size") || strings.Contains(string(root.stdout), "--idle-timeout-ms") {
		t.Fatalf("root help dumps leaf options: %s", root.stdout)
	}

	client := run(t, nil, "client", "--help")
	if client.code != 0 {
		t.Fatalf("exit %d: %s", client.code, client.stderr)
	}
	if !strings.Contains(string(client.stdout), "client locate") || strings.Contains(string(client.stdout), "--dry-run") {
		t.Fatalf("client help is not scoped to client commands: %s", client.stdout)
	}

	unpack := run(t, nil, "client", "files", "unpack", "--help")
	if unpack.code != 0 {
		t.Fatalf("exit %d: %s", unpack.code, unpack.stderr)
	}
	for _, option := range []string{"--output-dir", "--game-dir", "--cache-dir", "--no-cache", "--types", "--exclude", "--dry-run"} {
		if !strings.Contains(string(unpack.stdout), option) {
			t.Fatalf("unpack help misses %s: %s", option, unpack.stdout)
		}
	}

	if short := run(t, nil, "client", "files", "unpack", "-h"); short.code != 0 || !bytes.Equal(short.stdout, unpack.stdout) {
		t.Fatalf("-h does not match --help: exit %d", short.code)
	}
}

func TestGrammarRejectsInvalidOptions(t *testing.T) {
	for _, args := range [][]string{
		{"--bogus"},
		{"client", "files", "unpack"},
		{"client", "files", "unpack", "--output-dir", "out", "--no-cache", "--cache-dir", "cache"},
		{"client", "files", "unpack", "--output-dir", "out", "--types", "bogus"},
		{"client", "files", "unpack", "--output-dir", "out", "--types", "configs,"},
		{"client", "export", "--output-dir", "out", "--webp-quality", "101"},
		{"client", "export", "--output-dir", "out", "--texture-format", "jpeg"},
		{"client", "export", "--output-dir", "out", "--max-texture-size", "0"},
		{"client", "export", "--output-dir", "out", "--include", "all,data"},
		{"mod", "update-from-git", "mod"},
		{"replay", "follow", "replay.tbreplay", "--poll-ms", "90000000000"},
	} {
		out := run(t, nil, append(args, "--json")...)
		// follow and --watch errors always go to stderr, even in JSON mode.
		if out.code != 2 || (!strings.Contains(string(out.stdout), `"USAGE"`) && !strings.Contains(string(out.stderr), "USAGE")) {
			t.Errorf("%v: exit %d: %s %s", args, out.code, out.stdout, out.stderr)
		}
	}
}

func TestEmptyGameDirIsRejected(t *testing.T) {
	for _, args := range [][]string{
		{"--json", "client", "tanks", "list", "--game-dir", ""},
		{"client", "files", "unpack", "--output-dir", "out", "--json", "--game-dir", ""},
	} {
		out := run(t, nil, args...)
		if out.code != 2 || !strings.Contains(string(out.stdout), "GAME_DIR_NOT_FOUND") {
			t.Fatalf("%v: exit %d: %s %s", args, out.code, out.stdout, out.stderr)
		}
	}
}
