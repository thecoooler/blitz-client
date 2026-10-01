use blitz_client::{
    ClientExportOptions, InstallerError, ReplayAnalysis, ReplayFollower, ReplayOptions,
    TankExportOptions, analyze_replay, export_client, export_tank_catalog, find_latest_replay,
    install_mod, is_known_nation, list_client_tanks, remove_mod, resolve_game_dir,
    update_from_git_diff, validate_mod,
};
use std::env;
use std::path::PathBuf;
use std::process;

fn main() {
    process::exit(run());
}

fn run() -> i32 {
    let mut args = env::args().skip(1).collect::<Vec<_>>();
    if args.is_empty() || args.iter().any(|arg| arg == "--help" || arg == "-h") {
        print_usage();
        return 0;
    }

    let json = args.iter().any(|arg| arg == "--json");
    args.retain(|arg| arg != "--json");
    let result = match args[0].as_str() {
        "client" => run_client(&args[1..], json),
        "mod" => run_mod(&args[1..], json),
        "replay" => run_replay(&args[1..], json),
        command => Err(usage_error(&format!("unknown command: {command}"))),
    };

    match result {
        Ok(true) => 0,
        Ok(false) => 4,
        Err(error) => {
            if json && !(args[0] == "replay" && args.get(1).is_some_and(|arg| arg == "follow")) {
                print_json(&serde_json::json!({"ok": false, "error": error}));
            } else {
                eprintln!("{error}");
            }
            2
        }
    }
}

fn run_client(args: &[String], json: bool) -> blitz_client::Result<bool> {
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
            }
            true
        });
    }
    match args {
        [action, rest @ ..] if action == "export" => run_client_export(rest, json),
        [group, action, rest @ ..] if group == "tanks" && action == "list" => {
            run_tank_list(rest, json)
        }
        [group, action, rest @ ..] if group == "tanks" && action == "export" => {
            run_tank_export(rest, json)
        }
        _ => Err(usage_error(
            "expected client locate, client export, or client tanks list/export",
        )),
    }
}

fn run_mod(args: &[String], json: bool) -> blitz_client::Result<bool> {
    match args.first().map(String::as_str) {
        Some("validate") => run_validate(&args[1..], json),
        Some("install") => run_install(&args[1..], json),
        Some("uninstall") => run_uninstall(&args[1..], json),
        Some("update-from-git") => run_update(&args[1..], json),
        _ => Err(usage_error(
            "expected mod validate/install/uninstall/update-from-git",
        )),
    }
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

fn run_validate(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let (positional, game_dir) = take_game_dir(args)?;
    if positional.len() != 1 {
        return Err(usage_error(
            "validate requires a mod directory or mod.toml path",
        ));
    }
    let report = validate_mod(&positional[0], game_dir)?;
    if json {
        print_json(&report);
    } else {
        println!("{} — {}", report.mod_id, report.mod_name);
        for file in &report.files {
            if file.valid {
                println!(
                    "OK   {} -> {}",
                    file.path,
                    file.target
                        .as_ref()
                        .map(|p| p.display().to_string())
                        .unwrap_or_default()
                );
            } else {
                println!(
                    "FAIL {} [{}] {}",
                    file.path,
                    file.error_code.as_deref().unwrap_or("ERROR"),
                    file.message.as_deref().unwrap_or("")
                );
            }
        }
        println!(
            "{}",
            if report.valid {
                "Validation passed"
            } else {
                "Validation failed"
            }
        );
    }
    Ok(report.valid)
}

fn run_install(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let (positional, game_dir) = take_game_dir(args)?;
    if positional.len() != 1 {
        return Err(usage_error(
            "install requires a mod directory or mod.toml path",
        ));
    }
    let report = install_mod(&positional[0], game_dir)?;
    if json {
        print_json(&report);
    } else if report.already_installed {
        println!("{} is already installed", report.mod_id);
    } else {
        println!("Installed {} — {}", report.mod_id, report.mod_name);
        for target in report.targets {
            println!("  {}", target.display());
        }
    }
    Ok(true)
}

fn run_uninstall(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let (positional, game_dir) = take_game_dir(args)?;
    if positional.len() != 1 {
        return Err(usage_error("uninstall requires an installed mod id"));
    }
    let report = remove_mod(&positional[0], game_dir)?;
    if json {
        print_json(&report);
    } else {
        println!("Removed {}", report.mod_id);
        for target in report.restored_files {
            println!("  restored {}", target.display());
        }
    }
    Ok(true)
}

fn run_update(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let mut positional = Vec::new();
    let mut repo = None;
    let mut base = None;
    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "--repo" => {
                i += 1;
                repo = args.get(i).map(PathBuf::from);
                if repo.is_none() {
                    return Err(usage_error("--repo requires a path"));
                }
            }
            "--base" => {
                i += 1;
                base = args.get(i).cloned();
                if base.is_none() {
                    return Err(usage_error("--base requires a revision"));
                }
            }
            flag if flag.starts_with('-') => {
                return Err(usage_error(&format!("unknown option: {flag}")));
            }
            value => positional.push(value.to_owned()),
        }
        i += 1;
    }
    if positional.len() != 1 {
        return Err(usage_error(
            "update-from-git requires a mod directory or mod.toml path",
        ));
    }
    let repo = repo.ok_or_else(|| usage_error("update-from-git requires --repo <path>"))?;
    let manifest = update_from_git_diff(&positional[0], repo, base.as_deref())?;
    if json {
        print_json(&manifest);
    } else {
        let generated = manifest
            .files
            .iter()
            .filter(|file| file.generated_by.as_deref() == Some("git"))
            .count();
        println!(
            "Updated {}: {generated} Git patch file(s) in mod.toml",
            manifest.id
        );
    }
    Ok(true)
}

fn run_replay(args: &[String], json: bool) -> blitz_client::Result<bool> {
    match args.first().map(String::as_str) {
        Some("inspect") => run_replay_analysis(&args[1..], json, false),
        Some("latest") => run_replay_analysis(&args[1..], json, true),
        Some("follow") => run_replay_follow(&args[1..]),
        _ => Err(usage_error(
            "expected replay inspect, replay latest or replay follow",
        )),
    }
}

fn run_replay_follow(args: &[String]) -> blitz_client::Result<bool> {
    use std::io::Write;
    use std::time::{Duration, Instant};
    let mut path = None;
    let mut out = None;
    let mut poll_ms = 25_u64;
    let mut idle_timeout = None;
    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "--poll-ms" | "--idle-timeout-ms" => {
                let flag = &args[i];
                i += 1;
                let value = args
                    .get(i)
                    .and_then(|value| value.parse::<u64>().ok())
                    .filter(|value| *value > 0 && *value <= 86_400_000)
                    .ok_or_else(|| {
                        usage_error(&format!(
                            "{flag} requires milliseconds between 1 and 86400000"
                        ))
                    })?;
                if flag == "--poll-ms" {
                    poll_ms = value;
                } else {
                    idle_timeout = Some(Duration::from_millis(value));
                }
            }
            "--out" => {
                i += 1;
                out = Some(PathBuf::from(
                    args.get(i)
                        .ok_or_else(|| usage_error("--out requires a path"))?,
                ));
            }
            flag if flag.starts_with('-') => {
                return Err(usage_error(&format!("unknown option: {flag}")));
            }
            value if path.is_none() => path = Some(PathBuf::from(value)),
            value => return Err(usage_error(&format!("unexpected argument: {value}"))),
        }
        i += 1;
    }
    let path = path.ok_or_else(|| usage_error("replay follow requires a data.replay path"))?;
    if let Some(out) = &out {
        let resolve = |value: &std::path::Path| -> blitz_client::Result<PathBuf> {
            let parent = value
                .parent()
                .filter(|parent| !parent.as_os_str().is_empty())
                .unwrap_or_else(|| std::path::Path::new("."));
            let parent = parent.canonicalize().map_err(|error| {
                InstallerError::at(
                    "IO_ERROR",
                    format!("cannot resolve replay path: {error}"),
                    value,
                )
            })?;
            Ok(parent.join(
                value
                    .file_name()
                    .ok_or_else(|| usage_error("expected a file path"))?,
            ))
        };
        if resolve(&path)? == resolve(out)? {
            return Err(usage_error("--out must differ from the live replay path"));
        }
    }
    // Create exclusively: never truncate the source or an existing output file.
    let mut output: Box<dyn Write> = if let Some(out) = &out {
        Box::new(
            std::fs::OpenOptions::new()
                .write(true)
                .create_new(true)
                .open(out)
                .map_err(|error| {
                    InstallerError::at(
                        "IO_ERROR",
                        format!("cannot create NDJSON output: {error}"),
                        out,
                    )
                })?,
        )
    } else {
        Box::new(std::io::stdout().lock())
    };
    let mut follower = ReplayFollower::new(&path);
    let mut last_activity = Instant::now();
    loop {
        let mut reset = false;
        let bytes = follower.poll(|event| {
            reset |= matches!(&event.event, blitz_client::ReplayStreamEvent::Reset { .. });
            let line = serde_json::to_vec(&event)
                .map_err(|error| InstallerError::new("JSON_SERIALIZE_FAILED", error.to_string()))?;
            output
                .write_all(&line)
                .and_then(|_| output.write_all(b"\n"))
                .and_then(|_| output.flush())
                .map_err(|error| InstallerError::new("REPLAY_OUTPUT_FAILED", error.to_string()))
        })?;
        if bytes > 0 || reset {
            last_activity = Instant::now();
        }
        if idle_timeout.is_some_and(|timeout| last_activity.elapsed() >= timeout) {
            return Ok(true);
        }
        let wait = idle_timeout.map_or(Duration::from_millis(poll_ms), |timeout| {
            Duration::from_millis(poll_ms).min(timeout.saturating_sub(last_activity.elapsed()))
        });
        std::thread::sleep(wait);
    }
}

fn run_replay_analysis(
    args: &[String],
    json: bool,
    use_latest: bool,
) -> blitz_client::Result<bool> {
    let mut path = None;
    let mut directory = None;
    let mut out = None;
    let mut include_data = false;
    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "--data" => include_data = true,
            "--dir" => {
                i += 1;
                directory = args.get(i).map(PathBuf::from);
                if directory.is_none() {
                    return Err(usage_error("--dir requires a path"));
                }
            }
            "--out" => {
                i += 1;
                out = args.get(i).map(PathBuf::from);
                if out.is_none() {
                    return Err(usage_error("--out requires a path"));
                }
            }
            flag if flag.starts_with('-') => {
                return Err(usage_error(&format!("unknown option: {flag}")));
            }
            value if path.is_none() => path = Some(PathBuf::from(value)),
            value => return Err(usage_error(&format!("unexpected argument: {value}"))),
        }
        i += 1;
    }

    let replay = if use_latest {
        if path.is_some() {
            return Err(usage_error("replay latest accepts no replay path"));
        }
        find_latest_replay(directory)?
    } else {
        if directory.is_some() {
            return Err(usage_error("--dir is only valid for replay latest"));
        }
        path.ok_or_else(|| usage_error("replay inspect requires a .tbreplay or data.replay path"))?
    };

    let analysis = analyze_replay(&replay, ReplayOptions { include_data })?;
    if let Some(out) = &out {
        write_replay_json(out, &analysis)?;
    }
    if json {
        print_json(&analysis);
    } else {
        print_replay_summary(&analysis);
        if let Some(out) = &out {
            println!("json: {}", out.display());
        }
    }
    Ok(true)
}

fn write_replay_json(path: &PathBuf, analysis: &ReplayAnalysis) -> blitz_client::Result<()> {
    let json = serde_json::to_vec_pretty(analysis)
        .map_err(|error| InstallerError::new("JSON_SERIALIZE_FAILED", error.to_string()))?;
    std::fs::write(path, json).map_err(|error| {
        InstallerError::at(
            "IO_ERROR",
            format!("cannot write the replay JSON: {error}"),
            path,
        )
    })
}

fn print_replay_summary(analysis: &ReplayAnalysis) {
    let meta = &analysis.meta;
    let results = &analysis.battle_results;
    let author = &results["author"];
    println!("{}", analysis.source);
    println!(
        "client: {}",
        analysis
            .data
            .as_ref()
            .map(|data| json_field(data, "client_version"))
            .unwrap_or_else(|| json_field(meta, "version"))
    );
    if !meta.is_null() {
        println!(
            "map: {} (id {}), vehicle: {}",
            json_field(meta, "mapName"),
            json_field(meta, "mapId"),
            json_field(meta, "playerVehicleName")
        );
        println!(
            "author: {} (dbid {}, damage {}, shots {}, hits {}, hp left {})",
            json_field(meta, "playerName"),
            json_field(meta, "dbid"),
            json_field(author, "damage_dealt"),
            json_field(author, "n_shots"),
            json_field(author, "n_hits"),
            json_field(author, "hitpoints_left")
        );
        println!(
            "battle: {:.1}s, arena {}, winner team {}, players {}",
            meta["battleDuration"].as_f64().unwrap_or_default(),
            json_field(meta, "arenaUniqueId"),
            json_field(results, "winner_team_number"),
            results["players"].as_array().map_or(0, Vec::len)
        );
    }

    let Some(telemetry) = &analysis.telemetry else {
        return;
    };
    if let Some(packets) = analysis
        .data
        .as_ref()
        .and_then(|data| data["packets"].as_array())
    {
        println!("packets: {}", packets.len());
    }
    println!("decoder: {}", json_field(telemetry, "decoder"));
    let summary = &telemetry["summary"];
    if summary["packet_layout_decoding_supported"] != serde_json::Value::Bool(true) {
        println!(
            "telemetry: packet layouts are verified only for {}",
            json_field(telemetry, "supported_client_version")
        );
        return;
    }
    println!(
        "telemetry: shots {}/{}, hits {}/{}, health updates {}, transform updates {}, results snapshots {}",
        json_field(summary, "projectile_position_packets"),
        json_field(summary, "shots_in_battle_results"),
        json_field(summary, "hit_messages"),
        json_field(summary, "hits_received_in_battle_results"),
        json_field(summary, "vehicle_health_updates"),
        json_field(summary, "vehicle_transform_updates"),
        json_field(summary, "battle_results_snapshot_packets")
    );
}

fn json_field(value: &serde_json::Value, key: &str) -> String {
    match &value[key] {
        serde_json::Value::String(text) => text.clone(),
        serde_json::Value::Null => "?".to_owned(),
        other => other.to_string(),
    }
}

fn take_game_dir(args: &[String]) -> blitz_client::Result<(Vec<String>, Option<PathBuf>)> {
    let mut positional = Vec::new();
    let mut game_dir = None;
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
            flag if flag.starts_with('-') => {
                return Err(usage_error(&format!("unknown option: {flag}")));
            }
            value => positional.push(value.to_owned()),
        }
        i += 1;
    }
    Ok((positional, game_dir))
}

fn print_json(value: &impl serde::Serialize) {
    match serde_json::to_string_pretty(value) {
        Ok(json) => println!("{json}"),
        Err(error) => eprintln!("could not serialize JSON output: {error}"),
    }
}

fn usage_error(message: &str) -> InstallerError {
    InstallerError::new("USAGE", message)
}

fn print_usage() {
    println!(
        "blitz-client commands:\n\
         \n\
         blitz-client client locate [--game-dir <path>] [--json]\n\
         blitz-client client tanks list [--game-dir <path>] [--json]\n\
         blitz-client client export --output-dir <path> [--game-dir <path>] [--data-only] [model options]\n\
         blitz-client client tanks export [--game-dir <path>] --viewer-dir <path> [--tank <nation/ID>] [--limit <N>] [--pbr] [--no-compress] [--json]\n\
         blitz-client mod validate <mod-dir|mod.toml> [--game-dir <path>] [--json]\n\
         blitz-client mod install <mod-dir|mod.toml> [--game-dir <path>] [--json]\n\
         blitz-client mod uninstall <mod-id> [--game-dir <path>] [--json]\n\
         blitz-client mod update-from-git <mod-dir|mod.toml> --repo <path> [--base <rev>] [--json]\n\
         blitz-client replay inspect <replay.tbreplay|data.replay> [--data] [--out <path>] [--json]\n\
         blitz-client replay latest [--dir <path>] [--data] [--out <path>] [--json]\n\
         blitz-client replay follow <data.replay> [--poll-ms <N>] [--idle-timeout-ms <N>] [--out <new-path>]\n\
         \n\
         --game-dir is optional. Without it the first existing default path for this OS is used.\n\
         replay latest uses the newest .tbreplay of the platform replay folder; --dir overrides it."
    );
}
