package replay

import (
	"fmt"

	"github.com/thecoooler/blitz-client/internal/jsonx"
)

// This file owns every user-facing string and annotation kind used by replay
// telemetry. Decoding code chooses a kind; this module decides how it is
// presented. Keep the values stable: they are part of the NDJSON contract.

// Annotation kinds. These strings are the telemetry "kind" values.
const (
	annotationProjectilePosition        = "projectile_position"
	annotationProjectileLaunch          = "projectile_launch"
	annotationHitReceived               = "hit_received"
	annotationProjectileAuxiliary       = "projectile_auxiliary"
	annotationMethod51Counters          = "method_51_counters"
	annotationMethod48TaskMarker        = "method_48_task_marker"
	annotationVehicleTransform          = "vehicle_transform"
	annotationVehicleHealthUpdate       = "vehicle_health_update"
	annotationUnclassifiedEntityUpdate  = "unclassified_entity_update"
	annotationUnclassifiedEntityMessage = "unclassified_entity_message"
	annotationBattleResultsSnapshot     = "battle_results_snapshot"
	annotationBattleCounterUpdate       = "battle_counter_update"
)

// Decoder names reported in complete telemetry reports.
const (
	telemetryDecoderEmpirical          = "empirical-ruby-apple-26.8"
	telemetryDecoderUnsupported        = "unsupported-client-version"
	telemetryDecoderBattleCountersOnly = "battle-counters-only"
)

// recoveredLayoutVersion is the client version whose packet layouts are
// verified. It is also the version text reported in telemetry summaries.
const recoveredLayoutVersion = "26.8.0_ruby_apple"

var annotationLabels = map[string]string{
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

// annotationField is one JSON member of an annotation. Annotation objects are
// jsonx.Object maps, so member order is not part of the output.
type annotationField struct {
	key   string
	value any
}

// annotation builds a live packet annotation from its kind and fields. Labels
// are supplied here, never by decoding callers.
func annotation(kind string, fields ...annotationField) obj {
	return annotationWithLabel(kind, annotationLabels[kind], fields...)
}

func annotationWithLabel(kind, label string, fields ...annotationField) obj {
	out := obj{"kind": kind}
	if label != "" {
		out["label"] = label
	}
	for _, field := range fields {
		out[field.key] = field.value
	}
	return out
}

func unclassifiedType7Label(opcode uint32) string {
	return fmt.Sprintf("Сообщение типа 7 · код %d", opcode)
}

func unclassifiedType5Label(marker byte) string {
	return fmt.Sprintf("Сообщение типа 5 · маркер %d", marker)
}

func unsupportedLayoutSchemaNotes() []any {
	return []any{
		"Байтовые смещения подтверждены только для указанной версии клиента; исходные пакеты доступны без этих предположений.",
	}
}

func supportedLayoutSchemaNotes() []any {
	return []any{
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
	}
}

func battleCounterSchemaNote() string {
	return "Метод 14 автора боя: ribbon 5 — накопленный заблокированный урон; ribbon 17 — накопленная помощь по засвету или обездвиживанию. Значения u16 читаются прямо из пакета; отдельного разделения видов помощи нет."
}

// TelemetryUpdate is the typed, packet-local live telemetry behind one NDJSON
// telemetry record. It carries decoded state, not presentation strings;
// OnPacket renders it into the stable JSON shape.
type TelemetryUpdate struct {
	PacketLayoutDecodingSupported  bool
	BattleCounterDecodingSupported bool
	BattleCounters                 BattleCounters
	BattleCounterUpdates           []*BattleCounterUpdate
	// Annotation is nil when the packet produced no live annotation.
	Annotation              map[string]any
	Shots                   []map[string]any
	Hits                    []map[string]any
	HealthUpdates           []map[string]any
	VehicleUpdates          []map[string]any
	BoosterCounterSnapshots []map[string]any
	Method48TaskMarkers     []map[string]any
}

// details renders the live telemetry object for one packet. The shape and
// values match the historical OnPacket output.
func (u *TelemetryUpdate) details() obj {
	updates := make([]any, len(u.BattleCounterUpdates))
	for index, update := range u.BattleCounterUpdates {
		updates[index] = jsonx.ToValue(update)
	}
	var packetAnnotation any
	if u.Annotation != nil {
		packetAnnotation = u.Annotation
	}
	return obj{
		"packet_layout_decoding_supported":  u.PacketLayoutDecodingSupported,
		"battle_counter_decoding_supported": u.BattleCounterDecodingSupported,
		"battle_counters":                   jsonx.ToValue(u.BattleCounters),
		"battle_counter_updates":            updates,
		"annotation":                        packetAnnotation,
		"shots":                             objects(u.Shots),
		"hits":                              objects(u.Hits),
		"health_updates":                    objects(u.HealthUpdates),
		"vehicle_updates":                   objects(u.VehicleUpdates),
		"booster_counter_snapshots":         objects(u.BoosterCounterSnapshots),
		"method_48_task_markers":            objects(u.Method48TaskMarkers),
	}
}
