package export

// StrictLocalization describes the localization included by the v1 exporter.
// A dictionary entry may be null; an absent key is not the same as null.
type StrictLocalization struct {
	RU Optional[map[string]Optional[string]] `json:"ru,omitzero"`
}

// StrictExportIssue describes an issue reported while exporting media.
type StrictExportIssue struct {
	Tank    string `json:"tank"`
	Section string `json:"section"`
	Message string `json:"message"`
}
