package cli

import (
	"strings"
	"testing"
)

func TestGameStatusReportsMissingClient(t *testing.T) {
	out := run(t, nil, "game", "status", "--process-name", "no-such-blitz-client-process", "--json")
	if out.code != 0 {
		t.Fatalf("exit %d: %s", out.code, out.stderr)
	}
	value := out.json(t)
	if value["running"] != false || value["in_game"] != false || value["window"] != nil {
		t.Fatalf("unexpected state: %s", out.stdout)
	}

	out = run(t, nil, "game", "status", "--process-name", "no-such-blitz-client-process")
	if out.code != 0 || string(out.stdout) != "running: no\n" {
		t.Fatalf("exit %d: %q", out.code, out.stdout)
	}
}

func TestGameStatusRejectsInvalidOptions(t *testing.T) {
	for _, args := range [][]string{
		{"game"},
		{"game", "status", "--poll-ms", "0"},
		{"game", "status", "--process-name"},
		{"game", "status", "extra"},
	} {
		out := run(t, nil, append(args, "--json")...)
		if out.code != 2 || !strings.Contains(string(out.stdout), `"USAGE"`) {
			t.Errorf("%v: exit %d: %s %s", args, out.code, out.stdout, out.stderr)
		}
	}
}
