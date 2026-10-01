use crate::client::{
    self, ClientTank, TankModelFiles, child_text, parse_xml, read_client_asset,
    read_localized_names, resolve_client_asset,
};
use crate::{InstallerError, Result};
use serde::Serialize;
use std::collections::{BTreeMap, HashMap};
use std::path::{Path, PathBuf};

/// Data exported from `maps.yaml` and `XML/item_defs/vehicles/*/list.xml`.
#[derive(Debug, Clone)]
pub struct ClientCatalog {
    pub maps: Vec<ClientMap>,
    pub tanks: Vec<ClientTankMetadata>,
    /// Vehicles with both showroom model resources, kept in legacy model order.
    pub model_tanks: Vec<ClientTank>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ClientMap {
    pub id: i64,
    #[serde(rename = "localName")]
    pub local_name: String,
    pub available_in_training_room: bool,
    pub available_modes: Vec<i64>,
    pub sprite_frame: i64,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ClientTankMetadata {
    pub id: String,
    pub user_string: String,
    pub short_user_string: String,
    pub description: String,
    pub price: ClientPrice,
    pub sell_price: Option<ClientPrice>,
    pub not_in_shop: bool,
    pub enrichment_permanent_cost: i64,
    pub tags: Vec<String>,
    pub level: i64,
    pub nation: String,
    pub combat_role: BTreeMap<String, String>,
    pub configuration_modes: Vec<String>,
    pub leasing: Option<String>,
}

#[derive(Debug, Clone, Serialize)]
pub struct ClientPrice {
    pub value: i64,
    #[serde(rename = "is_gold")]
    pub is_gold: bool,
}

/// Collect the complete data catalog and the subset of tanks with model files.
/// XML lists and Russian localization are read once and shared by the metadata
/// and model outputs.
pub fn discover(data_dir: &Path) -> Result<ClientCatalog> {
    let discovered = discover_tanks(data_dir, TankDiscoveryScope::FullCatalog)?;
    let maps = read_maps(data_dir, &discovered.localized)?;
    Ok(ClientCatalog {
        maps,
        tanks: discovered.metadata,
        model_tanks: discovered.models,
    })
}

/// Discover paired showroom files without requiring full catalog metadata.
pub fn discover_models(data_dir: &Path) -> Result<Vec<ClientTank>> {
    discover_tanks(data_dir, TankDiscoveryScope::ModelsOnly).map(|discovered| discovered.models)
}

#[derive(Clone, Copy)]
enum TankDiscoveryScope {
    ModelsOnly,
    FullCatalog,
}

struct DiscoveredTanks {
    metadata: Vec<ClientTankMetadata>,
    models: Vec<ClientTank>,
    localized: HashMap<String, String>,
}

// Both projections share resource loading; only full catalog discovery applies
// metadata invariants, including to entries without showroom model files.
fn discover_tanks(data_dir: &Path, scope: TankDiscoveryScope) -> Result<DiscoveredTanks> {
    let vehicles_dir = data_dir.join("XML/item_defs/vehicles");
    if !vehicles_dir.is_dir() {
        return Err(InstallerError::at(
            "CLIENT_DATA_INVALID",
            "the selected directory does not contain XML/item_defs/vehicles",
            data_dir.to_path_buf(),
        ));
    }

    let localized = read_localized_names(data_dir)?;
    let available_nations = read_available_nations(data_dir)?;
    let mut metadata = Vec::new();
    let mut model_tanks_by_nation = HashMap::<String, Vec<ClientTank>>::new();

    for nation in available_nations {
        let relative_list = format!("XML/item_defs/vehicles/{nation}/list.xml");
        let Some(list_path) = resolve_client_asset(data_dir, &relative_list) else {
            continue;
        };
        let bytes = read_client_asset(&list_path)?;
        let root = parse_xml(&bytes).map_err(|message| {
            InstallerError::at("CLIENT_XML_INVALID", message, list_path.clone())
        })?;

        let nation_info = client::nations()
            .iter()
            .find(|(known_nation, _, _)| *known_nation == nation);

        for item in &root.children {
            if matches!(scope, TankDiscoveryScope::FullCatalog) {
                metadata.push(tank_metadata(item, &nation, &localized, &list_path)?);
            }
            if let Some((_, model_folder, nation_label)) = nation_info
                && let Some(model) = showroom_model(
                    data_dir,
                    item,
                    &nation,
                    model_folder,
                    nation_label,
                    &localized,
                )
            {
                model_tanks_by_nation
                    .entry(nation.clone())
                    .or_default()
                    .push(model);
            }
        }
    }

    let mut models = Vec::new();
    for (nation, _, _) in client::nations() {
        if let Some(mut nation_tanks) = model_tanks_by_nation.remove(*nation) {
            models.append(&mut nation_tanks);
        }
    }
    Ok(DiscoveredTanks {
        metadata,
        models,
        localized,
    })
}

fn tank_metadata(
    item: &client::XmlNode,
    nation: &str,
    localized: &HashMap<String, String>,
    list_path: &Path,
) -> Result<ClientTankMetadata> {
    let user_string = required_text(item, "userString", list_path)?;
    Ok(ClientTankMetadata {
        id: required_text(item, "id", list_path)?,
        user_string: localize(&user_string, localized),
        short_user_string: child_text(item, "shortUserString")
            .map(|value| localize(&value, localized))
            .unwrap_or_default(),
        description: child_text(item, "description")
            .map(|value| localize(&value, localized))
            .unwrap_or_default(),
        price: parse_price(required_node(item, "price", list_path)?, list_path)?,
        sell_price: item
            .children
            .iter()
            .find(|child| child.name == "sellPrice")
            .map(|node| parse_price(node, list_path))
            .transpose()?,
        not_in_shop: child_text(item, "notInShop")
            .is_some_and(|value| value.eq_ignore_ascii_case("true")),
        enrichment_permanent_cost: required_number(item, "enrichmentPermanentCost", list_path)?,
        tags: required_text(item, "tags", list_path)?
            .split_whitespace()
            .map(str::to_owned)
            .collect(),
        level: required_number(item, "level", list_path)?,
        nation: nation.to_owned(),
        combat_role: item
            .children
            .iter()
            .find(|child| child.name == "combatRole")
            .map(|node| {
                node.children
                    .iter()
                    .map(|role| (role.name.clone(), role.text.trim().to_owned()))
                    .collect()
            })
            .unwrap_or_default(),
        configuration_modes: required_text(item, "configurationModes", list_path)?
            .split_whitespace()
            .map(str::to_owned)
            .collect(),
        leasing: child_text(item, "leasing")
            .map(|value| parse_leasing(&value, list_path))
            .transpose()?,
    })
}

fn showroom_model(
    data_dir: &Path,
    item: &client::XmlNode,
    nation: &str,
    model_folder: &str,
    nation_label: &str,
    localized: &HashMap<String, String>,
) -> Option<ClientTank> {
    let model_root = PathBuf::from("3d/Tanks").join(model_folder);
    let scene = resolve_client_asset(
        data_dir,
        &model_root
            .join(format!("{}.sc2.dvpl", item.name))
            .to_string_lossy(),
    )?;
    let geometry = resolve_client_asset(
        data_dir,
        &model_root
            .join(format!("{}.scg.dvpl", item.name))
            .to_string_lossy(),
    )?;
    let user_string = child_text(item, "userString");
    let name = user_string
        .as_ref()
        .and_then(|key| localized.get(key))
        .cloned()
        .unwrap_or_else(|| item.name.replace('_', " "));
    Some(ClientTank {
        id: item.name.clone(),
        nation: nation.to_owned(),
        nation_label: nation_label.to_owned(),
        level: child_text(item, "level"),
        user_string,
        name,
        model_files: TankModelFiles { scene, geometry },
    })
}

fn read_available_nations(data_dir: &Path) -> Result<Vec<String>> {
    let Some(path) = resolve_client_asset(data_dir, "available_nations.yaml") else {
        return Ok(client::nations()
            .iter()
            .map(|(nation, _, _)| (*nation).to_owned())
            .collect());
    };
    let source = String::from_utf8_lossy(&read_client_asset(&path)?).into_owned();
    let mut in_list = false;
    let mut nations = Vec::new();
    for line in source.lines() {
        let trimmed = line.trim();
        if trimmed.is_empty() || trimmed.starts_with('#') {
            continue;
        }
        if trimmed == "available_nations:" {
            in_list = true;
            continue;
        }
        if !in_list {
            continue;
        }
        let Some(value) = trimmed.strip_prefix('-') else {
            break;
        };
        let nation = yaml_string(value.trim());
        if !nation.is_empty() {
            nations.push(nation);
        }
    }
    if nations.is_empty() {
        return Err(InstallerError::at(
            "CLIENT_YAML_INVALID",
            "available_nations.yaml contains no nation entries",
            path,
        ));
    }
    Ok(nations)
}

fn read_maps(data_dir: &Path, localized: &HashMap<String, String>) -> Result<Vec<ClientMap>> {
    let path = resolve_client_asset(data_dir, "maps.yml")
        .or_else(|| resolve_client_asset(data_dir, "maps.yaml"))
        .ok_or_else(|| {
            InstallerError::at(
                "CLIENT_DATA_INVALID",
                "maps.yml or maps.yaml was not found in the client resource directory",
                data_dir.to_path_buf(),
            )
        })?;
    let source = String::from_utf8_lossy(&read_client_asset(&path)?).into_owned();
    parse_maps(&source, localized)
        .map_err(|message| InstallerError::at("CLIENT_YAML_INVALID", message, path))
}

fn parse_maps(
    source: &str,
    localized: &HashMap<String, String>,
) -> std::result::Result<Vec<ClientMap>, String> {
    let mut in_maps = false;
    let mut current_name = None;
    let mut current = BTreeMap::<String, String>::new();
    let mut maps = Vec::new();
    for line in source.lines() {
        let trimmed = line.trim();
        if !in_maps {
            if trimmed == "maps:" {
                in_maps = true;
            }
            continue;
        }
        if trimmed.is_empty() || trimmed.starts_with('#') {
            continue;
        }
        if line.starts_with("    ") && !line.starts_with("        ") {
            finish_map(current_name.take(), &current, localized, &mut maps)?;
            current.clear();
            let key = trimmed
                .strip_suffix(':')
                .ok_or_else(|| format!("invalid map entry {trimmed:?}"))?;
            current_name = Some(yaml_string(key));
        } else if line.starts_with("        ") {
            if let Some((key, value)) = trimmed.split_once(':') {
                current.insert(key.to_owned(), value.trim().to_owned());
            }
        } else if !line.starts_with(' ') && trimmed.ends_with(':') {
            break;
        }
    }
    finish_map(current_name, &current, localized, &mut maps)?;
    if !in_maps || maps.is_empty() {
        return Err("maps.yaml contains no map entries".to_owned());
    }
    Ok(maps)
}

fn finish_map(
    name: Option<String>,
    values: &BTreeMap<String, String>,
    localized: &HashMap<String, String>,
    output: &mut Vec<ClientMap>,
) -> std::result::Result<(), String> {
    let Some(name) = name else {
        return Ok(());
    };
    let get = |key: &str| values.get(key).map(String::as_str);
    let local_name = get("localName").map(yaml_string).unwrap_or_default();
    let id = get("id")
        .ok_or_else(|| format!("map {name:?} is missing id"))?
        .parse::<i64>()
        .map_err(|_| format!("map {name:?} has invalid id"))?;
    let frame = get("frame")
        .and_then(yaml_optional_number)
        .or_else(|| get("spriteFrame").and_then(yaml_optional_number))
        .ok_or_else(|| format!("map {name:?} has neither frame nor spriteFrame"))?;
    let available_modes = get("availableModes")
        .map(yaml_number_list)
        .transpose()?
        .unwrap_or_default();
    let available_in_training_room = get("availableInTrainingRoom")
        .and_then(yaml_bool)
        .unwrap_or(false)
        || get("avaliableInTrainingRoom")
            .and_then(yaml_bool)
            .unwrap_or(false);
    let localization_key = format!("#maps:{name}:{local_name}");
    let local_name = localized
        .get(&localization_key)
        .cloned()
        .unwrap_or(local_name);
    output.push(ClientMap {
        id,
        local_name,
        available_in_training_room,
        available_modes,
        sprite_frame: frame,
    });
    Ok(())
}

fn yaml_string(value: &str) -> String {
    let value = value.trim();
    if value.starts_with('"') {
        let mut escaped = false;
        for (index, byte) in value.bytes().enumerate().skip(1) {
            match byte {
                b'\\' if !escaped => escaped = true,
                b'"' if !escaped => {
                    if let Ok(parsed) = serde_json::from_str::<String>(&value[..=index]) {
                        return parsed;
                    }
                    break;
                }
                _ => escaped = false,
            }
        }
    }
    if value.starts_with('\'') && value.ends_with('\'') && value.len() >= 2 {
        return value[1..value.len() - 1].replace("''", "'");
    }
    value.to_owned()
}

fn yaml_optional_number(value: &str) -> Option<i64> {
    let value = value.trim();
    if value.is_empty() || value == "null" || value == "~" {
        None
    } else {
        value.parse().ok()
    }
}

fn yaml_number_list(value: &str) -> std::result::Result<Vec<i64>, String> {
    let value = value.trim();
    let body = value
        .strip_prefix('[')
        .and_then(|value| value.strip_suffix(']'))
        .ok_or_else(|| format!("expected a numeric YAML list, got {value:?}"))?;
    body.split(',')
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(|value| {
            value
                .parse::<i64>()
                .map_err(|_| format!("invalid number {value:?} in YAML list"))
        })
        .collect()
}

fn yaml_bool(value: &str) -> Option<bool> {
    match value.trim() {
        "true" | "True" | "TRUE" => Some(true),
        "false" | "False" | "FALSE" => Some(false),
        _ => None,
    }
}

fn required_node<'a>(
    node: &'a client::XmlNode,
    name: &str,
    path: &Path,
) -> Result<&'a client::XmlNode> {
    node.children
        .iter()
        .find(|child| child.name == name)
        .ok_or_else(|| {
            InstallerError::at(
                "CLIENT_XML_INVALID",
                format!("{} is missing <{name}>", node.name),
                path.to_path_buf(),
            )
        })
}

fn required_text(node: &client::XmlNode, name: &str, path: &Path) -> Result<String> {
    child_text(node, name).ok_or_else(|| {
        InstallerError::at(
            "CLIENT_XML_INVALID",
            format!("{} has an empty or missing <{name}>", node.name),
            path.to_path_buf(),
        )
    })
}

fn required_number(node: &client::XmlNode, name: &str, path: &Path) -> Result<i64> {
    required_text(node, name, path)?.parse().map_err(|_| {
        InstallerError::at(
            "CLIENT_XML_INVALID",
            format!("{} has an invalid integer in <{name}>", node.name),
            path.to_path_buf(),
        )
    })
}

fn parse_price(node: &client::XmlNode, path: &Path) -> Result<ClientPrice> {
    let value = node.text.trim().parse().map_err(|_| {
        InstallerError::at(
            "CLIENT_XML_INVALID",
            format!("{} contains an invalid price", node.name),
            path.to_path_buf(),
        )
    })?;
    Ok(ClientPrice {
        value,
        is_gold: node.children.iter().any(|child| child.name == "gold"),
    })
}

fn parse_leasing(value: &str, path: &Path) -> Result<String> {
    let invalid = || {
        InstallerError::at(
            "CLIENT_XML_INVALID",
            format!("invalid leasing date {value:?}"),
            path.to_path_buf(),
        )
    };
    let (date, time) = value.split_once(' ').ok_or_else(invalid)?;
    let date_parts = date.split('.').collect::<Vec<_>>();
    let [day, month, year] = date_parts.as_slice() else {
        return Err(invalid());
    };
    let time_parts = time.split(':').collect::<Vec<_>>();
    let [hour, minute] = time_parts.as_slice() else {
        return Err(invalid());
    };
    if !day
        .parse::<u32>()
        .is_ok_and(|number| (1..=31).contains(&number))
        || !month
            .parse::<u32>()
            .is_ok_and(|number| (1..=12).contains(&number))
        || !year.parse::<u32>().is_ok_and(|number| number > 0)
        || !hour.parse::<u32>().is_ok_and(|number| number < 24)
        || !minute.parse::<u32>().is_ok_and(|number| number < 60)
    {
        return Err(invalid());
    }
    Ok(format!("{year}-{month}-{day}T{time}:00"))
}

fn localize(value: &str, localized: &HashMap<String, String>) -> String {
    localized
        .get(value)
        .cloned()
        .unwrap_or_else(|| value.to_owned())
}

#[cfg(test)]
mod tests;
