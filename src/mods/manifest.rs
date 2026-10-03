use super::git_diff;
use super::patch_sources::load_patches;
use crate::{Error, Result};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use std::fs;
use std::path::{Component, Path, PathBuf};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Manifest {
    pub id: String,
    pub name: String,
    /// Patch files/directories in their declared order.
    #[serde(default)]
    pub patch: Vec<String>,
    #[serde(default)]
    pub files: Vec<ModFile>,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum Action {
    Replace,
    Before,
    After,
    Clear,
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
    /// Resolved single-resource diff; never serialized into a manifest.
    #[serde(skip)]
    pub patch_diff: Option<String>,
    #[serde(
        default,
        rename = "ignore-trailing-whitespace",
        alias = "ignore_trailing_whitespace"
    )]
    pub ignore_trailing_whitespace: bool,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct YamlManifest {
    #[serde(default = "manifest_version")]
    version: u32,
    id: String,
    name: String,
    #[serde(default)]
    patch: Option<PatchSources>,
    #[serde(default)]
    files: serde_yaml_ng::Mapping,
}

#[derive(Deserialize)]
#[serde(untagged)]
enum PatchSources {
    One(String),
    Many(Vec<String>),
}

fn manifest_version() -> u32 {
    1
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct YamlOperation {
    #[serde(default)]
    search: Option<String>,
    #[serde(default, alias = "insert")]
    replace: Option<String>,
    #[serde(default, rename = "ignore-trailing-whitespace")]
    ignore_trailing_whitespace: bool,
}

pub(super) fn locate_mod_dir(path: &Path) -> PathBuf {
    if path.is_file() {
        path.parent()
            .unwrap_or_else(|| Path::new("."))
            .to_path_buf()
    } else {
        path.to_path_buf()
    }
}

pub(super) fn manifest_file(path: &Path) -> Result<PathBuf> {
    if path.is_file() && !is_yaml(path) {
        return Err(Error::at(
            "MANIFEST_FORMAT_UNSUPPORTED",
            "only YAML mod manifests are supported",
            path,
        ));
    }
    let dir = locate_mod_dir(path);
    let candidates = ["mod.yaml", "mod.yml"]
        .into_iter()
        .map(|name| dir.join(name))
        .filter(|candidate| candidate.is_file())
        .collect::<Vec<_>>();
    if candidates.len() > 1 {
        return Err(Error::at(
            "MANIFEST_AMBIGUOUS",
            "multiple mod manifests in the same directory",
            dir,
        ));
    }
    if path.is_file() {
        Ok(path.to_path_buf())
    } else {
        candidates.into_iter().next().ok_or_else(|| {
            Error::at(
                "MANIFEST_NOT_FOUND",
                "no mod.yaml or mod.yml in the directory",
                path,
            )
        })
    }
}

fn is_yaml(path: &Path) -> bool {
    matches!(
        path.extension().and_then(|ext| ext.to_str()),
        Some("yaml" | "yml")
    )
}

fn parse_manifest(text: &str, path: &Path) -> Result<Manifest> {
    let invalid = |message: String| Error::at("MANIFEST_INVALID", message, path);
    let yaml: YamlManifest = serde_yaml_ng::from_str(text).map_err(|e| invalid(e.to_string()))?;
    if yaml.version != 1 {
        return Err(invalid(format!(
            "unsupported manifest version: {}",
            yaml.version
        )));
    }
    let mut files = Vec::new();
    for (name, actions) in yaml.files {
        let name = name
            .as_str()
            .ok_or_else(|| invalid("file path must be a string".to_owned()))?;
        let actions: Vec<BTreeMap<String, YamlOperation>> =
            serde_yaml_ng::from_value(actions).map_err(|e| invalid(e.to_string()))?;
        if actions.is_empty() {
            return Err(invalid(format!("file {name} has no operations")));
        }
        for mut action in actions {
            if action.len() != 1 {
                return Err(invalid(
                    "each operation must contain exactly one action".to_owned(),
                ));
            }
            let (action_name, operation) = action.pop_first().unwrap();
            let action = match action_name.as_str() {
                "replace" => Action::Replace,
                "before" => Action::Before,
                "after" => Action::After,
                "clear" => Action::Clear,
                "patch" => {
                    return Err(invalid(
                        "patch must be declared at the top level of the manifest".to_owned(),
                    ));
                }
                _ => return Err(invalid(format!("unknown action: {action_name}"))),
            };
            files.push(ModFile {
                name: name.to_owned(),
                action,
                find: operation.search,
                replace: operation.replace,
                patch: None,
                patch_diff: None,
                ignore_trailing_whitespace: operation.ignore_trailing_whitespace,
            });
        }
    }
    Ok(Manifest {
        id: yaml.id,
        name: yaml.name,
        patch: match yaml.patch {
            Some(PatchSources::One(path)) => vec![path],
            Some(PatchSources::Many(paths)) => paths,
            None => Vec::new(),
        },
        files,
    })
}

pub fn load_manifest(path: impl AsRef<Path>) -> Result<Manifest> {
    let manifest_path = manifest_file(path.as_ref())?;
    let text = fs::read_to_string(&manifest_path)
        .map_err(|e| crate::io_error("could not read mod manifest", &manifest_path, e))?;
    let mut manifest = parse_manifest(&text, &manifest_path)?;
    validate_manifest(&manifest, &manifest_path)?;
    expand_patches(&mut manifest, &locate_mod_dir(&manifest_path), None)?;
    Ok(manifest)
}

pub(super) fn validate_manifest(manifest: &Manifest, path: &Path) -> Result<()> {
    if manifest.id.trim().is_empty()
        || !manifest
            .id
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || c == '-' || c == '_')
    {
        return Err(Error::at(
            "MANIFEST_INVALID_ID",
            "id must contain only ASCII letters, digits, '-' or '_'",
            path.to_path_buf(),
        ));
    }
    if manifest.name.trim().is_empty() {
        return Err(Error::at(
            "MANIFEST_INVALID_NAME",
            "name must not be empty",
            path.to_path_buf(),
        ));
    }
    if manifest.files.is_empty() && manifest.patch.is_empty() {
        return Err(Error::at(
            "MANIFEST_EMPTY",
            "mod has no file operations",
            path.to_path_buf(),
        ));
    }
    for source in &manifest.patch {
        if source != "." {
            safe_relative_path(source)
                .map_err(|e| Error::at("MANIFEST_INVALID_PATCH_PATH", e, path))?;
        }
    }
    for file in &manifest.files {
        safe_relative_path(&file.name)
            .map_err(|e| Error::at("MANIFEST_INVALID_PATH", e, path.to_path_buf()))?;
        match file.action {
            Action::Replace | Action::Before | Action::After
                if file.find.as_deref().is_none_or(str::is_empty) || file.replace.is_none() =>
            {
                return Err(Error::at(
                    "MANIFEST_INVALID_ACTION",
                    format!(
                        "{:?} action for {} requires non-empty find and present replace",
                        file.action, file.name
                    ),
                    path.to_path_buf(),
                ));
            }
            Action::Patch if file.patch.is_none() => {
                return Err(Error::at(
                    "MANIFEST_INVALID_ACTION",
                    format!("patch action for {} requires a patch path", file.name),
                    path.to_path_buf(),
                ));
            }
            _ => {}
        }
        if file.ignore_trailing_whitespace {
            if !matches!(
                file.action,
                Action::Replace | Action::Before | Action::After
            ) {
                return Err(Error::at(
                    "MANIFEST_INVALID_ACTION",
                    "ignore-trailing-whitespace is only supported for replace/before/after",
                    path,
                ));
            }
            if file
                .find
                .as_deref()
                .unwrap_or_default()
                .trim_matches([' ', '\t', '\r', '\n'])
                .is_empty()
            {
                return Err(Error::at(
                    "MANIFEST_INVALID_ACTION",
                    "normalized find must contain non-whitespace text",
                    path,
                ));
            }
        }
        if let Some(patch) = &file.patch {
            safe_relative_path(patch)
                .map_err(|e| Error::at("MANIFEST_INVALID_PATCH_PATH", e, path.to_path_buf()))?;
        }
    }
    Ok(())
}

pub(super) fn safe_relative_path(path: &str) -> std::result::Result<PathBuf, String> {
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

fn expand_patches(manifest: &mut Manifest, mod_dir: &Path, generated: Option<&str>) -> Result<()> {
    let mut operations = load_patches(mod_dir, &manifest.patch, generated)?;
    operations.append(&mut manifest.files);
    manifest.files = operations;
    Ok(())
}

/// Generate one multi-resource mod.patch, preserving manual operations and sources.
pub fn update_from_git_diff(
    mod_path: impl AsRef<Path>,
    repo: impl AsRef<Path>,
    base_revision: Option<&str>,
) -> Result<Manifest> {
    let manifest_path = manifest_file(mod_path.as_ref())?;
    let mod_dir = locate_mod_dir(&manifest_path);
    let raw_text = fs::read_to_string(&manifest_path)
        .map_err(|e| crate::io_error("could not read mod manifest", &manifest_path, e))?;
    let mut manifest = parse_manifest(&raw_text, &manifest_path)?;
    manifest.patch.retain(|source| {
        source == "." || safe_relative_path(source).ok().as_deref() != Some(Path::new("mod.patch"))
    });
    manifest.patch.insert(0, "mod.patch".to_owned());
    validate_manifest(&manifest, &manifest_path)?;
    let changes = git_diff::collect_changes(repo.as_ref(), base_revision.unwrap_or("HEAD"))?;
    if changes.is_empty() {
        return Err(Error::new(
            "GIT_DIFF_EMPTY",
            "no modified text resources to generate mod.patch",
        ));
    }
    let generated = changes
        .into_iter()
        .map(|change| change.diff)
        .collect::<String>();
    // Resolve every source before writing, substituting the proposed generated diff.
    expand_patches(&mut manifest, &mod_dir, Some(&generated))?;
    let mut document: serde_yaml_ng::Value = serde_yaml_ng::from_str(&raw_text)
        .map_err(|e| Error::at("MANIFEST_INVALID", e.to_string(), &manifest_path))?;
    document["patch"] = if manifest.patch.len() == 1 {
        serde_yaml_ng::Value::String(manifest.patch[0].clone())
    } else {
        serde_yaml_ng::to_value(&manifest.patch)
            .map_err(|e| Error::new("MANIFEST_INVALID", e.to_string()))?
    };
    let serialized = serde_yaml_ng::to_string(&document)
        .map_err(|e| Error::at("MANIFEST_INVALID", e.to_string(), &manifest_path))?;
    let patch_path = mod_dir.join("mod.patch");
    let previous = fs::read(&patch_path).ok();
    write_replacing(&patch_path, generated.as_bytes())?;
    if let Err(error) = write_replacing(&manifest_path, serialized.as_bytes()) {
        if let Some(previous) = previous {
            let _ = write_replacing(&patch_path, &previous);
        } else {
            let _ = fs::remove_file(&patch_path);
        }
        return Err(error);
    }
    Ok(manifest)
}

fn write_replacing(path: &Path, bytes: &[u8]) -> Result<()> {
    let temp = path.with_extension("bc-tmp");
    fs::write(&temp, bytes)
        .map_err(|e| crate::io_error("could not write temporary manifest", &temp, e))?;
    #[cfg(windows)]
    if path.exists() {
        let previous = fs::read(path).map_err(|e| {
            let _ = fs::remove_file(&temp);
            crate::io_error("could not read existing mod manifest", path, e)
        })?;
        fs::remove_file(path).map_err(|e| {
            let _ = fs::remove_file(&temp);
            crate::io_error("could not replace mod manifest", path, e)
        })?;
        if let Err(e) = fs::rename(&temp, path) {
            let _ = fs::write(path, previous);
            let _ = fs::remove_file(&temp);
            return Err(crate::io_error("could not replace mod manifest", path, e));
        }
        return Ok(());
    }
    fs::rename(&temp, path).map_err(|e| {
        let _ = fs::remove_file(&temp);
        crate::io_error("could not replace mod manifest", path, e)
    })
}
