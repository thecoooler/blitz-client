// Package glob ports the pattern syntax and matching rules of the Rust glob
// crate (0.3), so --exclude accepts and rejects the same patterns.
package glob

import (
	"fmt"
	"os"
)

type tokenKind int

const (
	char tokenKind = iota
	anyChar
	anySequence
	anyRecursiveSequence
	anyWithin
	anyExcept
)

type specifier struct {
	low, high rune
}

type token struct {
	kind       tokenKind
	char       rune
	specifiers []specifier
}

// Pattern is a compiled glob pattern.
type Pattern struct {
	tokens   []token
	original string
}

// Error is a pattern syntax error.
type Error struct {
	Position int
	Message  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("Pattern syntax error near position %d: %s", e.Position, e.Message)
}

const (
	errWildcards          = "wildcards are either regular `*` or recursive `**`"
	errRecursiveWildcards = "recursive wildcards must form a single path component"
	errInvalidRange       = "invalid range pattern"
)

func isSeparator(c rune) bool {
	return c == '/' || (os.PathSeparator == '\\' && c == '\\')
}

// New compiles a pattern.
func New(pattern string) (*Pattern, error) {
	chars := []rune(pattern)
	var tokens []token
	for i := 0; i < len(chars); {
		switch chars[i] {
		case '?':
			tokens = append(tokens, token{kind: anyChar})
			i++
		case '*':
			old := i
			for i < len(chars) && chars[i] == '*' {
				i++
			}
			count := i - old
			switch {
			case count > 2:
				return nil, &Error{old + 2, errWildcards}
			case count == 2:
				if i == 2 || isSeparator(chars[i-count-1]) {
					if i < len(chars) && isSeparator(chars[i]) {
						i++
					} else if i != len(chars) {
						return nil, &Error{i, errRecursiveWildcards}
					}
				} else {
					return nil, &Error{old - 1, errRecursiveWildcards}
				}
				if n := len(tokens); !(n > 1 && tokens[n-1].kind == anyRecursiveSequence) {
					tokens = append(tokens, token{kind: anyRecursiveSequence})
				}
			default:
				tokens = append(tokens, token{kind: anySequence})
			}
		case '[':
			if i+4 <= len(chars) && chars[i+1] == '!' {
				if j := index(chars[i+3:], ']'); j >= 0 {
					tokens = append(tokens, token{kind: anyExcept, specifiers: parseSpecifiers(chars[i+2 : i+3+j])})
					i += j + 4
					continue
				}
			} else if i+3 <= len(chars) && chars[i+1] != '!' {
				if j := index(chars[i+2:], ']'); j >= 0 {
					tokens = append(tokens, token{kind: anyWithin, specifiers: parseSpecifiers(chars[i+1 : i+2+j])})
					i += j + 3
					continue
				}
			}
			return nil, &Error{i, errInvalidRange}
		default:
			tokens = append(tokens, token{kind: char, char: chars[i]})
			i++
		}
	}
	return &Pattern{tokens: tokens, original: pattern}, nil
}

func index(chars []rune, target rune) int {
	for position, c := range chars {
		if c == target {
			return position
		}
	}
	return -1
}

func parseSpecifiers(chars []rune) []specifier {
	var out []specifier
	for i := 0; i < len(chars); {
		if i+3 <= len(chars) && chars[i+1] == '-' {
			out = append(out, specifier{chars[i], chars[i+2]})
			i += 3
		} else {
			out = append(out, specifier{chars[i], chars[i]})
			i++
		}
	}
	return out
}

func (p *Pattern) String() string { return p.original }

type matchResult int

const (
	match matchResult = iota
	subPatternDoesntMatch
	entirePatternDoesntMatch
)

// Match reports whether the path matches with case-sensitive matching,
// literal separators, and no special treatment of leading dots.
func (p *Pattern) Match(path string) bool {
	return p.matchesFrom(true, []rune(path), 0) == match
}

func (p *Pattern) matchesFrom(followsSeparator bool, file []rune, start int) matchResult {
	for ti := start; ti < len(p.tokens); ti++ {
		tok := p.tokens[ti]
		switch tok.kind {
		case anySequence, anyRecursiveSequence:
			if m := p.matchesFrom(followsSeparator, file, ti+1); m != subPatternDoesntMatch {
				return m
			}
			for len(file) > 0 {
				c := file[0]
				file = file[1:]
				followsSeparator = isSeparator(c)
				if tok.kind == anyRecursiveSequence && !followsSeparator {
					continue
				}
				if tok.kind == anySequence && followsSeparator {
					return subPatternDoesntMatch
				}
				if m := p.matchesFrom(followsSeparator, file, ti+1); m != subPatternDoesntMatch {
					return m
				}
			}
		default:
			if len(file) == 0 {
				return entirePatternDoesntMatch
			}
			c := file[0]
			file = file[1:]
			separator := isSeparator(c)
			var ok bool
			switch tok.kind {
			case anyChar:
				ok = !separator
			case anyWithin:
				ok = !separator && inSpecifiers(tok.specifiers, c)
			case anyExcept:
				ok = !separator && !inSpecifiers(tok.specifiers, c)
			case char:
				ok = c == tok.char
			}
			if !ok {
				return subPatternDoesntMatch
			}
			followsSeparator = separator
		}
	}
	if len(file) == 0 {
		return match
	}
	return subPatternDoesntMatch
}

func inSpecifiers(specifiers []specifier, c rune) bool {
	for _, s := range specifiers {
		if s.low <= c && c <= s.high {
			return true
		}
	}
	return false
}
