use crate::{InstallerError, Result, dvpl, resolve_game_dir};
use serde::Serialize;
use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};

const NATIONS: [(&str, &str, &str); 9] = [
    ("china", "China", "Китай"),
    ("usa", "USA", "США"),
    ("ussr", "USSR", "СССР"),
    ("germany", "German", "Германия"),
    ("france", "France", "Франция"),
    ("uk", "GB", "Великобритания"),
    ("japan", "Japan", "Япония"),
    ("european", "European", "Европа"),
    ("other", "Other", "Другое"),
];

pub(crate) fn nations() -> &'static [(&'static str, &'static str, &'static str); 9] {
    &NATIONS
}

pub fn is_known_nation(nation: &str) -> bool {
    NATIONS.iter().any(|(candidate, _, _)| *candidate == nation)
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ClientTank {
    pub id: String,
    pub nation: String,
    pub nation_label: String,
    pub level: Option<String>,
    pub user_string: Option<String>,
    pub name: String,
    pub model_files: TankModelFiles,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TankModelFiles {
    pub scene: PathBuf,
    pub geometry: PathBuf,
}

/// List vehicles whose installed client has both showroom model resources.
///
/// `game_dir` accepts the game directory or its `Data` directory. When it is
/// `None`, the platform's ordered default locations are searched.
pub fn list_client_tanks(game_dir: Option<PathBuf>) -> Result<Vec<ClientTank>> {
    let resolved = resolve_game_dir(game_dir)?;
    discover_client_tanks(&resolved.resource_dir)
}

/// Read the client's vehicle lists and keep entries with a paired SC2/SCG model.
pub fn discover_client_tanks(data_dir: &Path) -> Result<Vec<ClientTank>> {
    let vehicles_dir = data_dir.join("XML/item_defs/vehicles");
    if !vehicles_dir.is_dir() {
        return Err(InstallerError::at(
            "CLIENT_DATA_INVALID",
            "the selected directory does not contain XML/item_defs/vehicles",
            data_dir.to_path_buf(),
        ));
    }
    let localized_names = read_localized_names(data_dir)?;
    let mut vehicles = Vec::new();

    for (nation, model_folder, nation_label) in NATIONS {
        let list_path = vehicles_dir.join(nation).join("list.xml.dvpl");
        if !list_path.is_file() {
            continue;
        }
        let list_bytes = read_client_asset(&list_path)?;
        let root = parse_xml(&list_bytes).map_err(|message| {
            InstallerError::at("CLIENT_XML_INVALID", message, list_path.clone())
        })?;
        let model_dir = data_dir.join("3d/Tanks").join(model_folder);

        for item in &root.children {
            let scene = model_dir.join(format!("{}.sc2.dvpl", item.name));
            let geometry = model_dir.join(format!("{}.scg.dvpl", item.name));
            if !scene.is_file() || !geometry.is_file() {
                continue;
            }
            let user_string = child_text(item, "userString");
            let fallback_name = item.name.replace('_', " ");
            let name = user_string
                .as_deref()
                .and_then(|key| localized_names.get(key))
                .cloned()
                .unwrap_or(fallback_name);
            vehicles.push(ClientTank {
                id: item.name.clone(),
                nation: nation.to_owned(),
                nation_label: nation_label.to_owned(),
                level: child_text(item, "level"),
                user_string,
                name,
                model_files: TankModelFiles { scene, geometry },
            });
        }
    }
    Ok(vehicles)
}

fn read_client_asset(path: &Path) -> Result<Vec<u8>> {
    let bytes = fs::read(path)
        .map_err(|error| crate::io_error("could not read client resource", path, error))?;
    dvpl::decode(&bytes)
        .map(|(plain, _)| plain)
        .map_err(|mut error| {
            error.path = Some(path.to_path_buf());
            error
        })
}

fn read_localized_names(data_dir: &Path) -> Result<HashMap<String, String>> {
    let path = data_dir.join("Strings/ru.yaml.dvpl");
    if !path.is_file() {
        return Ok(HashMap::new());
    }
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

#[derive(Debug)]
pub(crate) struct XmlNode {
    pub(crate) name: String,
    pub(crate) text: String,
    pub(crate) children: Vec<XmlNode>,
}

pub(crate) fn child_text(node: &XmlNode, name: &str) -> Option<String> {
    node.children
        .iter()
        .find(|child| child.name == name)
        .map(|child| child.text.trim().to_owned())
        .filter(|text| !text.is_empty())
}

pub(crate) fn parse_xml(data: &[u8]) -> std::result::Result<XmlNode, String> {
    let decoded = String::from_utf8_lossy(data);
    let source = decoded.trim_start_matches('\u{feff}');
    let mut roots = Vec::<XmlNode>::new();
    let mut stack = Vec::<XmlNode>::new();
    let mut cursor = 0;

    while cursor < source.len() {
        let Some(relative_open) = source[cursor..].find('<') else {
            append_xml_text(&mut stack, &source[cursor..])?;
            break;
        };
        let open = cursor + relative_open;
        if open > cursor {
            append_xml_text(&mut stack, &source[cursor..open])?;
        }

        if source[open..].starts_with("<!--") {
            let end = source[open + 4..]
                .find("-->")
                .ok_or_else(|| "unterminated XML comment".to_owned())?;
            cursor = open + 4 + end + 3;
            continue;
        }
        if source[open..].starts_with("<![CDATA[") {
            let end = source[open + 9..]
                .find("]]>")
                .ok_or_else(|| "unterminated XML CDATA section".to_owned())?;
            let content_start = open + 9;
            let content_end = content_start + end;
            if let Some(node) = stack.last_mut() {
                node.text.push_str(&source[content_start..content_end]);
            }
            cursor = content_end + 3;
            continue;
        }
        if source[open..].starts_with("<?") || source[open..].starts_with("<!") {
            cursor = find_xml_tag_end(&source, open)? + 1;
            continue;
        }

        let end = find_xml_tag_end(&source, open)?;
        let mut body = source[open + 1..end].trim();
        if let Some(close) = body.strip_prefix('/') {
            let close_name = close.trim();
            let node = stack
                .pop()
                .ok_or_else(|| format!("unexpected closing XML tag </{close_name}>"))?;
            if node.name != close_name {
                return Err(format!(
                    "mismatched XML closing tag </{close_name}> for <{}>",
                    node.name
                ));
            }
            append_xml_node(&mut roots, &mut stack, node);
        } else {
            let self_closing = body.ends_with('/');
            if self_closing {
                body = body[..body.len() - 1].trim_end();
            }
            let name_end = body.find(char::is_whitespace).unwrap_or(body.len());
            if name_end == 0 {
                return Err("empty XML tag name".to_owned());
            }
            let node = XmlNode {
                name: body[..name_end].to_owned(),
                text: String::new(),
                children: Vec::new(),
            };
            if self_closing {
                append_xml_node(&mut roots, &mut stack, node);
            } else {
                stack.push(node);
            }
        }
        cursor = end + 1;
    }

    if !stack.is_empty() {
        return Err(format!("unclosed XML tag <{}>", stack.last().unwrap().name));
    }
    if roots.len() != 1 {
        return Err(format!(
            "expected one XML root element, found {}",
            roots.len()
        ));
    }
    Ok(roots.remove(0))
}

fn append_xml_node(roots: &mut Vec<XmlNode>, stack: &mut [XmlNode], node: XmlNode) {
    if let Some(parent) = stack.last_mut() {
        parent.children.push(node);
    } else {
        roots.push(node);
    }
}

fn append_xml_text(stack: &mut [XmlNode], text: &str) -> std::result::Result<(), String> {
    let Some(node) = stack.last_mut() else {
        if text.trim().is_empty() {
            return Ok(());
        }
        return Err("text outside the XML root element".to_owned());
    };
    node.text.push_str(&decode_xml_entities(text)?);
    Ok(())
}

fn find_xml_tag_end(source: &str, open: usize) -> std::result::Result<usize, String> {
    let mut quote = None;
    for (offset, ch) in source[open + 1..].char_indices() {
        match (quote, ch) {
            (Some(current), value) if current == value => quote = None,
            (None, '\'' | '"') => quote = Some(ch),
            (None, '>') => return Ok(open + 1 + offset),
            _ => {}
        }
    }
    Err("unterminated XML tag".to_owned())
}

fn decode_xml_entities(text: &str) -> std::result::Result<String, String> {
    let mut result = String::with_capacity(text.len());
    let mut rest = text;
    while let Some(start) = rest.find('&') {
        result.push_str(&rest[..start]);
        let entity_end = rest[start..]
            .find(';')
            .ok_or_else(|| "unterminated XML entity".to_owned())?
            + start;
        let entity = &rest[start + 1..entity_end];
        let decoded = match entity {
            "amp" => '&',
            "lt" => '<',
            "gt" => '>',
            "quot" => '"',
            "apos" => '\'',
            value if value.starts_with("#x") || value.starts_with("#X") => {
                let scalar = u32::from_str_radix(&value[2..], 16)
                    .map_err(|_| format!("invalid XML entity &{value};"))?;
                char::from_u32(scalar).ok_or_else(|| format!("invalid XML scalar in &{value};"))?
            }
            value if value.starts_with('#') => {
                let scalar = value[1..]
                    .parse::<u32>()
                    .map_err(|_| format!("invalid XML entity &{value};"))?;
                char::from_u32(scalar).ok_or_else(|| format!("invalid XML scalar in &{value};"))?
            }
            value => return Err(format!("unsupported XML entity &{value};")),
        };
        result.push(decoded);
        rest = &rest[entity_end + 1..];
    }
    result.push_str(rest);
    Ok(result)
}
