package client

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	blitz "github.com/thecoooler/blitz-client"
)

func TestResolveGameInstallationRequiresResourceRoot(t *testing.T) {
	plain := t.TempDir()
	_, err := ResolveGameInstallation(plain)
	var typed *blitz.Error
	if err == nil || !errors.As(err, &typed) || typed.Code != "GAME_DIR_INVALID" {
		t.Fatalf("error = %v", err)
	}

	game := t.TempDir()
	data := filepath.Join(game, "Data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveGameInstallation(game)
	if err != nil {
		t.Fatal(err)
	}
	wantGame, err := filepath.EvalSymlinks(game)
	if err != nil {
		t.Fatal(err)
	}
	wantData, err := filepath.EvalSymlinks(data)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.GameDir != wantGame || resolved.ResourceDir != wantData {
		t.Fatalf("resolved = %+v", resolved)
	}

	selected, err := ResolveGameInstallation(data)
	if err != nil {
		t.Fatal(err)
	}
	if selected.GameDir != wantGame || selected.ResourceDir != wantData {
		t.Fatalf("selected Data resolved = %+v", selected)
	}
}
