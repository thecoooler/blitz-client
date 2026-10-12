package mods

import "strings"

type span struct{ start, end int }

// findMatches matches normalized text but returns ranges in the untouched
// resource. Only spaces/tabs at line ends and CRLF line endings are normalized.
func findMatches(source, find string, ignoreTrailing bool) []span {
	if !ignoreTrailing {
		var out []span
		for offset := 0; offset <= len(source); {
			index := strings.Index(source[offset:], find)
			if index < 0 {
				break
			}
			start := offset + index
			out = append(out, span{start, start + len(find)})
			offset = start + max(len(find), 1)
		}
		return out
	}
	sourceText, spans := normalize(source)
	findText, _ := normalize(find)
	if findText == "" {
		return nil
	}
	// Block searches must retain their original indentation. A trimmed final
	// space is only optional at a line end, never inside another token.
	lineStart := strings.Contains(findText, "\n") || strings.HasPrefix(findText, " ") || strings.HasPrefix(findText, "\t")
	lineEnd := strings.HasSuffix(find, " ") || strings.HasSuffix(find, "\t")
	var out []span
	for offset := 0; offset <= len(sourceText); {
		index := strings.Index(sourceText[offset:], findText)
		if index < 0 {
			break
		}
		start := offset + index
		end := start + len(findText)
		offset = end
		if lineStart && start != 0 && sourceText[start-1] != '\n' {
			continue
		}
		if lineEnd && end != len(sourceText) && sourceText[end] != '\n' {
			continue
		}
		out = append(out, span{spans[start].start, spans[end-1].end})
	}
	return out
}

func normalize(text string) (string, []span) {
	var normalized strings.Builder
	normalized.Grow(len(text))
	spans := make([]span, 0, len(text))
	offset := 0
	for _, line := range splitInclusive(text) {
		hasNewline := strings.HasSuffix(line, "\n")
		content := line
		if hasNewline {
			content = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		}
		trimmed := strings.TrimRight(content, " \t")
		normalized.WriteString(trimmed)
		for index := range len(trimmed) {
			end := index + 1
			if index+1 == len(trimmed) {
				end = len(content)
			}
			spans = append(spans, span{offset + index, offset + end})
		}
		if hasNewline {
			normalized.WriteByte('\n')
			spans = append(spans, span{offset + len(trimmed), offset + len(line)})
		}
		offset += len(line)
	}
	return normalized.String(), spans
}

// splitInclusive splits after each "\n", keeping the separator, like
// str::split_inclusive.
func splitInclusive(text string) []string {
	var out []string
	for text != "" {
		index := strings.IndexByte(text, '\n')
		if index < 0 {
			out = append(out, text)
			break
		}
		out = append(out, text[:index+1])
		text = text[index+1:]
	}
	return out
}
