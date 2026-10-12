// Package pickle decodes Python pickles into builtin values, following
// serde-pickle: strings from protocol 2 stay bytes, and globals other than a
// few builtin containers are never imported or executed.
package pickle

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"unicode/utf8"
)

type (
	// Bytes is a Python bytes or py2 str value.
	Bytes []byte
	// List is a mutable Python list.
	List struct{ Items []Value }
	// Tuple is a Python tuple.
	Tuple []Value
	// Dict is a Python dict in insertion order.
	Dict struct{ Entries []Entry }
	// Set is a Python set or frozenset.
	Set struct {
		Items  []Value
		Frozen bool
	}
	// Entry is one dict item.
	Entry struct{ Key, Value Value }
	// Global is an unresolved module global.
	Global struct{ kind string }
)

// Value is nil (None), bool, *big.Int, int64, float64, string, Bytes, *List,
// Tuple, *Dict, *Set, or Global.
type Value any

// Error is a decoding error with its stream offset.
type Error struct {
	Offset  int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("eval error at offset %d: %s", e.Offset, e.Message)
}

const (
	opMark           = '('
	opStop           = '.'
	opPop            = '0'
	opPopMark        = '1'
	opDup            = '2'
	opFloat          = 'F'
	opInt            = 'I'
	opBinInt         = 'J'
	opBinInt1        = 'K'
	opLong           = 'L'
	opBinInt2        = 'M'
	opNone           = 'N'
	opPersID         = 'P'
	opBinPersID      = 'Q'
	opReduce         = 'R'
	opString         = 'S'
	opBinString      = 'T'
	opShortBinString = 'U'
	opUnicode        = 'V'
	opBinUnicode     = 'X'
	opAppend         = 'a'
	opBuild          = 'b'
	opGlobal         = 'c'
	opDict           = 'd'
	opEmptyDict      = '}'
	opAppends        = 'e'
	opGet            = 'g'
	opBinGet         = 'h'
	opInst           = 'i'
	opLongBinGet     = 'j'
	opList           = 'l'
	opEmptyList      = ']'
	opObj            = 'o'
	opPut            = 'p'
	opBinPut         = 'q'
	opLongBinPut     = 'r'
	opSetItem        = 's'
	opTuple          = 't'
	opEmptyTuple     = ')'
	opSetItems       = 'u'
	opBinFloat       = 'G'
	opProto          = 0x80
	opNewObj         = 0x81
	opTuple1         = 0x85
	opTuple2         = 0x86
	opTuple3         = 0x87
	opNewTrue        = 0x88
	opNewFalse       = 0x89
	opLong1          = 0x8a
	opLong4          = 0x8b
	opBinBytes       = 'B'
	opShortBinBytes  = 'C'
	opShortBinUni    = 0x8c
	opBinUnicode8    = 0x8d
	opBinBytes8      = 0x8e
	opEmptySet       = 0x8f
	opAddItems       = 0x90
	opFrozenSet      = 0x91
	opNewObjEx       = 0x92
	opStackGlobal    = 0x93
	opMemoize        = 0x94
	opFrame          = 0x95
	opByteArray8     = 0x96
)

type decoder struct {
	data   []byte
	pos    int
	stack  []Value
	stacks [][]Value
	memo   map[uint32]Value
}

// Decode parses a complete pickle, rejecting trailing bytes.
func Decode(data []byte) (Value, error) {
	d := &decoder{data: data, memo: map[uint32]Value{}}
	value, err := d.parse()
	if err != nil {
		return nil, err
	}
	if d.pos < len(d.data) {
		return nil, d.fail("Trailing bytes found")
	}
	return value, nil
}

func (d *decoder) fail(message string) error { return &Error{Offset: d.pos, Message: message} }

func (d *decoder) read(n int) ([]byte, error) {
	if n < 0 || d.pos+n > len(d.data) || d.pos+n < d.pos {
		d.pos = len(d.data)
		return nil, d.fail("EOF while parsing")
	}
	out := d.data[d.pos : d.pos+n]
	d.pos += n
	return out, nil
}

func (d *decoder) byte() (byte, error) {
	b, err := d.read(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (d *decoder) line() ([]byte, error) {
	index := bytes.IndexByte(d.data[d.pos:], '\n')
	var line []byte
	if index < 0 {
		line = d.data[d.pos:]
		d.pos = len(d.data)
	} else {
		line = d.data[d.pos : d.pos+index]
		d.pos += index + 1
	}
	return bytes.TrimSuffix(line, []byte("\r")), nil
}

func (d *decoder) prefixed(size int) ([]byte, error) {
	header, err := d.read(size)
	if err != nil {
		return nil, err
	}
	var length uint64
	switch size {
	case 1:
		length = uint64(header[0])
	case 4:
		length = uint64(binary.LittleEndian.Uint32(header))
	default:
		length = binary.LittleEndian.Uint64(header)
	}
	if length > uint64(len(d.data)) {
		d.pos = len(d.data)
		return nil, d.fail("EOF while parsing")
	}
	data, err := d.read(int(length))
	return bytes.Clone(data), err
}

func (d *decoder) pop() (Value, error) {
	if len(d.stack) == 0 {
		return nil, d.fail("pickle stack underflow")
	}
	value := d.stack[len(d.stack)-1]
	d.stack = d.stack[:len(d.stack)-1]
	return value, nil
}

func (d *decoder) popMark() ([]Value, error) {
	if len(d.stacks) == 0 {
		return nil, d.fail("pickle stack underflow")
	}
	items := d.stack
	d.stack = d.stacks[len(d.stacks)-1]
	d.stacks = d.stacks[:len(d.stacks)-1]
	return items, nil
}

func (d *decoder) top() (Value, error) {
	if len(d.stack) == 0 {
		return nil, d.fail("pickle stack underflow")
	}
	return d.stack[len(d.stack)-1], nil
}

func (d *decoder) memoize(id uint32) error {
	value, err := d.top()
	if err != nil {
		return err
	}
	d.memo[id] = value
	return nil
}

func (d *decoder) get(id uint32) error {
	value, ok := d.memo[id]
	if !ok {
		return d.fail(fmt.Sprintf("missing memo with id %d", id))
	}
	d.stack = append(d.stack, value)
	return nil
}

func (d *decoder) ascii(line []byte) (uint32, error) {
	value, err := strconv.ParseUint(string(line), 10, 32)
	if err != nil {
		return 0, d.fail(fmt.Sprintf("invalid literal: %q", line))
	}
	return uint32(value), nil
}

func (d *decoder) parse() (Value, error) {
	for {
		op, err := d.byte()
		if err != nil {
			return nil, err
		}
		switch op {
		case opProto:
			if _, err := d.byte(); err != nil {
				return nil, err
			}
		case opFrame:
			if _, err := d.read(8); err != nil {
				return nil, err
			}
		case opStop:
			return d.pop()
		case opMark:
			d.stacks = append(d.stacks, d.stack)
			d.stack = nil
		case opPop:
			if len(d.stack) == 0 {
				if _, err := d.popMark(); err != nil {
					return nil, err
				}
			} else {
				d.pop()
			}
		case opPopMark:
			if _, err := d.popMark(); err != nil {
				return nil, err
			}
		case opDup:
			value, err := d.top()
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, value)
		case opPut:
			line, _ := d.line()
			id, err := d.ascii(line)
			if err != nil {
				return nil, err
			}
			if err := d.memoize(id); err != nil {
				return nil, err
			}
		case opBinPut:
			id, err := d.byte()
			if err != nil {
				return nil, err
			}
			if err := d.memoize(uint32(id)); err != nil {
				return nil, err
			}
		case opLongBinPut:
			b, err := d.read(4)
			if err != nil {
				return nil, err
			}
			if err := d.memoize(binary.LittleEndian.Uint32(b)); err != nil {
				return nil, err
			}
		case opMemoize:
			if err := d.memoize(uint32(len(d.memo))); err != nil {
				return nil, err
			}
		case opGet:
			line, _ := d.line()
			id, err := d.ascii(line)
			if err != nil {
				return nil, err
			}
			if err := d.get(id); err != nil {
				return nil, err
			}
		case opBinGet:
			id, err := d.byte()
			if err != nil {
				return nil, err
			}
			if err := d.get(uint32(id)); err != nil {
				return nil, err
			}
		case opLongBinGet:
			b, err := d.read(4)
			if err != nil {
				return nil, err
			}
			if err := d.get(binary.LittleEndian.Uint32(b)); err != nil {
				return nil, err
			}
		case opNone:
			d.stack = append(d.stack, nil)
		case opNewFalse:
			d.stack = append(d.stack, false)
		case opNewTrue:
			d.stack = append(d.stack, true)
		case opInt:
			line, _ := d.line()
			switch string(line) {
			case "00":
				d.stack = append(d.stack, false)
			case "01":
				d.stack = append(d.stack, true)
			default:
				value, err := strconv.ParseInt(string(line), 10, 64)
				if err != nil {
					return nil, d.fail(fmt.Sprintf("invalid literal: %q", line))
				}
				d.stack = append(d.stack, value)
			}
		case opLong:
			line, _ := d.line()
			line = bytes.TrimSuffix(line, []byte("L"))
			value, ok := new(big.Int).SetString(string(line), 10)
			if !ok {
				return nil, d.fail(fmt.Sprintf("invalid literal: %q", line))
			}
			d.stack = append(d.stack, value)
		case opFloat:
			line, _ := d.line()
			value, err := strconv.ParseFloat(string(line), 64)
			if err != nil {
				return nil, d.fail(fmt.Sprintf("invalid literal: %q", line))
			}
			d.stack = append(d.stack, value)
		case opString:
			line, _ := d.line()
			value, err := d.escapedString(line)
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, value)
		case opUnicode:
			line, _ := d.line()
			value, err := d.escapedUnicode(line)
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, value)
		case opBinFloat:
			b, err := d.read(8)
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, math.Float64frombits(binary.BigEndian.Uint64(b)))
		case opBinInt:
			b, err := d.read(4)
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, int64(int32(binary.LittleEndian.Uint32(b))))
		case opBinInt1:
			b, err := d.byte()
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, int64(b))
		case opBinInt2:
			b, err := d.read(2)
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, int64(binary.LittleEndian.Uint16(b)))
		case opLong1, opLong4:
			size := 1
			if op == opLong4 {
				size = 4
			}
			if size == 4 {
				header, err := d.read(4)
				if err != nil {
					return nil, err
				}
				if int32(binary.LittleEndian.Uint32(header)) < 0 {
					return nil, d.fail("negative length prefix")
				}
				d.pos -= 4
			}
			data, err := d.prefixed(size)
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, binaryLong(data))
		case opShortBinBytes, opBinBytes, opBinBytes8, opByteArray8:
			data, err := d.prefixed(map[byte]int{opShortBinBytes: 1, opBinBytes: 4, opBinBytes8: 8, opByteArray8: 8}[op])
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, Bytes(data))
		case opShortBinString:
			data, err := d.prefixed(1)
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, Bytes(data))
		case opBinString:
			header, err := d.read(4)
			if err != nil {
				return nil, err
			}
			if int32(binary.LittleEndian.Uint32(header)) < 0 {
				return nil, d.fail("negative length prefix")
			}
			d.pos -= 4
			data, err := d.prefixed(4)
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, Bytes(data))
		case opShortBinUni, opBinUnicode, opBinUnicode8:
			data, err := d.prefixed(map[byte]int{opShortBinUni: 1, opBinUnicode: 4, opBinUnicode8: 8}[op])
			if err != nil {
				return nil, err
			}
			if !utf8.Valid(data) {
				return nil, d.fail("string is not UTF-8 encoded")
			}
			d.stack = append(d.stack, string(data))
		case opEmptyTuple:
			d.stack = append(d.stack, Tuple{})
		case opTuple1, opTuple2, opTuple3:
			count := int(op-opTuple1) + 1
			if len(d.stack) < count {
				return nil, d.fail("pickle stack underflow")
			}
			items := append(Tuple(nil), d.stack[len(d.stack)-count:]...)
			d.stack = d.stack[:len(d.stack)-count]
			d.stack = append(d.stack, items)
		case opTuple:
			items, err := d.popMark()
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, Tuple(append([]Value{}, items...)))
		case opEmptyList:
			d.stack = append(d.stack, &List{})
		case opList:
			items, err := d.popMark()
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, &List{Items: items})
		case opAppend, opAppends:
			var items []Value
			if op == opAppend {
				value, err := d.pop()
				if err != nil {
					return nil, err
				}
				items = []Value{value}
			} else if items, err = d.popMark(); err != nil {
				return nil, err
			}
			top, err := d.top()
			if err != nil {
				return nil, err
			}
			list, ok := top.(*List)
			if !ok {
				return nil, d.fail("invalid stack top, expected list")
			}
			list.Items = append(list.Items, items...)
		case opEmptyDict:
			d.stack = append(d.stack, &Dict{})
		case opDict:
			items, err := d.popMark()
			if err != nil {
				return nil, err
			}
			dict := &Dict{}
			extendDict(dict, items)
			d.stack = append(d.stack, dict)
		case opSetItem, opSetItems:
			var items []Value
			if op == opSetItem {
				value, err := d.pop()
				if err != nil {
					return nil, err
				}
				key, err := d.pop()
				if err != nil {
					return nil, err
				}
				items = []Value{key, value}
			} else if items, err = d.popMark(); err != nil {
				return nil, err
			}
			top, err := d.top()
			if err != nil {
				return nil, err
			}
			dict, ok := top.(*Dict)
			if !ok {
				return nil, d.fail("invalid stack top, expected dict")
			}
			extendDict(dict, items)
		case opEmptySet:
			d.stack = append(d.stack, &Set{})
		case opFrozenSet:
			items, err := d.popMark()
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, &Set{Items: items, Frozen: true})
		case opAddItems:
			items, err := d.popMark()
			if err != nil {
				return nil, err
			}
			top, err := d.top()
			if err != nil {
				return nil, err
			}
			set, ok := top.(*Set)
			if !ok {
				return nil, d.fail("invalid stack top, expected set")
			}
			set.Items = append(set.Items, items...)
		case opGlobal:
			module, _ := d.line()
			name, _ := d.line()
			d.stack = append(d.stack, global(string(module), string(name)))
		case opStackGlobal:
			name, err := d.pop()
			if err != nil {
				return nil, err
			}
			module, err := d.pop()
			if err != nil {
				return nil, err
			}
			nameText, ok1 := name.(string)
			moduleText, ok2 := module.(string)
			if !ok1 || !ok2 {
				return nil, d.fail("invalid stack top, expected string")
			}
			d.stack = append(d.stack, global(moduleText, nameText))
		case opReduce:
			args, err := d.pop()
			if err != nil {
				return nil, err
			}
			tuple, ok := args.(Tuple)
			if !ok {
				return nil, d.fail("invalid stack top, expected tuple")
			}
			callable, err := d.pop()
			if err != nil {
				return nil, err
			}
			result, err := d.reduce(callable, tuple)
			if err != nil {
				return nil, err
			}
			d.stack = append(d.stack, result)
		case opInst:
			d.line()
			d.line()
			if _, err := d.popMark(); err != nil {
				return nil, err
			}
			d.stack = append(d.stack, &Dict{})
		case opObj:
			if _, err := d.popMark(); err != nil {
				return nil, err
			}
			if _, err := d.pop(); err != nil {
				return nil, err
			}
			d.stack = append(d.stack, &Dict{})
		case opNewObj:
			for range 2 {
				if _, err := d.pop(); err != nil {
					return nil, err
				}
			}
			d.stack = append(d.stack, &Dict{})
		case opNewObjEx:
			for range 3 {
				if _, err := d.pop(); err != nil {
					return nil, err
				}
			}
			d.stack = append(d.stack, &Dict{})
		case opBuild:
			state, err := d.pop()
			if err != nil {
				return nil, err
			}
			if _, err := d.pop(); err != nil {
				return nil, err
			}
			d.stack = append(d.stack, state)
		default:
			return nil, d.fail(fmt.Sprintf("unsupported opcode '%c'", rune(op)))
		}
	}
}

func extendDict(dict *Dict, items []Value) {
	for index := 0; index+1 < len(items); index += 2 {
		dict.Entries = append(dict.Entries, Entry{items[index], items[index+1]})
	}
}

func global(module, name string) Value {
	switch {
	case module == "_codecs" && name == "encode":
		return Global{"encode"}
	case (module == "copy_reg" || module == "copyreg") && name == "_reconstructor":
		return Global{"reconstructor"}
	case module == "__builtin__" || module == "builtins":
		switch name {
		case "set", "frozenset", "list", "bytearray", "int":
			return Global{name}
		}
	}
	return Global{"other"}
}

func (d *decoder) reduce(callable Value, args Tuple) (Value, error) {
	g, ok := callable.(Global)
	if !ok {
		return nil, d.fail("invalid stack top, expected global reference")
	}
	last := func() Value {
		if len(args) == 0 {
			return nil
		}
		value := args[len(args)-1]
		args = args[:len(args)-1]
		return value
	}
	invalid := func(what string) error { return d.fail("invalid value: " + what) }
	switch g.kind {
	case "set", "frozenset":
		list, ok := last().(*List)
		if !ok {
			return nil, invalid(g.kind + "() arg")
		}
		return &Set{Items: list.Items, Frozen: g.kind == "frozenset"}, nil
	case "bytearray":
		if len(args) > 1 {
			args = args[:1]
		}
		switch value := last().(type) {
		case Bytes:
			return value, nil
		case string:
			out := make(Bytes, 0, len(value))
			for _, r := range value {
				out = append(out, byte(r))
			}
			return out, nil
		}
		return nil, invalid("bytearray() arg")
	case "list":
		list, ok := last().(*List)
		if !ok {
			return nil, invalid("list() arg")
		}
		return &List{Items: list.Items}, nil
	case "int":
		value, ok := last().(*big.Int)
		if !ok {
			return nil, invalid("int() arg")
		}
		return value, nil
	case "encode":
		if _, ok := last().(string); !ok {
			return nil, invalid("encode() arg")
		}
		text, ok := last().(string)
		if !ok {
			return nil, invalid("encode() arg")
		}
		out := make(Bytes, 0, len(text))
		for _, r := range text {
			out = append(out, byte(r))
		}
		return out, nil
	case "reconstructor":
		return &Dict{}, nil
	}
	return Global{"other"}, nil
}

func binaryLong(data []byte) *big.Int {
	reversed := make([]byte, len(data))
	for index, b := range data {
		reversed[len(data)-1-index] = b
	}
	value := new(big.Int).SetBytes(reversed)
	if len(data) > 0 && data[len(data)-1]&0x80 != 0 {
		value.Sub(value, new(big.Int).Lsh(big.NewInt(1), uint(len(data)*8)))
	}
	return value
}

func (d *decoder) escapedString(line []byte) (Value, error) {
	if len(line) >= 2 && line[0] == line[len(line)-1] && (line[0] == '"' || line[0] == '\'') {
		line = line[1 : len(line)-1]
	}
	var out Bytes
	for index := 0; index < len(line); index++ {
		b := line[index]
		if b != '\\' {
			out = append(out, b)
			continue
		}
		index++
		if index >= len(line) {
			return nil, d.fail(fmt.Sprintf("invalid literal: %q", line))
		}
		switch line[index] {
		case '\\':
			out = append(out, '\\')
		case 'a':
			out = append(out, 7)
		case 'b':
			out = append(out, 8)
		case 't':
			out = append(out, 9)
		case 'n':
			out = append(out, 10)
		case 'v':
			out = append(out, 11)
		case 'f':
			out = append(out, 12)
		case 'r':
			out = append(out, 13)
		case 'x':
			if index+3 > len(line) {
				return nil, d.fail(fmt.Sprintf("invalid literal: %q", line))
			}
			value, err := strconv.ParseUint(string(line[index+1:index+3]), 16, 8)
			if err != nil {
				return nil, d.fail(fmt.Sprintf("invalid literal: %q", line))
			}
			out = append(out, byte(value))
			index += 2
		default:
			return nil, d.fail(fmt.Sprintf("invalid literal: %q", line))
		}
	}
	return out, nil
}

func (d *decoder) escapedUnicode(line []byte) (Value, error) {
	var out []rune
	for index := 0; index < len(line); index++ {
		b := line[index]
		if b != '\\' {
			out = append(out, rune(b))
			continue
		}
		index++
		size := 0
		if index < len(line) && line[index] == 'u' {
			size = 4
		} else if index < len(line) && line[index] == 'U' {
			size = 8
		}
		if size == 0 || index+1+size > len(line) {
			return nil, d.fail(fmt.Sprintf("invalid literal: %q", line))
		}
		value, err := strconv.ParseUint(string(line[index+1:index+1+size]), 16, 32)
		if err != nil || value > 0x10ffff || (value >= 0xd800 && value <= 0xdfff) {
			return nil, d.fail(fmt.Sprintf("invalid literal: %q", line))
		}
		out = append(out, rune(value))
		index += size
	}
	return string(out), nil
}
