use super::{print_json, take_game_dir, usage_error};
use blitz_client::mods::ValidationReport;
use blitz_client::mods::{install_mod, remove_mod, update_from_git_diff, validate_mod};
use std::path::{Path, PathBuf};

pub(super) fn run(args: &[String], json: bool) -> blitz_client::Result<bool> {
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

fn run_validate(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let (positional, game_dir) = take_game_dir(args)?;
    if positional.len() != 1 {
        return Err(usage_error(
            "validate requires a mods directory, mod directory or mod.toml path",
        ));
    }
    let path = Path::new(&positional[0]);
    if !path.is_dir() {
        let report = validate_mod(path, game_dir)?;
        if json {
            print_json(&report);
        } else {
            print_validation(&report);
        }
        return Ok(report.valid);
    }

    let mut manifests = Vec::new();
    collect_manifests(path, &mut manifests)?;
    manifests.sort();
    if manifests.is_empty() {
        return Err(blitz_client::Error::at(
            "MANIFEST_NOT_FOUND",
            "no mod.toml found in the directory or its subdirectories",
            path,
        ));
    }
    // Keep the existing report shape for a single mod directory.
    if manifests.len() == 1 && manifests[0] == path.join("mod.toml") {
        let report = validate_mod(&manifests[0], game_dir)?;
        if json {
            print_json(&report);
        } else {
            print_validation(&report);
        }
        return Ok(report.valid);
    }

    let mut results = Vec::new();
    let mut passed = 0;
    for manifest in &manifests {
        if !json {
            println!("{}", manifest.display());
        }
        match validate_mod(manifest, game_dir.clone()) {
            Ok(report) => {
                if report.valid {
                    passed += 1;
                }
                if !json {
                    print_validation(&report);
                }
                results.push(serde_json::json!({"manifest": manifest, "report": report}));
            }
            Err(error) => {
                if !json {
                    println!("FAIL {error}");
                }
                results.push(serde_json::json!({"manifest": manifest, "error": error}));
            }
        }
    }
    let valid = passed == manifests.len();
    if json {
        print_json(&serde_json::json!({"valid": valid, "mods": results}));
    } else {
        println!(
            "Validated {} mod(s): {passed} passed, {} failed",
            manifests.len(),
            manifests.len() - passed
        );
    }
    Ok(valid)
}

fn collect_manifests(dir: &Path, manifests: &mut Vec<PathBuf>) -> blitz_client::Result<()> {
    let io_error = |error: std::io::Error| {
        blitz_client::Error::at(
            "IO_ERROR",
            format!("could not scan mod directory: {error}"),
            dir,
        )
    };
    for entry in std::fs::read_dir(dir).map_err(io_error)? {
        let entry = entry.map_err(io_error)?;
        let kind = entry.file_type().map_err(io_error)?;
        if kind.is_dir() {
            collect_manifests(&entry.path(), manifests)?;
        } else if entry.file_name() == "mod.toml" && entry.path().is_file() {
            manifests.push(entry.path());
        }
    }
    Ok(())
}

fn print_validation(report: &ValidationReport) {
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
