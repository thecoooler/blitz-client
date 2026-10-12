package pickle

import (
	"math/big"
	"reflect"
	"testing"
)

func TestDecodesProtocolTwoContainersAndMemo(t *testing.T) {
	// A MARK-delimited tuple with a memoized list referenced twice.
	data := []byte("\x80\x02(K{U\x02abq\x00}q\x01U\x01k]q\x02(K\x01N\x88G@\x04\x00\x00\x00\x00\x00\x00esX\x02\x00\x00\x00\xc3\xa9h\x02\x8a\x01\xfftq\x03.")
	value, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	tuple, ok := value.(Tuple)
	if !ok || len(tuple) != 6 || tuple[3] != "é" || tuple[5].(*big.Int).Int64() != -1 {
		t.Fatalf("%#v", value)
	}
	if tuple[0] != int64(123) || !reflect.DeepEqual(tuple[1], Bytes("ab")) {
		t.Fatalf("%#v", tuple)
	}
	dict := tuple[2].(*Dict)
	list := dict.Entries[0].Value.(*List)
	if !reflect.DeepEqual(dict.Entries[0].Key, Bytes("k")) || len(list.Items) != 4 || list.Items[3] != 2.5 || tuple[4] != list {
		t.Fatalf("%#v", dict)
	}
}

func TestRejectsTruncationTrailingBytesAndUnknownOpcodes(t *testing.T) {
	for _, data := range []string{"", "\x80\x02K", "\x80\x02K\x01.x", "\x80\x02z.", "invalid pickle"} {
		if _, err := Decode([]byte(data)); err == nil {
			t.Fatalf("%q decoded", data)
		}
	}
	value, err := Decode([]byte("\x8a\x02\x00\x80."))
	if err != nil || value.(*big.Int).Int64() != -32768 {
		t.Fatal(value, err)
	}
}
