//! Incremental framing and polling of an append-only `data.replay` file.
use crate::replay::TelemetryDecoder;
use crate::{InstallerError, Result, io_error};
use serde::Serialize;
use std::fs::{File, Metadata};
use std::io::{Read, Seek, SeekFrom};
use std::path::{Path, PathBuf};
use wotbreplay_parser::models::data::{packet::Packet, payload::Payload};

const MAX_PAYLOAD_BYTES: usize = 16 * 1024 * 1024;

#[derive(Debug, Serialize)]
#[serde(tag = "event", rename_all = "snake_case")]
pub enum ReplayStreamEvent {
    Header {
        client_version: String,
        header_bytes: u64,
    },
    Packet {
        seq: usize,
        offset: u64,
        packet_type: u32,
        #[serde(flatten)]
        packet: Packet,
    },
    Reset {
        reason: String,
    },
}

/// Buffers only an incomplete frame. Call `finish` only for a completed input,
/// never at a temporary EOF of a growing file. Callback errors terminate decoding.
#[derive(Default)]
pub struct ReplayStreamDecoder {
    pending: Vec<u8>,
    client_version: Option<String>,
    offset: u64,
    seq: usize,
}

fn invalid(message: impl Into<String>) -> InstallerError {
    InstallerError::new("REPLAY_DATA_INVALID", message)
}

impl ReplayStreamDecoder {
    pub fn client_version(&self) -> Option<&str> {
        self.client_version.as_deref()
    }

    pub fn feed(
        &mut self,
        bytes: &[u8],
        mut emit: impl FnMut(ReplayStreamEvent) -> Result<()>,
    ) -> Result<()> {
        // Chunk large inputs so an entire replay is not copied into pending.
        for chunk in bytes.chunks(64 * 1024) {
            self.pending.extend_from_slice(chunk);
            let mut consumed = 0;
            let result = (|| {
                if self.client_version.is_none() {
                    if self.pending.len() >= 4 && self.pending[..4] != 0x12345678_u32.to_le_bytes()
                    {
                        return Err(invalid("invalid data.replay magic"));
                    }
                    if self.pending.len() < 13 {
                        return Ok(());
                    }
                    let version_length_offset = 13 + self.pending[12] as usize;
                    let Some(&version_length) = self.pending.get(version_length_offset) else {
                        return Ok(());
                    };
                    let version_start = version_length_offset + 1;
                    let header_length = version_start + version_length as usize + 1;
                    if self.pending.len() < header_length {
                        return Ok(());
                    }
                    let version =
                        std::str::from_utf8(&self.pending[version_start..header_length - 1])
                            .map_err(|error| invalid(format!("invalid client version: {error}")))?
                            .to_owned();
                    self.client_version = Some(version.clone());
                    consumed = header_length;
                    self.offset = header_length as u64;
                    emit(ReplayStreamEvent::Header {
                        client_version: version,
                        header_bytes: self.offset,
                    })?;
                }
                while self.pending.len() - consumed >= 4 {
                    let raw = &self.pending[consumed..];
                    let length = u32::from_le_bytes(raw[..4].try_into().unwrap()) as usize;
                    if length > MAX_PAYLOAD_BYTES {
                        return Err(invalid(format!(
                            "payload length {length} exceeds {MAX_PAYLOAD_BYTES} at offset {}",
                            self.offset
                        )));
                    }
                    let frame_length = 12 + length;
                    if raw.len() < frame_length {
                        break;
                    }
                    let packet_type = u32::from_le_bytes(raw[4..8].try_into().unwrap());
                    let clock_secs = f32::from_le_bytes(raw[8..12].try_into().unwrap());
                    if !clock_secs.is_finite() {
                        return Err(invalid(format!(
                            "non-finite packet clock at offset {}",
                            self.offset
                        )));
                    }
                    let raw_payload = raw[12..frame_length].to_vec();
                    let payload = Payload::new(packet_type, &raw_payload).map_err(|error| {
                        invalid(format!(
                            "cannot decode packet {} at offset {}: {error}",
                            self.seq, self.offset
                        ))
                    })?;
                    let event = ReplayStreamEvent::Packet {
                        seq: self.seq,
                        offset: self.offset,
                        packet_type,
                        packet: Packet {
                            clock_secs,
                            payload,
                            raw_payload,
                        },
                    };
                    consumed += frame_length;
                    self.offset += frame_length as u64;
                    self.seq += 1;
                    emit(event)?;
                }
                Ok(())
            })();
            self.pending.drain(..consumed);
            result?;
        }
        Ok(())
    }

    pub fn finish(&self) -> Result<()> {
        if self.client_version.is_none() || !self.pending.is_empty() {
            return Err(invalid(format!(
                "truncated data.replay at offset {} ({} buffered bytes)",
                self.offset,
                self.pending.len()
            )));
        }
        Ok(())
    }
}

#[derive(Debug, Serialize)]
pub struct ReplayFollowEvent {
    pub session_id: u64,
    #[serde(flatten)]
    pub event: ReplayStreamEvent,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub telemetry: Option<serde_json::Value>,
}

#[cfg(unix)]
type FileIdentity = (u64, u64);
#[cfg(not(unix))]
type FileIdentity = Option<std::time::SystemTime>;

fn file_identity(metadata: &Metadata) -> FileIdentity {
    #[cfg(unix)]
    {
        use std::os::unix::fs::MetadataExt;
        (metadata.dev(), metadata.ino())
    }
    #[cfg(not(unix))]
    {
        metadata.created().ok()
    }
}

/// Poll explicitly so library callers control scheduling, cancellation and output.
/// Missing files are awaited. Replacement or observed truncation starts a new session.
/// Like tail, this expects append-only writes within each session.
pub struct ReplayFollower {
    path: PathBuf,
    file: Option<File>,
    identity: Option<FileIdentity>,
    read_offset: u64,
    anchor: Vec<u8>,
    session_id: u64,
    decoder: ReplayStreamDecoder,
    telemetry: TelemetryDecoder,
}

impl ReplayFollower {
    pub fn new(path: impl AsRef<Path>) -> Self {
        Self {
            path: path.as_ref().to_owned(),
            file: None,
            identity: None,
            read_offset: 0,
            anchor: Vec::new(),
            session_id: 1,
            decoder: ReplayStreamDecoder::default(),
            telemetry: TelemetryDecoder::default(),
        }
    }

    fn reset(
        &mut self,
        reason: &str,
        emit: &mut impl FnMut(ReplayFollowEvent) -> Result<()>,
    ) -> Result<()> {
        self.file = None;
        self.identity = None;
        self.read_offset = 0;
        self.anchor.clear();
        self.session_id += 1;
        self.decoder = ReplayStreamDecoder::default();
        self.telemetry = TelemetryDecoder::default();
        emit(ReplayFollowEvent {
            session_id: self.session_id,
            event: ReplayStreamEvent::Reset {
                reason: reason.to_owned(),
            },
            telemetry: None,
        })
    }

    /// Returns the number of newly read bytes. EOF leaves incomplete packets pending.
    pub fn poll(&mut self, mut emit: impl FnMut(ReplayFollowEvent) -> Result<()>) -> Result<u64> {
        let metadata = match std::fs::metadata(&self.path) {
            Ok(metadata) => metadata,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                if self.file.is_some() {
                    self.reset("file_removed", &mut emit)?;
                }
                return Ok(0);
            }
            Err(error) => return Err(io_error("cannot stat live replay", &self.path, error)),
        };
        if !metadata.is_file() {
            return Err(invalid("live replay path is not a regular file"));
        }
        let identity = file_identity(&metadata);
        if self
            .identity
            .as_ref()
            .is_some_and(|previous| *previous != identity)
        {
            self.reset("file_replaced", &mut emit)?;
        } else if metadata.len() < self.read_offset {
            self.reset("file_truncated", &mut emit)?;
        }
        if self.file.is_none() {
            let file = match File::open(&self.path) {
                Ok(file) => file,
                Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(0),
                Err(error) => return Err(io_error("cannot open live replay", &self.path, error)),
            };
            self.identity = Some(file_identity(&file.metadata().map_err(|error| {
                io_error("cannot stat live replay handle", &self.path, error)
            })?));
            self.file = Some(file);
        }
        // Detect a truncate-and-regrow between polls, even if the size has
        // already caught up. Checking the last read bytes is cheap and avoids
        // silently combining frames from two battles in the usual rewrite case.
        if !self.anchor.is_empty() {
            let file = self.file.as_mut().unwrap();
            file.seek(SeekFrom::Start(self.read_offset - self.anchor.len() as u64))
                .map_err(|error| io_error("cannot seek replay anchor", &self.path, error))?;
            let mut anchor = vec![0; self.anchor.len()];
            let rewritten = match file.read_exact(&mut anchor) {
                Ok(()) => anchor != self.anchor,
                Err(error) if error.kind() == std::io::ErrorKind::UnexpectedEof => true,
                Err(error) => return Err(io_error("cannot read replay anchor", &self.path, error)),
            };
            if rewritten {
                self.reset("file_rewritten", &mut emit)?;
                // Open again on the next poll, after any concurrent rewrite.
                return Ok(0);
            }
        }
        let file = self.file.as_mut().unwrap();
        let end = file
            .metadata()
            .map_err(|error| io_error("cannot stat live replay handle", &self.path, error))?
            .len();
        file.seek(SeekFrom::Start(self.read_offset))
            .map_err(|error| io_error("cannot seek live replay", &self.path, error))?;
        let mut remaining = end.saturating_sub(self.read_offset);
        let mut read_bytes = 0;
        let mut chunk = [0; 64 * 1024];
        while remaining > 0 {
            let count = file
                .read(&mut chunk[..remaining.min(64 * 1024) as usize])
                .map_err(|error| io_error("cannot read live replay", &self.path, error))?;
            if count == 0 {
                break;
            }
            self.anchor.extend_from_slice(&chunk[..count]);
            if self.anchor.len() > 64 {
                self.anchor.drain(..self.anchor.len() - 64);
            }
            self.read_offset += count as u64;
            remaining -= count as u64;
            read_bytes += count as u64;
            let telemetry = &mut self.telemetry;
            let mut version = self.decoder.client_version().unwrap_or_default().to_owned();
            let session_id = self.session_id;
            self.decoder.feed(&chunk[..count], |event| {
                let details = match &event {
                    ReplayStreamEvent::Header { client_version, .. } => {
                        version.clone_from(client_version);
                        None
                    }
                    ReplayStreamEvent::Packet { seq, packet, .. } => {
                        Some(telemetry.on_packet(packet, *seq, &version))
                    }
                    ReplayStreamEvent::Reset { .. } => None,
                };
                emit(ReplayFollowEvent {
                    session_id,
                    event,
                    telemetry: details,
                })
            })?;
        }
        Ok(read_bytes)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{ReplayOptions, analyze_replay};
    use std::io::{Cursor, Write};
    use std::sync::atomic::{AtomicU64, Ordering};
    use wotbreplay_parser::models::data::Data;

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
    fn arbitrary_chunk_boundaries_match_dependency_decoder() {
        let bytes = replay();
        let expected = Data::from_reader(Cursor::new(&bytes)).unwrap();
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
                                assert_eq!(client_version, expected.client_version);
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
            assert_eq!(
                serde_json::to_value(packets).unwrap(),
                serde_json::to_value(&expected.packets).unwrap()
            );
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
                Err(InstallerError::new("STOP", "consumer stopped"))
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
        assert!(
            events[1].telemetry.as_ref().unwrap()["health_updates"][0]["previous_hp"].is_null()
        );
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
        assert!(
            events[1].telemetry.as_ref().unwrap()["health_updates"][0]["previous_hp"].is_null()
        );
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
        assert!(
            standalone.telemetry.as_ref().unwrap()["summary"]["shots_in_battle_results"].is_null()
        );
        assert!(
            standalone.telemetry.as_ref().unwrap()["summary"]["shot_count_matches_results"]
                .is_null()
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
}
