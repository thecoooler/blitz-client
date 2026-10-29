package export

import "encoding/json"

// StrictLocalization describes the localization included by the v1 exporter.
// A dictionary entry may be null; an absent key is not the same as null.
type StrictLocalization struct {
	RU Optional[map[string]Optional[string]] `json:"ru,omitempty"`
}

// StrictExportIssue describes an issue reported while exporting media.
type StrictExportIssue struct {
	Tank    string `json:"tank"`
	Section string `json:"section"`
	Message string `json:"message"`
}

func (v *StrictExportIssue) UnmarshalJSON(data []byte) error {
	type plain StrictExportIssue
	var p plain
	if err := decodeStrictJSON(data, &p); err != nil {
		return err
	}
	if err := requireFields(data, "tank", "section", "message"); err != nil {
		return err
	}
	*v = StrictExportIssue(p)
	return nil
}

var _ json.Unmarshaler = (*StrictExportIssue)(nil)
