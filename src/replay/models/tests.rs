use super::{BattleResults, BattleResultsDat, EntityMethod, Payload};

#[test]
fn base_player_context_decodes_pickle_and_preserves_json_fields() {
    let context = serde_json::json!({
        "battleLevel": 10, "battleCategoryId": 1, "mouseEnabled": true,
        "mmType": 0, "camouflageSlot": 2, "accountDatabaseIds": [42],
        "avgMmr": [null, 3500.5]
    });
    let pickle = serde_pickle::to_vec(&context, Default::default()).unwrap();
    for extended in [false, true] {
        let mut bytes = vec![0; 10];
        bytes.extend_from_slice(b"\x04Test");
        bytes.extend_from_slice(&123_u64.to_le_bytes());
        bytes.extend_from_slice(&7_u32.to_le_bytes());
        if extended {
            bytes.push(255);
            bytes.extend_from_slice(&(pickle.len() as u16).to_le_bytes());
            bytes.push(0);
        } else {
            assert!(pickle.len() < 255);
            bytes.push(pickle.len() as u8);
        }
        bytes.extend_from_slice(&pickle);
        let payload = Payload::new(0, &bytes).unwrap();
        let json = serde_json::to_value(payload).unwrap();
        let player = &json["BasePlayerCreate"];
        assert_eq!(player["author_nickname"], "Test");
        assert_eq!(player["arena_unique_id"], 123);
        assert_eq!(player["arena_type_id"], 7);
        assert_eq!(player["arguments"]["battle_level"], 10);
        assert_eq!(player["arguments"]["average_mm_ratings"], context["avgMmr"]);
        assert_eq!(
            player["arguments"]["account_database_ids"],
            serde_json::json!([42])
        );
        assert!(player["arguments"]["turbo_battles_statistics"].is_null());
        // Every incomplete prefix must return an error, including inside pickle.
        for end in 0..bytes.len() {
            assert!(Payload::new(0, &bytes[..end]).is_err(), "prefix {end}");
        }
    }
}

#[test]
fn update_arena_decodes_players_and_skips_unknown_protobuf_fields() {
    // UpdateArena.players.players: nickname=Test, team=1, account=42.
    // Final field 2 is unmodeled and must be skipped by the protobuf codec.
    let protobuf = b"\x0a\x0c\x0a\x0a\x1a\x04Test\x20\x01\x38\x2a\x10\x01";
    for extended in [false, true] {
        let mut bytes = vec![0; 4];
        bytes.extend_from_slice(&47_u32.to_le_bytes());
        bytes.extend_from_slice(&((protobuf.len() + 3) as u32).to_le_bytes());
        bytes.extend_from_slice(&[0xac, 0x02]); // field_number=300 (varint).
        if extended {
            bytes.extend_from_slice(&[255, protobuf.len() as u8, 0, 0]);
        } else {
            bytes.push(protobuf.len() as u8);
        }
        bytes.extend_from_slice(protobuf);
        match EntityMethod::new(&bytes).unwrap() {
            EntityMethod::UpdateArena {
                field_number,
                arguments,
            } => {
                assert_eq!(field_number, 300);
                let players = arguments.players.unwrap().players;
                assert_eq!(players.len(), 1);
                assert_eq!(players[0].nickname, "Test");
                assert_eq!(players[0].team_number, 1);
                assert_eq!(players[0].account_id, 42);
            }
            other => panic!("unexpected method: {other:?}"),
        }
        for end in 0..bytes.len() {
            assert!(EntityMethod::new(&bytes[..end]).is_err(), "prefix {end}");
        }
    }
}

#[test]
fn invalid_lengths_and_varints_return_errors_without_panicking() {
    let mut prefix = vec![0; 4];
    prefix.extend_from_slice(&47_u32.to_le_bytes());
    prefix.extend_from_slice(&0_u32.to_le_bytes());
    for suffix in [
        vec![0, 255, 255, 255, 0],        // Declared 65535 bytes, none available.
        vec![0, 255, 0, 0, 1],            // Invalid extended-length terminator.
        vec![128; 10],                    // Unterminated overflowing varint.
        [vec![255; 9], vec![2]].concat(), // Overflow in the tenth byte.
        vec![0, 1, 255],                  // Invalid protobuf.
    ] {
        let mut bytes = prefix.clone();
        bytes.extend_from_slice(&suffix);
        assert!(EntityMethod::new(&bytes).is_err());
    }
    let mut bytes = vec![0; 4];
    bytes.extend_from_slice(&999_u32.to_le_bytes());
    assert!(matches!(
        EntityMethod::new(&bytes).unwrap(),
        EntityMethod::Unknown { sub_type: 999 }
    ));
    assert!(matches!(
        Payload::new(999, &[]).unwrap(),
        Payload::Unknown { packet_type: 999 }
    ));
}

#[test]
fn battle_results_pickle_decodes_and_rejects_corrupt_protobuf() {
    // Minimal results: map=7, timestamp=123, winner=1, empty author, room=0.
    let protobuf = b"\x08\x07\x10\x7b\x18\x01\x42\x00\x48\x00";
    let pickle = serde_pickle::to_vec(
        &(123_u64, serde_bytes::Bytes::new(protobuf)),
        Default::default(),
    )
    .unwrap();
    let results =
        BattleResults::try_from(BattleResultsDat::from_reader(pickle.as_slice()).unwrap()).unwrap();
    assert_eq!(results.mode_map_id, 7);
    assert_eq!(results.timestamp_secs, 123);
    assert_eq!(results.winner_team_number, Some(1));
    assert!(BattleResultsDat::from_reader(&b"invalid pickle"[..]).is_err());
    assert!(BattleResults::try_from(BattleResultsDat { buffer: vec![255] }).is_err());
}
