use super::{print_json, usage_error};
use blitz_client::{
    Error,
    replay::{
        ReplayAnalysis, ReplayFollower, ReplayOptions, ReplayStreamEvent, analyze_replay,
        find_latest_replay,
    },
};
use std::path::PathBuf;

pub(super) fn run(args: &[String], json: bool) -> blitz_client::Result<bool> {
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
                Error::at(
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
                    Error::at(
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
            reset |= matches!(&event.event, ReplayStreamEvent::Reset { .. });
            let line = serde_json::to_vec(&event)
                .map_err(|error| Error::new("JSON_SERIALIZE_FAILED", error.to_string()))?;
            output
                .write_all(&line)
                .and_then(|_| output.write_all(b"\n"))
                .and_then(|_| output.flush())
                .map_err(|error| Error::new("REPLAY_OUTPUT_FAILED", error.to_string()))
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
        .map_err(|error| Error::new("JSON_SERIALIZE_FAILED", error.to_string()))?;
    std::fs::write(path, json).map_err(|error| {
        Error::at(
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
