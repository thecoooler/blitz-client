// Package lossy converts bytes to strings the way Rust's from_utf8_lossy does.
package lossy

import (
	"strings"
	"unicode/utf8"
)

// String replaces invalid UTF-8 sequences with U+FFFD.
func String(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	var builder strings.Builder
	builder.Grow(len(data) + 8)
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size <= 1 {
			builder.WriteRune(utf8.RuneError)
			data = data[invalidLength(data):]
			continue
		}
		builder.WriteRune(r)
		data = data[size:]
	}
	return builder.String()
}

// invalidLength returns the length of the maximal prefix of an ill-formed
// sequence, which Rust replaces with a single U+FFFD.
func invalidLength(data []byte) int {
	first := data[0]
	var need int
	low, high := byte(0x80), byte(0xbf)
	switch {
	case first >= 0xc2 && first <= 0xdf:
		need = 1
	case first == 0xe0:
		need, low = 2, 0xa0
	case first >= 0xe1 && first <= 0xec, first == 0xee, first == 0xef:
		need = 2
	case first == 0xed:
		need, high = 2, 0x9f
	case first == 0xf0:
		need, low = 3, 0x90
	case first >= 0xf1 && first <= 0xf3:
		need = 3
	case first == 0xf4:
		need, high = 3, 0x8f
	default:
		return 1
	}
	length := 1
	for index := 1; index <= need && index < len(data); index++ {
		value := data[index]
		if value < low || value > high {
			break
		}
		low, high = 0x80, 0xbf
		length++
	}
	return length
}
