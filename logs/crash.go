package logs

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	blitz "github.com/thecoooler/blitz-client"
)

// crashBundleID identifies the game client in operating-system crash reports.
const crashBundleID = "com.tanksblitz"

// Crash is one operating-system crash report for the game client. Crash
// reports carry no reliable symbol names, so the fields describe what the
// system recorded rather than a stack trace.
type Crash struct {
	Path      string
	Time      time.Time
	Version   string
	OS        string
	Exception string
	Reason    string
}

// CrashDir returns the directory where macOS stores crash reports. Other
// systems do not expose a stable user-level location, so callers hide the
// crash section when this returns an error.
func CrashDir() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", blitz.NewError("CRASH_FOLDER_NOT_FOUND", "crash reports are only discovered on macOS")
	}
	home, ok := os.LookupEnv("HOME")
	if !ok {
		return "", blitz.NewError("CRASH_FOLDER_NOT_FOUND", "HOME is not set")
	}
	return filepath.Join(home, "Library/Logs/DiagnosticReports"), nil
}

// ListCrashes returns the client's crash reports in dir, newest first. A
// missing directory yields an empty list, and unrelated or malformed reports
// are skipped.
func ListCrashes(dir string, limit int) ([]Crash, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []Crash{}, nil
		}
		return nil, blitz.IOError("could not read the crash-report directory", dir, err)
	}
	crashes := []Crash{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".ips" {
			continue
		}
		crash, ok := readCrash(filepath.Join(dir, entry.Name()))
		if !ok {
			continue
		}
		crashes = append(crashes, crash)
	}
	slices.SortFunc(crashes, func(a, b Crash) int {
		return b.Time.Compare(a.Time)
	})
	if limit > 0 && len(crashes) > limit {
		crashes = crashes[:limit]
	}
	return crashes, nil
}

func readCrash(path string) (Crash, bool) {
	file, err := os.Open(path)
	if err != nil {
		return Crash{}, false
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	header, err := reader.ReadBytes('\n')
	if err != nil && len(header) == 0 {
		return Crash{}, false
	}
	var summary struct {
		BundleID   string `json:"bundleID"`
		AppVersion string `json:"app_version"`
		Timestamp  string `json:"timestamp"`
	}
	if err := json.Unmarshal(header, &summary); err != nil {
		return Crash{}, false
	}
	// Skip unrelated applications before reading their full report.
	if summary.BundleID != "" && summary.BundleID != crashBundleID {
		return Crash{}, false
	}
	body, err := io.ReadAll(io.LimitReader(reader, 16<<20))
	if err != nil {
		return Crash{}, false
	}
	var report struct {
		OSVersion struct {
			Train string `json:"train"`
			Build string `json:"build"`
		} `json:"osVersion"`
		BundleInfo struct {
			Identifier string `json:"CFBundleIdentifier"`
			Version    string `json:"CFBundleShortVersionString"`
		} `json:"bundleInfo"`
		Exception struct {
			Type   string `json:"type"`
			Signal string `json:"signal"`
		} `json:"exception"`
		Termination struct {
			Indicator string `json:"indicator"`
		} `json:"termination"`
		CaptureTime string `json:"captureTime"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return Crash{}, false
	}
	if !isClientCrash(summary.BundleID, report.BundleInfo.Identifier) {
		return Crash{}, false
	}
	crashTime := parseCrashTime(summary.Timestamp)
	if crashTime.IsZero() {
		crashTime = parseCrashTime(report.CaptureTime)
	}
	version := summary.AppVersion
	if version == "" {
		version = report.BundleInfo.Version
	}
	system := report.OSVersion.Train
	if system == "" {
		system = report.OSVersion.Build
	}
	return Crash{
		Path:      path,
		Time:      crashTime,
		Version:   version,
		OS:        system,
		Exception: exceptionText(report.Exception.Type, report.Exception.Signal),
		Reason:    strings.TrimSpace(report.Termination.Indicator),
	}, true
}

func isClientCrash(identifiers ...string) bool {
	for _, identifier := range identifiers {
		if identifier == crashBundleID {
			return true
		}
	}
	return false
}

func exceptionText(kind, signal string) string {
	kind = strings.TrimSpace(kind)
	signal = strings.TrimSpace(signal)
	switch {
	case kind == "":
		return signal
	case signal == "", strings.Contains(kind, signal):
		return kind
	default:
		return kind + " (" + signal + ")"
	}
}

func parseCrashTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range [...]string{
		"2006-01-02 15:04:05.00 -0700",
		"2006-01-02 15:04:05.9999 -0700",
		"2006-01-02 15:04:05 -0700",
		time.RFC3339,
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
