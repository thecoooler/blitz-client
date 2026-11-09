package replay

import (
	"encoding/binary"
	"strings"
	"testing"
)

func representativeLayoutPackets() []*Packet {
	position := make([]byte, 28)
	binary.LittleEndian.PutUint32(position, 7)
	binary.LittleEndian.PutUint32(position[12:], 99)

	launch := make([]byte, 49)
	binary.LittleEndian.PutUint32(launch[12:], 11)
	binary.LittleEndian.PutUint32(launch[16:], 99)
	launch[20] = 4

	auxiliary := make([]byte, 46)
	binary.LittleEndian.PutUint32(auxiliary[12:], 99)

	encoded := pickleOf(map[string]any{"Booster_shells": 12})
	boosterBody := append(append(make([]byte, 9), byte(len(encoded))), encoded...)
	booster := binary.LittleEndian.AppendUint32(nil, 4)
	booster = binary.LittleEndian.AppendUint32(booster, 51)
	booster = binary.LittleEndian.AppendUint32(booster, uint32(len(boosterBody)))
	booster = append(booster, boosterBody...)

	key := "tj_kill_btls_lvl4_slot2"
	taskBody := append([]byte{byte(len(key))}, key...)
	task := binary.LittleEndian.AppendUint32(nil, 7)
	task = binary.LittleEndian.AppendUint32(task, 48)
	task = binary.LittleEndian.AppendUint32(task, uint32(len(taskBody)))
	task = append(task, taskBody...)

	health := make([]byte, 14)
	binary.LittleEndian.PutUint32(health, 42)
	binary.LittleEndian.PutUint32(health[4:], 3)
	binary.LittleEndian.PutUint32(health[8:], 2)
	binary.LittleEndian.PutUint16(health[12:], 100)

	unclassifiedType7 := make([]byte, 12)
	binary.LittleEndian.PutUint32(unclassifiedType7, 42)
	binary.LittleEndian.PutUint32(unclassifiedType7[4:], 9)
	binary.LittleEndian.PutUint32(unclassifiedType7[8:], 7)

	return []*Packet{
		{ClockSecs: 1, Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 25}}, RawPayload: position},
		{ClockSecs: 2, Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 35}}, RawPayload: launch},
		{ClockSecs: 3.5, Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 9}}, RawPayload: method9BodyBytes(11, 22, 600, 1)},
		{ClockSecs: 4, Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 33}}, RawPayload: auxiliary},
		{ClockSecs: 5, Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 51}}, RawPayload: booster},
		{ClockSecs: 6, Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 48}}, RawPayload: task},
		{ClockSecs: 7, Payload: Payload{Kind: PayloadUnknown, PacketType: 10}, RawPayload: make([]byte, 49)},
		{ClockSecs: 8, Payload: Payload{Kind: PayloadUnknown, PacketType: 7}, RawPayload: health},
		{ClockSecs: 9, Payload: Payload{Kind: PayloadUnknown, PacketType: 7}, RawPayload: unclassifiedType7},
		{ClockSecs: 10, Payload: Payload{Kind: PayloadUnknown, PacketType: 5}, RawPayload: []byte{1, 0, 0, 0, 9}},
	}
}

func method9BodyBytes(source, target uint32, damage uint16, flag byte) []byte {
	raw := make([]byte, 33)
	binary.LittleEndian.PutUint32(raw, target) // Entity/receiver ID.
	binary.LittleEndian.PutUint32(raw[12:], source)
	binary.LittleEndian.PutUint32(raw[16:], target)
	binary.LittleEndian.PutUint16(raw[30:], damage)
	raw[32] = flag
	return raw
}

// TestTelemetryUpdateRendersTheSameDetailsAsOnPacket exercises the typed seam
// and its renderer over every recovered live annotation family.
func TestTelemetryUpdateRendersTheSameDetailsAsOnPacket(t *testing.T) {
	packets := representativeLayoutPackets()
	rendered := NewTelemetryDecoder()
	typed := NewTelemetryDecoder()
	for index, packet := range packets {
		details := rendered.OnPacket(packet, index, recoveredLayoutVersion)
		update := typed.OnPacketUpdate(packet, index, recoveredLayoutVersion)
		if update == nil {
			t.Fatalf("packet %d: typed update is nil", index)
		}
		if got, want := valueJSON(t, update.details()), valueJSON(t, details); got != want {
			t.Fatalf("packet %d:\n got %s\nwant %s", index, got, want)
		}
		object := details.(obj)
		if (update.Annotation != nil) != (object["annotation"] != nil) {
			t.Fatalf("packet %d: annotation presence differs", index)
		}
		for _, pair := range []struct {
			name     string
			typed    []obj
			rendered any
		}{
			{"shots", update.Shots, object["shots"]},
			{"hits", update.Hits, object["hits"]},
			{"health_updates", update.HealthUpdates, object["health_updates"]},
			{"vehicle_updates", update.VehicleUpdates, object["vehicle_updates"]},
			{"booster_counter_snapshots", update.BoosterCounterSnapshots, object["booster_counter_snapshots"]},
			{"method_48_task_markers", update.Method48TaskMarkers, object["method_48_task_markers"]},
		} {
			if got, want := len(pair.typed), len(pair.rendered.([]any)); got != want {
				t.Fatalf("packet %d: %s has %d typed entries, rendered %d", index, pair.name, got, want)
			}
		}
		if update.PacketLayoutDecodingSupported != object["packet_layout_decoding_supported"] ||
			update.BattleCounterDecodingSupported != object["battle_counter_decoding_supported"] {
			t.Fatalf("packet %d: support flags differ", index)
		}
	}
}

func TestTelemetryUpdateTypesBattleCountersAndKeepsTotals(t *testing.T) {
	packets := []*Packet{basePlayer(t, 42), ribbon(42, 17, 706), ribbon(42, 5, 540)}
	rendered := NewTelemetryDecoder()
	typed := NewTelemetryDecoder()
	var last *TelemetryUpdate
	for index, packet := range packets {
		details := rendered.OnPacket(packet, index, counterVersion).(obj)
		last = typed.OnPacketUpdate(packet, index, counterVersion)
		if last == nil {
			t.Fatalf("packet %d: typed update is nil", index)
		}
		if got, want := valueJSON(t, last.details()), valueJSON(t, details); got != want {
			t.Fatalf("packet %d:\n got %s\nwant %s", index, got, want)
		}
	}
	if last.BattleCounters.DamageAssisted == nil || *last.BattleCounters.DamageAssisted != 706 {
		t.Fatalf("assisted total: %+v", last.BattleCounters)
	}
	if last.BattleCounters.DamageBlocked == nil || *last.BattleCounters.DamageBlocked != 540 {
		t.Fatalf("blocked total: %+v", last.BattleCounters)
	}
}

func TestTelemetryUpdateRetainsCounterUpdateShape(t *testing.T) {
	decoder := NewTelemetryDecoder()
	decoder.OnPacketUpdate(basePlayer(t, 42), 0, counterVersion)
	update := decoder.OnPacketUpdate(ribbon(42, 17, 706), 1, counterVersion)
	if len(update.BattleCounterUpdates) != 1 || update.Annotation == nil {
		t.Fatalf("update: %+v", update)
	}
	if got := update.Annotation["kind"]; got != annotationBattleCounterUpdate {
		t.Fatalf("kind: %v", got)
	}
	counter := update.BattleCounterUpdates[0]
	if counter.Counter != CounterDamageAssisted || counter.Value != 706 || counter.PacketIndex != 1 {
		t.Fatalf("counter: %+v", counter)
	}
	if update.BattleCounters.DamageAssisted == nil || *update.BattleCounters.DamageAssisted != 706 {
		t.Fatalf("totals: %+v", update.BattleCounters)
	}
}

func TestTelemetryRendererEmitsStableLabelsAndNotes(t *testing.T) {
	labels := map[string]string{
		annotationProjectilePosition:    "Точка снаряда",
		annotationProjectileLaunch:      "Запуск снаряда",
		annotationHitReceived:           "Сообщение о попадании",
		annotationProjectileAuxiliary:   "Дополнительное сообщение снаряда",
		annotationMethod51Counters:      "Счётчики предметов · метод 51",
		annotationMethod48TaskMarker:    "Ключ задачи · метод 48",
		annotationVehicleTransform:      "Обновление положения техники",
		annotationVehicleHealthUpdate:   "Обновление оставшегося HP",
		annotationBattleResultsSnapshot: "Снимок итогов боя",
	}
	for kind, want := range labels {
		if got := annotation(kind)["label"]; got != want {
			t.Errorf("%s label = %v, want %q", kind, got, want)
		}
	}
	if _, ok := annotation(annotationBattleCounterUpdate)["label"]; ok {
		t.Fatal("battle counter annotation must not grow a label")
	}
	if annotationUnclassifiedEntityUpdate != "unclassified_entity_update" ||
		annotationUnclassifiedEntityMessage != "unclassified_entity_message" {
		t.Fatal("unclassified annotation kinds changed")
	}
	if telemetryDecoderEmpirical != "empirical-ruby-apple-26.8" ||
		telemetryDecoderUnsupported != "unsupported-client-version" ||
		telemetryDecoderBattleCountersOnly != "battle-counters-only" ||
		recoveredLayoutVersion != "26.8.0_ruby_apple" {
		t.Fatal("decoder or version text changed")
	}
	notes := supportedLayoutSchemaNotes()
	if len(notes) != 11 || notes[0] != "Подтверждены только указанные варианты длины; остальные payload остаются сырыми." {
		t.Fatalf("supported schema notes: %v", notes)
	}
	if last, _ := notes[len(notes)-1].(string); !strings.Contains(last, "Subtype 48") {
		t.Fatalf("last schema note: %q", notes[len(notes)-1])
	}
	if unsupported := unsupportedLayoutSchemaNotes(); len(unsupported) != 1 ||
		!strings.Contains(unsupported[0].(string), "Байтовые смещения подтверждены только для указанной версии") {
		t.Fatalf("unsupported schema notes: %v", unsupported)
	}
	if note := battleCounterSchemaNote(); !strings.Contains(note, "ribbon 5") || !strings.Contains(note, "ribbon 17") {
		t.Fatalf("battle counter note: %q", note)
	}
}

func TestTelemetryRendererEmitsStableAnnotationJSON(t *testing.T) {
	hitState := newTelemetryState()
	hit := decodeTelemetryPacket(
		&Packet{ClockSecs: 3.5, Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 9}}, RawPayload: method9BodyBytes(11, 22, 600, 1)},
		2, nil, nil, hitState)
	wantHit := `{"kind":"hit_received","label":"Сообщение о попадании","opaque_hex":"00000000000000000000580201","source_player":null,"source_result_id":11,"target_player":null,"target_result_id":22}`
	if got := valueJSON(t, hit); got != wantHit {
		t.Fatalf("hit annotation:\n got %s\nwant %s", got, wantHit)
	}

	health := make([]byte, 14)
	binary.LittleEndian.PutUint32(health, 42)
	binary.LittleEndian.PutUint32(health[4:], 3)
	binary.LittleEndian.PutUint32(health[8:], 2)
	binary.LittleEndian.PutUint16(health[12:], 100)
	healthState := newTelemetryState()
	healthAnnotation := decodeTelemetryPacket(
		&Packet{ClockSecs: 8, Payload: Payload{Kind: PayloadUnknown, PacketType: 7}, RawPayload: health},
		7, nil, nil, healthState)
	wantHealth := `{"entity_id":42,"health_change":null,"health_update_index":0,"hp_lost_since_previous_update":null,"kind":"vehicle_health_update","label":"Обновление оставшегося HP","player":null,"previous_hp":null,"remaining_hp":100}`
	if got := valueJSON(t, healthAnnotation); got != wantHealth {
		t.Fatalf("health annotation:\n got %s\nwant %s", got, wantHealth)
	}

	type5State := newTelemetryState()
	type5 := decodeTelemetryPacket(
		&Packet{ClockSecs: 10, Payload: Payload{Kind: PayloadUnknown, PacketType: 5}, RawPayload: []byte{1, 0, 0, 0, 9}},
		9, nil, nil, type5State)
	wantType5 := `{"first_word":1,"kind":"unclassified_entity_message","label":"Сообщение типа 5 · маркер 9","marker":9,"tail_hex":""}`
	if got := valueJSON(t, type5); got != wantType5 {
		t.Fatalf("type 5 annotation:\n got %s\nwant %s", got, wantType5)
	}
}

// TestOnPacketRendersTheHistoricalDetailsShape pins one complete live
// telemetry object, so splitting decoding from rendering cannot drift.
func TestOnPacketRendersTheHistoricalDetailsShape(t *testing.T) {
	decoder := NewTelemetryDecoder()
	packet := &Packet{
		ClockSecs:  3.5,
		Payload:    Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 9}},
		RawPayload: method9BodyBytes(11, 22, 600, 1),
	}
	want := `{"annotation":{"kind":"hit_received","label":"Сообщение о попадании","opaque_hex":"00000000000000000000580201","source_player":null,"source_result_id":11,"target_player":null,"target_result_id":22},"battle_counter_decoding_supported":true,"battle_counter_updates":[],"battle_counters":{"author_nickname":null,"damage_assisted":null,"damage_blocked":null,"damage_dealt":null,"player_entity_id":null},"booster_counter_snapshots":[],"health_updates":[],"hits":[{"clock":3.5,"entity_id":22,"opaque_hex":"00000000000000000000580201","packet_index":2,"source_player":null,"source_result_id":11,"target_matches_entity_id":true,"target_player":null,"target_result_id":22}],"method_48_task_markers":[],"packet_layout_decoding_supported":true,"shots":[],"vehicle_updates":[]}`
	if got := valueJSON(t, decoder.OnPacket(packet, 2, recoveredLayoutVersion)); got != want {
		t.Fatalf("details shape:\n got %s\nwant %s", got, want)
	}
}

func TestMethod9BodyIsSharedByHitAndDamagePaths(t *testing.T) {
	packet := &Packet{
		ClockSecs:  3.5,
		Payload:    Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 9}},
		RawPayload: method9BodyBytes(11, 22, 600, 1),
	}
	body, ok := parseMethod9Body(packet.RawPayload)
	if !ok || body.EntityID != 22 || body.SourceResultID != 11 || body.TargetResultID != 22 ||
		body.Damage != 600 || body.Flag != 1 || len(body.Opaque) != 13 {
		t.Fatalf("parsed body: %+v %v", body, ok)
	}
	if hit := AppliedDamage(packet, counterVersion); hit == nil || *hit != (DamageEvent{11, 22, 600}) {
		t.Fatalf("applied damage: %+v", hit)
	}
	state := newTelemetryState()
	annotation := decodeTelemetryPacket(packet, 0, nil, nil, state)
	if got := annotation.(obj)["kind"]; got != annotationHitReceived {
		t.Fatalf("hit annotation kind: %v", got)
	}
	if len(state.hits) != 1 || state.hits[0]["source_result_id"] != uint32(11) || state.hits[0]["target_result_id"] != uint32(22) {
		t.Fatalf("hit records: %+v", state.hits)
	}

	// Bounce flags still produce a hit message but no applied damage.
	packet.RawPayload = method9BodyBytes(11, 22, 600, 0)
	if _, ok := parseMethod9Body(packet.RawPayload); !ok {
		t.Fatal("parser rejected a structurally valid body")
	}
	if AppliedDamage(packet, counterVersion) != nil {
		t.Fatal("bounce counted as applied damage")
	}
	state = newTelemetryState()
	if annotation := decodeTelemetryPacket(packet, 0, nil, nil, state); annotation == nil || len(state.hits) != 1 {
		t.Fatal("hit message was dropped for a bounce flag")
	}

	// Damage flags 1 and 3 are accepted; any other flag is not.
	for _, flag := range []byte{1, 3} {
		packet.RawPayload = method9BodyBytes(11, 22, 600, flag)
		if AppliedDamage(packet, counterVersion) == nil {
			t.Fatalf("flag %d rejected", flag)
		}
	}

	// Body-target mismatch and zero damage keep the hit message but are not
	// counted as applied damage.
	mismatch := method9BodyBytes(11, 22, 600, 1)
	binary.LittleEndian.PutUint32(mismatch, 23)
	packet.RawPayload = mismatch
	if AppliedDamage(packet, counterVersion) != nil {
		t.Fatal("mismatched receiver counted")
	}
	if _, ok := parseMethod9Body(packet.RawPayload); !ok {
		t.Fatal("parser must not apply damage acceptance rules")
	}
	packet.RawPayload = method9BodyBytes(11, 22, 0, 1)
	if AppliedDamage(packet, counterVersion) != nil {
		t.Fatal("zero damage counted")
	}

	// A malformed length is rejected by both the parser and callers.
	for _, raw := range [][]byte{
		make([]byte, 32),
		make([]byte, 34),
	} {
		if _, ok := parseMethod9Body(raw); ok {
			t.Fatalf("parser accepted %d-byte body", len(raw))
		}
		packet.RawPayload = raw
		if AppliedDamage(packet, counterVersion) != nil {
			t.Fatalf("damage accepted %d-byte body", len(raw))
		}
		state = newTelemetryState()
		if annotation := decodeTelemetryPacket(packet, 0, nil, nil, state); annotation != nil || len(state.hits) != 0 {
			t.Fatalf("hit accepted %d-byte body", len(raw))
		}
	}
}
