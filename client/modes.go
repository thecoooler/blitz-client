package client

import "slices"

// ClientGameMode is a battle mode offered by the client. ID is the client's
// mode string, used in localization keys such as battleType/<id>. Name,
// Description and ShortDescription retain localization keys; the description
// fields are null when the client defines no text for them.
type ClientGameMode struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Description      *string `json:"description"`
	ShortDescription *string `json:"shortDescription"`
}

// clientGameModes follows the order the client uses for its mode selector.
// Descriptions are the hangar/battleType keys; a few client builds spell their
// key differently from the mode ID and are listed explicitly.
var clientGameModes = []ClientGameMode{
	{ID: "regular", Name: "battleType/regular", Description: modeText("hangar/battleType/regularDescription")},
	{ID: "training", Name: "battleType/training", Description: modeText("hangar/battleType/trainingDescription")},
	{ID: "rating", Name: "battleType/rating", Description: modeText("hangar/battleType/ratingDescription"), ShortDescription: modeText("hangar/battleType/ratingShortDescription")},
	{ID: "tournament", Name: "battleResults/Footer/Tournament", Description: nil},
	{ID: "arcade", Name: "battleType/arcade", Description: modeText("hangar/battleType/arcadeDescription"), ShortDescription: modeText("hangar/battleType/MadGamesShortDescription")},
	{ID: "scuffle", Name: "battleType/scuffle", Description: modeText("hangar/battleType/scuffleDescription"), ShortDescription: modeText("hangar/battleType/scuffleShortDescription")},
	{ID: "hellgames", Name: "battleType/hellgames", Description: modeText("hangar/battleType/hellgamesDescription"), ShortDescription: modeText("hangar/battleType/hellgamesShortDescription")},
	{ID: "Lunar", Name: "battleType/Lunar", Description: modeText("hangar/battleType/LunarDescription"), ShortDescription: modeText("hangar/battleType/moonFightShortDescription")},
	{ID: "nanomaps", Name: "battleType/nanomaps", Description: modeText("hangar/battleType/nanomapsDescription"), ShortDescription: modeText("hangar/battleType/nanomapsShortDescription")},
	{ID: "vampiric", Name: "battleType/vampiric", Description: modeText("hangar/battleType/vampiricDescription"), ShortDescription: modeText("hangar/battleType/vampiricShortDescription")},
	{ID: "bossmode", Name: "battleType/bossmode", Description: modeText("hangar/battleType/bossmodeDescription"), ShortDescription: modeText("hangar/battleType/bossmodeShortDescription")},
	{ID: "gravitizing", Name: "battleType/gravitizing", Description: modeText("hangar/battleType/gravitizingDescription"), ShortDescription: modeText("hangar/battleType/gravitizingShortDescription")},
	{ID: "tentoten", Name: "battleType/tentoten", Description: modeText("hangar/battleType/tentotenDescription"), ShortDescription: modeText("hangar/battleType/tenToTenShortDescription")},
	{ID: "battle-royale", Name: "battleType/battle-royale", Description: modeText("hangar/battleType/battle-royaleDescription")},
}

func modeText(key string) *string {
	return &key
}

// GameModes returns the client's battle modes as a detached slice.
func GameModes() []ClientGameMode {
	modes := slices.Clone(clientGameModes)
	for i := range modes {
		modes[i].Description = cloneModeText(modes[i].Description)
		modes[i].ShortDescription = cloneModeText(modes[i].ShortDescription)
	}
	return modes
}

func cloneModeText(text *string) *string {
	if text == nil {
		return nil
	}
	return modeText(*text)
}
