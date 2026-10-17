// Package xml is a small, permissive parser for the client's XML resources. It
// keeps element names, scalar text, attributes, and ordered children.
package xml

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode"

	"github.com/thecoooler/blitz-client/internal/lossy"
)

// Node is one XML element.
type Node struct {
	Name       string
	Text       string
	Children   []*Node
	Attributes map[string]string
}

// Child returns the first child with the given name.
func (n *Node) Child(name string) *Node {
	for _, child := range n.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

// ChildText returns the trimmed text of the first child with the given name.
// The second result is false when the child is missing or its text is empty.
func ChildText(node *Node, name string) (string, bool) {
	child := node.Child(name)
	if child == nil {
		return "", false
	}
	text := strings.TrimSpace(child.Text)
	return text, text != ""
}

// Parse parses a document with exactly one root element.
func Parse(data []byte) (*Node, error) {
	source := strings.TrimLeft(lossy.String(data), "\xef\xbb\xbf")
	var roots []*Node
	var stack []*Node
	cursor := 0

	for cursor < len(source) {
		relative := strings.IndexByte(source[cursor:], '<')
		if relative < 0 {
			if err := appendText(stack, source[cursor:]); err != nil {
				return nil, err
			}
			break
		}
		open := cursor + relative
		if open > cursor {
			if err := appendText(stack, source[cursor:open]); err != nil {
				return nil, err
			}
		}
		rest := source[open:]
		if strings.HasPrefix(rest, "<!--") {
			end := strings.Index(source[open+4:], "-->")
			if end < 0 {
				return nil, fmt.Errorf("unterminated XML comment")
			}
			cursor = open + 4 + end + 3
			continue
		}
		if strings.HasPrefix(rest, "<![CDATA[") {
			end := strings.Index(source[open+9:], "]]>")
			if end < 0 {
				return nil, fmt.Errorf("unterminated XML CDATA section")
			}
			start := open + 9
			if len(stack) > 0 {
				stack[len(stack)-1].Text += source[start : start+end]
			}
			cursor = start + end + 3
			continue
		}
		if strings.HasPrefix(rest, "<?") || strings.HasPrefix(rest, "<!") {
			end, err := findTagEnd(source, open)
			if err != nil {
				return nil, err
			}
			cursor = end + 1
			continue
		}

		end, err := findTagEnd(source, open)
		if err != nil {
			return nil, err
		}
		body := strings.TrimSpace(source[open+1 : end])
		if closeName, ok := strings.CutPrefix(body, "/"); ok {
			closeName = strings.TrimSpace(closeName)
			if len(stack) == 0 {
				return nil, fmt.Errorf("unexpected closing XML tag </%s>", closeName)
			}
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if node.Name != closeName {
				return nil, fmt.Errorf("mismatched XML closing tag </%s> for <%s>", closeName, node.Name)
			}
			roots = appendNode(roots, stack, node)
		} else {
			selfClosing := strings.HasSuffix(body, "/")
			if selfClosing {
				body = strings.TrimRightFunc(body[:len(body)-1], unicode.IsSpace)
			}
			nameEnd := strings.IndexFunc(body, unicode.IsSpace)
			if nameEnd < 0 {
				nameEnd = len(body)
			}
			if nameEnd == 0 {
				return nil, fmt.Errorf("empty XML tag name")
			}
			node := &Node{Name: body[:nameEnd]}
			if attributes := strings.TrimSpace(body[nameEnd:]); attributes != "" {
				node.Attributes = parseAttributes(attributes)
			}
			if selfClosing {
				roots = appendNode(roots, stack, node)
			} else {
				stack = append(stack, node)
			}
		}
		cursor = end + 1
	}

	if len(stack) > 0 {
		return nil, fmt.Errorf("unclosed XML tag <%s>", stack[len(stack)-1].Name)
	}
	if len(roots) != 1 {
		return nil, fmt.Errorf("expected one XML root element, found %d", len(roots))
	}
	return roots[0], nil
}

func appendNode(roots []*Node, stack []*Node, node *Node) []*Node {
	if len(stack) > 0 {
		parent := stack[len(stack)-1]
		parent.Children = append(parent.Children, node)
		return roots
	}
	return append(roots, node)
}

func appendText(stack []*Node, text string) error {
	if len(stack) == 0 {
		if strings.TrimSpace(text) == "" {
			return nil
		}
		return fmt.Errorf("text outside the XML root element")
	}
	decoded, err := decodeEntities(text)
	if err != nil {
		return err
	}
	stack[len(stack)-1].Text += decoded
	return nil
}

func findTagEnd(source string, open int) (int, error) {
	var quote byte
	for index := open + 1; index < len(source); index++ {
		ch := source[index]
		switch {
		case quote != 0 && ch == quote:
			quote = 0
		case quote == 0 && (ch == '\'' || ch == '"'):
			quote = ch
		case quote == 0 && ch == '>':
			return index, nil
		}
	}
	return 0, fmt.Errorf("unterminated XML tag")
}

func decodeEntities(text string) (string, error) {
	if !strings.Contains(text, "&") {
		return text, nil
	}
	var result strings.Builder
	result.Grow(len(text))
	rest := text
	for {
		start := strings.IndexByte(rest, '&')
		if start < 0 {
			break
		}
		result.WriteString(rest[:start])
		relativeEnd := strings.IndexByte(rest[start:], ';')
		if relativeEnd < 0 {
			return "", fmt.Errorf("unterminated XML entity")
		}
		end := start + relativeEnd
		entity := rest[start+1 : end]
		switch {
		case entity == "amp":
			result.WriteByte('&')
		case entity == "lt":
			result.WriteByte('<')
		case entity == "gt":
			result.WriteByte('>')
		case entity == "quot":
			result.WriteByte('"')
		case entity == "apos":
			result.WriteByte('\'')
		case strings.HasPrefix(entity, "#x") || strings.HasPrefix(entity, "#X"):
			r, err := scalar(entity, entity[2:], 16)
			if err != nil {
				return "", err
			}
			result.WriteRune(r)
		case strings.HasPrefix(entity, "#"):
			r, err := scalar(entity, entity[1:], 10)
			if err != nil {
				return "", err
			}
			result.WriteRune(r)
		default:
			return "", fmt.Errorf("unsupported XML entity &%s;", entity)
		}
		rest = rest[end+1:]
	}
	result.WriteString(rest)
	return result.String(), nil
}

func scalar(entity, digits string, base int) (rune, error) {
	digits = strings.TrimPrefix(digits, "+")
	if digits == "" || digits[0] == '+' || digits[0] == '-' {
		return 0, fmt.Errorf("invalid XML entity &%s;", entity)
	}
	value, err := strconv.ParseUint(digits, base, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid XML entity &%s;", entity)
	}
	if value > unicode.MaxRune || (value >= 0xd800 && value <= 0xdfff) {
		return 0, fmt.Errorf("invalid XML scalar in &%s;", entity)
	}
	return rune(value), nil
}

// parseAttributes is permissive like the rest of the parser: unquoted values,
// HTML entities and stray ampersands are accepted. Malformed fragments are
// skipped without losing the attributes around them; the first duplicate wins.
func parseAttributes(attributes string) map[string]string {
	result := map[string]string{}
	isSpace := func(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }
	i := 0
	for i < len(attributes) {
		for i < len(attributes) && isSpace(attributes[i]) {
			i++
		}
		start := i
		for i < len(attributes) && !isSpace(attributes[i]) && attributes[i] != '=' {
			i++
		}
		name := attributes[start:i]
		for i < len(attributes) && isSpace(attributes[i]) {
			i++
		}
		value := ""
		if i < len(attributes) && attributes[i] == '=' {
			i++
			for i < len(attributes) && isSpace(attributes[i]) {
				i++
			}
			if i < len(attributes) && (attributes[i] == '"' || attributes[i] == '\'') {
				quote := attributes[i]
				i++
				end := strings.IndexByte(attributes[i:], quote)
				if end < 0 {
					end = len(attributes) - i
				}
				value = attributes[i : i+end]
				i = min(i+end+1, len(attributes))
			} else {
				start := i
				for i < len(attributes) && !isSpace(attributes[i]) {
					i++
				}
				value = attributes[start:i]
			}
		}
		if name == "" || strings.ContainsAny(name, "\"'<>") {
			continue
		}
		if _, exists := result[name]; !exists {
			result[name] = html.UnescapeString(value)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
