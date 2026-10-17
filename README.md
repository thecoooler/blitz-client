# Tanks Blitz Client Tools

Go library and CLI for inspecting the installed Tanks Blitz client, exporting tank models for the local viewer, installing resource mods, and analyzing complete or live replays. This is a Go port of the Rust `blitz-client`; behaviour, CLI, JSON output, and exit codes match the original.

## Installation and packages

```sh
go install github.com/thecoooler/blitz-client/cmd/blitz-client@latest
# or, from a checkout:
go build ./cmd/blitz-client
go run ./cmd/blitz-client client locate
go test ./...
```

The module is `github.com/thecoooler/blitz-client`. Go compiles only the packages
you import, so there are no feature flags: import just `replay` and nothing else
is built. CLI argument parsing and output formatting live in `internal/cli`; the
executable is `cmd/blitz-client`.

| Package | Responsibility | Depends on |
| --- | --- | --- |
| `blitzclient` (root) | Shared `Error` type (`NewError`, `ErrorAt`, `AsError`), default cache and replay directories | none |
| `resources` | DVPL codecs | root |
| `resources/xml`, `resources/archive`, `resources/sprite` | Client XML parser, SC2/SCG archives, sprite descriptors | `resources` |
| `client` | Client location, resource lookup, localization, tank and map catalogs | `resources` |
| `export` | Unpacked client files, client data, icons and tank bundles, model/armor conversion | `client` |
| `mods` | Manifests, validation, installation, removal, Git patches | `client` |
| `replay` | Archive/standalone analysis, byte framing, file following, telemetry | none |
| `game` | Running client process, window focus and state for overlays | none |
| `internal/cli` | The `blitz-client` command line | all of the above |

Module dependencies run from high-level operations toward resource readers:
`export` and `mods` use `client` and `resources`; `client` uses `resources`;
`replay` is independent. `resources` never imports the exporter. Replay packet
decoding and schemas live in the `replay` package; the adapted schemas retain
their MIT attribution and license in `replay/LICENSE-schemas`.

Library-only examples:

```sh
go run ./examples/replay_follow ./data.replay
go run ./examples/client_catalog /path/to/Data
go run ./examples/decode_resource /path/to/file.dvpl
```

Run `bash scripts/check.sh` for the gofmt check, `go vet`, `go test`, and `go build`.

## Differences from the Rust implementation

- Maps inside `BasePlayerCreate` arguments (`players_battle_categories_ids`, `player_wait_times`) are emitted with sorted keys in `replay follow` NDJSON; Rust emits them in random `HashMap` order.
- `blitz-client --json` with no command reports a `USAGE` error instead of panicking.
- DVPL containers recompressed by the mod installer with LZ4/DEFLATE may differ byte-wise from the Rust encoder while decoding to identical content. Installer state stays compatible both ways: mods installed by one implementation can be removed by the other.
- Operating-system error texts come from Go and may differ slightly.
- Diagnostics for malformed inputs keep the same error codes, but their message text can differ: YAML syntax errors in manifests have no `at line L column C, while …` suffix, and JSON, pickle, and protobuf decoding errors use Go wording.
- A manifest path given relative to the current directory (`mod validate mod.yaml` run inside the mod folder) works with patch sources; the Rust tool fails to resolve the empty parent directory.
- An explicitly empty `--game-dir ""` is rejected while arguments are parsed, so it is reported before other errors.
- The CLI uses cobra for the command tree, but leaf commands parse arguments themselves to keep the original options and diagnostics.

## CLI

The executable is `blitz-client`.

```text
blitz-client client locate [--game-dir <path>] [--json]
blitz-client client files export --output-dir <path> [--game-dir <path>] [--cache-dir <path> | --no-cache] [--types <types>] [--exclude <pattern>] [--dry-run] [--json]
blitz-client client tanks list [--game-dir <path>] [--nation <nation>] [--json]
blitz-client client export --output-dir <path> [--game-dir <path>] [--data-only] [model options]
blitz-client client tanks export --viewer-dir <path> [--game-dir <path>] [options]

blitz-client mod list [--game-dir <path>] [--json]
blitz-client mod validate <mods-dir|mod-dir|mod.yaml|mod.yml> [--game-dir <path>] [--json]
blitz-client mod install <mod-dir|mod.yaml|mod.yml> [--game-dir <path>] [--json]
blitz-client mod uninstall <mod-id> [--game-dir <path>] [--json]
blitz-client mod update-from-git <mod-dir|mod.yaml|mod.yml> --repo <path> [--base <rev>] [--json]

blitz-client replay inspect <replay.tbreplay|data.replay> [--data] [--out <path>] [--json]
blitz-client replay latest [--dir <path>] [--data] [--out <path>] [--json]
blitz-client replay follow <data.replay> [--poll-ms <N>] [--idle-timeout-ms <N>] [--out <new-path>]

blitz-client game status [--process-name <name>]... [--watch] [--poll-ms <N>] [--json]
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

`client export` writes `maps.json`, `tanks.json`, and `items.json` into the output directory, exports icons under `assets/icons` (see [Items and icons](#items-and-icons)), then exports the same tank model bundles as `client tanks export` under `assets/tanks`. The two metadata files preserve the former Python JSON export fields and localized Russian strings. Model filters such as `--tank`, `--nation`, and `--limit` affect only model bundles; the metadata files, items, and icons always cover the full client. Add `--data-only` to skip model conversion; icons are still exported.

The combined exporter reads plain client resources and `.dvpl` resources through the same client reader used for model discovery. It does not process map images or tank icons and does not make API requests. The legacy `client tanks export --viewer-dir` command remains available for model-only exports.

The model exporter is implemented in Go and writes to `<viewer-dir>/assets/tanks` (or `<output-dir>/assets/tanks` for the combined command). It generates the viewer's `catalog.json` format (`formatVersion: 1`, `count`, `entries`) and `export-report.json`. Each successful entry points to the model bundle, optional armor files, and texture assets using the same URL layout the viewer already reads. Tank bundles are left in place unless `--prune` is explicitly supplied.

The following model options apply to both model export commands:

- `--tank <ID|nation/ID>` can be repeated. An unqualified ID must be unique across nations.
- `--nation <nation>` filters automatic discovery; `--limit N` exports at most N successful models. A limit of 0 means all models.
- `--max-texture-size original|N|WIDTHxHEIGHT` caps source textures; the default is `1024x1024`.
- `--texture-format webp|png` selects texture format; WebP is the default. `--webp-quality 0..100` sets lossy base-color quality.
- `--pbr` exports the PBR maps and tangent basis. `--no-compress` writes plain model JSON and geometry instead of `.br` assets.
- `--force` is accepted for explicit full regeneration. Texture caching is not enabled in this Go exporter yet, so exports currently rewrite textures on every run. `--prune` removes bundles omitted from the new catalog.

SC2/SCG decoding, mesh transforms, geometry generation, armor joining, catalogs, and reports are Go code. Texture conversion uses `ffmpeg`; WebP output uses `cwebp`; Brotli model compression uses the `brotli` command. Add these codec executables to `PATH` for the corresponding export options. Python is not used by this application.

SCG groups share binary-layout, buffer-size, and triangle-index validation.
Geometry is decoded only for groups referenced by the selected scene nodes;
unused groups still require valid, unique IDs. Used groups with unknown vertex
layout bits or overflowing counts produce an explicit error. Showroom UV/normal
repairs and collision armor-zone checks remain specific to their export paths.

## Items and icons

`client export` writes `items.json` with the shell types, the regular consumables, provisions, and equipment, and the crew skills, and exports their icons as PNG under `assets/icons`. Event-specific items and icon folders are not included.

### Items

A shell's icon depends only on its type, so `shells` lists shell types rather than individual shells. The types come from the `<icons>` table of each available nation's `components/shells.xml`, merged in first-seen order; a type must have the same image name in every nation. The type `id` is the key shells use in `<icon>`. `premium` comes from the `_PREMIUM` suffix of `name`, and `userString` is the localized `<name without _PREMIUM>_FULL` string, empty when the client has none (as for ATGM types).

Consumables, provisions, and equipment come from `XML/item_defs/vehicles/common/consumables/common.xml`, `provisions/common.xml`, and `optional_devices.xml`; every element with an `<id>` is an item. A missing list contributes nothing.

```json
{
  "shells": [
    { "id": "ap_cr_premium", "name": "ARMOR_PIERCING_CR_PREMIUM", "premium": true,
      "userString": "Подкалиберный снаряд", "icon": "assets/icons/shells/ap_cr_premium_l.png" }
  ],
  "consumables": [
    { "id": 45, "name": "tungstenTip", "userString": "Вольфрамовые сердечники", "description": "…",
      "iconKey": "tungstentip", "icon": "assets/icons/consumables/consumable_tungsten-tip_m.png" }
  ],
  "provisions": [],
  "equipment": []
}
```

Crew skills come from the `<skills>` of `XML/item_defs/tankmen/avatar.xml`, in XML order. `id` is the skill's element name; `userString`, `description`, `tip`, and `features` are localized from `<userString>`, `<effectDescription>`, `<tipDescription>`, and `<featuresDescription>` and are empty when absent. `tankClass` is the `<skillsByClasses>` entry that lists the skill, and `type` is its `<type>` (`trigger` or `continuous`). A skill listed for two classes fails the export.

```json
{ "id": "skilled_ranging", "userString": "Лихая пристрелка", "description": "…", "tip": "…", "features": "…",
  "tankClass": "lightTank", "type": "continuous", "icon": "assets/icons/crew-skills/skilled_ranging_1.png" }
```

Item `userString` and `description` are localized from `Strings/ru.yaml` and keep the key when no translation exists; hidden service items have none and export empty strings. `icon` is relative to the export directory and is `null` when there is no icon.

`iconKey` is the item's `<icon>`; shell types and crew skills are keyed by their `id`. The client UI, not the item, decides which sprite shows for a key, so the exporter reads the same UI resources:

| Items | UI resource | Selector |
| --- | --- | --- |
| Consumables, provisions | `UI/Screens3/Lobby/Inventory/Equipment/EquipmentItemImage.style` | `.equipment_item.<key> * .equipment-img` |
| | `UI/Styles/Lobby/Inventory/InventoryBigStyles.style` | `.equipment_item.<key> #icon` |
| | `UI/Styles/Lobby/Inventory/InventoryNormalStyles.style` | `.equipment_item.<key> #icon`, then `#icon.small` |
| Equipment | `UI/Screens3/Lobby/Inventory/OptionalDevices/OptionalDevicesItemImage.style` | `.optional_device_item.<key> #Img` |
| Shells | `UI/Screens3/Lobby/Inventory/Ammo/AmmoInfoOverlay.yaml` | `slot.iconType == "<key>"` cases |
| Crew skills | `UI/Screens3/Lobby/Inventory/Crew/SkillImage.style` | `.crew_skills.<key> * .skill-img` |

The first resource and selector that define a key win. Selectors with extra classes, such as event variants or ability states (`.ready`, `.cooldown`), are ignored, so abilities that only have state icons get no icon. Crew skills get their lobby icon; the per-state battle icons listed in `avatar.xml` are not exported. The report's `missingItemIcons` lists `kind/key` pairs left without an icon, either because no selector defines the key or because its sprite does not exist.

### Icons

Icons are grouped into sets, one folder each, under `assets/icons/<set>/<id>.png`. Each set contains every icon of its client directories and every sprite its items reference from elsewhere:

| Set | Client directories under `Data` |
| --- | --- |
| `shells` | `Gfx/Shared/tank-supply/ammunition/big` |
| `consumables` | `Gfx/UI/InventoryIcons/Big/Consumables` |
| `provisions` | `Gfx/UI/InventoryIcons/Big/Provisions` |
| `equipment` | `Gfx/UI/InventoryIcons/Big/OptionalDevices`, `Gfx/Lobby/tank-supply/optional-devices` |
| `crew-skills` | `Gfx/UI/CrewSkillIcons` |

Every sprite descriptor (`.txt`) and image in these directories becomes one icon; atlases referenced by descriptors are not icons themselves. Packed WebP images usually embed their own descriptor in an `extr` chunk; it places the image on its canvas like a separate descriptor. The icon ID is the client file name without the `@2x` suffix and extension, and the `@2x` variant is used when present. A UI sprite reference may omit the extension or name a `.psd` source; it resolves to the icon with the same ID in the referenced directory. Frames are placed on the descriptor's full canvas, so icons keep the client's alignment. Output is lossless PNG and needs no external tools. Two icons with the same ID in one set, a missing source directory, or an invalid descriptor or image fail the export. Images in different formats with the same ID are duplicates, including when their resolution suffixes differ. Descriptor canvases are limited to 16,777,216 pixels (64 MiB of RGBA data); oversized canvases and overflowing frame coordinates are rejected.

`assets/icons/catalog.json` lists every icon:

```json
{
  "formatVersion": 1,
  "count": 123,
  "sets": [
    {
      "id": "shells",
      "sources": ["Gfx/Shared/tank-supply/ammunition/big"],
      "count": 17,
      "icons": [
        {
          "id": "ap_l",
          "file": "shells/ap_l.png",
          "width": 360,
          "height": 360,
          "scale": 2,
          "source": {
            "kind": "sprite",
            "path": "Gfx/Shared/tank-supply/ammunition/big/ap_l@2x.txt",
            "texture": "Gfx/Shared/tank-supply/ammunition/big/texture0@2x.packed.webp",
            "frame": { "x": 973, "y": 308, "width": 303, "height": 304, "offsetX": 36, "offsetY": 31 }
          }
        }
      ]
    }
  ]
}
```

`file` is relative to the catalog. Source paths are relative to `Data` and omit `.dvpl`. A packed image with an embedded descriptor is a `sprite` whose `texture` is its own `path`; an image without one has `"kind": "image"` and no `texture` or `frame`. Unchanged PNGs are not rewritten. Icons listed by the previous catalog that are gone from the client are deleted; other files in the folder are kept. Catalog paths are normalized before comparison and deletion. UI sprite and atlas references must stay inside `Data`; both slash styles are accepted. Symlinks below the resource or output root are rejected when traversed by icon discovery, atlas reads, or output operations. Atlas exclusion uses full resource paths across all discovered directories, so unrelated images with the same basename remain icons.

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

The `export` package exposes `export.ExportClientFiles`,
`ClientFilesExportOptions` (from `export.NewClientFilesExportOptions`),
`ClientFilesExportReport`, and `ClientFileType`.
`export.ExportClientFilesWithProgress` also accepts a synchronous callback for
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

Following expects append-only writes within a session. It detects replacement by file identity (device/inode on Unix, creation time elsewhere), observed truncation, and changes to the last 64 read bytes. A rewrite that restores those same bytes can escape detection. Consumer callbacks run synchronously, providing backpressure without accumulating the whole live history. The library exposes `replay.StreamDecoder` (`Feed`, `Finish`), `TelemetryDecoder.OnPacket`, and `Follower.Poll`; callers control scheduling and cancellation. Payload size is limited to 16 MiB. Actual latency also depends on how frequently the game flushes its recording buffer.

`client locate` also returns `replay_dir` in JSON and prints the replay directory
in text output. This is the fixed OS path, independent of `--game-dir`, and is
returned even when the folder does not exist yet:

- macOS: `~/Library/Containers/com.tanksblitz/Data/Documents/tb-project/replays`
- Windows: `%USERPROFILE%\Documents\TanksBlitz\replays`, as documented by
  [Tanks Blitz support](https://lesta.ru/support/ru/products/tb/article/16159/).

`blitzclient.DefaultReplayDir()` exposes the same
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
Standalone streams use the same decoder. The public `replay.BattleCounterDecoder`
API can also consume complete packets directly; `Counters()` returns its typed
state and `Reset()` starts a new session. Support for these two versions does
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

A real replay copy is included at `testdata/sample.tbreplay`. The Python
standard-library script extracts its `data.replay` and writes it in chunks,
flushing after every write:

```sh
python3 scripts/simulate-replay.py
```

Each run creates `/tmp` (or the platform temp directory) with a random
`blitz-live-replay-*` subdirectory containing `data.replay`. The script prints the
full destination and a `go run ./cmd/blitz-client replay follow ...` command, waits three
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
go run ./cmd/blitz-client replay follow /tmp/live-test/data.replay
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

## Game presence

An overlay should appear only while the player is in the client. The `game`
package answers two questions: is a client process running, and is the player
in it — the client window is the foreground window and is visible on screen.

```go
state, err := game.Detect(game.Options{})
if err == nil && state.InGame {
	// show the overlay
}

// Or receive a state whenever it changes.
game.Watch(ctx, 250*time.Millisecond, game.Options{}, func(state game.State, err error) {
	overlay.SetVisible(err == nil && state.InGame)
})
```

`State` reports `Running`, the client `PID` and `Executable`, `Focused`,
`InGame`, and the client `Window`: bounds, `Visible`, `Minimized`, `Maximized`,
`Fullscreen`, and, where the platform reports it, `CursorHidden` and
`CursorCaptured`. Processes are matched by executable name, ignoring case and a
trailing `.exe`; `DefaultProcessNames` covers the macOS app (`Tanks Blitz`) and
the Windows builds (`tanksblitz.exe`, `wotblitz.exe`). Pass `ProcessNames` to
replace the list.

The overlay itself must not hide the game. Windows of the current process, and
of any `IgnorePIDs`, are transparent to focus detection: when the overlay is
active, the topmost window of another process decides whether the client is
focused. Prefer a non-activating overlay window anyway, so the client keeps
keyboard input.

| Platform | Process | Focus | Window state | Cursor |
| --- | --- | --- | --- | --- |
| Windows | Toolhelp snapshot | foreground window | minimized, maximized, fullscreen, cloaked on another virtual desktop | hidden, clipped to a region |
| macOS | `sysctl` process list | frontmost application | on screen, fullscreen (also reported as maximized) | hidden |
| Linux | `/proc`, including Wine/Proton | `_NET_ACTIVE_WINDOW` | EWMH hidden, maximized, fullscreen | — |

macOS needs no Screen Recording or Accessibility permission: window titles are
never read. On macOS, `Minimized` also covers a hidden application and a window
on another Space. Linux reads windows through libX11 loaded at run time, so it
works on X11 and for XWayland windows such as Wine and Proton games; without a
display or libX11, `FocusKnown` is false and `Focused` and `InGame` stay false.
The package uses no cgo.

`blitz-client game status` prints one snapshot; `--json` prints `State`.
`--watch` prints an NDJSON line on start and on every change until interrupted,
polling every `--poll-ms` milliseconds (500 by default). `--process-name` can
be repeated.

```sh
blitz-client game status
blitz-client game status --watch --poll-ms 250
```

## Mod commands

`mod list` lists installed mods for the selected game, sorted by id:

```sh
blitz-client mod list --game-dir /path/to/game --json
```

JSON contains `game_dir`, `resource_dir` and `mods`. Each mod has `id`, `name`
and `files` (unique logical resource paths in sorted order). A game without saved
installation state returns an empty `mods` array. Listing reads saved state;
it does not require mod source files or verify resource integrity. Invalid or
unsupported installer state returns an error instead of an empty list.

`mod install` also reinstalls an existing id, whether or not its source changed.
It replaces the saved operations and recomposes every resource touched by the
old or new version from saved originals, retaining other mods. Removed edits
are undone, new edits are added, and repeated installation does not accumulate
changes. All affected resources are prepared before writing; conflicts leave
the old installation intact. Write errors trigger rollback. External changes
to managed game files are rejected. The installation JSON reports `installed`
and `reinstalled`; `reinstalled` replaces the former `already_installed` field.

For Go managers, `mods.ListInstalledMods(gameDir)` returns the same installed
metadata. `mods.DiscoverMods(directory)` returns sorted manifest paths recursively,
without following directory symlinks or parsing manifests. Empty folders return
an empty list; each discovered path can be passed to `mods.LoadManifest` to retrieve
metadata and operations or report a broken mod independently. Both APIs are
available from the `mods` package, without the CLI.

`mod validate <directory>` recursively finds `mod.yaml` and `mod.yml`
in the directory and its subdirectories (without following directory symlinks). Each mod is
validated independently against the game and already installed mods; this does
not check whether all discovered mods can be installed together. No files are
changed. A failed mod does not stop validation of the others.

```sh
blitz-client mod validate ./mods --game-dir /path/to/game
```

The text output uses one status line per mod, unique file counts for successful
mods, and paths/codes/reasons only for failures. Successful mods appear first;
a separate `Failures` block collects all failed mods immediately before the
final passed/failed counts. Each group retains manifest-path order. Duplicate diagnostics for the same file are shown once.
Terminal output uses green success markers, red failures and dim counts. Colors
are disabled when output is redirected or `NO_COLOR` is nonempty;
`CLICOLOR_FORCE=1` enables them explicitly. JSON never contains color escapes.

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
(or `replace`) for the inserted text. Use `clear: {}` to empty a resource. YAML rejects unknown fields and
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

Declare unified diffs at the top level. One patch can modify multiple resources:

```yaml
id: HideRatingValue
name: Hide rating value
patch: mod.patch
```

`patch` accepts a file, a directory, or an ordered list of either:

```yaml
patch: patches
```

```yaml
patch:
  - mod.patch
  - patches/extra.diff
  - patches/optional
```

Paths are relative to the mod directory. Directories are scanned recursively for
`.patch` and `.diff` files, sorted by path; directory symlinks are not followed.
Lists run in the specified order. Each source file is loaded only once, even if
selected through both a file and a directory. Empty directories are rejected.
Patches run before operations under `files`, preserving section order within each
patch. Target paths inside diffs are relative to the game's resource directory.
Only changes to existing UTF-8 resources are supported; additions, deletions,
renames, mode changes and binary patches are rejected. Per-resource `patch`
actions under `files` are not supported.

Installed operations retain their patch content, so removal and recomposition do
not require the original patch files.

`update-from-git` writes all generated resource changes to one `mod.patch` and
sets the top-level `patch` reference. It refreshes this file on subsequent runs,
preserving manual patch sources and operations under `files`.

Only YAML mod manifests are supported (`mod.yaml` or `mod.yml`). Resource paths
are emitted without quotes where YAML permits it; quoting is added only when
needed to preserve a string value.
`update-from-git` reserializes YAML, retaining operation values but not comments
or formatting.

`update-from-git` compares the selected base revision (default `HEAD`) with the working tree, including staged and unstaged changes. Repository paths must mirror the game's unpacked resource paths. It handles modifications to existing tracked text resources; additions, deletions, renames, and binary changes are rejected. `.dvpl` files must be unpacked before generating a diff.

The game must be closed during installation and uninstallation. Active mods are recomposed from the current game resources and saved installation state, so mod removal order does not affect the remaining patches. The installer does not use a fixed game-file hash to identify a clean client.

## Game directory defaults

The ordered installation candidates are implemented in `client/location.go`.

- Windows: `C:\Games\Tanks_Blitz` first (resources in its `Data` subdirectory), followed by standard Steam, WindowsApps, and user game folders.
- macOS: `/Applications/Tanks Blitz.app/Contents/Resources/Data` first, then the Steam installation under the user's Library.
- Linux: common Steam, Flatpak, and Snap installation folders.

`client locate` returns `game_dir` (installation), `resource_dir` (installed
resources), `cache_dir` (downloaded resources in `packs`), and `replay_dir`.
The cache path is independent of `--game-dir` and does not need to exist yet.
It is also available through `client.DefaultCacheDir()`:

- macOS: `~/Library/Containers/com.tanksblitz/Data/Library/Caches/packs`
- Windows (Tanks Blitz): `%USERPROFILE%\Documents\TanksBlitz\packs`, confirmed
  in the inspected wblitz and BlitzHata mod sources. This path has not been
  verified on a Windows installation of this project.

This Windows default targets Tanks Blitz (Lesta). On other operating systems,
or when the required
home-directory environment variable is missing, `DefaultCacheDir()` returns
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

## Go API

```go
package main

import (
	"log"

	"github.com/thecoooler/blitz-client/export"
	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/mods"
)

func main() {
	gameDir := "" // empty selects the platform defaults
	tanks, err := client.ListClientTanks(gameDir)
	if err != nil {
		log.Fatal(err)
	}

	tankOptions := export.NewTankExportOptions("/path/to/viewer")
	tankOptions.GameDir = gameDir
	tankOptions.Limit = 10
	catalog, err := export.ExportTankCatalog(tankOptions)
	if err != nil {
		log.Fatal(err)
	}

	clientOptions := export.NewClientExportOptions("/path/to/export")
	clientOptions.GameDir = gameDir
	clientOptions.DataOnly = true
	exported, err := export.ExportClient(clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	installed, err := mods.ListInstalledMods(gameDir)
	if err != nil {
		log.Fatal(err)
	}
	paths, err := mods.DiscoverMods("mods")
	if err != nil {
		log.Fatal(err)
	}
	for _, path := range paths {
		manifest, err := mods.LoadManifest(path)
		if err != nil {
			continue // report the broken mod independently
		}
		for _, item := range installed.Mods {
			_ = item.ID == manifest.ID
		}
	}

	report, err := mods.ValidateMod("mods/NoMapName", gameDir)
	if err == nil && report.Valid {
		_, err = mods.InstallMod("mods/NoMapName", gameDir)
	}
	if err != nil {
		log.Fatal(err)
	}
	if _, err := mods.RemoveMod("NoMapName", ""); err != nil {
		log.Fatal(err)
	}
	head := "HEAD"
	if _, err := mods.UpdateFromGitDiff("mods/NoMapName", "mods/NoMapName", &head); err != nil {
		log.Fatal(err)
	}
	_, _, _ = tanks, catalog, exported
}
```

Errors are `*blitzclient.Error` values with `Code`, `Message`, and an optional
`Path`; use `blitzclient.AsError(err)` to recover them. The replay API follows the
same shape:

```go
import "github.com/thecoooler/blitz-client/replay"

analysis, err := replay.AnalyzeReplay("battle.tbreplay", replay.Options{IncludeData: true})

follower := replay.NewFollower("data.replay")
offset, err := follower.Poll(func(event replay.FollowEvent) error {
	// event marshals to one NDJSON line
	return nil
})
```

## License

The adapted replay schemas retain their MIT attribution; see `replay/LICENSE-schemas`.
