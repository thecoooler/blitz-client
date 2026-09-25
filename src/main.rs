use blitz_mod_installer::{
    InstallerError, discover_game_dir, install_mod, remove_mod, update_from_git_diff, validate_mod,
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
        "locate-game" => {
            if args.len() != 1 {
                Err(usage_error("locate-game accepts no positional arguments"))
            } else {
                discover_game_dir().map(|resolved| {
                    if json {
                        print_json(&resolved);
                    } else {
                        println!("{}", resolved.game_dir.display());
                        println!("resource directory: {}", resolved.resource_dir.display());
                    }
                    true
                })
            }
        }
        "validate" => run_validate(&args[1..], json),
        "install" => run_install(&args[1..], json),
        "uninstall" | "remove" => run_uninstall(&args[1..], json),
        "update-from-git" => run_update(&args[1..], json),
        "mod" if args.get(1).map(String::as_str) == Some("update-from-git") => {
            run_update(&args[2..], json)
        }
        command => Err(usage_error(&format!("unknown command: {command}"))),
    };

    match result {
        Ok(true) => 0,
        Ok(false) => 4,
        Err(error) => {
            if json {
                print_json(&serde_json::json!({"ok": false, "error": error}));
            } else {
                eprintln!("{error}");
            }
            2
        }
    }
}

fn run_validate(args: &[String], json: bool) -> blitz_mod_installer::Result<bool> {
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

fn run_install(args: &[String], json: bool) -> blitz_mod_installer::Result<bool> {
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

fn run_uninstall(args: &[String], json: bool) -> blitz_mod_installer::Result<bool> {
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

fn run_update(args: &[String], json: bool) -> blitz_mod_installer::Result<bool> {
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

fn take_game_dir(args: &[String]) -> blitz_mod_installer::Result<(Vec<String>, Option<PathBuf>)> {
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
        "blitz-mod commands:\n\
         \n\
         blitz-mod locate-game [--json]\n\
         blitz-mod validate <mod-dir|mod.toml> [--game-dir <path>] [--json]\n\
         blitz-mod install <mod-dir|mod.toml> [--game-dir <path>] [--json]\n\
         blitz-mod uninstall <mod-id> [--game-dir <path>] [--json]\n\
         blitz-mod mod update-from-git <mod-dir|mod.toml> --repo <path> [--base <rev>] [--json]\n\
         \n\
         --game-dir is optional. Without it the first existing default path for this OS is used."
    );
}
