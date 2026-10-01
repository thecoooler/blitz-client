# Tanks Blitz Client Tools

Rust library and CLI for inspecting the installed Tanks Blitz client, exporting tank models for the local viewer, installing resource mods, and analyzing complete or live replays. The same library can be called from a Tauri Rust backend.

## Library modules and selective builds

The project is one crate with five functional modules. Each module exposes its
interface from `mod.rs`; implementation files stay private unless they are useful
resource parsers. CLI argument parsing and output formatting live in `src/cli/`.

| Feature / module | Responsibility | Enables |
| --- | --- | --- |
| `resources` | DVPL codecs, XML, SC2/SCG archives | Resource codec dependencies |
| `client` | Client location, resource lookup, localization, tank and map catalogs | `resources` |
| `export` | Client data and tank bundles, model/armor conversion, export output | `client` |
| `mods` | Manifests, validation, installation, removal, Git patches | `client` |
| `replay` | Archive/standalone analysis, byte framing, file following, telemetry | Replay parser dependencies |
| `cli` | The existing `blitz-client` executable and all commands | `export`, `mods`, `replay` |

The default feature is `cli`, preserving the complete application. To use just
replays in another Rust project:

```toml
[dependencies]
blitz-client = { path = "../blitz-client", default-features = false, features = ["replay"] }
```

```rust
use blitz_client::replay::{ReplayFollower, ReplayOptions, analyze_replay};
```

For resource parsing or client catalogs, select `resources` or `client` instead.
Features can be combined, for example `features = ["client", "replay"]`. The binary
is built only with `cli`; library builds don't need CLI modules. With no features,
only the shared `Error` / `Result` interface and its Serde support remain. Optional
unselected dependencies are not compiled; selected libraries can still bring
transitive dependencies. Cargo merges features requested by different dependents,
so every dependency that wants a selective build must disable default features.

The preferred imports are `blitz_client::client::*`, `blitz_client::export::*`,
`blitz_client::mods::*`, `blitz_client::replay::*`, and `blitz_client::resources::*`.
Original crate-root imports remain available under the corresponding feature.
`InstallerError` is an alias for `Error`; its serialized fields and error codes are
unchanged. Replay `Packet`, `Payload`, and `EntityMethod` types are reexported from
`replay`, so callers need no direct dependency on the underlying parser.

Module dependencies run from high-level operations toward resource readers:
`export` and `mods` use `client` and `resources`; `client` uses `resources`;
`replay` is independent. `export/output.rs` owns the shared file-writing logic.
`resources` never imports the exporter. Tests for resource formats live with
those parsers; tests for model and armor policies live in `export`. They share
byte-level fixtures compiled only for tests.

Library-only examples:

```sh
cargo run --no-default-features --features replay --example replay_follow -- ./data.replay
cargo run --no-default-features --features client --example client_catalog -- /path/to/Data
cargo run --no-default-features --features resources --example decode_resource -- /path/to/file.dvpl
```

Run `bash scripts/check-features.sh` to test the full application and each feature
in isolation. Add `--offline` when dependencies are already cached. Existing replay
CLI integration tests run only with `cli`; byte decoders and library tests also run
in their isolated feature builds.

## CLI

The executable is `blitz-client`.

```text
blitz-client client locate [--game-dir <path>] [--json]
blitz-client client tanks list [--game-dir <path>] [--nation <nation>] [--json]
blitz-client client export --output-dir <path> [--game-dir <path>] [--data-only] [model options]
blitz-client client tanks export --viewer-dir <path> [--game-dir <path>] [options]

blitz-client mod validate <mod-dir|mod.toml> [--game-dir <path>] [--json]
blitz-client mod install <mod-dir|mod.toml> [--game-dir <path>] [--json]
blitz-client mod uninstall <mod-id> [--game-dir <path>] [--json]
blitz-client mod update-from-git <mod-dir|mod.toml> --repo <path> [--base <rev>] [--json]

blitz-client replay inspect <replay.tbreplay|data.replay> [--data] [--out <path>] [--json]
blitz-client replay latest [--dir <path>] [--data] [--out <path>] [--json]
blitz-client replay follow <data.replay> [--poll-ms <N>] [--idle-timeout-ms <N>] [--out <new-path>]
```

`--game-dir` accepts either the game directory or the `Data` directory. It is optional; the first existing default path for the current OS is used. An invalid explicit path is an error and never falls back to auto-detection.

```sh
blitz-client client locate
blitz-client client tanks list --json
blitz-client client tanks list --nation ussr

blitz-client client export \
  --game-dir "/path/to/Tanks Blitz" \
  --output-dir "/path/to/export"

blitz-client client export \
  --game-dir "/path/to/Tanks Blitz" \
  --output-dir "/path/to/export" \
  --data-only

blitz-client client tanks export \
  --viewer-dir "/path/to/viewer" \
  --limit 10

blitz-client client tanks export \
  --viewer-dir "/path/to/viewer" \
  --tank china/Ch48_BZ_75
```

`client tanks list` reports the vehicles with both showroom model files installed. The JSON response contains `count` and a `tanks` array with the ID, localized name, nation, level, `userString`, and source model paths.

Model discovery uses the vehicle XML node name as the tank ID. Localization,
`userString`, and level are optional; the display name falls back to the node
name with underscores replaced by spaces. Missing or invalid price, shop, and
configuration metadata does not prevent model discovery. Full catalog export
continues to reject incomplete or invalid required tank metadata, including
records without model files. Unreadable resources and invalid XML or DVPL
containers remain errors in both commands.

`client export` writes `maps.json` and `tanks.json` into the output directory, then exports the same tank model bundles as `client tanks export` under `assets/tanks`. The two metadata files preserve the former Python JSON export fields and localized Russian strings. Model filters such as `--tank`, `--nation`, and `--limit` affect only model bundles; the metadata files always contain the full client catalog. Add `--data-only` to skip model conversion.

The combined exporter reads plain client resources and `.dvpl` resources through the same client reader used for model discovery. It does not process map images or tank icons and does not make API requests. The legacy `client tanks export --viewer-dir` command remains available for model-only exports.

The model exporter is implemented in Rust and writes to `<viewer-dir>/assets/tanks` (or `<output-dir>/assets/tanks` for the combined command). It generates the viewer's `catalog.json` format (`formatVersion: 1`, `count`, `entries`) and `export-report.json`. Each successful entry points to the model bundle, optional armor files, and texture assets using the same URL layout the viewer already reads. Tank bundles are left in place unless `--prune` is explicitly supplied.

The following model options apply to both model export commands:

- `--tank <ID|nation/ID>` can be repeated. An unqualified ID must be unique across nations.
- `--nation <nation>` filters automatic discovery; `--limit N` exports at most N successful models. A limit of 0 means all models.
- `--max-texture-size original|N|WIDTHxHEIGHT` caps source textures; the default is `1024x1024`.
- `--texture-format webp|png` selects texture format; WebP is the default. `--webp-quality 0..100` sets lossy base-color quality.
- `--pbr` exports the PBR maps and tangent basis. `--no-compress` writes plain model JSON and geometry instead of `.br` assets.
- `--force` is accepted for explicit full regeneration. Texture caching is not enabled in this Rust exporter yet, so exports currently rewrite textures on every run. `--prune` removes bundles omitted from the new catalog.

SC2/SCG decoding, mesh transforms, geometry generation, armor joining, catalogs, and reports are Rust code. Texture conversion uses `ffmpeg`; WebP output uses `cwebp`; Brotli model compression uses the `brotli` command. Add these codec executables to `PATH` for the corresponding export options. Python is not used by this application.

SCG groups share binary-layout, buffer-size, and triangle-index validation.
Geometry is decoded only for groups referenced by the selected scene nodes;
unused groups still require valid, unique IDs. Used groups with unknown vertex
layout bits or overflowing counts produce an explicit error. Showroom UV/normal
repairs and collision armor-zone checks remain specific to their export paths.

## Replay analysis

`replay inspect` decodes a `.tbreplay` ZIP archive: `meta.json`, the protocol-2 pickle `battle_results.dat` and, with `--data`, the framed `data.replay` packet stream with the recovered telemetry (shots, hits, health and transform updates, booster counters, task markers, packet annotations). A human-readable summary goes to stdout, `--json` prints the whole document and `--out <path>` writes it to a file.

`replay inspect data.replay` also accepts the standalone stream (detected by its magic or `.replay` extension). It always includes packets and telemetry; `--data` is unnecessary. `meta` and `battle_results` are `null`, as are result-based counters and comparisons. Player names from final results and exact result-snapshot matching are unavailable. Complete standalone and archived streams use the same incremental decoder; a trailing incomplete header or packet is an error in inspect mode.

`replay follow data.replay` reads from the beginning and then polls every **25 ms** by default. It waits for file creation, buffers partial headers and packets, and emits complete packets immediately as **NDJSON**, flushing every line. Use `--poll-ms 10` to change the interval. The output contains:

- `header`: `session_id`, `client_version`, `header_bytes`.
- `packet`: `session_id`, `seq`, absolute byte `offset`, `packet_type`, `clock`, decoded `payload`, hex `raw_payload`, and `telemetry`.
- `reset`: new `session_id` and a reason (`file_removed`, `file_replaced`, `file_truncated`, or `file_rewritten`). Sequence numbers and HP state restart for the new session.

Live telemetry uses the same packet layouts as inspect and keeps previous HP per entity. Its records describe the current packet; linking shots to launches and hits to HP updates is performed by the complete inspect report. For unsupported client versions, packet `telemetry` is `null` and raw payloads remain available.

Errors go to stderr, including with `--json`; follow always writes compact NDJSON. `--out <new-path>` writes to a new file instead of stdout and refuses to overwrite existing files. Ctrl+C stops following. `--idle-timeout-ms <N>` optionally stops after no new bytes or session resets for the specified duration (also applies while waiting for creation). An idle timeout does not certify a complete replay and may leave an incomplete frame unreported.

Following expects append-only writes within a session. It detects replacement by file identity (device/inode on Unix, creation time elsewhere), observed truncation, and changes to the last 64 read bytes. A rewrite that restores those same bytes can escape detection. Consumer callbacks run synchronously, providing backpressure without accumulating the whole live history. The library exposes `ReplayStreamDecoder::feed/finish`, `TelemetryDecoder::on_packet`, and `ReplayFollower::poll`; callers control scheduling and cancellation. Payload size is limited to 16 MiB. Actual latency also depends on how frequently the game flushes its recording buffer.

`replay latest` picks the newest `.tbreplay` of the platform replay folder (macOS: `~/Library/Containers/com.tanksblitz/Data/Documents/tb-project/replays`); `--dir <path>` selects another folder.

Packet layouts were verified on `26.8.0_ruby_apple` replays. Other client versions still decode the archive and battle results, keep every raw payload and report `unsupported-client-version` telemetry. Pickle values are read as builtins only: the decoder never imports or runs Python code.

### Simulating a growing replay file

A real replay copy is included at `tests/fixtures/sample.tbreplay`. The Python
standard-library script extracts its `data.replay` and writes it in chunks,
flushing after every write:

```sh
python3 scripts/simulate-replay.py
```

Each run creates `/tmp` (or the platform temp directory) with a random
`blitz-live-replay-*` subdirectory containing `data.replay`. The script prints the
full destination and a `cargo run -- replay follow ...` command, waits three
seconds, then writes 1,024 bytes every 25 ms. Start the printed follower command
in another terminal; it also reads any bytes written before it starts. The
completed or interrupted file is kept for inspection.

For a known destination, start the follower first in one terminal:

```sh
cargo run -- replay follow /tmp/live-test/data.replay
```

Then start the writer in another terminal:

```sh
python3 scripts/simulate-replay.py --output /tmp/live-test/data.replay
```

The destination must be a new file; existing files are never overwritten. Remove
the old test output or choose another path before repeating. You can supply a
different archive or standalone stream and change the write cadence:

```sh
python3 scripts/simulate-replay.py /path/to/battle.tbreplay \
  --output /tmp/another-test/data.replay \
  --chunk-size 127 --interval-ms 10 --start-delay 0
```

Chunks may split both headers and packet bodies, exercising the follower's
incomplete-frame buffering. Cadence simulates filesystem appends; packet clocks
inside the original recording are preserved and do not determine write timing.
Use `--interval-ms 0 --start-delay 0` for a fast copy. Stop the follower with
Ctrl+C after the writer finishes, or configure its `--idle-timeout-ms` longer
than the initial wait and chunk interval.

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
use blitz_client::{
    ClientExportOptions, TankExportOptions, export_client, export_tank_catalog, install_mod,
    list_client_tanks,
    remove_mod, update_from_git_diff, validate_mod,
};

let game_dir: Option<PathBuf> = None;
let tanks = list_client_tanks(game_dir.clone())?;
let mut export = TankExportOptions::new("/path/to/viewer");
export.game_dir = game_dir.clone();
export.limit = 10;
let catalog = export_tank_catalog(&export)?;

let mut client_export = ClientExportOptions::new("/path/to/export");
client_export.game_dir = game_dir.clone();
client_export.data_only = true;
let exported = export_client(&client_export)?;

let report = validate_mod("mods/NoMapName", game_dir.clone())?;
if report.valid {
    install_mod("mods/NoMapName", game_dir)?;
}
remove_mod("NoMapName", None)?;
update_from_git_diff("mods/NoMapName", "mods/NoMapName", Some("HEAD"))?;
# let _ = (tanks, catalog, exported);
# Ok::<(), blitz_client::InstallerError>(())
```
