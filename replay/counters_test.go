package replay

import (
	"encoding/binary"
	"math"
	"testing"
)

const counterVersion = "26.10.0_ruby_apple"

func basePlayer(t *testing.T, entity uint32) *Packet {
	t.Helper()
	encoded := pickleOf(map[string]any{
		"battleLevel": 10, "battleCategoryId": 1, "mouseEnabled": true,
		"mmType": 0, "camouflageSlot": 0, "accountDatabaseIds": []any{42},
	})
	raw := binary.LittleEndian.AppendUint32(nil, entity)
	raw = append(raw, make([]byte, 6)...)
	raw = append(raw, "\x04Test"...)
	raw = binary.LittleEndian.AppendUint64(raw, 1)
	raw = binary.LittleEndian.AppendUint32(raw, 1)
	raw = append(raw, 255)
	raw = binary.LittleEndian.AppendUint16(raw, uint16(len(encoded)))
	raw = append(append(raw, 0), encoded...)
	payload, err := NewPayload(0, raw)
	if err != nil {
		t.Fatal(err)
	}
	return &Packet{Payload: payload, RawPayload: raw}
}

func ribbon(entity uint32, id byte, value uint16) *Packet {
	raw := binary.LittleEndian.AppendUint32(nil, entity)
	raw = binary.LittleEndian.AppendUint32(raw, 14)
	raw = binary.LittleEndian.AppendUint32(raw, 6)
	raw = append(raw, id, 0, 0, 0)
	raw = binary.LittleEndian.AppendUint16(raw, value)
	return &Packet{
		ClockSecs:  1,
		Payload:    Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 14}},
		RawPayload: raw,
	}
}

func TestUsesReportedTotalsAndHandlesDuplicatesCorrectionsAndResets(t *testing.T) {
	decoder := &BattleCounterDecoder{}
	if decoder.OnPacket(ribbon(42, 17, 100), 0, counterVersion) != nil {
		t.Fatal("update before author")
	}
	decoder.OnPacket(basePlayer(t, 42), 1, counterVersion)
	if decoder.Counters().DamageAssisted != nil || decoder.OnPacket(ribbon(99, 17, 999), 2, counterVersion) != nil {
		t.Fatal("unexpected counter")
	}
	first := decoder.OnPacket(ribbon(42, 17, 100), 3, counterVersion)
	if first.Value != 100 || first.Delta != nil {
		t.Fatalf("%+v", first)
	}
	second := decoder.OnPacket(ribbon(42, 17, 250), 4, counterVersion)
	if *second.PreviousValue != 100 || *second.Delta != 150 {
		t.Fatalf("%+v", second)
	}
	if *decoder.OnPacket(ribbon(42, 17, 250), 5, counterVersion).Delta != 0 ||
		*decoder.OnPacket(ribbon(42, 17, 240), 6, counterVersion).Delta != -10 {
		t.Fatal("delta")
	}
	decoder.OnPacket(ribbon(42, 5, 540), 7, counterVersion)
	if *decoder.Counters().DamageBlocked != 540 || *decoder.Counters().DamageAssisted != 240 {
		t.Fatal("totals")
	}
	decoder.OnPacket(basePlayer(t, 7), 8, counterVersion)
	if *decoder.Counters().PlayerEntityID != 7 || decoder.Counters().DamageBlocked != nil ||
		decoder.OnPacket(ribbon(42, 5, 100), 9, counterVersion) != nil {
		t.Fatal("new author")
	}
	decoder.Reset()
	if decoder.Counters().PlayerEntityID != nil {
		t.Fatal("reset")
	}
}

func TestRejectsUnverifiedVersionsAndMalformedOrUnrelatedPackets(t *testing.T) {
	decoder := &BattleCounterDecoder{}
	decoder.OnPacket(basePlayer(t, 42), 0, counterVersion)
	for length := range 18 {
		packet := ribbon(42, 5, 540)
		packet.RawPayload = packet.RawPayload[:length]
		if decoder.OnPacket(packet, 1, counterVersion) != nil {
			t.Fatal(length)
		}
	}
	for _, change := range [][2]int{{4, 13}, {8, 5}, {12, 1}} {
		packet := ribbon(42, 5, 540)
		packet.RawPayload[change[0]] = byte(change[1])
		if decoder.OnPacket(packet, 1, counterVersion) != nil {
			t.Fatal(change)
		}
	}
	packet := ribbon(42, 5, 540)
	packet.RawPayload = append(packet.RawPayload, 0)
	if decoder.OnPacket(packet, 1, counterVersion) != nil {
		t.Fatal("long payload")
	}
	packet = ribbon(42, 5, 540)
	packet.ClockSecs = float32(math.NaN())
	if decoder.OnPacket(packet, 1, counterVersion) != nil {
		t.Fatal("NaN clock")
	}
	if decoder.OnPacket(ribbon(42, 5, 540), 1, "26.11.0_ruby_apple") != nil || decoder.Counters().PlayerEntityID != nil {
		t.Fatal("unverified version")
	}
}

func TestLive2610ExposesCountersWithoutEnablingLegacyLayouts(t *testing.T) {
	decoder := NewTelemetryDecoder()
	context := decoder.OnPacket(basePlayer(t, 42), 0, counterVersion).(obj)
	if context["battle_counters"].(obj)["player_entity_id"] != uint64(42) || context["packet_layout_decoding_supported"] != false {
		t.Fatalf("%v", context)
	}
	update := decoder.OnPacket(ribbon(42, 17, 706), 1, counterVersion).(obj)
	if update["annotation"].(obj)["kind"] != "battle_counter_update" || len(update["battle_counter_updates"].([]any)) != 1 {
		t.Fatalf("%v", update)
	}
	health := append(binary.LittleEndian.AppendUint32(nil, 42), make([]byte, 10)...)
	binary.LittleEndian.PutUint32(health[4:], 3)
	binary.LittleEndian.PutUint32(health[8:], 2)
	other := decoder.OnPacket(&Packet{Payload: Payload{PacketType: 7}, RawPayload: health}, 2, counterVersion).(obj)
	if other["annotation"] != nil || len(other["health_updates"].([]any)) != 0 {
		t.Fatalf("%v", other)
	}
}

func TestTelemetryHelpers(t *testing.T) {
	raw := []byte{1, 0, 0, 0, 0, 0, 0x80, 0x3f}
	if v, _ := readU32(raw, 0); v != 1 {
		t.Fatal(v)
	}
	if v, ok := readF32(raw, 4); !ok || v != 1 {
		t.Fatal(v)
	}
	if _, ok := readU32(raw, 5); ok {
		t.Fatal("out of range")
	}
	if _, ok := readVec3(raw, 0); ok {
		t.Fatal("vec3 out of range")
	}
	if _, ok := readF32([]byte{0xff, 0xff, 0xff, 0x7f}, 0); ok {
		t.Fatal("NaN accepted")
	}
	if hexString([]byte{0, 0x0f, 0xa5, 0xff}) != "000fa5ff" {
		t.Fatal("hex")
	}
	method := func(entity, subType uint32, body []byte) []byte {
		raw := binary.LittleEndian.AppendUint32(nil, entity)
		raw = binary.LittleEndian.AppendUint32(raw, subType)
		raw = binary.LittleEndian.AppendUint32(raw, uint32(len(body)))
		return append(raw, body...)
	}
	key := "tj_kill_btls_lvl4_slot2"
	if got, ok := decodeMethod48TaskKey(method(7, 48, append([]byte{byte(len(key))}, key...))); !ok || got != key {
		t.Fatal(got)
	}
	if _, ok := decodeMethod48TaskKey(method(7, 48, []byte("\x05other"))); ok {
		t.Fatal("other key accepted")
	}
	encoded := pickleOf(map[string]any{"Booster_shells": 12, "Other": 3})
	body := append(append(make([]byte, 9), byte(len(encoded))), encoded...)
	counters, ok := decodeMethod51Counters(method(4, 51, body))
	if !ok || len(counters) != 1 || counters["Booster_shells"] != uint64(12) {
		t.Fatal(counters)
	}
	if _, ok := decodeMethod51Counters(method(4, 51, body[:len(body)-1])); ok {
		t.Fatal("truncated pickle accepted")
	}
	if _, ok := decodeMethod51Counters(method(4, 50, body)); ok {
		t.Fatal("wrong method accepted")
	}
}

func TestLiveCountersRetainTotalsWithoutRepeatingUpdates(t *testing.T) {
	decoder := NewTelemetryDecoder()
	decoder.OnPacket(basePlayer(t, 42), 0, counterVersion)
	details := decoder.OnPacket(ribbon(42, 17, 1920), 1, counterVersion).(obj)
	if details["battle_counters"].(obj)["damage_assisted"] != uint64(1920) {
		t.Fatalf("%v", details)
	}
	next := decoder.OnPacket(ribbon(42, 2, 0), 2, counterVersion).(obj)
	if next["battle_counters"].(obj)["damage_assisted"] != uint64(1920) || len(next["battle_counter_updates"].([]any)) != 0 {
		t.Fatalf("%v", next)
	}
}

func TestRealStreamEmitsExactAssistanceAndBlockedValuesBeforeBattleEnd(t *testing.T) {
	data := sampleData(t)
	stream := &StreamDecoder{}
	counters := &BattleCounterDecoder{}
	version := ""
	var assisted, blocked []uint32
	for start := 0; start < len(data); start += 127 {
		err := stream.Feed(data[start:min(start+127, len(data))], func(event StreamEvent) error {
			switch event.Kind {
			case EventHeader:
				version = event.ClientVersion
			case EventPacket:
				if update := counters.OnPacket(event.Packet, event.Seq, version); update != nil {
					if update.Clock >= 182 {
						t.Fatalf("%+v", update)
					}
					if update.Counter == CounterDamageAssisted {
						assisted = append(assisted, update.Value)
					} else {
						blocked = append(blocked, update.Value)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := stream.Finish(); err != nil || version != counterVersion {
		t.Fatal(err, version)
	}
	got := counters.Counters()
	if *got.AuthorNickname != "iCooLER" || *got.DamageAssisted != 1920 || *got.DamageBlocked != 540 {
		t.Fatalf("%+v", got)
	}
	if !slicesEqual(assisted, []uint32{706, 1083, 1499, 1920}) || !slicesEqual(blocked, []uint32{540}) {
		t.Fatal(assisted, blocked)
	}
}

func slicesEqual(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}
