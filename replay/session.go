package replay

import (
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
// counted separately and excluded from the win-rate denominator.
type SessionStats struct {
	AccountID                             uint32
	Nickname                              string
	Battles, Wins, Losses, Draws, Unknown int
	Damage, Assisted, Blocked, Destroyed  uint64
	History                               []SessionBattleResult
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
	Damage                       uint32
	Assisted, Blocked, Destroyed uint32
	Excluded                     bool
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
	p.Battles, p.Wins, p.Losses, p.Draws, p.Unknown = 0, 0, 0, 0, 0
	p.Damage, p.Assisted, p.Blocked, p.Destroyed = 0, 0, 0, 0
	for _, battle := range p.History {
		if battle.Excluded {
			continue
		}
		p.Battles++
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
		entry := SessionBattleResult{ArenaID: arena, StartedAt: battle.time.In(now.Location()), Outcome: outcome, Damage: r.Author.DamageDealt}
		for _, row := range r.PlayerResults {
			if row.Info.AccountID != p.AccountID {
				continue
			}
			entry.Assisted, entry.Blocked, entry.Destroyed = row.Info.DamageAssisted1, row.Info.DamageBlocked, row.Info.NEnemiesDestroyed
			break
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
