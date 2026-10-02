use crate::{Error, Result};
use serde::Serialize;
use std::fs;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicUsize, Ordering};
static TEMP_FILE_SEQUENCE: AtomicUsize = AtomicUsize::new(0);

#[derive(Debug, Default)]
pub(super) struct AssetStats {
    pub(super) stored_bytes: usize,
    pub(super) uncompressed_bytes: usize,
}

pub(super) fn write_compressed_asset(
    path: &Path,
    bytes: &[u8],
    compress: bool,
) -> Result<AssetStats> {
    let stored_path = if compress {
        append_suffix(path, ".br")
    } else {
        path.to_path_buf()
    };
    if let Some(parent) = stored_path.parent() {
        fs::create_dir_all(parent)
            .map_err(|error| crate::io_error("could not create asset directory", parent, error))?;
    }
    let stored = if compress {
        brotli_compress(bytes)?
    } else {
        bytes.to_vec()
    };
    atomic_write(&stored_path, &stored)?;
    let stale = if compress {
        path.to_path_buf()
    } else {
        append_suffix(path, ".br")
    };
    let _ = fs::remove_file(stale);
    Ok(AssetStats {
        stored_bytes: stored.len(),
        uncompressed_bytes: bytes.len(),
    })
}

pub(super) fn write_json(path: &Path, value: &impl Serialize, compress: bool) -> Result<()> {
    let bytes = serde_json::to_vec_pretty(value)
        .map_err(|error| Error::new("JSON_SERIALIZE_FAILED", error.to_string()))?;
    write_compressed_asset(path, &bytes, compress).map(|_| ())
}

fn brotli_compress(bytes: &[u8]) -> Result<Vec<u8>> {
    let mut child = Command::new("brotli")
        .args(["-q", "9", "-c"])
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|error| {
            Error::new(
                "BROTLI_NOT_FOUND",
                format!("could not start brotli: {error}; use --no-compress or install Brotli CLI"),
            )
        })?;
    child
        .stdin
        .take()
        .ok_or_else(|| Error::new("BROTLI_FAILED", "could not open brotli stdin"))?
        .write_all(bytes)
        .map_err(|error| Error::new("BROTLI_FAILED", error.to_string()))?;
    let output = child
        .wait_with_output()
        .map_err(|error| Error::new("BROTLI_FAILED", error.to_string()))?;
    if !output.status.success() {
        return Err(Error::new(
            "BROTLI_FAILED",
            String::from_utf8_lossy(&output.stderr).trim().to_owned(),
        ));
    }
    Ok(output.stdout)
}

pub(super) fn atomic_write(path: &Path, bytes: &[u8]) -> Result<()> {
    let sequence = TEMP_FILE_SEQUENCE.fetch_add(1, Ordering::Relaxed);
    let temporary = path.with_file_name(format!(
        ".{}.{}.{}.tmp",
        path.file_name().unwrap_or_default().to_string_lossy(),
        std::process::id(),
        sequence
    ));
    fs::write(&temporary, bytes)
        .map_err(|error| crate::io_error("could not write asset", &temporary, error))?;
    if path.exists() {
        fs::remove_file(path)
            .map_err(|error| crate::io_error("could not replace asset", path, error))?;
    }
    fs::rename(&temporary, path)
        .map_err(|error| crate::io_error("could not replace asset", path, error))
}

pub(super) fn append_suffix(path: &Path, suffix: &str) -> PathBuf {
    let mut result = path.as_os_str().to_os_string();
    result.push(suffix);
    PathBuf::from(result)
}
