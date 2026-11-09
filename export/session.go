package export

import (
	"os"

	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/internal/pathx"
)

// exportSessionOptions is the path, cache and output request shared by both
// client export entry points.
type exportSessionOptions struct {
	GameDir   string
	CacheDir  string
	NoCache   bool
	OutputDir string
	// RequireEmptyOutput rejects an existing non-empty output directory.
	RequireEmptyOutput bool
}

// exportRoots is the resolved source and output state of one export run.
type exportRoots struct {
	Game          *client.ResolvedGameDir
	ResourceDir   string
	CacheDir      string // empty when the cache overlay is disabled
	OutputDir     string
	OutputExisted bool
}

// outputOverlapError reports that the output is inside a source directory or
// contains one.
type outputOverlapError struct {
	output string
}

func (e *outputOverlapError) Error() string {
	return "output and source directories must not overlap"
}

// outputNotEmptyError reports that a snapshot export needs a new or empty
// output directory.
type outputNotEmptyError struct {
	output string
}

func (e *outputNotEmptyError) Error() string {
	return "export requires a new or empty output directory: " + e.output
}

// exportCacheConflict reports the mutually exclusive cache options. Each entry
// point maps it to its own error code and message.
func exportCacheConflict(cacheDir string, noCache bool) bool {
	return noCache && cacheDir != ""
}

// selectExportCache applies the cache policy: an explicit folder wins, the
// installation's default cache is used only when it exists, and --no-cache
// disables the overlay entirely.
func selectExportCache(defaultCache *string, explicit string, noCache bool) (string, error) {
	switch {
	case noCache:
		return "", nil
	case explicit != "":
		return canonicalDirectory(explicit)
	case defaultCache != nil && isDir(*defaultCache):
		return canonicalDirectory(*defaultCache)
	}
	return "", nil
}

// openExportRoots resolves the game installation, canonical source roots and
// output path for one export. It owns the overlap rule shared by the snapshot
// and raw-files exports; RequireEmptyOutput adds the snapshot's new-directory
// rule.
func openExportRoots(options exportSessionOptions) (*exportRoots, error) {
	game, err := client.ResolveGameDir(options.GameDir)
	if err != nil {
		return nil, err
	}
	// Canonical roots keep the overlap check below correct for symlinked Data.
	resourceDir, err := canonicalDirectory(game.ResourceDir)
	if err != nil {
		return nil, err
	}
	cacheDir, err := selectExportCache(game.CacheDir, options.CacheDir, options.NoCache)
	if err != nil {
		return nil, err
	}
	outputDir, err := resolveOutputPath(options.OutputDir)
	if err != nil {
		return nil, err
	}
	roots := &exportRoots{
		Game:        game,
		ResourceDir: resourceDir,
		CacheDir:    cacheDir,
		OutputDir:   outputDir,
	}
	for _, source := range roots.sources() {
		if pathx.HasPrefix(outputDir, source) || pathx.HasPrefix(source, outputDir) {
			return nil, &outputOverlapError{output: outputDir}
		}
	}
	if options.RequireEmptyOutput {
		entries, err := os.ReadDir(outputDir)
		roots.OutputExisted = err == nil
		switch {
		case err == nil && len(entries) != 0:
			return nil, &outputNotEmptyError{output: outputDir}
		case err != nil && !os.IsNotExist(err):
			return nil, err
		}
	}
	return roots, nil
}

// sources returns the canonical resource roots the output must not overlap.
func (r *exportRoots) sources() []string {
	sources := []string{r.ResourceDir}
	if r.CacheDir != "" {
		sources = append(sources, r.CacheDir)
	}
	return sources
}
