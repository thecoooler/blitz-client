package export

import (
	"fmt"
	"strconv"
	"strings"
)

// StrictInteger retains a JSON integer's exact spelling, including an exponent.
// It does not impose a machine integer range on source values.
type StrictInteger string

func (n *StrictInteger) UnmarshalJSON(data []byte) error {
	var value StrictNumber
	if err := value.UnmarshalJSON(data); err != nil {
		return err
	}
	s := string(value)
	mantissa, exponentText, hasExponent := strings.Cut(strings.ToLower(s), "e")
	mantissa = strings.TrimPrefix(mantissa, "-")
	whole, fraction, _ := strings.Cut(mantissa, ".")
	digits := whole + fraction
	if strings.Trim(digits, "0") == "" {
		*n = StrictInteger(value)
		return nil
	}
	trailingZeros := len(digits) - len(strings.TrimRight(digits, "0"))
	exponent := int64(0)
	if hasExponent {
		parsed, err := strconv.ParseInt(exponentText, 10, 64)
		if err != nil {
			// An exponent larger than int64's range is necessarily large
			// enough to settle integrality without allocating a huge integer.
			if strings.HasPrefix(exponentText, "-") {
				return fmt.Errorf("expected integer, got %s", data)
			}
			*n = StrictInteger(value)
			return nil
		}
		exponent = parsed
	}
	if exponent < int64(len(fraction)-trailingZeros) {
		return fmt.Errorf("expected integer, got %s", data)
	}
	*n = StrictInteger(value)
	return nil
}

func (n StrictInteger) MarshalJSON() ([]byte, error) {
	var checked StrictInteger
	if err := checked.UnmarshalJSON([]byte(n)); err != nil {
		return nil, err
	}
	return []byte(n), nil
}
