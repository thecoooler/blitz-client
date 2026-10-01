use super::{catalog, resolve_game_dir};
use crate::Result;
use serde::Serialize;
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
    catalog::discover_models(data_dir)
}
