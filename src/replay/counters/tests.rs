use super::*;
use crate::replay::{EntityMethod, ReplayStreamDecoder, ReplayStreamEvent, TelemetryDecoder};
use std::io::{Cursor, Read};

const VERSION: &str = "26.10.0_ruby_apple";

fn base_player(entity: u32) -> Packet {
    let context = serde_json::json!({
        "battleLevel": 10, "battleCategoryId": 1, "mouseEnabled": true,
        "mmType": 0, "camouflageSlot": 0, "accountDatabaseIds": [42]
    });
    let pickle = serde_pickle::to_vec(&context, Default::default()).unwrap();
    let mut raw = entity.to_le_bytes().to_vec();
    raw.extend_from_slice(&[0; 6]);
    raw.extend_from_slice(b"\x04Test");
    raw.extend_from_slice(&1_u64.to_le_bytes());
    raw.extend_from_slice(&1_u32.to_le_bytes());
    raw.push(255);
    raw.extend_from_slice(&(pickle.len() as u16).to_le_bytes());
    raw.push(0);
    raw.extend_from_slice(&pickle);
    Packet {
        clock_secs: 0.0,
        payload: Payload::new(0, &raw).unwrap(),
        raw_payload: raw,
    }
}

fn ribbon(entity: u32, id: u8, value: u16) -> Packet {
    let mut raw = entity.to_le_bytes().to_vec();
    raw.extend_from_slice(&14_u32.to_le_bytes());
    raw.extend_from_slice(&6_u32.to_le_bytes());
    raw.extend_from_slice(&[id, 0, 0, 0]);
    raw.extend_from_slice(&value.to_le_bytes());
    Packet {
        clock_secs: 1.0,
        payload: Payload::EntityMethod(EntityMethod::Unknown { sub_type: 14 }),
        raw_payload: raw,
    }
}

#[test]
fn uses_reported_totals_and_handles_duplicates_corrections_and_resets() {
    let mut decoder = BattleCounterDecoder::default();
    assert!(
        decoder
            .on_packet(&ribbon(42, 17, 100), 0, VERSION)
            .is_none()
    );
    decoder.on_packet(&base_player(42), 1, VERSION);
    assert!(decoder.counters().damage_assisted.is_none());
    assert!(
        decoder
            .on_packet(&ribbon(99, 17, 999), 2, VERSION)
            .is_none()
    );
    let first = decoder.on_packet(&ribbon(42, 17, 100), 3, VERSION).unwrap();
    assert_eq!(first.value, 100);
    assert!(first.delta.is_none());
    let second = decoder.on_packet(&ribbon(42, 17, 250), 4, VERSION).unwrap();
    assert_eq!(second.previous_value, Some(100));
    assert_eq!(second.delta, Some(150));
    assert_eq!(
        decoder
            .on_packet(&ribbon(42, 17, 250), 5, VERSION)
            .unwrap()
            .delta,
        Some(0)
    );
    assert_eq!(
        decoder
            .on_packet(&ribbon(42, 17, 240), 6, VERSION)
            .unwrap()
            .delta,
        Some(-10)
    );
    decoder.on_packet(&ribbon(42, 5, 540), 7, VERSION);
    assert_eq!(decoder.counters().damage_blocked, Some(540));
    assert_eq!(decoder.counters().damage_assisted, Some(240));
    decoder.on_packet(&base_player(7), 8, VERSION);
    assert_eq!(decoder.counters().player_entity_id, Some(7));
    assert!(decoder.counters().damage_blocked.is_none());
    assert!(decoder.on_packet(&ribbon(42, 5, 100), 9, VERSION).is_none());
    decoder.reset();
    assert!(decoder.counters().player_entity_id.is_none());
}

#[test]
fn rejects_unverified_versions_and_malformed_or_unrelated_packets() {
    let mut decoder = BattleCounterDecoder::default();
    decoder.on_packet(&base_player(42), 0, VERSION);
    for length in 0..18 {
        let mut packet = ribbon(42, 5, 540);
        packet.raw_payload.truncate(length);
        assert!(decoder.on_packet(&packet, 1, VERSION).is_none());
    }
    for (offset, value) in [(4, 13), (8, 5), (12, 1)] {
        let mut packet = ribbon(42, 5, 540);
        packet.raw_payload[offset] = value;
        assert!(decoder.on_packet(&packet, 1, VERSION).is_none());
    }
    let mut packet = ribbon(42, 5, 540);
    packet.raw_payload.push(0);
    assert!(decoder.on_packet(&packet, 1, VERSION).is_none());
    packet = ribbon(42, 5, 540);
    packet.clock_secs = f32::NAN;
    assert!(decoder.on_packet(&packet, 1, VERSION).is_none());
    assert!(
        decoder
            .on_packet(&ribbon(42, 5, 540), 1, "26.11.0_ruby_apple")
            .is_none()
    );
    assert!(decoder.counters().player_entity_id.is_none());
}

#[test]
fn live_26_10_exposes_counters_without_enabling_unverified_legacy_layouts() {
    let mut decoder = TelemetryDecoder::default();
    let context = decoder.on_packet(&base_player(42), 0, VERSION);
    assert_eq!(context["battle_counters"]["player_entity_id"], 42);
    let details = decoder.on_packet(&ribbon(42, 17, 1920), 1, VERSION);
    assert_eq!(details["battle_counters"]["damage_assisted"], 1920);
    assert_eq!(details["battle_counter_updates"][0]["value"], 1920);
    assert_eq!(details["annotation"]["kind"], "battle_counter_update");
    assert_eq!(details["packet_layout_decoding_supported"], false);
    assert_eq!(details["battle_counter_decoding_supported"], true);
    assert!(details["health_updates"].as_array().unwrap().is_empty());
    // Later packets retain totals without emitting the old update again.
    let next = decoder.on_packet(&ribbon(42, 2, 0), 2, VERSION);
    assert_eq!(next["battle_counters"]["damage_assisted"], 1920);
    assert!(
        next["battle_counter_updates"]
            .as_array()
            .unwrap()
            .is_empty()
    );
}

#[test]
fn real_stream_emits_exact_assistance_and_blocked_values_before_battle_end() {
    let mut zip = zip::ZipArchive::new(Cursor::new(include_bytes!(
        "../../../tests/fixtures/sample.tbreplay"
    )))
    .unwrap();
    let mut bytes = Vec::new();
    zip.by_name("data.replay")
        .unwrap()
        .read_to_end(&mut bytes)
        .unwrap();
    let mut stream = ReplayStreamDecoder::default();
    let mut counters = BattleCounterDecoder::default();
    let mut version = String::new();
    let mut updates = Vec::new();
    for chunk in bytes.chunks(127) {
        stream
            .feed(chunk, |event| {
                match event {
                    ReplayStreamEvent::Header { client_version, .. } => version = client_version,
                    ReplayStreamEvent::Packet { seq, packet, .. } => {
                        if let Some(update) = counters.on_packet(&packet, seq, &version) {
                            // No completed archive results are supplied to this decoder.
                            assert!(update.clock < 182.0);
                            updates.push(update);
                        }
                    }
                    _ => unreachable!(),
                }
                Ok(())
            })
            .unwrap();
    }
    stream.finish().unwrap();
    assert_eq!(version, VERSION);
    assert_eq!(
        counters.counters().author_nickname.as_deref(),
        Some("iCooLER")
    );
    assert_eq!(counters.counters().damage_assisted, Some(1920));
    assert_eq!(counters.counters().damage_blocked, Some(540));
    let assisted: Vec<_> = updates
        .iter()
        .filter(|u| u.counter == BattleCounterKind::DamageAssisted)
        .map(|u| u.value)
        .collect();
    assert_eq!(assisted, vec![706, 1083, 1499, 1920]);
    let blocked: Vec<_> = updates
        .iter()
        .filter(|u| u.counter == BattleCounterKind::DamageBlocked)
        .map(|u| u.value)
        .collect();
    assert_eq!(blocked, vec![540]);
}
