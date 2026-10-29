package export

import "fmt"

// StrictNull represents an XML field for which only null has been observed.
// Use Optional[StrictNull]: absent and explicit null remain distinct.
type StrictNull struct{}

func (*StrictNull) UnmarshalJSON([]byte) error {
	return fmt.Errorf("expected null")
}

func (StrictNull) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("use NullValue to represent null")
}
