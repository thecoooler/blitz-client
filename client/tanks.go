package client

// Nation describes one nation's vehicle-list key, showroom model folder, and
// Russian label.
type Nation struct {
	ID          string
	ModelFolder string
	Label       string
}

var nations = []Nation{
	{"china", "China", "Китай"},
	{"usa", "USA", "США"},
	{"ussr", "USSR", "СССР"},
	{"germany", "German", "Германия"},
	{"france", "France", "Франция"},
	{"uk", "GB", "Великобритания"},
	{"japan", "Japan", "Япония"},
	{"european", "European", "Европа"},
	{"other", "Other", "Другое"},
}

// Nations returns the known nations in legacy order.
func Nations() []Nation {
	return append([]Nation(nil), nations...)
}

// IsKnownNation reports whether nation is a known vehicle-list key.
func IsKnownNation(nation string) bool {
	for _, candidate := range nations {
		if candidate.ID == nation {
			return true
		}
	}
	return false
}

// ClientTank is a vehicle with both showroom model resources installed.
type ClientTank struct {
	ID          string         `json:"id"`
	Nation      string         `json:"nation"`
	NationLabel string         `json:"nationLabel"`
	Level       *string        `json:"level"`
	UserString  *string        `json:"userString"`
	Name        string         `json:"name"`
	ModelFiles  TankModelFiles `json:"modelFiles"`
}

// TankModelFiles holds the paths of a showroom model's SC2 scene and SCG
// geometry resources.
type TankModelFiles struct {
	Scene    string `json:"scene"`
	Geometry string `json:"geometry"`
}

// ListClientTanks lists vehicles whose installed client has both showroom
// model resources. gameDir accepts the game directory or its Data directory;
// when it is empty, the platform's ordered default locations are searched.
func ListClientTanks(gameDir string) ([]ClientTank, error) {
	resolved, err := ResolveGameDir(gameDir)
	if err != nil {
		return nil, err
	}
	return DiscoverClientTanks(resolved.ResourceDir)
}

// DiscoverClientTanks reads the client's vehicle lists and keeps entries with
// a paired SC2/SCG model.
func DiscoverClientTanks(dataDir string) ([]ClientTank, error) {
	return discoverModels(dataDir)
}
