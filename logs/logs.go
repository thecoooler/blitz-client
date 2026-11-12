// Package logs locates the client's own journals and crash reports and parses
// their line structure. It never interprets game state: callers receive the
// original text grouped by journal entry.
package logs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	blitz "github.com/thecoooler/blitz-client"
)

const (
	// MaxReadBytes is the default cap on a single journal read. The newest
	// portion of a larger journal is read and a partial first line is dropped.
	MaxReadBytes int64 = 4 << 20

	sessionPrefix = "blitz-logs_"
	sessionSuffix = ".txt"
	timestampLen  = len("15:04:05.000Z")
)

// Level is the severity printed in a journal entry header.
type Level uint8

const (
	LevelInfo Level = iota + 1
	LevelWarning
	LevelError
)

// String returns the journal's one-letter severity marker.
func (l Level) String() string {
	switch l {
	case LevelWarning:
		return "W"
	case LevelError:
		return "E"
	default:
		return "I"
	}
}

// File is one journal written by a client launch.
type File struct {
	Path     string
	Name     string
	Started  time.Time
	Modified time.Time
	Size     int64
}

// List returns the journals in dir, newest launch first. A missing directory
// means the client has never written a journal and yields an empty list.
func List(dir string, limit int) ([]File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []File{}, nil
		}
		return nil, blitz.IOError("could not read the journal directory", dir, err)
	}
	files := []File{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		started, ok := sessionStarted(entry.Name())
		if !ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, File{
			Path:     filepath.Join(dir, entry.Name()),
			Name:     entry.Name(),
			Started:  started,
			Modified: info.ModTime(),
			Size:     info.Size(),
		})
	}
	slices.SortFunc(files, func(a, b File) int {
		if order := b.Started.Compare(a.Started); order != 0 {
			return order
		}
		return strings.Compare(b.Name, a.Name)
	})
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}
	return files, nil
}

// sessionStarted reads the launch time from a journal name. The client names
// journals after local time, not the UTC timestamps inside them.
func sessionStarted(name string) (time.Time, bool) {
	if !strings.HasPrefix(name, sessionPrefix) || !strings.HasSuffix(name, sessionSuffix) {
		return time.Time{}, false
	}
	stamp := name[len(sessionPrefix) : len(name)-len(sessionSuffix)]
	if len(stamp) != len("20060102150405") {
		return time.Time{}, false
	}
	started, err := time.ParseInLocation("20060102150405", stamp, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return started, true
}

// Entry is one journal message. Lines without their own header, such as the
// lines of a multi-line block, stay attached to the preceding entry.
type Entry struct {
	Level Level
	Time  string
	Text  string
}

// Log is the newest readable part of one journal.
type Log struct {
	Path      string
	Entries   []Entry
	Truncated bool
}

// Counts summarizes the entries in a log.
type Counts struct {
	Entries  int
	Errors   int
	Warnings int
}

// Counts returns the number of entries and how many are errors or warnings.
func (l Log) Counts() Counts {
	counts := Counts{Entries: len(l.Entries)}
	for _, entry := range l.Entries {
		switch entry.Level {
		case LevelError:
			counts.Errors++
		case LevelWarning:
			counts.Warnings++
		}
	}
	return counts
}

// Selection is the rendered subset of a log.
type Selection struct {
	Text      string
	Matched   int
	Shown     int
	Truncated bool
}

// Select keeps entries at or above min whose text contains query,
// case-insensitively. A positive limit caps the rendered entries while Matched
// still reports every match.
func (l Log) Select(min Level, query string, limit int) Selection {
	needle := strings.ToLower(query)
	matched := 0
	lines := []string{}
	for _, entry := range l.Entries {
		if entry.Level < min {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(entry.Text), needle) {
			continue
		}
		matched++
		if limit > 0 && len(lines) >= limit {
			continue
		}
		lines = append(lines, entry.Text)
	}
	return Selection{
		Text:      strings.Join(lines, "\n"),
		Matched:   matched,
		Shown:     len(lines),
		Truncated: limit > 0 && matched > len(lines),
	}
}

// Read returns the newest maxBytes of a journal, or MaxReadBytes when maxBytes
// is not positive. The file is opened only for this call and no handle stays
// open, so the client can keep appending to the same journal.
func Read(path string, maxBytes int64) (Log, error) {
	if maxBytes <= 0 {
		maxBytes = MaxReadBytes
	}
	file, err := os.Open(path)
	if err != nil {
		return Log{}, blitz.IOError("could not open the journal", path, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Log{}, blitz.IOError("could not read the journal", path, err)
	}
	truncated := info.Size() > maxBytes
	if truncated {
		if _, err := file.Seek(info.Size()-maxBytes, io.SeekStart); err != nil {
			return Log{}, blitz.IOError("could not read the journal", path, err)
		}
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return Log{}, blitz.IOError("could not read the journal", path, err)
	}
	if truncated {
		if index := bytes.IndexByte(data, '\n'); index >= 0 {
			data = data[index+1:]
		} else {
			data = nil
		}
	}
	text := strings.ToValidUTF8(string(data), "\uFFFD")
	return Log{Path: path, Entries: parseEntries(text), Truncated: truncated}, nil
}

func parseEntries(text string) []Entry {
	entries := []Entry{}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSuffix(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		stamp, level, ok := splitEntry(line)
		if !ok {
			if len(entries) == 0 {
				entries = append(entries, Entry{Level: LevelInfo, Text: line})
			} else {
				entries[len(entries)-1].Text += "\n" + line
			}
			continue
		}
		entries = append(entries, Entry{Level: level, Time: stamp, Text: line})
	}
	return entries
}

// splitEntry recognizes the "HH:MM:SS.mmmZ [L] " header the client writes for
// each new journal entry. Continuation and preamble lines fail the check.
func splitEntry(line string) (string, Level, bool) {
	if len(line) < timestampLen+4 {
		return "", LevelInfo, false
	}
	for _, index := range [...]int{0, 1, 3, 4, 6, 7, 9, 10, 11} {
		if line[index] < '0' || line[index] > '9' {
			return "", LevelInfo, false
		}
	}
	if line[2] != ':' || line[5] != ':' || line[8] != '.' || line[12] != 'Z' ||
		line[13] != ' ' || line[14] != '[' || line[16] != ']' {
		return "", LevelInfo, false
	}
	level := LevelInfo
	switch line[15] {
	case 'W':
		level = LevelWarning
	case 'E':
		level = LevelError
	}
	return line[:timestampLen], level, true
}
