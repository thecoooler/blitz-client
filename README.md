# Tanks Blitz Client Tools

Rust library and CLI for inspecting the installed Tanks Blitz client, exporting tank models for the local viewer, and installing resource mods. The same library can be called from a Tauri Rust backend.

## CLI

The executable is `blitz-client`.

```text
blitz-client client locate [--game-dir <path>] [--json]
blitz-client client tanks list [--game-dir <path>] [--nation <nation>] [--json]
blitz-client client tanks export --viewer-dir <path> [--game-dir <path>] [options]

blitz-client mod validate <mod-dir|mod.toml> [--game-dir <path>] [--json]
blitz-client mod install <mod-dir|mod.toml> [--game-dir <path>] [--json]
blitz-client mod uninstall <mod-id> [--game-dir <path>] [--json]
blitz-client mod update-from-git <mod-dir|mod.toml> --repo <path> [--base <rev>] [--json]

blitz-client replay inspect <replay.tbreplay> [--data] [--out <path>] [--json]
blitz-client replay latest [--dir <path>] [--data] [--out <path>] [--json]
```

`--game-dir` accepts either the game directory or the `Data` directory. It is optional; the first existing default path for the current OS is used. An invalid explicit path is an error and never falls back to auto-detection.

```sh
blitz-client client locate
blitz-client client tanks list --json
blitz-client client tanks list --nation ussr

blitz-client client tanks export \
  --viewer-dir "/path/to/viewer" \
  --limit 10

blitz-client client tanks export \
  --viewer-dir "/path/to/viewer" \
  --tank china/Ch48_BZ_75
```

`client tanks list` reports the vehicles with both showroom model files installed. The JSON response contains `count` and a `tanks` array with the ID, localized name, nation, level, `userString`, and source model paths.

The exporter is implemented in Rust and writes to `<viewer-dir>/assets/tanks`. It generates the viewer's `catalog.json` format (`formatVersion: 1`, `count`, `entries`) and `export-report.json`. Each successful entry points to the model bundle, optional armor files, and texture assets using the same URL layout the viewer already reads. Tank bundles are left in place unless `--prune` is explicitly supplied.

Exporter options:

- `--tank <ID|nation/ID>` can be repeated. An unqualified ID must be unique across nations.
- `--nation <nation>` filters automatic discovery; `--limit N` exports at most N successful models. A limit of 0 means all models.
- `--max-texture-size original|N|WIDTHxHEIGHT` caps source textures; the default is `1024x1024`.
- `--texture-format webp|png` selects texture format; WebP is the default. `--webp-quality 0..100` sets lossy base-color quality.
- `--pbr` exports the PBR maps and tangent basis. `--no-compress` writes plain model JSON and geometry instead of `.br` assets.
- `--force` is accepted for explicit full regeneration. Texture caching is not enabled in this Rust exporter yet, so exports currently rewrite textures on every run. `--prune` removes bundles omitted from the new catalog.

SC2/SCG decoding, mesh transforms, geometry generation, armor joining, catalogs, and reports are Rust code. Texture conversion uses `ffmpeg`; WebP output uses `cwebp`; Brotli model compression uses the `brotli` command. Add these codec executables to `PATH` for the corresponding export options. Python is not used by this application.

## Replay analysis

`replay inspect` decodes a `.tbreplay` ZIP archive: `meta.json`, the protocol-2 pickle `battle_results.dat` and, with `--data`, the framed `data.replay` packet stream with the recovered telemetry (shots, hits, health and transform updates, booster counters, task markers, packet annotations). A human-readable summary goes to stdout, `--json` prints the whole document and `--out <path>` writes it to a file.

`replay latest` picks the newest `.tbreplay` of the platform replay folder (macOS: `~/Library/Containers/com.tanksblitz/Data/Documents/tb-project/replays`); `--dir <path>` selects another folder.

Packet layouts were verified on `26.8.0_ruby_apple` replays. Other client versions still decode the archive and battle results, keep every raw payload and report `unsupported-client-version` telemetry. Pickle values are read as builtins only: the decoder never imports or runs Python code.

## Mod commands

The logical target in a manifest omits the final `.dvpl` suffix. The installer resolves `UI/.../Minimap.yaml` to either `Minimap.yaml.dvpl` or a plain `Minimap.yaml`, rejecting the case where both exist.

Exact replacement:

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

`update-from-git` compares the selected base revision (default `HEAD`) with the working tree, including staged and unstaged changes. Repository paths must mirror the game's unpacked resource paths. It handles modifications to existing tracked text resources; additions, deletions, renames, and binary changes are rejected. `.dvpl` files must be unpacked before generating a diff.

The game must be closed during installation and uninstallation. Active mods are recomposed from the current game resources and saved installation state, so mod removal order does not affect the remaining patches. The installer does not use a fixed game-file hash to identify a clean client.

## Game directory defaults

The ordered platform candidates are implemented in `src/game_dir.rs`.

- Windows: `C:\Games\TanksBlitz\Data` first, followed by standard Steam, WindowsApps, and user game folders.
- macOS: `/Applications/Tanks Blitz.app/Contents/Resources/Data` first, then the Steam installation under the user's Library.
- Linux: common Steam, Flatpak, and Snap installation folders.

## Rust API

```rust,no_run
use std::path::PathBuf;
use blitz_mod_installer::{
    TankExportOptions, export_tank_catalog, install_mod, list_client_tanks,
    remove_mod, update_from_git_diff, validate_mod,
};

let game_dir: Option<PathBuf> = None;
let tanks = list_client_tanks(game_dir.clone())?;
let mut export = TankExportOptions::new("/path/to/viewer");
export.game_dir = game_dir.clone();
export.limit = 10;
let catalog = export_tank_catalog(&export)?;

let report = validate_mod("mods/NoMapName", game_dir.clone())?;
if report.valid {
    install_mod("mods/NoMapName", game_dir)?;
}
remove_mod("NoMapName", None)?;
update_from_git_diff("mods/NoMapName", "mods/NoMapName", Some("HEAD"))?;
# let _ = (tanks, catalog);
# Ok::<(), blitz_mod_installer::InstallerError>(())
```
