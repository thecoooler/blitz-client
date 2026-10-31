package export

// StrictWorldCatalog preserves optional catalog fields in data exports.
type StrictWorldCatalog struct {
	Maps           []StrictWorldMap                  `json:"maps"`
	Mechanics      []StrictWorldMechanic             `json:"mechanics"`
	MechanicsMedia Optional[[]StrictWorldMedia]      `json:"mechanicsMedia,omitzero"`
	MediaIssues    Optional[[]StrictWorldMediaIssue] `json:"mediaIssues,omitzero"`
}
type StrictWorldMap struct {
	Key            string                       `json:"key"`
	NameKey        Optional[string]             `json:"nameKey,omitzero"`
	DescriptionKey Optional[string]             `json:"descriptionKey,omitzero"`
	TacticsKey     Optional[string]             `json:"tacticsKey,omitzero"`
	Images         Optional[[]StrictWorldMedia] `json:"images,omitzero"`
}
type StrictWorldMechanic struct {
	ID             string           `json:"id"`
	NameKey        Optional[string] `json:"nameKey,omitzero"`
	DescriptionKey Optional[string] `json:"descriptionKey,omitzero"`
	Tanks          []string         `json:"tanks"`
}
type StrictWorldMedia struct {
	Source string `json:"source"`
	Frame  int    `json:"frame"`
	File   string `json:"file"`
}
type StrictWorldMediaIssue struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}
