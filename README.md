# Tanks Blitz Mod Installer

Rust library and CLI for installing text-resource mods in Tanks Blitz clients. The core library can also be called directly from a Tauri Rust backend.

## Commands

```text
blitz-mod locate-game [--json]
blitz-mod validate <mod-dir|mod.toml> [--game-dir <path>] [--json]
blitz-mod install <mod-dir|mod.toml> [--game-dir <path>] [--json]
blitz-mod uninstall <mod-id> [--game-dir <path>] [--json]
blitz-mod mod update-from-git <mod-dir|mod.toml> --repo <path> [--base <rev>] [--json]
```

`--game-dir` is optional. Without it, the first existing default candidate for the current OS is used. If it is supplied, an invalid path is an error and automatic discovery is not attempted.

The default search candidates are ordered in `src/game_dir.rs`:

- Windows: `C:\Games\TanksBlitz\Data` first, then standard Steam locations under `ProgramFiles(x86)`, `ProgramFiles`, and `LOCALAPPDATA`; `C:\Steam`, `D:\Steam`, `C:\Games\Steam`; WindowsApps package folders; and common per-user Games/XboxGames folders.
- macOS: `/Applications/Tanks Blitz.app/Contents/Resources/Data` first, then `~/Library/Application Support/Steam/steamapps/common/World of Tanks Blitz`, including the app bundle's `Contents/Resources/Data` folder.
- Linux: the usual Steam locations under `~/.local/share/Steam`, `~/.steam/steam`, Steam Flatpak, and Steam Snap, plus system Steam directories.

Custom Steam libraries and other non-default locations can be selected with `--game-dir`.

## Mod manifest

The logical target omits the final `.dvpl` suffix. The installer resolves `UI/.../Minimap.yaml` to either `Minimap.yaml.dvpl` or a plain `Minimap.yaml`, rejecting the case where both exist.

Exact replacement is supported:

```toml
id = "NoMapName"
name = "Remove map name above the minimap"

[[files]]
name = "UI/Screens/Battle/Minimap.yaml"
action = "replace"
find = '''exact text to find'''
replace = ''
```

Unified diff patches are stored as sidecar files:

```toml
[[files]]
name = "UI/Screens/Battle/Minimap.yaml"
action = "patch"
patch = "patches/UI/Screens/Battle/Minimap.yaml.<diff-crc>.patch"
generated_by = "git"
```

`update-from-git` compares the selected base revision (default `HEAD`) with the working tree, including staged and unstaged changes. The repository paths must mirror the game's unpacked resource paths. It handles modifications to existing tracked text resources; additions, deletions, renames, and binary changes are rejected. `.dvpl` files must be unpacked before generating a diff.

## Install and restore behavior

- `validate` is read-only and checks the manifest, resource, DVPL footer/CRC, text decoding, and patch application.
- `install` uses the file's current contents as the base and does not require a hash for a clean game version. It saves a neighboring `*.tanks-blitz-mod-installer-original` file if one does not already exist, and stores a managed base snapshot under `.tanks-blitz-mod-installer/bases`.
- A state file in `.tanks-blitz-mod-installer/state.json` tracks installed mods and their operations. Each install recomputes affected resources from the saved base, applying active mods in sorted mod ID order. Independent patches compose regardless of installation order. Incompatible edits to the same text fail with a conflict.
- `uninstall` removes a mod from the active set and recomputes affected resources, so mods can be uninstalled in any order. The managed base snapshot is deleted when its last mod is removed; the adjacent original file is kept. The stored CRC detects changes made outside the installer and prevents overwriting them.
- The game must be closed during installation and uninstallation.

## Rust API

```rust,no_run
use std::path::PathBuf;
use blitz_mod_installer::{install_mod, remove_mod, update_from_git_diff, validate_mod};

let game_dir: Option<PathBuf> = None; // auto-discover
let report = validate_mod("mods/NoMapName", game_dir.clone())?;
if report.valid {
    install_mod("mods/NoMapName", game_dir)?;
}
remove_mod("NoMapName", None)?;
update_from_git_diff("mods/NoMapName", "mods/NoMapName", Some("HEAD"))?;
# Ok::<(), blitz_mod_installer::InstallerError>(())
```
