package mods

import (
	"os"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/resources"
)

// targetPlan is the shared preparation policy for one logical resource. It
// records the current and original bytes, the operations that compose the
// installed output, and that output itself.
type targetPlan struct {
	logicalPath  string
	target       string
	current      []byte
	base         []byte
	baseSnapshot string
	managed      bool
	// added holds this mod's operations; operations holds every operation
	// composed into output, sorted by mod id.
	added      []storedOperation
	operations []modOperation
	output     []byte
}

// targetPlanner builds target plans for one manifest and remembers which
// physical targets have already been claimed. duplicate renders the error a
// caller reports when two logical paths resolve to the same target, because
// validation and installation attach different paths to that error.
type targetPlanner struct {
	game      *client.ResolvedGameDir
	state     *installerState
	manifest  *Manifest
	occupied  map[string]string
	duplicate func(target string) error
}

func newTargetPlanner(game *client.ResolvedGameDir, state *installerState, manifest *Manifest,
	duplicate func(target string) error) *targetPlanner {
	return &targetPlanner{
		game: game, state: state, manifest: manifest,
		occupied: map[string]string{}, duplicate: duplicate,
	}
}

// plan resolves and verifies one logical resource, then composes the
// operations that would be installed. Validation and installation share this
// policy; each caller keeps its own reporting and write behavior.
func (p *targetPlanner) plan(logicalPath string) (*targetPlan, error) {
	target, err := resolveLogicalTarget(p.game, logicalPath)
	if err != nil {
		return nil, err
	}
	key := lowerKey(target)
	previous, exists := p.occupied[key]
	p.occupied[key] = logicalPath
	if exists && previous != logicalPath {
		return nil, p.duplicate(target)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		return nil, blitz.IOError("could not read game resource", target, err)
	}
	snapshot, crc, managed, err := managedResource(p.state, target)
	if err != nil {
		return nil, err
	}
	plan := &targetPlan{
		logicalPath: logicalPath,
		target:      target,
		current:     current,
		base:        current,
		managed:     managed,
	}
	if managed {
		if err := verifyInstalledCRC(current, crc, target); err != nil {
			return nil, err
		}
		if plan.base, err = readBaseSnapshot(p.game, target, snapshot); err != nil {
			return nil, err
		}
		plan.baseSnapshot = snapshot
	} else if plan.baseSnapshot, err = baseSnapshotPath(p.game, target); err != nil {
		return nil, err
	}
	if plan.added, err = newOperations(p.manifest, logicalPath); err != nil {
		return nil, err
	}
	plan.operations = otherOperations(p.state, p.manifest.ID, target)
	for _, operation := range plan.added {
		plan.operations = append(plan.operations, modOperation{p.manifest.ID, operation})
	}
	sortByMod(plan.operations)
	plan.output = plan.base
	if len(plan.operations) > 0 {
		if plan.output, err = composeResource(target, plan.base, plan.operations); err != nil {
			return nil, err
		}
	}
	return plan, nil
}

// verifyInstalledCRC rejects a managed resource changed outside the installer.
func verifyInstalledCRC(current []byte, expected uint32, target string) error {
	if resources.CRC32(current) != expected {
		return blitz.ErrorAt("INSTALLED_FILE_CHANGED", "managed resource changed outside the installer", target)
	}
	return nil
}

// readCurrentChecked reads a managed resource and verifies its installed CRC.
func readCurrentChecked(target string, expected uint32, readContext string) ([]byte, error) {
	current, err := os.ReadFile(target)
	if err != nil {
		return nil, blitz.IOError(readContext, target, err)
	}
	if err := verifyInstalledCRC(current, expected, target); err != nil {
		return nil, err
	}
	return current, nil
}
