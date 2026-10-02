use super::stream::MAX_PAYLOAD_BYTES;
use super::*;
use crate::Error;
use std::fs::File;
use std::io::{Read, Write};
use std::path::PathBuf;
use std::sync::atomic::{AtomicU64, Ordering};

fn header() -> Vec<u8> {
    let mut bytes = 0x12345678_u32.to_le_bytes().to_vec();
    bytes.extend_from_slice(&0_u64.to_le_bytes());
    bytes.extend_from_slice(b"\x04hash");
    let version = b"26.8.0_ruby_apple";
    bytes.push(version.len() as u8);
    bytes.extend_from_slice(version);
    bytes.push(0);
    bytes
}

fn frame(kind: u32, clock: f32, payload: &[u8]) -> Vec<u8> {
    let mut bytes = (payload.len() as u32).to_le_bytes().to_vec();
    bytes.extend_from_slice(&kind.to_le_bytes());
    bytes.extend_from_slice(&clock.to_le_bytes());
    bytes.extend_from_slice(payload);
    bytes
}

fn replay() -> Vec<u8> {
    let mut bytes = header();
    bytes.extend(frame(99, 0.0, &[1, 2, 3]));
    bytes.extend(frame(7, 0.2, &[0; 12]));
    bytes.extend(frame(99, 0.3, &[]));
    bytes
}

struct TempDir(PathBuf);
impl TempDir {
    fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let path = std::env::temp_dir().join(format!(
            "blitz-replay-test-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        std::fs::create_dir(&path).unwrap();
        Self(path)
    }
}
impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

#[test]
fn arbitrary_chunk_boundaries_preserve_expected_packets() {
    let bytes = replay();
    let expected = serde_json::json!([
        {"clock": 0.0, "payload": {"Unknown": {"packet_type": 99}}, "raw_payload": "010203"},
        {"clock": f64::from(0.2_f32), "payload": {"Unknown": {"packet_type": 7}}, "raw_payload": "000000000000000000000000"},
        {"clock": f64::from(0.3_f32), "payload": {"Unknown": {"packet_type": 99}}, "raw_payload": ""},
    ]);
    for chunk_size in 1..=bytes.len() {
        let mut decoder = ReplayStreamDecoder::default();
        let mut packets = Vec::new();
        let mut headers = 0;
        let mut offset = header().len() as u64;
        for chunk in bytes.chunks(chunk_size) {
            decoder
                .feed(chunk, |event| {
                    match event {
                        ReplayStreamEvent::Header {
                            client_version,
                            header_bytes,
                        } => {
                            assert_eq!(client_version, "26.8.0_ruby_apple");
                            assert_eq!(header_bytes, offset);
                            headers += 1;
                        }
                        ReplayStreamEvent::Packet {
                            seq,
                            offset: actual,
                            packet,
                            ..
                        } => {
                            assert_eq!(seq, packets.len());
                            assert_eq!(actual, offset);
                            offset += 12 + packet.raw_payload.len() as u64;
                            packets.push(packet);
                        }
                        _ => panic!("unexpected reset"),
                    }
                    Ok(())
                })
                .unwrap();
        }
        decoder.finish().unwrap();
        assert_eq!(headers, 1);
        assert_eq!(serde_json::to_value(packets).unwrap(), expected);
    }
}

#[test]
fn partial_frames_wait_but_completed_input_rejects_truncation() {
    let bytes = replay();
    for length in 0..bytes.len() {
        let mut decoder = ReplayStreamDecoder::default();
        decoder.feed(&bytes[..length], |_| Ok(())).unwrap();
        let boundary = length == header().len()
            || length == header().len() + 15
            || length == header().len() + 39;
        assert_eq!(decoder.finish().is_ok(), boundary, "prefix {length}");
    }
}

#[test]
fn invalid_headers_lengths_payloads_and_clocks_are_errors() {
    let mut decoder = ReplayStreamDecoder::default();
    assert!(decoder.feed(&[0; 4], |_| Ok(())).is_err());
    let mut invalid_version = header();
    invalid_version[18] = 0xff;
    assert!(
        ReplayStreamDecoder::default()
            .feed(&invalid_version, |_| Ok(()))
            .is_err()
    );
    for malformed in [
        (MAX_PAYLOAD_BYTES as u32 + 1).to_le_bytes().to_vec(),
        frame(99, f32::NAN, &[]),
        frame(8, 0.0, &[]),
    ] {
        let mut decoder = ReplayStreamDecoder::default();
        decoder.feed(&header(), |_| Ok(())).unwrap();
        assert!(decoder.feed(&malformed, |_| Ok(())).is_err());
    }
}

#[test]
fn callback_failure_stops_delivery() {
    let mut calls = 0;
    let error = ReplayStreamDecoder::default()
        .feed(&replay(), |_| {
            calls += 1;
            Err(Error::new("STOP", "consumer stopped"))
        })
        .unwrap_err();
    assert_eq!(calls, 1);
    assert_eq!(error.code, "STOP");
}

#[test]
fn follower_waits_for_creation_and_partial_packets_without_duplicates() {
    let temp = TempDir::new();
    let path = temp.0.join("data.replay");
    let mut follower = ReplayFollower::new(&path);
    let mut events = Vec::new();
    assert_eq!(
        follower
            .poll(|event| {
                events.push(event);
                Ok(())
            })
            .unwrap(),
        0
    );
    let bytes = replay();
    let mut writer = File::create(&path).unwrap();
    for byte in &bytes {
        writer.write_all(&[*byte]).unwrap();
        assert_eq!(
            follower
                .poll(|event| {
                    events.push(event);
                    Ok(())
                })
                .unwrap(),
            1
        );
    }
    assert_eq!(events.len(), 4);
    assert!(events.iter().all(|event| event.session_id == 1));
    assert_eq!(follower.poll(|_| panic!("duplicate event")).unwrap(), 0);
    let json = serde_json::to_value(&events[1]).unwrap();
    assert_eq!(json["event"], "packet");
    assert_eq!(json["seq"], 0);
    assert_eq!(json["clock"], 0.0);
    assert_eq!(json["packet_type"], 99);
}

#[test]
fn follower_resets_on_truncation_replacement_and_removal() {
    let temp = TempDir::new();
    let path = temp.0.join("data.replay");
    std::fs::write(&path, replay()).unwrap();
    let mut follower = ReplayFollower::new(&path);
    follower.poll(|_| Ok(())).unwrap();
    std::fs::write(&path, []).unwrap();
    let mut events = Vec::new();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert!(
        matches!(&events[0].event, ReplayStreamEvent::Reset { reason } if reason == "file_truncated")
    );
    assert_eq!(events[0].session_id, 2);
    std::fs::write(&path, replay()).unwrap();
    events.clear();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert!(matches!(
        events[1].event,
        ReplayStreamEvent::Packet { seq: 0, .. }
    ));
    let replacement = temp.0.join("replacement");
    std::fs::write(&replacement, replay()).unwrap();
    std::fs::rename(&replacement, &path).unwrap();
    events.clear();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert_eq!(events.len(), 5);
    assert_eq!(events[0].session_id, 3);
    assert!(
        matches!(&events[0].event, ReplayStreamEvent::Reset { reason } if reason == "file_replaced")
    );
    std::fs::remove_file(&path).unwrap();
    events.clear();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert_eq!(events.len(), 1);
    follower.poll(|_| panic!("repeated removal")).unwrap();
    std::fs::write(&path, replay()).unwrap();
    events.clear();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert_eq!(events[0].session_id, 4);
}

#[test]
fn follower_preserves_hp_state_and_clears_it_after_rewrite() {
    let temp = TempDir::new();
    let path = temp.0.join("data.replay");
    let health_frame = |hp: u16| {
        let mut payload = 42_u32.to_le_bytes().to_vec();
        payload.extend_from_slice(&3_u32.to_le_bytes());
        payload.extend_from_slice(&2_u32.to_le_bytes());
        payload.extend_from_slice(&hp.to_le_bytes());
        frame(7, 1.0, &payload)
    };
    let mut bytes = header();
    bytes.extend(health_frame(100));
    std::fs::write(&path, &bytes).unwrap();
    let mut follower = ReplayFollower::new(&path);
    let mut events = Vec::new();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert!(events[1].telemetry.as_ref().unwrap()["health_updates"][0]["previous_hp"].is_null());
    let mut writer = std::fs::OpenOptions::new()
        .append(true)
        .open(&path)
        .unwrap();
    writer.write_all(&health_frame(70)).unwrap();
    events.clear();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    let telemetry = events[0].telemetry.as_ref().unwrap();
    assert_eq!(telemetry["health_updates"][0]["previous_hp"], 100);
    assert_eq!(
        telemetry["health_updates"][0]["hp_lost_since_previous_update"],
        30
    );
    assert_eq!(telemetry["health_updates"][0]["packet_index"], 1);
    // Same inode and same length, rewritten between polls.
    let mut rewritten = header();
    rewritten.extend(health_frame(200));
    rewritten.extend(health_frame(180));
    std::fs::write(&path, &rewritten).unwrap();
    events.clear();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert!(
        matches!(&events[0].event, ReplayStreamEvent::Reset { reason } if reason == "file_rewritten")
    );
    events.clear();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert_eq!(events[1].session_id, 2);
    assert!(events[1].telemetry.as_ref().unwrap()["health_updates"][0]["previous_hp"].is_null());
    assert_eq!(
        events[2].telemetry.as_ref().unwrap()["health_updates"][0]["hp_lost_since_previous_update"],
        20
    );
}

#[test]
fn unsupported_versions_keep_packets_without_live_telemetry() {
    let temp = TempDir::new();
    let path = temp.0.join("data.replay");
    let mut bytes = replay();
    bytes[18] = b'X';
    std::fs::write(&path, &bytes).unwrap();
    let mut events = Vec::new();
    ReplayFollower::new(&path)
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert!(events[1].telemetry.as_ref().unwrap().is_null());
    assert!(
        matches!(&events[1].event, ReplayStreamEvent::Packet { packet, .. } if packet.raw_payload == [1, 2, 3])
    );
}

#[test]
fn follower_keeps_battle_counters_between_polls_and_resets_on_truncation() {
    let mut archive = zip::ZipArchive::new(std::io::Cursor::new(include_bytes!(
        "../../tests/fixtures/sample.tbreplay"
    )))
    .unwrap();
    let mut data = Vec::new();
    archive
        .by_name("data.replay")
        .unwrap()
        .read_to_end(&mut data)
        .unwrap();
    let mut base = Vec::new();
    let mut assist = Vec::new();
    let mut blocked = Vec::new();
    let mut offset = 65; // The real fixture's header, verified by its stream decoder.
    while offset + 12 <= data.len() {
        let length = u32::from_le_bytes(data[offset..offset + 4].try_into().unwrap()) as usize;
        let kind = u32::from_le_bytes(data[offset + 4..offset + 8].try_into().unwrap());
        let end = offset + 12 + length;
        let raw = &data[offset + 12..end];
        if kind == 0 {
            base = data[offset..end].to_vec();
        }
        if kind == 8 && raw.len() == 18 && raw[4..8] == 14_u32.to_le_bytes() {
            if raw[12] == 17 && assist.is_empty() {
                assist = data[offset..end].to_vec();
            }
            if raw[12] == 5 {
                blocked = data[offset..end].to_vec();
            }
        }
        offset = end;
    }
    assert!(!base.is_empty() && !assist.is_empty() && !blocked.is_empty());
    let temp = TempDir::new();
    let path = temp.0.join("data.replay");
    let initial = [data[..65].to_vec(), base, assist].concat();
    std::fs::write(&path, &initial).unwrap();
    let mut follower = ReplayFollower::new(&path);
    let mut events = Vec::new();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert_eq!(
        events.last().unwrap().telemetry.as_ref().unwrap()["battle_counters"]["damage_assisted"],
        706
    );
    assert!(
        events.last().unwrap().telemetry.as_ref().unwrap()["battle_counters"]["damage_blocked"]
            .is_null()
    );
    std::fs::OpenOptions::new()
        .append(true)
        .open(&path)
        .unwrap()
        .write_all(&blocked)
        .unwrap();
    events.clear();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    let counters = &events[0].telemetry.as_ref().unwrap()["battle_counters"];
    assert_eq!(counters["damage_assisted"], 706);
    assert_eq!(counters["damage_blocked"], 540);
    std::fs::write(&path, &initial).unwrap();
    events.clear();
    follower
        .poll(|event| {
            events.push(event);
            Ok(())
        })
        .unwrap();
    assert!(
        matches!(&events[0].event, ReplayStreamEvent::Reset { reason } if reason == "file_truncated")
    );
    assert_eq!(events.last().unwrap().session_id, 2);
    let counters = &events.last().unwrap().telemetry.as_ref().unwrap()["battle_counters"];
    assert_eq!(counters["damage_assisted"], 706);
    assert!(counters["damage_blocked"].is_null());
}

#[test]
fn standalone_matches_stream_decoder_and_rejects_truncation() {
    let temp = TempDir::new();
    let path = temp.0.join("data.replay");
    let bytes = replay();
    std::fs::write(&path, &bytes).unwrap();
    let standalone = analyze_replay(&path, ReplayOptions::default()).unwrap();
    assert!(standalone.meta.is_null());
    assert!(standalone.battle_results.is_null());
    assert_eq!(
        standalone.data.as_ref().unwrap()["packets"]
            .as_array()
            .unwrap()
            .len(),
        3
    );
    assert!(standalone.telemetry.as_ref().unwrap()["summary"]["shots_in_battle_results"].is_null());
    assert!(
        standalone.telemetry.as_ref().unwrap()["summary"]["shot_count_matches_results"].is_null()
    );
    // Exercise the same reader used by ZIP entries, with an ordinary complete frame stream.
    let mut decoder = ReplayStreamDecoder::default();
    let mut packets = Vec::new();
    decoder
        .feed(&bytes, |event| {
            if let ReplayStreamEvent::Packet { packet, .. } = event {
                packets.push(packet);
            }
            Ok(())
        })
        .unwrap();
    assert_eq!(
        standalone.data.as_ref().unwrap()["packets"],
        serde_json::to_value(packets).unwrap()
    );
    std::fs::write(&path, &bytes[..bytes.len() - 1]).unwrap();
    assert_eq!(
        analyze_replay(&path, ReplayOptions::default())
            .unwrap_err()
            .code,
        "REPLAY_DATA_INVALID"
    );
}
