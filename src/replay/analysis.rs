use super::stream::{ReplayStreamDecoder, ReplayStreamEvent};
use super::telemetry::recover_telemetry;
use crate::{Error, Result, io_error};
use serde::Serialize;
use serde_json::Value as Json;
use std::env;
use std::ffi::OsStr;
use std::fs::{self, File};
use std::io::{Cursor, Read};
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};
use wotbreplay_parser::models::battle_results::BattleResults;
use wotbreplay_parser::models::battle_results_dat::BattleResultsDat;
use wotbreplay_parser::models::data::Data;
use zip::ZipArchive;

const META_ENTRY: &str = "meta.json";
const RESULTS_ENTRY: &str = "battle_results.dat";
const DATA_ENTRY: &str = "data.replay";
const REPLAY_EXTENSION: &str = "tbreplay";

#[derive(Debug, Clone, Copy, Default)]
pub struct ReplayOptions {
    /// Decode the `data.replay` packet stream and add the recovered telemetry.
    pub include_data: bool,
}

impl ReplayOptions {
    pub fn new() -> Self {
        Self::default()
    }
}

/// Decoded `.tbreplay` archive or standalone `data.replay` (metadata/results are null).
/// JSON field names match the recovered schema of
/// the original replay CLI so existing tooling keeps working.
#[derive(Debug, Clone, Serialize)]
pub struct ReplayAnalysis {
    pub source: String,
    pub meta: Json,
    pub battle_results: Json,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub telemetry: Option<Json>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<Json>,
}

/// Read a replay archive: `meta.json`, the pickled protobuf battle results and,
/// when requested, the framed packet stream with the empirically decoded
/// telemetry. Packet payloads the recovered layout does not cover stay raw.
pub fn analyze_replay(path: impl AsRef<Path>, options: ReplayOptions) -> Result<ReplayAnalysis> {
    let path = path.as_ref();
    let mut file = File::open(path).map_err(|error| io_error("cannot open replay", path, error))?;
    let mut magic = [0; 4];
    let count = file
        .read(&mut magic)
        .map_err(|error| io_error("cannot read replay", path, error))?;
    use std::io::{Seek, SeekFrom};
    file.seek(SeekFrom::Start(0))
        .map_err(|error| io_error("cannot rewind replay", path, error))?;
    if (count == 4 && magic == 0x12345678_u32.to_le_bytes())
        || path.extension() == Some(OsStr::new("replay"))
    {
        return analyze_data_replay(path, file);
    }
    let mut archive = ZipArchive::new(file).map_err(|error| {
        Error::at(
            "REPLAY_ARCHIVE_INVALID",
            format!("not a readable ZIP replay archive: {error}"),
            path,
        )
    })?;

    let meta_bytes = read_entry(&mut archive, path, META_ENTRY)?;
    let meta = serde_json::from_slice(&meta_bytes).map_err(|error| {
        Error::at(
            "REPLAY_META_INVALID",
            format!("cannot decode {META_ENTRY}: {error}"),
            path,
        )
    })?;

    let results_bytes = read_entry(&mut archive, path, RESULTS_ENTRY)?;
    let raw_results =
        BattleResultsDat::from_reader(Cursor::new(&results_bytes[..])).map_err(|error| {
            Error::at(
                "REPLAY_RESULTS_INVALID",
                format!("cannot decode the {RESULTS_ENTRY} pickle: {error}"),
                path,
            )
        })?;
    let results_blob = raw_results.buffer.to_vec();
    let battle_results: BattleResults = raw_results.try_into().map_err(|error| {
        Error::at(
            "REPLAY_RESULTS_INVALID",
            format!("cannot decode the battle results protobuf: {error}"),
            path,
        )
    })?;

    let mut telemetry = None;
    let mut data = None;
    if options.include_data {
        let data_bytes = read_entry(&mut archive, path, DATA_ENTRY)?;
        let packets = read_data_replay(Cursor::new(&data_bytes), path)?;
        telemetry = Some(recover_telemetry(
            &packets.packets,
            &packets.client_version,
            Some(&battle_results),
            &results_blob,
        ));
        data = Some(serialize(&packets)?);
    }

    Ok(ReplayAnalysis {
        source: path
            .file_name()
            .and_then(OsStr::to_str)
            .unwrap_or("replay.tbreplay")
            .to_owned(),
        meta,
        battle_results: serialize(&battle_results)?,
        telemetry,
        data,
    })
}

fn read_data_replay(mut reader: impl Read, path: &Path) -> Result<Data> {
    let mut decoder = ReplayStreamDecoder::default();
    let mut packets = Vec::new();
    let mut client_version = String::new();
    let mut chunk = [0; 64 * 1024];
    loop {
        let count = reader
            .read(&mut chunk)
            .map_err(|error| io_error("cannot read data.replay", path, error))?;
        if count == 0 {
            break;
        }
        decoder.feed(&chunk[..count], |event| {
            match event {
                ReplayStreamEvent::Header {
                    client_version: version,
                    ..
                } => client_version = version,
                ReplayStreamEvent::Packet { packet, .. } => packets.push(packet),
                ReplayStreamEvent::Reset { .. } => unreachable!("decoder does not emit resets"),
            }
            Ok(())
        })?;
    }
    decoder.finish()?;
    Ok(Data {
        client_version,
        packets,
    })
}

fn analyze_data_replay(path: &Path, file: File) -> Result<ReplayAnalysis> {
    let packets = read_data_replay(file, path)?;
    Ok(ReplayAnalysis {
        source: path
            .file_name()
            .and_then(OsStr::to_str)
            .unwrap_or("data.replay")
            .to_owned(),
        meta: Json::Null,
        battle_results: Json::Null,
        telemetry: Some(recover_telemetry(
            &packets.packets,
            &packets.client_version,
            None,
            &[],
        )),
        data: Some(serialize(&packets)?),
    })
}

/// Newest `.tbreplay` file in an explicit directory, or in the first existing
/// platform default replay folder.
pub fn find_latest_replay(directory: Option<PathBuf>) -> Result<PathBuf> {
    let directory = match directory {
        Some(directory) => directory,
        None => {
            let candidates = default_replay_dirs();
            candidates
                .iter()
                .find(|candidate| candidate.is_dir())
                .cloned()
                .ok_or_else(|| {
                    let checked = candidates
                        .iter()
                        .map(|path| path.display().to_string())
                        .collect::<Vec<_>>()
                        .join(", ");
                    Error::new(
                        "REPLAY_FOLDER_NOT_FOUND",
                        format!("no default replay folder exists; checked: {checked}; pass --dir"),
                    )
                })?
        }
    };

    let entries = fs::read_dir(&directory)
        .map_err(|error| io_error("cannot scan the replay folder", &directory, error))?;
    let mut latest: Option<(SystemTime, PathBuf)> = None;
    for entry in entries {
        let entry = entry
            .map_err(|error| io_error("cannot read a replay folder entry", &directory, error))?;
        let path = entry.path();
        if path.extension() != Some(OsStr::new(REPLAY_EXTENSION)) {
            continue;
        }
        let modified = entry
            .metadata()
            .and_then(|metadata| metadata.modified())
            .unwrap_or(UNIX_EPOCH);
        if latest.as_ref().is_none_or(|(time, _)| modified > *time) {
            latest = Some((modified, path));
        }
    }

    latest.map(|(_, path)| path).ok_or_else(|| {
        Error::at(
            "REPLAY_NOT_FOUND",
            format!("no .{REPLAY_EXTENSION} files found"),
            &directory,
        )
    })
}

fn read_entry(archive: &mut ZipArchive<File>, path: &Path, name: &str) -> Result<Vec<u8>> {
    let mut entry = archive.by_name(name).map_err(|error| {
        Error::at(
            "REPLAY_ENTRY_MISSING",
            format!("replay archive is missing {name} ({error})"),
            path,
        )
    })?;
    let mut bytes = Vec::new();
    entry
        .read_to_end(&mut bytes)
        .map_err(|error| io_error(&format!("cannot read {name}"), path, error))?;
    Ok(bytes)
}

fn serialize(value: &impl Serialize) -> Result<Json> {
    serde_json::to_value(value)
        .map_err(|error| Error::new("JSON_SERIALIZE_FAILED", error.to_string()))
}

fn default_replay_dirs() -> Vec<PathBuf> {
    let mut out = Vec::new();
    #[cfg(target_os = "macos")]
    if let Some(home) = env::var_os("HOME") {
        out.push(
            PathBuf::from(home)
                .join("Library")
                .join("Containers")
                .join("com.tanksblitz")
                .join("Data")
                .join("Documents")
                .join("tb-project")
                .join("replays"),
        );
    }
    out
}
