package export

import (
	"encoding/json"
	"testing"
)

func TestStrictInteger(t *testing.T) {
	for _, tc := range []struct {
		input string
		valid bool
	}{
		{"0", true}, {"-123456789012345678901234567890", true}, {"1.20e1", true}, {"100e-2", true}, {"0e-9999999999999999999999", true}, {"1e9999999999999999999999", true},
		{"0.01", false}, {"1.2e0", false}, {"1e-9999999999999999999999", false}, {`"1"`, false}, {"null", false}, {"true", false},
	} {
		var n StrictInteger
		err := json.Unmarshal([]byte(tc.input), &n)
		if (err == nil) != tc.valid {
			t.Errorf("%s: %v", tc.input, err)
			continue
		}
		if err == nil {
			out, err := json.Marshal(n)
			if err != nil || string(out) != tc.input {
				t.Errorf("%s: round trip %s: %v", tc.input, out, err)
			}
		}
	}
}
