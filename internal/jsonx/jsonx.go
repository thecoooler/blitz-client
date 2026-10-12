// Package jsonx encodes JSON like serde_json: no HTML escaping and, for
// indented output, two-space indentation without a trailing newline.
package jsonx

import (
	"bytes"
	"encoding/json"
)

// Marshal encodes compact JSON.
func Marshal(value any) ([]byte, error) {
	return encode(value, false)
}

// MarshalIndent encodes pretty JSON with two-space indentation.
func MarshalIndent(value any) ([]byte, error) {
	return encode(value, true)
}

func encode(value any, indent bool) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if indent {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return unescapeLineSeparators(bytes.TrimSuffix(buffer.Bytes(), []byte("\n"))), nil
}

// unescapeLineSeparators undoes encoding/json's escaping of U+2028 and
// U+2029, which serde_json writes raw. Escaped backslashes are skipped.
func unescapeLineSeparators(data []byte) []byte {
	if !bytes.Contains(data, []byte(`\u202`)) {
		return data
	}
	out := make([]byte, 0, len(data))
	for index := 0; index < len(data); index++ {
		if data[index] != '\\' || index+1 >= len(data) {
			out = append(out, data[index])
			continue
		}
		if data[index+1] == 'u' && index+6 <= len(data) {
			switch string(data[index+2 : index+6]) {
			case "2028":
				out = append(out, "\u2028"...)
				index += 5
				continue
			case "2029":
				out = append(out, "\u2029"...)
				index += 5
				continue
			}
		}
		out = append(out, data[index], data[index+1])
		index++
	}
	return out
}
