use super::assets::{read_client_asset, resolve_client_asset};
use crate::Result;
use std::collections::HashMap;
use std::path::Path;

pub(crate) fn read_localized_names(data_dir: &Path) -> Result<HashMap<String, String>> {
    let Some(path) = resolve_client_asset(data_dir, "Strings/ru.yaml") else {
        return Ok(HashMap::new());
    };
    let bytes = read_client_asset(&path)?;
    let content = String::from_utf8_lossy(&bytes);
    let mut names = HashMap::new();
    for line in content.lines() {
        let line = line.trim_start();
        let Some((key, value)) = parse_quoted_yaml_pair(line) else {
            continue;
        };
        names.insert(key, value);
    }
    Ok(names)
}

fn parse_quoted_yaml_pair(line: &str) -> Option<(String, String)> {
    let (key, consumed) = quoted_json_string(line)?;
    let rest = line[consumed..].trim_start();
    let rest = rest.strip_prefix(':')?.trim_start();
    let (value, consumed) = quoted_json_string(rest)?;
    if rest[consumed..].trim().is_empty() {
        Some((key, value))
    } else {
        None
    }
}

fn quoted_json_string(source: &str) -> Option<(String, usize)> {
    if !source.starts_with('"') {
        return None;
    }
    let bytes = source.as_bytes();
    let mut index = 1;
    let mut escaped = false;
    while index < bytes.len() {
        match bytes[index] {
            b'\\' if !escaped => escaped = true,
            b'"' if !escaped => {
                let end = index + 1;
                let parsed = serde_json::from_str(&source[..end]).ok()?;
                return Some((parsed, end));
            }
            _ => escaped = false,
        }
        index += 1;
    }
    None
}
