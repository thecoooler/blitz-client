package export

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StrictNumber retains the original spelling of a JSON number, without a
// float64 conversion. Unlike json.Number, it rejects quoted numbers and null.
type StrictNumber string

func (n *StrictNumber) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || (data[0] != '-' && (data[0] < '0' || data[0] > '9')) || !json.Valid(data) {
		return fmt.Errorf("expected JSON number, got %s", data)
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return err
	}
	*n = StrictNumber(number)
	return nil
}

func (n StrictNumber) MarshalJSON() ([]byte, error) {
	var checked StrictNumber
	if err := checked.UnmarshalJSON([]byte(n)); err != nil {
		return nil, err
	}
	return []byte(n), nil
}
