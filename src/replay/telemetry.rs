use flate2::read::ZlibDecoder;
use serde_json::{Value as Json, json};
use std::collections::HashMap;
use std::io::{Cursor, Read};
use wotbreplay_parser::models::{
    battle_results::BattleResults,
    data::{entity_method::EntityMethod, packet::Packet, payload::Payload},
};
const RECOVERED_LAYOUT_VERSION: &str = "26.8.0_ruby_apple";

/// Decode the packet layouts verified against local replays while keeping every
/// original payload available.
pub(super) fn recover_telemetry(
    packets: &[Packet],
    client_version: &str,
    results: Option<&BattleResults>,
    results_blob: &[u8],
) -> Json {
    let arena_context = packets
        .iter()
        .enumerate()
        .find_map(|(packet_index, packet)| {
            let Payload::BasePlayerCreate {
                author_nickname,
                arena_unique_id,
                arena_type_id,
                arguments,
            } = &packet.payload
            else {
                return None;
            };
            Some(json!({
                "packet_index": packet_index,
                "clock": packet.clock_secs,
                "author_nickname": author_nickname,
                "arena_unique_id": arena_unique_id,
                "arena_type_id": arena_type_id,
                "arguments": arguments,
            }))
        });
    if client_version != RECOVERED_LAYOUT_VERSION {
        return json!({
            "decoder": "unsupported-client-version",
            "client_version": client_version,
            "supported_client_version": RECOVERED_LAYOUT_VERSION,
            "summary": {
                "packet_layout_decoding_supported": false,
                "arena_context_packets": packets.iter().filter(|packet| matches!(packet.payload, Payload::BasePlayerCreate { .. })).count(),
                "arena_context_player_ids": arena_context.as_ref().and_then(|context| context["arguments"]["account_database_ids"].as_array()).map_or(0, |ids| ids.len()),
            },
            "schema_notes": ["Байтовые смещения подтверждены только для указанной версии клиента; исходные пакеты доступны без этих предположений."],
            "shots": [],
            "hits": [],
            "arena_context": arena_context,
            "booster_counter_snapshots": [],
            "method_48_task_markers": [],
            "vehicle_updates": [],
            "packet_annotations": vec![Json::Null; packets.len()],
        });
    }

    let players_by_result_id = results.map(player_lookup).unwrap_or_default();
    let mut packet_annotations = vec![Json::Null; packets.len()];
    let mut state = TelemetryState::default();
    for (packet_index, packet) in packets.iter().enumerate() {
        packet_annotations[packet_index] = decode_telemetry_packet(
            packet,
            packet_index,
            &players_by_result_id,
            results_blob,
            &mut state,
        );
    }
    let TelemetryState {
        mut shots,
        shot_indices,
        launches_by_projectile,
        auxiliaries_by_projectile,
        hits,
        vehicle_updates,
        mut vehicle_health_updates,
        booster_counter_snapshots,
        method_48_task_markers,
        previous_health_by_entity: _,
        result_snapshots,
        type_7_opcodes,
        type_7_sizes,
        type_5_sizes,
    } = state;

    for health_update in &mut vehicle_health_updates {
        let entity_id = health_update["entity_id"].as_u64().unwrap_or_default() as u32;
        let clock = health_update["clock"].as_f64().unwrap_or_default() as f32;
        let matching_hits: Vec<_> = hits
            .iter()
            .filter(|hit| {
                hit["target_result_id"].as_u64() == Some(u64::from(entity_id))
                    && hit["clock"].as_f64().map(|value| (value as f32).to_bits())
                        == Some(clock.to_bits())
            })
            .collect();
        let hit_packet_indices: Vec<_> = matching_hits
            .iter()
            .filter_map(|hit| hit["packet_index"].as_u64())
            .collect();
        let source_result_ids: Vec<_> = matching_hits
            .iter()
            .filter_map(|hit| hit["source_result_id"].as_u64())
            .collect();
        let packet_index = health_update["packet_index"].as_u64().unwrap_or_default() as usize;
        health_update["matched_hit_packet_indices"] = Json::Array(
            hit_packet_indices
                .iter()
                .map(|index| json!(index))
                .collect(),
        );
        health_update["matched_hit_source_result_ids"] =
            Json::Array(source_result_ids.iter().map(|id| json!(id)).collect());
        packet_annotations[packet_index]["matched_hit_packet_indices"] = Json::Array(
            hit_packet_indices
                .iter()
                .map(|index| json!(index))
                .collect(),
        );
        packet_annotations[packet_index]["matched_hit_source_result_ids"] =
            Json::Array(source_result_ids.iter().map(|id| json!(id)).collect());
    }

    let mut linked_shots = 0_usize;
    for (projectile_id, indices) in shot_indices {
        if let Some(launches) = launches_by_projectile.get(&projectile_id) {
            for shot_index in &indices {
                shots[*shot_index]["launch_records"] = Json::Array(launches.clone());
                linked_shots += 1;
            }
        }
        if let Some(auxiliaries) = auxiliaries_by_projectile.get(&projectile_id) {
            for shot_index in indices {
                shots[shot_index]["auxiliary_records"] = Json::Array(auxiliaries.clone());
            }
        }
    }

    let expected_shots: Option<u32> = results.map(|results| {
        results
            .player_results
            .iter()
            .map(|row| row.info.n_shots)
            .sum()
    });
    let expected_hits_received: Option<u32> = results.map(|results| {
        results
            .player_results
            .iter()
            .map(|row| row.info.n_hits_received)
            .sum()
    });
    let mapped_hits = hits
        .iter()
        .filter(|hit| hit["target_matches_entity_id"] == true)
        .count();
    let mapped_transform_players = vehicle_updates
        .iter()
        .filter(|update| !update["player"].is_null())
        .count();
    let health_updates_linked_to_hits = vehicle_health_updates
        .iter()
        .filter(|update| {
            !update["matched_hit_packet_indices"]
                .as_array()
                .unwrap()
                .is_empty()
        })
        .count();
    let hp_loss_updates = vehicle_health_updates
        .iter()
        .filter(|update| {
            update["hp_lost_since_previous_update"]
                .as_i64()
                .unwrap_or_default()
                > 0
        })
        .count();
    let destroyed_vehicle_updates = vehicle_health_updates
        .iter()
        .filter(|update| update["destroyed_at_zero_hp"] == true)
        .count();
    // HashMap iteration order is random, so keep the exported roster stable.
    let mut players = players_by_result_id.values().cloned().collect::<Vec<_>>();
    players.sort_by_key(|player| player["result_id"].as_u64().unwrap_or_default());

    json!({
        "decoder": "empirical-ruby-apple-26.8",
        "client_version": client_version,
        "summary": {
            "packet_layout_decoding_supported": true,
            "arena_context_packets": packets.iter().filter(|packet| matches!(packet.payload, Payload::BasePlayerCreate { .. })).count(),
            "arena_context_player_ids": arena_context.as_ref().and_then(|context| context["arguments"]["account_database_ids"].as_array()).map_or(0, |ids| ids.len()),
            "booster_counter_snapshots": booster_counter_snapshots.len(),
            "booster_item_counters": booster_counter_snapshots.first().and_then(|snapshot| snapshot["item_counters"].as_object()).map_or(0, |counters| counters.len()),
            "method_48_task_markers": method_48_task_markers.len(),
            "projectile_position_packets": shots.len(),
            "shots_in_battle_results": expected_shots,
            "shot_count_matches_results": expected_shots.map(|expected| shots.len() == expected as usize),
            "shots_with_launch_metadata": linked_shots,
            "hit_messages": hits.len(),
            "hits_received_in_battle_results": expected_hits_received,
            "hit_target_matches_entity": mapped_hits,
            "vehicle_health_updates": vehicle_health_updates.len(),
            "health_updates_linked_to_hit_messages": health_updates_linked_to_hits,
            "health_updates_with_hp_loss_since_previous": hp_loss_updates,
            "updates_at_zero_hp": destroyed_vehicle_updates,
            "vehicle_transform_updates": vehicle_updates.len(),
            "vehicle_updates_mapped_to_players": mapped_transform_players,
            "battle_results_snapshot_packets": result_snapshots,
            "vehicle_packet_type_7": packets.iter().filter(|packet| matches!(packet.payload, Payload::Unknown { packet_type: 7 })).count(),
            "unclassified_packet_type_5": packets.iter().filter(|packet| matches!(packet.payload, Payload::Unknown { packet_type: 5 })).count(),
            "packet_type_7_opcode_counts": type_7_opcodes,
            "packet_type_7_payload_size_counts": type_7_sizes,
            "packet_type_5_payload_size_counts": type_5_sizes,
        },
        "players_by_result_id": players,
        "arena_context": arena_context,
        "booster_counter_snapshots": booster_counter_snapshots,
        "method_48_task_markers": method_48_task_markers,
        "schema_notes": [
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
        ],
        "shots": shots,
        "hits": hits,
        "health_updates": vehicle_health_updates,
        "vehicle_updates": vehicle_updates,
        "packet_annotations": packet_annotations,
    })
}

#[derive(Default)]
struct TelemetryState {
    shots: Vec<Json>,
    shot_indices: HashMap<u32, Vec<usize>>,
    launches_by_projectile: HashMap<u32, Vec<Json>>,
    auxiliaries_by_projectile: HashMap<u32, Vec<Json>>,
    hits: Vec<Json>,
    vehicle_updates: Vec<Json>,
    vehicle_health_updates: Vec<Json>,
    booster_counter_snapshots: Vec<Json>,
    method_48_task_markers: Vec<Json>,
    previous_health_by_entity: HashMap<u32, u16>,
    result_snapshots: usize,
    type_7_opcodes: HashMap<u32, usize>,
    type_7_sizes: HashMap<usize, usize>,
    type_5_sizes: HashMap<usize, usize>,
}

fn decode_telemetry_packet(
    packet: &Packet,
    packet_index: usize,
    players_by_result_id: &HashMap<u32, Json>,
    results_blob: &[u8],
    state: &mut TelemetryState,
) -> Json {
    let TelemetryState {
        shots,
        shot_indices,
        launches_by_projectile,
        auxiliaries_by_projectile,
        hits,
        vehicle_updates,
        vehicle_health_updates,
        booster_counter_snapshots,
        method_48_task_markers,
        previous_health_by_entity,
        result_snapshots,
        type_7_opcodes,
        type_7_sizes,
        type_5_sizes,
    } = state;
    let mut annotation = Json::Null;
    let raw = &packet.raw_payload;
    let clock = packet.clock_secs;
    if matches!(&packet.payload, Payload::Unknown { packet_type: 7 }) {
        *type_7_sizes.entry(raw.len()).or_default() += 1;
        if let Some(opcode) = read_u32(raw, 4) {
            *type_7_opcodes.entry(opcode).or_default() += 1;
        }
    }
    if matches!(&packet.payload, Payload::Unknown { packet_type: 5 }) {
        *type_5_sizes.entry(raw.len()).or_default() += 1;
    }

    match &packet.payload {
        Payload::EntityMethod(EntityMethod::Unknown { sub_type: 25 }) if raw.len() == 28 => {
            if let (Some(projectile_id), Some(position), Some(entity_id)) =
                (read_u32(raw, 12), read_vec3(raw, 16), read_u32(raw, 0))
            {
                let shot_index = shots.len();
                shot_indices
                    .entry(projectile_id)
                    .or_default()
                    .push(shot_index);
                shots.push(json!({
                    "clock": clock,
                    "projectile_id": projectile_id,
                    "position": position,
                    "entity_id": entity_id,
                    "packet_index": packet_index,
                    "launch_records": [],
                    "auxiliary_records": [],
                }));
                annotation = json!({
                    "kind": "projectile_position",
                    "label": "Точка снаряда",
                    "projectile_id": projectile_id,
                    "position": position,
                });
            }
        }
        Payload::EntityMethod(EntityMethod::Unknown { sub_type: 35 }) if raw.len() == 49 => {
            if let (
                Some(source_result_id),
                Some(projectile_id),
                Some(origin),
                Some(velocity),
                Some(parameter),
            ) = (
                read_u32(raw, 12),
                read_u32(raw, 16),
                read_vec3(raw, 21),
                read_vec3(raw, 33),
                read_f32(raw, 45),
            ) {
                let launch = json!({
                    "clock": clock,
                    "projectile_id": projectile_id,
                    "source_result_id": source_result_id,
                    "source_player": player_json(&players_by_result_id, source_result_id),
                    "entity_id": read_u32(raw, 0),
                    "marker": raw[20],
                    "origin": origin,
                    "velocity": velocity,
                    "unknown_scalar": parameter,
                    "packet_index": packet_index,
                });
                launches_by_projectile
                    .entry(projectile_id)
                    .or_default()
                    .push(launch);
                annotation = json!({
                    "kind": "projectile_launch",
                    "label": "Запуск снаряда",
                    "projectile_id": projectile_id,
                    "source_result_id": source_result_id,
                    "source_player": player_json(&players_by_result_id, source_result_id),
                    "origin": origin,
                    "velocity": velocity,
                    "marker": raw[20],
                    "unknown_scalar": parameter,
                });
            }
        }
        Payload::EntityMethod(EntityMethod::Unknown { sub_type: 9 }) if raw.len() == 33 => {
            if let (Some(target_result_id), Some(source_result_id), Some(body_target_id)) =
                (read_u32(raw, 0), read_u32(raw, 12), read_u32(raw, 16))
            {
                let hit = json!({
                    "clock": clock,
                    "source_result_id": source_result_id,
                    "source_player": player_json(&players_by_result_id, source_result_id),
                    "target_result_id": body_target_id,
                    "target_player": player_json(&players_by_result_id, body_target_id),
                    "entity_id": target_result_id,
                    "target_matches_entity_id": target_result_id == body_target_id,
                    "opaque_hex": hex(&raw[20..]),
                    "packet_index": packet_index,
                });
                hits.push(hit);
                annotation = json!({
                    "kind": "hit_received",
                    "label": "Сообщение о попадании",
                    "source_result_id": source_result_id,
                    "source_player": player_json(&players_by_result_id, source_result_id),
                    "target_result_id": body_target_id,
                    "target_player": player_json(&players_by_result_id, body_target_id),
                    "opaque_hex": hex(&raw[20..]),
                });
            }
        }
        Payload::EntityMethod(EntityMethod::Unknown { sub_type: 33 }) if raw.len() == 46 => {
            if let (Some(projectile_id), Some(position)) = (read_u32(raw, 12), read_vec3(raw, 21)) {
                let auxiliary = json!({
                    "clock": clock,
                    "projectile_id": projectile_id,
                    "position": position,
                    "opaque_hex": hex(&raw[20..]),
                    "packet_index": packet_index,
                });
                auxiliaries_by_projectile
                    .entry(projectile_id)
                    .or_default()
                    .push(auxiliary);
                annotation = json!({
                    "kind": "projectile_auxiliary",
                    "label": "Дополнительное сообщение снаряда",
                    "projectile_id": projectile_id,
                    "position": position,
                    "opaque_hex": hex(&raw[20..]),
                });
            }
        }
        Payload::EntityMethod(EntityMethod::Unknown { sub_type: 51 }) => {
            if let (Some(entity_id), Some(item_counters), Some(prefix)) = (
                read_u32(raw, 0),
                decode_method_51_counters(raw),
                raw.get(12..21),
            ) {
                let counter_count = item_counters.len();
                let snapshot_index = booster_counter_snapshots.len();
                booster_counter_snapshots.push(json!({
                    "clock": clock,
                    "entity_id": entity_id,
                    "packet_index": packet_index,
                    "opaque_prefix_hex": hex(prefix),
                    "item_counters": item_counters,
                }));
                annotation = json!({
                    "kind": "method_51_counters",
                    "label": "Счётчики предметов · метод 51",
                    "entity_id": entity_id,
                    "item_counter_count": counter_count,
                    "snapshot_index": snapshot_index,
                });
            }
        }
        Payload::EntityMethod(EntityMethod::Unknown { sub_type: 48 }) => {
            if let (Some(entity_id), Some(task_key)) =
                (read_u32(raw, 0), decode_method_48_task_key(raw))
            {
                let marker_index = method_48_task_markers.len();
                method_48_task_markers.push(json!({
                    "clock": clock,
                    "entity_id": entity_id,
                    "task_key": task_key,
                    "packet_index": packet_index,
                }));
                annotation = json!({
                    "kind": "method_48_task_marker",
                    "label": "Ключ задачи · метод 48",
                    "entity_id": entity_id,
                    "task_key": task_key,
                    "marker_index": marker_index,
                });
            }
        }
        Payload::Unknown { packet_type: 10 } if raw.len() == 49 => {
            if let (Some(result_id), Some(position), Some(correction), Some(orientation)) = (
                read_u32(raw, 0),
                read_vec3(raw, 12),
                read_vec3(raw, 24),
                read_vec3(raw, 36),
            ) {
                let update_index = vehicle_updates.len();
                vehicle_updates.push(json!({
                    "clock": clock,
                    "entity_id": result_id,
                    "player": player_json(&players_by_result_id, result_id),
                    "secondary_word": read_u32(raw, 4),
                    "unknown_word": read_u32(raw, 8),
                    "position": position,
                    "correction_vector": correction,
                    "orientation_values": orientation,
                    "flag": raw[48],
                    "packet_index": packet_index,
                }));
                annotation = json!({
                    "kind": "vehicle_transform",
                    "label": "Обновление положения техники",
                    "entity_id": result_id,
                    "player": player_json(&players_by_result_id, result_id),
                    "position": position,
                    "update_index": update_index,
                });
            }
        }
        Payload::Unknown { packet_type: 7 }
            if raw.len() == 14 && read_u32(raw, 4) == Some(3) && read_u32(raw, 8) == Some(2) =>
        {
            if let (Some(entity_id), Some(remaining_hp)) = (read_u32(raw, 0), read_u16(raw, 12)) {
                let previous_hp = previous_health_by_entity.insert(entity_id, remaining_hp);
                let health_change =
                    previous_hp.map(|previous| i32::from(remaining_hp) - i32::from(previous));
                let hp_lost = health_change.map(|change| (-change).max(0));
                let update_index = vehicle_health_updates.len();
                vehicle_health_updates.push(json!({
                    "clock": clock,
                    "entity_id": entity_id,
                    "player": player_json(&players_by_result_id, entity_id),
                    "remaining_hp": remaining_hp,
                    "previous_hp": previous_hp,
                    "health_change": health_change,
                    "hp_lost_since_previous_update": hp_lost,
                    "destroyed_at_zero_hp": remaining_hp == 0,
                    "packet_index": packet_index,
                    "matched_hit_packet_indices": [],
                    "matched_hit_source_result_ids": [],
                }));
                annotation = json!({
                    "kind": "vehicle_health_update",
                    "label": "Обновление оставшегося HP",
                    "entity_id": entity_id,
                    "player": player_json(&players_by_result_id, entity_id),
                    "remaining_hp": remaining_hp,
                    "previous_hp": previous_hp,
                    "health_change": health_change,
                    "hp_lost_since_previous_update": hp_lost,
                    "health_update_index": update_index,
                });
            }
        }
        Payload::Unknown { packet_type: 7 } if raw.len() >= 12 => {
            if let (Some(entity_id), Some(opcode), Some(body_word)) =
                (read_u32(raw, 0), read_u32(raw, 4), read_u32(raw, 8))
            {
                annotation = json!({
                    "kind": "unclassified_entity_update",
                    "label": format!("Сообщение типа 7 · код {opcode}"),
                    "entity_id": entity_id,
                    "opcode": opcode,
                    "body_word": body_word,
                    "tail_hex": hex(&raw[12..]),
                    "candidate_tail_float": if raw.len() == 16 { read_f32(raw, 12) } else { None },
                });
            }
        }
        Payload::Unknown { packet_type: 5 } if raw.len() >= 5 => {
            annotation = json!({
                "kind": "unclassified_entity_message",
                "label": format!("Сообщение типа 5 · маркер {}", raw[4]),
                "first_word": read_u32(raw, 0),
                "marker": raw[4],
                "tail_hex": hex(&raw[5..]),
            });
        }
        Payload::Unknown { packet_type: 13 } => {
            if let Some(snapshot) = find_results_snapshot(raw, results_blob) {
                *result_snapshots += 1;
                annotation = json!({
                    "kind": "battle_results_snapshot",
                    "label": "Снимок итогов боя",
                    "snapshot": snapshot,
                });
            }
        }
        _ => {}
    }
    annotation
}

/// Stateful live annotations. Packet-local records are emitted immediately;
/// offline analysis additionally links records and verifies final battle results.
#[derive(Default)]
pub struct TelemetryDecoder {
    state: TelemetryState,
}

impl TelemetryDecoder {
    pub fn on_packet(
        &mut self,
        packet: &Packet,
        packet_index: usize,
        client_version: &str,
    ) -> Json {
        if client_version != RECOVERED_LAYOUT_VERSION {
            return Json::Null;
        }
        let annotation =
            decode_telemetry_packet(packet, packet_index, &HashMap::new(), &[], &mut self.state);
        let details = json!({
            "annotation": annotation,
            "shots": self.state.shots,
            "hits": self.state.hits,
            "health_updates": self.state.vehicle_health_updates,
            "vehicle_updates": self.state.vehicle_updates,
            "booster_counter_snapshots": self.state.booster_counter_snapshots,
            "method_48_task_markers": self.state.method_48_task_markers,
        });
        // Keep only state needed by subsequent packets, never the full live history.
        let previous_health = std::mem::take(&mut self.state.previous_health_by_entity);
        self.state = TelemetryState::default();
        self.state.previous_health_by_entity = previous_health;
        details
    }
}

fn player_lookup(results: &BattleResults) -> HashMap<u32, Json> {
    let account_names: HashMap<u32, &str> = results
        .players
        .iter()
        .map(|player| (player.account_id, player.info.nickname.as_str()))
        .collect();
    results
        .player_results
        .iter()
        .map(|row| {
            (
                row.result_id,
                json!({
                    "result_id": row.result_id,
                    "account_id": row.info.account_id,
                    "nickname": account_names.get(&row.info.account_id).copied(),
                    "tank_id": row.info.tank_id,
                    "n_shots": row.info.n_shots,
                    "n_hits_received": row.info.n_hits_received,
                }),
            )
        })
        .collect()
}

fn player_json(players: &HashMap<u32, Json>, result_id: u32) -> Json {
    players.get(&result_id).cloned().unwrap_or(Json::Null)
}

fn decode_method_48_task_key(raw: &[u8]) -> Option<String> {
    if read_u32(raw, 4) != Some(48) {
        return None;
    }
    let body = raw.get(12..)?;
    if read_u32(raw, 8)? as usize != body.len() {
        return None;
    }
    let length = *body.first()? as usize;
    if length == 0 || length > 64 || body.len() != length + 1 {
        return None;
    }
    let value = std::str::from_utf8(&body[1..]).ok()?;
    if !(value.starts_with("tj_") || value.starts_with("clan_task_"))
        || !value
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || byte == b'_')
    {
        return None;
    }
    Some(value.to_owned())
}

/// Decode only the bounded primitive dictionary inside the observed method-51
/// envelope. `serde-pickle` returns builtin values and never imports or runs
/// Python globals.
fn decode_method_51_counters(raw: &[u8]) -> Option<serde_json::Map<String, Json>> {
    use serde_pickle::value::{HashableValue, Value};

    if raw.len() > 4096 || read_u32(raw, 4) != Some(51) {
        return None;
    }
    let body = raw.get(12..)?;
    if read_u32(raw, 8)? as usize != body.len() {
        return None;
    }

    // This version's method-51 body begins with 9 opaque bytes followed by the
    // replay's one-byte / 0xFF-u16-0x00 length encoding for a pickle value.
    let marker = *body.get(9)?;
    let (pickle_length, pickle_offset) = if marker == 0xff {
        let length = u16::from_le_bytes(body.get(10..12)?.try_into().ok()?) as usize;
        if body.get(12) != Some(&0) {
            return None;
        }
        (length, 13)
    } else {
        (marker as usize, 10)
    };
    if pickle_length == 0
        || pickle_length > 2048
        || body.len().checked_sub(pickle_offset)? != pickle_length
    {
        return None;
    }

    let value: Value = serde_pickle::de::value_from_slice(
        &body[pickle_offset..],
        serde_pickle::de::DeOptions::new(),
    )
    .ok()?;
    let Value::Dict(entries) = value else {
        return None;
    };
    let mut counters = serde_json::Map::new();
    for (key, value) in entries {
        let key = match key {
            HashableValue::String(value) => value,
            HashableValue::Bytes(value) => String::from_utf8(value).ok()?,
            _ => continue,
        };
        if !key.contains("Booster") {
            continue;
        }
        let count = match value {
            Value::I64(value) => u64::try_from(value).ok(),
            Value::Int(value) => value.to_string().parse::<u64>().ok(),
            _ => None,
        };
        let Some(count) = count else { continue };
        counters.insert(key, json!(count));
    }
    Some(counters)
}

fn read_u32(raw: &[u8], offset: usize) -> Option<u32> {
    Some(u32::from_le_bytes(
        raw.get(offset..offset + 4)?.try_into().ok()?,
    ))
}

fn read_u16(raw: &[u8], offset: usize) -> Option<u16> {
    Some(u16::from_le_bytes(
        raw.get(offset..offset + 2)?.try_into().ok()?,
    ))
}

fn read_f32(raw: &[u8], offset: usize) -> Option<f32> {
    let value = f32::from_le_bytes(raw.get(offset..offset + 4)?.try_into().ok()?);
    value.is_finite().then_some(value)
}

fn read_vec3(raw: &[u8], offset: usize) -> Option<[f32; 3]> {
    Some([
        read_f32(raw, offset)?,
        read_f32(raw, offset + 4)?,
        read_f32(raw, offset + 8)?,
    ])
}

fn hex(bytes: &[u8]) -> String {
    const DIGITS: &[u8; 16] = b"0123456789abcdef";
    let mut output = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        output.push(DIGITS[(byte >> 4) as usize] as char);
        output.push(DIGITS[(byte & 0x0f) as usize] as char);
    }
    output
}

/// Find the final-results protobuf inside a compressed packet without decoding
/// pickle opcodes.
fn find_results_snapshot(raw: &[u8], results_blob: &[u8]) -> Option<Json> {
    if results_blob.is_empty() || raw.len() < 2 {
        return None;
    }
    for offset in 0..raw.len() - 1 {
        let cmf = raw[offset];
        let flg = raw[offset + 1];
        if cmf & 0x0f != 8 || (u16::from(cmf) * 256 + u16::from(flg)) % 31 != 0 {
            continue;
        }
        let decoder = ZlibDecoder::new(Cursor::new(&raw[offset..]));
        let mut inflated = Vec::new();
        if decoder
            .take(2 * 1024 * 1024)
            .read_to_end(&mut inflated)
            .is_err()
        {
            continue;
        }
        if let Some(results_offset) = find_subslice(&inflated, results_blob) {
            return Some(json!({
                "zlib_offset": offset,
                "inflated_bytes": inflated.len(),
                "results_protobuf_offset": results_offset,
                "protobuf_bytes": results_blob.len(),
                "exact_match_to_archive_results": true,
            }));
        }
    }
    None
}

fn find_subslice(haystack: &[u8], needle: &[u8]) -> Option<usize> {
    haystack
        .windows(needle.len())
        .position(|candidate| candidate == needle)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn method_payload(entity_id: u32, sub_type: u32, body: &[u8]) -> Vec<u8> {
        let mut raw = Vec::new();
        raw.extend_from_slice(&entity_id.to_le_bytes());
        raw.extend_from_slice(&sub_type.to_le_bytes());
        raw.extend_from_slice(&(body.len() as u32).to_le_bytes());
        raw.extend_from_slice(body);
        raw
    }

    #[test]
    fn reads_little_endian_fields_only_when_in_range() {
        let raw = [1, 0, 0, 0, 0, 0, 0x80, 0x3f];
        assert_eq!(read_u32(&raw, 0), Some(1));
        assert_eq!(read_f32(&raw, 4), Some(1.0));
        assert_eq!(read_u16(&raw, 0), Some(1));
        assert_eq!(read_u32(&raw, 5), None);
        assert_eq!(read_vec3(&raw, 0), None);
        assert_eq!(read_f32(&[0xff, 0xff, 0xff, 0x7f], 0), None);
    }

    #[test]
    fn decodes_method_48_task_key() {
        let key = b"tj_kill_btls_lvl4_slot2";
        let mut body = vec![key.len() as u8];
        body.extend_from_slice(key);
        let raw = method_payload(7, 48, &body);
        assert_eq!(
            decode_method_48_task_key(&raw).as_deref(),
            Some("tj_kill_btls_lvl4_slot2")
        );

        let mut clan = vec![b"clan_task_kill_at_lvl4_slot2".len() as u8];
        clan.extend_from_slice(b"clan_task_kill_at_lvl4_slot2");
        assert!(
            decode_method_48_task_key(&method_payload(7, 48, &clan))
                .is_some_and(|key| key.starts_with("clan_task_"))
        );

        assert_eq!(
            decode_method_48_task_key(&method_payload(7, 48, b"\x05other")),
            None
        );
        assert_eq!(
            decode_method_48_task_key(&method_payload(7, 47, b"\x03tj_")),
            None
        );
    }

    #[test]
    fn decodes_method_51_booster_counters() {
        let mut counters = std::collections::BTreeMap::new();
        counters.insert("Booster_shells".to_owned(), 12_i64);
        counters.insert("Other".to_owned(), 3_i64);
        let pickle = serde_pickle::to_vec(&counters, serde_pickle::SerOptions::new())
            .expect("pickle encoding");

        let mut body = vec![0_u8; 9];
        body.push(pickle.len() as u8);
        body.extend_from_slice(&pickle);
        let decoded =
            decode_method_51_counters(&method_payload(4, 51, &body)).expect("counter snapshot");
        assert_eq!(decoded.len(), 1);
        assert_eq!(decoded["Booster_shells"], json!(12));

        let mut truncated = body.clone();
        truncated.pop();
        assert!(decode_method_51_counters(&method_payload(4, 51, &truncated)).is_none());
        assert!(decode_method_51_counters(&method_payload(4, 50, &body)).is_none());
    }

    #[test]
    fn hex_encodes_lowercase_pairs() {
        assert_eq!(hex(&[0x00, 0x0f, 0xa5, 0xff]), "000fa5ff");
        assert_eq!(hex(&[]), "");
    }

    #[test]
    fn finds_subslice_positions() {
        assert_eq!(find_subslice(b"abcdef", b"cd"), Some(2));
        assert_eq!(find_subslice(b"abcdef", b"zz"), None);
    }
}
