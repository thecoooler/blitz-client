package replay

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveMetadataCacheLifecycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, metaEntry)
	var cache liveMetadataCache
	check := func(arena uint64, name string, id uint32) {
		t.Helper()
		got := cache.snapshot(dir, arena)
		if name == "" && id == 0 {
			if got != nil {
				t.Fatalf("expected no metadata, got %+v", got)
			}
			return
		}
		if got == nil || got.Name != name || (id == 0 && got.ID != nil) || (id != 0 && (got.ID == nil || *got.ID != id)) {
			t.Fatalf("metadata: %+v, want %q %d", got, name, id)
		}
	}
	write := func(data string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	check(1, "", 0)
	write(`{"arenaUniqueId":1,"mapId":`)
	check(1, "", 0)
	if cache.entry != nil {
		t.Fatal("partial metadata was cached")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString(`20,"mapName":"first"}`); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	check(1, "first", 20)
	first := cache.entry
	snapshot := cache.snapshot(dir, 1)
	*snapshot.ID = 999
	snapshot.Name = "changed"
	check(1, "first", 20)
	if cache.entry != first {
		t.Fatal("unchanged metadata was read again")
	}
	check(2, "", 0)
	check(1, "first", 20)
	if cache.entry != first {
		t.Fatal("arena change needlessly discarded metadata")
	}
	write(`{"arenaUniqueId":2,"mapId":21,"mapName":"second"}`)
	check(1, "", 0)
	check(2, "second", 21)
	// Replacement must be detected even when size and mtime match.
	old, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	replacement := filepath.Join(dir, "new")
	if err := os.WriteFile(replacement, []byte(`{"arenaUniqueId":2,"mapId":22,"mapName":"third!"}`), 0600); err != nil {
		t.Fatal(err)
	}
	replacementInfo, err := os.Stat(replacement)
	if err != nil {
		t.Fatal(err)
	}
	if old.Size() != replacementInfo.Size() {
		t.Fatal("test requires equal-size replacement")
	}
	if err := os.Chtimes(replacement, old.ModTime(), old.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	check(2, "third!", 22)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	check(2, "", 0)
	if cache.entry != nil {
		t.Fatal("deleted metadata retained")
	}
	write(`{"arenaUniqueId":2,"mapId":23,"mapName":"return"}`)
	check(2, "return", 23)
}

func TestLiveMetadataRefreshesEvenWhenAttributesUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, metaEntry)
	before := `{"arenaUniqueId":1,"mapId":20,"mapName":"first"}`
	after := `{"arenaUniqueId":1,"mapId":21,"mapName":"other"}`
	if err := os.WriteFile(path, []byte(before), 0600); err != nil {
		t.Fatal(err)
	}
	var cache liveMetadataCache
	if cache.snapshot(dir, 1) == nil {
		t.Fatal("initial read")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(after), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	cache.readAt = time.Now().Add(-metadataRefreshInterval)
	result := cache.snapshot(dir, 1)
	if result == nil || result.Name != "other" || result.ID == nil || *result.ID != 21 {
		t.Fatalf("refresh: %+v", result)
	}
}

func TestLiveMetadataZeroMapIDUsesCatalog(t *testing.T) {
	dir := t.TempDir()
	w := LiveReplayWatcher{mapCatalog: map[uint32]string{20: "Каталог"}}
	r := liveRecording{path: dir, players: livePlayers{hasArena: true, arenaID: 1, arenaTypeID: 20}}
	for _, tc := range []struct {
		data, name, display string
		id                  uint32
	}{
		{`{"arenaUniqueId":1,"mapId":0,"mapName":"internal"}`, "internal", "Каталог", 20},
		{`{"arenaUniqueId":1,"mapId":0}`, "", "Каталог", 20},
		{`{"arenaUniqueId":1,"mapId":21,"mapName":"different"}`, "different", "", 21},
	} {
		if err := os.WriteFile(filepath.Join(dir, metaEntry), []byte(tc.data), 0600); err != nil {
			t.Fatal(err)
		}
		state := LiveReplayState{}
		w.enrichLiveState(&state, &r)
		if state.Map == nil || state.Map.Name != tc.name || state.Map.DisplayName != tc.display || state.Map.ID == nil || *state.Map.ID != tc.id {
			t.Fatalf("%s: %+v", tc.data, state.Map)
		}
	}
}
