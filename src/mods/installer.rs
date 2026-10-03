use super::manifest::{Action, ModFile, safe_relative_path};
use super::patch::apply_unified_diff;
use super::text_match::find_matches;
use crate::client::{ResolvedGameDir, resolve_game_dir};
use crate::resources::{ContainerKind, decode_dvpl, encode_dvpl};
use crate::{Error, Result};
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

const BACKUP_SUFFIX: &str = ".tanks-blitz-client-original";
const STATE_DIR: &str = ".tanks-blitz-client";

#[derive(Debug, Clone, Serialize)]
pub struct FileValidation {
    pub path: String,
    pub target: Option<PathBuf>,
    pub valid: bool,
    pub output_size: Option<usize>,
    pub error_code: Option<String>,
    pub message: Option<String>,
}

#[derive(Debug, Clone, Serialize)]
pub struct ValidationReport {
    pub mod_id: String,
    pub mod_name: String,
    pub game_dir: PathBuf,
    pub resource_dir: PathBuf,
    pub valid: bool,
    pub files: Vec<FileValidation>,
}

#[derive(Debug, Clone, Serialize)]
pub struct InstallReport {
    pub mod_id: String,
    pub mod_name: String,
    pub installed: bool,
    pub already_installed: bool,
    pub targets: Vec<PathBuf>,
}

#[derive(Debug, Clone, Serialize)]
pub struct RemovalReport {
    pub mod_id: String,
    pub removed: bool,
    pub restored_files: Vec<PathBuf>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
struct InstallerState {
    #[serde(default)]
    version: u32,
    #[serde(default)]
    mods: Vec<InstalledMod>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct InstalledMod {
    id: String,
    name: String,
    files: Vec<InstalledFile>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct InstalledFile {
    logical_path: String,
    target: PathBuf,
    base_snapshot: PathBuf,
    operation: StoredOperation,
    installed_crc32: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct StoredOperation {
    action: Action,
    find: Option<String>,
    replace: Option<String>,
    patch_diff: Option<String>,
    #[serde(default)]
    ignore_trailing_whitespace: bool,
}

struct PreparedFile {
    logical_path: String,
    target: PathBuf,
    base_snapshot: PathBuf,
    base_bytes: Vec<u8>,
    create_base_snapshot: bool,
    create_adjacent_original: bool,
    operations: Vec<StoredOperation>,
    output: Vec<u8>,
    old_current: Vec<u8>,
}

pub fn validate_mod(
    mod_path: impl AsRef<Path>,
    game_dir: Option<PathBuf>,
) -> Result<ValidationReport> {
    let manifest = super::manifest::load_manifest(mod_path)?;
    let game = resolve_game_dir(game_dir)?;
    let state = load_state(&game.game_dir)?;
    let mut files = Vec::with_capacity(manifest.files.len());
    let mut occupied = BTreeMap::new();

    for entry in &manifest.files {
        let item = match resolve_target(&game, entry) {
            Ok(target) => {
                let key = target.to_string_lossy().to_ascii_lowercase();
                if occupied
                    .insert(key, entry.name.clone())
                    .is_some_and(|name| name != entry.name)
                {
                    Err(Error::new(
                        "MANIFEST_DUPLICATE_TARGET",
                        "multiple entries resolve to the same game file",
                    ))
                } else {
                    (|| {
                        let current = fs::read(&target).map_err(|e| {
                            crate::io_error("could not read game resource", &target, e)
                        })?;
                        let managed = managed_resource(&state, &target)?;
                        let base = if let Some((snapshot, installed_crc32)) = managed {
                            if crate::resources::crc32(&current) != installed_crc32 {
                                return Err(Error::at(
                                    "INSTALLED_FILE_CHANGED",
                                    "managed resource changed outside the installer",
                                    &target,
                                ));
                            }
                            read_base_snapshot(&game, &target, &snapshot)?
                        } else {
                            current
                        };
                        let new_operations = manifest
                            .files
                            .iter()
                            .filter(|file| file.name == entry.name)
                            .map(operation_from_entry)
                            .collect::<Result<Vec<_>>>()?;
                        let mut operations = state
                            .mods
                            .iter()
                            .filter(|item| item.id != manifest.id)
                            .flat_map(|item| {
                                item.files
                                    .iter()
                                    .filter(|file| same_path(&file.target, &target))
                                    .map(|file| (item.id.clone(), file.operation.clone()))
                            })
                            .collect::<Vec<_>>();
                        operations.extend(
                            new_operations
                                .into_iter()
                                .map(|operation| (manifest.id.clone(), operation)),
                        );
                        operations.sort_by(|left, right| left.0.cmp(&right.0));
                        compose_resource(&target, &base, &operations)
                            .map(|output| (target, output.len()))
                    })()
                }
            }
            Err(error) => Err(error),
        };
        match item {
            Ok((target, size)) => files.push(FileValidation {
                path: entry.name.clone(),
                target: Some(target),
                valid: true,
                output_size: Some(size),
                error_code: None,
                message: None,
            }),
            Err(error) => files.push(FileValidation {
                path: entry.name.clone(),
                target: error.path.clone(),
                valid: false,
                output_size: None,
                error_code: Some(error.code),
                message: Some(error.message),
            }),
        }
    }

    let valid = files.iter().all(|item| item.valid);
    Ok(ValidationReport {
        mod_id: manifest.id,
        mod_name: manifest.name,
        game_dir: game.game_dir,
        resource_dir: game.resource_dir,
        valid,
        files,
    })
}

pub fn install_mod(mod_path: impl AsRef<Path>, game_dir: Option<PathBuf>) -> Result<InstallReport> {
    let manifest = super::manifest::load_manifest(mod_path)?;
    let game = resolve_game_dir(game_dir)?;
    let state_path = state_path(&game.game_dir);
    let mut state = load_state(&game.game_dir)?;
    if let Some(installed) = state.mods.iter().find(|item| item.id == manifest.id) {
        for file in &installed.files {
            let expected = resolve_logical_target(&game, &file.logical_path)?;
            if !same_path(&file.target, &expected) {
                return Err(Error::at(
                    "STATE_INVALID_TARGET",
                    "installed state target does not match its logical resource path",
                    &file.target,
                ));
            }
            let current = fs::read(&expected)
                .map_err(|e| crate::io_error("could not read managed game file", &expected, e))?;
            if crate::resources::crc32(&current) != file.installed_crc32 {
                return Err(Error::at(
                    "INSTALLED_FILE_CHANGED",
                    "managed resource changed outside the installer",
                    &expected,
                ));
            }
        }
        return Ok(InstallReport {
            mod_id: manifest.id,
            mod_name: manifest.name,
            installed: false,
            already_installed: true,
            targets: Vec::new(),
        });
    }

    let mut prepared = Vec::with_capacity(manifest.files.len());
    let mut logical_paths = BTreeSet::new();
    let mut targets = BTreeSet::new();
    for entry in &manifest.files {
        if !logical_paths.insert(entry.name.clone()) {
            continue;
        }
        let target = resolve_target(&game, entry)?;
        let key = target.to_string_lossy().to_ascii_lowercase();
        if !targets.insert(key) {
            return Err(Error::at(
                "MANIFEST_DUPLICATE_TARGET",
                "multiple entries resolve to the same game file",
                &target,
            ));
        }
        let old_current = fs::read(&target)
            .map_err(|e| crate::io_error("could not read game resource", &target, e))?;
        let managed = managed_resource(&state, &target)?;
        let (base_snapshot, base_bytes, create_base_snapshot) =
            if let Some((snapshot, crc)) = managed {
                if crate::resources::crc32(&old_current) != crc {
                    return Err(Error::at(
                        "INSTALLED_FILE_CHANGED",
                        "managed resource changed outside the installer",
                        &target,
                    ));
                }
                let base = read_base_snapshot(&game, &target, &snapshot)?;
                (snapshot, base, false)
            } else {
                (
                    base_snapshot_path(&game, &target)?,
                    old_current.clone(),
                    true,
                )
            };
        let adjacent_original = append_suffix(&target, BACKUP_SUFFIX);
        let create_adjacent_original = inspect_adjacent_original(&adjacent_original)?;
        let new_operations = manifest
            .files
            .iter()
            .filter(|file| file.name == entry.name)
            .map(operation_from_entry)
            .collect::<Result<Vec<_>>>()?;
        let mut operations = state
            .mods
            .iter()
            .flat_map(|item| {
                item.files
                    .iter()
                    .filter(|file| same_path(&file.target, &target))
                    .map(|file| (item.id.clone(), file.operation.clone()))
            })
            .collect::<Vec<_>>();
        operations.extend(
            new_operations
                .iter()
                .cloned()
                .map(|operation| (manifest.id.clone(), operation)),
        );
        operations.sort_by(|left, right| left.0.cmp(&right.0));
        let output = compose_resource(&target, &base_bytes, &operations)?;
        prepared.push(PreparedFile {
            logical_path: entry.name.clone(),
            target,
            base_snapshot,
            base_bytes,
            create_base_snapshot,
            create_adjacent_original,
            operations: new_operations,
            output,
            old_current,
        });
    }

    let mut created_snapshots = Vec::new();
    for file in &prepared {
        if file.create_base_snapshot {
            if let Err(error) = create_backup(&file.base_snapshot, &file.base_bytes) {
                for path in &created_snapshots {
                    let _ = fs::remove_file(path);
                }
                return Err(error);
            }
            created_snapshots.push(file.base_snapshot.clone());
        }
        if file.create_adjacent_original {
            let original = append_suffix(&file.target, BACKUP_SUFFIX);
            if let Err(error) = create_backup(&original, &file.base_bytes) {
                for path in &created_snapshots {
                    let _ = fs::remove_file(path);
                }
                return Err(error);
            }
            created_snapshots.push(original);
        }
    }

    let mut committed = Vec::new();
    for file in &prepared {
        if let Err(error) = replace_file(&file.target, &file.output) {
            rollback_targets(&committed);
            for path in &created_snapshots {
                let _ = fs::remove_file(path);
            }
            return Err(error);
        }
        committed.push((file.target.clone(), file.old_current.clone()));
    }

    for existing in &mut state.mods {
        for file in &mut existing.files {
            if let Some(output) = prepared
                .iter()
                .find(|item| same_path(&item.target, &file.target))
                .map(|item| item.output.as_slice())
            {
                file.installed_crc32 = crate::resources::crc32(output);
            }
        }
    }
    let installed_files = prepared
        .iter()
        .flat_map(|file| {
            file.operations.iter().map(|operation| InstalledFile {
                logical_path: file.logical_path.clone(),
                target: file.target.clone(),
                base_snapshot: file.base_snapshot.clone(),
                operation: operation.clone(),
                installed_crc32: crate::resources::crc32(&file.output),
            })
        })
        .collect::<Vec<_>>();
    state.version = 2;
    state.mods.push(InstalledMod {
        id: manifest.id.clone(),
        name: manifest.name.clone(),
        files: installed_files,
    });
    if let Err(error) = save_state(&state_path, &state) {
        rollback_targets(&committed);
        for path in &created_snapshots {
            let _ = fs::remove_file(path);
        }
        return Err(error);
    }

    Ok(InstallReport {
        mod_id: manifest.id,
        mod_name: manifest.name,
        installed: true,
        already_installed: false,
        targets: prepared.into_iter().map(|file| file.target).collect(),
    })
}

pub fn remove_mod(mod_id: &str, game_dir: Option<PathBuf>) -> Result<RemovalReport> {
    let game = resolve_game_dir(game_dir)?;
    let state_path = state_path(&game.game_dir);
    let mut state = load_state(&game.game_dir)?;
    let Some(index) = state.mods.iter().position(|item| item.id == mod_id) else {
        return Err(Error::new(
            "MOD_NOT_INSTALLED",
            format!("mod {mod_id:?} is not installed"),
        ));
    };
    let installed = state.mods[index].clone();
    for file in &installed.files {
        let expected = resolve_logical_target(&game, &file.logical_path)?;
        if !same_path(&file.target, &expected) {
            return Err(Error::at(
                "STATE_INVALID_TARGET",
                "installed state target does not match its logical resource path",
                &file.target,
            ));
        }
        let expected_snapshot = base_snapshot_path(&game, &expected)?;
        if !same_path(&file.base_snapshot, &expected_snapshot) {
            return Err(Error::at(
                "STATE_INVALID_BASE",
                "installed state base snapshot path is invalid",
                &file.base_snapshot,
            ));
        }
        let current = fs::read(&expected)
            .map_err(|e| crate::io_error("could not read installed game file", &expected, e))?;
        if crate::resources::crc32(&current) != file.installed_crc32 {
            return Err(Error::at(
                "INSTALLED_FILE_CHANGED",
                "managed resource changed outside the installer",
                &expected,
            ));
        }
    }

    state.mods.remove(index);
    let mut restore = Vec::new();
    let mut restored_targets = BTreeSet::new();
    for file in &installed.files {
        if !restored_targets.insert(file.target.clone()) {
            continue;
        }
        let base = read_base_snapshot(&game, &file.target, &file.base_snapshot)?;
        let operations = operations_for_target(&state, &file.target);
        let output = if operations.is_empty() {
            base
        } else {
            compose_resource(&file.target, &base, &operations)?
        };
        let current = fs::read(&file.target).map_err(|e| {
            crate::io_error("could not read game file before restore", &file.target, e)
        })?;
        if crate::resources::crc32(&current) != file.installed_crc32 {
            return Err(Error::at(
                "INSTALLED_FILE_CHANGED",
                "managed resource changed outside the installer",
                &file.target,
            ));
        }
        restore.push((
            file.target.clone(),
            file.base_snapshot.clone(),
            output,
            current,
        ));
    }

    for (target, _, output, _) in &restore {
        for mod_item in &mut state.mods {
            for file in &mut mod_item.files {
                if same_path(&file.target, target) {
                    file.installed_crc32 = crate::resources::crc32(output);
                }
            }
        }
    }
    let mut committed = Vec::new();
    for (target, _, bytes, current) in &restore {
        if let Err(error) = replace_file(target, bytes) {
            rollback_targets(&committed);
            return Err(error);
        }
        committed.push((target.clone(), current.clone()));
    }

    state.version = 2;
    if let Err(error) = save_state(&state_path, &state) {
        rollback_targets(&committed);
        return Err(error);
    }

    for (_, snapshot, _, _) in &restore {
        if !state.mods.iter().any(|item| {
            item.files
                .iter()
                .any(|file| same_path(&file.base_snapshot, snapshot))
        }) {
            let _ = fs::remove_file(snapshot);
        }
    }
    Ok(RemovalReport {
        mod_id: installed.id,
        removed: true,
        restored_files: restore
            .into_iter()
            .map(|(target, _, _, _)| target)
            .collect(),
    })
}

fn resolve_target(game: &ResolvedGameDir, entry: &ModFile) -> Result<PathBuf> {
    resolve_logical_target(game, &entry.name)
}

fn resolve_logical_target(game: &ResolvedGameDir, name: &str) -> Result<PathBuf> {
    let relative = safe_relative_path(name).map_err(|e| Error::new("MANIFEST_INVALID_PATH", e))?;
    let logical = game.resource_dir.join(relative);
    let mut dvpl_name = logical.as_os_str().to_os_string();
    let is_explicit_dvpl = name.to_ascii_lowercase().ends_with(".dvpl");
    if !is_explicit_dvpl {
        dvpl_name.push(".dvpl");
    }
    let dvpl_path = PathBuf::from(dvpl_name);
    let selected = if is_explicit_dvpl {
        dvpl_path.clone()
    } else {
        let compressed_exists = dvpl_path.is_file();
        let plain_exists = logical.is_file();
        match (compressed_exists, plain_exists) {
            (true, true) => {
                return Err(Error::at(
                    "TARGET_AMBIGUOUS",
                    "both plain and .dvpl versions of this resource exist",
                    &logical,
                ));
            }
            (true, false) => dvpl_path,
            (false, true) => logical,
            (false, false) => {
                return Err(Error::at(
                    "TARGET_NOT_FOUND",
                    "resource file was not found",
                    logical,
                ));
            }
        }
    };
    if !selected.is_file() {
        return Err(Error::at(
            "TARGET_NOT_FOUND",
            "resource file was not found",
            selected,
        ));
    }
    let root = game.resource_dir.canonicalize().map_err(|e| {
        crate::io_error(
            "could not resolve resource directory",
            &game.resource_dir,
            e,
        )
    })?;
    let resolved = selected
        .canonicalize()
        .map_err(|e| crate::io_error("could not resolve resource file", &selected, e))?;
    if !resolved.starts_with(&root) {
        return Err(Error::at(
            "TARGET_OUTSIDE_GAME_DIR",
            "resolved resource escapes the game directory",
            resolved,
        ));
    }
    Ok(resolved)
}

fn operation_from_entry(entry: &ModFile) -> Result<StoredOperation> {
    let patch_diff = if entry.action == Action::Patch {
        Some(entry.patch_diff.clone().ok_or_else(|| {
            Error::new(
                "MANIFEST_INVALID_ACTION",
                "patch operation has no resolved diff",
            )
        })?)
    } else {
        None
    };
    Ok(StoredOperation {
        action: entry.action,
        find: entry.find.clone(),
        replace: entry.replace.clone(),
        patch_diff,
        ignore_trailing_whitespace: entry.ignore_trailing_whitespace,
    })
}

fn compose_resource(
    target: &Path,
    base: &[u8],
    operations: &[(String, StoredOperation)],
) -> Result<Vec<u8>> {
    let is_dvpl = target
        .to_string_lossy()
        .to_ascii_lowercase()
        .ends_with(".dvpl");
    let (plain, kind) = decode_dvpl(base)?;
    if is_dvpl && kind == ContainerKind::Plain {
        return Err(Error::at(
            "DVPL_MAGIC_INVALID",
            "file has a .dvpl suffix but no valid DVPL footer",
            target,
        ));
    }
    let mut output = plain;
    for (mod_id, operation) in operations {
        let result = if operation.action == Action::Clear {
            Ok(Vec::new())
        } else {
            std::str::from_utf8(&output)
                .map_err(|e| {
                    Error::at(
                        "RESOURCE_NOT_UTF8",
                        format!("resource is not UTF-8 text: {e}"),
                        target,
                    )
                })
                .and_then(|source| apply_operation(source, operation, target))
                .map(String::into_bytes)
        };
        output = result.map_err(|error| {
            if operations.len() > 1 {
                Error::at(
                    "MOD_CONFLICT",
                    format!("could not compose mod {mod_id}: {}", error.message),
                    target,
                )
            } else {
                error
            }
        })?;
    }
    encode_dvpl(&output, kind)
}

fn apply_operation(source: &str, operation: &StoredOperation, target: &Path) -> Result<String> {
    match operation.action {
        Action::Replace | Action::Before | Action::After => {
            let find = operation.find.as_deref().unwrap_or_default();
            let replacement = operation.replace.as_deref().unwrap_or_default();
            if find.is_empty() {
                return Err(Error::at(
                    "REPLACE_EMPTY_FIND",
                    "find must not be empty",
                    target,
                ));
            }
            let matches = find_matches(source, find, operation.ignore_trailing_whitespace);
            if matches.len() != 1 {
                return Err(Error::at(
                    "REPLACE_MATCH_COUNT",
                    format!(
                        "expected one {}match, found {}",
                        if operation.ignore_trailing_whitespace {
                            "normalized "
                        } else {
                            "exact "
                        },
                        matches.len()
                    ),
                    target,
                ));
            }
            let range = &matches[0];
            let matched = &source[range.clone()];
            let replacement = match operation.action {
                Action::Before => format!("{replacement}{matched}"),
                Action::After => format!("{matched}{replacement}"),
                _ => replacement.to_owned(),
            };
            let mut output = source.to_owned();
            output.replace_range(range.clone(), &replacement);
            Ok(output)
        }

        Action::Clear => Ok(String::new()),
        Action::Patch => {
            let diff = operation.patch_diff.as_deref().ok_or_else(|| {
                Error::at(
                    "MANIFEST_INVALID_ACTION",
                    "patch action has no patch content",
                    target,
                )
            })?;
            apply_unified_diff(source, diff).map_err(|mut error| {
                if error.path.is_none() {
                    error.path = Some(target.to_path_buf());
                }
                error
            })
        }
    }
}

fn operations_for_target(state: &InstallerState, target: &Path) -> Vec<(String, StoredOperation)> {
    let mut operations = state
        .mods
        .iter()
        .flat_map(|item| {
            item.files
                .iter()
                .filter(|file| same_path(&file.target, target))
                .map(|file| (item.id.clone(), file.operation.clone()))
        })
        .collect::<Vec<_>>();
    operations.sort_by(|left, right| left.0.cmp(&right.0));
    operations
}

fn managed_resource(state: &InstallerState, target: &Path) -> Result<Option<(PathBuf, u32)>> {
    let mut matching = state
        .mods
        .iter()
        .flat_map(|item| item.files.iter())
        .filter(|file| same_path(&file.target, target));
    let Some(first) = matching.next() else {
        return Ok(None);
    };
    let snapshot = first.base_snapshot.clone();
    let crc = first.installed_crc32;
    if matching
        .any(|file| !same_path(&file.base_snapshot, &snapshot) || file.installed_crc32 != crc)
    {
        return Err(Error::at(
            "STATE_INVALID",
            "mods disagree about the managed resource state",
            target,
        ));
    }
    Ok(Some((snapshot, crc)))
}

fn base_snapshot_path(game: &ResolvedGameDir, target: &Path) -> Result<PathBuf> {
    let root = game.resource_dir.canonicalize().map_err(|e| {
        crate::io_error(
            "could not resolve resource directory",
            &game.resource_dir,
            e,
        )
    })?;
    let relative = target.strip_prefix(&root).map_err(|_| {
        Error::at(
            "TARGET_OUTSIDE_GAME_DIR",
            "resource is outside the game resource directory",
            target,
        )
    })?;
    Ok(game.game_dir.join(STATE_DIR).join("bases").join(relative))
}

fn read_base_snapshot(game: &ResolvedGameDir, target: &Path, snapshot: &Path) -> Result<Vec<u8>> {
    let expected = base_snapshot_path(game, target)?;
    if !same_path(snapshot, &expected) {
        return Err(Error::at(
            "STATE_INVALID_BASE",
            "installed state base snapshot path is invalid",
            snapshot,
        ));
    }
    let metadata = fs::symlink_metadata(snapshot)
        .map_err(|e| crate::io_error("could not inspect base snapshot", snapshot, e))?;
    if !metadata.file_type().is_file() {
        return Err(Error::at(
            "BACKUP_INVALID",
            "base snapshot must be a regular file, not a symlink",
            snapshot,
        ));
    }
    fs::read(snapshot).map_err(|e| crate::io_error("could not read base snapshot", snapshot, e))
}

/// Return true when the adjacent original should be created.
fn inspect_adjacent_original(path: &Path) -> Result<bool> {
    if !path.exists() {
        return Ok(true);
    }
    let metadata = fs::symlink_metadata(path)
        .map_err(|e| crate::io_error("could not inspect adjacent original", path, e))?;
    if !metadata.file_type().is_file() {
        return Err(Error::at(
            "BACKUP_INVALID",
            "adjacent original must be a regular file, not a symlink",
            path,
        ));
    }
    Ok(false)
}

fn append_suffix(path: &Path, suffix: &str) -> PathBuf {
    let mut value = path.as_os_str().to_os_string();
    value.push(suffix);
    PathBuf::from(value)
}

fn same_path(left: &Path, right: &Path) -> bool {
    #[cfg(windows)]
    {
        left.to_string_lossy()
            .eq_ignore_ascii_case(&right.to_string_lossy())
    }
    #[cfg(not(windows))]
    {
        left == right
    }
}

fn state_path(game_dir: &Path) -> PathBuf {
    game_dir.join(STATE_DIR).join("state.json")
}

fn load_state(game_dir: &Path) -> Result<InstallerState> {
    let path = state_path(game_dir);
    if !path.exists() {
        return Ok(InstallerState::default());
    }
    let bytes =
        fs::read(&path).map_err(|e| crate::io_error("could not read installer state", &path, e))?;
    let value: serde_json::Value = serde_json::from_slice(&bytes)
        .map_err(|e| Error::at("STATE_INVALID", e.to_string(), &path))?;
    let version = value
        .get("version")
        .and_then(serde_json::Value::as_u64)
        .unwrap_or_default();
    let has_mods = value
        .get("mods")
        .and_then(serde_json::Value::as_array)
        .is_some_and(|mods| !mods.is_empty());
    if version != 2 {
        if !has_mods && version <= 1 {
            return Ok(InstallerState::default());
        }
        return Err(Error::at(
            "STATE_VERSION_UNSUPPORTED",
            format!(
                "installer state version {version} cannot be recomposed; uninstall its mods with the previous installer version first"
            ),
            path,
        ));
    }
    serde_json::from_value(value).map_err(|e| Error::at("STATE_INVALID", e.to_string(), path))
}

fn save_state(path: &Path, state: &InstallerState) -> Result<()> {
    let parent = path.parent().unwrap_or_else(|| Path::new("."));
    fs::create_dir_all(parent)
        .map_err(|e| crate::io_error("could not create installer state directory", parent, e))?;
    let bytes = serde_json::to_vec_pretty(state)
        .map_err(|e| Error::new("STATE_WRITE_FAILED", e.to_string()))?;
    replace_file(path, &bytes)
}

fn create_backup(path: &Path, bytes: &[u8]) -> Result<()> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|e| crate::io_error("could not create backup directory", parent, e))?;
    }
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path)
        .map_err(|e| {
            crate::io_error(
                "could not create original backup without overwriting an existing file",
                path,
                e,
            )
        })?;
    if let Err(e) = file.write_all(bytes).and_then(|_| file.sync_all()) {
        let _ = fs::remove_file(path);
        return Err(crate::io_error("could not write original backup", path, e));
    }
    Ok(())
}

fn replace_file(path: &Path, bytes: &[u8]) -> Result<()> {
    let parent = path.parent().unwrap_or_else(|| Path::new("."));
    let filename = path.file_name().unwrap_or_default().to_string_lossy();
    let tick = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos();
    let temp = parent.join(format!(".{filename}.bc-{}-{tick}.tmp", std::process::id()));
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&temp)
        .map_err(|e| crate::io_error("could not create temporary file", &temp, e))?;
    if let Err(e) = file.write_all(bytes).and_then(|_| file.sync_all()) {
        let _ = fs::remove_file(&temp);
        return Err(crate::io_error("could not write temporary file", &temp, e));
    }
    drop(file);

    #[cfg(windows)]
    if path.exists() {
        let previous = fs::read(path).map_err(|e| {
            let _ = fs::remove_file(&temp);
            crate::io_error("could not read destination before replacement", path, e)
        })?;
        fs::remove_file(path).map_err(|e| {
            let _ = fs::remove_file(&temp);
            crate::io_error("could not replace destination", path, e)
        })?;
        if let Err(e) = fs::rename(&temp, path) {
            let _ = fs::write(path, previous);
            let _ = fs::remove_file(&temp);
            return Err(crate::io_error(
                "could not move temporary file into place",
                path,
                e,
            ));
        }
        return Ok(());
    }
    fs::rename(&temp, path).map_err(|e| {
        let _ = fs::remove_file(&temp);
        crate::io_error("could not move temporary file into place", path, e)
    })
}

fn rollback_targets(committed: &[(PathBuf, Vec<u8>)]) {
    for (path, bytes) in committed.iter().rev() {
        let _ = replace_file(path, bytes);
    }
}
