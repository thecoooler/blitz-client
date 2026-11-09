package replay

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveDetailsFixtureAndDetachedSnapshots(t *testing.T) {
	z, err := zip.OpenReader("../testdata/sample.tbreplay")
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	read := func(name string) []byte {
		t.Helper()
		f, err := z.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	root := t.TempDir()
	dir := filepath.Join(root, "recording_iCooLER.tbreplay")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	stream := read("data.replay")
	if err := os.WriteFile(filepath.Join(dir, "data.replay"), stream[:65536], 0600); err != nil {
		t.Fatal(err)
	}
	w := NewLiveReplayWatcher(root)
	defer w.Close()
	state, err := w.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if state.Map != nil {
		t.Fatalf("invented map without metadata or catalog: %+v", state.Map)
	}
	w.mapCatalog = map[uint32]string{20: "Карта из каталога"}
	state, err = w.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if state.Battle == nil || state.Battle.ArenaTypeID != 20 || state.Map == nil || state.Map.Name != "" || state.Map.DisplayName != "Карта из каталога" || state.Map.ID == nil || *state.Map.ID != 20 {
		t.Fatalf("map unavailable at battle start: %+v", state.Map)
	}
	if state.Kind != "battle" || state.Result != nil || state.Counters.Frags != nil {
		t.Fatalf("premature result at EOF: %+v", state)
	}
	if state.Mode == nil || state.Mode.BattleCategoryID != 3 || state.Mode.MatchmakerType != 0 || state.Roster.AuthorAccountID == 0 {
		t.Fatalf("missing arena context or author: %+v %+v", state.Mode, state.Roster)
	}
	for _, player := range state.Roster.Allies {
		if player.Nickname == state.Roster.AuthorNickname && player.AccountID != state.Roster.AuthorAccountID {
			t.Fatalf("author ID differs from roster: %+v", player)
		}
	}
	idle, err := w.Poll()
	if err != nil || idle.Result != nil || idle.Kind != "battle" {
		t.Fatalf("idle EOF: %+v %v", idle, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), read("meta.json"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "data.replay"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write(stream[65536:]); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	state, err = w.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if state.Battle == nil || state.Battle.ArenaID != 14044716647467574 {
		t.Fatalf("identity: %+v", state.Battle)
	}
	if state.Map == nil || state.Map.ID == nil || *state.Map.ID != 20 || state.Map.Name != "pliego" {
		t.Fatalf("map: %+v", state.Map)
	}
	if w.recording.metadata.entry == nil {
		t.Fatal("fixture metadata was not cached")
	}
	if state.Vehicle == nil || state.Vehicle.TypeID != 19217 {
		t.Fatalf("vehicle: %+v", state.Vehicle)
	}
	blob, err := decodeBattleResultsDat(read("battle_results.dat"))
	if err != nil {
		t.Fatal(err)
	}
	results, err := decodeBattleResults(blob)
	if err != nil {
		t.Fatal(err)
	}
	if state.Result == nil || results.WinnerTeamNumber == nil || state.Result.WinnerTeamNumber == nil || *state.Result.WinnerTeamNumber != *results.WinnerTeamNumber {
		t.Fatalf("result: %+v", state.Result)
	}
	var row *PlayerResultsInfo
	for i := range results.PlayerResults {
		if results.PlayerResults[i].Info.AccountID == results.Author.AccountID {
			row = &results.PlayerResults[i].Info
			break
		}
	}
	if row == nil || state.Counters.Frags == nil || *state.Counters.Frags != uint64(row.NEnemiesDestroyed) || state.Counters.Damage == nil || *state.Counters.Damage != uint64(results.Author.DamageDealt) || state.Counters.BlockedDamage == nil || *state.Counters.BlockedDamage != uint64(row.DamageBlocked) {
		t.Fatalf("counters: %+v", state.Counters)
	}
	if state.Counters.SpottingDamage != nil {
		t.Fatal("combined assistance presented as spotting")
	}
	encoded, err := json.Marshal(state.Battle)
	if err != nil || !strings.Contains(string(encoded), `"arena_id":"14044716647467574"`) {
		t.Fatalf("identity JSON: %s %v", encoded, err)
	}
	*state.Counters.Damage = 1
	*state.Counters.Frags = 99
	*state.Result.WinnerTeamNumber = 77
	*state.Map.ID = 99
	state.Battle.ArenaID = 1
	state.Mode.BattleCategoryID = 255
	state.Roster.AuthorAccountID = 1
	again, err := w.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if *again.Counters.Damage == 1 || *again.Counters.Frags == 99 || *again.Result.WinnerTeamNumber == 77 || *again.Map.ID != 20 || again.Battle.ArenaID != 14044716647467574 {
		t.Fatal("snapshot was not detached")
	}
	if again.Mode == nil || again.Mode.BattleCategoryID != 3 || again.Roster.AuthorAccountID == 1 {
		t.Fatal("mode or author shares mutable snapshot state")
	}
	if err := os.WriteFile(filepath.Join(dir, "replacement"), stream[:3], 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "replacement"), filepath.Join(dir, "data.replay")); err != nil {
		t.Fatal(err)
	}
	reset, err := w.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if w.recording.metadata.entry != nil {
		t.Fatal("metadata cache survived recording reset")
	}
	if reset.Battle != nil || reset.Mode != nil || reset.Map != nil || reset.Vehicle != nil || reset.Result != nil || reset.Counters.Damage != nil || reset.Counters.Frags != nil {
		t.Fatalf("stale details after reset: %+v", reset)
	}
}

func TestLiveMetadataRequiresMatchingArena(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		data string
		want bool
	}{
		{`{"arenaUniqueId":"14044716647467574","mapId":20,"mapName":"pliego"}`, true},
		{`{"arenaUniqueId":14044716647467574,"mapId":20}`, true},
		{`{"arenaUniqueId":"14044716647467575","mapId":20}`, false},
		{`{"mapId":20}`, false},
		{`{"arenaUniqueId":"14044716647467574",`, false},
	} {
		if err := os.WriteFile(filepath.Join(dir, "meta.json"), []byte(tc.data), 0600); err != nil {
			t.Fatal(err)
		}
		got := (&liveMetadataCache{}).snapshot(dir, 14044716647467574)
		if (got != nil) != tc.want {
			t.Fatalf("metadata %s: %+v", tc.data, got)
		}
	}
}

func TestLiveResultsOutcomeAndUnavailableCounters(t *testing.T) {
	for _, tc := range []struct {
		winner *int32
		team   int32
		want   SessionOutcome
	}{
		{sessionInt32(1), 1, SessionVictory}, {sessionInt32(2), 1, SessionDefeat}, {sessionInt32(2), 2, SessionUnknown}, {sessionInt32(0), 1, SessionDraw}, {nil, 1, SessionUnknown}, {sessionInt32(3), 1, SessionUnknown}, {sessionInt32(0), 0, SessionUnknown},
	} {
		p := livePlayers{author: "author", players: map[uint32]ArenaPlayer{42: {AccountID: 42, Nickname: "author", TeamNumber: 1, VehicleTypeID: 123}}, results: &BattleResults{Author: Author{AccountID: 42, TeamNumber: tc.team}, WinnerTeamNumber: tc.winner}}
		roster, damage := p.snapshot("")
		state := LiveReplayState{Roster: roster, Damage: damage}
		(&LiveReplayWatcher{}).enrichLiveState(&state, &liveRecording{players: p})
		if state.Result == nil || state.Result.Outcome != tc.want || state.Counters.Frags != nil || state.Counters.SpottingDamage != nil {
			t.Fatalf("outcome: %+v %+v", state.Result, state.Counters)
		}
		p.results.Author.AccountID = 99
		state = LiveReplayState{Roster: roster, Damage: damage}
		(&LiveReplayWatcher{}).enrichLiveState(&state, &liveRecording{players: p})
		if state.Result != nil || state.Counters.Frags != nil {
			t.Fatal("other player's result accepted")
		}
	}
}

func TestLiveResultsRequireMatchingArena(t *testing.T) {
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(pickleOf([2]any{123, []byte{8, 7}})); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	raw := append(make([]byte, 22), compressed.Bytes()...)
	packet := &Packet{Payload: Payload{Kind: PayloadUnknown, PacketType: 13}, RawPayload: raw}
	if liveResults(packet, "26.10.0_ruby_apple", true, 123) == nil {
		t.Fatal("matching results rejected")
	}
	if liveResults(packet, "26.10.0_ruby_apple", true, 124) != nil || liveResults(packet, "26.10.0_ruby_apple", false, 123) != nil {
		t.Fatal("unmatched results accepted")
	}
}

func TestLiveRosterLateAuthorAndChangedPlatoons(t *testing.T) {
	p := livePlayers{author: "author"}
	feed := func(id, team uint32, name string, platoon *uint32) {
		t.Helper()
		// Exercise actual protobuf fields: descriptor (2), team (4), account
		// (7), platoon number (11). Team/platoon IDs remain raw identifiers.
		body := []byte{0x12, 2, 0x11, 0x4b, 0x1a}
		body = binary.AppendUvarint(body, uint64(len(name)))
		body = append(body, name...)
		body = binary.AppendUvarint(append(body, 0x20), uint64(team))
		body = binary.AppendUvarint(append(body, 0x38), uint64(id))
		if platoon != nil {
			body = binary.AppendUvarint(append(body, 0x58), uint64(*platoon))
		}
		player, err := decodeArenaPlayer(body)
		if err != nil {
			t.Fatal(err)
		}
		packet := &Packet{Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{UpdateArena: &UpdateArena{Players: &ArenaPlayers{Players: []ArenaPlayer{*player}}}}}}
		if err := p.event(FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: packet}}); err != nil {
			t.Fatal(err)
		}
	}
	platoon := uint32(4)
	feed(2, 3, "ally", &platoon)
	if roster, _ := p.snapshot(""); roster != nil {
		t.Fatal("invented author before author roster entry")
	}
	feed(1, 3, "author", &platoon)
	feed(3, 7, "enemy", &platoon)
	roster, _ := p.snapshot("")
	if roster == nil || roster.AuthorAccountID != 1 || len(roster.Allies) != 2 || len(roster.Enemies) != 1 {
		t.Fatalf("roster: %+v", roster)
	}
	for _, team := range [][]ArenaPlayer{roster.Allies, roster.Enemies} {
		for _, player := range team {
			if player.VehicleTypeID != 19217 || player.PlatoonNumber == nil || *player.PlatoonNumber != 4 {
				t.Fatalf("descriptor/platoon: %+v", player)
			}
		}
	}
	// A number reused on the other team remains an enemy, and later roster
	// updates replace platoon membership without mutating previous snapshots.
	zero := uint32(0)
	feed(2, 3, "ally", &zero)
	changed, _ := p.snapshot("")
	if *changed.Allies[1].PlatoonNumber != 0 || *roster.Allies[1].PlatoonNumber != 4 || changed.Enemies[0].TeamNumber != 7 {
		t.Fatalf("changed roster: %+v", changed)
	}
}
