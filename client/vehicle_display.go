package client

import (
	"strconv"
	"strings"
)

// VehicleClass is a vehicle's game class, taken from its catalog tags.
type VehicleClass string

const (
	VehicleClassLight     VehicleClass = "light"
	VehicleClassMedium    VehicleClass = "medium"
	VehicleClassHeavy     VehicleClass = "heavy"
	VehicleClassDestroyer VehicleClass = "destroyer"
)

// vehicleClass reads the class from a vehicle's tag list. Other tags describe
// rarity or role, and a vehicle without a class tag keeps an empty class.
func vehicleClass(tags []string) VehicleClass {
	for _, tag := range tags {
		switch tag {
		case "lightTank":
			return VehicleClassLight
		case "mediumTank":
			return VehicleClassMedium
		case "heavyTank":
			return VehicleClassHeavy
		case "AT-SPG":
			return VehicleClassDestroyer
		}
	}
	return ""
}

// VehicleDisplay is the localized vehicle name, tier and class used by battle
// lists.
type VehicleDisplay struct {
	Name  string
	Level int
	Class VehicleClass
}

// LoadVehicleDisplayCatalog indexes installed vehicle metadata by compact type
// descriptor. Nation indices are protocol IDs, not the display order in Nations.
func LoadVehicleDisplayCatalog(gameDir string) (map[uint32]VehicleDisplay, error) {
	resolved, err := ResolveGameDir(gameDir)
	if err != nil {
		return nil, err
	}
	catalog, err := DiscoverClientCatalog(resolved.ResourceDir)
	if err != nil {
		return nil, err
	}
	nationIDs := map[string]uint32{"ussr": 0, "germany": 1, "usa": 2, "china": 3, "france": 4, "uk": 5, "japan": 6, "other": 7, "european": 8}
	displays := make(map[uint32]VehicleDisplay, len(catalog.Tanks))
	for _, tank := range catalog.Tanks {
		nation, known := nationIDs[tank.Nation]
		id, err := strconv.ParseUint(tank.ID, 10, 8)
		if !known || err != nil || tank.Level < 1 {
			continue
		}
		name := tank.ShortUserString
		if name == "" || strings.HasPrefix(name, "#") {
			name = tank.UserString
		}
		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}
		descriptor := uint32(id)<<8 | nation<<4 | 1
		displays[descriptor] = VehicleDisplay{Name: name, Level: int(tank.Level), Class: vehicleClass(tank.Tags)}
	}
	return displays, nil
}
