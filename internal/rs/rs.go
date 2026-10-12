// Package rs reproduces small pieces of Rust standard-library behavior that
// appear in user-visible output, so messages match the original tool.
package rs

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lines splits text like str::lines: on \n, dropping a trailing \r from each
// line and not yielding an empty final line.
func Lines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for index, line := range lines {
		lines[index] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

// Quote formats a string like Rust's Debug implementation for str.
func Quote(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			builder.WriteString(`\"`)
		case '\\':
			builder.WriteString(`\\`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		case 0:
			builder.WriteString(`\0`)
		default:
			if unicode.IsPrint(r) || r == ' ' {
				builder.WriteRune(r)
			} else {
				fmt.Fprintf(&builder, `\u{%x}`, r)
			}
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

// ParseUint parses an unsigned decimal integer like Rust's str::parse, which
// accepts one leading plus sign.
func ParseUint(value string, bitSize int) (uint64, error) {
	value = strings.TrimPrefix(value, "+")
	if value == "" || value[0] == '+' || value[0] == '-' {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseUint(value, 10, bitSize)
}

// ParseInt parses a signed decimal integer like Rust's str::parse.
func ParseInt(value string, bitSize int) (int64, error) {
	return strconv.ParseInt(value, 10, bitSize)
}

// UTF8Error formats invalid UTF-8 like Rust's Utf8Error Display.
func UTF8Error(data []byte) string {
	valid := 0
	for valid < len(data) {
		r, size := utf8.DecodeRune(data[valid:])
		if r == utf8.RuneError && size <= 1 {
			break
		}
		valid += size
	}
	rest := data[valid:]
	length, complete := invalidSequence(rest)
	if !complete {
		return fmt.Sprintf("incomplete utf-8 byte sequence from index %d", valid)
	}
	return fmt.Sprintf("invalid utf-8 sequence of %d bytes from index %d", length, valid)
}

// invalidSequence returns Rust's error_len for an invalid prefix, and false
// when the input ends inside an otherwise valid sequence.
func invalidSequence(data []byte) (int, bool) {
	first := data[0]
	need, low, high := 0, byte(0x80), byte(0xbf)
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
		return 1, true
	}
	for index := 1; index <= need; index++ {
		if index >= len(data) {
			return 0, false
		}
		if data[index] < low || data[index] > high {
			return index, true
		}
		low, high = 0x80, 0xbf
	}
	return need + 1, true
}
