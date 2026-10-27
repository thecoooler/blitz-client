package replay

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
)

func valueJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := jsonx.Marshal(jsonx.ToValue(value))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func basePlayerPayload(t *testing.T, extended bool) ([]byte, map[string]any) {
	context := map[string]any{
		"battleLevel": 10, "battleCategoryId": 1, "mouseEnabled": true,
		"mmType": 0, "camouflageSlot": 2, "accountDatabaseIds": []any{42},
		"avgMmr": []any{nil, 3500.5},
	}
	encoded := pickleOf(context)
	out := make([]byte, 10)
	out = append(out, "\x04Test"...)
	out = binary.LittleEndian.AppendUint64(out, 123)
	out = binary.LittleEndian.AppendUint32(out, 7)
	if extended {
		out = append(out, 255)
		out = binary.LittleEndian.AppendUint16(out, uint16(len(encoded)))
		out = append(out, 0)
	} else {
		out = append(out, byte(len(encoded)))
	}
	return append(out, encoded...), context
}

func TestBasePlayerContextDecodesPickleAndPreservesJSONFields(t *testing.T) {
	for _, extended := range []bool{false, true} {
		raw, _ := basePlayerPayload(t, extended)
		payload, err := NewPayload(0, raw)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]map[string]any
		json.Unmarshal([]byte(valueJSON(t, payload)), &decoded)
		player := decoded["BasePlayerCreate"]
		arguments := player["arguments"].(map[string]any)
		if player["author_nickname"] != "Test" || player["arena_unique_id"] != 123.0 || player["arena_type_id"] != 7.0 ||
			arguments["battle_level"] != 10.0 || !reflect.DeepEqual(arguments["average_mm_ratings"], []any{nil, 3500.5}) ||
			!reflect.DeepEqual(arguments["account_database_ids"], []any{42.0}) || arguments["turbo_battles_statistics"] != nil {
			t.Fatalf("%v", decoded)
		}
		// Every incomplete prefix must return an error, including inside pickle.
		for end := range raw {
			if _, err := NewPayload(0, raw[:end]); err == nil {
				t.Fatalf("prefix %d decoded", end)
			}
		}
	}
}

func TestUpdateArenaDecodesPlayersAndSkipsUnknownProtobufFields(t *testing.T) {
	// UpdateArena.players.players: nickname=Test, team=1, account=42; the
	// final field 2 is unmodeled and must be skipped.
	protobuf := []byte("\x0a\x0c\x0a\x0a\x1a\x04Test\x20\x01\x38\x2a\x10\x01")
	for _, extended := range []bool{false, true} {
		raw := make([]byte, 4)
		raw = binary.LittleEndian.AppendUint32(raw, 47)
		raw = binary.LittleEndian.AppendUint32(raw, uint32(len(protobuf)+3))
		raw = append(raw, 0xac, 0x02) // field_number=300 (varint).
		if extended {
			raw = append(raw, 255, byte(len(protobuf)), 0, 0)
		} else {
			raw = append(raw, byte(len(protobuf)))
		}
		raw = append(raw, protobuf...)
		method, err := NewEntityMethod(raw)
		if err != nil {
			t.Fatal(err)
		}
		players := method.UpdateArena.Players.Players
		if method.FieldNumber != 300 || len(players) != 1 || players[0].Nickname != "Test" ||
			players[0].TeamNumber != 1 || players[0].AccountID != 42 {
			t.Fatalf("%+v", method)
		}
		for end := range raw {
			if _, err := NewEntityMethod(raw[:end]); err == nil {
				t.Fatalf("prefix %d decoded", end)
			}
		}
	}
}

func TestWindows26101RosterMethod(t *testing.T) {
	protobuf := []byte("\x0a\x0c\x0a\x0a\x1a\x04Test\x20\x01\x38\x2a")
	raw := make([]byte, 4)
	raw = binary.LittleEndian.AppendUint32(raw, 55)
	raw = binary.LittleEndian.AppendUint32(raw, uint32(len(protobuf)+2))
	raw = append(raw, 1, byte(len(protobuf)))
	raw = append(raw, protobuf...)
	method, err := NewEntityMethodForVersion(raw, "26.10.1_ruby")
	if err != nil || method.UpdateArena == nil || method.UpdateArena.Players == nil {
		t.Fatalf("Windows roster method not decoded: %+v %v", method, err)
	}
	players := method.UpdateArena.Players.Players
	if len(players) != 1 || players[0].Nickname != "Test" || players[0].AccountID != 42 || players[0].TeamNumber != 1 {
		t.Fatalf("Windows roster: %+v", players)
	}
	p := livePlayers{}
	if err := p.event(FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: &Packet{Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: method}}}}); err != nil {
		t.Fatal(err)
	}
	roster, _ := p.snapshot("Test")
	if roster == nil || len(roster.Allies) != 1 {
		t.Fatalf("Windows roster did not start battle: %+v", roster)
	}
	unknown, err := NewEntityMethodForVersion(raw, "26.11.0_ruby")
	if err != nil || unknown.UpdateArena != nil {
		t.Fatalf("unverified version decoded method 55: %+v %v", unknown, err)
	}
}

func TestInvalidLengthsAndVarintsReturnErrorsWithoutPanicking(t *testing.T) {
	prefix := make([]byte, 4)
	prefix = binary.LittleEndian.AppendUint32(prefix, 47)
	prefix = binary.LittleEndian.AppendUint32(prefix, 0)
	for _, suffix := range [][]byte{
		{0, 255, 255, 255, 0},
		{0, 255, 0, 0, 1},
		bytes.Repeat([]byte{128}, 10),
		append(bytes.Repeat([]byte{255}, 9), 2),
		{0, 1, 255},
	} {
		if _, err := NewEntityMethod(append(append([]byte(nil), prefix...), suffix...)); err == nil {
			t.Fatalf("%v decoded", suffix)
		}
	}
	unknown := binary.LittleEndian.AppendUint32(make([]byte, 4), 999)
	if method, err := NewEntityMethod(unknown); err != nil || method.SubType != 999 || method.UpdateArena != nil {
		t.Fatal(method, err)
	}
	if payload, err := NewPayload(999, nil); err != nil || payload.Kind != PayloadUnknown || payload.PacketType != 999 {
		t.Fatal(payload, err)
	}
}

func TestBattleResultsPickleDecodesAndRejectsCorruptProtobuf(t *testing.T) {
	protobuf := []byte("\x08\x07\x10\x7b\x18\x01\x42\x00\x48\x00")
	blob, err := decodeBattleResultsDat(pickleOf([2]any{123, protobuf}))
	if err != nil {
		t.Fatal(err)
	}
	results, err := decodeBattleResults(blob)
	if err != nil || results.ModeMapID != 7 || results.TimestampSecs != 123 || *results.WinnerTeamNumber != 1 {
		t.Fatal(results, err)
	}
	if _, err := decodeBattleResultsDat([]byte("invalid pickle")); err == nil {
		t.Fatal("invalid pickle decoded")
	}
	if _, err := decodeBattleResults([]byte{255}); err == nil {
		t.Fatal("invalid protobuf decoded")
	}
}

func TestArbitraryChunkBoundariesPreserveExpectedPackets(t *testing.T) {
	data := sampleReplay()
	expected := `[{"clock":0.0,"payload":{"Unknown":{"packet_type":99}},"raw_payload":"010203"},` +
		`{"clock":0.20000000298023224,"payload":{"Unknown":{"packet_type":7}},"raw_payload":"000000000000000000000000"},` +
		`{"clock":0.30000001192092896,"payload":{"Unknown":{"packet_type":99}},"raw_payload":""}]`
	for chunkSize := 1; chunkSize <= len(data); chunkSize++ {
		decoder := &StreamDecoder{}
		var packets []*Packet
		headers := 0
		offset := uint64(len(header()))
		for start := 0; start < len(data); start += chunkSize {
			err := decoder.Feed(data[start:min(start+chunkSize, len(data))], func(event StreamEvent) error {
				switch event.Kind {
				case EventHeader:
					if event.ClientVersion != "26.8.0_ruby_apple" || event.HeaderBytes != offset {
						t.Fatalf("%+v", event)
					}
					headers++
				case EventPacket:
					if event.Seq != len(packets) || event.Offset != offset {
						t.Fatalf("%+v", event)
					}
					offset += 12 + uint64(len(event.Packet.RawPayload))
					packets = append(packets, event.Packet)
				default:
					t.Fatal("unexpected reset")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := decoder.Finish(); err != nil || headers != 1 {
			t.Fatal(err, headers)
		}
		if got := valueJSON(t, packets); got != expected {
			t.Fatalf("chunk %d: %s", chunkSize, got)
		}
	}
}

func TestPartialFramesWaitButCompletedInputRejectsTruncation(t *testing.T) {
	data := sampleReplay()
	headerLength := len(header())
	for length := range data {
		decoder := &StreamDecoder{}
		if err := decoder.Feed(data[:length], func(StreamEvent) error { return nil }); err != nil {
			t.Fatal(err)
		}
		boundary := length == headerLength || length == headerLength+15 || length == headerLength+39
		if (decoder.Finish() == nil) != boundary {
			t.Fatalf("prefix %d", length)
		}
	}
}

func TestInvalidHeadersLengthsPayloadsAndClocksAreErrors(t *testing.T) {
	ignore := func(StreamEvent) error { return nil }
	if err := (&StreamDecoder{}).Feed(make([]byte, 4), ignore); err == nil {
		t.Fatal("invalid magic accepted")
	}
	invalidVersion := header()
	invalidVersion[18] = 0xff
	if err := (&StreamDecoder{}).Feed(invalidVersion, ignore); err == nil {
		t.Fatal("invalid version accepted")
	}
	for _, malformed := range [][]byte{
		binary.LittleEndian.AppendUint32(nil, MaxPayloadBytes+1),
		frame(99, float32(math.NaN()), nil),
		frame(8, 0, nil),
	} {
		decoder := &StreamDecoder{}
		decoder.Feed(header(), ignore)
		if err := decoder.Feed(malformed, ignore); err == nil {
			t.Fatalf("%v accepted", malformed)
		}
	}
}

func TestCallbackFailureStopsDelivery(t *testing.T) {
	calls := 0
	err := (&StreamDecoder{}).Feed(sampleReplay(), func(StreamEvent) error {
		calls++
		return blitz.NewError("STOP", "consumer stopped")
	})
	if calls != 1 || blitz.AsError(err).Code != "STOP" {
		t.Fatal(calls, err)
	}
}

type collector struct{ events []FollowEvent }

func (c *collector) emit(event FollowEvent) error {
	c.events = append(c.events, event)
	return nil
}

func poll(t *testing.T, follower *Follower) *collector {
	t.Helper()
	c := &collector{}
	if _, err := follower.Poll(c.emit); err != nil {
		t.Fatal(err)
	}
	return c
}

func eventJSON(t *testing.T, event FollowEvent) map[string]any {
	t.Helper()
	data, err := jsonx.Marshal(&event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	json.Unmarshal(data, &decoded)
	return decoded
}

func TestFollowerWaitsForCreationAndPartialPacketsWithoutDuplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.replay")
	follower := NewFollower(path)
	defer follower.Close()
	c := &collector{}
	if count, err := follower.Poll(c.emit); err != nil || count != 0 {
		t.Fatal(count, err)
	}
	writer, _ := os.Create(path)
	defer writer.Close()
	for _, b := range sampleReplay() {
		writer.Write([]byte{b})
		if count, err := follower.Poll(c.emit); err != nil || count != 1 {
			t.Fatal(count, err)
		}
	}
	if len(c.events) != 4 {
		t.Fatal(len(c.events))
	}
	for _, event := range c.events {
		if event.SessionID != 1 {
			t.Fatal(event.SessionID)
		}
	}
	if count, _ := follower.Poll(func(FollowEvent) error { t.Fatal("duplicate event"); return nil }); count != 0 {
		t.Fatal(count)
	}
	decoded := eventJSON(t, c.events[1])
	if decoded["event"] != "packet" || decoded["seq"] != 0.0 || decoded["clock"] != 0.0 || decoded["packet_type"] != 99.0 {
		t.Fatal(decoded)
	}
}

func TestFollowerResetsOnTruncationReplacementAndRemoval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.replay")
	os.WriteFile(path, sampleReplay(), 0o644)
	follower := NewFollower(path)
	defer follower.Close()
	poll(t, follower)
	os.WriteFile(path, nil, 0o644)
	c := poll(t, follower)
	if c.events[0].Event.Reason != "file_truncated" || c.events[0].SessionID != 2 {
		t.Fatalf("%+v", c.events[0])
	}
	os.WriteFile(path, sampleReplay(), 0o644)
	if c = poll(t, follower); c.events[1].Event.Kind != EventPacket || c.events[1].Event.Seq != 0 {
		t.Fatalf("%+v", c.events)
	}
	replacement := filepath.Join(dir, "replacement")
	os.WriteFile(replacement, sampleReplay(), 0o644)
	os.Rename(replacement, path)
	if c = poll(t, follower); len(c.events) != 5 || c.events[0].SessionID != 3 || c.events[0].Event.Reason != "file_replaced" {
		t.Fatalf("%+v", c.events)
	}
	os.Remove(path)
	if c = poll(t, follower); len(c.events) != 1 {
		t.Fatalf("%+v", c.events)
	}
	follower.Poll(func(FollowEvent) error { t.Fatal("repeated removal"); return nil })
	os.WriteFile(path, sampleReplay(), 0o644)
	if c = poll(t, follower); c.events[0].SessionID != 4 {
		t.Fatalf("%+v", c.events)
	}
}

func healthFrame(hp uint16) []byte {
	payload := binary.LittleEndian.AppendUint32(nil, 42)
	payload = binary.LittleEndian.AppendUint32(payload, 3)
	payload = binary.LittleEndian.AppendUint32(payload, 2)
	payload = binary.LittleEndian.AppendUint16(payload, hp)
	return frame(7, 1, payload)
}

func healthUpdate(t *testing.T, event FollowEvent) map[string]any {
	t.Helper()
	telemetry := eventJSON(t, event)["telemetry"].(map[string]any)
	return telemetry["health_updates"].([]any)[0].(map[string]any)
}

func TestFollowerPreservesHPStateAndClearsItAfterRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.replay")
	os.WriteFile(path, append(header(), healthFrame(100)...), 0o644)
	follower := NewFollower(path)
	defer follower.Close()
	c := poll(t, follower)
	if healthUpdate(t, c.events[1])["previous_hp"] != nil {
		t.Fatal("previous hp")
	}
	writer, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	writer.Write(healthFrame(70))
	writer.Close()
	c = poll(t, follower)
	update := healthUpdate(t, c.events[0])
	if update["previous_hp"] != 100.0 || update["hp_lost_since_previous_update"] != 30.0 || update["packet_index"] != 1.0 {
		t.Fatal(update)
	}
	// Same inode and same length, rewritten between polls.
	rewritten := append(append(header(), healthFrame(200)...), healthFrame(180)...)
	file, _ := os.OpenFile(path, os.O_WRONLY, 0)
	file.WriteAt(rewritten, 0)
	file.Close()
	if c = poll(t, follower); c.events[0].Event.Reason != "file_rewritten" {
		t.Fatalf("%+v", c.events)
	}
	c = poll(t, follower)
	if c.events[1].SessionID != 2 || healthUpdate(t, c.events[1])["previous_hp"] != nil ||
		healthUpdate(t, c.events[2])["hp_lost_since_previous_update"] != 20.0 {
		t.Fatalf("%+v", c.events)
	}
}

func TestUnsupportedVersionsKeepPacketsWithoutLiveTelemetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.replay")
	data := sampleReplay()
	data[18] = 'X'
	os.WriteFile(path, data, 0o644)
	follower := NewFollower(path)
	defer follower.Close()
	c := poll(t, follower)
	if c.events[1].Telemetry != nil || !c.events[1].HasTelemetry || !bytes.Equal(c.events[1].Event.Packet.RawPayload, []byte{1, 2, 3}) {
		t.Fatalf("%+v", c.events[1])
	}
	if !strings.Contains(string(mustMarshal(t, &c.events[1])), `"telemetry":null`) {
		t.Fatal("telemetry must be null")
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	data, err := jsonx.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func sampleData(t *testing.T) []byte {
	t.Helper()
	archive, err := zip.OpenReader("../testdata/sample.tbreplay")
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	entry, err := archive.Open("data.replay")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(entry)
	return data
}

func TestFollowerKeepsBattleCountersBetweenPollsAndResetsOnTruncation(t *testing.T) {
	data := sampleData(t)
	var base, assist, blocked []byte
	for offset := 65; offset+12 <= len(data); {
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		kind := binary.LittleEndian.Uint32(data[offset+4:])
		end := offset + 12 + length
		raw := data[offset+12 : end]
		if kind == 0 {
			base = data[offset:end]
		}
		if kind == 8 && len(raw) == 18 && binary.LittleEndian.Uint32(raw[4:]) == 14 {
			if raw[12] == 17 && assist == nil {
				assist = data[offset:end]
			}
			if raw[12] == 5 {
				blocked = data[offset:end]
			}
		}
		offset = end
	}
	if base == nil || assist == nil || blocked == nil {
		t.Fatal("fixture frames missing")
	}
	path := filepath.Join(t.TempDir(), "data.replay")
	initial := append(append(append([]byte(nil), data[:65]...), base...), assist...)
	os.WriteFile(path, initial, 0o644)
	follower := NewFollower(path)
	defer follower.Close()
	counters := func(event FollowEvent) map[string]any {
		return eventJSON(t, event)["telemetry"].(map[string]any)["battle_counters"].(map[string]any)
	}
	c := poll(t, follower)
	if got := counters(c.events[len(c.events)-1]); got["damage_assisted"] != 706.0 || got["damage_blocked"] != nil {
		t.Fatal(got)
	}
	writer, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	writer.Write(blocked)
	writer.Close()
	c = poll(t, follower)
	if got := counters(c.events[0]); got["damage_assisted"] != 706.0 || got["damage_blocked"] != 540.0 {
		t.Fatal(got)
	}
	os.WriteFile(path, initial, 0o644)
	c = poll(t, follower)
	last := c.events[len(c.events)-1]
	if c.events[0].Event.Reason != "file_truncated" || last.SessionID != 2 {
		t.Fatalf("%+v", c.events)
	}
	if got := counters(last); got["damage_assisted"] != 706.0 || got["damage_blocked"] != nil {
		t.Fatal(got)
	}
}

func TestStandaloneMatchesStreamDecoderAndRejectsTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.replay")
	data := sampleReplay()
	os.WriteFile(path, data, 0o644)
	analysis, err := AnalyzeReplay(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	summary := analysis.Telemetry.(jsonx.Object)["summary"].(jsonx.Object)
	packets := analysis.Data.(jsonx.Object)["packets"].([]any)
	if analysis.Meta != nil || analysis.BattleResults != nil || len(packets) != 3 ||
		summary["shots_in_battle_results"] != nil || summary["shot_count_matches_results"] != nil {
		t.Fatalf("%+v", analysis)
	}
	var streamed []*Packet
	(&StreamDecoder{}).Feed(data, func(event StreamEvent) error {
		if event.Kind == EventPacket {
			streamed = append(streamed, event.Packet)
		}
		return nil
	})
	if valueJSON(t, packets) != valueJSON(t, streamed) {
		t.Fatal("standalone packets differ from stream decoder")
	}
	os.WriteFile(path, data[:len(data)-1], 0o644)
	if _, err := AnalyzeReplay(path, Options{}); blitz.AsError(err).Code != "REPLAY_DATA_INVALID" {
		t.Fatal(err)
	}
}

func TestFollowerReleasesRecordingBetweenPolls(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "recording_test.tbreplay")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "data.replay")
	data := sampleReplay()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	follower := NewFollower(path)
	defer follower.Close()
	first := poll(t, follower)
	if len(first.events) == 0 {
		t.Fatal("no initial events")
	}
	if follower.stream.file != nil {
		t.Fatal("poll retained recording handle")
	}
	// Reopening must preserve the offset and avoid replaying old packets.
	if again := poll(t, follower); len(again.events) != 0 {
		t.Fatal("reopen duplicated events")
	}
	if err := os.RemoveAll(directory); err != nil {
		t.Fatalf("follower blocked recording finalization: %v", err)
	}
	removed := poll(t, follower)
	if len(removed.events) != 1 || removed.events[0].Event.Reason != "file_removed" {
		t.Fatalf("missing removal reset: %+v", removed.events)
	}
	if again := poll(t, follower); len(again.events) != 0 {
		t.Fatal("removal reset repeated")
	}
}
