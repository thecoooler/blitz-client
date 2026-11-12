package replay

import (
	"archive/zip"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLiveWatcherGrowingStreamResetAndRemoval(t *testing.T) {
	root := t.TempDir()
	watcher := NewLiveReplayWatcher(root)
	defer watcher.Close()
	state, err := watcher.Poll()
	if err != nil || state.Kind != "waiting" {
		t.Fatalf("initial: %v %v", state, err)
	}
	if err = os.WriteFile(filepath.Join(root, "recording_saved.tbreplay"), []byte("archive"), 0600); err != nil {
		t.Fatal(err)
	}
	state, err = watcher.Poll()
	if err != nil || state.Kind != "waiting" {
		t.Fatalf("archive opened: %v %v", state, err)
	}
	directory := filepath.Join(root, "recording_iCooLER.tbreplay")
	os.Mkdir(directory, 0700)
	state, err = watcher.Poll()
	if err != nil || state.Kind != "reading" {
		t.Fatalf("empty dir: %v %v", state, err)
	}
	z, err := zip.OpenReader("../testdata/sample.tbreplay")
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	stream, err := z.Open("data.replay")
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	data, err := io.ReadAll(io.LimitReader(stream, 65536))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "data.replay")
	output, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for offset := 0; offset < len(data); offset += 127 {
		_, err = output.Write(data[offset:min(offset+127, len(data))])
		if err != nil {
			t.Fatal(err)
		}
		state, err = watcher.Poll()
		if err != nil {
			t.Fatal(err)
		}
	}
	output.Close()
	if state.Kind != "battle" || len(state.Roster.Allies) != 10 || len(state.Roster.Enemies) != 10 {
		t.Fatalf("roster: %+v", state)
	}
	d := state.Diagnostics
	if d.BytesRead != int64(len(data)) || d.Packets == 0 || d.ClientVersion == "" || d.Players != 20 || !d.AuthorFound || d.ArenaUpdates == 0 || len(d.MethodIDs) == 0 {
		t.Fatalf("decoding diagnostics: %+v", d)
	}
	state.Diagnostics.MethodIDs[0] = ^uint32(0)
	stableDiagnostics, err := watcher.Poll()
	if err != nil || stableDiagnostics.Diagnostics.MethodIDs[0] == ^uint32(0) {
		t.Fatalf("diagnostics share method IDs: %+v %v", stableDiagnostics.Diagnostics, err)
	}
	state = stableDiagnostics
	var own *ArenaPlayer
	for i := range state.Roster.Allies {
		player := &state.Roster.Allies[i]
		if player.Nickname == "iCooLER" {
			own = player
		}
	}
	if own == nil || own.VehicleEntityID == nil {
		t.Fatal("author vehicle missing")
	}
	for _, player := range state.Roster.Enemies {
		if player.TeamNumber == own.TeamNumber {
			t.Fatal("enemy assigned to own team")
		}
	}
	idle, err := watcher.Poll()
	if err != nil || !reflect.DeepEqual(idle, state) {
		t.Fatalf("idle changed snapshot: %v", err)
	}
	*state.Roster.Allies[0].VehicleEntityID = 999999
	stable, err := watcher.Poll()
	if err != nil || *stable.Roster.Allies[0].VehicleEntityID == 999999 {
		t.Fatal("snapshot shares pointer fields")
	}
	replacement := filepath.Join(directory, "replacement")
	os.WriteFile(replacement, data[:3], 0600)
	os.Rename(replacement, path)
	state, err = watcher.Poll()
	if err != nil || state.Kind != "reading" || state.SessionID != 2 {
		t.Fatalf("reset: %+v %v", state, err)
	}
	if state.Diagnostics.Packets != 0 || state.Diagnostics.Players != 0 || state.Diagnostics.BytesRead != 3 {
		t.Fatalf("stale diagnostics after reset: %+v", state.Diagnostics)
	}
	os.Remove(path)
	state, err = watcher.Poll()
	if err != nil || state.Kind != "reading" {
		t.Fatalf("stream removed: %+v %v", state, err)
	}
	os.Remove(directory)
	state, err = watcher.Poll()
	if err != nil || state.Kind != "waiting" {
		t.Fatalf("dir removed: %+v %v", state, err)
	}
}
func TestAppliedDamageAndRosterTotals(t *testing.T) {
	raw := make([]byte, 33)
	binary.LittleEndian.PutUint32(raw, 22)
	binary.LittleEndian.PutUint32(raw[12:], 11)
	binary.LittleEndian.PutUint32(raw[16:], 22)
	binary.LittleEndian.PutUint16(raw[30:], 600)
	raw[32] = 1
	packet := &Packet{Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 9}}, RawPayload: raw}
	hit := AppliedDamage(packet, "26.10.0_ruby_apple")
	if hit == nil || hit.Damage != 600 {
		t.Fatal("applied hit missing")
	}
	if AppliedDamage(packet, "unverified") != nil {
		t.Fatal("unverified version interpreted")
	}
	raw[32] = 0
	if AppliedDamage(packet, "26.10.0_ruby_apple") != nil {
		t.Fatal("bounce counted")
	}
	raw[32] = 1
	binary.LittleEndian.PutUint32(raw, 23)
	if AppliedDamage(packet, "26.10.0_ruby_apple") != nil {
		t.Fatal("invalid receiver counted")
	}
	ownID, enemyID, allyID := uint32(11), uint32(22), uint32(33)
	p := livePlayers{author: "Author", version: "26.10.0_ruby_apple", players: map[uint32]ArenaPlayer{
		1: {AccountID: 1, Nickname: "Author", TeamNumber: 2, VehicleEntityID: &ownID},
		2: {AccountID: 2, Nickname: "Enemy", TeamNumber: 1, VehicleEntityID: &enemyID},
		3: {AccountID: 3, Nickname: "Ally", TeamNumber: 2, VehicleEntityID: &allyID},
	}, pairs: map[[2]uint32]uint64{{11, 22}: 600, {11, 33}: 200, {22, 11}: 300}}
	_, damage := p.snapshot("")
	if damage.Dealt == nil || *damage.Dealt != 600 {
		t.Fatalf("damage: %+v", damage)
	}
	p.version = "unverified"
	_, damage = p.snapshot("")
	if damage.Dealt != nil {
		t.Fatal("unknown damage became zero")
	}
	p.event(FollowEvent{Event: StreamEvent{Kind: EventReset}})
	roster, _ := p.snapshot("Author")
	if roster != nil {
		t.Fatal("reset retained roster")
	}
}

func TestWindows26101LiveDamageAndCounters(t *testing.T) {
	ownID, enemyID := uint32(11), uint32(22)
	p := livePlayers{version: "26.10.1_ruby", players: map[uint32]ArenaPlayer{
		42: {AccountID: 42, Nickname: "Test", TeamNumber: 1, VehicleEntityID: &ownID},
		99: {AccountID: 99, Nickname: "Enemy", TeamNumber: 2, VehicleEntityID: &enemyID},
	}}
	feed := func(packet *Packet) {
		t.Helper()
		if err := p.event(FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: packet}}); err != nil {
			t.Fatal(err)
		}
	}
	feed(basePlayer(t, 42))
	_, initial := p.snapshot("")
	if initial.Assisted != nil || initial.Blocked != nil {
		t.Fatal("missing counters presented as zero")
	}
	raw := make([]byte, 33)
	binary.LittleEndian.PutUint32(raw, enemyID)
	binary.LittleEndian.PutUint32(raw[4:], 9)
	binary.LittleEndian.PutUint32(raw[8:], 21)
	binary.LittleEndian.PutUint32(raw[12:], ownID)
	binary.LittleEndian.PutUint32(raw[16:], enemyID)
	binary.LittleEndian.PutUint16(raw[30:], 600)
	raw[32] = 1
	feed(&Packet{Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 9}}, RawPayload: raw})
	feed(ribbon(42, 17, 706))
	feed(ribbon(42, 5, 540))
	feed(ribbon(99, 5, 999)) // Another player's counter cannot affect the author.
	_, damage := p.snapshot("")
	if damage.Dealt == nil || *damage.Dealt != 600 || damage.Assisted == nil || *damage.Assisted != 706 || damage.Blocked == nil || *damage.Blocked != 540 {
		t.Fatalf("Windows live totals: %+v", damage)
	}
	if p.damageEvents != 1 || p.counterUpdates != 2 {
		t.Fatalf("diagnostic counts: hits=%d counters=%d", p.damageEvents, p.counterUpdates)
	}
	if err := p.event(FollowEvent{Event: StreamEvent{Kind: EventReset}}); err != nil {
		t.Fatal(err)
	}
	if p.damageEvents != 0 || p.counterUpdates != 0 || p.counters.Counters().DamageAssisted != nil {
		t.Fatal("reset retained battle counters")
	}
}

func TestLiveDamageMatchesRealReplayResults(t *testing.T) {
	z, err := zip.OpenReader("../testdata/sample.tbreplay")
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	stream, err := z.Open("data.replay")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(stream)
	stream.Close()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	directory := filepath.Join(root, "recording_iCooLER.tbreplay")
	os.Mkdir(directory, 0700)
	os.WriteFile(filepath.Join(directory, "data.replay"), data, 0600)
	watcher := NewLiveReplayWatcher(root)
	defer watcher.Close()
	state, err := watcher.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if state.Damage.Dealt == nil || *state.Damage.Dealt != 2066 {
		t.Fatalf("real applied damage: %+v", state.Damage)
	}
	for _, ally := range state.Roster.Allies {
		if ally.Nickname != "iCooLER" {
			continue
		}
		if ally.Damage.Dealt == nil || *ally.Damage.Dealt != *state.Damage.Dealt {
			t.Fatalf("author roster damage: %+v", ally.Damage)
		}
	}
	resultsFile, err := z.Open("battle_results.dat")
	if err != nil {
		t.Fatal(err)
	}
	resultsData, err := io.ReadAll(resultsFile)
	resultsFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	blob, err := decodeBattleResultsDat(resultsData)
	if err != nil {
		t.Fatal(err)
	}
	results, err := decodeBattleResults(blob)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range results.PlayerResults {
		if row.Info.AccountID == results.Author.AccountID {
			if state.Damage.Assisted == nil || *state.Damage.Assisted != uint64(row.Info.DamageAssisted1) {
				t.Fatalf("assistance: %+v", state.Damage)
			}
			if state.Damage.Blocked == nil || *state.Damage.Blocked != uint64(row.Info.DamageBlocked) {
				t.Fatalf("blocked: %+v", state.Damage)
			}
			return
		}
	}
	t.Fatal("missing author results")
}

func TestLiveDamageIncludesBothAppliedFlagsAndServerTotal(t *testing.T) {
	for _, version := range []string{"26.8.0_ruby_apple", "26.10.0_ruby_apple", "26.10.1_ruby", "26.10.1_ruby_apple"} {
		t.Run(version, func(t *testing.T) {
			own, enemy, ally := uint32(11), uint32(22), uint32(33)
			p := livePlayers{version: version, players: map[uint32]ArenaPlayer{
				42: {AccountID: 42, Nickname: "Test", TeamNumber: 1, VehicleEntityID: &own},
				99: {AccountID: 99, Nickname: "Enemy", TeamNumber: 2, VehicleEntityID: &enemy},
				77: {AccountID: 77, Nickname: "Ally", TeamNumber: 1, VehicleEntityID: &ally},
			}}
			feed := func(packet *Packet) {
				t.Helper()
				if err := p.event(FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: packet}}); err != nil {
					t.Fatal(err)
				}
			}
			feed(basePlayer(t, 42))
			hit := func(source, target uint32, amount uint16, flag byte) *Packet {
				raw := make([]byte, 33)
				binary.LittleEndian.PutUint32(raw, target)
				binary.LittleEndian.PutUint32(raw[4:], 9)
				binary.LittleEndian.PutUint32(raw[8:], 21)
				binary.LittleEndian.PutUint32(raw[12:], source)
				binary.LittleEndian.PutUint32(raw[16:], target)
				binary.LittleEndian.PutUint16(raw[30:], amount)
				raw[32] = flag
				return &Packet{Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 9}}, RawPayload: raw}
			}
			feed(hit(own, enemy, 2672, 1))
			feed(hit(own, enemy, 500, 3))
			feed(hit(own, enemy, 700, 0))
			feed(hit(own, enemy, 700, 2))
			feed(hit(own, enemy, 700, 255))
			feed(hit(own, ally, 600, 3))
			feed(hit(enemy, own, 400, 3))
			_, damage := p.snapshot("")
			if damage.Dealt == nil || *damage.Dealt != 3172 {
				t.Fatalf("applied flags: %+v", damage)
			}
			// An author-only total recovers direct damage when a hit was not visible.
			feed(ribbon(99, 1, 9999))
			feed(ribbon(42, 1, 3232))
			feed(ribbon(42, 1, 3232))
			_, damage = p.snapshot("")
			if damage.Dealt == nil || *damage.Dealt != 3232 {
				t.Fatalf("cumulative total was lost or double-counted: %+v", damage)
			}
			// Results include other sources of damage and can correct either way.
			for _, total := range []uint32{3753, 3100, 0} {
				p.results = &BattleResults{Author: Author{AccountID: 42, DamageDealt: total}}
				_, damage = p.snapshot("")
				if damage.Dealt == nil || *damage.Dealt != uint64(total) {
					t.Fatalf("final total %d: %+v", total, damage)
				}
			}
			p.results = &BattleResults{Author: Author{AccountID: 99, DamageDealt: 9999}}
			_, damage = p.snapshot("")
			if *damage.Dealt != 3232 {
				t.Fatal("another author's results replaced damage")
			}
		})
	}
}
