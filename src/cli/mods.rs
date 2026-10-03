use super::{print_json, take_game_dir, usage_error};
use blitz_client::mods::ValidationReport;
use blitz_client::mods::{
    discover_mods, install_mod, list_installed_mods, remove_mod, update_from_git_diff, validate_mod,
};
use std::collections::BTreeSet;
use std::io::IsTerminal;
use std::path::{Path, PathBuf};

pub(super) fn run(args: &[String], json: bool) -> blitz_client::Result<bool> {
    match args.first().map(String::as_str) {
        Some("list") => run_list(&args[1..], json),
        Some("validate") => run_validate(&args[1..], json),
        Some("install") => run_install(&args[1..], json),
        Some("uninstall") => run_uninstall(&args[1..], json),
        Some("update-from-git") => run_update(&args[1..], json),
        _ => Err(usage_error(
            "expected mod list/validate/install/uninstall/update-from-git",
        )),
    }
}

fn run_validate(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let (positional, game_dir) = take_game_dir(args)?;
    if positional.len() != 1 {
        return Err(usage_error(
            "validate requires a mods directory, mod directory or mod.yaml/mod.yml path",
        ));
    }
    let style = ValidationStyle::new(std::io::stdout().is_terminal());
    let path = Path::new(&positional[0]);
    if !path.is_dir() {
        let report = validate_mod(path, game_dir)?;
        if json {
            print_json(&report);
        } else {
            print_validation(&report, &style);
        }
        return Ok(report.valid);
    }

    let manifests = discover_mods(path)?;
    if manifests.is_empty() {
        return Err(blitz_client::Error::at(
            "MANIFEST_NOT_FOUND",
            "no mod.yaml/mod.yml found in the directory or its subdirectories",
            path,
        ));
    }
    // Keep the existing report shape for a single mod directory.
    if manifests.len() == 1 && manifests[0].parent() == Some(path) {
        let report = validate_mod(&manifests[0], game_dir)?;
        if json {
            print_json(&report);
        } else {
            print_validation(&report, &style);
        }
        return Ok(report.valid);
    }

    let mut results = Vec::new();
    let mut passed = 0;
    let mut failures = Vec::new();
    for manifest in &manifests {
        match validate_mod(manifest, game_dir.clone()) {
            Ok(report) => {
                if report.valid {
                    passed += 1;
                }
                if json {
                    results.push(serde_json::json!({"manifest": manifest, "report": report}));
                } else if report.valid {
                    print_validation(&report, &style);
                } else {
                    failures.push((manifest, Ok(report)));
                }
            }
            Err(error) => {
                if json {
                    results.push(serde_json::json!({"manifest": manifest, "error": error}));
                } else {
                    failures.push((manifest, Err(error)));
                }
            }
        }
    }
    let valid = passed == manifests.len();
    if json {
        print_json(&serde_json::json!({"valid": valid, "mods": results}));
    } else {
        if !failures.is_empty() {
            if passed > 0 {
                println!();
            }
            println!(
                "{}",
                style.paint(31, &format!("Failures ({})", failures.len()))
            );
            for (manifest, result) in failures {
                match result {
                    Ok(report) => print_validation(&report, &style),
                    Err(error) => print_manifest_failure(path, manifest, &error, &style),
                }
            }
        }
        println!(
            "\n{} · {} · {}",
            style.paint(2, &format!("{} mods", manifests.len())),
            style.paint(32, &format!("{passed} passed")),
            style.paint(
                if valid { 2 } else { 31 },
                &format!("{} failed", manifests.len() - passed)
            )
        );
    }
    Ok(valid)
}

fn run_list(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let (positional, game_dir) = take_game_dir(args)?;
    if !positional.is_empty() {
        return Err(usage_error("list accepts only --game-dir and --json"));
    }
    let report = list_installed_mods(game_dir)?;
    if json {
        print_json(&report);
    } else if report.mods.is_empty() {
        println!("No mods installed");
    } else {
        for item in &report.mods {
            println!(
                "{} — {} ({} {})",
                item.id,
                item.name,
                item.files.len(),
                if item.files.len() == 1 {
                    "file"
                } else {
                    "files"
                }
            );
        }
    }
    Ok(true)
}

struct ValidationStyle {
    color: bool,
}

impl ValidationStyle {
    fn new(terminal: bool) -> Self {
        let nonzero =
            |name| std::env::var_os(name).is_some_and(|value| !value.is_empty() && value != "0");
        let no_color = std::env::var_os("NO_COLOR").is_some_and(|value| !value.is_empty());
        Self {
            color: !no_color
                && (nonzero("CLICOLOR_FORCE")
                    || (terminal
                        && std::env::var_os("TERM").is_none_or(|value| value != "dumb")
                        && std::env::var_os("CLICOLOR").is_none_or(|value| value != "0"))),
        }
    }

    fn paint(&self, code: u8, value: &str) -> String {
        if self.color {
            format!("\x1b[{code}m{value}\x1b[0m")
        } else {
            value.to_owned()
        }
    }
}

fn print_validation(report: &ValidationReport, style: &ValidationStyle) {
    let mut seen = BTreeSet::new();
    let errors = report
        .files
        .iter()
        .filter(|file| !file.valid)
        .filter(|file| seen.insert((&file.path, &file.error_code, &file.message)))
        .collect::<Vec<_>>();
    let (symbol, count) = if report.valid {
        let files = report
            .files
            .iter()
            .map(|file| &file.path)
            .collect::<BTreeSet<_>>()
            .len();
        (
            style.paint(32, "✓"),
            format!("{files} {}", if files == 1 { "file" } else { "files" }),
        )
    } else {
        (
            style.paint(31, "✗"),
            format!(
                "{} {}",
                errors.len(),
                if errors.len() == 1 { "error" } else { "errors" }
            ),
        )
    };
    println!(
        "{symbol} {} {}",
        report.mod_id,
        style.paint(2, &format!("({count})"))
    );
    for file in errors {
        println!("  {}", file.path);
        print_diagnostic(
            style,
            file.error_code.as_deref().unwrap_or("ERROR"),
            file.message.as_deref().unwrap_or(""),
        );
    }
}

fn print_manifest_failure(
    root: &Path,
    manifest: &Path,
    error: &blitz_client::Error,
    style: &ValidationStyle,
) {
    let relative = manifest.strip_prefix(root).unwrap_or(manifest);
    let label = relative
        .parent()
        .filter(|dir| !dir.as_os_str().is_empty())
        .unwrap_or(relative);
    println!("{} {}", style.paint(31, "✗"), label.display());
    print_diagnostic(style, &error.code, &error.message);
    if let Some(error_path) = &error.path {
        if error_path != manifest {
            println!("  {}", style.paint(2, &error_path.display().to_string()));
        }
    }
}

fn print_diagnostic(style: &ValidationStyle, code: &str, message: &str) {
    let mut lines = message.lines();
    println!(
        "  {}: {}",
        style.paint(31, code),
        lines.next().unwrap_or("")
    );
    for line in lines {
        println!("  {line}");
    }
}

pub(super) fn print_validation_error(error: &blitz_client::Error) {
    let style = ValidationStyle::new(std::io::stderr().is_terminal());
    eprintln!(
        "{} {}: {}",
        style.paint(31, "✗"),
        style.paint(31, &error.code),
        error.message
    );
    if let Some(path) = &error.path {
        eprintln!("  {}", style.paint(2, &path.display().to_string()));
    }
}

fn run_install(args: &[String], json: bool) -> blitz_client::Result<bool> {
    let (positional, game_dir) = take_game_dir(args)?;
    if positional.len() != 1 {
        return Err(usage_error(
            "install requires a mod directory or mod.yaml/mod.yml path",
        ));
    }
    let report = install_mod(&positional[0], game_dir)?;
    if json {
        print_json(&report);
    } else {
        println!(
            "{} {} — {}",
            if report.reinstalled {
                "Reinstalled"
            } else {
                "Installed"
            },
            report.mod_id,
            report.mod_name
        );
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
            "update-from-git requires a mod directory or mod.yaml/mod.yml path",
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
            .filter(|file| file.patch.as_deref() == Some("mod.patch"))
            .map(|file| &file.name)
            .collect::<BTreeSet<_>>()
            .len();
        println!("Updated {}: mod.patch ({generated} resources)", manifest.id);
    }
    Ok(true)
}
