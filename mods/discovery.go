package mods

import (
	"os"
	"slices"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/pathx"
)

// ModDiscoveryError describes a source directory that could not be scanned.
type ModDiscoveryError struct {
	Directory string
	Err       error
}

// ModDiscovery is the combined result of scanning independent roots.
type ModDiscovery struct {
	Manifests []string
	Errors    []ModDiscoveryError
}

// DiscoverModsIn scans independent roots, preserving healthy sources when
// another is missing. Manifests found before an unreadable nested directory
// are kept and the failure is reported. Canonical manifest paths deduplicate
// overlapping roots and directory aliases.
func DiscoverModsIn(dirs []string) ModDiscovery {
	report := ModDiscovery{Manifests: []string{}}
	seen := map[string]bool{}
	for _, dir := range dirs {
		manifests, failures := discoverModsResilient(dir)
		report.Errors = append(report.Errors, failures...)
		for _, manifest := range manifests {
			canonical := manifest
			if resolved, err := pathx.Canonicalize(manifest); err == nil {
				canonical = resolved
			}
			if seen[canonical] {
				continue
			}
			seen[canonical] = true
			report.Manifests = append(report.Manifests, canonical)
		}
	}
	slices.SortFunc(report.Manifests, pathx.Compare)
	return report
}

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

// discoverModsResilient walks one root, keeping manifests from readable
// directories and reporting every unreadable directory separately.
func discoverModsResilient(dir string) ([]string, []ModDiscoveryError) {
	manifests := []string{}
	failures := []ModDiscoveryError{}
	if err := collectManifestsResilient(dir, true, &manifests, &failures); err != nil {
		return nil, []ModDiscoveryError{{Directory: dir, Err: err}}
	}
	slices.SortFunc(manifests, pathx.Compare)
	return manifests, failures
}

func collectManifestsResilient(dir string, root bool, manifests *[]string, failures *[]ModDiscoveryError) error {
	entries, err := readDirUnsorted(dirOrDot(dir))
	if err != nil {
		err = blitz.ErrorAt("IO_ERROR", "could not scan mod directory: "+blitz.IOMessage(err), dir)
		if root {
			return err
		}
		*failures = append(*failures, ModDiscoveryError{Directory: dir, Err: err})
		return nil
	}
	var local []string
	for _, entry := range entries {
		entryPath := pathx.Join(dir, entry.Name())
		if entry.IsDir() {
			_ = collectManifestsResilient(entryPath, false, manifests, failures)
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
