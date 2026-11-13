// Package account identifies the client account signed in to the game from
// the client's own local state.
package account

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/resources/archive"
)

const (
	// deviceFile stores device-wide client preferences, including the sign-in
	// token of the account authorized in the last session.
	deviceFile = "device_data.dat"
	tokenKey   = "InternalWebToken2"
)

// Account identifies the client account authorized in the game.
type Account struct {
	ID uint32
}

// Default returns the account authorized in the client's last session. found
// is false when the client has not signed in yet, its local state is
// unavailable, or the current platform has no standard local-data path.
func Default() (account Account, found bool) {
	dir, err := blitz.DefaultLocalDataDir()
	if err != nil {
		return Account{}, false
	}
	account, found, err = Read(dir)
	if err != nil {
		return Account{}, false
	}
	return account, found
}

// Read returns the account recorded in the client's device settings in dir.
// The settings file appears once the client stores device-wide preferences, so
// a missing file is not an error. The stored sign-in token begins with the
// account ID.
func Read(dir string) (account Account, found bool, err error) {
	path := filepath.Join(dir, deviceFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Account{}, false, nil
		}
		return Account{}, false, blitz.IOError("could not read the device settings", path, err)
	}
	root, err := archive.ParseKeyedArchive(data)
	if err != nil {
		return Account{}, false, blitz.ErrorAt("ACCOUNT_INVALID", "device settings are not a keyed archive", path)
	}
	token, ok := root.Get(tokenKey).AsString()
	if !ok {
		return Account{}, false, nil
	}
	id, ok := tokenAccountID(token)
	if !ok {
		return Account{}, false, nil
	}
	return Account{ID: id}, true, nil
}

// tokenAccountID reads the account ID that prefixes a sign-in token.
func tokenAccountID(token string) (uint32, bool) {
	prefix, _, ok := strings.Cut(token, ":")
	if !ok || prefix == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(prefix, 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint32(id), true
}
