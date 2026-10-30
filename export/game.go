package export

import (
	"runtime/debug"
	"strings"
)

// SnapshotGame identifies the client build a snapshot was read from.
type SnapshotGame struct {
	// VersionText is the first line of version.txt, for example
	// "26.10.70_533068 master MacOS". Version, Build, Branch and Platform are
	// its fields; they are null when the line has no such part.
	VersionText *string `json:"versionText"`
	Version     *string `json:"version"`
	Build       *string `json:"build"`
	Branch      *string `json:"branch"`
	Platform    *string `json:"platform"`
}

// extractGame reads version.txt; without it every field is null.
func extractGame(r snapshotReader) (*SnapshotGame, error) {
	g := &SnapshotGame{}
	if r.has("version.txt") {
		b, err := r.read("version.txt")
		if err != nil {
			return nil, err
		}
		parseVersionText(g, string(b))
	}
	return g, nil
}

const modulePath = "github.com/thecoooler/blitz-client"

// exporterVersion is the blitz-client module version of the running binary,
// such as "v1.2.0" or a pseudo-version "v0.0.0-20261029201357-ed5695baa714"
// (with "+dirty" for uncommitted changes). It is null when the build has no
// version: go run, go test, a build outside a VCS checkout or a dependency
// replaced by a local directory.
func exporterVersion() *string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	module := &info.Main
	if module.Path != modulePath {
		module = nil
		for _, dep := range info.Deps {
			if dep.Path == modulePath {
				module = dep
				break
			}
		}
	}
	if module == nil {
		return nil
	}
	if module.Replace != nil {
		// A local directory replacement has no version of its own.
		module = module.Replace
	}
	if module.Version == "" || module.Version == "(devel)" {
		return nil
	}
	return &module.Version
}

// parseVersionText splits "<version>_<build> <branch> <platform>". A first
// token without a numeric "_" suffix is the version alone; the platform is the
// rest of the line, which may contain spaces.
func parseVersionText(g *SnapshotGame, text string) {
	line, _, _ := strings.Cut(strings.TrimPrefix(text, "\ufeff"), "\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	g.VersionText = &line
	fields := strings.Fields(line)
	version := fields[0]
	if i := strings.LastIndexByte(version, '_'); i > 0 && i < len(version)-1 && strings.Trim(version[i+1:], "0123456789") == "" {
		build := version[i+1:]
		version = version[:i]
		g.Build = &build
	}
	g.Version = &version
	if len(fields) > 1 {
		g.Branch = &fields[1]
	}
	if len(fields) > 2 {
		platform := strings.Join(fields[2:], " ")
		g.Platform = &platform
	}
}
