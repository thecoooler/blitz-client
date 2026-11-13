package replay

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/thecoooler/blitz-client/internal/jsonx"
)

// SessionStats contains totals for one replay author. Unknown outcomes are
// counted separately and excluded from the win-rate denominator. Survived
// counts battles the author finished with positive hitpoints.
type SessionStats struct {
	AccountID                             uint32
	Nickname                              string
	Battles, Wins, Losses, Draws, Unknown int
	Survived                              int
	Damage, Assisted, Blocked, Destroyed  uint64
	History                               []SessionBattleResult
	// Platoon aggregates the author's platoon mates over the battles they
	// played together, in order of first appearance.
	Platoon []SessionPlatoonMember
}

// SessionPlatoonMember holds one platoon mate's totals across the session
// battles that mate played with the author. Its outcome counters repeat the
// author's, since platoon mates share the team result. Survived counts the
// battles the mate finished alive and SurvivalKnown the battles whose ending
// the recording exposed, so a mate's survival rate can read "—" instead of
// guessing when the recordings stop short of the battle end.
type SessionPlatoonMember struct {
	AccountID                             uint32
	Nickname                              string
	Battles, Wins, Losses, Draws, Unknown int
	Survived, SurvivalKnown               int
	Damage, Assisted, Blocked, Destroyed  uint64
}

type SessionOutcome string

const (
	SessionVictory SessionOutcome = "victory"
	SessionDefeat  SessionOutcome = "defeat"
	SessionDraw    SessionOutcome = "draw"
	SessionUnknown SessionOutcome = "unknown"
)

// SessionBattleResult is one completed battle, relative to the replay author.
type SessionBattleResult struct {
	ArenaID                      string
	StartedAt                    time.Time
	Outcome                      SessionOutcome
	Survived                     bool
	Damage                       uint32
	Assisted, Blocked, Destroyed uint32
	Excluded                     bool
	// Platoon holds the author's platoon mates of this battle and their
	// results. A battle without a known platoon leaves it empty.
	Platoon []SessionPlatoonResult
}

// SessionPlatoonResult is one platoon mate's result in one battle. Survived
// stays nil when the recording does not expose the mate's final hitpoints;
// only the author's are part of the battle results.
type SessionPlatoonResult struct {
	AccountID                            uint32
	Nickname                             string
	Damage, Assisted, Blocked, Destroyed uint32
	Survived                             *bool
}

// Key identifies the author's battle independently of the replay filename.
func (b SessionBattleResult) Key(accountID uint32) string {
	arena := b.ArenaID
	if arena == "" {
		arena = fmt.Sprint(b.StartedAt.Unix())
	}
	return fmt.Sprintf("%d/%s", accountID, arena)
}

// WithExcludedBattles returns a detached session. Excluded battles stay in
// history, but do not contribute to any total or average denominator.
func (d DailySession) WithExcludedBattles(excluded map[string]bool) DailySession {
	d.Players = append([]SessionStats(nil), d.Players...)
	for i := range d.Players {
		p := &d.Players[i]
		p.History = append([]SessionBattleResult(nil), p.History...)
		for j := range p.History {
			p.History[j].Excluded = excluded[p.History[j].Key(p.AccountID)]
		}
		p.recalculate()
	}
	return d
}

func (p *SessionStats) recalculate() {
	p.Battles, p.Wins, p.Losses, p.Draws, p.Unknown, p.Survived = 0, 0, 0, 0, 0, 0
	p.Damage, p.Assisted, p.Blocked, p.Destroyed = 0, 0, 0, 0
	p.Platoon = nil
	mates := map[uint32]int{}
	for _, battle := range p.History {
		if battle.Excluded {
			continue
		}
		p.Battles++
		if battle.Survived {
			p.Survived++
		}
		switch battle.Outcome {
		case SessionVictory:
			p.Wins++
		case SessionDefeat:
			p.Losses++
		case SessionDraw:
			p.Draws++
		default:
			p.Unknown++
		}
		p.Damage += uint64(battle.Damage)
		p.Assisted += uint64(battle.Assisted)
		p.Blocked += uint64(battle.Blocked)
		p.Destroyed += uint64(battle.Destroyed)
		for _, mate := range battle.Platoon {
			index, ok := mates[mate.AccountID]
			if !ok {
				index = len(p.Platoon)
				mates[mate.AccountID] = index
				p.Platoon = append(p.Platoon, SessionPlatoonMember{AccountID: mate.AccountID, Nickname: mate.Nickname})
			}
			entry := &p.Platoon[index]
			if entry.Nickname == "" {
				entry.Nickname = mate.Nickname
			}
			entry.Battles++
			switch battle.Outcome {
			case SessionVictory:
				entry.Wins++
			case SessionDefeat:
				entry.Losses++
			case SessionDraw:
				entry.Draws++
			default:
				entry.Unknown++
			}
			entry.Damage += uint64(mate.Damage)
			entry.Assisted += uint64(mate.Assisted)
			entry.Blocked += uint64(mate.Blocked)
			entry.Destroyed += uint64(mate.Destroyed)
			if mate.Survived != nil {
				entry.SurvivalKnown++
				if *mate.Survived {
					entry.Survived++
				}
			}
		}
	}
}

// DailySession uses battle timestamps in the supplied time's location, rather
// than file modification times. Invalid or incomplete archives are skipped.
type DailySession struct {
	Date    string
	Players []SessionStats
	Skipped int
}

type sessionBattle struct {
	time     time.Time
	arena    string
	nickname string
	results  *BattleResults
	// survived maps an account to whether it finished the battle alive. Nil
	// means the recording did not expose the endings of the other players.
	survived map[uint32]bool
}
type cachedSessionBattle struct {
	size     int64
	modified time.Time
	battle   sessionBattle
}

// SessionReader caches successfully decoded archives. Use from one worker only.
// Failed reads are retried, since the game may still be writing an archive.
type SessionReader struct {
	cache map[string]cachedSessionBattle
}

func (s *SessionReader) Read(ctx context.Context, directory string, now time.Time) (DailySession, error) {
	day := DailySession{Date: now.Format("2006-01-02")}
	entries, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		s.cache = nil
		return day, nil
	}
	if err != nil {
		return day, err
	}
	next := make(map[string]cachedSessionBattle)
	var battles []sessionBattle
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return day, err
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".tbreplay") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		info, err := entry.Info()
		if err != nil {
			day.Skipped++
			continue
		}
		cached, ok := s.cache[path]
		if !ok || cached.size != info.Size() || !cached.modified.Equal(info.ModTime()) {
			analysis, err := AnalyzeReplay(path, Options{})
			if err != nil || analysis.results == nil || analysis.results.Author.AccountID == 0 || analysis.results.TimestampSecs <= 0 {
				day.Skipped++
				continue
			}
			results := analysis.results
			battle := sessionBattle{time: time.Unix(results.TimestampSecs, 0), results: results}
			if meta, ok := analysis.Meta.(jsonx.Object); ok {
				if id, ok := meta["arenaUniqueId"]; ok {
					battle.arena = fmt.Sprint(id)
				}
			}
			for _, player := range results.Players {
				if player.AccountID == results.Author.AccountID {
					battle.nickname = player.Info.Nickname
					break
				}
			}
			// Only a battle with platoon mates needs the extra pass over the
			// packet stream, and only it pays for one.
			if len(sessionPlatoon(results)) > 0 {
				battle.survived = sessionSurvival(path, results)
			}
			cached = cachedSessionBattle{size: info.Size(), modified: info.ModTime(), battle: battle}
		}
		next[path] = cached
		battles = append(battles, cached.battle)
	}
	s.cache = next
	day.Players = summarizeSession(battles, now)
	return day, nil
}

func summarizeSession(battles []sessionBattle, now time.Time) []SessionStats {
	players := map[uint32]*SessionStats{}
	seen := map[string]bool{}
	for _, battle := range battles {
		if battle.time.In(now.Location()).Format("2006-01-02") != now.Format("2006-01-02") {
			continue
		}
		r := battle.results
		arena := battle.arena
		if arena == "" {
			arena = fmt.Sprint(r.TimestampSecs)
		}
		key := fmt.Sprintf("%d/%s", r.Author.AccountID, arena)
		if seen[key] {
			continue
		}
		seen[key] = true
		p := players[r.Author.AccountID]
		if p == nil {
			p = &SessionStats{AccountID: r.Author.AccountID, Nickname: battle.nickname}
			players[p.AccountID] = p
		}
		outcome := SessionUnknown
		switch {
		case r.WinnerTeamNumber == nil:
		case *r.WinnerTeamNumber == r.Author.TeamNumber:
			outcome = SessionVictory
		case *r.WinnerTeamNumber == 0:
			outcome = SessionDraw
		default:
			outcome = SessionDefeat
		}
		entry := SessionBattleResult{ArenaID: arena, StartedAt: battle.time.In(now.Location()), Outcome: outcome, Survived: r.Author.HitpointsLeft > 0, Damage: r.Author.DamageDealt}
		for _, row := range r.PlayerResults {
			if row.Info.AccountID != p.AccountID {
				continue
			}
			entry.Assisted, entry.Blocked, entry.Destroyed = row.Info.DamageAssisted1, row.Info.DamageBlocked, row.Info.NEnemiesDestroyed
			break
		}
		entry.Platoon = sessionPlatoon(r)
		for i := range entry.Platoon {
			if alive, ok := battle.survived[entry.Platoon[i].AccountID]; ok {
				value := alive
				entry.Platoon[i].Survived = &value
			}
		}
		p.History = append(p.History, entry)
	}
	out := make([]SessionStats, 0, len(players))
	for _, p := range players {
		p.recalculate()
		sort.Slice(p.History, func(i, j int) bool {
			if p.History[i].StartedAt.Equal(p.History[j].StartedAt) {
				return p.History[i].ArenaID > p.History[j].ArenaID
			}
			return p.History[i].StartedAt.After(p.History[j].StartedAt)
		})
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AccountID < out[j].AccountID })
	return out
}

// sessionPlatoon collects the author's platoon mates of one battle. Platoon
// numbers are local to a team, so only allies sharing the author's positive
// number count; a missing, nil or zero number means no known platoon. Mates
// keep the roster order, and their counters come from the battle results.
func sessionPlatoon(r *BattleResults) []SessionPlatoonResult {
	own := -1
	for i := range r.Players {
		if r.Players[i].AccountID == r.Author.AccountID {
			own = i
			break
		}
	}
	if own < 0 || r.Players[own].Info.PlatoonID == nil || *r.Players[own].Info.PlatoonID == 0 {
		return nil
	}
	results := make(map[uint32]PlayerResultsInfo, len(r.PlayerResults))
	for _, row := range r.PlayerResults {
		results[row.Info.AccountID] = row.Info
	}
	var mates []SessionPlatoonResult
	for i, player := range r.Players {
		info := player.Info
		if i == own || info.Team != r.Players[own].Info.Team || info.PlatoonID == nil || *info.PlatoonID != *r.Players[own].Info.PlatoonID {
			continue
		}
		mate := SessionPlatoonResult{AccountID: player.AccountID, Nickname: info.Nickname}
		if row, ok := results[player.AccountID]; ok {
			mate.Damage, mate.Assisted, mate.Blocked, mate.Destroyed = row.DamageDealt, row.DamageAssisted1, row.DamageBlocked, row.NEnemiesDestroyed
		}
		mates = append(mates, mate)
	}
	return mates
}

// sessionSurvival reports which players of a completed battle finished alive,
// read from the recording's packet stream: battle results carry the final
// hitpoints of the author alone. A recording that stops before the battle ends
// does not reproduce the author's known hitpoints, and then every ending stays
// unknown instead of a guess.
func sessionSurvival(path string, results *BattleResults) map[uint32]bool {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return nil
	}
	data, err := readEntry(archive, path, dataEntry)
	if err != nil {
		return nil
	}
	stream, err := readDataReplay(bytes.NewReader(data), path)
	if err != nil {
		return nil
	}
	state := &livePlayers{}
	if err := state.event(FollowEvent{Event: StreamEvent{Kind: EventHeader, ClientVersion: stream.clientVersion}}); err != nil {
		return nil
	}
	for index, packet := range stream.packets {
		if err := state.event(FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: packet, Seq: index}}); err != nil {
			return nil
		}
	}
	alive := make(map[uint32]bool, len(state.players))
	for accountID, player := range state.players {
		if player.VehicleEntityID == nil {
			continue
		}
		if health, known := state.health.vehicles[*player.VehicleEntityID]; known {
			alive[accountID] = health.hp > 0
		}
	}
	own, ok := alive[results.Author.AccountID]
	if !ok || own != (results.Author.HitpointsLeft > 0) {
		return nil
	}
	return alive
}
