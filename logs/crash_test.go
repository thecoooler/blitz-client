package logs

import (
	"path/filepath"
	"testing"
	"time"
)

func TestListCrashesFiltersClientReports(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Game-2026-11-12-194553.ips", crashReport(
		`{"bundleID":"com.tanksblitz","app_version":"26.10.1","timestamp":"2026-11-12 19:45:53.00 +0300"}`,
		`{"osVersion":{"train":"macOS 15.7.9"},"bundleInfo":{"CFBundleIdentifier":"com.tanksblitz"},`+
			`"exception":{"type":"EXC_BAD_ACCESS","signal":"SIGSEGV"},`+
			`"termination":{"indicator":"Segmentation fault: 11"}}`,
	))
	writeFile(t, dir, "Game-2026-11-12-200000.ips", crashReport(
		`{"bundleID":"com.tanksblitz","app_version":"26.10.2","timestamp":"2026-11-12 20:00:00.00 +0300"}`,
		`{"bundleInfo":{"CFBundleIdentifier":"com.tanksblitz"},"exception":{"type":"EXC_CRASH"}}`,
	))
	writeFile(t, dir, "Other-2026-11-12-210000.ips", crashReport(
		`{"bundleID":"com.example","app_version":"1.0","timestamp":"2026-11-12 21:00:00.00 +0300"}`,
		`{"bundleInfo":{"CFBundleIdentifier":"com.example"}}`,
	))
	writeFile(t, dir, "Broken.ips", "not json\nnor this")
	writeFile(t, dir, "notes.txt", "ignored")

	crashes, err := ListCrashes(dir, 0)
	if err != nil {
		t.Fatalf("ListCrashes: %v", err)
	}
	if len(crashes) != 2 {
		t.Fatalf("got %d crashes, want 2: %+v", len(crashes), crashes)
	}
	if crashes[0].Version != "26.10.2" {
		t.Errorf("newest crash is %+v", crashes[0])
	}
	crashed := crashes[1]
	if want := time.Date(2026, 11, 12, 19, 45, 53, 0, time.FixedZone("", 3*3600)); !crashed.Time.Equal(want) {
		t.Errorf("crash time is %v, want %v", crashed.Time, want)
	}
	if crashed.Exception != "EXC_BAD_ACCESS (SIGSEGV)" {
		t.Errorf("exception is %q", crashed.Exception)
	}
	if crashed.Reason != "Segmentation fault: 11" {
		t.Errorf("reason is %q", crashed.Reason)
	}
	if crashed.OS != "macOS 15.7.9" {
		t.Errorf("system is %q", crashed.OS)
	}
	if crashed.Path != filepath.Join(dir, "Game-2026-11-12-194553.ips") {
		t.Errorf("path is %q", crashed.Path)
	}

	limited, err := ListCrashes(dir, 1)
	if err != nil {
		t.Fatalf("ListCrashes limited: %v", err)
	}
	if len(limited) != 1 || limited[0].Version != "26.10.2" {
		t.Fatalf("limited crashes are %+v", limited)
	}
}

func TestListCrashesAcceptsIdentifierFromBody(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Game.ips", crashReport(
		`{"timestamp":"2026-11-12 19:45:53.00 +0300"}`,
		`{"bundleInfo":{"CFBundleIdentifier":"com.tanksblitz"}}`,
	))
	crashes, err := ListCrashes(dir, 0)
	if err != nil {
		t.Fatalf("ListCrashes: %v", err)
	}
	if len(crashes) != 1 {
		t.Fatalf("got %d crashes, want 1", len(crashes))
	}
}

func TestListCrashesMissingDirectoryIsEmpty(t *testing.T) {
	crashes, err := ListCrashes(filepath.Join(t.TempDir(), "absent"), 0)
	if err != nil {
		t.Fatalf("ListCrashes: %v", err)
	}
	if len(crashes) != 0 {
		t.Fatalf("got %d crashes, want none", len(crashes))
	}
}

func crashReport(header, body string) string {
	return header + "\n" + body + "\n"
}
