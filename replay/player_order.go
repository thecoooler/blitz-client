package replay

import (
	"slices"
	"strings"

	"github.com/thecoooler/blitz-client/client"
)

// LoadVehicleCatalog enables native battle-list ordering using the installed
// game's localized names and tiers. Call before Poll, on the watcher worker.
// A failure leaves roster discovery working with its existing stable order.
func (w *LiveReplayWatcher) LoadVehicleCatalog(gameDir string) error {
	catalog, err := client.LoadVehicleDisplayCatalog(gameDir)
	if err != nil {
		return err
	}
	w.vehicleCatalog = catalog
	return nil
}

func (w *LiveReplayWatcher) describePlayers(roster *BattleRoster) {
	if roster == nil {
		return
	}
	for _, team := range [][]ArenaPlayer{roster.Allies, roster.Enemies} {
		for i := range team {
			if info, ok := w.vehicleCatalog[team[i].VehicleTypeID]; ok {
				team[i].VehicleName, team[i].VehicleLevel = info.Name, info.Level
			}
		}
	}
}

// OrderBattlePlayers keeps destroyed vehicles last, then orders each group by
// descending tier and ascending localized vehicle name. Ties retain the source
// order; players with unavailable vehicle metadata remain below known tiers.
func OrderBattlePlayers(team []ArenaPlayer) []ArenaPlayer {
	ordered := slices.Clone(team)
	slices.SortStableFunc(ordered, func(a, b ArenaPlayer) int {
		if a.Destroyed != b.Destroyed {
			if a.Destroyed {
				return 1
			}
			return -1
		}
		if a.VehicleLevel != b.VehicleLevel {
			if a.VehicleLevel > b.VehicleLevel {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.VehicleName), strings.ToLower(b.VehicleName))
	})
	return ordered
}

// LoadMapCatalog enables localized map names from the installed client. Call
// before Poll, on the watcher worker. A failed load retains the previous catalog.
func (w *LiveReplayWatcher) LoadMapCatalog(gameDir string) error {
	catalog, err := client.LoadMapDisplayCatalog(gameDir)
	if err != nil {
		return err
	}
	w.mapCatalog = catalog
	return nil
}
