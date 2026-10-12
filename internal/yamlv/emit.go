package yamlv

import (
	"bytes"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Marshal serializes a value like serde_yaml_ng::to_string: block style, two
// space indentation, unindented sequences, no line folding, and serde's
// scalar style choices.
func Marshal(value *Value) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	encoder.CompactSeqIndent()
	if err := encoder.Encode(toNode(value)); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func toNode(value *Value) *yaml.Node {
	switch value.Kind {
	case Null:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: "null"}
	case Bool:
		text := "false"
		if value.Bool {
			text = "true"
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Value: text}
	case Int, Float:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: FormatNumber(value)}
	case String:
		node := &yaml.Node{Kind: yaml.ScalarNode, Value: value.Str}
		switch {
		case strings.Contains(value.Str, "\n"):
			node.Style = yaml.LiteralStyle
		case IsNonString(value.Str):
			node.Style = yaml.SingleQuotedStyle
		}
		return node
	case Seq:
		node := &yaml.Node{Kind: yaml.SequenceNode}
		for _, item := range value.Items {
			node.Content = append(node.Content, toNode(item))
		}
		return node
	default:
		node := &yaml.Node{Kind: yaml.MappingNode}
		for index := range value.Keys {
			node.Content = append(node.Content, toNode(value.Keys[index]), toNode(value.Vals[index]))
		}
		return node
	}
}
