package replay

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/pickle"
)

const recoveredLayoutVersion = "26.8.0_ruby_apple"

type obj = jsonx.Object

func readU16(data []byte, offset int) (uint16, bool) {
	if offset < 0 || offset+2 > len(data) {
		return 0, false
	}
	return binary.LittleEndian.Uint16(data[offset:]), true
}

func readF32(data []byte, offset int) (float32, bool) {
	value, ok := readU32(data, offset)
	if !ok {
		return 0, false
	}
	number := math.Float32frombits(value)
	if math.IsNaN(float64(number)) || math.IsInf(float64(number), 0) {
		return 0, false
	}
	return number, true
}

func readVec3(data []byte, offset int) ([]any, bool) {
	out := make([]any, 3)
	for axis := range 3 {
		value, ok := readF32(data, offset+axis*4)
		if !ok {
			return nil, false
		}
		out[axis] = jsonx.F64(value)
	}
	return out, true
}

// optU32 mirrors json!(Option<u32>).
func optU32(data []byte, offset int) any {
	if value, ok := readU32(data, offset); ok {
		return value
	}
	return nil
}

type hitRecord struct {
	clock          float32
	targetResultID uint32
	sourceResultID uint32
	packetIndex    int
}

type healthRecord struct {
	clock       float32
	entityID    uint32
	packetIndex int
	hpLost      int32
}

type telemetryState struct {
	shots                   []obj
	shotIndices             map[uint32][]int
	launchesByProjectile    map[uint32][]any
	auxiliariesByProjectile map[uint32][]any
	hits                    []obj
	hitRecords              []hitRecord
	vehicleUpdates          []obj
	healthUpdates           []obj
	healthRecords           []healthRecord
	boosterSnapshots        []obj
	method48Markers         []obj
	previousHealth          map[uint32]uint16
	resultSnapshots         int
	type7Opcodes            map[uint32]int
	type7Sizes              map[int]int
	type5Sizes              map[int]int
}

func newTelemetryState() *telemetryState {
	return &telemetryState{
		shotIndices:             map[uint32][]int{},
		launchesByProjectile:    map[uint32][]any{},
		auxiliariesByProjectile: map[uint32][]any{},
		previousHealth:          map[uint32]uint16{},
		type7Opcodes:            map[uint32]int{},
		type7Sizes:              map[int]int{},
		type5Sizes:              map[int]int{},
	}
}

func objects(items []obj) []any {
	out := make([]any, len(items))
	for index, item := range items {
		out[index] = item
	}
	return out
}

func countObject[K uint32 | int](counts map[K]int) obj {
	out := obj{}
	for key, value := range counts {
		out[strconv.FormatInt(int64(key), 10)] = value
	}
	return out
}

func playerJSON(players map[uint32]obj, resultID uint32) any {
	if player, ok := players[resultID]; ok {
		return player
	}
	return nil
}

// recoverTelemetry decodes the packet layouts verified against local
// replays while keeping every original payload available.
func recoverTelemetry(packets []*Packet, clientVersion string, results *BattleResults, resultsBlob []byte) obj {
	report := recoverLayoutTelemetry(packets, clientVersion, results, resultsBlob)
	supported := SupportsBattleCounters(clientVersion)
	report["summary"].(obj)["battle_counter_decoding_supported"] = supported
	if !supported {
		return report
	}
	decoder := &BattleCounterDecoder{}
	updates := []any{}
	annotations := report["packet_annotations"].([]any)
	for index, packet := range packets {
		if update := decoder.OnPacket(packet, index, clientVersion); update != nil {
			value := jsonx.ToValue(update)
			annotations[index] = obj{"kind": "battle_counter_update", "update": value}
			updates = append(updates, value)
		}
	}
	counters := decoder.Counters()
	report["battle_counters"] = jsonx.ToValue(counters)
	report["battle_counter_updates"] = updates
	report["battle_counter_supported_client_versions"] = jsonx.ToValue(BattleCounterClientVersions)
	if clientVersion != recoveredLayoutVersion {
		report["decoder"] = "battle-counters-only"
		delete(report, "supported_client_version")
	}
	var blocked, assisted any
	var expectedBlocked, expectedAssisted *uint64
	if results != nil {
		for _, row := range results.PlayerResults {
			if row.Info.AccountID == results.Author.AccountID {
				b := uint64(row.Info.DamageBlocked)
				a := uint64(row.Info.DamageAssisted1) + uint64(row.Info.DamageAssisted2)
				expectedBlocked, expectedAssisted = &b, &a
				blocked, assisted = b, a
				break
			}
		}
	}
	matches := func(actual *uint32, expected *uint64) any {
		if actual == nil || expected == nil {
			return nil
		}
		return uint64(*actual) == *expected
	}
	report["battle_counter_result_comparison"] = obj{
		"damage_blocked_in_results":       blocked,
		"damage_assisted_in_results":      assisted,
		"damage_blocked_matches_results":  matches(counters.DamageBlocked, expectedBlocked),
		"damage_assisted_matches_results": matches(counters.DamageAssisted, expectedAssisted),
	}
	report["schema_notes"] = append(report["schema_notes"].([]any),
		"Метод 14 автора боя: ribbon 5 — накопленный заблокированный урон; ribbon 17 — накопленная помощь по засвету или обездвиживанию. Значения u16 читаются прямо из пакета; отдельного разделения видов помощи нет.")
	return report
}

func arenaContext(packets []*Packet) (any, int) {
	for index, packet := range packets {
		if packet.Payload.Kind != PayloadBasePlayerCreate {
			continue
		}
		ids := 0
		if arguments := packet.Payload.Arguments; arguments != nil && arguments.AccountDatabaseIDs != nil {
			ids = len(*arguments.AccountDatabaseIDs)
		}
		return obj{
			"packet_index":    index,
			"clock":           jsonx.F64(packet.ClockSecs),
			"author_nickname": packet.Payload.AuthorNickname,
			"arena_unique_id": packet.Payload.ArenaUniqueID,
			"arena_type_id":   packet.Payload.ArenaTypeID,
			"arguments":       jsonx.ToValue(packet.Payload.Arguments),
		}, ids
	}
	return nil, 0
}

func countPackets(packets []*Packet, match func(*Payload) bool) int {
	count := 0
	for _, packet := range packets {
		if match(&packet.Payload) {
			count++
		}
	}
	return count
}

func isBasePlayerCreate(payload *Payload) bool { return payload.Kind == PayloadBasePlayerCreate }

func isUnknownType(packetType uint32) func(*Payload) bool {
	return func(payload *Payload) bool {
		return payload.Kind == PayloadUnknown && payload.PacketType == packetType
	}
}

func nullAnnotations(count int) []any { return make([]any, count) }

func recoverLayoutTelemetry(packets []*Packet, clientVersion string, results *BattleResults, resultsBlob []byte) obj {
	context, contextIDs := arenaContext(packets)
	contextPackets := countPackets(packets, isBasePlayerCreate)
	if clientVersion != recoveredLayoutVersion {
		return obj{
			"decoder":                  "unsupported-client-version",
			"client_version":           clientVersion,
			"supported_client_version": recoveredLayoutVersion,
			"summary": obj{
				"packet_layout_decoding_supported": false,
				"arena_context_packets":            contextPackets,
				"arena_context_player_ids":         contextIDs,
			},
			"schema_notes":              []any{"Байтовые смещения подтверждены только для указанной версии клиента; исходные пакеты доступны без этих предположений."},
			"shots":                     []any{},
			"hits":                      []any{},
			"arena_context":             context,
			"booster_counter_snapshots": []any{},
			"method_48_task_markers":    []any{},
			"vehicle_updates":           []any{},
			"packet_annotations":        nullAnnotations(len(packets)),
		}
	}

	players := map[uint32]obj{}
	if results != nil {
		players = playerLookup(results)
	}
	annotations := nullAnnotations(len(packets))
	state := newTelemetryState()
	for index, packet := range packets {
		annotations[index] = decodeTelemetryPacket(packet, index, players, resultsBlob, state)
	}

	for index, update := range state.healthUpdates {
		record := state.healthRecords[index]
		hitIndices, sources := []any{}, []any{}
		for _, hit := range state.hitRecords {
			if hit.targetResultID == record.entityID && math.Float32bits(hit.clock) == math.Float32bits(record.clock) {
				hitIndices = append(hitIndices, hit.packetIndex)
				sources = append(sources, hit.sourceResultID)
			}
		}
		update["matched_hit_packet_indices"] = hitIndices
		update["matched_hit_source_result_ids"] = sources
		if annotation, ok := annotations[record.packetIndex].(obj); ok {
			annotation["matched_hit_packet_indices"] = append([]any(nil), hitIndices...)
			annotation["matched_hit_source_result_ids"] = append([]any(nil), sources...)
		}
	}

	linkedShots := 0
	for projectileID, indices := range state.shotIndices {
		if launches, ok := state.launchesByProjectile[projectileID]; ok {
			for _, shotIndex := range indices {
				state.shots[shotIndex]["launch_records"] = append([]any(nil), launches...)
				linkedShots++
			}
		}
		if auxiliaries, ok := state.auxiliariesByProjectile[projectileID]; ok {
			for _, shotIndex := range indices {
				state.shots[shotIndex]["auxiliary_records"] = append([]any(nil), auxiliaries...)
			}
		}
	}

	var expectedShots, expectedHits, shotsMatch any
	if results != nil {
		var shots, hits uint32
		for _, row := range results.PlayerResults {
			shots += row.Info.NShots
			hits += row.Info.NHitsReceived
		}
		expectedShots, expectedHits = shots, hits
		shotsMatch = len(state.shots) == int(shots)
	}
	mappedHits := 0
	for _, hit := range state.hits {
		if hit["target_matches_entity_id"] == true {
			mappedHits++
		}
	}
	mappedPlayers := 0
	for _, update := range state.vehicleUpdates {
		if update["player"] != nil {
			mappedPlayers++
		}
	}
	linkedHealth, hpLoss, destroyed := 0, 0, 0
	for index, update := range state.healthUpdates {
		if len(update["matched_hit_packet_indices"].([]any)) > 0 {
			linkedHealth++
		}
		if state.healthRecords[index].hpLost > 0 {
			hpLoss++
		}
		if update["destroyed_at_zero_hp"] == true {
			destroyed++
		}
	}
	// Map iteration order is random, so keep the exported roster stable.
	roster := make([]obj, 0, len(players))
	for _, player := range players {
		roster = append(roster, player)
	}
	sort.Slice(roster, func(i, j int) bool { return roster[i]["result_id"].(uint32) < roster[j]["result_id"].(uint32) })
	boosterItems := 0
	if len(state.boosterSnapshots) > 0 {
		boosterItems = len(state.boosterSnapshots[0]["item_counters"].(obj))
	}

	return obj{
		"decoder":        "empirical-ruby-apple-26.8",
		"client_version": clientVersion,
		"summary": obj{
			"packet_layout_decoding_supported":           true,
			"arena_context_packets":                      contextPackets,
			"arena_context_player_ids":                   contextIDs,
			"booster_counter_snapshots":                  len(state.boosterSnapshots),
			"booster_item_counters":                      boosterItems,
			"method_48_task_markers":                     len(state.method48Markers),
			"projectile_position_packets":                len(state.shots),
			"shots_in_battle_results":                    expectedShots,
			"shot_count_matches_results":                 shotsMatch,
			"shots_with_launch_metadata":                 linkedShots,
			"hit_messages":                               len(state.hits),
			"hits_received_in_battle_results":            expectedHits,
			"hit_target_matches_entity":                  mappedHits,
			"vehicle_health_updates":                     len(state.healthUpdates),
			"health_updates_linked_to_hit_messages":      linkedHealth,
			"health_updates_with_hp_loss_since_previous": hpLoss,
			"updates_at_zero_hp":                         destroyed,
			"vehicle_transform_updates":                  len(state.vehicleUpdates),
			"vehicle_updates_mapped_to_players":          mappedPlayers,
			"battle_results_snapshot_packets":            state.resultSnapshots,
			"vehicle_packet_type_7":                      countPackets(packets, isUnknownType(7)),
			"unclassified_packet_type_5":                 countPackets(packets, isUnknownType(5)),
			"packet_type_7_opcode_counts":                countObject(state.type7Opcodes),
			"packet_type_7_payload_size_counts":          countObject(state.type7Sizes),
			"packet_type_5_payload_size_counts":          countObject(state.type5Sizes),
		},
		"players_by_result_id":      objects(roster),
		"arena_context":             context,
		"booster_counter_snapshots": objects(state.boosterSnapshots),
		"method_48_task_markers":    objects(state.method48Markers),
		"schema_notes": []any{
			"Подтверждены только указанные варианты длины; остальные payload остаются сырыми.",
			"Subtype 25: projectile_id и 3 координаты; число записей сверяется с n_shots в итогах.",
			"Subtype 35 длиной 49 байт: result_id игрока, projectile_id, marker, origin, velocity и неизвестный float.",
			"Subtype 9 длиной 33 байта: source result_id, target result_id и 13 непрозрачных байт.",
			"Пакет 10 длиной 49 байт: entity_id (часто совпадает с result_id игрока), позиция, ещё один вектор, 3 значения ориентации и флаг; названия части полей предварительные.",
			"Subtype 33 связан с projectile_id из subtype 25; смысл остальных полей не установлен.",
			"Type 7 opcode 3 с 16-битным значением декодирован как оставшееся HP: значения падают вместе с сообщениями о попаданиях, а ноль совпадает с уничтожением техники. Разницы между последовательными обновлениями HP являются наблюдаемой потерей HP; начальное значение до первого обновления в реплее может отсутствовать.",
			"Для остальных opcode типа 7 показаны entity_id, числовой opcode и сырые байты тела; смысл opcode не установлен. Тип 5 показан через маркер и сырые байты.",
			"BasePlayerCreate содержит контекст матча, account_database_ids, player_wait_times и players_battle_categories_ids. Множество account ID совпало с составом результатов в четырёх локальных реплеях, но порядок списка отличается от порядка в результатах.",
			"Метод 51 в изученных реплеях содержит небольшой pickle-словарь с ключами Booster и целыми значениями. Не установлено, означают ли они остаток предметов, выбранное снаряжение или что-то другое; 9-байтовый префикс сохраняется отдельно.",
			"Subtype 48 иногда содержит строковый ключ формата tj_* или clan_task_*; значение показывается буквально, связь entity_id с конкретным игроком не установлена.",
		},
		"shots":              objects(state.shots),
		"hits":               objects(state.hits),
		"health_updates":     objects(state.healthUpdates),
		"vehicle_updates":    objects(state.vehicleUpdates),
		"packet_annotations": annotations,
	}
}

func isUnknownMethod(payload *Payload, subType uint32) bool {
	return payload.Kind == PayloadEntityMethod && payload.EntityMethod.UpdateArena == nil && payload.EntityMethod.SubType == subType
}

func decodeTelemetryPacket(packet *Packet, packetIndex int, players map[uint32]obj, resultsBlob []byte, state *telemetryState) any {
	raw := packet.RawPayload
	clock := jsonx.F64(packet.ClockSecs)
	payload := &packet.Payload
	if isUnknownType(7)(payload) {
		state.type7Sizes[len(raw)]++
		if opcode, ok := readU32(raw, 4); ok {
			state.type7Opcodes[opcode]++
		}
	}
	if isUnknownType(5)(payload) {
		state.type5Sizes[len(raw)]++
	}

	switch {
	case isUnknownMethod(payload, 25) && len(raw) == 28:
		projectileID, ok1 := readU32(raw, 12)
		position, ok2 := readVec3(raw, 16)
		entityID, ok3 := readU32(raw, 0)
		if ok1 && ok2 && ok3 {
			state.shotIndices[projectileID] = append(state.shotIndices[projectileID], len(state.shots))
			state.shots = append(state.shots, obj{
				"clock": clock, "projectile_id": projectileID, "position": position, "entity_id": entityID,
				"packet_index": packetIndex, "launch_records": []any{}, "auxiliary_records": []any{},
			})
			return obj{"kind": "projectile_position", "label": "Точка снаряда", "projectile_id": projectileID, "position": position}
		}
	case isUnknownMethod(payload, 35) && len(raw) == 49:
		sourceResultID, ok1 := readU32(raw, 12)
		projectileID, ok2 := readU32(raw, 16)
		origin, ok3 := readVec3(raw, 21)
		velocity, ok4 := readVec3(raw, 33)
		parameter, ok5 := readF32(raw, 45)
		if ok1 && ok2 && ok3 && ok4 && ok5 {
			launch := obj{
				"clock": clock, "projectile_id": projectileID, "source_result_id": sourceResultID,
				"source_player": playerJSON(players, sourceResultID), "entity_id": optU32(raw, 0), "marker": raw[20],
				"origin": origin, "velocity": velocity, "unknown_scalar": jsonx.F64(parameter), "packet_index": packetIndex,
			}
			state.launchesByProjectile[projectileID] = append(state.launchesByProjectile[projectileID], launch)
			return obj{
				"kind": "projectile_launch", "label": "Запуск снаряда", "projectile_id": projectileID,
				"source_result_id": sourceResultID, "source_player": playerJSON(players, sourceResultID),
				"origin": origin, "velocity": velocity, "marker": raw[20], "unknown_scalar": jsonx.F64(parameter),
			}
		}
	case isUnknownMethod(payload, 9) && len(raw) == 33:
		targetEntity, ok1 := readU32(raw, 0)
		sourceResultID, ok2 := readU32(raw, 12)
		bodyTarget, ok3 := readU32(raw, 16)
		if ok1 && ok2 && ok3 {
			state.hits = append(state.hits, obj{
				"clock": clock, "source_result_id": sourceResultID, "source_player": playerJSON(players, sourceResultID),
				"target_result_id": bodyTarget, "target_player": playerJSON(players, bodyTarget), "entity_id": targetEntity,
				"target_matches_entity_id": targetEntity == bodyTarget, "opaque_hex": hexString(raw[20:]), "packet_index": packetIndex,
			})
			state.hitRecords = append(state.hitRecords, hitRecord{packet.ClockSecs, bodyTarget, sourceResultID, packetIndex})
			return obj{
				"kind": "hit_received", "label": "Сообщение о попадании", "source_result_id": sourceResultID,
				"source_player": playerJSON(players, sourceResultID), "target_result_id": bodyTarget,
				"target_player": playerJSON(players, bodyTarget), "opaque_hex": hexString(raw[20:]),
			}
		}
	case isUnknownMethod(payload, 33) && len(raw) == 46:
		projectileID, ok1 := readU32(raw, 12)
		position, ok2 := readVec3(raw, 21)
		if ok1 && ok2 {
			auxiliary := obj{
				"clock": clock, "projectile_id": projectileID, "position": position,
				"opaque_hex": hexString(raw[20:]), "packet_index": packetIndex,
			}
			state.auxiliariesByProjectile[projectileID] = append(state.auxiliariesByProjectile[projectileID], auxiliary)
			return obj{
				"kind": "projectile_auxiliary", "label": "Дополнительное сообщение снаряда",
				"projectile_id": projectileID, "position": position, "opaque_hex": hexString(raw[20:]),
			}
		}
	case isUnknownMethod(payload, 51):
		entityID, ok1 := readU32(raw, 0)
		counters, ok2 := decodeMethod51Counters(raw)
		if ok1 && ok2 && len(raw) >= 21 {
			snapshotIndex := len(state.boosterSnapshots)
			state.boosterSnapshots = append(state.boosterSnapshots, obj{
				"clock": clock, "entity_id": entityID, "packet_index": packetIndex,
				"opaque_prefix_hex": hexString(raw[12:21]), "item_counters": counters,
			})
			return obj{
				"kind": "method_51_counters", "label": "Счётчики предметов · метод 51", "entity_id": entityID,
				"item_counter_count": len(counters), "snapshot_index": snapshotIndex,
			}
		}
	case isUnknownMethod(payload, 48):
		entityID, ok1 := readU32(raw, 0)
		taskKey, ok2 := decodeMethod48TaskKey(raw)
		if ok1 && ok2 {
			markerIndex := len(state.method48Markers)
			state.method48Markers = append(state.method48Markers, obj{
				"clock": clock, "entity_id": entityID, "task_key": taskKey, "packet_index": packetIndex,
			})
			return obj{
				"kind": "method_48_task_marker", "label": "Ключ задачи · метод 48", "entity_id": entityID,
				"task_key": taskKey, "marker_index": markerIndex,
			}
		}
	case isUnknownType(10)(payload) && len(raw) == 49:
		resultID, ok1 := readU32(raw, 0)
		position, ok2 := readVec3(raw, 12)
		correction, ok3 := readVec3(raw, 24)
		orientation, ok4 := readVec3(raw, 36)
		if ok1 && ok2 && ok3 && ok4 {
			updateIndex := len(state.vehicleUpdates)
			state.vehicleUpdates = append(state.vehicleUpdates, obj{
				"clock": clock, "entity_id": resultID, "player": playerJSON(players, resultID),
				"secondary_word": optU32(raw, 4), "unknown_word": optU32(raw, 8), "position": position,
				"correction_vector": correction, "orientation_values": orientation, "flag": raw[48],
				"packet_index": packetIndex,
			})
			return obj{
				"kind": "vehicle_transform", "label": "Обновление положения техники", "entity_id": resultID,
				"player": playerJSON(players, resultID), "position": position, "update_index": updateIndex,
			}
		}
	case isUnknownType(7)(payload) && len(raw) == 14 && u32Is(raw, 4, 3) && u32Is(raw, 8, 2):
		entityID, ok1 := readU32(raw, 0)
		remaining, ok2 := readU16(raw, 12)
		if ok1 && ok2 {
			previous, hadPrevious := state.previousHealth[entityID]
			state.previousHealth[entityID] = remaining
			var previousHP, change, lost any
			hpLost := int32(0)
			if hadPrevious {
				delta := int32(remaining) - int32(previous)
				hpLost = max(-delta, 0)
				previousHP, change, lost = previous, delta, hpLost
			}
			updateIndex := len(state.healthUpdates)
			state.healthUpdates = append(state.healthUpdates, obj{
				"clock": clock, "entity_id": entityID, "player": playerJSON(players, entityID),
				"remaining_hp": remaining, "previous_hp": previousHP, "health_change": change,
				"hp_lost_since_previous_update": lost, "destroyed_at_zero_hp": remaining == 0,
				"packet_index": packetIndex, "matched_hit_packet_indices": []any{}, "matched_hit_source_result_ids": []any{},
			})
			state.healthRecords = append(state.healthRecords, healthRecord{packet.ClockSecs, entityID, packetIndex, hpLost})
			return obj{
				"kind": "vehicle_health_update", "label": "Обновление оставшегося HP", "entity_id": entityID,
				"player": playerJSON(players, entityID), "remaining_hp": remaining, "previous_hp": previousHP,
				"health_change": change, "hp_lost_since_previous_update": lost, "health_update_index": updateIndex,
			}
		}
	case isUnknownType(7)(payload) && len(raw) >= 12:
		entityID, _ := readU32(raw, 0)
		opcode, _ := readU32(raw, 4)
		bodyWord, _ := readU32(raw, 8)
		var candidate any
		if len(raw) == 16 {
			if value, ok := readF32(raw, 12); ok {
				candidate = jsonx.F64(value)
			}
		}
		return obj{
			"kind": "unclassified_entity_update", "label": fmt.Sprintf("Сообщение типа 7 · код %d", opcode),
			"entity_id": entityID, "opcode": opcode, "body_word": bodyWord, "tail_hex": hexString(raw[12:]),
			"candidate_tail_float": candidate,
		}
	case isUnknownType(5)(payload) && len(raw) >= 5:
		return obj{
			"kind": "unclassified_entity_message", "label": fmt.Sprintf("Сообщение типа 5 · маркер %d", raw[4]),
			"first_word": optU32(raw, 0), "marker": raw[4], "tail_hex": hexString(raw[5:]),
		}
	case isUnknownType(13)(payload):
		if snapshot := findResultsSnapshot(raw, resultsBlob); snapshot != nil {
			state.resultSnapshots++
			return obj{"kind": "battle_results_snapshot", "label": "Снимок итогов боя", "snapshot": snapshot}
		}
	}
	return nil
}

func u32Is(raw []byte, offset int, expected uint32) bool {
	value, ok := readU32(raw, offset)
	return ok && value == expected
}

// TelemetryDecoder produces live, packet-local annotations. Linking records
// and verifying final battle results happen only in offline analysis.
type TelemetryDecoder struct {
	state    *telemetryState
	counters BattleCounterDecoder
}

// NewTelemetryDecoder returns a decoder for a new session.
func NewTelemetryDecoder() *TelemetryDecoder {
	return &TelemetryDecoder{state: newTelemetryState()}
}

// OnPacket returns the telemetry for one packet, or nil for client versions
// with neither supported layouts nor battle counters.
func (d *TelemetryDecoder) OnPacket(packet *Packet, packetIndex int, clientVersion string) any {
	layoutSupported := clientVersion == recoveredLayoutVersion
	counterSupported := SupportsBattleCounters(clientVersion)
	if !layoutSupported && !counterSupported {
		return nil
	}
	update := d.counters.OnPacket(packet, packetIndex, clientVersion)
	var annotation any
	updates := []any{}
	switch {
	case update != nil:
		value := jsonx.ToValue(update)
		annotation = obj{"kind": "battle_counter_update", "update": value}
		updates = append(updates, value)
	case layoutSupported:
		annotation = decodeTelemetryPacket(packet, packetIndex, map[uint32]obj{}, nil, d.state)
	}
	details := obj{
		"packet_layout_decoding_supported":  layoutSupported,
		"battle_counter_decoding_supported": counterSupported,
		"battle_counters":                   jsonx.ToValue(d.counters.Counters()),
		"battle_counter_updates":            updates,
		"annotation":                        annotation,
		"shots":                             objects(d.state.shots),
		"hits":                              objects(d.state.hits),
		"health_updates":                    objects(d.state.healthUpdates),
		"vehicle_updates":                   objects(d.state.vehicleUpdates),
		"booster_counter_snapshots":         objects(d.state.boosterSnapshots),
		"method_48_task_markers":            objects(d.state.method48Markers),
	}
	// Keep only state needed by subsequent packets, never the full live history.
	previous := d.state.previousHealth
	d.state = newTelemetryState()
	d.state.previousHealth = previous
	return details
}

func playerLookup(results *BattleResults) map[uint32]obj {
	names := map[uint32]string{}
	for _, player := range results.Players {
		names[player.AccountID] = player.Info.Nickname
	}
	out := map[uint32]obj{}
	for _, row := range results.PlayerResults {
		var nickname any
		if name, ok := names[row.Info.AccountID]; ok {
			nickname = name
		}
		out[row.ResultID] = obj{
			"result_id": row.ResultID, "account_id": row.Info.AccountID, "nickname": nickname,
			"tank_id": row.Info.TankID, "n_shots": row.Info.NShots, "n_hits_received": row.Info.NHitsReceived,
		}
	}
	return out
}

func decodeMethod48TaskKey(raw []byte) (string, bool) {
	if !u32Is(raw, 4, 48) || len(raw) < 12 {
		return "", false
	}
	body := raw[12:]
	if length, ok := readU32(raw, 8); !ok || int(length) != len(body) {
		return "", false
	}
	if len(body) == 0 {
		return "", false
	}
	length := int(body[0])
	if length == 0 || length > 64 || len(body) != length+1 {
		return "", false
	}
	value := body[1:]
	if !utf8.Valid(value) {
		return "", false
	}
	text := string(value)
	if !strings.HasPrefix(text, "tj_") && !strings.HasPrefix(text, "clan_task_") {
		return "", false
	}
	for _, b := range value {
		if !(b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_') {
			return "", false
		}
	}
	return text, true
}

// decodeMethod51Counters decodes only the bounded primitive dictionary inside
// the observed method-51 envelope. The pickle decoder returns builtin values
// and never imports or runs Python globals.
func decodeMethod51Counters(raw []byte) (obj, bool) {
	if len(raw) > 4096 || !u32Is(raw, 4, 51) || len(raw) < 12 {
		return nil, false
	}
	body := raw[12:]
	if length, ok := readU32(raw, 8); !ok || int(length) != len(body) {
		return nil, false
	}
	// This version's method-51 body begins with 9 opaque bytes followed by
	// the replay's one-byte / 0xFF-u16-0x00 length encoding for a pickle.
	if len(body) < 10 {
		return nil, false
	}
	pickleLength, pickleOffset := int(body[9]), 10
	if body[9] == 0xff {
		if len(body) < 13 || body[12] != 0 {
			return nil, false
		}
		pickleLength, pickleOffset = int(binary.LittleEndian.Uint16(body[10:])), 13
	}
	if pickleLength == 0 || pickleLength > 2048 || len(body) < pickleOffset || len(body)-pickleOffset != pickleLength {
		return nil, false
	}
	value, err := pickle.Decode(body[pickleOffset:])
	if err != nil {
		return nil, false
	}
	dict, ok := value.(*pickle.Dict)
	if !ok {
		return nil, false
	}
	counters := obj{}
	for _, entry := range dict.Entries {
		var key string
		switch typed := entry.Key.(type) {
		case string:
			key = typed
		case pickle.Bytes:
			if !utf8.Valid(typed) {
				return nil, false
			}
			key = string(typed)
		default:
			continue
		}
		if !strings.Contains(key, "Booster") {
			continue
		}
		switch count := entry.Value.(type) {
		case int64:
			if count >= 0 {
				counters[key] = uint64(count)
			}
		case *big.Int:
			if count.Sign() >= 0 && count.IsUint64() {
				counters[key] = count.Uint64()
			}
		}
	}
	return counters, true
}

// findResultsSnapshot finds the final-results protobuf inside a compressed
// packet without decoding pickle opcodes.
func findResultsSnapshot(raw, resultsBlob []byte) any {
	if len(resultsBlob) == 0 || len(raw) < 2 {
		return nil
	}
	for offset := 0; offset < len(raw)-1; offset++ {
		cmf, flg := raw[offset], raw[offset+1]
		if cmf&0x0f != 8 || (uint16(cmf)*256+uint16(flg))%31 != 0 {
			continue
		}
		reader, err := zlib.NewReader(bytes.NewReader(raw[offset:]))
		if err != nil {
			continue
		}
		inflated, err := io.ReadAll(io.LimitReader(reader, 2*1024*1024))
		// A truncated stream yields its decoded prefix, like flate2.
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			continue
		}
		if index := bytes.Index(inflated, resultsBlob); index >= 0 {
			return obj{
				"zlib_offset": offset, "inflated_bytes": len(inflated), "results_protobuf_offset": index,
				"protobuf_bytes": len(resultsBlob), "exact_match_to_archive_results": true,
			}
		}
	}
	return nil
}
