package mods

import (
	"fmt"
	"strings"
	"unicode/utf8"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/rs"
)

type lineKind int

const (
	kindContext lineKind = iota
	kindRemove
	kindAdd
)

type hunkLine struct {
	kind      lineKind
	text      string
	noNewline bool
}

type hunk struct {
	oldStart, oldLen, newLen int
	lines                    []hunkLine
}

type filePatch struct {
	path string
	diff string
}

func patchError(code, message string) *blitz.Error { return blitz.NewError(code, message) }

var unsupportedPrefixes = []string{
	"new file mode ", "deleted file mode ", "rename from ", "rename to ", "copy from ", "copy to ",
	"old mode ", "new mode ", "Binary files ", "GIT binary patch",
}

// splitPatch splits a multi-file Git/unified diff, validating its paths and
// hunk framing.
func splitPatch(diff string) ([]filePatch, error) {
	rawLines := splitInclusive(diff)
	lines := make([]string, len(rawLines))
	offsets := []int{0}
	for index, line := range rawLines {
		lines[index] = strings.TrimRight(strings.TrimSuffix(line, "\n"), "\r")
		offsets = append(offsets, offsets[len(offsets)-1]+len(line))
	}
	var sections []filePatch
	start := -1
	var paths *[2]string
	for index := 0; index < len(lines); {
		line := lines[index]
		switch {
		case strings.HasPrefix(line, "diff --git "):
			if start >= 0 {
				section, err := makeFilePatch(diff[offsets[start]:offsets[index]], paths)
				if err != nil {
					return nil, err
				}
				sections = append(sections, section)
				paths = nil
			}
			start = index
		case strings.HasPrefix(line, "--- ") && index+1 < len(lines) && strings.HasPrefix(lines[index+1], "+++ "):
			if paths != nil {
				section, err := makeFilePatch(diff[offsets[start]:offsets[index]], paths)
				if err != nil {
					return nil, err
				}
				sections = append(sections, section)
				paths = nil
				start = index
			} else if start < 0 {
				start = index
			}
			oldPath, err := decodeHeaderPath(line[4:])
			if err != nil {
				return nil, err
			}
			newPath, err := decodeHeaderPath(lines[index+1][4:])
			if err != nil {
				return nil, err
			}
			paths = &[2]string{oldPath, newPath}
			index++
		case strings.HasPrefix(line, "@@ "):
			if paths == nil {
				return nil, patchError("PATCH_INVALID", "hunk has no ---/+++ file headers")
			}
			_, oldCount, newCount, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			index++
			for oldCount != 0 || newCount != 0 {
				if index >= len(lines) {
					return nil, patchError("PATCH_INVALID", "truncated hunk")
				}
				row := lines[index]
				consumeOld, consumeNew := 0, 0
				switch {
				case strings.HasPrefix(row, " "):
					consumeOld, consumeNew = 1, 1
				case strings.HasPrefix(row, "-"):
					consumeOld = 1
				case strings.HasPrefix(row, "+"):
					consumeNew = 1
				case strings.HasPrefix(row, `\ No newline at end of file`):
				default:
					return nil, patchError("PATCH_INVALID", "unexpected line in hunk")
				}
				if oldCount < consumeOld {
					return nil, patchError("PATCH_INVALID", "too many old lines in hunk")
				}
				if newCount < consumeNew {
					return nil, patchError("PATCH_INVALID", "too many new lines in hunk")
				}
				oldCount -= consumeOld
				newCount -= consumeNew
				index++
			}
			continue
		default:
			for _, prefix := range unsupportedPrefixes {
				if strings.HasPrefix(line, prefix) {
					return nil, patchError("PATCH_UNSUPPORTED_CHANGE", "only modifications to existing text resources are supported")
				}
			}
		}
		index++
	}
	if start >= 0 {
		section, err := makeFilePatch(diff[offsets[start]:], paths)
		if err != nil {
			return nil, err
		}
		sections = append(sections, section)
	}
	if len(sections) == 0 {
		return nil, patchError("PATCH_EMPTY", "patch contains no file changes")
	}
	return sections, nil
}

func makeFilePatch(diff string, paths *[2]string) (filePatch, error) {
	if paths == nil {
		return filePatch{}, patchError("PATCH_INVALID", "missing ---/+++ file headers")
	}
	oldPath, newPath := paths[0], paths[1]
	if oldPath == "/dev/null" || newPath == "/dev/null" {
		return filePatch{}, patchError("PATCH_UNSUPPORTED_CHANGE", "file additions and deletions are unsupported")
	}
	if strings.HasPrefix(oldPath, "a/") && strings.HasPrefix(newPath, "b/") {
		oldPath, newPath = oldPath[2:], newPath[2:]
	}
	if oldPath != newPath {
		return filePatch{}, patchError("PATCH_UNSUPPORTED_CHANGE", "file renames are unsupported")
	}
	target, err := safeRelativePath(newPath)
	if err != nil {
		return filePatch{}, patchError("PATCH_INVALID_TARGET", err.Error())
	}
	if strings.HasSuffix(strings.ToLower(newPath), ".dvpl") {
		return filePatch{}, patchError("PATCH_DVPL_INPUT", "patch paths must refer to unpacked resources, without .dvpl")
	}
	hunks, err := parseHunks(diff)
	if err != nil {
		return filePatch{}, err
	}
	if len(hunks) == 0 {
		return filePatch{}, patchError("PATCH_EMPTY", "file section contains no hunks")
	}
	return filePatch{path: target, diff: diff}, nil
}

func decodeHeaderPath(header string) (string, error) {
	if !strings.HasPrefix(header, `"`) {
		before, _, _ := strings.Cut(header, "\t")
		return before, nil
	}
	// Git quotes paths containing special characters using C escapes and octal bytes.
	var out []byte
	raw := []byte(header)
	for index := 1; index < len(raw); index++ {
		switch raw[index] {
		case '"':
			if !utf8.Valid(out) {
				return "", patchError("PATCH_INVALID_TARGET", rs.UTF8Error(out))
			}
			return string(out), nil
		case '\\':
			index++
			if index >= len(raw) {
				return "", patchError("PATCH_INVALID_TARGET", "incomplete quoted path")
			}
			escaped := raw[index]
			var b byte
			switch escaped {
			case '0', '1', '2', '3', '4', '5', '6', '7':
				if index+3 > len(raw) {
					return "", patchError("PATCH_INVALID_TARGET", "invalid octal path escape")
				}
				value := 0
				for _, digit := range raw[index : index+3] {
					if digit < '0' || digit > '7' {
						return "", patchError("PATCH_INVALID_TARGET", "invalid octal path escape")
					}
					value = value*8 + int(digit-'0')
				}
				if value > 255 {
					return "", patchError("PATCH_INVALID_TARGET", "invalid octal path escape")
				}
				index += 2
				b = byte(value)
			case 't':
				b = '\t'
			case 'n':
				b = '\n'
			case 'r':
				b = '\r'
			case 'a':
				b = 7
			case 'b':
				b = 8
			case 'f':
				b = 12
			case 'v':
				b = 11
			case '\\', '"':
				b = escaped
			default:
				return "", patchError("PATCH_INVALID_TARGET", "unknown quoted path escape")
			}
			out = append(out, b)
		default:
			out = append(out, raw[index])
		}
	}
	return "", patchError("PATCH_INVALID_TARGET", "unterminated quoted path")
}

// applyUnifiedDiff applies a single-file unified diff. Context is exact and
// every hunk must apply.
func applyUnifiedDiff(source, diff string) (string, error) {
	files := 0
	for _, line := range rs.Lines(diff) {
		if strings.HasPrefix(line, "diff --git ") {
			files++
		}
	}
	if files > 1 {
		return "", patchError("PATCH_MULTIPLE_FILES", "each patch operation must reference a diff for exactly one resource")
	}
	hunks, err := parseHunks(diff)
	if err != nil {
		return "", err
	}
	if len(hunks) == 0 {
		return "", patchError("PATCH_EMPTY", "unified diff contains no hunks")
	}
	newline := "\n"
	if strings.Contains(source, "\r\n") {
		newline = "\r\n"
	}
	lines := splitInclusive(source)
	cursor, offset := 0, 0

	for _, h := range hunks {
		var oldLines []hunkLine
		for _, line := range h.lines {
			if line.kind != kindAdd {
				oldLines = append(oldLines, line)
			}
		}
		originalStart := max(h.oldStart-1, 0)
		if h.oldLen == 0 {
			originalStart = h.oldStart
		}
		expected := max(originalStart+offset, cursor, 0)

		var start int
		if len(oldLines) == 0 {
			if expected > len(lines) {
				return "", patchError("PATCH_CONTEXT_MISMATCH",
					fmt.Sprintf("insertion hunk points past end of file at line %d", expected+1))
			}
			start = expected
		} else {
			matches := findLineMatches(lines, oldLines, cursor)
			switch {
			case containsInt(matches, expected):
				start = expected
			case len(matches) == 1:
				start = matches[0]
			case len(matches) == 0:
				return "", patchError("PATCH_CONTEXT_MISMATCH",
					fmt.Sprintf("could not match diff hunk near source line %d", expected+1))
			default:
				return "", patchError("PATCH_CONTEXT_AMBIGUOUS",
					fmt.Sprintf("diff hunk near line %d matches multiple locations", expected+1))
			}
		}
		offset = start - originalStart + h.newLen - h.oldLen

		consumed := 0
		var replacement []string
		sourceIndex := start
		for _, line := range h.lines {
			switch line.kind {
			case kindContext, kindRemove:
				what := "diff context runs past end of file"
				differs := "context differs near source line %d"
				if line.kind == kindRemove {
					what = "removed line runs past end of file"
					differs = "removed line differs near source line %d"
				}
				if sourceIndex >= len(lines) {
					return "", patchError("PATCH_CONTEXT_MISMATCH", what)
				}
				existing := lines[sourceIndex]
				if lineContent(existing) != line.text {
					return "", patchError("PATCH_CONTEXT_MISMATCH", fmt.Sprintf(differs, sourceIndex+1))
				}
				if line.kind == kindContext {
					replacement = append(replacement, existing)
				}
				sourceIndex++
				consumed++
			case kindAdd:
				added := line.text
				if !line.noNewline {
					added += newline
				}
				replacement = append(replacement, added)
			}
		}
		if consumed != h.oldLen {
			return "", patchError("PATCH_INVALID", fmt.Sprintf("hunk declares %d old lines but contains %d", h.oldLen, consumed))
		}
		if len(replacement) != h.newLen {
			return "", patchError("PATCH_INVALID", fmt.Sprintf("hunk declares %d new lines but contains %d", h.newLen, len(replacement)))
		}
		lines = append(lines[:start], append(replacement, lines[start+consumed:]...)...)
		cursor = start + h.newLen
	}
	return strings.Join(lines, ""), nil
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func parseHunks(diff string) ([]hunk, error) {
	var hunks []hunk
	raw := splitInclusive(diff)
	lines := make([]string, len(raw))
	for index, line := range raw {
		lines[index] = strings.TrimSuffix(line, "\n")
	}
	for index := 0; index < len(lines); {
		if !strings.HasPrefix(lines[index], "@@ ") {
			index++
			continue
		}
		oldStart, oldLen, newLen, err := parseHunkHeader(lines[index])
		if err != nil {
			return nil, err
		}
		index++
		var body []hunkLine
	body:
		for index < len(lines) {
			row := lines[index]
			if strings.HasPrefix(row, "@@ ") || strings.HasPrefix(row, "diff --git ") {
				break
			}
			if strings.HasPrefix(row, `\ No newline at end of file`) {
				if len(body) > 0 {
					body[len(body)-1].noNewline = true
				}
				index++
				continue
			}
			if row == "" {
				return nil, patchError("PATCH_INVALID", "empty line in unified diff hunk")
			}
			var kind lineKind
			switch row[0] {
			case ' ':
				kind = kindContext
			case '-':
				kind = kindRemove
			case '+':
				kind = kindAdd
			default:
				break body
			}
			body = append(body, hunkLine{kind: kind, text: row[1:]})
			index++
		}
		actualOld, actualNew := 0, 0
		for _, row := range body {
			if row.kind != kindAdd {
				actualOld++
			}
			if row.kind != kindRemove {
				actualNew++
			}
		}
		if actualOld != oldLen || actualNew != newLen {
			return nil, patchError("PATCH_INVALID", fmt.Sprintf(
				"hunk header declares -%d/+%d lines but body has -%d/+%d", oldLen, newLen, actualOld, actualNew))
		}
		hunks = append(hunks, hunk{oldStart: oldStart, oldLen: oldLen, newLen: newLen, lines: body})
	}
	return hunks, nil
}

func parseHunkHeader(line string) (int, int, int, error) {
	end := strings.Index(line[3:], " @@")
	if end < 0 {
		return 0, 0, 0, patchError("PATCH_INVALID", "malformed unified diff hunk header")
	}
	specs := strings.Fields(line[3 : end+3])
	if len(specs) < 1 || !strings.HasPrefix(specs[0], "-") {
		return 0, 0, 0, patchError("PATCH_INVALID", "hunk header has no old range")
	}
	if len(specs) < 2 || !strings.HasPrefix(specs[1], "+") {
		return 0, 0, 0, patchError("PATCH_INVALID", "hunk header has no new range")
	}
	oldStart, oldLen, err := parseRange(specs[0][1:])
	if err != nil {
		return 0, 0, 0, err
	}
	_, newLen, err := parseRange(specs[1][1:])
	if err != nil {
		return 0, 0, 0, err
	}
	return oldStart, oldLen, newLen, nil
}

func parseRange(value string) (int, int, error) {
	parts := strings.Split(value, ",")
	start, err := rs.ParseUint(parts[0], 64)
	if err != nil {
		return 0, 0, patchError("PATCH_INVALID", "invalid hunk line number")
	}
	length := uint64(1)
	if len(parts) > 1 {
		if length, err = rs.ParseUint(parts[1], 64); err != nil {
			return 0, 0, patchError("PATCH_INVALID", "invalid hunk line count")
		}
	}
	if len(parts) > 2 {
		return 0, 0, patchError("PATCH_INVALID", "invalid hunk range")
	}
	return int(start), int(length), nil
}

func findLineMatches(lines []string, oldLines []hunkLine, from int) []int {
	if len(oldLines) == 0 || len(oldLines) > len(lines) {
		return nil
	}
	var out []int
	for start := from; start <= len(lines)-len(oldLines); start++ {
		matched := true
		for offset, expected := range oldLines {
			if lineContent(lines[start+offset]) != expected.text {
				matched = false
				break
			}
		}
		if matched {
			out = append(out, start)
		}
	}
	return out
}

func lineContent(line string) string { return strings.TrimSuffix(line, "\n") }
