package replay

import (
	"context"
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
		r := &BattleResults{TimestampSecs: now.Unix() + int64(i), Author: Author{AccountID: 42, TeamNumber: 1, DamageDealt: 100}, WinnerTeamNumber: winner,
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
	if p.Battles != 4 || p.Wins != 1 || p.Losses != 1 || p.Draws != 1 || p.Unknown != 1 || p.Damage != 400 || p.Assisted != 80 || p.Blocked != 120 || p.Destroyed != 8 {
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
}
func sessionInt32(v int32) *int32 { return &v }

func TestSessionExclusionsRecalculateAndPreserveHistory(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.Local)
	day := DailySession{Date: "2026-10-24", Players: []SessionStats{
		{AccountID: 42, History: []SessionBattleResult{
			{ArenaID: "win", StartedAt: now, Outcome: SessionVictory, Damage: 1000, Assisted: 100, Blocked: 200, Destroyed: 1},
			{ArenaID: "loss", StartedAt: now.Add(time.Minute), Outcome: SessionDefeat, Damage: 3000, Assisted: 500, Blocked: 800, Destroyed: 3},
			{ArenaID: "draw", StartedAt: now.Add(2 * time.Minute), Outcome: SessionDraw, Damage: 2000},
			{ArenaID: "unknown", StartedAt: now.Add(3 * time.Minute), Outcome: SessionUnknown, Damage: 500},
		}},
		{AccountID: 99, History: []SessionBattleResult{{ArenaID: "loss", Outcome: SessionVictory, Damage: 600}}},
	}}
	day = day.WithExcludedBattles(nil)
	excluded := map[string]bool{day.Players[0].History[1].Key(42): true}
	filtered := day.WithExcludedBattles(excluded)
	p := filtered.Players[0]
	if p.Battles != 3 || p.Wins != 1 || p.Losses != 0 || p.Draws != 1 || p.Unknown != 1 || p.Damage != 3500 || p.Assisted != 100 || p.Blocked != 200 || p.Destroyed != 1 {
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
	if all.Battles != 0 || all.Wins != 0 || all.Losses != 0 || all.Draws != 0 || all.Unknown != 0 || all.Damage != 0 || all.Assisted != 0 || all.Blocked != 0 || all.Destroyed != 0 || len(all.History) != 4 {
		t.Fatalf("all excluded: %+v", all)
	}
	restored := filtered.WithExcludedBattles(nil)
	if !reflect.DeepEqual(restored, day) {
		t.Fatalf("restored stats differ: %+v", restored)
	}
}
