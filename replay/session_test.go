package replay

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSessionReaderFixtureDeduplicatesAndRefreshes(t *testing.T) {
	data, err := os.ReadFile("../testdata/sample.tbreplay")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range []string{"one.tbreplay", "copy.tbreplay"} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	broken := filepath.Join(dir, "incomplete.tbreplay")
	os.WriteFile(broken, []byte("partial"), 0600)
	// Recording directories must never count as completed battles.
	os.Mkdir(filepath.Join(dir, "recording_test.tbreplay"), 0700)
	now := time.Unix(1790855739, 0).In(time.FixedZone("local", 3*3600))
	reader := &SessionReader{}
	day, err := reader.Read(context.Background(), dir, now)
	if err != nil || day.Skipped != 1 || len(day.Players) != 1 {
		t.Fatalf("%+v %v", day, err)
	}
	p := day.Players[0]
	if p.Battles != 1 || p.AccountID != 106461188 || p.Nickname != "iCooLER" || p.Damage == 0 {
		t.Fatalf("%+v", p)
	}
	if len(p.History) != 1 || uint64(p.History[0].Damage) != p.Damage || p.History[0].Outcome != SessionVictory {
		t.Fatalf("fixture history: %+v", p.History)
	}
	// Corrupt files are retried and a repaired duplicate is deduplicated, too.
	os.WriteFile(broken, data, 0600)
	day, err = reader.Read(context.Background(), dir, now)
	if err != nil || day.Skipped != 0 || day.Players[0].Battles != 1 {
		t.Fatalf("%+v %v", day, err)
	}
	day, err = reader.Read(context.Background(), dir, now.AddDate(0, 0, 1))
	if err != nil || len(day.Players) != 0 {
		t.Fatalf("midnight: %+v %v", day, err)
	}
	for _, name := range []string{"one.tbreplay", "copy.tbreplay", "incomplete.tbreplay"} {
		os.Remove(filepath.Join(dir, name))
	}
	day, err = reader.Read(context.Background(), dir, now)
	if err != nil || len(day.Players) != 0 || len(reader.cache) != 0 {
		t.Fatalf("removed: %+v %v", day, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	os.WriteFile(broken, data, 0600)
	if _, err := reader.Read(ctx, dir, now); err != context.Canceled {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestSummarizeSessionLocalDayAccountsAndOutcomes(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.FixedZone("local", 3*3600))
	var battles []sessionBattle
	for i, winner := range []*int32{sessionInt32(1), sessionInt32(2), sessionInt32(0), nil} {
		r := &BattleResults{TimestampSecs: now.Unix() + int64(i), Author: Author{AccountID: 42, TeamNumber: 1, DamageDealt: 100, HitpointsLeft: []int32{100, 0, -2, 250}[i]}, WinnerTeamNumber: winner,
			PlayerResults: []PlayerResults{{Info: PlayerResultsInfo{AccountID: 99, DamageAssisted1: 999}}, {Info: PlayerResultsInfo{AccountID: 42, DamageAssisted1: 20, DamageBlocked: 30, NEnemiesDestroyed: 2}}}}
		battles = append(battles, sessionBattle{time: now, arena: string(rune('a' + i)), results: r})
	}
	// 21:30 UTC on the preceding UTC date belongs to today's local session.
	localStart := time.Date(2026, 10, 23, 21, 30, 0, 0, time.UTC)
	battles = append(battles, sessionBattle{time: localStart, arena: "a", results: &BattleResults{Author: Author{AccountID: 7}}})
	battles = append(battles, sessionBattle{time: localStart.Add(-time.Hour), arena: "old", results: &BattleResults{Author: Author{AccountID: 42}}})
	players := summarizeSession(battles, now)
	if len(players) != 2 || players[0].AccountID != 7 || players[0].Battles != 1 {
		t.Fatalf("%+v", players)
	}
	p := players[1]
	if p.Battles != 4 || p.Wins != 1 || p.Losses != 1 || p.Draws != 1 || p.Unknown != 1 || p.Survived != 2 || p.Damage != 400 || p.Assisted != 80 || p.Blocked != 120 || p.Destroyed != 8 {
		t.Fatalf("%+v", p)
	}
	if len(p.History) != 4 {
		t.Fatalf("history: %+v", p.History)
	}
	for i, outcome := range []SessionOutcome{SessionUnknown, SessionDraw, SessionDefeat, SessionVictory} {
		if p.History[i].Outcome != outcome || p.History[i].Damage != 100 {
			t.Fatalf("history[%d]: %+v", i, p.History[i])
		}
	}
	for i, survived := range []bool{true, false, false, true} {
		if p.History[i].Survived != survived {
			t.Fatalf("history[%d] survival: %+v", i, p.History[i])
		}
	}
}
func sessionInt32(v int32) *int32 { return &v }

func TestSessionExclusionsRecalculateAndPreserveHistory(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.Local)
	day := DailySession{Date: "2026-10-24", Players: []SessionStats{
		{AccountID: 42, History: []SessionBattleResult{
			{ArenaID: "win", StartedAt: now, Outcome: SessionVictory, Survived: true, Damage: 1000, Assisted: 100, Blocked: 200, Destroyed: 1},
			{ArenaID: "loss", StartedAt: now.Add(time.Minute), Outcome: SessionDefeat, Survived: true, Damage: 3000, Assisted: 500, Blocked: 800, Destroyed: 3},
			{ArenaID: "draw", StartedAt: now.Add(2 * time.Minute), Outcome: SessionDraw, Damage: 2000},
			{ArenaID: "unknown", StartedAt: now.Add(3 * time.Minute), Outcome: SessionUnknown, Survived: true, Damage: 500},
		}},
		{AccountID: 99, History: []SessionBattleResult{{ArenaID: "loss", Outcome: SessionVictory, Damage: 600}}},
	}}
	day = day.WithExcludedBattles(nil)
	excluded := map[string]bool{day.Players[0].History[1].Key(42): true}
	filtered := day.WithExcludedBattles(excluded)
	p := filtered.Players[0]
	if p.Battles != 3 || p.Wins != 1 || p.Losses != 0 || p.Draws != 1 || p.Unknown != 1 || p.Survived != 2 || p.Damage != 3500 || p.Assisted != 100 || p.Blocked != 200 || p.Destroyed != 1 {
		t.Fatalf("excluded totals: %+v", p)
	}
	if len(p.History) != 4 || !p.History[1].Excluded || day.Players[0].History[1].Excluded || day.Players[0].Battles != 4 {
		t.Fatal("history was removed or source snapshot mutated")
	}
	if filtered.Players[1].Battles != 1 || filtered.Players[1].Damage != 600 {
		t.Fatal("another author's battle was excluded")
	}
	for _, battle := range day.Players[0].History {
		excluded[battle.Key(42)] = true
	}
	all := day.WithExcludedBattles(excluded).Players[0]
	if all.Battles != 0 || all.Wins != 0 || all.Losses != 0 || all.Draws != 0 || all.Unknown != 0 || all.Survived != 0 || all.Damage != 0 || all.Assisted != 0 || all.Blocked != 0 || all.Destroyed != 0 || len(all.History) != 4 {
		t.Fatalf("all excluded: %+v", all)
	}
	restored := filtered.WithExcludedBattles(nil)
	if !reflect.DeepEqual(restored, day) {
		t.Fatalf("restored stats differ: %+v", restored)
	}
}

func TestSessionPlatoonMatesAggregateAcrossBattles(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.Local)
	platoon := uint32(5)
	author := func(platoonID *uint32, team int32) ResultPlayer {
		return ResultPlayer{AccountID: 42, Info: ResultPlayerInfo{Team: team, PlatoonID: platoonID}}
	}
	mate := func(accountID uint32, platoonID *uint32, team int32, damage uint32) ResultPlayer {
		return ResultPlayer{AccountID: accountID, Info: ResultPlayerInfo{Nickname: fmt.Sprintf("mate-%d", accountID), Team: team, PlatoonID: platoonID}}
	}
	row := func(accountID, damage, assisted, blocked, destroyed uint32) PlayerResults {
		return PlayerResults{Info: PlayerResultsInfo{AccountID: accountID, DamageDealt: damage, DamageAssisted1: assisted, DamageBlocked: blocked, NEnemiesDestroyed: destroyed}}
	}
	win, lose, zero := int32(1), int32(2), uint32(0)
	battles := []sessionBattle{
		{time: now, arena: "win", results: &BattleResults{TimestampSecs: now.Unix(), Author: Author{AccountID: 42, TeamNumber: 1, DamageDealt: 100}, WinnerTeamNumber: &win,
			Players: []ResultPlayer{
				author(&platoon, 1),
				mate(7, &platoon, 1, 700),
				// Zero numbers are not a platoon, and an enemy sharing the
				// author's number is a different platoon.
				mate(8, &zero, 1, 800),
				mate(9, &platoon, 2, 900),
			},
			PlayerResults: []PlayerResults{row(7, 700, 70, 21, 2)}}},
		{time: now.Add(time.Minute), arena: "loss", results: &BattleResults{TimestampSecs: now.Unix() + 60, Author: Author{AccountID: 42, TeamNumber: 1, DamageDealt: 50}, WinnerTeamNumber: &lose,
			Players: []ResultPlayer{
				author(&platoon, 1),
				mate(7, &platoon, 1, 300),
			},
			PlayerResults: []PlayerResults{row(7, 300, 30, 9, 1)}}},
		{time: now.Add(2 * time.Minute), arena: "solo", results: &BattleResults{TimestampSecs: now.Unix() + 120, Author: Author{AccountID: 42, TeamNumber: 2}}},
	}
	players := summarizeSession(battles, now)
	if len(players) != 1 {
		t.Fatalf("%+v", players)
	}
	p := players[0]
	if len(p.Platoon) != 1 {
		t.Fatalf("platoon: %+v", p.Platoon)
	}
	m := p.Platoon[0]
	if m.AccountID != 7 || m.Nickname != "mate-7" || m.Battles != 2 || m.Wins != 1 || m.Losses != 1 || m.Draws != 0 || m.Unknown != 0 ||
		m.Damage != 1000 || m.Assisted != 100 || m.Blocked != 30 || m.Destroyed != 3 {
		t.Fatalf("mate totals: %+v", m)
	}
	// History is newest first: the solo battle has no platoon, the others
	// carry the mate.
	if len(p.History) != 3 || len(p.History[0].Platoon) != 0 {
		t.Fatalf("solo battle platoon: %+v", p.History)
	}
	for _, battle := range p.History[1:] {
		if len(battle.Platoon) != 1 || battle.Platoon[0].AccountID != 7 {
			t.Fatalf("battle platoon: %+v", battle)
		}
	}
	// Excluded battles leave the platoon totals, but stay in history.
	excluded := map[string]bool{p.History[2].Key(42): true}
	filtered := DailySession{Date: "2026-10-24", Players: players}.WithExcludedBattles(excluded).Players[0]
	if len(filtered.Platoon) != 1 || filtered.Platoon[0].Battles != 1 || filtered.Platoon[0].Damage != 300 || filtered.Platoon[0].Wins != 0 || filtered.Platoon[0].Losses != 1 {
		t.Fatalf("excluded platoon: %+v", filtered.Platoon)
	}
	if len(filtered.History) != 3 || !filtered.History[2].Excluded {
		t.Fatalf("excluded history: %+v", filtered.History)
	}
	if len(p.Platoon) != 1 || p.Platoon[0].Battles != 2 {
		t.Fatalf("source mutated: %+v", p.Platoon)
	}
}

func TestSessionPlatoonSurvivalCountsKnownBattles(t *testing.T) {
	alive, dead := true, false
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.Local)
	day := DailySession{Date: "2026-10-24", Players: []SessionStats{
		{AccountID: 42, History: []SessionBattleResult{
			{ArenaID: "one", StartedAt: now, Outcome: SessionVictory, Damage: 1000, Platoon: []SessionPlatoonResult{
				{AccountID: 7, Nickname: "mate", Damage: 700, Survived: &alive},
			}},
			{ArenaID: "two", StartedAt: now.Add(time.Minute), Outcome: SessionDefeat, Damage: 500, Platoon: []SessionPlatoonResult{
				{AccountID: 7, Nickname: "mate", Damage: 300, Survived: &dead},
			}},
			// A recording that stops before the battle end leaves the ending
			// unknown, and unknown must not count as a death.
			{ArenaID: "three", StartedAt: now.Add(2 * time.Minute), Outcome: SessionVictory, Damage: 200, Platoon: []SessionPlatoonResult{
				{AccountID: 7, Nickname: "mate", Damage: 100},
			}},
		}},
	}}
	day = day.WithExcludedBattles(nil)
	mate := day.Players[0].Platoon[0]
	if mate.Battles != 3 || mate.Wins != 2 || mate.Losses != 1 || mate.Survived != 1 || mate.SurvivalKnown != 2 || mate.Damage != 1100 {
		t.Fatalf("mate: %+v", mate)
	}
	excluded := map[string]bool{day.Players[0].History[0].Key(42): true}
	filtered := day.WithExcludedBattles(excluded).Players[0].Platoon[0]
	if filtered.Battles != 2 || filtered.Survived != 0 || filtered.SurvivalKnown != 1 {
		t.Fatalf("excluded mate: %+v", filtered)
	}
}

func TestSessionSurvivalMatchesFixtureAuthor(t *testing.T) {
	path := "../testdata/sample.tbreplay"
	analysis, err := AnalyzeReplay(path, Options{})
	if err != nil || analysis.results == nil {
		t.Fatalf("fixture: %v", err)
	}
	alive := sessionSurvival(path, analysis.results)
	if alive == nil {
		t.Fatal("fixture stream did not expose the battle ending")
	}
	own, ok := alive[analysis.results.Author.AccountID]
	if !ok || own != (analysis.results.Author.HitpointsLeft > 0) {
		t.Fatalf("author survival %v (known %v), hitpoints %d", own, ok, analysis.results.Author.HitpointsLeft)
	}
}
