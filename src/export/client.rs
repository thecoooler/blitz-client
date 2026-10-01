use super::tanks::{TankExportOptions, TankExportReport};
use crate::Result;
use crate::client::{discover_client_catalog, resolve_game_dir};
use serde::Serialize;
use std::fs;
use std::path::PathBuf;
use std::time::Instant;

#[derive(Debug, Clone)]
pub struct ClientExportOptions {
    /// Install directory or its `Data` directory; omitted to use platform defaults.
    pub game_dir: Option<PathBuf>,
    /// Destination for `maps.json`, `tanks.json` and `assets/tanks`.
    pub output_dir: PathBuf,
    /// Write only the map and tank metadata files.
    pub data_only: bool,
    /// Model export options. Its `viewer_dir` is set to `output_dir` by this API.
    pub model_options: TankExportOptions,
}

impl ClientExportOptions {
    pub fn new(output_dir: impl Into<PathBuf>) -> Self {
        let output_dir = output_dir.into();
        Self {
            game_dir: None,
            model_options: TankExportOptions::new(&output_dir),
            output_dir,
            data_only: false,
        }
    }
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ClientExportReport {
    pub output_dir: PathBuf,
    pub maps_file: PathBuf,
    pub tanks_file: PathBuf,
    pub map_count: usize,
    pub tank_count: usize,
    pub data_only: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub model_export: Option<TankExportReport>,
    pub elapsed_seconds: f64,
}

/// Export client maps, full tank metadata, and optionally the existing model
/// bundles in one run. Resource lists and localization are shared with model
/// discovery, and the metadata files use the Python exporter's JSON schema.
pub fn export_client(options: &ClientExportOptions) -> Result<ClientExportReport> {
    let started = Instant::now();
    let resolved = resolve_game_dir(options.game_dir.clone())?;
    let catalog_started = Instant::now();
    let catalog = discover_client_catalog(&resolved.resource_dir)?;
    let discovery_seconds = catalog_started.elapsed().as_secs_f64();

    let output_dir = prepare_output_dir(&options.output_dir)?;
    let maps_file = output_dir.join("maps.json");
    let tanks_file = output_dir.join("tanks.json");
    super::output::write_json(&maps_file, &catalog.maps, false)?;
    super::output::write_json(&tanks_file, &catalog.tanks, false)?;

    let model_export = if options.data_only {
        None
    } else {
        let mut model_options = options.model_options.clone();
        model_options.game_dir = Some(resolved.game_dir);
        model_options.viewer_dir = output_dir.clone();
        Some(super::tanks::export_tank_catalog_from_discovered(
            &model_options,
            &resolved.resource_dir,
            catalog.model_tanks,
            discovery_seconds,
            started,
        )?)
    };

    Ok(ClientExportReport {
        output_dir,
        maps_file,
        tanks_file,
        map_count: catalog.maps.len(),
        tank_count: catalog.tanks.len(),
        data_only: options.data_only,
        model_export,
        elapsed_seconds: started.elapsed().as_secs_f64(),
    })
}

fn prepare_output_dir(path: &std::path::Path) -> Result<PathBuf> {
    fs::create_dir_all(path)
        .map_err(|error| crate::io_error("could not create export directory", path, error))?;
    path.canonicalize()
        .map_err(|error| crate::io_error("could not resolve export directory", path, error))
}
