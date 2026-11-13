package account

import (
	"os"
	"path/filepath"
	"testing"

	ka "github.com/thecoooler/blitz-client/internal/katest"
)

func writeDeviceData(t *testing.T, dir string, fields ka.Fields) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, deviceFile), ka.Archive(fields), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestReadReturnsSignedInAccount(t *testing.T) {
	dir := t.TempDir()
	// A client that has never stored device preferences has no account.
	if account, found, err := Read(dir); err != nil || found || account != (Account{}) {
		t.Fatalf("empty dir: %+v found=%t err=%v", account, found, err)
	}
	writeDeviceData(t, dir, ka.Fields{
		{Key: "Region", Value: ka.Text("Russia")},
		{Key: "InternalWebToken2", Value: ka.Text("91803618:8576886749150241595:88874650957510729723608388572935342432")},
	})
	account, found, err := Read(dir)
	if err != nil || !found || account.ID != 91803618 {
		t.Fatalf("got %+v found=%t err=%v", account, found, err)
	}
}

func TestReadIgnoresMissingOrMalformedTokens(t *testing.T) {
	for name, fields := range map[string]ka.Fields{
		"no token":     {{Key: "Region", Value: ka.Text("Russia")}},
		"empty token":  {{Key: "InternalWebToken2", Value: ka.Text("")}},
		"no prefix":    {{Key: "InternalWebToken2", Value: ka.Text("91803618")}},
		"zero prefix":  {{Key: "InternalWebToken2", Value: ka.Text("0:1:2")}},
		"numeric text": {{Key: "InternalWebToken2", Value: ka.Text("account:1:2")}},
		"non-string":   {{Key: "InternalWebToken2", Value: ka.Number(91803618)}},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeDeviceData(t, dir, fields)
			account, found, err := Read(dir)
			if err != nil || found || account != (Account{}) {
				t.Fatalf("got %+v found=%t err=%v", account, found, err)
			}
		})
	}
}

func TestReadRejectsNonArchive(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, deviceFile), []byte("not a keyed archive"), 0600); err != nil {
		t.Fatal(err)
	}
	_, found, err := Read(dir)
	if err == nil || found {
		t.Fatalf("found=%t err=%v", found, err)
	}
}
