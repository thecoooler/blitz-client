use crate::{InstallerError, Result, git_diff};
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;
use std::fs;
use std::path::{Component, Path, PathBuf};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Manifest {
    pub id: String,
    pub name: String,
    #[serde(default)]
    pub files: Vec<ModFile>,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum Action {
    Replace,
    Patch,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModFile {
    /// Logical path relative to the game's resource root. For a DVPL file, omit `.dvpl`.
    pub name: String,
    pub action: Action,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub find: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub replace: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub patch: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub generated_by: Option<String>,
}

pub(crate) fn locate_mod_dir(path: &Path) -> PathBuf {
    if path.is_file() {
        path.parent()
            .unwrap_or_else(|| Path::new("."))
            .to_path_buf()
    } else {
        path.to_path_buf()
    }
}

pub(crate) fn manifest_file(path: &Path) -> PathBuf {
    if path.is_file() {
        path.to_path_buf()
    } else {
        path.join("mod.toml")
    }
}

pub fn load_manifest(path: impl AsRef<Path>) -> Result<Manifest> {
    let path = path.as_ref();
    let manifest_path = manifest_file(path);
    let text = fs::read_to_string(&manifest_path)
        .map_err(|e| crate::io_error("could not read mod.toml", &manifest_path, e))?;
    let manifest: Manifest = toml::from_str(&text).map_err(|e| {
        InstallerError::at("MANIFEST_INVALID", e.to_string(), manifest_path.clone())
    })?;
    validate_manifest(&manifest, &manifest_path)?;
    Ok(manifest)
}

pub(crate) fn validate_manifest(manifest: &Manifest, path: &Path) -> Result<()> {
    if manifest.id.trim().is_empty()
        || !manifest
            .id
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || c == '-' || c == '_')
    {
        return Err(InstallerError::at(
            "MANIFEST_INVALID_ID",
            "id must contain only ASCII letters, digits, '-' or '_'",
            path.to_path_buf(),
        ));
    }
    if manifest.name.trim().is_empty() {
        return Err(InstallerError::at(
            "MANIFEST_INVALID_NAME",
            "name must not be empty",
            path.to_path_buf(),
        ));
    }
    if manifest.files.is_empty() {
        return Err(InstallerError::at(
            "MANIFEST_EMPTY",
            "mod has no [[files]] entries",
            path.to_path_buf(),
        ));
    }
    let mut seen = BTreeSet::new();
    for file in &manifest.files {
        safe_relative_path(&file.name)
            .map_err(|e| InstallerError::at("MANIFEST_INVALID_PATH", e, path.to_path_buf()))?;
        if !seen.insert(&file.name) {
            return Err(InstallerError::at(
                "MANIFEST_DUPLICATE_TARGET",
                format!("target path is listed more than once: {}", file.name),
                path.to_path_buf(),
            ));
        }
        match file.action {
            Action::Replace
                if file.find.as_deref().is_none_or(str::is_empty) || file.replace.is_none() =>
            {
                return Err(InstallerError::at(
                    "MANIFEST_INVALID_ACTION",
                    format!(
                        "replace action for {} requires non-empty find and present replace",
                        file.name
                    ),
                    path.to_path_buf(),
                ));
            }
            Action::Patch if file.patch.is_none() => {
                return Err(InstallerError::at(
                    "MANIFEST_INVALID_ACTION",
                    format!("patch action for {} requires a patch path", file.name),
                    path.to_path_buf(),
                ));
            }
            _ => {}
        }
        if let Some(patch) = &file.patch {
            safe_relative_path(patch).map_err(|e| {
                InstallerError::at("MANIFEST_INVALID_PATCH_PATH", e, path.to_path_buf())
            })?;
        }
    }
    Ok(())
}

pub(crate) fn safe_relative_path(path: &str) -> std::result::Result<PathBuf, String> {
    if path.is_empty() || path.contains('\0') || path.contains(':') {
        return Err(format!("invalid relative path: {path:?}"));
    }
    let portable = path.replace('\\', "/");
    let parsed = Path::new(&portable);
    if parsed.is_absolute() {
        return Err(format!("absolute paths are not allowed: {path}"));
    }
    let mut result = PathBuf::new();
    for component in parsed.components() {
        match component {
            Component::Normal(part) => result.push(part),
            Component::CurDir => {}
            _ => return Err(format!("path traversal is not allowed: {path}")),
        }
    }
    if result.as_os_str().is_empty() {
        return Err("path must not be empty".to_owned());
    }
    Ok(result)
}

pub fn update_from_git_diff(
    mod_path: impl AsRef<Path>,
    repo: impl AsRef<Path>,
    base_revision: Option<&str>,
) -> Result<Manifest> {
    let mod_dir = locate_mod_dir(mod_path.as_ref());
    let manifest_path = mod_dir.join("mod.toml");
    let raw_text = fs::read_to_string(&manifest_path)
        .map_err(|e| crate::io_error("could not read mod.toml", &manifest_path, e))?;
    let mut manifest: Manifest = toml::from_str(&raw_text)
        .map_err(|e| InstallerError::at("MANIFEST_INVALID", e.to_string(), &manifest_path))?;
    validate_manifest(&manifest, &manifest_path)?;
    let changes = git_diff::collect_changes(repo.as_ref(), base_revision.unwrap_or("HEAD"))?;

    let old_generated = manifest
        .files
        .iter()
        .filter(|file| file.generated_by.as_deref() == Some("git"))
        .filter_map(|file| file.patch.clone())
        .collect::<Vec<_>>();
    manifest
        .files
        .retain(|file| file.generated_by.as_deref() != Some("git"));

    let mut patch_writes = Vec::new();
    for change in changes {
        let logical = change.path.replace('\\', "/");
        let digest = crate::dvpl::crc32(change.diff.as_bytes());
        let patch_path_string = format!("patches/{logical}.{digest:08x}.patch");
        let patch_path = mod_dir.join(
            safe_relative_path(&patch_path_string)
                .map_err(|e| InstallerError::new("MANIFEST_INVALID_PATCH_PATH", e))?,
        );
        patch_writes.push((patch_path, change.diff));
        manifest.files.push(ModFile {
            name: logical,
            action: Action::Patch,
            find: None,
            replace: None,
            patch: Some(patch_path_string),
            generated_by: Some("git".to_owned()),
        });
    }

    validate_manifest(&manifest, &manifest_path)?;
    for (path, diff) in patch_writes {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent)
                .map_err(|e| crate::io_error("could not create patch directory", parent, e))?;
        }
        fs::write(&path, diff)
            .map_err(|e| crate::io_error("could not write generated patch", &path, e))?;
    }

    let mut document: toml_edit::DocumentMut =
        raw_text.parse().map_err(|e: toml_edit::TomlError| {
            InstallerError::at("MANIFEST_INVALID", e.to_string(), &manifest_path)
        })?;
    let files = document
        .get_mut("files")
        .and_then(toml_edit::Item::as_array_of_tables_mut)
        .ok_or_else(|| {
            InstallerError::at(
                "MANIFEST_INVALID",
                "expected [[files]] array-of-table entries",
                &manifest_path,
            )
        })?;
    files
        .retain(|table| table.get("generated_by").and_then(toml_edit::Item::as_str) != Some("git"));
    for file in manifest
        .files
        .iter()
        .filter(|file| file.generated_by.as_deref() == Some("git"))
    {
        let mut table = toml_edit::Table::new();
        table["name"] = toml_edit::value(file.name.clone());
        table["action"] = toml_edit::value("patch");
        table["patch"] = toml_edit::value(file.patch.as_deref().unwrap_or_default());
        table["generated_by"] = toml_edit::value("git");
        files.push(table);
    }
    let serialized = document.to_string();
    write_replacing(&manifest_path, serialized.as_bytes())?;

    for relative in old_generated {
        if let Ok(safe) = safe_relative_path(&relative) {
            let stale = mod_dir.join(safe);
            if !manifest
                .files
                .iter()
                .any(|file| file.patch.as_deref() == Some(&relative))
            {
                let _ = fs::remove_file(stale);
            }
        }
    }
    Ok(manifest)
}

fn write_replacing(path: &Path, bytes: &[u8]) -> Result<()> {
    let temp = path.with_extension("toml.bc-tmp");
    fs::write(&temp, bytes)
        .map_err(|e| crate::io_error("could not write temporary manifest", &temp, e))?;
    #[cfg(windows)]
    if path.exists() {
        let previous = fs::read(path).map_err(|e| {
            let _ = fs::remove_file(&temp);
            crate::io_error("could not read existing mod.toml", path, e)
        })?;
        fs::remove_file(path).map_err(|e| {
            let _ = fs::remove_file(&temp);
            crate::io_error("could not replace mod.toml", path, e)
        })?;
        if let Err(e) = fs::rename(&temp, path) {
            let _ = fs::write(path, previous);
            let _ = fs::remove_file(&temp);
            return Err(crate::io_error("could not replace mod.toml", path, e));
        }
        return Ok(());
    }
    fs::rename(&temp, path).map_err(|e| {
        let _ = fs::remove_file(&temp);
        crate::io_error("could not replace mod.toml", path, e)
    })
}
