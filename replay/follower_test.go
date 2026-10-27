package replay

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/thecoooler/blitz-client/internal/jsonx"
)

func TestStreamFollowerAndReportingFollowerIncrementalParity(t *testing.T) {
	data := sampleReplay()
	rawDir := t.TempDir()
	reportingDir := t.TempDir()
	rawPath := filepath.Join(rawDir, "data.replay")
	reportingPath := filepath.Join(reportingDir, "data.replay")

	rawFollower := newStreamFollower(rawPath)
	defer rawFollower.Close()
	reportingFollower := NewFollower(reportingPath)
	defer reportingFollower.Close()

	// Initial poll on missing file returns 0, nil without advancing session.
	rawCount, err := rawFollower.Poll(func(FollowEvent) error {
		t.Fatal("unexpected event from missing file")
		return nil
	})
	if err != nil || rawCount != 0 {
		t.Fatalf("raw initial poll: %d %v", rawCount, err)
	}
	repCount, err := reportingFollower.Poll(func(FollowEvent) error {
		t.Fatal("unexpected event from missing file")
		return nil
	})
	if err != nil || repCount != 0 {
		t.Fatalf("reporting initial poll: %d %v", repCount, err)
	}

	rawFile, err := os.Create(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	defer rawFile.Close()

	repFile, err := os.Create(reportingPath)
	if err != nil {
		t.Fatal(err)
	}
	defer repFile.Close()

	var rawEvents []FollowEvent
	var repEvents []FollowEvent

	// Feed incrementally in small unaligned chunks (7 bytes) across partial headers and frames.
	chunkSize := 7
	for offset := 0; offset < len(data); offset += chunkSize {
		end := min(offset+chunkSize, len(data))
		chunk := data[offset:end]

		if _, err := rawFile.Write(chunk); err != nil {
			t.Fatal(err)
		}
		if _, err := repFile.Write(chunk); err != nil {
			t.Fatal(err)
		}

		if _, err := rawFollower.Poll(func(e FollowEvent) error {
			rawEvents = append(rawEvents, e)
			return nil
		}); err != nil {
			t.Fatalf("raw poll at %d: %v", offset, err)
		}

		if _, err := reportingFollower.Poll(func(e FollowEvent) error {
			repEvents = append(repEvents, e)
			return nil
		}); err != nil {
			t.Fatalf("reporting poll at %d: %v", offset, err)
		}
	}

	// Idle EOF poll must return 0 bytes and no duplicate events on both.
	idleRaw, err := rawFollower.Poll(func(FollowEvent) error {
		t.Fatal("raw duplicate event on idle EOF")
		return nil
	})
	if err != nil || idleRaw != 0 {
		t.Fatalf("idle raw poll: %d %v", idleRaw, err)
	}
	idleRep, err := reportingFollower.Poll(func(FollowEvent) error {
		t.Fatal("reporting duplicate event on idle EOF")
		return nil
	})
	if err != nil || idleRep != 0 {
		t.Fatalf("idle rep poll: %d %v", idleRep, err)
	}

	if len(rawEvents) == 0 || len(rawEvents) != len(repEvents) {
		t.Fatalf("event count mismatch: raw=%d reporting=%d", len(rawEvents), len(repEvents))
	}

	for i := range rawEvents {
		raw := rawEvents[i]
		rep := repEvents[i]

		// Session IDs and full StreamEvent payloads must match exactly.
		if raw.SessionID != rep.SessionID {
			t.Fatalf("event %d session mismatch: raw=%d rep=%d", i, raw.SessionID, rep.SessionID)
		}
		if !reflect.DeepEqual(raw.Event, rep.Event) {
			t.Fatalf("event %d StreamEvent payload mismatch:\nraw: %+v\nrep: %+v", i, raw.Event, rep.Event)
		}

		// Raw stream route must have NO reporting telemetry.
		if raw.HasTelemetry || raw.Telemetry != nil {
			t.Fatalf("event %d raw route unexpectedly had telemetry: has=%v val=%v", i, raw.HasTelemetry, raw.Telemetry)
		}
		rawJSON, err := jsonx.Marshal(&raw)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(rawJSON), `"telemetry"`) {
			t.Fatalf("event %d raw JSON contains telemetry: %s", i, string(rawJSON))
		}

		// Public reporting path must have reporting telemetry on packets.
		if rep.Event.Kind == EventPacket {
			if !rep.HasTelemetry {
				t.Fatalf("event %d reporting route missing HasTelemetry", i)
			}
			repJSON, err := jsonx.Marshal(&rep)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(repJSON), `"telemetry"`) {
				t.Fatalf("event %d reporting JSON missing telemetry key: %s", i, string(repJSON))
			}
		}
	}
}

func TestStreamFollowerAndReportingFollowerResets(t *testing.T) {
	sample := sampleReplay()
	rawDir := t.TempDir()
	repDir := t.TempDir()
	rawPath := filepath.Join(rawDir, "data.replay")
	repPath := filepath.Join(repDir, "data.replay")

	if err := os.WriteFile(rawPath, sample, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repPath, sample, 0600); err != nil {
		t.Fatal(err)
	}

	rawFollower := newStreamFollower(rawPath)
	defer rawFollower.Close()
	repFollower := NewFollower(repPath)
	defer repFollower.Close()

	// Initial poll reads sample frames (session 1).
	var rawInitial, repInitial []FollowEvent
	if _, err := rawFollower.Poll(func(e FollowEvent) error { rawInitial = append(rawInitial, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := repFollower.Poll(func(e FollowEvent) error { repInitial = append(repInitial, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(rawInitial) != len(repInitial) || len(rawInitial) < 2 {
		t.Fatalf("initial frames mismatch: %d vs %d", len(rawInitial), len(repInitial))
	}

	// 1. Truncation reset: truncate to 0 while readOffset > 0.
	if err := os.WriteFile(rawPath, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repPath, nil, 0600); err != nil {
		t.Fatal(err)
	}

	var rawTrunc, repTrunc []FollowEvent
	if _, err := rawFollower.Poll(func(e FollowEvent) error { rawTrunc = append(rawTrunc, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := repFollower.Poll(func(e FollowEvent) error { repTrunc = append(repTrunc, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(rawTrunc) != 1 || rawTrunc[0].Event.Kind != EventReset || rawTrunc[0].Event.Reason != "file_truncated" || rawTrunc[0].SessionID != 2 {
		t.Fatalf("raw truncation: %+v", rawTrunc)
	}
	if len(repTrunc) != 1 || repTrunc[0].Event.Kind != EventReset || repTrunc[0].Event.Reason != "file_truncated" || repTrunc[0].SessionID != 2 {
		t.Fatalf("rep truncation: %+v", repTrunc)
	}

	// Re-write sample after truncation.
	if err := os.WriteFile(rawPath, sample, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repPath, sample, 0600); err != nil {
		t.Fatal(err)
	}
	var rawPostTrunc, repPostTrunc []FollowEvent
	if _, err := rawFollower.Poll(func(e FollowEvent) error { rawPostTrunc = append(rawPostTrunc, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := repFollower.Poll(func(e FollowEvent) error { repPostTrunc = append(repPostTrunc, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(rawPostTrunc) != len(repPostTrunc) || rawPostTrunc[0].SessionID != 2 {
		t.Fatalf("post truncation mismatch: raw=%+v rep=%+v", rawPostTrunc, repPostTrunc)
	}
	for _, e := range rawPostTrunc {
		if e.HasTelemetry || e.Telemetry != nil {
			t.Fatalf("raw post-truncation has telemetry: %+v", e)
		}
	}
	for _, e := range repPostTrunc {
		if e.Event.Kind == EventPacket && !e.HasTelemetry {
			t.Fatalf("rep post-truncation missing telemetry: %+v", e)
		}
	}

	// 2. Rewrite reset: overwrite with identical size but mutated anchor bytes.
	mutated := make([]byte, len(sample))
	copy(mutated, sample)
	mutated[len(mutated)-1] ^= 0xff
	if err := os.WriteFile(rawPath, mutated, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repPath, mutated, 0600); err != nil {
		t.Fatal(err)
	}
	var rawRewritten, repRewritten []FollowEvent
	if _, err := rawFollower.Poll(func(e FollowEvent) error { rawRewritten = append(rawRewritten, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := repFollower.Poll(func(e FollowEvent) error { repRewritten = append(repRewritten, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(rawRewritten) != 1 || rawRewritten[0].Event.Kind != EventReset || rawRewritten[0].Event.Reason != "file_rewritten" || rawRewritten[0].SessionID != 3 {
		t.Fatalf("raw rewrite: %+v", rawRewritten)
	}
	if len(repRewritten) != 1 || repRewritten[0].Event.Kind != EventReset || repRewritten[0].Event.Reason != "file_rewritten" || repRewritten[0].SessionID != 3 {
		t.Fatalf("rep rewrite: %+v", repRewritten)
	}

	// Re-write sample after rewrite.
	if err := os.WriteFile(rawPath, sample, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repPath, sample, 0600); err != nil {
		t.Fatal(err)
	}
	var rawPostRewrite, repPostRewrite []FollowEvent
	if _, err := rawFollower.Poll(func(e FollowEvent) error { rawPostRewrite = append(rawPostRewrite, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := repFollower.Poll(func(e FollowEvent) error { repPostRewrite = append(repPostRewrite, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(rawPostRewrite) != len(repPostRewrite) || rawPostRewrite[0].SessionID != 3 {
		t.Fatalf("post rewrite mismatch: raw=%+v rep=%+v", rawPostRewrite, repPostRewrite)
	}

	// 3. Replacement reset: atomic rename.
	rawRepl := filepath.Join(rawDir, "replacement")
	repRepl := filepath.Join(repDir, "replacement")
	if err := os.WriteFile(rawRepl, sample, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repRepl, sample, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(rawRepl, rawPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(repRepl, repPath); err != nil {
		t.Fatal(err)
	}
	var rawReplEvents, repReplEvents []FollowEvent
	if _, err := rawFollower.Poll(func(e FollowEvent) error { rawReplEvents = append(rawReplEvents, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := repFollower.Poll(func(e FollowEvent) error { repReplEvents = append(repReplEvents, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if rawReplEvents[0].Event.Kind != EventReset || rawReplEvents[0].Event.Reason != "file_replaced" || rawReplEvents[0].SessionID != 4 {
		t.Fatalf("raw replacement: %+v", rawReplEvents)
	}
	if repReplEvents[0].Event.Kind != EventReset || repReplEvents[0].Event.Reason != "file_replaced" || repReplEvents[0].SessionID != 4 {
		t.Fatalf("rep replacement: %+v", repReplEvents)
	}

	// 4. Removal reset: remove file.
	if err := os.Remove(rawPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(repPath); err != nil {
		t.Fatal(err)
	}
	var rawRemEvents, repRemEvents []FollowEvent
	if _, err := rawFollower.Poll(func(e FollowEvent) error { rawRemEvents = append(rawRemEvents, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := repFollower.Poll(func(e FollowEvent) error { repRemEvents = append(repRemEvents, e); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(rawRemEvents) != 1 || rawRemEvents[0].Event.Reason != "file_removed" {
		t.Fatalf("raw removal: %+v", rawRemEvents)
	}
	if len(repRemEvents) != 1 || repRemEvents[0].Event.Reason != "file_removed" {
		t.Fatalf("rep removal: %+v", repRemEvents)
	}

	// Repeated poll while missing returns 0, nil without emitting.
	if count, err := rawFollower.Poll(func(FollowEvent) error { t.Fatal("unexpected raw event"); return nil }); err != nil || count != 0 {
		t.Fatal(count, err)
	}
	if count, err := repFollower.Poll(func(FollowEvent) error { t.Fatal("unexpected rep event"); return nil }); err != nil || count != 0 {
		t.Fatal(count, err)
	}
}

func toUint64(v any) uint64 {
	switch n := v.(type) {
	case uint64:
		return n
	case uint32:
		return uint64(n)
	case int:
		return uint64(n)
	case float64:
		return uint64(n)
	default:
		return 0
	}
}

func TestStreamFollowerFeedsLivePlayersWithoutReportingTelemetry(t *testing.T) {
	data := sampleData(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "data.replay")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	// Raw stream follower feeding livePlayers.
	streamFollower := newStreamFollower(path)
	defer streamFollower.Close()

	var players livePlayers
	var sessionID uint64
	readCount, err := streamFollower.Poll(func(e FollowEvent) error {
		sessionID = e.SessionID
		if e.HasTelemetry || e.Telemetry != nil {
			t.Fatalf("unexpected telemetry in stream follower: %+v", e)
		}
		return players.event(e)
	})
	if err != nil {
		t.Fatal(err)
	}
	if readCount != int64(len(data)) || sessionID != 1 {
		t.Fatalf("read: count=%d session=%d", readCount, sessionID)
	}

	roster, damage := players.snapshot("iCooLER")
	if roster == nil || len(roster.Allies) != 10 || len(roster.Enemies) != 10 {
		t.Fatalf("roster not populated: %+v", roster)
	}
	if damage.Dealt == nil || *damage.Dealt != 2066 {
		t.Fatalf("expected 2066 damage, got: %+v", damage)
	}

	// Compare with reporting follower on the exact same file.
	reportingFollower := NewFollower(path)
	defer reportingFollower.Close()

	var lastCounters map[string]any
	_, err = reportingFollower.Poll(func(e FollowEvent) error {
		if e.Event.Kind == EventPacket && e.HasTelemetry && e.Telemetry != nil {
			m, ok := e.Telemetry.(map[string]any)
			if !ok {
				// Serialized / reconstructed object.
				data, _ := json.Marshal(e.Telemetry)
				json.Unmarshal(data, &m)
			}
			if bc, ok := m["battle_counters"].(map[string]any); ok {
				lastCounters = bc
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if lastCounters == nil {
		t.Fatal("reporting follower produced no battle counters")
	}
	// Reporting counters must agree with live totals.
	if damage.Assisted != nil && lastCounters["damage_assisted"] != nil {
		repAssisted := toUint64(lastCounters["damage_assisted"])
		if *damage.Assisted != repAssisted {
			t.Fatalf("assisted mismatch: live=%d reporting=%d", *damage.Assisted, repAssisted)
		}
	}
	if damage.Blocked != nil && lastCounters["damage_blocked"] != nil {
		repBlocked := toUint64(lastCounters["damage_blocked"])
		if *damage.Blocked != repBlocked {
			t.Fatalf("blocked mismatch: live=%d reporting=%d", *damage.Blocked, repBlocked)
		}
	}
}

// BenchmarkFollowerVsStreamFollower compares raw stream following throughput
// against enriched reporting telemetry decoding for a complete recording.
// It measures follower/telemetry decoding overhead only, not desktop app UI.
func BenchmarkFollowerVsStreamFollower(b *testing.B) {
	archive, err := zip.OpenReader("../testdata/sample.tbreplay")
	if err != nil {
		b.Fatal(err)
	}
	defer archive.Close()
	entry, err := archive.Open("data.replay")
	if err != nil {
		b.Fatal(err)
	}
	data, err := io.ReadAll(entry)
	_ = entry.Close()
	if err != nil {
		b.Fatal(err)
	}

	dir := b.TempDir()
	path := filepath.Join(dir, "data.replay")
	if err := os.WriteFile(path, data, 0600); err != nil {
		b.Fatal(err)
	}

	b.Run("streamFollower_Raw", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			f := newStreamFollower(path)
			var count int
			if _, err := f.Poll(func(e FollowEvent) error {
				count++
				return nil
			}); err != nil {
				b.Fatal(err)
			}
			if err := f.Close(); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Follower_Reporting", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			f := NewFollower(path)
			var count int
			if _, err := f.Poll(func(e FollowEvent) error {
				count++
				return nil
			}); err != nil {
				b.Fatal(err)
			}
			if err := f.Close(); err != nil {
				b.Fatal(err)
			}
		}
	})
}
