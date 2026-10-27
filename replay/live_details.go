package replay

// LiveBattleIdentity identifies the battle from BasePlayerCreate. ArenaID is
// encoded as a JSON string to preserve all 64 bits in JavaScript clients.
type LiveBattleIdentity struct {
	ArenaID     uint64 `json:"arena_id,string"`
	ArenaTypeID uint32 `json:"arena_type_id"`
}

// LiveBattleMap contains the numeric map ID and the internal map name from
// matching replay metadata, when available. Name is an internal identifier;
// DisplayName is a localized label from an explicitly loaded client catalog.
type LiveBattleMap struct {
	ID          *uint32 `json:"id,omitempty"`
	Name        string  `json:"name,omitempty"`
	DisplayName string  `json:"display_name,omitempty"`
}

// LiveBattleVehicle describes the author's vehicle from the arena roster.
// Name and Level require a successfully loaded vehicle catalog.
type LiveBattleVehicle struct {
	TypeID uint32 `json:"type_id"`
	Name   string `json:"name,omitempty"`
	Level  int    `json:"level,omitempty"`
}

// LiveBattleResult is present only after matching author results arrive.
// Outcome is unknown if the winner or author's team cannot be identified.
type LiveBattleResult struct {
	Outcome          SessionOutcome `json:"outcome"`
	WinnerTeamNumber *int32         `json:"winner_team_number,omitempty"`
}

// LiveBattleCounters are author totals; nil means unavailable. Damage can be
// an incremental estimate until results arrive. Assistance from live ribbons
// combines spotting and immobilization and is not a spotting-only total.
type LiveBattleCounters struct {
	Damage         *uint64 `json:"damage"`
	Frags          *uint64 `json:"frags"`
	SpottingDamage *uint64 `json:"spotting_damage"`
	BlockedDamage  *uint64 `json:"blocked_damage"`
}

func copyCounter(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func (w *LiveReplayWatcher) enrichLiveState(state *LiveReplayState, r *liveRecording) {
	p := &r.players
	if p.hasArena {
		state.Battle = &LiveBattleIdentity{ArenaID: p.arenaID, ArenaTypeID: p.arenaTypeID}
		state.Map = r.metadata.snapshot(r.path, p.arenaID)
		if name, ok := w.mapCatalog[p.arenaTypeID]; ok && (state.Map == nil || state.Map.ID == nil || *state.Map.ID == p.arenaTypeID) {
			if state.Map == nil {
				state.Map = &LiveBattleMap{}
			}
			if state.Map.ID == nil {
				id := p.arenaTypeID
				state.Map.ID = &id
			}
			state.Map.DisplayName = name
		}
	}
	state.Counters.Damage = copyCounter(state.Damage.Dealt)
	state.Counters.BlockedDamage = copyCounter(state.Damage.Blocked)
	if state.Roster == nil {
		return
	}
	var own *ArenaPlayer
	for i := range state.Roster.Allies {
		player := &state.Roster.Allies[i]
		if player.Nickname == state.Roster.AuthorNickname {
			own = player
			break
		}
	}
	if own == nil {
		return
	}
	if own.VehicleTypeID != 0 {
		state.Vehicle = &LiveBattleVehicle{TypeID: own.VehicleTypeID, Name: own.VehicleName, Level: own.VehicleLevel}
	}
	results := p.results
	if results == nil || own.AccountID == 0 || results.Author.AccountID != own.AccountID {
		return
	}
	result := &LiveBattleResult{Outcome: SessionUnknown}
	if results.WinnerTeamNumber != nil {
		winner := *results.WinnerTeamNumber
		result.WinnerTeamNumber = &winner
		if (results.Author.TeamNumber == 1 || results.Author.TeamNumber == 2) && int32(own.TeamNumber) == results.Author.TeamNumber {
			switch winner {
			case 0:
				result.Outcome = SessionDraw
			case results.Author.TeamNumber:
				result.Outcome = SessionVictory
			case 1, 2:
				result.Outcome = SessionDefeat
			}
		}
	}
	state.Result = result
	if results.ModeMapID != 0 {
		id := results.ModeMapID & 0xffff
		if state.Map == nil {
			state.Map = &LiveBattleMap{}
		}
		if state.Map.ID == nil {
			state.Map.ID = &id
		}
	}
	for _, row := range results.PlayerResults {
		if row.Info.AccountID == own.AccountID {
			frags := uint64(row.Info.NEnemiesDestroyed)
			state.Counters.Frags = &frags
			break
		}
	}
}
