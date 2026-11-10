package client

// clientGameModeNames maps the client's serialized eGameMode values to the mode
// strings used by localization keys such as battleType/<id>. The table was read
// from the client executable's eGameMode serializer; IDs omitted here are not
// battle modes the exporter knows.
var clientGameModeNames = map[int64]string{
	1:  "regular",
	2:  "training",
	3:  "company",
	4:  "tournament",
	5:  "clan",
	6:  "tutorial",
	7:  "rating",
	8:  "arcade",
	18: "newbie_tutorial_neo",
	21: "virtual",
	22: "scuffle",
	23: "hellgames",
	24: "Lunar",
	25: "nanomaps",
	26: "vampiric",
	27: "bossmode",
	28: "tournament_hellgames",
	29: "tournament_arcade",
	30: "tournament_realistic",
	31: "tournament_lunar",
	32: "tournament_nanomaps",
	33: "tournament_vampiric",
	34: "tournament_bossmode",
	35: "quick_tournament_hellgames",
	36: "quick_tournament_arcade",
	37: "quick_tournament_realistic",
	38: "quick_tournament_lunar",
	39: "quick_tournament_nanomaps",
	40: "quick_tournament_vampiric",
	41: "quick_tournament_bossmode",
	42: "gravitizing",
	43: "tournament_gravitizing",
	44: "quick_tournament_gravitizing",
	45: "tentoten",
	46: "tournament_tentoten",
	47: "quick_tournament_tentoten",
	48: "battle-royale",
}

// GameModeName returns the client mode string of a serialized game mode value.
func GameModeName(id int64) (string, bool) {
	name, ok := clientGameModeNames[id]
	return name, ok
}
