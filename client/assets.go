package client

import (
	"os"
	"path/filepath"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/resources"
)

// ReadClientAsset reads a plain or DVPL resource and returns its payload.
func ReadClientAsset(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, blitz.IOError("could not read client resource", path, err)
	}
	plain, _, err := resources.DecodeDVPL(data)
	if err != nil {
		typed := *blitz.AsError(err)
		typed.Path = path
		return nil, &typed
	}
	return plain, nil
}

// ResolveClientAsset finds a resource as a plain file or as its DVPL
// sibling. A relative path ending in .dvpl prefers the unpacked file. The
// result is empty when neither exists.
func ResolveClientAsset(dataDir, relative string) string {
	var candidates []string
	if plain, ok := strings.CutSuffix(relative, ".dvpl"); ok {
		candidates = []string{plain, relative}
	} else {
		candidates = []string{relative, relative + ".dvpl"}
	}
	for _, candidate := range candidates {
		path := filepath.Join(dataDir, candidate)
		if isFile(path) {
			return path
		}
	}
	return ""
}
