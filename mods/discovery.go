package mods

import (
	"os"
	"slices"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/pathx"
)

// DiscoverMods returns manifest paths recursively in stable order, without
// loading mods. Empty directories return an empty list and malformed mods
// remain discoverable; loading a discovered path reports conflicting
// mod.yaml/mod.yml formats. Directory symlinks are not followed.
func DiscoverMods(dir string) ([]string, error) {
	manifests := []string{}
	if err := collectManifests(dir, &manifests); err != nil {
		return nil, err
	}
	slices.SortFunc(manifests, pathx.Compare)
	return manifests, nil
}

func collectManifests(dir string, manifests *[]string) error {
	entries, err := readDirUnsorted(dirOrDot(dir))
	if err != nil {
		return blitz.ErrorAt("IO_ERROR", "could not scan mod directory: "+blitz.IOMessage(err), dir)
	}
	var local []string
	for _, entry := range entries {
		entryPath := pathx.Join(dir, entry.Name())
		if entry.IsDir() {
			if err := collectManifests(entryPath, manifests); err != nil {
				return err
			}
		} else if (entry.Name() == "mod.yaml" || entry.Name() == "mod.yml") && isFile(entryPath) {
			local = append(local, entryPath)
		}
	}
	slices.SortFunc(local, pathx.Compare)
	if len(local) > 0 {
		*manifests = append(*manifests, local[0])
	}
	return nil
}

func readDirUnsorted(dir string) ([]os.DirEntry, error) {
	handle, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	return handle.ReadDir(-1)
}
