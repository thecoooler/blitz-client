use crate::{Result, resources::decode_dvpl};
use std::fs;
use std::path::{Path, PathBuf};

pub(crate) fn read_client_asset(path: &Path) -> Result<Vec<u8>> {
    let bytes = fs::read(path)
        .map_err(|error| crate::io_error("could not read client resource", path, error))?;
    decode_dvpl(&bytes)
        .map(|(plain, _)| plain)
        .map_err(|mut error| {
            error.path = Some(path.to_path_buf());
            error
        })
}

pub(crate) fn resolve_client_asset(data_dir: &Path, relative: &str) -> Option<PathBuf> {
    if let Some(plain_relative) = relative.strip_suffix(".dvpl") {
        let plain_path = data_dir.join(plain_relative);
        if plain_path.is_file() {
            return Some(plain_path);
        }
        let dvpl_path = data_dir.join(relative);
        if dvpl_path.is_file() {
            return Some(dvpl_path);
        }
    } else {
        let path = data_dir.join(relative);
        if path.is_file() {
            return Some(path);
        }
        let dvpl_path = data_dir.join(format!("{relative}.dvpl"));
        if dvpl_path.is_file() {
            return Some(dvpl_path);
        }
    }
    None
}
