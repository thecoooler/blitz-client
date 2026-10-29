package export

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeStrictJSON(t *testing.T) {
	type inner struct {
		Number json.Number `json:"number"`
	}
	type doc struct {
		Items   []inner          `json:"items"`
		Dynamic map[string]inner `json:"dynamic"`
	}
	for _, tc := range []struct{ name, input, errorPart string }{
		{"valid exact number", `{"items":[{"number":9007199254740993}],"dynamic":{}}`, ""},
		{"wrong key case", `{"ITEMS":[]}`, `unknown field "ITEMS"`},
		{"case collision", `{"items":[],"ITEMS":[]}`, `unknown field "ITEMS"`},
		{"unknown nested field", `{"items":[{"number":1,"new":2}]}`, `unknown field "new"`},
		{"duplicate escaped key", `{"items":[],"\u0069tems":[]}`, `duplicate object key "items"`},
		{"duplicate in dynamic map", `{"dynamic":{"a":{"number":1},"a":{"number":2}}}`, `duplicate object key "a"`},
		{"duplicate nested value", `{"items":[{"number":1,"number":2}]}`, `duplicate object key "number"`},
		{"multiple values", `{} {}`, "multiple JSON values"},
		{"trailing malformed", `{} x`, "invalid character"},
		{"empty", ``, "EOF"},
		{"unterminated", `{"items":[`, "unexpected end"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got doc
			err := decodeStrictJSON([]byte(tc.input), &got)
			if tc.errorPart == "" {
				if err != nil {
					t.Fatal(err)
				}
				if got.Items[0].Number != "9007199254740993" {
					t.Fatalf("number: %q", got.Items[0].Number)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.errorPart) {
				t.Fatalf("got %v, want error containing %q", err, tc.errorPart)
			}
		})
	}
}
