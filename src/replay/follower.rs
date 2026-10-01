use super::{
    stream::{ReplayStreamDecoder, ReplayStreamEvent, invalid},
    telemetry::TelemetryDecoder,
};
use crate::{Result, io_error};
use serde::Serialize;
use std::fs::{File, Metadata};
use std::io::{Read, Seek, SeekFrom};
use std::path::{Path, PathBuf};

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
