package export

import (
	"fmt"
	"strconv"
	"strings"
)

// StrictInteger retains a JSON integer's exact spelling, including an exponent.
// It does not impose a machine integer range on source values.
type StrictInteger string

func (n *StrictInteger) UnmarshalJSON(data []byte) error { return decodeStrictJSON(data, n) }

func (n *StrictInteger) decodeSchema(d *schemaDecoder) error {
	var value StrictNumber
	if err := value.decodeSchema(d); err != nil {
		return err
	}
	if !isJSONInteger(string(value)) {
		return d.valueErr("expected integer, got %s", value)
	}
	*n = StrictInteger(value)
	return nil
}

func (n StrictInteger) MarshalJSON() ([]byte, error) {
	if !isJSONNumber(string(n)) || !isJSONInteger(string(n)) {
		return nil, fmt.Errorf("invalid JSON integer %q", string(n))
	}
	return []byte(n), nil
}

// isJSONInteger reports whether a valid JSON number denotes an integer.
func isJSONInteger(s string) bool {
	mantissa, exponentText, hasExponent := strings.Cut(strings.ToLower(s), "e")
	mantissa = strings.TrimPrefix(mantissa, "-")
	whole, fraction, _ := strings.Cut(mantissa, ".")
	digits := whole + fraction
	if strings.Trim(digits, "0") == "" {
		return true
	}
	trailingZeros := len(digits) - len(strings.TrimRight(digits, "0"))
	exponent := int64(0)
	if hasExponent {
		parsed, err := strconv.ParseInt(exponentText, 10, 64)
		if err != nil {
			// An exponent beyond int64 settles integrality by its sign.
			return !strings.HasPrefix(exponentText, "-")
		}
		exponent = parsed
	}
	return exponent >= int64(len(fraction)-trailingZeros)
}
