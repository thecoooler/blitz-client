//! Incremental framing of `data.replay` bytes.
use crate::{Error, Result};
use serde::Serialize;
use wotbreplay_parser::models::data::{packet::Packet, payload::Payload};

pub(super) const MAX_PAYLOAD_BYTES: usize = 16 * 1024 * 1024;

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

pub(super) fn invalid(message: impl Into<String>) -> Error {
    Error::new("REPLAY_DATA_INVALID", message)
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
