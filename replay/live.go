package replay

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
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
	Map         *LiveBattleMap        `json:"map,omitempty"`
	Vehicle     *LiveBattleVehicle    `json:"vehicle,omitempty"`
	Result      *LiveBattleResult     `json:"result,omitempty"`
	Counters    LiveBattleCounters    `json:"counters"`
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
	AuthorNickname string        `json:"author_nickname"`
	Allies         []ArenaPlayer `json:"allies"`
	Enemies        []ArenaPlayer `json:"enemies"`
}

// Nil damage means unavailable, never zero.
type BattleDamage struct {
	Dealt    *uint64 `json:"dealt"`
	Assisted *uint64 `json:"damage_assisted_1"`
	Blocked  *uint64 `json:"damage_blocked"`
}
type DamageEvent struct {
	SourceEntityID, TargetEntityID uint32
	Damage                         uint16
}

func SupportsDamageEvents(version string) bool {
	return version == "26.8.0_ruby_apple" || version == "26.10.0_ruby_apple" || version == "26.10.1_ruby"
}

// AppliedDamage accepts supported versions with matching damage envelopes.
func AppliedDamage(packet *Packet, version string) *DamageEvent {
	if !SupportsDamageEvents(version) || packet == nil || packet.Payload.Kind != PayloadEntityMethod || packet.Payload.EntityMethod == nil || packet.Payload.EntityMethod.SubType != 9 || len(packet.RawPayload) != 33 {
		return nil
	}
	raw := packet.RawPayload
	// Both observed applied-hit flags (1 and 3) carry actual HP damage.
	// Flag 3 appears on hits that were previously omitted from the total.
	if (raw[32] != 1 && raw[32] != 3) || binary.LittleEndian.Uint32(raw) != binary.LittleEndian.Uint32(raw[16:]) {
		return nil
	}
	damage := binary.LittleEndian.Uint16(raw[30:])
	if damage == 0 {
		return nil
	}
	return &DamageEvent{binary.LittleEndian.Uint32(raw[12:]), binary.LittleEndian.Uint32(raw[16:]), damage}
}

type livePlayers struct {
	packets, arenaUpdates        uint64
	damageEvents, counterUpdates uint64
	methods                      map[uint32]bool
	author, version              string
	arenaID                      uint64
	arenaTypeID                  uint32
	hasArena                     bool
	players                      map[uint32]ArenaPlayer
	pairs                        map[[2]uint32]uint64
	health                       healthDamageDecoder
	healthPairs                  map[[2]uint32]uint64
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
	roster := &BattleRoster{AuthorNickname: author, Allies: []ArenaPlayer{}, Enemies: []ArenaPlayer{}}
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
			health, known := p.health.vehicles[v]
			player.Destroyed = known && health.hp == 0
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
			roster.Allies = append(roster.Allies, player)
		} else {
			roster.Enemies = append(roster.Enemies, player)
		}
	}
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
			if row.Info.AccountID == own.AccountID {
				assisted, blocked := uint64(row.Info.DamageAssisted1), uint64(row.Info.DamageBlocked)
				damage.Assisted, damage.Blocked = &assisted, &blocked
				break
			}
		}
	}
	return roster, damage
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
}

func NewLiveReplayWatcher(directory string) *LiveReplayWatcher {
	return &LiveReplayWatcher{directory: directory}
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
	entries, err := os.ReadDir(w.directory)
	if errors.Is(err, fs.ErrNotExist) {
		w.Close()
		return LiveReplayState{Kind: "waiting"}, nil
	}
	if err != nil {
		return state, err
	}
	var newest, nickname string
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
		path := filepath.Join(w.directory, name)
		info, e := os.Stat(filepath.Join(path, "data.replay"))
		if e != nil {
			info, e = entry.Info()
		}
		if e != nil {
			return state, e
		}
		t := info.ModTime()
		if newest == "" || t.After(modified) || (t.Equal(modified) && path > newest) {
			newest, nickname, modified = path, nick, t
		}
	}
	if newest == "" {
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
		return state, err
	}
	roster, damage := r.players.snapshot(r.nickname)
	w.describePlayers(roster)
	kind := "reading"
	if roster != nil {
		kind = "battle"
	}
	state = LiveReplayState{Kind: kind, Path: newest, SessionID: r.session, Roster: roster, Damage: damage}
	w.enrichLiveState(&state, r)
	return state, nil
}
