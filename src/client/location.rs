use crate::{Error, Result};
use serde::Serialize;
use std::env;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Serialize)]
pub struct ResolvedGameDir {
    /// The install directory selected by the user or the default-path search.
    pub game_dir: PathBuf,
    /// Resource root used to resolve paths from mod.toml.
    pub resource_dir: PathBuf,
    /// Standard downloaded-resource cache (packs), independent of the install path.
    /// The directory need not exist yet.
    pub cache_dir: Option<PathBuf>,
    pub source: String,
    /// Standard replay folder for this OS; it need not exist yet.
    pub replay_dir: Option<PathBuf>,
}

/// Use an explicitly supplied directory, or search the ordered platform defaults.
/// An invalid explicit path is an error and never falls back to auto-detection.
pub fn resolve_game_dir(explicit: Option<PathBuf>) -> Result<ResolvedGameDir> {
    if let Some(path) = explicit {
        if !path.is_dir() {
            return Err(Error::at(
                "GAME_DIR_NOT_FOUND",
                "the explicitly supplied game directory does not exist",
                path,
            ));
        }
        let canonical = path
            .canonicalize()
            .map_err(|e| crate::io_error("could not resolve game directory", &path, e))?;
        let (game_dir, resource_dir) = normalize_explicit_dir(canonical);
        return Ok(ResolvedGameDir {
            game_dir,
            resource_dir,
            cache_dir: crate::paths::default_cache_dir().ok(),
            source: "explicit".to_owned(),
            replay_dir: crate::paths::default_replay_dir().ok(),
        });
    }

    discover_game_dir()
}

/// Return the first existing candidate in the platform-specific search order.
pub fn discover_game_dir() -> Result<ResolvedGameDir> {
    let candidates = default_candidates();
    for candidate in &candidates {
        if candidate.is_dir() {
            let canonical = candidate
                .canonicalize()
                .unwrap_or_else(|_| candidate.clone());
            let (game_dir, resource_dir) = normalize_explicit_dir(canonical);
            return Ok(ResolvedGameDir {
                game_dir,
                resource_dir,
                cache_dir: crate::paths::default_cache_dir().ok(),
                source: "default".to_owned(),
                replay_dir: crate::paths::default_replay_dir().ok(),
            });
        }
    }

    let paths = candidates
        .iter()
        .map(|p| p.display().to_string())
        .collect::<Vec<_>>()
        .join(", ");
    Err(Error::new(
        "GAME_DIR_NOT_FOUND",
        format!("no default Tanks Blitz directory exists; checked: {paths}; pass --game-dir"),
    ))
}

fn find_resource_root(game_dir: &Path) -> PathBuf {
    let windows_data = game_dir.join("Data");
    if windows_data.is_dir() {
        return windows_data;
    }
    let mac_data = game_dir.join("Contents").join("Resources").join("Data");
    if mac_data.is_dir() {
        return mac_data;
    }
    let nested_mac_data = game_dir
        .join("World of Tanks Blitz.app")
        .join("Contents")
        .join("Resources")
        .join("Data");
    if nested_mac_data.is_dir() {
        return nested_mac_data;
    }
    game_dir.to_path_buf()
}

fn normalize_explicit_dir(path: PathBuf) -> (PathBuf, PathBuf) {
    if path
        .file_name()
        .is_some_and(|name| name.eq_ignore_ascii_case("Data"))
    {
        if path
            .parent()
            .and_then(Path::file_name)
            .is_some_and(|name| name.eq_ignore_ascii_case("Resources"))
        {
            if let Some(app_root) = path.parent().and_then(Path::parent).and_then(Path::parent) {
                return (app_root.to_path_buf(), path);
            }
        } else if let Some(parent) = path.parent() {
            return (parent.to_path_buf(), path);
        }
    }
    let resource_dir = find_resource_root(&path);
    (path, resource_dir)
}

fn default_candidates() -> Vec<PathBuf> {
    let mut out = Vec::new();
    #[cfg(target_os = "windows")]
    windows_candidates(&mut out);
    #[cfg(target_os = "macos")]
    macos_candidates(&mut out);
    #[cfg(target_os = "linux")]
    linux_candidates(&mut out);
    deduplicate(&mut out);
    out
}

#[cfg(any(target_os = "windows", target_os = "linux"))]
fn push_steam_install(out: &mut Vec<PathBuf>, steam_root: &Path) {
    let common = steam_root
        .join("steamapps")
        .join("common")
        .join("World of Tanks Blitz");
    out.push(common);
}

#[cfg(target_os = "windows")]
fn windows_candidates(out: &mut Vec<PathBuf>) {
    out.push(PathBuf::from(r"C:\Games\Tanks_Blitz"));
    out.push(PathBuf::from(r"C:\Games\Tanks_Blitz\Data"));

    let mut steam_roots = Vec::new();
    for key in ["ProgramFiles(x86)", "ProgramFiles"] {
        if let Some(value) = env::var_os(key) {
            steam_roots.push(PathBuf::from(value).join("Steam"));
        }
    }
    if let Some(local) = env::var_os("LOCALAPPDATA") {
        steam_roots.push(PathBuf::from(local).join("Programs").join("Steam"));
    }
    steam_roots.extend([
        PathBuf::from(r"C:\Steam"),
        PathBuf::from(r"D:\Steam"),
        PathBuf::from(r"C:\Games\Steam"),
    ]);
    for root in steam_roots {
        push_steam_install(out, &root);
    }

    if let Some(program_files) = env::var_os("ProgramFiles") {
        let store = PathBuf::from(program_files).join("WindowsApps");
        if let Ok(entries) = std::fs::read_dir(store) {
            let mut entries = entries.flatten().collect::<Vec<_>>();
            entries.sort_by_key(|entry| entry.file_name().to_string_lossy().to_ascii_lowercase());
            entries.reverse();
            for entry in entries {
                let name = entry.file_name().to_string_lossy().to_ascii_lowercase();
                if name.starts_with("7458be2c.worldoftanksblitz_") {
                    out.push(entry.path().join("Data"));
                    out.push(entry.path());
                }
            }
        }
    }

    if let Some(profile) = env::var_os("USERPROFILE") {
        for relative in [
            r"Games\World of Tanks Blitz\Data",
            r"Games\World of Tanks Blitz",
            r"XboxGames\World of Tanks Blitz\Content\Data",
        ] {
            out.push(PathBuf::from(&profile).join(relative));
        }
    }
}

#[cfg(target_os = "macos")]
fn macos_candidates(out: &mut Vec<PathBuf>) {
    out.push(PathBuf::from(
        "/Applications/Tanks Blitz.app/Contents/Resources/Data",
    ));
    out.push(PathBuf::from("/Applications/Tanks Blitz.app"));

    if let Some(home) = env::var_os("HOME") {
        let steam = PathBuf::from(home)
            .join("Library")
            .join("Application Support")
            .join("Steam");
        let common = steam
            .join("steamapps")
            .join("common")
            .join("World of Tanks Blitz");
        out.push(common);
        out.push(
            steam
                .join("steamapps")
                .join("common")
                .join("World of Tanks Blitz")
                .join("World of Tanks Blitz.app"),
        );
    }
}

#[cfg(target_os = "linux")]
fn linux_candidates(out: &mut Vec<PathBuf>) {
    if let Some(home) = env::var_os("HOME") {
        let home = PathBuf::from(home);
        for steam in [
            home.join(".local/share/Steam"),
            home.join(".steam/steam"),
            home.join(".var/app/com.valvesoftware.Steam/.local/share/Steam"),
            home.join("snap/steam/common/.local/share/Steam"),
        ] {
            push_steam_install(out, &steam);
        }
    }
    for steam in [
        PathBuf::from("/usr/share/steam"),
        PathBuf::from("/usr/lib/steam"),
    ] {
        push_steam_install(out, &steam);
    }
}

fn deduplicate(paths: &mut Vec<PathBuf>) {
    let mut unique = Vec::new();
    for path in paths.drain(..) {
        if !unique.contains(&path) {
            unique.push(path);
        }
    }
    *paths = unique;
}
