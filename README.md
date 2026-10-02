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
| `export` | Unpacked client files, client data and tank bundles, model/armor conversion | `client` |
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
unchanged. Replay `Packet`, `Payload`, and `EntityMethod` are local types exported
from `replay`. Packet decoding and replay schemas live in `src/replay/models/`;
`prost` and `serde-pickle` handle the generic protobuf and pickle formats. The
adapted schemas retain their MIT attribution and license in that directory.

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
blitz-client client files export --output-dir <path> [--game-dir <path>] [--cache-dir <path> | --no-cache] [--types <types>] [--exclude <pattern>] [--dry-run] [--json]
blitz-client client tanks list [--game-dir <path>] [--nation <nation>] [--json]
blitz-client client export --output-dir <path> [--game-dir <path>] [--data-only] [model options]
blitz-client client tanks export --viewer-dir <path> [--game-dir <path>] [options]

blitz-client mod validate <mods-dir|mod-dir|mod.yaml|mod.yml> [--game-dir <path>] [--json]
blitz-client mod install <mod-dir|mod.yaml|mod.yml> [--game-dir <path>] [--json]
blitz-client mod uninstall <mod-id> [--game-dir <path>] [--json]
blitz-client mod update-from-git <mod-dir|mod.yaml|mod.yml> --repo <path> [--base <rev>] [--json]

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

## Unpacked client files

Export the effective client resource tree to a directory:

```sh
blitz-client client files export --output-dir ./Data
```

Count the selected files and their stored size without creating or changing the
output directory, reading file contents, or unpacking DVPL:

```sh
blitz-client client files export --output-dir ./Data --types configs,images --dry-run
```

Dry runs apply the same filters and source precedence as exports. Only the
winning source for each output path is counted. Sizes include the DVPL container
as stored on disk. `--json` reports `dry_run`, `file_count`, `source_bytes`,
`dvpl_count`, and `cache_file_count`; written/unchanged counts and `output_bytes`
remain zero. Source and output path validation still applies.

All files are selected by default. The exporter merges the installation's
resource directory with the standard `packs` cache. Cache files override the
same relative path from the installation, regardless of modification times.
DVPL containers are decoded and their final `.dvpl` suffix is removed:
`UI/example.yaml.dvpl` becomes `UI/example.yaml`. Other files are copied as-is,
preserving the directory structure and original resource formats.

Use `--types` to select comma-separated categories (the option can be repeated):

```sh
blitz-client client files export --output-dir ./Data --types configs,images
blitz-client client files export --output-dir ./Models --types models
```

Categories are selected by the extension after removing `.dvpl`, independently
of the directory name:

| Type | Extensions |
| --- | --- |
| `all` | Every file, including unknown extensions |
| `configs` | `yaml`, `yml`, `xml`, `xsd`, `txt`, `json`, `ini`, `cfg`, `toml`, `csv`, `tab`, `actions`, `style`, `material`, `html`, `htm` |
| `images` | `webp`, `png`, `jpg`, `jpeg`, `dds`, `pvr`, `tex`, `tga`, `bmp`, `gif`, `svg`, `ktx`, `ktx2` |
| `models` | `sc2`, `scg`, `anim`, `animation`, `model`, `mesh`, `heightmap`, `mkm`, `lka` |
| `sounds` | `bnk`, `pck`, `wem`, `wav`, `ogg`, `mp3`, `flac`, `fsb` |
| `fonts` | `ttf`, `otf`, `fnt`, `woff`, `woff2` |
| `shaders` | `sl`, `slh`, `glsl`, `hlsl`, `vert`, `frag`, `metal` |
| `other` | All extensions not assigned to the categories above, and extensionless files |

Models remain in their native game formats; the existing `client tanks export`
command produces viewer bundles. Selecting `models` does not automatically
include textures; use `models,images` to include both.

Exclude files using repeatable glob patterns:

```sh
blitz-client client files export --output-dir ./Data \
  --exclude '3d/**' --exclude '**/*.dds' --dry-run
```

Patterns match the entire unpacked path relative to `Data` or `packs`, with
case-sensitive matching. Use `/` as the separator: `*` matches within one path
component, `?` matches one character, `**` matches across directories, and
`[abc]`/`[!abc]` match character sets. For example, `**/*.dds` excludes
`Gfx/tank.dds.dvpl` as well as plain `.dds` files. Quote patterns to prevent
shell expansion. Multiple patterns are combined with OR. Invalid patterns
fail before writing anything. Exclusions apply after source precedence and
category selection, so an excluded cache file does not fall back to an
installation copy. Existing destination files are retained. JSON reports
`exclude` and `excluded_file_count` (the number of effective paths excluded).

Override the sources or disable the cache overlay:

```sh
blitz-client client files export --game-dir /path/to/game \
  --cache-dir /path/to/packs --output-dir ./Data --types configs
blitz-client client files export --output-dir ./Data --no-cache
```

A missing default cache is skipped. An explicitly supplied `--cache-dir` must
exist. `--cache-dir` and `--no-cache` cannot be combined. Within each source tree,
an unpacked file takes precedence over its `.dvpl` sibling; the cache still has
priority over the installation when their stored filenames differ.

Repeated exports keep byte-identical destination files and replace changed
ones. Unrelated destination files are retained. Source symlinks are skipped and
counted in the report; symlinks within the destination tree and overlapping
source/output directories are rejected. An invalid DVPL stops the export with its source path
in the error; files already exported are kept.

Text mode reports progress to stderr and a final summary to stdout. `--json`
outputs one report with source paths, selected categories, written/unchanged
file counts, cache/DVPL counts, decoded byte totals, and elapsed time.

The `export` Cargo feature exposes `export::export_client_files`,
`ClientFilesExportOptions`, `ClientFilesExportReport`, and `ClientFileType`.
`export_client_files_with_progress` also accepts a synchronous callback for
progress reporting or cancellation. This export requires no catalog metadata.

## Replay analysis

`replay inspect` decodes a `.tbreplay` ZIP archive: `meta.json`, the protocol-2 pickle `battle_results.dat` and, with `--data`, the framed `data.replay` packet stream with the recovered telemetry (shots, hits, health and transform updates, booster counters, task markers, packet annotations). A human-readable summary goes to stdout, `--json` prints the whole document and `--out <path>` writes it to a file.

`replay inspect data.replay` also accepts the standalone stream (detected by its magic or `.replay` extension). It always includes packets and telemetry; `--data` is unnecessary. `meta` and `battle_results` are `null`, as are result-based counters and comparisons. Player names from final results and exact result-snapshot matching are unavailable. Complete standalone and archived streams use the same incremental decoder; a trailing incomplete header or packet is an error in inspect mode.

`replay follow data.replay` reads from the beginning and then polls every **25 ms** by default. It waits for file creation, buffers partial headers and packets, and emits complete packets immediately as **NDJSON**, flushing every line. Use `--poll-ms 10` to change the interval. The output contains:

- `header`: `session_id`, `client_version`, `header_bytes`.
- `packet`: `session_id`, `seq`, absolute byte `offset`, `packet_type`, `clock`, decoded `payload`, hex `raw_payload`, and `telemetry`.
- `reset`: new `session_id` and a reason (`file_removed`, `file_replaced`, `file_truncated`, or `file_rewritten`). Sequence numbers and HP state restart for the new session.

Live telemetry uses the same packet layouts as inspect and keeps previous HP per entity. Its records describe the current packet; linking shots to launches and hits to HP updates is performed by the complete inspect report. For versions with neither supported packet layouts nor battle counters, packet `telemetry` is `null` and raw payloads remain available.

Errors go to stderr, including with `--json`; follow always writes compact NDJSON. `--out <new-path>` writes to a new file instead of stdout and refuses to overwrite existing files. Ctrl+C stops following. `--idle-timeout-ms <N>` optionally stops after no new bytes or session resets for the specified duration (also applies while waiting for creation). An idle timeout does not certify a complete replay and may leave an incomplete frame unreported.

Following expects append-only writes within a session. It detects replacement by file identity (device/inode on Unix, creation time elsewhere), observed truncation, and changes to the last 64 read bytes. A rewrite that restores those same bytes can escape detection. Consumer callbacks run synchronously, providing backpressure without accumulating the whole live history. The library exposes `ReplayStreamDecoder::feed/finish`, `TelemetryDecoder::on_packet`, and `ReplayFollower::poll`; callers control scheduling and cancellation. Payload size is limited to 16 MiB. Actual latency also depends on how frequently the game flushes its recording buffer.

`client locate` also returns `replay_dir` in JSON and prints the replay directory
in text output. This is the fixed OS path, independent of `--game-dir`, and is
returned even when the folder does not exist yet:

- macOS: `~/Library/Containers/com.tanksblitz/Data/Documents/tb-project/replays`
- Windows: `%USERPROFILE%\Documents\TanksBlitz\replays`, as documented by
  [Tanks Blitz support](https://lesta.ru/support/ru/products/tb/article/16159/).

`client::default_replay_dir()` and `replay::default_replay_dir()` expose the same
path calculation to library callers. On other operating systems, or when the
required home-directory environment variable is missing, this function returns
`REPLAY_FOLDER_NOT_FOUND` and `client locate` reports `replay_dir: null`.

`replay latest` picks the newest `.tbreplay` in this folder; `--dir <path>`
selects another folder, including on operating systems without a configured
default path.

Full packet telemetry is verified on `26.8.0_ruby_apple`. Battle counters are
also verified on `26.10.0_ruby_apple`; inspect reports `battle-counters-only`
for that version and follow emits counters while leaving other telemetry lists
empty. Other versions retain raw packets and report unsupported telemetry.
Pickle values are read as builtins only: the decoder never imports or runs
Python code.

### Live assisted and blocked damage

`replay follow` emits the author's cumulative server-reported values during
the battle, without reading `battle_results.dat` or estimating damage from HP:

- `telemetry.battle_counters.damage_blocked`: damage blocked by armor.
- `telemetry.battle_counters.damage_assisted`: combined damage assisted by
  spotting or immobilizing opponents. The packet does not separate these kinds.
- `telemetry.battle_counter_updates`: updates from the current packet, with
  `counter`, `value`, `previous_value`, `delta`, `clock`, `packet_index`,
  `entity_id`, and `ribbon_id`.

Use `value` as the new absolute total. `delta` is the difference from the
previous report; it is `null` on the first update, zero for a duplicate, and
may be negative for a correction. Each counter is `null` until its first
update, so an unavailable value is not presented as zero. Totals are retained
across polls and cleared when the follower starts a new recording session.

For example, a packet from the real test recording includes:

```json
{
  "battle_counters": {
    "author_nickname": "iCooLER",
    "player_entity_id": 11025794,
    "damage_blocked": 540,
    "damage_assisted": 706
  },
  "battle_counter_updates": [{
    "packet_index": 48418,
    "clock": 163.58837890625,
    "entity_id": 11025794,
    "ribbon_id": 5,
    "counter": "damage_blocked",
    "value": 540,
    "previous_value": null,
    "delta": null
  }]
}
```

The verified layout is packet type 8, EntityMethod 14: a 6-byte body with
ribbon ID at offset 12 of the payload and cumulative `u16` value at offset 16.
The receiver must match the author entity ID from the BasePlayerCreate prefix.
Ribbon IDs are confirmed by the client's `XML/item_defs/ribbons.xml`: 5 is
`ribbonArmorBlock`, 17 is `ribbonDamageAssist`. Localized descriptions confirm
that the latter includes both spotting and immobilization assistance.

The test replay reports assistance 706 → 1083 → 1499 → 1920 during the battle,
and blocked damage 540 at 163.588 seconds. These match the final results.
A checked 26.8 replay reported assistance 420 → 732, matching the sum of its
two assisted-damage result fields (312 + 420).

`replay inspect --data` includes the same `battle_counters` and update history,
plus `battle_counter_result_comparison` for completed archives. An absent
counter or absent final results produces a `null` comparison, not a mismatch.
Standalone streams use the same decoder. The public `replay::BattleCounterDecoder`
API can also consume complete packets directly; `counters()` returns its typed
state and `reset()` starts a new session. Support for these two versions does
not enable unverified layouts for other packet methods.

### Live game recording

During a battle, the game creates a directory named
`recording_<nickname>.tbreplay` inside the replay folder. `<nickname>` is the
player's nickname: for example, `iCooLER` in `recording_iCooLER.tbreplay`.
Despite its `.tbreplay` suffix, this recording is a directory, rather than a
completed ZIP archive.

The live recording inspected on macOS on 2026-10-01 contained only:

```text
replays/
└── recording_iCooLER.tbreplay/
    └── data.replay
```

`data.replay` grew during the battle: observed sizes were 65,536, 131,072,
and 258,048 bytes. Updates arrive when the game flushes its buffer, so polling
more frequently cannot expose bytes that the game has not written yet. These
sizes are observations, not a guaranteed write-chunk size.

The file uses the same framing as a standalone replay: a stream header followed
by packets with a little-endian payload length (`u32`), packet type (`u32`),
clock in seconds (`f32`), and payload bytes. In this recording the header was
65 bytes and identified client `26.10.0_ruby_apple`. The follower successfully
decoded 2,959 packets from the 131,072-byte snapshot, through clock 32.387
seconds, with no incomplete packet remaining at that snapshot's end. Packet
framing and the battle counters described above work for this version; other
telemetry layouts remain version-gated.

Follow the live file directly, replacing `iCooLER` with the player's nickname:

```sh
blitz-client replay follow ~/Library/Containers/com.tanksblitz/Data/Documents/tb-project/replays/recording_iCooLER.tbreplay/data.replay
```

The follower can start before the directory or file exists and waits for its
creation. It buffers any incomplete packet at a temporary end of file.
`meta.json` and `battle_results.dat` were absent from the inspected live
directory; the completed test replay ZIP contains both alongside `data.replay`.

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

To simulate the game's recording layout in its real replay folder:

```sh
python3 scripts/simulate-replay.py --game-replays
```

This uses the same macOS/Windows default paths as `client locate` and creates
`recording_ReplaySimulator.tbreplay/data.replay`. Choose another nickname with
`--nickname`; the nickname only changes the directory name, not player names or
other bytes inside the source replay:

```sh
python3 scripts/simulate-replay.py --game-replays --nickname iCooLER
```

The recording directory must not already exist, even if it is empty. The script
refuses to reuse it to protect an ongoing real battle. Choose another nickname
for repeated runs. After writing, the directory is kept for inspection; the
script does not package it into a finished replay ZIP.

Use a specific replay folder (also on OSes without a configured default):

```sh
python3 scripts/simulate-replay.py --replay-dir /tmp/test-replays --nickname TestPlayer
```

`--game-replays`, `--replay-dir`, and `--output` are mutually exclusive.
Chunk sizes, delays, and source selection work the same in all modes.

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

Run the simulator checks (they write only into temporary directories):

```sh
python3 -B -m unittest discover -s tests -p test_simulate_replay.py
```

## Mod commands

`mod validate <directory>` recursively finds `mod.yaml` and `mod.yml`
in the directory and its subdirectories (without following directory symlinks). Each mod is
validated independently against the game and already installed mods; this does
not check whether all discovered mods can be installed together. No files are
changed. A failed mod does not stop validation of the others.

```sh
blitz-client mod validate ./mods --game-dir /path/to/game
```

For a collection, `--json` returns `{ "valid": bool, "mods": [...] }`, with each
entry containing `manifest` and either `report` or `error`. A single mod directory
or explicit manifest keeps the existing single-report format. Exit status is
`0` when all mods pass, `4` when any collection member fails, and `2` for a scan
error or when no manifests are found.

The logical target in a manifest omits the final `.dvpl` suffix. The installer resolves `UI/.../Minimap.yaml` to either `Minimap.yaml.dvpl` or a plain `Minimap.yaml`, rejecting the case where both exist.

Supported actions under `files`:

- `replace`: replace the single exact match of `search` with `replace`.
- `before` / `after`: insert `insert` before / after the single exact match of
  `find`, preserving the matched text. No whitespace or newlines are added
  automatically.
- `clear`: empty the resource payload, including binary resources; no `search` or
  `replace` is required. DVPL resources remain valid DVPL containers with an empty
  payload. This does not generate a transparent image.
- `patch`: apply a unified diff from the file specified by `patch`.

Text actions require UTF-8 resources. `search` is a string. Multiple operations for
one file run in manifest order. Different logical paths resolving to the same
physical resource are rejected.

YAML manifests group ordered operations under each resource path:

```yaml
version: 1
id: CameraPosition
name: Camera position
files:
  optionsDesktop.yaml:
    - replace:
        ignore-trailing-whitespace: true
        search: |2
              Camera fixed positions:
                  keyedArchive:
                      1) Arcade:
                          keyedArchive:
                              max:
                                  float: 40.0000
        replace: |2
              Camera fixed positions:
                  keyedArchive:
                      1) Arcade:
                          keyedArchive:
                              max:
                                  float: 24.0000
```

For `before` and `after`, use `insert`
(or `replace`) for the inserted text. Other actions use `clear: {}` or
`patch: { patch: "patches/resource.patch" }`. YAML rejects unknown fields and
unsupported versions; `version` defaults to `1`. A directory must contain only
one manifest format, including when validating a collection.

`ignore-trailing-whitespace: true` applies only to its own `replace`, `before`
or `after` operation. It ignores spaces/tabs at line ends and LF/CRLF differences
while searching. Leading indentation, internal whitespace, case and blank lines
are preserved. Exactly one match is required. Only the matched range is changed;
replacement text is written as supplied, and insertions preserve the original
matched text. Without the flag (or with `false`), matching remains exact.

The option is saved with installed operations, so subsequent composition and
uninstallation use the same matching behavior. Older installer state defaults
to exact matching.

Exact replacement:

```yaml
id: NoMapName
name: Remove map name above the minimap
files:
  UI/Screens/Battle/Minimap.yaml:
    - replace:
        search: exact text to find
        replace: ""
```

Unified diff patches are stored as sidecar files:

```yaml
files:
  UI/Screens/Battle/Minimap.yaml:
    - patch:
        patch: patches/UI/Screens/Battle/Minimap.yaml.<diff-crc>.patch
        generated-by: git
```

Only YAML mod manifests are supported (`mod.yaml` or `mod.yml`). Resource paths
are emitted without quotes where YAML permits it; quoting is added only when
needed to preserve a string value.
`update-from-git` reserializes YAML, retaining operation values but not comments
or formatting.

`update-from-git` compares the selected base revision (default `HEAD`) with the working tree, including staged and unstaged changes. Repository paths must mirror the game's unpacked resource paths. It handles modifications to existing tracked text resources; additions, deletions, renames, and binary changes are rejected. `.dvpl` files must be unpacked before generating a diff.

The game must be closed during installation and uninstallation. Active mods are recomposed from the current game resources and saved installation state, so mod removal order does not affect the remaining patches. The installer does not use a fixed game-file hash to identify a clean client.

## Game directory defaults

The ordered installation candidates are implemented in `src/client/location.rs`.

- Windows: `C:\Games\Tanks_Blitz` first (resources in its `Data` subdirectory), followed by standard Steam, WindowsApps, and user game folders.
- macOS: `/Applications/Tanks Blitz.app/Contents/Resources/Data` first, then the Steam installation under the user's Library.
- Linux: common Steam, Flatpak, and Snap installation folders.

`client locate` returns `game_dir` (installation), `resource_dir` (installed
resources), `cache_dir` (downloaded resources in `packs`), and `replay_dir`.
The cache path is independent of `--game-dir` and does not need to exist yet.
It is also available through `client::default_cache_dir()`:

- macOS: `~/Library/Containers/com.tanksblitz/Data/Library/Caches/packs`
- Windows (Tanks Blitz): `%USERPROFILE%\Documents\TanksBlitz\packs`, confirmed
  in the inspected wblitz and BlitzHata mod sources. This path has not been
  verified on a Windows installation of this project.

This Windows default targets Tanks Blitz (Lesta). On other operating systems,
or when the required
home-directory environment variable is missing, `default_cache_dir()` returns
`CACHE_DIR_NOT_FOUND` and `client locate` reports `cache_dir: null`.

The inspected Windows mods use the following paths beneath the Windows
Documents folder:

| Path | Purpose | Source |
| --- | --- | --- |
| `TanksBlitz\packs` | Downloaded resource cache | wblitz `ModInstaller.cs`, BlitzHata `scripts/lib-client.js` |
| `TanksBlitz\replays` | Battle recordings | wblitz `MainWindow.cs`, BlitzHata `scripts/live-battle.js` |
| `TanksBlitz\cache` | Player dossier data | wblitz `DossierUploader.cs`, `DossierMarks.cs` |
| `TanksBlitz\battle_results\<accountId>_Russia` | Battle results | wblitz `BattleResultsUploader.cs` |

The mods account for Windows Documents-folder redirection (for example to
OneDrive): wblitz uses `Environment.GetFolderPath(SpecialFolder.Personal)`, and
BlitzHata accepts the resolved folder through `BH_DOCUMENTS_DIR`. This project's
current defaults assume `%USERPROFILE%\Documents`.

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
