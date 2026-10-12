package client

import (
	"encoding/json"
	"strings"
	"unicode"

	"github.com/thecoooler/blitz-client/internal/lossy"
	"github.com/thecoooler/blitz-client/internal/rs"
)

func readLocalizedNames(dataDir string) (map[string]string, error) {
	names := make(map[string]string)
	path := ResolveClientAsset(dataDir, "Strings/ru.yaml")
	if path == "" {
		return names, nil
	}
	data, err := ReadClientAsset(path)
	if err != nil {
		return nil, err
	}
	for _, line := range rs.Lines(lossy.String(data)) {
		if key, value, ok := parseQuotedYAMLPair(strings.TrimLeftFunc(line, unicode.IsSpace)); ok {
			names[key] = value
		}
	}
	return names, nil
}

func parseQuotedYAMLPair(line string) (string, string, bool) {
	key, consumed, ok := quotedJSONString(line)
	if !ok {
		return "", "", false
	}
	rest, ok := strings.CutPrefix(strings.TrimLeftFunc(line[consumed:], unicode.IsSpace), ":")
	if !ok {
		return "", "", false
	}
	rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
	value, consumed, ok := quotedJSONString(rest)
	if !ok || strings.TrimSpace(rest[consumed:]) != "" {
		return "", "", false
	}
	return key, value, true
}

func quotedJSONString(source string) (string, int, bool) {
	if !strings.HasPrefix(source, `"`) {
		return "", 0, false
	}
	escaped := false
	for index := 1; index < len(source); index++ {
		switch {
		case source[index] == '\\' && !escaped:
			escaped = true
		case source[index] == '"' && !escaped:
			var parsed string
			if json.Unmarshal([]byte(source[:index+1]), &parsed) != nil {
				return "", 0, false
			}
			return parsed, index + 1, true
		default:
			escaped = false
		}
	}
	return "", 0, false
}
