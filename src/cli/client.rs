use super::{print_json, take_game_dir, usage_error};
use blitz_client::{
    client::{is_known_nation, list_client_tanks, resolve_game_dir},
    export::{
        ClientExportOptions, ClientFilesExportOptions, TankExportOptions, export_client,
        export_client_files_with_progress, export_tank_catalog,
    },
};
use std::path::PathBuf;

pub(super) fn run(args: &[String], json: bool) -> blitz_client::Result<bool> {
    if args.first().map(String::as_str) == Some("locate") {
        let (positional, game_dir) = take_game_dir(&args[1..])?;
        if !positional.is_empty() {
            return Err(usage_error("client locate accepts no positional arguments"));
        }
        return resolve_game_dir(game_dir).map(|resolved| {
            if json {
                print_json(&resolved);
            } else {
                println!("{}", resolved.game_dir.display());
                println!("resource directory: {}", resolved.resource_dir.display());
                if let Some(cache_dir) = &resolved.cache_dir {
                    println!("cache directory: {}", cache_dir.display());
                }
                if let Some(replay_dir) = &resolved.replay_dir {
                    println!("replay directory: {}", replay_dir.display());
                }
            }
            true
        });
    }
    match args {
        [group, action, rest @ ..] if group == "files" && action == "export" => {
            run_files_export(rest, json)
        }
        [action, rest @ ..] if action == "export" => run_client_export(rest, json),
        [group, action, rest @ ..] if group == "tanks" && action == "list" => {
            run_tank_list(rest, json)
        }
        [group, action, rest @ ..] if group == "tanks" && action == "export" => {
            run_tank_export(rest, json)
        }
        _ => Err(usage_error(
            "expected client locate, client export, client files export, or client tanks list/export",
        )),
    }
}

fn run_files_export(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let mut options = ClientFilesExportOptions::new(".");
    let mut output_dir = None;
    let mut types = Vec::new();
    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "--game-dir" | "--cache-dir" | "--output-dir" => {
                let flag = &args[i];
                i += 1;
                let path = PathBuf::from(
                    args.get(i)
                        .ok_or_else(|| usage_error(&format!("{flag} requires a path")))?,
                );
                match flag.as_str() {
                    "--game-dir" => options.game_dir = Some(path),
                    "--cache-dir" => options.cache_dir = Some(path),
                    _ => output_dir = Some(path),
                }
            }
            "--types" => {
                i += 1;
                let value = args
                    .get(i)
                    .ok_or_else(|| usage_error("--types requires comma-separated file types"))?;
                for name in value.split(',') {
                    let kind = name
                        .parse()
                        .map_err(|error: blitz_client::Error| usage_error(&error.message))?;
                    if !types.contains(&kind) {
                        types.push(kind);
                    }
                }
            }
            "--no-cache" => options.no_cache = true,
            "--dry-run" => options.dry_run = true,
            flag if flag.starts_with('-') => {
                return Err(usage_error(&format!("unknown option: {flag}")));
            }
            value => return Err(usage_error(&format!("unexpected argument: {value}"))),
        }
        i += 1;
    }
    if options.no_cache && options.cache_dir.is_some() {
        return Err(usage_error("--cache-dir and --no-cache cannot be combined"));
    }
    options.output_dir = output_dir
        .ok_or_else(|| usage_error("client files export requires --output-dir <path>"))?;
    if !types.is_empty() {
        options.types = types;
    }
    let report = export_client_files_with_progress(&options, |report| {
        let processed = report.written_count + report.unchanged_count;
        if !json
            && !report.dry_run
            && (processed == 0 || processed % 1000 == 0 || processed == report.file_count)
        {
            eprintln!(
                "Files: {processed}/{}; written: {}, unchanged: {}",
                report.file_count, report.written_count, report.unchanged_count
            );
        }
        Ok(())
    })?;
    if json {
        print_json(&report);
    } else if report.dry_run {
        println!(
            "Dry run: {} file(s), {} bytes ({:.2} MiB) on disk",
            report.file_count,
            report.source_bytes,
            report.source_bytes as f64 / 1_048_576.0
        );
        println!(
            "Selected {} DVPL file(s); {} file(s) from cache",
            report.dvpl_count, report.cache_file_count
        );
    } else {
        println!(
            "Exported {} file(s): {} written, {} unchanged",
            report.file_count, report.written_count, report.unchanged_count
        );
        println!(
            "Unpacked {} DVPL file(s); {} file(s) from cache",
            report.dvpl_count, report.cache_file_count
        );
        println!("Output: {}", report.output_dir.display());
    }
    Ok(true)
}

fn run_tank_list(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let mut positional = Vec::new();
    let mut game_dir = None;
    let mut nation = None;
    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "--game-dir" => {
                i += 1;
                game_dir = args.get(i).map(PathBuf::from);
                if game_dir.is_none() {
                    return Err(usage_error("--game-dir requires a path"));
                }
            }
            "--nation" => {
                i += 1;
                nation = Some(
                    args.get(i)
                        .ok_or_else(|| usage_error("--nation requires a nation name"))?
                        .to_ascii_lowercase(),
                );
            }
            flag if flag.starts_with('-') => {
                return Err(usage_error(&format!("unknown option: {flag}")));
            }
            value => positional.push(value.to_owned()),
        }
        i += 1;
    }
    if !positional.is_empty() {
        return Err(usage_error(
            "client tanks list accepts no positional arguments",
        ));
    }
    if nation
        .as_deref()
        .is_some_and(|value| !is_known_nation(value))
    {
        return Err(usage_error(&format!(
            "unknown nation {:?}",
            nation.unwrap()
        )));
    }
    let mut tanks = list_client_tanks(game_dir)?;
    if let Some(nation) = nation {
        tanks.retain(|tank| tank.nation == nation);
    }
    if json {
        print_json(&serde_json::json!({ "count": tanks.len(), "tanks": tanks }));
    } else {
        for tank in &tanks {
            println!(
                "{}\t{}\t{}\t{}",
                tank.nation,
                tank.id,
                tank.level.as_deref().unwrap_or("?"),
                tank.name
            );
        }
        println!("{} tanks", tanks.len());
    }
    Ok(true)
}

fn run_tank_export(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let (mut options, viewer_dir, _) = parse_model_export_options(args, false)?;
    options.viewer_dir = viewer_dir
        .ok_or_else(|| usage_error("client tanks export requires --viewer-dir <path>"))?;
    let report = export_tank_catalog(&options)?;
    if json {
        print_json(&report);
    } else {
        println!(
            "Exported {} tank model(s); {} failed",
            report.exported_count, report.failed_count
        );
        println!("catalog: {}", report.catalog.display());
        println!("report: {}", report.export_report.display());
        for failure in report.failures {
            eprintln!("FAIL {}: {}", failure.id, failure.error);
        }
    }
    Ok(true)
}

fn run_client_export(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let (model_options, output_dir, data_only) = parse_model_export_options(args, true)?;
    let mut options = ClientExportOptions::new(
        output_dir.ok_or_else(|| usage_error("client export requires --output-dir <path>"))?,
    );
    options.game_dir = model_options.game_dir.clone();
    options.model_options = model_options;
    options.data_only = data_only;
    let report = export_client(&options)?;
    if json {
        print_json(&report);
    } else {
        println!(
            "Exported {} map(s) and {} tank record(s)",
            report.map_count, report.tank_count
        );
        println!("maps: {}", report.maps_file.display());
        println!("tanks: {}", report.tanks_file.display());
        if let Some(models) = report.model_export {
            println!(
                "Exported {} tank model(s); {} failed",
                models.exported_count, models.failed_count
            );
            println!("model catalog: {}", models.catalog.display());
            println!("model report: {}", models.export_report.display());
            for failure in models.failures {
                eprintln!("FAIL {}: {}", failure.id, failure.error);
            }
        }
    }
    Ok(true)
}

fn parse_model_export_options(
    args: &[String],
    unified: bool,
) -> blitz_client::Result<(TankExportOptions, Option<PathBuf>, bool)> {
    let mut options = TankExportOptions::new(".");
    let mut output_dir = None;
    let mut data_only = false;
    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "--game-dir" => {
                i += 1;
                options.game_dir = args.get(i).map(PathBuf::from);
                if options.game_dir.is_none() {
                    return Err(usage_error("--game-dir requires a path"));
                }
            }
            "--viewer-dir" if !unified => {
                i += 1;
                output_dir = args.get(i).map(PathBuf::from);
                if output_dir.is_none() {
                    return Err(usage_error("--viewer-dir requires a path"));
                }
            }
            "--output-dir" if unified => {
                i += 1;
                output_dir = args.get(i).map(PathBuf::from);
                if output_dir.is_none() {
                    return Err(usage_error("--output-dir requires a path"));
                }
            }
            "--data-only" if unified => data_only = true,
            "--tank" => {
                i += 1;
                options.tanks.push(
                    args.get(i)
                        .cloned()
                        .ok_or_else(|| usage_error("--tank requires an ID"))?,
                );
            }
            "--nation" => {
                i += 1;
                options.nation = Some(
                    args.get(i)
                        .cloned()
                        .ok_or_else(|| usage_error("--nation requires a name"))?
                        .to_ascii_lowercase(),
                );
            }
            "--limit" => {
                i += 1;
                let value = args
                    .get(i)
                    .ok_or_else(|| usage_error("--limit requires a number"))?;
                options.limit = value
                    .parse::<usize>()
                    .map_err(|_| usage_error("--limit must be a non-negative integer"))?;
            }
            "--max-texture-size" => {
                i += 1;
                options.max_texture_size = parse_texture_size(
                    args.get(i)
                        .ok_or_else(|| usage_error("--max-texture-size requires a size"))?,
                )?;
            }
            "--texture-format" => {
                i += 1;
                options.texture_format = args
                    .get(i)
                    .cloned()
                    .ok_or_else(|| usage_error("--texture-format requires webp or png"))?;
            }
            "--webp-quality" => {
                i += 1;
                options.webp_quality = args
                    .get(i)
                    .ok_or_else(|| usage_error("--webp-quality requires a number"))?
                    .parse::<u8>()
                    .map_err(|_| usage_error("--webp-quality must be from 0 to 100"))?;
            }
            "--pbr" => options.include_pbr = true,
            "--no-compress" => options.compress = false,
            "--force" => options.force_textures = true,
            "--prune" => options.prune = true,
            flag if flag.starts_with('-') => {
                return Err(usage_error(&format!("unknown option: {flag}")));
            }
            value => return Err(usage_error(&format!("unexpected argument: {value}"))),
        }
        i += 1;
    }
    Ok((options, output_dir, data_only))
}

fn parse_texture_size(value: &str) -> blitz_client::Result<Option<(u32, u32)>> {
    let normalized = value.trim().to_ascii_lowercase().replace('×', "x");
    if normalized == "original" {
        return Ok(None);
    }
    let dimensions = if let Some((width, height)) = normalized.split_once('x') {
        let width = width
            .parse::<u32>()
            .map_err(|_| usage_error("texture width must be a positive integer"))?;
        let height = height
            .parse::<u32>()
            .map_err(|_| usage_error("texture height must be a positive integer"))?;
        (width, height)
    } else {
        let width = normalized
            .parse::<u32>()
            .map_err(|_| usage_error("texture size must be a positive integer"))?;
        (width, width)
    };
    match dimensions {
        (width, height) if width > 0 && height > 0 => Ok(Some((width, height))),
        _ => Err(usage_error(
            "texture size must be N or WIDTHxHEIGHT with positive dimensions",
        )),
    }
}
