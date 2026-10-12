package replay

import (
	"encoding/binary"
	"math"
	"sort"
)

// pickleOf encodes test values: nil, bool, int, float64, string, []any, and
// map[string]any (keys sorted), like serde_pickle::to_vec.
func pickleOf(value any) []byte {
	out := []byte{0x80, 3}
	out = appendPickle(out, value)
	return append(out, '.')
}

func appendPickle(out []byte, value any) []byte {
	switch typed := value.(type) {
	case nil:
		return append(out, 'N')
	case bool:
		if typed {
			return append(out, 0x88)
		}
		return append(out, 0x89)
	case int:
		return binary.LittleEndian.AppendUint32(append(out, 'J'), uint32(int32(typed)))
	case float64:
		return binary.BigEndian.AppendUint64(append(out, 'G'), math.Float64bits(typed))
	case string:
		out = binary.LittleEndian.AppendUint32(append(out, 'X'), uint32(len(typed)))
		return append(out, typed...)
	case []byte:
		out = binary.LittleEndian.AppendUint32(append(out, 'B'), uint32(len(typed)))
		return append(out, typed...)
	case []any:
		out = append(out, ']', '(')
		for _, item := range typed {
			out = appendPickle(out, item)
		}
		return append(out, 'e')
	case [2]any:
		out = appendPickle(out, typed[0])
		out = appendPickle(out, typed[1])
		return append(out, 0x86)
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out = append(out, '}', '(')
		for _, key := range keys {
			out = appendPickle(out, key)
			out = appendPickle(out, typed[key])
		}
		return append(out, 'u')
	}
	panic("unsupported test value")
}

func header() []byte {
	out := binary.LittleEndian.AppendUint32(nil, streamMagic)
	out = binary.LittleEndian.AppendUint64(out, 0)
	out = append(out, "\x04hash"...)
	version := "26.8.0_ruby_apple"
	out = append(out, byte(len(version)))
	out = append(out, version...)
	return append(out, 0)
}

func frame(kind uint32, clock float32, payload []byte) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(payload)))
	out = binary.LittleEndian.AppendUint32(out, kind)
	out = binary.LittleEndian.AppendUint32(out, math.Float32bits(clock))
	return append(out, payload...)
}

func sampleReplay() []byte {
	out := header()
	out = append(out, frame(99, 0, []byte{1, 2, 3})...)
	out = append(out, frame(7, 0.2, make([]byte, 12))...)
	return append(out, frame(99, 0.3, nil)...)
}
