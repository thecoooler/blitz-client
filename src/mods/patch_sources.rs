//! Resolve manifest patch sources into self-contained, per-resource operations.
use super::manifest::{Action, ModFile, safe_relative_path};
use super::patch::split_patch;
use crate::{Error, Result};
use std::collections::BTreeSet;
use std::fs;
use std::path::{Path, PathBuf};

pub(super) fn load_patches(
    mod_dir: &Path,
    sources: &[String],
    generated: Option<&str>,
) -> Result<Vec<ModFile>> {
    if sources.is_empty() {
        return Ok(Vec::new());
    }
    let root = mod_dir
        .canonicalize()
        .map_err(|e| crate::io_error("could not resolve mod directory", mod_dir, e))?;
    let mut seen = BTreeSet::new();
    let mut operations = Vec::new();
    for source in sources {
        let relative = if source == "." {
            PathBuf::from(".")
        } else {
            safe_relative_path(source).map_err(|e| Error::new("MANIFEST_INVALID_PATCH_PATH", e))?
        };
        let path = root.join(relative);
        let mut files = Vec::new();
        collect_source(&path, &root, generated.is_some(), &mut files)?;
        files.sort();
        if files.is_empty() {
            return Err(Error::at(
                "PATCH_DIRECTORY_EMPTY",
                "directory contains no .patch or .diff files",
                path,
            ));
        }
        for file in files {
            let canonical = if file == root.join("mod.patch") && generated.is_some() {
                file.clone()
            } else {
                checked_path(&file, &root)?
            };
            if !seen.insert(canonical.clone()) {
                continue;
            }
            let text = if canonical == root.join("mod.patch") && generated.is_some() {
                generated.unwrap().to_owned()
            } else {
                fs::read_to_string(&canonical)
                    .map_err(|e| crate::io_error("could not read UTF-8 patch", &canonical, e))?
            };
            let relative = canonical
                .strip_prefix(&root)
                .unwrap()
                .to_string_lossy()
                .replace('\\', "/");
            let sections = split_patch(&text).map_err(|mut error| {
                error.path = Some(canonical.clone());
                error
            })?;
            for section in sections {
                operations.push(ModFile {
                    name: section.path,
                    action: Action::Patch,
                    find: None,
                    replace: None,
                    patch: Some(relative.clone()),
                    patch_diff: Some(section.diff),
                    ignore_trailing_whitespace: false,
                });
            }
        }
    }
    Ok(operations)
}

fn checked_path(path: &Path, root: &Path) -> Result<PathBuf> {
    let canonical = path
        .canonicalize()
        .map_err(|e| crate::io_error("could not resolve patch source", path, e))?;
    if !canonical.starts_with(root) {
        return Err(Error::at(
            "PATCH_OUTSIDE_MOD_DIR",
            "patch source escapes the mod directory",
            path,
        ));
    }
    Ok(canonical)
}

fn collect_source(
    path: &Path,
    root: &Path,
    generated: bool,
    files: &mut Vec<PathBuf>,
) -> Result<()> {
    if generated && path == root.join("mod.patch") {
        if let Ok(metadata) = fs::symlink_metadata(path) {
            if !metadata.file_type().is_file() {
                return Err(Error::at(
                    "PATCH_INVALID_SOURCE",
                    "generated mod.patch must be a regular file",
                    path,
                ));
            }
        }
        files.push(path.to_path_buf());
        return Ok(());
    }
    let canonical = checked_path(path, root)?;
    if canonical.is_file() {
        files.push(canonical);
    } else if canonical.is_dir() {
        collect_directory(&canonical, root, files)?;
        if generated && canonical == root && !files.contains(&root.join("mod.patch")) {
            files.push(root.join("mod.patch"));
        }
    } else {
        return Err(Error::at(
            "PATCH_INVALID_SOURCE",
            "patch source must be a file or directory",
            path,
        ));
    }
    Ok(())
}

fn collect_directory(dir: &Path, root: &Path, files: &mut Vec<PathBuf>) -> Result<()> {
    for entry in
        fs::read_dir(dir).map_err(|e| crate::io_error("could not scan patch directory", dir, e))?
    {
        let entry =
            entry.map_err(|e| crate::io_error("could not read patch directory entry", dir, e))?;
        let kind = entry
            .file_type()
            .map_err(|e| crate::io_error("could not inspect patch source", &entry.path(), e))?;
        if kind.is_dir() {
            collect_directory(&entry.path(), root, files)?;
        } else if entry
            .path()
            .extension()
            .and_then(|ext| ext.to_str())
            .is_some_and(|ext| {
                ext.eq_ignore_ascii_case("patch") || ext.eq_ignore_ascii_case("diff")
            })
            && entry.path().is_file()
        {
            files.push(checked_path(&entry.path(), root)?);
        }
        // Directory symlinks are not followed, avoiding recursive cycles.
    }
    Ok(())
}
