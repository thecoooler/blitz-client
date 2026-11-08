package export

// StrictGameMode describes one battle mode record. Name, Description and
// ShortDescription are localization keys, not localized text; the description
// fields are null when the client defines no text for them.
type StrictGameMode struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Description      Optional[string] `json:"description,omitzero"`
	ShortDescription Optional[string] `json:"shortDescription,omitzero"`
}
