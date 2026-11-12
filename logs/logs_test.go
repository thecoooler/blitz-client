package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListFindsLaunchesNewestFirst(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "blitz-logs_20261112194314.txt", "")
	writeFile(t, dir, "blitz-logs_20261113001547.txt", "")
	writeFile(t, dir, "blitz-logs_20261112205724.txt", "")
	writeFile(t, dir, "dlc_manager_20261112194320.log", "")
	writeFile(t, dir, "notes.txt", "")
	if err := os.Mkdir(filepath.Join(dir, "blitz-logs_20250101000000.txt"), 0o755); err != nil {
		t.Fatal(err)
	}

	files, err := List(dir, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("got %d journals, want 3", len(files))
	}
	want := []string{
		"blitz-logs_20261113001547.txt",
		"blitz-logs_20261112205724.txt",
		"blitz-logs_20261112194314.txt",
	}
	for index, name := range want {
		if files[index].Name != name {
			t.Errorf("journal %d is %q, want %q", index, files[index].Name, name)
		}
	}
	wantStarted := time.Date(2026, 11, 13, 0, 15, 47, 0, time.Local)
	if !files[0].Started.Equal(wantStarted) {
		t.Errorf("started is %v, want %v", files[0].Started, wantStarted)
	}

	limited, err := List(dir, 2)
	if err != nil {
		t.Fatalf("List limited: %v", err)
	}
	if len(limited) != 2 || limited[1].Name != want[1] {
		t.Fatalf("limited list is %+v", limited)
	}
}

func TestListMissingDirectoryIsEmpty(t *testing.T) {
	files, err := List(filepath.Join(t.TempDir(), "absent"), 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("got %d journals, want none", len(files))
	}
}

func TestReadGroupsContinuationLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blitz-logs_20261112194314.txt")
	writeFile(t, dir, filepath.Base(path), ""+
		"12:00:00.001Z [I] [engine] start\r\n"+
		"12:00:01.000Z [W] [ui] public\n"+
		"  detail line\n"+
		"\n"+
		"12:00:02.000Z [E] [render] Failed thing\n"+
		"12:00:03.000Z [D] [base] debug\n")

	log, err := Read(path, 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(log.Entries) != 4 {
		t.Fatalf("got %d entries, want 4: %+v", len(log.Entries), log.Entries)
	}
	if got := log.Entries[1].Text; !strings.Contains(got, "detail line") || strings.Count(got, "\n") != 1 {
		t.Errorf("continuation line is not attached: %q", got)
	}
	if log.Entries[1].Level != LevelWarning || log.Entries[1].Time != "12:00:01.000Z" {
		t.Errorf("warning header parsed as %+v", log.Entries[1])
	}
	if log.Entries[2].Level != LevelError {
		t.Errorf("error level parsed as %v", log.Entries[2].Level)
	}
	if log.Entries[3].Level != LevelInfo {
		t.Errorf("unknown level parsed as %v", log.Entries[3].Level)
	}
	counts := log.Counts()
	if counts != (Counts{Entries: 4, Errors: 1, Warnings: 1}) {
		t.Errorf("counts are %+v", counts)
	}
}

func TestSelectFiltersLevelAndQuery(t *testing.T) {
	log := Log{Entries: []Entry{
		{Level: LevelInfo, Text: "12:00:00.001Z [I] start"},
		{Level: LevelWarning, Text: "12:00:01.000Z [W] Render pass limit"},
		{Level: LevelError, Text: "12:00:02.000Z [E] Failed to load mod"},
	}}
	selected := log.Select(LevelWarning, "", 0)
	if selected.Matched != 2 || selected.Shown != 2 || selected.Truncated {
		t.Fatalf("warning selection is %+v", selected)
	}
	if strings.Contains(selected.Text, "[I]") {
		t.Errorf("info entry leaked into warnings: %q", selected.Text)
	}

	searched := log.Select(LevelInfo, "FAILED", 0)
	if searched.Matched != 1 || !strings.Contains(searched.Text, "Failed to load mod") {
		t.Fatalf("search selection is %+v", searched)
	}

	limited := log.Select(LevelInfo, "", 1)
	if limited.Matched != 3 || limited.Shown != 1 || !limited.Truncated {
		t.Fatalf("limited selection is %+v", limited)
	}
}

func TestReadDropsPartialFirstLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blitz-logs_20261112194314.txt")
	first := "01:00:00.000Z [I] first line with padding\n"
	second := "01:00:01.000Z [E] second line\n"
	writeFile(t, dir, filepath.Base(path), first+second)

	log, err := Read(path, int64(len(second)+5))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !log.Truncated {
		t.Fatal("truncated log was not reported")
	}
	if len(log.Entries) != 1 || !strings.Contains(log.Entries[0].Text, "second line") {
		t.Fatalf("entries are %+v", log.Entries)
	}
}

func TestSessionStartedRejectsForeignNames(t *testing.T) {
	for _, name := range []string{
		"blitz-logs_20261112194314.log",
		"blitz-logs_2026111219431.txt",
		"blitz-logs_20261312194314.txt",
		"blitz-logs_.txt",
		"dlc_manager_20261112194314.log",
	} {
		if _, ok := sessionStarted(name); ok {
			t.Errorf("%q was accepted as a launch journal", name)
		}
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
