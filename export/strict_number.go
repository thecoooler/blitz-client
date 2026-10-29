package export

import (
	"encoding/json"
	"fmt"
)

// StrictNumber retains the original spelling of a JSON number, without a
// float64 conversion. Unlike json.Number, it rejects quoted numbers and null.
type StrictNumber string

func (n *StrictNumber) UnmarshalJSON(data []byte) error { return decodeStrictJSON(data, n) }

func (n *StrictNumber) decodeSchema(d *schemaDecoder) error {
	if c := d.peek(); c != '-' && (c < '0' || c > '9') {
		return d.valueErr("expected number, got %s", d.tokenName())
	}
	*n = StrictNumber(d.readNumber())
	return nil
}

func (n StrictNumber) MarshalJSON() ([]byte, error) {
	if !isJSONNumber(string(n)) {
		return nil, fmt.Errorf("invalid JSON number %q", string(n))
	}
	return []byte(n), nil
}

func isJSONNumber(s string) bool {
	if s == "" || s[0] != '-' && (s[0] < '0' || s[0] > '9') {
		return false
	}
	var n json.Number
	return json.Unmarshal([]byte(s), &n) == nil
}
