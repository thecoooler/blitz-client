use crate::{Error, Result};
use std::path::{Path, PathBuf};

/// Discover manifest paths recursively in stable order, without loading mods.
/// Empty directories return an empty list; malformed mods remain discoverable.
/// Loading a discovered path reports conflicting mod.yaml/mod.yml formats.
pub fn discover_mods(dir: impl AsRef<Path>) -> Result<Vec<PathBuf>> {
    let mut manifests = Vec::new();
    collect_manifests(dir.as_ref(), &mut manifests)?;
    manifests.sort();
    Ok(manifests)
}

fn collect_manifests(dir: &Path, manifests: &mut Vec<PathBuf>) -> Result<()> {
    let io_error = |error: std::io::Error| {
        Error::at(
            "IO_ERROR",
            format!("could not scan mod directory: {error}"),
            dir,
        )
    };
    let mut local = Vec::new();
    for entry in std::fs::read_dir(dir).map_err(io_error)? {
        let entry = entry.map_err(io_error)?;
        let kind = entry.file_type().map_err(io_error)?;
        if kind.is_dir() {
            collect_manifests(&entry.path(), manifests)?;
        } else if matches!(entry.file_name().to_str(), Some("mod.yaml" | "mod.yml"))
            && entry.path().is_file()
        {
            local.push(entry.path());
        }
    }
    local.sort();
    if let Some(manifest) = local.into_iter().next() {
        manifests.push(manifest);
    }
    Ok(())
}
