package replay

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/thecoooler/blitz-client/client"
)

type LiveReplayState struct {
	Kind        string                `json:"kind"`
	Path        string                `json:"path,omitempty"`
	SessionID   uint64                `json:"session_id,omitempty"`
	Roster      *BattleRoster         `json:"roster,omitempty"`
	Damage      BattleDamage          `json:"damage"`
	Diagnostics LiveReplayDiagnostics `json:"diagnostics"`
	Battle      *LiveBattleIdentity   `json:"battle,omitempty"`
	Mode        *LiveBattleMode       `json:"mode,omitempty"`
	Map         *LiveBattleMap        `json:"map,omitempty"`
	Vehicle     *LiveBattleVehicle    `json:"vehicle,omitempty"`
	// Positions holds the last reported placement of each roster vehicle. A
	// position stops updating for a vehicle out of view; compare its Clock with
	// the replay clock to tell a live position from a last known one.
	Positions []LiveVehiclePosition `json:"positions,omitempty"`
	Result    *LiveBattleResult     `json:"result,omitempty"`
	Counters  LiveBattleCounters    `json:"counters"`
	// Clock is the battle clock of the newest packet, in seconds since the
	// recording started. It is what the recording reports, not a wall clock,
	// so a paused recording pauses it. Nil means no packet has been read yet.
	Clock *float32 `json:"clock,omitempty"`
}

// LiveReplayDiagnostics is a detached snapshot of stream decoding progress.
type LiveReplayDiagnostics struct {
	ClientVersion     string   `json:"client_version"`
	BytesRead         int64    `json:"bytes_read"`
	PendingBytes      int      `json:"pending_bytes"`
	Packets           uint64   `json:"packets"`
	Players           int      `json:"players"`
	Author            string   `json:"author"`
	AuthorFound       bool     `json:"author_found"`
	ArenaUpdates      uint64   `json:"arena_updates"`
	MethodIDs         []uint32 `json:"method_ids,omitempty"`
	DamageEvents      uint64   `json:"damage_events"`
	CounterUpdates    uint64   `json:"counter_updates"`
	DamageSupported   bool     `json:"damage_supported"`
	CountersSupported bool     `json:"counters_supported"`
}
type BattleRoster struct {
	AuthorNickname  string        `json:"author_nickname"`
	AuthorAccountID uint32        `json:"author_account_id"`
	Allies          []ArenaPlayer `json:"allies"`
	Enemies         []ArenaPlayer `json:"enemies"`
}

// Nil damage means unavailable, never zero.
type BattleDamage struct {
	Dealt *uint64 `json:"dealt"`
	// Assisted combines assistance for spotting opponents with assistance for
	// immobilizing them, which is the single total the client reports to a
	// player. The battle results keep the two kinds in separate fields.
	Assisted *uint64 `json:"damage_assisted"`
	Blocked  *uint64 `json:"damage_blocked"`
}
type DamageEvent struct {
	SourceEntityID, TargetEntityID uint32
	Damage                         uint16
}

func SupportsDamageEvents(version string) bool {
	return version == "26.8.0_ruby_apple" || version == "26.10.0_ruby_apple" || version == "26.10.1_ruby" || version == "26.10.1_ruby_apple"
}

// AppliedDamage accepts supported versions with matching damage envelopes.
func AppliedDamage(packet *Packet, version string) *DamageEvent {
	if !SupportsDamageEvents(version) || packet == nil || packet.Payload.Kind != PayloadEntityMethod || packet.Payload.EntityMethod == nil || packet.Payload.EntityMethod.SubType != 9 {
		return nil
	}
	body, ok := parseMethod9Body(packet.RawPayload)
	if !ok {
		return nil
	}
	// Both observed applied-hit flags (1 and 3) carry actual HP damage.
	// Flag 3 appears on hits that were previously omitted from the total.
	if (body.Flag != 1 && body.Flag != 3) || body.EntityID != body.TargetResultID || body.Damage == 0 {
		return nil
	}
	return &DamageEvent{body.SourceResultID, body.TargetResultID, body.Damage}
}

type livePlayers struct {
	packets, arenaUpdates        uint64
	damageEvents, counterUpdates uint64
	methods                      map[uint32]bool
	author, version              string
	arenaID                      uint64
	arenaTypeID                  uint32
	mode                         *LiveBattleMode
	hasArena                     bool
	clock                        float32
	hasClock                     bool
	players                      map[uint32]ArenaPlayer
	pairs                        map[[2]uint32]uint64
	health                       healthDamageDecoder
	healthPairs                  map[[2]uint32]uint64
	positions                    vehiclePositionDecoder
	counters                     BattleCounterDecoder
	results                      *BattleResults
}

func (p *livePlayers) event(e FollowEvent) error {
	switch e.Event.Kind {
	case EventReset:
		*p = livePlayers{}
	case EventHeader:
		p.version = e.Event.ClientVersion
	case EventPacket:
		p.packets++
		packet := e.Event.Packet
		p.observeClock(packet.ClockSecs)
		if method := packet.Payload.EntityMethod; method != nil {
			if p.methods == nil {
				p.methods = map[uint32]bool{}
			}
			p.methods[method.SubType] = true
			if method.UpdateArena != nil {
				p.arenaUpdates++
			}
		}
		if p.counters.OnPacket(packet, e.Event.Seq, p.version) != nil {
			p.counterUpdates++
		}
		if packet.Payload.Kind == PayloadBasePlayerCreate {
			p.author = packet.Payload.AuthorNickname
			p.arenaID, p.arenaTypeID = packet.Payload.ArenaUniqueID, packet.Payload.ArenaTypeID
			p.mode = nil
			if args := packet.Payload.Arguments; args != nil {
				p.mode = &LiveBattleMode{BattleCategoryID: args.BattleCategoryID, MatchmakerType: args.MatchmakerType}
			}
			p.hasArena = true
		}
		if method := packet.Payload.EntityMethod; method != nil && method.UpdateArena != nil && method.UpdateArena.Players != nil {
			if p.players == nil {
				p.players = map[uint32]ArenaPlayer{}
			}
			for _, player := range method.UpdateArena.Players.Players {
				if player.Nickname != "" && player.TeamNumber != 0 {
					p.players[player.AccountID] = player
				}
			}
		}
		if hit := AppliedDamage(packet, p.version); hit != nil {
			p.damageEvents++
			if p.pairs == nil {
				p.pairs = map[[2]uint32]uint64{}
			}
			key := [2]uint32{hit.SourceEntityID, hit.TargetEntityID}
			p.pairs[key] = saturatingAdd(p.pairs[key], uint64(hit.Damage))
		}
		p.positions.onPacket(packet, p.version)
		if hit := p.health.onPacket(packet, p.version); hit != nil {
			if p.healthPairs == nil {
				p.healthPairs = map[[2]uint32]uint64{}
			}
			key := [2]uint32{hit.SourceEntityID, hit.TargetEntityID}
			p.healthPairs[key] = saturatingAdd(p.healthPairs[key], uint64(hit.Damage))
		}
		if results := liveResults(packet, p.version, p.hasArena, p.arenaID); results != nil {
			p.results = results
		}
	}
	return nil
}

// observeClock keeps the battle clock the recording last reported. Some packets
// carry a zero clock, so the largest report of the session stands: a later zero
// must not rewind a running battle. A reset clears it with the rest of the
// battle through livePlayers' zero value.
func (p *livePlayers) observeClock(clock float32) {
	if math.IsNaN(float64(clock)) || math.IsInf(float64(clock), 0) {
		return
	}
	if !p.hasClock || clock > p.clock {
		p.clock, p.hasClock = clock, true
	}
}

// snapshotClock returns the battle clock, or nil while no packet has been read.
func (p *livePlayers) snapshotClock() *float32 {
	if !p.hasClock {
		return nil
	}
	clock := p.clock
	return &clock
}

func saturatingAdd(a, b uint64) uint64 {
	if math.MaxUint64-a < b {
		return math.MaxUint64
	}
	return a + b
}
func liveResults(packet *Packet, version string, hasArena bool, arenaID uint64) *BattleResults {
	if !SupportsBattleCounters(version) || packet.Payload.Kind != PayloadUnknown || packet.Payload.PacketType != 13 || len(packet.RawPayload) <= 22 {
		return nil
	}
	r, err := zlib.NewReader(bytes.NewReader(packet.RawPayload[22:]))
	if err != nil {
		return nil
	}
	defer r.Close()
	const limit = 2 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil || len(data) > limit {
		return nil
	}
	resultArena, blob, err := decodeBattleResultsDatWithArena(data)
	if err != nil || !hasArena || arenaID == 0 || resultArena != arenaID {
		return nil
	}
	results, err := decodeBattleResults(blob)
	if err != nil {
		return nil
	}
	return results
}
func (p *livePlayers) snapshot(fallback string) (*BattleRoster, BattleDamage) {
	author := p.author
	if author == "" {
		author = fallback
	}
	var own *ArenaPlayer
	for _, player := range p.players {
		if player.Nickname == author {
			copy := player
			own = &copy
			break
		}
	}
	if own == nil {
		return nil, BattleDamage{}
	}
	roster := &BattleRoster{AuthorNickname: author, AuthorAccountID: own.AccountID, Allies: []ArenaPlayer{}, Enemies: []ArenaPlayer{}}
	ownIndex := -1
	ids := make([]uint32, 0, len(p.players))
	for id := range p.players {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		player := p.players[id]
		if player.VehicleEntityID != nil {
			v := *player.VehicleEntityID
			player.VehicleEntityID = &v
			if health, known := p.health.vehicles[v]; known {
				hitpoints, maxHitpoints := health.hp, health.peak
				player.Hitpoints, player.MaxHitpoints = &hitpoints, &maxHitpoints
				player.Destroyed = health.hp == 0
			}
		}
		if player.ClanTag != nil {
			v := *player.ClanTag
			player.ClanTag = &v
		}
		if player.PlatoonNumber != nil {
			v := *player.PlatoonNumber
			player.PlatoonNumber = &v
		}
		if player.TeamNumber == own.TeamNumber {
			if ownIndex < 0 && player.Nickname == own.Nickname {
				ownIndex = len(roster.Allies)
			}
			roster.Allies = append(roster.Allies, player)
		} else {
			roster.Enemies = append(roster.Enemies, player)
		}
	}
	p.applyLiveDamage(roster)
	damage := BattleDamage{}
	if own.VehicleEntityID != nil && SupportsDamageEvents(p.version) {
		var total uint64
		var healthTotal uint64
		for _, enemy := range roster.Enemies {
			if enemy.VehicleEntityID != nil {
				total = saturatingAdd(total, p.pairs[[2]uint32{*own.VehicleEntityID, *enemy.VehicleEntityID}])
				healthTotal = saturatingAdd(healthTotal, p.healthPairs[[2]uint32{*own.VehicleEntityID, *enemy.VehicleEntityID}])
			}
		}
		total = max(total, healthTotal)
		damage.Dealt = &total
	}
	if value := p.counters.Counters().DamageBlocked; value != nil {
		v := uint64(*value)
		damage.Blocked = &v
	}
	if value := p.counters.Counters().DamageAssisted; value != nil {
		v := uint64(*value)
		damage.Assisted = &v
	}
	if value := p.counters.Counters().DamageDealt; value != nil {
		v := uint64(*value)
		if damage.Dealt == nil || v > *damage.Dealt {
			damage.Dealt = &v
		}
	}
	if p.results != nil && p.results.Author.AccountID == own.AccountID {
		dealt := uint64(p.results.Author.DamageDealt)
		damage.Dealt = &dealt
		for _, row := range p.results.PlayerResults {
			applyResultDamage(roster, row)
			if row.Info.AccountID == own.AccountID {
				assisted := uint64(row.Info.DamageAssisted1) + uint64(row.Info.DamageAssisted2)
				blocked := uint64(row.Info.DamageBlocked)
				damage.Assisted, damage.Blocked = &assisted, &blocked
			}
		}
	}
	if ownIndex >= 0 {
		roster.Allies[ownIndex].Damage = damage
	}
	return roster, damage
}

// applyLiveDamage attributes direct damage to each ally from applied hits.
// Only the author receives live counter ribbons, so assisted and blocked
// values for other players stay nil until battle results arrive.
func (p *livePlayers) applyLiveDamage(roster *BattleRoster) {
	if !SupportsDamageEvents(p.version) {
		return
	}
	for i := range roster.Allies {
		player := &roster.Allies[i]
		if player.VehicleEntityID == nil {
			continue
		}
		var dealt, health uint64
		for _, enemy := range roster.Enemies {
			if enemy.VehicleEntityID == nil {
				continue
			}
			key := [2]uint32{*player.VehicleEntityID, *enemy.VehicleEntityID}
			dealt = saturatingAdd(dealt, p.pairs[key])
			health = saturatingAdd(health, p.healthPairs[key])
		}
		dealt = max(dealt, health)
		player.Damage.Dealt = &dealt
	}
}

// applyResultDamage replaces one ally's live estimate with final results. The
// two assisted-damage fields add up to the assistance the client reports.
func applyResultDamage(roster *BattleRoster, row PlayerResults) {
	for i := range roster.Allies {
		player := &roster.Allies[i]
		if player.AccountID == 0 || player.AccountID != row.Info.AccountID {
			continue
		}
		dealt := uint64(row.Info.DamageDealt)
		assisted := uint64(row.Info.DamageAssisted1) + uint64(row.Info.DamageAssisted2)
		blocked := uint64(row.Info.DamageBlocked)
		player.Damage = BattleDamage{Dealt: &dealt, Assisted: &assisted, Blocked: &blocked}
	}
}

type liveRecording struct {
	path, nickname string
	session        uint64
	follower       *streamFollower
	players        livePlayers
	metadata       liveMetadataCache
}

// LiveReplayWatcher discovers recording directories and follows only the newest.
// Poll and Close must be serialized. Idle EOF is not a battle end.
type LiveReplayWatcher struct {
	directory      string
	recording      *liveRecording
	vehicleCatalog map[uint32]client.VehicleDisplay
	mapCatalog     map[uint32]string
	// discovery remembers the last directory scan so a caller polling many
	// times per second reads only the followed replay. A zero interval rescans
	// on every poll, which is what tests and short-lived callers want.
	discovery         recordingDiscovery
	discoveryInterval time.Duration
	// now is replaced by tests; nil means time.Now.
	now func() time.Time
}

// defaultDiscoveryInterval bounds how often the directory is rescanned. The
// followed data.replay is still polled on every call; only discovering a new
// recording directory is throttled, so a session switch is noticed within it.
const defaultDiscoveryInterval = time.Second

// recordingDiscovery is the memoized result of one directory scan.
type recordingDiscovery struct {
	path, nickname string
	found          bool
	checked        time.Time
}

func NewLiveReplayWatcher(directory string) *LiveReplayWatcher {
	return &LiveReplayWatcher{directory: directory, discoveryInterval: defaultDiscoveryInterval}
}

// forgetDiscovery forces the next poll to rescan the directory.
func (w *LiveReplayWatcher) forgetDiscovery() {
	w.discovery = recordingDiscovery{}
}

// discover returns the newest recording directory, reusing the previous scan
// while it is still fresh. An unreadable directory aborts the watch.
func (w *LiveReplayWatcher) discover() (path, nickname string, found bool, err error) {
	now := time.Now()
	if w.now != nil {
		now = w.now()
	}
	if w.discovery.checked != (time.Time{}) && now.Sub(w.discovery.checked) < w.discoveryInterval {
		return w.discovery.path, w.discovery.nickname, w.discovery.found, nil
	}
	w.discovery = recordingDiscovery{checked: now}
	entries, err := os.ReadDir(w.directory)
	if errors.Is(err, fs.ErrNotExist) {
		return "", "", false, nil
	}
	if err != nil {
		w.forgetDiscovery()
		return "", "", false, err
	}
	var modified time.Time
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || !strings.HasPrefix(name, "recording_") || !strings.HasSuffix(name, ".tbreplay") {
			continue
		}
		nick := strings.TrimSuffix(strings.TrimPrefix(name, "recording_"), ".tbreplay")
		if nick == "" {
			continue
		}
		candidate := filepath.Join(w.directory, name)
		info, e := os.Stat(filepath.Join(candidate, "data.replay"))
		if e != nil {
			info, e = entry.Info()
		}
		if e != nil {
			w.forgetDiscovery()
			return "", "", false, e
		}
		t := info.ModTime()
		if !found || t.After(modified) || (t.Equal(modified) && candidate > path) {
			path, nickname, modified, found = candidate, nick, t, true
		}
	}
	w.discovery = recordingDiscovery{path: path, nickname: nickname, found: found, checked: now}
	return path, nickname, found, nil
}
func (w *LiveReplayWatcher) Close() error {
	if w.recording != nil {
		err := w.recording.follower.Close()
		w.recording = nil
		return err
	}
	return nil
}
func (w *LiveReplayWatcher) Poll() (state LiveReplayState, err error) {
	defer func() {
		if r := w.recording; r != nil {
			state.Path = r.path
			state.SessionID = r.session
			d := LiveReplayDiagnostics{ClientVersion: r.players.version,
				BytesRead: r.follower.readOffset, PendingBytes: len(r.follower.decoder.pending),
				Packets: r.players.packets, Players: len(r.players.players), ArenaUpdates: r.players.arenaUpdates,
				Author:       r.players.author,
				DamageEvents: r.players.damageEvents, CounterUpdates: r.players.counterUpdates,
				DamageSupported: SupportsDamageEvents(r.players.version), CountersSupported: SupportsBattleCounters(r.players.version)}
			if d.Author == "" {
				d.Author = r.nickname
			}
			for _, player := range r.players.players {
				if player.Nickname == d.Author {
					d.AuthorFound = true
					break
				}
			}
			for id := range r.players.methods {
				d.MethodIDs = append(d.MethodIDs, id)
			}
			sort.Slice(d.MethodIDs, func(i, j int) bool { return d.MethodIDs[i] < d.MethodIDs[j] })
			state.Diagnostics = d
		}
		if err != nil {
			w.Close()
		}
	}()
	newest, nickname, found, err := w.discover()
	if err != nil {
		return state, err
	}
	if !found {
		w.Close()
		return LiveReplayState{Kind: "waiting"}, nil
	}
	if w.recording == nil || w.recording.path != newest {
		w.Close()
		w.recording = &liveRecording{path: newest, nickname: nickname, session: 1, follower: newStreamFollower(filepath.Join(newest, "data.replay"))}
	}
	r := w.recording
	_, err = r.follower.Poll(func(e FollowEvent) error {
		r.session = e.SessionID
		if e.Event.Kind == EventReset {
			r.metadata = liveMetadataCache{}
		}
		return r.players.event(e)
	})
	if err != nil {
		// The followed directory may have gone away; rescan on the next poll
		// instead of waiting for the discovery interval.
		w.forgetDiscovery()
		return state, err
	}
	roster, damage := r.players.snapshot(r.nickname)
	w.describePlayers(roster)
	kind := "reading"
	if roster != nil {
		kind = "battle"
	}
	state = LiveReplayState{Kind: kind, Path: newest, SessionID: r.session, Roster: roster, Damage: damage,
		Clock: r.players.snapshotClock()}
	w.enrichLiveState(&state, r)
	return state, nil
}
