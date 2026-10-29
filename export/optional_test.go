package export

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOptionalRoundTrip(t *testing.T) {
	type nested struct {
		Number json.Number `json:"number"`
	}
	type doc struct {
		Field Optional[nested] `json:"field,omitempty"`
	}
	for _, input := range []string{`{}`, `{"field":null}`, `{"field":{"number":9007199254740993}}`} {
		var got doc
		if err := decodeStrictJSON([]byte(input), &got); err != nil {
			t.Fatal(err)
		}
		if got.Field.Present() != (input != `{}`) || got.Field.Null() != (input == `{"field":null}`) {
			t.Fatalf("state for %s: %#v", input, got.Field)
		}
		if input == `{"field":{"number":9007199254740993}}` {
			value, ok := got.Field.Get()
			if !ok || value.Number != "9007199254740993" {
				t.Fatalf("value: %#v %v", value, ok)
			}
		}
		encoded, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		if string(encoded) != input {
			t.Fatalf("got %s, want %s", encoded, input)
		}
	}
	var v doc
	if err := decodeStrictJSON([]byte(`{"field":{"Number":1}}`), &v); err == nil || !strings.Contains(err.Error(), `unknown field "Number"`) {
		t.Fatalf("wrong field case: %v", err)
	}
	if err := decodeStrictJSON([]byte(`{"field":{"number":1,"future":true}}`), &v); err == nil || !strings.Contains(err.Error(), `unknown field "future"`) {
		t.Fatalf("unknown field: %v", err)
	}
}
