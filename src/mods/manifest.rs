use super::git_diff;
use crate::{Error, Result};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
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
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub generated_by: Option<String>,
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
    files: serde_yaml_ng::Mapping,
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
    #[serde(default)]
    patch: Option<String>,
    #[serde(default, rename = "generated-by")]
    generated_by: Option<String>,
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
                "patch" => Action::Patch,
                _ => return Err(invalid(format!("unknown action: {action_name}"))),
            };
            files.push(ModFile {
                name: name.to_owned(),
                action,
                find: operation.search,
                replace: operation.replace,
                patch: operation.patch,
                generated_by: operation.generated_by,
                ignore_trailing_whitespace: operation.ignore_trailing_whitespace,
            });
        }
    }
    Ok(Manifest {
        id: yaml.id,
        name: yaml.name,
        files,
    })
}

pub fn load_manifest(path: impl AsRef<Path>) -> Result<Manifest> {
    let manifest_path = manifest_file(path.as_ref())?;
    let text = fs::read_to_string(&manifest_path)
        .map_err(|e| crate::io_error("could not read mod manifest", &manifest_path, e))?;
    let manifest = parse_manifest(&text, &manifest_path)?;
    validate_manifest(&manifest, &manifest_path)?;
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
    if manifest.files.is_empty() {
        return Err(Error::at(
            "MANIFEST_EMPTY",
            "mod has no file operations",
            path.to_path_buf(),
        ));
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

pub fn update_from_git_diff(
    mod_path: impl AsRef<Path>,
    repo: impl AsRef<Path>,
    base_revision: Option<&str>,
) -> Result<Manifest> {
    let mod_dir = locate_mod_dir(mod_path.as_ref());
    let manifest_path = manifest_file(mod_path.as_ref())?;
    let raw_text = fs::read_to_string(&manifest_path)
        .map_err(|e| crate::io_error("could not read mod manifest", &manifest_path, e))?;
    let mut manifest = parse_manifest(&raw_text, &manifest_path)?;
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
        let digest = crate::resources::crc32(change.diff.as_bytes());
        let patch_path_string = format!("patches/{logical}.{digest:08x}.patch");
        let patch_path = mod_dir.join(
            safe_relative_path(&patch_path_string)
                .map_err(|e| Error::new("MANIFEST_INVALID_PATCH_PATH", e))?,
        );
        patch_writes.push((patch_path, change.diff));
        manifest.files.push(ModFile {
            name: logical,
            action: Action::Patch,
            find: None,
            replace: None,
            patch: Some(patch_path_string),
            generated_by: Some("git".to_owned()),
            ignore_trailing_whitespace: false,
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

    let serialized = update_yaml_manifest(&raw_text, &manifest, &manifest_path)?;
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

fn update_yaml_manifest(raw: &str, manifest: &Manifest, path: &Path) -> Result<String> {
    use serde_yaml_ng::Value;
    let invalid = |message: String| Error::at("MANIFEST_INVALID", message, path);
    let mut document: Value = serde_yaml_ng::from_str(raw).map_err(|e| invalid(e.to_string()))?;
    let files = document
        .get_mut("files")
        .and_then(Value::as_mapping_mut)
        .ok_or_else(|| invalid("expected files mapping".to_owned()))?;
    for (_, actions) in files.iter_mut() {
        let actions = actions
            .as_sequence_mut()
            .ok_or_else(|| invalid("expected operations list".to_owned()))?;
        actions.retain(|action| {
            !action.as_mapping().is_some_and(|map| {
                map.values().any(|operation| {
                    operation.get("generated-by").and_then(Value::as_str) == Some("git")
                })
            })
        });
    }
    files.retain(|_, actions| {
        actions
            .as_sequence()
            .is_some_and(|actions| !actions.is_empty())
    });
    for file in manifest
        .files
        .iter()
        .filter(|file| file.generated_by.as_deref() == Some("git"))
    {
        let actions = files
            .entry(Value::String(file.name.clone()))
            .or_insert_with(|| Value::Sequence(Vec::new()));
        let mut fields = serde_yaml_ng::Mapping::new();
        fields.insert(
            Value::String("patch".into()),
            Value::String(file.patch.clone().unwrap()),
        );
        fields.insert(
            Value::String("generated-by".into()),
            Value::String("git".into()),
        );
        let mut operation = serde_yaml_ng::Mapping::new();
        operation.insert(Value::String("patch".into()), Value::Mapping(fields));
        actions
            .as_sequence_mut()
            .unwrap()
            .push(Value::Mapping(operation));
    }
    serde_yaml_ng::to_string(&document).map_err(|e| invalid(e.to_string()))
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
