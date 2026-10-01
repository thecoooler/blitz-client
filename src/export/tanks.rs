use super::armor as tank_armor;
use super::model as tank_model;
use super::output::{AssetStats, append_suffix, write_compressed_asset, write_json};
use crate::client::resolve_game_dir;
use crate::client::{ClientTank, discover_client_tanks};
use crate::resources::{crc32, decode_dvpl};
use crate::{Error, Result};
use serde::Serialize;
use serde_json::{Value as Json, json};
use std::collections::{HashMap, HashSet};
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::time::{Instant, SystemTime, UNIX_EPOCH};

const CATALOG_SOURCE: &str =
    "Installed client XML/item_defs/vehicles/*/list.xml and paired 3d/Tanks SC2/SCG assets";

#[derive(Debug, Clone)]
pub struct TankExportOptions {
    /// Game install directory or its `Data` directory. `None` searches defaults.
    pub game_dir: Option<PathBuf>,
    /// Viewer project root; generated assets are written under `assets/tanks`.
    pub viewer_dir: PathBuf,
    /// `0` means all tanks. Ignored when specific tank IDs are supplied.
    pub limit: usize,
    /// Tank IDs or `nation/ID` keys. May contain more than one tank.
    pub tanks: Vec<String>,
    /// Restricts automatic discovery to one nation.
    pub nation: Option<String>,
    /// `None` preserves source dimensions; otherwise a maximum width and height.
    pub max_texture_size: Option<(u32, u32)>,
    pub texture_format: String,
    pub webp_quality: u8,
    pub include_pbr: bool,
    pub compress: bool,
    pub force_textures: bool,
    /// Remove old tank bundles not exported by this invocation.
    pub prune: bool,
}

impl TankExportOptions {
    pub fn new(viewer_dir: impl Into<PathBuf>) -> Self {
        Self {
            game_dir: None,
            viewer_dir: viewer_dir.into(),
            limit: 0,
            tanks: Vec::new(),
            nation: None,
            max_texture_size: Some((1024, 1024)),
            texture_format: "webp".to_owned(),
            webp_quality: 90,
            include_pbr: false,
            compress: true,
            force_textures: false,
            prune: false,
        }
    }
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TankExportReport {
    pub catalog: PathBuf,
    pub export_report: PathBuf,
    pub discovered_client_tank_count: usize,
    pub exported_count: usize,
    pub failed_count: usize,
    pub failures: Vec<TankExportFailure>,
    pub entries: Vec<Json>,
    pub elapsed_seconds: f64,
}

#[derive(Debug, Clone, Serialize)]
pub struct TankExportFailure {
    pub id: String,
    pub error: String,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct ExportSummary<'a> {
    started_at: String,
    finished_at: String,
    data_root: String,
    discovered_client_tank_count: usize,
    discovery_seconds: f64,
    requested_limit: Json,
    max_texture_size: String,
    texture_format: &'a str,
    webp_quality: Option<u8>,
    pbr_enabled: bool,
    compression: &'a str,
    force_textures: bool,
    exported_count: usize,
    failed_count: usize,
    total_elapsed_seconds: f64,
    average_tank_seconds: Option<f64>,
    catalog: String,
    per_tank: Vec<Json>,
    failures: &'a [TankExportFailure],
}

/// Export client tank models into the existing viewer asset/catalog format.
///
/// Geometry decoding, transforms and binary bundle generation are implemented
/// in Rust. The image and Brotli codecs use the platform's `ffmpeg`, `cwebp`
/// and `brotli` executables; no Python runtime or scripts are used.
pub fn export_tank_catalog(options: &TankExportOptions) -> Result<TankExportReport> {
    validate_options(options)?;
    let total_started = Instant::now();
    let resolved = resolve_game_dir(options.game_dir.clone())?;
    let discovery_started = Instant::now();
    let discovered = discover_client_tanks(&resolved.resource_dir)?;
    let discovery_seconds = discovery_started.elapsed().as_secs_f64();
    export_tank_catalog_from_discovered(
        options,
        &resolved.resource_dir,
        discovered,
        discovery_seconds,
        total_started,
    )
}

pub(super) fn export_tank_catalog_from_discovered(
    options: &TankExportOptions,
    resource_dir: &Path,
    discovered: Vec<ClientTank>,
    discovery_seconds: f64,
    total_started: Instant,
) -> Result<TankExportReport> {
    validate_options(options)?;
    let ordered = select_tanks(&discovered, options)?;
    let requested_count = if options.tanks.is_empty() {
        if options.limit == 0 {
            None
        } else {
            Some(options.limit)
        }
    } else {
        Some(options.tanks.len())
    };

    let viewer_root = canonical_or_absolute(&options.viewer_dir)?;
    let output_root = viewer_root.join("assets/tanks");
    fs::create_dir_all(&output_root).map_err(|error| {
        crate::io_error(
            "could not create viewer asset directory",
            &output_root,
            error,
        )
    })?;

    let started_at = iso_timestamp(SystemTime::now());
    let mut entries = Vec::new();
    let mut per_tank = Vec::new();
    let mut failures = Vec::new();
    let mut attempted = HashSet::new();
    for tank in ordered {
        if requested_count.is_some_and(|target| entries.len() >= target) {
            break;
        }
        if !attempted.insert((tank.nation.clone(), tank.id.clone())) {
            continue;
        }
        match export_one(&tank, resource_dir, &output_root, options) {
            Ok((entry, measurement)) => {
                entries.push(entry);
                per_tank.push(measurement);
            }
            Err(error) => failures.push(TankExportFailure {
                id: format!("{}/{}", tank.nation, tank.id),
                error,
            }),
        }
    }

    if options.prune {
        prune_old_bundles(&output_root, &entries)?;
    }
    let exported_count = entries.len();
    let catalog_path = output_root.join("catalog.json");
    let report_path = output_root.join("export-report.json");
    let catalog = json!({
        "formatVersion": 1,
        "source": CATALOG_SOURCE,
        "generatedAt": started_at.clone(),
        "count": entries.len(),
        "entries": entries.clone(),
    });
    write_json(&catalog_path, &catalog, false)?;

    let elapsed_seconds = total_started.elapsed().as_secs_f64();
    let requested_limit = requested_count
        .map(Json::from)
        .unwrap_or_else(|| Json::String("all".to_owned()));
    let catalog_relative = catalog_path
        .strip_prefix(&viewer_root)
        .unwrap_or(&catalog_path)
        .to_string_lossy()
        .replace('\\', "/");
    let summary = ExportSummary {
        started_at,
        finished_at: iso_timestamp(SystemTime::now()),
        data_root: resource_dir.to_string_lossy().into_owned(),
        discovered_client_tank_count: discovered.len(),
        discovery_seconds,
        requested_limit,
        max_texture_size: texture_size_label(options.max_texture_size),
        texture_format: &options.texture_format,
        webp_quality: (options.texture_format == "webp").then_some(options.webp_quality),
        pbr_enabled: options.include_pbr,
        compression: if options.compress { "brotli-9" } else { "none" },
        force_textures: options.force_textures,
        exported_count: per_tank.len(),
        failed_count: failures.len(),
        total_elapsed_seconds: elapsed_seconds,
        average_tank_seconds: if per_tank.is_empty() {
            None
        } else {
            Some(
                per_tank
                    .iter()
                    .filter_map(|item| item.get("elapsedSeconds").and_then(Json::as_f64))
                    .sum::<f64>()
                    / per_tank.len() as f64,
            )
        },
        catalog: catalog_relative,
        per_tank,
        failures: &failures,
    };
    let summary_json = serde_json::to_value(&summary)
        .map_err(|error| Error::new("JSON_SERIALIZE_FAILED", error.to_string()))?;
    write_json(&report_path, &summary_json, false)?;

    let report = TankExportReport {
        catalog: catalog_path,
        export_report: report_path,
        discovered_client_tank_count: discovered.len(),
        exported_count,
        failed_count: failures.len(),
        failures: failures.clone(),
        entries,
        elapsed_seconds,
    };
    if let Some(target) = requested_count {
        if report.exported_count < target {
            return Err(Error::new(
                "TANK_EXPORT_INCOMPLETE",
                format!(
                    "exported {} of {target} requested tanks; see {}",
                    report.exported_count,
                    report.export_report.display()
                ),
            ));
        }
    }
    Ok(report)
}

fn validate_options(options: &TankExportOptions) -> Result<()> {
    if options.texture_format != "png" && options.texture_format != "webp" {
        return Err(Error::new("USAGE", "texture format must be webp or png"));
    }
    if options.webp_quality > 100 {
        return Err(Error::new("USAGE", "WebP quality must be from 0 to 100"));
    }
    if options
        .max_texture_size
        .is_some_and(|(width, height)| width == 0 || height == 0)
    {
        return Err(Error::new("USAGE", "texture dimensions must be positive"));
    }
    if let Some(nation) = &options.nation {
        if !crate::client::is_known_nation(nation) {
            return Err(Error::new("USAGE", format!("unknown nation {nation:?}")));
        }
    }
    Ok(())
}

fn select_tanks(discovered: &[ClientTank], options: &TankExportOptions) -> Result<Vec<ClientTank>> {
    if !options.tanks.is_empty() {
        return options
            .tanks
            .iter()
            .map(|requested| resolve_tank(discovered, requested))
            .collect();
    }
    let mut candidates: Vec<_> = discovered
        .iter()
        .filter(|tank| {
            options
                .nation
                .as_ref()
                .is_none_or(|nation| nation == &tank.nation)
        })
        .cloned()
        .collect();
    Ok(benchmark_order(&mut candidates))
}

fn resolve_tank(discovered: &[ClientTank], requested: &str) -> Result<ClientTank> {
    let (nation, id) = requested
        .split_once('/')
        .or_else(|| requested.split_once(':'))
        .map(|(nation, id)| (Some(nation.to_ascii_lowercase()), id))
        .unwrap_or((None, requested));
    if let Some(nation) = nation.as_deref() {
        if !crate::client::is_known_nation(nation) {
            return Err(Error::new(
                "TANK_NOT_FOUND",
                format!("unknown nation {nation:?}"),
            ));
        }
    }
    let matches: Vec<_> = discovered
        .iter()
        .filter(|tank| tank.id == id && nation.as_ref().is_none_or(|nation| nation == &tank.nation))
        .collect();
    match matches.as_slice() {
        [tank] => Ok((*tank).clone()),
        [] => Err(Error::new(
            "TANK_NOT_FOUND",
            format!("tank {requested:?} is not present with a model in this client"),
        )),
        many => Err(Error::new(
            "TANK_ID_AMBIGUOUS",
            format!(
                "tank ID {id:?} is ambiguous ({}); use nation/{id}",
                many.iter()
                    .map(|tank| tank.nation.as_str())
                    .collect::<Vec<_>>()
                    .join(", ")
            ),
        )),
    }
}

fn benchmark_order(candidates: &mut [ClientTank]) -> Vec<ClientTank> {
    let bz_index = candidates
        .iter()
        .position(|tank| tank.nation == "china" && tank.id == "Ch48_BZ_75");
    let mut bz75 = bz_index.map(|index| candidates[index].clone());
    let mut by_nation: HashMap<String, Vec<ClientTank>> = HashMap::new();
    for (index, tank) in candidates.iter().enumerate() {
        if Some(index) == bz_index {
            continue;
        }
        by_nation
            .entry(tank.nation.clone())
            .or_default()
            .push(tank.clone());
    }
    for tanks in by_nation.values_mut() {
        tanks.sort_by(|left, right| {
            right
                .level
                .as_deref()
                .unwrap_or("0")
                .parse::<i32>()
                .unwrap_or(0)
                .cmp(
                    &left
                        .level
                        .as_deref()
                        .unwrap_or("0")
                        .parse::<i32>()
                        .unwrap_or(0),
                )
                .then_with(|| {
                    left.id
                        .to_ascii_lowercase()
                        .cmp(&right.id.to_ascii_lowercase())
                })
        });
    }
    let mut ordered = Vec::new();
    if let Some(tank) = bz75.take() {
        ordered.push(tank);
    }
    let nations: Vec<_> = crate::client::nations()
        .iter()
        .map(|(nation, _, _)| *nation)
        .collect();
    loop {
        let mut moved = false;
        for nation in &nations {
            if let Some(group) = by_nation.get_mut(*nation) {
                if !group.is_empty() {
                    ordered.push(group.remove(0));
                    moved = true;
                }
            }
        }
        if !moved {
            break;
        }
    }
    ordered
}

fn export_one(
    tank: &ClientTank,
    data_root: &Path,
    output_root: &Path,
    options: &TankExportOptions,
) -> std::result::Result<(Json, Json), String> {
    let started = Instant::now();
    let bundle = output_root.join(&tank.nation).join(&tank.id);
    fs::create_dir_all(&bundle)
        .map_err(|error| format!("could not create {}: {error}", bundle.display()))?;
    let read_started = Instant::now();
    let sc2_data = read_decoded(&tank.model_files.scene)?;
    let scg_data = read_decoded(&tank.model_files.geometry)?;
    let read_seconds = read_started.elapsed().as_secs_f64();
    let geometry_started = Instant::now();
    let model = tank_model::export_model(
        &sc2_data,
        &scg_data,
        &tank.model_files.scene.to_string_lossy(),
        &tank.model_files.geometry.to_string_lossy(),
        &tank.id,
        options.include_pbr,
        &options.texture_format,
        options.webp_quality,
        options.compress,
    )?;
    let geometry_seconds = geometry_started.elapsed().as_secs_f64();

    let model_json_path = bundle.join("model.json");
    let model_json_bytes =
        serde_json::to_vec(&model.metadata).map_err(|error| error.to_string())?;
    let model_json_stats =
        write_compressed_asset(&model_json_path, &model_json_bytes, options.compress)
            .map_err(|error| error.to_string())?;
    let model_binary_stats =
        write_compressed_asset(&bundle.join("model.bin"), &model.binary, options.compress)
            .map_err(|error| error.to_string())?;
    let pbr_binary_stats = if let Some(data) = &model.pbr_binary {
        write_compressed_asset(&bundle.join("model.pbr.bin"), data, options.compress)
            .map_err(|error| error.to_string())?
    } else {
        remove_asset(&bundle.join("model.pbr.bin"));
        AssetStats::default()
    };
    let armor_started = Instant::now();
    let armor_path = bundle.join("armor.json");
    let armor_mesh_path = bundle.join("armor-mesh.json");
    remove_asset(&armor_path);
    remove_asset(&armor_mesh_path);
    let collision_sc2 = crate::client::resolve_client_asset(
        data_root,
        &format!(
            "3d/Tanks/CollisionMeshes/{}-{}.sc2.dvpl",
            tank.nation, tank.id
        ),
    );
    let collision_scg = crate::client::resolve_client_asset(
        data_root,
        &format!(
            "3d/Tanks/CollisionMeshes/{}-{}.scg.dvpl",
            tank.nation, tank.id
        ),
    );
    let vehicle_xml = crate::client::resolve_client_asset(
        data_root,
        &format!(
            "XML/item_defs/vehicles/{}/{}.xml.dvpl",
            tank.nation, tank.id
        ),
    );
    let mut has_armor = false;
    let mut armor_note = None;
    let mut armor_stats = AssetStats::default();
    let mut armor_mesh_stats = AssetStats::default();
    if let (Some(collision_sc2), Some(collision_scg), Some(vehicle_xml)) =
        (collision_sc2, collision_scg, vehicle_xml)
    {
        let armor_result = (|| {
            let result = tank_armor::export_armor(
                &read_decoded(&collision_sc2)?,
                &read_decoded(&collision_scg)?,
                &read_decoded(&vehicle_xml)?,
                &tank.nation,
                &tank.id,
                &collision_sc2.to_string_lossy(),
                &collision_scg.to_string_lossy(),
                &vehicle_xml.to_string_lossy(),
            )?;
            let armor_json =
                serde_json::to_vec_pretty(&result.values).map_err(|error| error.to_string())?;
            let armor_mesh_json =
                serde_json::to_vec_pretty(&result.mesh).map_err(|error| error.to_string())?;
            let armor_stats = write_compressed_asset(&armor_path, &armor_json, options.compress)
                .map_err(|error| error.to_string())?;
            let armor_mesh_stats =
                write_compressed_asset(&armor_mesh_path, &armor_mesh_json, options.compress)
                    .map_err(|error| error.to_string())?;
            Ok::<_, String>((
                armor_stats,
                armor_mesh_stats,
                result.face_count,
                result.vertex_count,
            ))
        })();
        match armor_result {
            Ok((values_stats, mesh_stats, _, _)) => {
                has_armor = true;
                armor_stats = values_stats;
                armor_mesh_stats = mesh_stats;
            }
            Err(error) => armor_note = Some(error),
        }
    }
    let armor_seconds = armor_started.elapsed().as_secs_f64();

    let texture_started = Instant::now();
    let model_directory = tank
        .model_files
        .scene
        .parent()
        .ok_or_else(|| "tank scene path has no parent directory".to_owned())?;
    let mut seen_textures = HashSet::new();
    let mut texture_conversions = 0usize;
    let mut placeholder_textures = 0usize;
    for texture in &model.textures {
        if !seen_textures.insert(texture.file.clone()) {
            continue;
        }
        let target = bundle.join(&texture.file);
        if texture.source.is_none() {
            write_fallback_png(&target, &texture.role)?;
            continue;
        }
        let relative_source = texture.source.as_ref().unwrap();
        let source = if relative_source.is_absolute() {
            relative_source.clone()
        } else {
            model_directory.join(relative_source)
        };
        if !source.is_file() {
            // Some client maps ship only as PVRTC variants, which ffmpeg cannot
            // decode; emit the placeholder color in the requested format so the
            // file still matches the extension the model declares.
            placeholder_textures += 1;
            let temp = TempDirectory::new("blitz-tank-texture")?;
            let placeholder = temp.path.join("placeholder.png");
            write_fallback_png(&placeholder, &texture.role)?;
            convert_texture(
                &placeholder,
                &target,
                options.max_texture_size,
                &options.texture_format,
                options.webp_quality,
                texture.lossless,
            )?;
            continue;
        }
        convert_texture(
            &source,
            &target,
            options.max_texture_size,
            &options.texture_format,
            options.webp_quality,
            texture.lossless,
        )?;
        texture_conversions += 1;
    }
    let texture_seconds = texture_started.elapsed().as_secs_f64();

    let elapsed = started.elapsed().as_secs_f64();
    let bundle_url = format!("./assets/tanks/{}/{}/", tank.nation, tank.id);
    let suffix = if options.compress { ".br" } else { "" };
    let fallback_maps = model.fallback_maps + placeholder_textures;
    let entry = json!({
        "id": format!("{}/{}", tank.nation, tank.id),
        "name": tank.name,
        "nation": tank.nation_label,
        "level": tank.level,
        "userString": tank.user_string,
        "bundle": bundle_url,
        "model": format!("{bundle_url}model.json{suffix}"),
        "armor": if has_armor { Json::String(format!("{bundle_url}armor.json{suffix}")) } else { Json::Null },
        "armorMesh": if has_armor { Json::String(format!("{bundle_url}armor-mesh.json{suffix}")) } else { Json::Null },
        "hasArmor": has_armor,
        "pbrEnabled": options.include_pbr,
        "compression": if options.compress { "brotli-9" } else { "none" },
        "armorNote": armor_note,
        "armorJsonBytes": armor_stats.stored_bytes,
        "armorJsonUncompressedBytes": armor_stats.uncompressed_bytes,
        "armorMeshBytes": armor_mesh_stats.stored_bytes,
        "armorMeshUncompressedBytes": armor_mesh_stats.uncompressed_bytes,
        "triangles": model.triangle_count,
        "vertices": model.vertex_count,
        "materials": model.material_count,
        "fallbackMaps": fallback_maps,
        "generatedUvGroups": model.generated_uv_groups,
        "modelJsonBytes": model_json_stats.stored_bytes,
        "modelJsonUncompressedBytes": model_json_stats.uncompressed_bytes,
        "binaryGeometryBytes": model_binary_stats.stored_bytes,
        "binaryGeometryUncompressedBytes": model_binary_stats.uncompressed_bytes,
        "pbrBinaryGeometryBytes": pbr_binary_stats.stored_bytes,
        "pbrBinaryGeometryUncompressedBytes": pbr_binary_stats.uncompressed_bytes,
        "totalBinaryGeometryBytes": model_binary_stats.stored_bytes + pbr_binary_stats.stored_bytes,
        "totalBinaryGeometryUncompressedBytes": model_binary_stats.uncompressed_bytes + pbr_binary_stats.uncompressed_bytes,
        "textureCacheHits": 0,
        "textureConversions": texture_conversions,
        "textureFormat": options.texture_format,
        "webpQuality": if options.texture_format == "webp" { Json::from(options.webp_quality) } else { Json::Null },
        "elapsedSeconds": round_millis(elapsed),
        "geometrySeconds": round_millis(geometry_seconds),
        "texturesSeconds": round_millis(texture_seconds),
        "readModelSeconds": round_millis(read_seconds),
        "armorSeconds": round_millis(armor_seconds),
    });
    let measurement = json!({
        "id": entry["id"],
        "elapsedSeconds": round_millis(elapsed),
        "geometrySeconds": round_millis(geometry_seconds),
        "texturesSeconds": round_millis(texture_seconds),
        "readModelSeconds": round_millis(read_seconds),
        "armorSeconds": round_millis(armor_seconds),
        "vertices": model.vertex_count,
        "triangles": model.triangle_count,
        "materials": model.material_count,
        "fallbackMaps": fallback_maps,
        "generatedUvGroups": model.generated_uv_groups,
        "modelJsonBytes": model_json_stats.stored_bytes,
        "modelJsonUncompressedBytes": model_json_stats.uncompressed_bytes,
        "binaryGeometryBytes": model_binary_stats.stored_bytes,
        "binaryGeometryUncompressedBytes": model_binary_stats.uncompressed_bytes,
        "pbrBinaryGeometryBytes": pbr_binary_stats.stored_bytes,
        "pbrBinaryGeometryUncompressedBytes": pbr_binary_stats.uncompressed_bytes,
        "totalBinaryGeometryBytes": model_binary_stats.stored_bytes + pbr_binary_stats.stored_bytes,
        "totalBinaryGeometryUncompressedBytes": model_binary_stats.uncompressed_bytes + pbr_binary_stats.uncompressed_bytes,
        "textureCacheHits": 0,
        "textureConversions": texture_conversions,
        "textureFormat": options.texture_format,
        "webpQuality": if options.texture_format == "webp" { Json::from(options.webp_quality) } else { Json::Null },
        "hasArmor": has_armor,
        "pbrEnabled": options.include_pbr,
        "compression": if options.compress { "brotli-9" } else { "none" },
        "armorJsonBytes": armor_stats.stored_bytes,
        "armorJsonUncompressedBytes": armor_stats.uncompressed_bytes,
        "armorMeshBytes": armor_mesh_stats.stored_bytes,
        "armorMeshUncompressedBytes": armor_mesh_stats.uncompressed_bytes,
        "armorError": armor_note,
    });
    Ok((entry, measurement))
}

fn read_decoded(path: &Path) -> std::result::Result<Vec<u8>, String> {
    let bytes =
        fs::read(path).map_err(|error| format!("could not read {}: {error}", path.display()))?;
    decode_dvpl(&bytes)
        .map(|(plain, _)| plain)
        .map_err(|error| format!("could not decode {}: {error}", path.display()))
}

fn convert_texture(
    source: &Path,
    destination: &Path,
    max_size: Option<(u32, u32)>,
    format: &str,
    quality: u8,
    lossless: bool,
) -> std::result::Result<(), String> {
    if let Some(parent) = destination.parent() {
        fs::create_dir_all(parent)
            .map_err(|error| format!("could not create {}: {error}", parent.display()))?;
    }
    let source_bytes = read_decoded(source)?;
    let temp = TempDirectory::new("blitz-tank-texture")?;
    let dds = temp.path.join("source.dds");
    let png = temp.path.join("decoded.png");
    fs::write(&dds, source_bytes)
        .map_err(|error| format!("could not stage DDS {}: {error}", source.display()))?;
    let mut command = Command::new("ffmpeg");
    command.args(["-hide_banner", "-v", "error", "-y", "-i"]);
    command.arg(&dds);
    if let Some((width, height)) = max_size {
        command.arg("-vf").arg(format!(
            "scale=w='min(iw,{width})':h='min(ih,{height})':force_original_aspect_ratio=decrease"
        ));
    }
    command.args(["-frames:v", "1", "-update", "1", "-compression_level", "9"]);
    command.arg(&png);
    let output = command
        .output()
        .map_err(|error| format!("ffmpeg is required to decode DDS textures: {error}"))?;
    if !output.status.success() {
        return Err(format!(
            "ffmpeg failed to decode {}: {}",
            source.display(),
            String::from_utf8_lossy(&output.stderr).trim()
        ));
    }
    if format == "png" {
        if let Some(parent) = destination.parent() {
            fs::create_dir_all(parent)
                .map_err(|error| format!("could not create {}: {error}", parent.display()))?;
        }
        fs::copy(&png, destination)
            .map_err(|error| format!("could not write {}: {error}", destination.display()))?;
    } else {
        let mut command = Command::new("cwebp");
        command.args(["-m", "6", "-alpha_q", "100", "-exact", "-metadata", "none"]);
        if lossless {
            command.args(["-lossless", "-z", "9"]);
        } else {
            command.args(["-q", &quality.to_string()]);
        }
        command.arg(&png).arg("-o").arg(destination);
        let output = command
            .output()
            .map_err(|error| format!("cwebp is required to export WebP textures: {error}"))?;
        if !output.status.success() {
            return Err(format!(
                "cwebp failed to encode {}: {}",
                source.display(),
                String::from_utf8_lossy(&output.stderr).trim()
            ));
        }
    }
    Ok(())
}

fn write_fallback_png(path: &Path, role: &str) -> std::result::Result<(), String> {
    let rgba = match role {
        "baseColor" => [151, 156, 141, 255],
        "normal" => [128, 128, 128, 128],
        "roughnessMetallic" => [0, 0, 0, 204],
        "misc" => [0, 0, 0, 255],
        _ => return Err(format!("unknown fallback texture role {role}")),
    };
    let mut png = b"\x89PNG\r\n\x1a\n".to_vec();
    let mut header = Vec::new();
    header.extend_from_slice(&1u32.to_be_bytes());
    header.extend_from_slice(&1u32.to_be_bytes());
    header.extend_from_slice(&[8, 6, 0, 0, 0]);
    png_chunk(&mut png, b"IHDR", &header);
    let mut raw = vec![0];
    raw.extend_from_slice(&rgba);
    let compressed = miniz_oxide::deflate::compress_to_vec_zlib(&raw, 9);
    png_chunk(&mut png, b"IDAT", &compressed);
    png_chunk(&mut png, b"IEND", &[]);
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|error| format!("could not create {}: {error}", parent.display()))?;
    }
    fs::write(path, png).map_err(|error| {
        format!(
            "could not write fallback texture {}: {error}",
            path.display()
        )
    })
}

fn png_chunk(output: &mut Vec<u8>, kind: &[u8; 4], data: &[u8]) {
    output.extend_from_slice(&(data.len() as u32).to_be_bytes());
    output.extend_from_slice(kind);
    output.extend_from_slice(data);
    let mut crc_input = Vec::with_capacity(4 + data.len());
    crc_input.extend_from_slice(kind);
    crc_input.extend_from_slice(data);
    output.extend_from_slice(&crc32(&crc_input).to_be_bytes());
}

fn remove_asset(path: &Path) {
    let _ = fs::remove_file(path);
    let _ = fs::remove_file(append_suffix(path, ".br"));
}

fn prune_old_bundles(output_root: &Path, entries: &[Json]) -> Result<()> {
    let included: HashSet<String> = entries
        .iter()
        .filter_map(|entry| entry.get("id").and_then(Json::as_str).map(str::to_owned))
        .collect();
    for (nation, _, _) in crate::client::nations() {
        let nation_dir = output_root.join(nation);
        let Ok(children) = fs::read_dir(&nation_dir) else {
            continue;
        };
        for child in children.flatten() {
            if !child.file_type().map(|kind| kind.is_dir()).unwrap_or(false) {
                continue;
            }
            let id = child.file_name().to_string_lossy().into_owned();
            if !included.contains(&format!("{nation}/{id}")) {
                fs::remove_dir_all(child.path()).map_err(|error| {
                    crate::io_error("could not prune stale tank bundle", &child.path(), error)
                })?;
            }
        }
    }
    Ok(())
}

fn canonical_or_absolute(path: &Path) -> Result<PathBuf> {
    if path.exists() {
        return path
            .canonicalize()
            .map_err(|error| crate::io_error("could not resolve viewer directory", path, error));
    }
    let absolute = if path.is_absolute() {
        path.to_path_buf()
    } else {
        std::env::current_dir()
            .map_err(|error| Error::new("IO_ERROR", error.to_string()))?
            .join(path)
    };
    Ok(absolute)
}

fn max_texture_size_label(size: Option<(u32, u32)>) -> String {
    size.map(|(width, height)| format!("{width}x{height}"))
        .unwrap_or_else(|| "original".to_owned())
}

fn texture_size_label(size: Option<(u32, u32)>) -> String {
    max_texture_size_label(size)
}
fn round_millis(value: f64) -> f64 {
    (value * 1000.0).round() / 1000.0
}

fn iso_timestamp(time: SystemTime) -> String {
    let seconds = time
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs() as i64;
    let days = seconds.div_euclid(86_400);
    let day_seconds = seconds.rem_euclid(86_400);
    let (year, month, day) = civil_from_days(days);
    format!(
        "{year:04}-{month:02}-{day:02}T{:02}:{:02}:{:02}+00:00",
        day_seconds / 3600,
        day_seconds / 60 % 60,
        day_seconds % 60
    )
}

fn civil_from_days(days_since_epoch: i64) -> (i64, i64, i64) {
    let z = days_since_epoch + 719_468;
    let era = if z >= 0 { z } else { z - 146_096 }.div_euclid(146_097);
    let day_of_era = z - era * 146_097;
    let year_of_era =
        (day_of_era - day_of_era / 1_460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
    let mut year = year_of_era + era * 400;
    let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let month_prime = (5 * day_of_year + 2) / 153;
    let day = day_of_year - (153 * month_prime + 2) / 5 + 1;
    let month = month_prime + if month_prime < 10 { 3 } else { -9 };
    year += i64::from(month <= 2);
    (year, month, day)
}

struct TempDirectory {
    path: PathBuf,
}

impl TempDirectory {
    fn new(prefix: &str) -> std::result::Result<Self, String> {
        let stamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_nanos();
        for attempt in 0..100u32 {
            let path = std::env::temp_dir()
                .join(format!("{prefix}-{}-{stamp}-{attempt}", std::process::id()));
            match fs::create_dir(&path) {
                Ok(()) => return Ok(Self { path }),
                Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => continue,
                Err(error) => return Err(format!("could not create temporary directory: {error}")),
            }
        }
        Err("could not allocate a unique temporary directory".to_owned())
    }
}

impl Drop for TempDirectory {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.path);
    }
}
