mod client;
mod mods;
mod replay;
use blitz_client::Error;
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
        "client" => client::run(&args[1..], json),
        "mod" => mods::run(&args[1..], json),
        "replay" => replay::run(&args[1..], json),
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

fn usage_error(message: &str) -> Error {
    Error::new("USAGE", message)
}

fn print_usage() {
    println!(
        "blitz-client commands:\n\
         \n\
         blitz-client client locate [--game-dir <path>] [--json]\n\
         blitz-client client files export --output-dir <path> [--game-dir <path>] [--cache-dir <path> | --no-cache] [--types <all|configs,images,models,sounds,fonts,shaders,other>] [--dry-run] [--json]\n\
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
