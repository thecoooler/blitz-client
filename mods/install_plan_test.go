package mods

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/resources"
)

type planFixture struct {
	t    *testing.T
	game *client.ResolvedGameDir
}

func newPlanFixture(t *testing.T) *planFixture {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "game", "Data"), 0o755); err != nil {
		t.Fatal(err)
	}
	game, err := client.ResolveGameDir(filepath.Join(root, "game"))
	if err != nil {
		t.Fatal(err)
	}
	return &planFixture{t: t, game: game}
}

func (f *planFixture) write(name string, data []byte) string {
	f.t.Helper()
	path := filepath.Join(f.game.ResourceDir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		f.t.Fatal(err)
	}
	return path
}

func planString(value string) *string { return &value }

func planReplace(name, find, replace string) ModFile {
	return ModFile{Name: name, Action: ActionReplace, Find: planString(find), Replace: planString(replace)}
}

func newTestPlanner(game *client.ResolvedGameDir, state *installerState, manifest *Manifest) *targetPlanner {
	return newTargetPlanner(game, state, manifest, func(target string) error {
		return blitz.ErrorAt("MANIFEST_DUPLICATE_TARGET", "multiple entries resolve to the same game file", target)
	})
}

func TestTargetPlanManagedTargetReadsBaseSnapshotAndComposesOtherMods(t *testing.T) {
	f := newPlanFixture(t)
	current := []byte("alpha: other\nbeta: original\n")
	base := []byte("alpha: original\nbeta: original\n")
	target := f.write("resource.yaml", current)
	snapshot, err := baseSnapshotPath(f.game, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := createBackup(snapshot, base); err != nil {
		t.Fatal(err)
	}
	state := &installerState{Version: 2, Mods: []installedMod{{
		ID: "Other", Name: "Other",
		Files: []installedFile{{
			LogicalPath: "resource.yaml", Target: target, BaseSnapshot: snapshot,
			Operation:      storedOperation{Action: ActionReplace, Find: planString("alpha: original"), Replace: planString("alpha: other")},
			InstalledCRC32: resources.CRC32(current),
		}},
	}}}
	manifest := &Manifest{ID: "New", Name: "New", Files: []ModFile{
		planReplace("resource.yaml", "beta: original", "beta: new"),
	}}
	plan, err := newTestPlanner(f.game, state, manifest).plan("resource.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.managed || plan.target != target {
		t.Fatalf("managed=%v target=%q", plan.managed, plan.target)
	}
	if !bytes.Equal(plan.current, current) || !bytes.Equal(plan.base, base) {
		t.Fatalf("current=%q base=%q", plan.current, plan.base)
	}
	if plan.baseSnapshot != snapshot {
		t.Fatalf("snapshot=%q", plan.baseSnapshot)
	}
	if len(plan.added) != 1 || len(plan.operations) != 2 {
		t.Fatalf("added=%d operations=%d", len(plan.added), len(plan.operations))
	}
	if expected := "alpha: other\nbeta: new\n"; string(plan.output) != expected {
		t.Fatalf("output=%q", plan.output)
	}
}

func TestTargetPlanRejectsManagedTargetChangedOutsideInstaller(t *testing.T) {
	f := newPlanFixture(t)
	target := f.write("resource.yaml", []byte("current\n"))
	snapshot, err := baseSnapshotPath(f.game, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := createBackup(snapshot, []byte("original\n")); err != nil {
		t.Fatal(err)
	}
	state := &installerState{Version: 2, Mods: []installedMod{{
		ID: "Other", Name: "Other",
		Files: []installedFile{{
			LogicalPath: "resource.yaml", Target: target, BaseSnapshot: snapshot,
			Operation:      storedOperation{Action: ActionReplace, Find: planString("original"), Replace: planString("other")},
			InstalledCRC32: resources.CRC32([]byte("different\n")),
		}},
	}}}
	_, err = newTestPlanner(f.game, state, &Manifest{ID: "New", Name: "New"}).plan("resource.yaml")
	typed := blitz.AsError(err)
	if typed == nil || typed.Code != "INSTALLED_FILE_CHANGED" || typed.Path != target {
		t.Fatalf("%+v", typed)
	}
}

func TestTargetPlanUnmanagedTargetUsesCurrentBytesAsBase(t *testing.T) {
	f := newPlanFixture(t)
	current := []byte("alpha: original\n")
	target := f.write("resource.yaml", current)
	manifest := &Manifest{ID: "New", Name: "New", Files: []ModFile{
		planReplace("resource.yaml", "alpha: original", "alpha: installed"),
	}}
	plan, err := newTestPlanner(f.game, &installerState{}, manifest).plan("resource.yaml")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := baseSnapshotPath(f.game, target)
	if err != nil {
		t.Fatal(err)
	}
	if plan.managed {
		t.Fatal("unmanaged target reported as managed")
	}
	if !bytes.Equal(plan.current, current) || !bytes.Equal(plan.base, current) {
		t.Fatalf("current=%q base=%q", plan.current, plan.base)
	}
	if plan.baseSnapshot != snapshot {
		t.Fatalf("snapshot=%q", plan.baseSnapshot)
	}
	if expected := "alpha: installed\n"; string(plan.output) != expected {
		t.Fatalf("output=%q", plan.output)
	}
}

func TestTargetPlanRejectsDuplicateTargets(t *testing.T) {
	f := newPlanFixture(t)
	target := f.write("resource.yaml", []byte("alpha\n"))
	planner := newTestPlanner(f.game, &installerState{}, &Manifest{ID: "New", Name: "New"})
	if _, err := planner.plan("resource.yaml"); err != nil {
		t.Fatal(err)
	}
	_, err := planner.plan("./resource.yaml")
	typed := blitz.AsError(err)
	if typed == nil || typed.Code != "MANIFEST_DUPLICATE_TARGET" || typed.Path != target {
		t.Fatalf("%+v", typed)
	}
}

func TestTargetPlanComposesOperationsInManifestOrder(t *testing.T) {
	f := newPlanFixture(t)
	f.write("resource.yaml", []byte("start middle end\n"))
	manifest := &Manifest{ID: "New", Name: "New", Files: []ModFile{
		planReplace("resource.yaml", "middle", "center"),
		{Name: "resource.yaml", Action: ActionBefore, Find: planString("center"), Replace: planString("before ")},
		{Name: "resource.yaml", Action: ActionAfter, Find: planString("end"), Replace: planString(" after")},
	}}
	plan, err := newTestPlanner(f.game, &installerState{}, manifest).plan("resource.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.added) != 3 || len(plan.operations) != 3 {
		t.Fatalf("added=%d operations=%d", len(plan.added), len(plan.operations))
	}
	if expected := "start before center end after\n"; string(plan.output) != expected {
		t.Fatalf("output=%q", plan.output)
	}
}
