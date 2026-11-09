# Client export, format v1

There is one entity export command and one interchange format:

```sh
# All sections (default): data + tank icons + 3D/PBR models + maps + mechanics
go run ./cmd/blitz-client client export --game-dir /path/to/Data --output-dir /tmp/blitz-all

# Fast offline data export: no image reads, no model conversion
go run ./cmd/blitz-client client export --game-dir /path/to/Data --output-dir /tmp/blitz-data --include data --no-cache

# One option selects one or several sections
go run ./cmd/blitz-client client export --output-dir /tmp/blitz-icons --include data,icons
go run ./cmd/blitz-client client export --output-dir /tmp/blitz-models --include models
go run ./cmd/blitz-client client export --output-dir /tmp/blitz-maps --include maps
go run ./cmd/blitz-client client export --output-dir /tmp/blitz-mechanics --include mechanics
```

`--include` accepts `all` (default) or any comma-separated subset of `data`,
`icons`, `models`, `maps`, `mechanics`; the option may be repeated. Order is immaterial; duplicates collapse. Unknown/empty values
and combining `all` with individual sections are errors. Every selection writes
**the same `data.json` schema**, with a canonical `included` array. Tank identities
are always present so media-only exports can be consumed without another catalog.
Absent optional sections are not loaded from game resources. Empty/omitted optional
containers inside an included section mean there are no records.

This replaces the previous CLI formats: `client data export`, `client tanks export`
and `client export --data-only` no longer exist. `client files unpack` remains a
separate utility for unpacking raw resources; those files are not an entity format.
There is no HTTP client, API dependency, image fetching or site-specific import.

## One file for entities

The output directory contains `data.json` plus referenced media, if requested:

```text
export/
  data.json
  items.json
  maps.json
  mechanics.json
  images/tanks/<nation>/<techName>/big.webp
  images/tanks/<nation>/<techName>/small.webp
  assets/icons/catalog.json
  assets/icons/<set>/<icon>.png
  assets/tanks/<nation>/<techName>/model.json[.br]
  assets/tanks/<nation>/<techName>/model.bin[.br]
  assets/tanks/<nation>/<techName>/model.pbr.bin[.br]
  assets/tanks/<nation>/<techName>/armor.json[.br]
  assets/tanks/<nation>/<techName>/armor-mesh.json[.br]
  assets/tanks/<nation>/<techName>/<texture files>
  assets/maps/<space>/preview.png
  assets/maps/<space>/minimap.png
  assets/mechanics/<folder>/image.png
  assets/mechanics/<folder>/icon.png
```

No separate manifest, maps/tanks/parameters/configuration/component files, model
catalog or duplicated raw documents are written. `items.json` is the normalized
index of shells, consumables, provisions, equipment and crew skills that links
each item to its exported icon; `assets/icons/catalog.json` describes the icon
sets and their source resources. `maps.json` is the normalized index of
maps.yaml entries that links each map to its exported images. `mechanics.json`
defines the tank mechanics referenced by vehicle parameter files, their
localization and their main artwork and icon. A tank record contains its own
metadata, parameters, configuration and media references. Shared definitions live
once under `components`, not as resolved copies inside every turret. Equipment,
provisions and localization also have one authoritative home. Model files contain
geometry/material payloads, not a second vehicle catalog.

`data.json` is written **last**, marking successful extraction. A failed run
removes the media it wrote, so the same directory can be reused for a retry.
Existing nonempty output directories and directories overlapping sources are
rejected. Missing media or per-model conversion failures are nonfatal and recorded
in `issues`; malformed/missing required data resources fail the command, as do
failures that affect every model (missing ffmpeg/cwebp/brotli, unwritable output)
or a run in which no installed model converts. Required resources depend on
`included`: models-only needs lists/identities and installed model resources, not
equipment, parameters or localization; icons without data record a missing
parameter file as an issue. Without available_nations.yaml the built-in nation
list is used, and nations without list.xml are skipped.

## Root schema

```text
Snapshot = {
  format: "blitz-client",
  version: 1,
  included: ("data" | "icons" | "models" | "maps" | "mechanics")[],
  game: Game,
  exportedAt: string,     // UTC, RFC 3339, e.g. "2026-10-29T02:03:16Z"
  exporterVersion: string | null, // blitz-client module version, see below
  nations: string[],
  tanks: Tank[],
  localization?: { ru: { [localizationKey: string]: string | null } },
  maps?: { maps: { [name: string]: Map } }, // the raw maps.yaml document, keys sorted
  modes?: GameMode[],
  components?: { [nation: string]: { [kind: string]: XMLValue } },
  equipment?: { devices: XMLValue, slots: XMLValue },
  provisions?: { [mode: string]: XMLValue },
  settings?: { [name: string]: XMLValue | YAMLValue },
  hangarParameters?: YAMLValue,
  crewSkills?: CrewSkill[],
  crewSkillTraining?: CrewSkillTraining,
  issues: { tank: string, section: "icons" | "models" | "maps" | "mechanics", message: string }[]
}
```

`Map` mirrors one maps.yaml entry: `id` (integer), `localName` (scene path),
`avaliableInTrainingRoom` (the client's own spelling), `availableModes`,
`spriteFrame`, and when present `tags`, `vehicleCamouflageKind`,
`shadowMapsAvailable`, `supremacyPointsThreshold`, `levels` and
`assaultRespawnPoints` (`allies`/`enemies` arrays of
`{ respawnNumber, points: [[x, y], …] }`).

`GameMode` is one battle mode named by the client's hangar UI:

```text
GameMode = {
  id: string,                     // client mode string, e.g. "regular", "bossmode"
  name: string,                   // localization key, e.g. "battleType/regular"
  description: string | null,     // localization key or null
  shortDescription: string | null // localization key or null
}
```

The client offers its battle modes through a build-time enum, not one resource
document. `modes` therefore lists the modes the exporter knows for this client
release in display order and resolves their text through `localization.ru`.
IDs match the client's `battleType/<id>` localization keys. A few client
builds spell a description key differently from the mode ID; those keys are
listed explicitly. Server-side availability, allowed levels, squad limits and
event schedules are not client data and are not exported. `modes` is present
only when `data` is selected.

Require `format == "blitz-client"` and `version == 1`. Unknown versions must be
rejected; additional fields may be introduced without changing the version.
`included` is always canonical: data, icons, models, maps, mechanics, filtered to the selection.
`nations` comes from available_nations.yaml, in source order. Tanks are in nation
order, then list.xml order. Map/object key order never determines component order.
All media paths are relative to this directory and use `/` (a `./` prefix is valid).
Media references contain no source-machine paths or timestamps. Diagnostic messages
may mention source paths when conversion fails.

## Game build and exporter

`game` identifies the client the snapshot was read from. It is written for
every `included` selection and comes from version.txt only:

```text
Game = {
  versionText: string | null, // e.g. "26.10.70_533068 master MacOS"
  version: string | null,     // "26.10.70"
  build: string | null,       // "533068"; a string, not a number
  branch: string | null,      // "master"
  platform: string | null     // "MacOS"; the rest of the line, may contain spaces
}
```

`versionText` is the first line of version.txt without surrounding whitespace.
Its first token is split at the last `_` into `version` and `build` when the
suffix is all digits; otherwise the whole token is `version` and `build` is
null. Without version.txt every field is null. Like every resource, version.txt
from the packs overlay wins over the installed one. Two exports of the same
`version` and `build` can still differ when downloaded packs change; compare
`exportedAt` as well.

`exporterVersion` is the blitz-client module version of the program that
wrote the export: a release tag such as `v1.2.0`, or a pseudo-version such as
`v0.0.0-20261029201357-ed5695baa714` (commit time and hash) built from a
checkout, with `+dirty` for uncommitted changes. It is null for `go run`,
`go test`, builds outside a Git checkout and dependencies replaced by a local
directory. A different exporter version can change `data.json` for the same
game build. The typed readers decode `game`, `exportedAt` and
`exporterVersion`; the schema tool does not infer XML scalar types for `game`.

## Tank schema and identity

```text
Tank = {
  id: string,             // e.g. "ussr:0"
  nation: string,         // e.g. "ussr"
  clientId: integer,      // decimal vehicle-list ID, zero is valid
  techName: string,       // e.g. "T-34"
  metadata?: XMLValue,
  parameters?: YAMLValue,
  configuration?: XMLValue,
  icons?: { big: string | null, small: string | null },
  model?: {
    bundle: string,
    model: string,
    armor: string | null,
    armorMesh: string | null,
    armorNote: string | null
  }
}
```

The canonical identity is `nation + ":" + decimal(clientId)`. It is not a global
API vehicle ID or computed compact descriptor. Leading zeros are normalized
(`02` becomes `2`). `(nation, techName)` locates resources; technical names alone
are not global identities. Metadata does not repeat the numeric ID. Model
attachments do not repeat vehicle IDs, labels or levels. Binary bundle names use
technical names only for predictable paths; join models to entities through the
containing tank record.

When `data` is selected, `metadata` contains list.xml fields (localization keys,
prices, tags, combatRole, configurationModes, etc.); `parameters` contains the entire
vehicle parameter YAML (resourcesPath, mechanics, etc.); `configuration` contains
the entire vehicle XML (hull/chassis, speed, invisibility, turrets, slot counts,
optDevicePreset, etc.). These are nested properties of **one tank**, not references
to separate documents. Localization keys remain keys; consumers choose display
language from `localization`, avoiding localized copies of every field.

When `icons` is selected, the `icons` object is present; unavailable icons are
null. When `models` is selected, `model` is present only after successful conversion;
missing/unconvertible models are described by `issues`. PBR and armor, when source
assets support them, are part of the `models` section; `--no-pbr` omits PBR maps.

## XML and YAML values

XML is represented with field names directly, rather than a recursive
name/text/children wrapper for every scalar:

```xml
<Gun>shared<reloadTime>4</reloadTime><empty/><price>2<gold/></price></Gun>
```

```json
{
  "#text": "shared",
  "reloadTime": "4",
  "empty": null,
  "price": {"#text": "2", "gold": null}
}
```

Leaf XML text is a string; an empty element is null. Missing fields stay absent.
Numbers, booleans and whitespace-separated game lists retain their XML scalar
spelling; consumers interpret the game's field types. XML text is trimmed and
entity-decoded. An element containing text plus child properties uses `#text`.
Attributes use `@name`, e.g. `{"@path":"common.xml"}`. Repeated same-name children
become an array, even when each value is null. Comments and formatting whitespace
are discarded. This handles scalar/list/#text/null forms without guessed defaults.

**Selection order is explicit:** `guns`, `shots`, `chassis`, `engines`, `fuelTanks`, `radios`, `turrets` and `turretsN` containers
are arrays of `{name: string, properties: XMLValue}` in source order. Never sort
them. A single element still yields a one-element array. Other maps are lookups.
Rare ordered containers with their own text/attributes use an object with `#items`
for that array and normal `#text`/`@attribute` fields, retaining all information.

For example:

```json
{
  "turrets0": [{
    "name": "Turret_1",
    "properties": {
      "guns": [{
        "name": "GunB",
        "properties": {
          "#text": "shared",
          "reloadTime": "4",
          "shots": [{"name": "ShellB", "properties": {"shell": "shared"}}]
        }
      }]
    }
  }]
}
```

YAML values use ordinary JSON object/array/string/number/boolean/null shapes.
Non-string YAML mapping keys become JSON string keys in their YAML scalar
spelling: integers and floats (`1` becomes `"1"`, `0.5` becomes `"0.5"`), booleans
(`"true"`), null (`"null"`) and timestamps (RFC 3339). Keys that become ambiguous
after this conversion (for example `1` and `"1"`) and collection keys fail
extraction. Non-finite floats, which JSON cannot represent, become the strings
`".inf"`, `"-.inf"` and `".nan"`.
YAML arrays and nulls retain their shapes.

## Shared definitions, modifiers and settings

`components[nation][kind]` contains each installed national component document
once, with its root wrapper removed. `guns` and `shells` are required for `data`;
other installed component XML/YAML are included under their extensionless relative
names. For example:

```text
components.ussr.guns.shared["GunB"]
components.ussr.guns.ids["GunB"]
components.ussr.shells["ShellB"]
```

A turret gun named `GunB` with scalar/`#text` token `shared` refers to
`components[tank.nation].guns.shared.GunB`. A shot named `ShellB` whose `shell` is
`shared` refers to `components[tank.nation].shells.ShellB`. Local nested properties
are overrides and must remain attached to the specific turret/gun/shot. Shared gun
shot properties, penetration, speed, damage, clip/burst and shell defaults remain
in their original fields. Empty/local definitions are not inferred shared refs.
Explicit source reference/namespace tokens are preserved verbatim. The exporter
does not resolve inheritance, flatten overrides, compute final statistics or
validate every reference; it exports the independent source definitions and links.

`equipment.devices` contains optional_devices.xml, `equipment.slots` contains
optional_device_slots.xml, including presets and supporting fields. A vehicle's
`optDevicePreset` refers to `equipment.slots.presets[presetName]`. Nine-slot presets,
slot option order, empty slots and script effects are retained.

`provisions[mode]` contains each document referenced by provisions/list.xml exactly
once; mode is its relative path without the final extension (e.g. `common`,
`hellgames`). The include listing itself is not duplicated. Single and repeated
`items path` includes both work. Provision identity is `(mode, itemName)`, independent
of the national component namespace. Vehicle/nation/level/mode include/exclude
filters and script effects remain intact. Applicability is a consumer decision.

`settings` retains other installed common XML/YAML, keyed by extensionless path
relative to vehicles/common (e.g. `vehicle`, `roles`, `consumables/common`). Equipment
and provisions are excluded here to avoid duplicate entities. Source IDs tables
and nextAvailableId fields are retained where present. `maps` is the complete
maps.yaml/maps.yml object if installed (maps.yaml wins); no second map schema is
produced. Only ru localization is currently selected by `data`.

`hangarParameters` preserves the installed root `hangarParamsConfig.yaml`, including
its `keyedArchive` wrappers, per-parameter `digits`, `units` and unknown fields.
It is optional when the resource is absent and is read only with `data`, through
the same Data/packs overlay. It is UI configuration, not computed tank statistics.
Binary-embedded calculation constants are documented for the researched binary in
[hangar-ttx.md](hangar-ttx.md); they are not resource fields or server defaults.

## Crew skills and training

The `data` section includes `crewSkills` from the installed
`XML/item_defs/tankmen/avatar.xml`, using the same Data/packs overlay. Missing
avatar.xml or an empty skill list omits `crewSkills`. Invalid installed XML or a
skill assigned to multiple training classes fails the export. Icons-only and
models-only exports do not read crew definitions.

Skills retain XML order. Each record has `id` (the XML element name), `tankClass`
(the class in `skillsByClasses` used for training), `type`, `parameters`, and
`levels`. `userString`, `description`, `tip`, and `features` retain localization
keys from `userString`, `effectDescription`, `tipDescription`, and
`featuresDescription`; resolve these through `localization.ru`. Missing text
fields are empty strings. The training class does not restrict where a learned
skill takes effect. With the `icons` section, skill icons are exported into
`assets/icons/crew-skills` and linked from `items.json`.

`parameters` contains all non-presentation fields with their original XML names.
Unlike the generic XMLValue representation, this typed skill section converts
finite numbers and booleans to JSON numbers and booleans. Nested objects, text,
and repeated fields are preserved. Fixed effects and activation conditions stay
in `parameters`; the exporter does not infer final vehicle statistics.

To include training levels and XP costs, supply an explicit local server cache:

```sh
go run ./cmd/blitz-client client export --include data,icons \
  --output-dir /tmp/blitz-export \
  --server-config /path/to/server_config_REGION_ACCOUNT.dat
```

`--server-config` requires the `data` section. The file is a zlib-compressed Python
pickle decoded in Go without executing Python globals. Only `gradeUpSkillCost`
and `avatarSkillsEconomics` are exported; account progress is not read. On macOS,
these caches are under
`~/Library/Containers/com.tanksblitz/Data/Documents/tb-project/cache`.

`crewSkillTraining` contains `maxLevel` (the length of `gradeUpSkillCost`), the
original `gradeUpSkillCost` list and `avatarSkillsEconomics`, whose numeric mode
IDs become string object keys. Each skill's `levels` contains levels 1 through
`maxLevel`, individual (not cumulative) XP prices, and `parameters` computed only
for numeric `*PerLevel` fields as coefficient × level. These are raw bonuses or
chances, not final vehicle multipliers.

Without `--server-config`, `crewSkillTraining` is omitted and `levels` is null:
maximum levels and XP prices are not specified in avatar.xml. Account-specific
availability, server feature switches and missing training prerequisites are not
inferred. Invalid server caches fail the export and remove partial output.

## Media, resources and consumers

Icon sources come from parameters.resourcesPath.bigIconPath/smallIconPath.
`~res:/` addresses the merged resource tree. Candidate order: exact path,
`.packed.webp`, `@2x.packed.webp`, `.webp`, `.png`. DVPL is decoded; images are not
transcoded/resized and retain their real extension. Icons never select models,
collision meshes, textures or photos. Missing icons are null plus an issue when
icons were requested.

The same `icons` section renders item icons as PNG into `assets/icons/<set>`:
`shells`, `consumables`, `provisions`, `equipment` and `crew-skills`. Sprite
descriptors and standalone images in those client directories are rendered at
their full canvas; `@2x` resources win when both resolutions exist, and atlases
referenced by descriptors are not exported as icons. UI sprite mappings add
icons referenced from other directories. Item icon sets missing from the client
are skipped. `assets/icons/catalog.json` lists every icon with its size,
resolution scale and source resource; `items.json` uses those paths. Icons
without a found sprite are `null` and reported in the export command's
`missingItemIcons`.

Two more sets are exported unconditionally when the directories exist:

- `flags` contains every icon in `Gfx/Lobby/flags`, so all country variants
  (`carousel`, `event`, `filter`, `flying`, `notification`, `tank-node`,
  `tank-tree`, `text`, `tutor-tank`) are kept.
- `tank-classes` collects the `icon_tank-class_*` heavy, medium, light and
  AT-SPG variants from `Gfx/Lobby/icons` and
  `Gfx/Lobby/icons/tank-tiers`, plus the `heavy-tank_xl`, `light-tank_xl`,
  `middle-tank_xl` and `pt-tank_xl` icons from
  `Gfx/Lobby/icons/tank-tiers/common`. Prefix filters keep tier numbers and
  other lobby icons stored beside them out of the set, and unused SPG
  artifacts (`icon_tank-class_spg`) are excluded because Blitz does not
  include artillery.

A sprite with several frames exports its first frame under the icon ID and the
remaining frames as `<id>-frame1`, `<id>-frame2` … so color and state variants
are kept without changing the base name. Each frame is a separate catalog
entry.

## Maps

The `maps` section renders map images as PNG into `assets/maps/<space>/` and
writes `maps.json`, the normalized index of `maps.yaml` entries:

- `preview` is cropped from the map preview atlas
  `Gfx/UI/MapsListDialog/back_maps[@2x].packed.webp`; the maps.yaml
  `spriteFrame` is the atlas frame index. The atlas is decoded once and shared.
- `minimap` comes from `Gfx/UI/BattleScreenHUD/minimap/<space>/MiniMapSmall[@2x].packed.webp`,
  where `<space>` is the maps.yaml entry key. `@2x` wins when both resolutions
  exist; the SD variant is the fallback.
- `effects` lists the surface effect names from the scene's
  `3d/Maps/<scene dir>/blitz/map_effects.yaml`, when present.

A missing maps.yaml records one issue and writes an empty index. Missing or
broken per-map images leave `preview`/`minimap` null and add an issue, like
icons; everything else fails the export. `data.json` still carries the raw
`maps.yaml` document in the `data` section.

Every rendered image names its client resource explicitly: `previewSource` and
`minimapSource` use the same source object as the icon catalog (`kind: sprite`
with the atlas `path` and the cropped `frame` rectangle, both relative to Data
and without the `.dvpl` suffix), so consumers can re-derive or re-crop the
images from the client themselves.

## Tank mechanics

The `mechanics` section writes `mechanics.json`, the normalized index of the
`mechanics` arrays in `3d/Tanks/Parameters/<nation>/<techName>.yaml`. The raw
vehicle parameter documents remain in `data.json` when `data` is selected.

Each entry contains:

- `id`: the source parameter name, such as `DrumShot`, `PumpShot` or `LVArmor`.
- `nameKey`/`descriptionKey`: the `TankMechanics/...` localization keys used by the client.
- `name`/`description`: the `ru` values of those keys; empty when absent. Only `ru` is exported.
- `image`: main artwork at `assets/mechanics/<folder>/image.png`, or null.
- `icon`: large mechanic icon at `assets/mechanics/<folder>/icon.png`, or null.
- `imageSource`/`iconSource`: the client resource and sprite frame, like map images.

`<folder>` is the client artwork directory, such as `drum-reload`; it is not
derived from `id`. Consumers should use the `image` and `icon` paths instead of
constructing them from the mechanic name.

The tank-to-mechanic relation is not repeated here. It stays in the vehicle
parameter documents: a tank belongs to a mechanic when its
`tanks[].parameters.mechanics[].name` contains that `id`. A mechanics-only
export contains the definitions and artwork; include `data` as well when the
tank membership is needed.

Only the main artwork and one large icon are exported from
`Gfx/Lobby/icons/tank-mechanics/`; the `_s` and `_m` sizes, battle HUD overlays
and unrelated lobby artwork are deliberately omitted. `@2x` wins over the SD
variant. Missing artwork is null and is not an issue.

The client's parameter names are not identical to all of its enum and
localization IDs. The exporter applies the confirmed aliases `LVArmor` →
`ArmorMoving`, `AlteredSpotDuration` → `ReducedSpotDuration` and `PumpShot` →
`ArmorPumpShot` (title only; its description key stays
`PumpShotDescription`). New mechanics whose enum ID cannot be derived from the
parameter name fall back to a kebab-case artwork folder and the same-name
localization keys. Mechanics referenced by no exported tank are not listed.

Models retain existing geometry, PBR and armor payload schemas. A viewer enumerates
`tanks` and selects records with `model`, loads `tank.model.model`, and resolves its
geometry/textures from `tank.model.bundle`. It obtains identity, labels and game
properties from the same tank and localization tables. The previous standalone
`assets/tanks/catalog.json` is removed. No viewer implementation exists in this
repository; external consumers must switch to this documented root schema.

Go consumers may use:

```go
snapshot, err := export.ReadClientExport("/path/to/export")
// Check err, then use snapshot.Included and snapshot.Tanks.
// tank.Configuration and tank.Parameters belong to that tank;
// tank.Model links to binary assets if models were selected.
```

`ReadClientExport` keeps nested game data as untyped values and does not
validate it. Package `export` also provides typed readers that validate every
data section against Go structs:

| Function | Accepts | On schema mismatch |
| --- | --- | --- |
| `DecodeStrictDataExport` / `ReadStrictDataExport` | data-only exports | error |
| `DecodePartialClientExport` / `ReadPartialClientExport` | any `--include` selection | error |
| `DecodePartialClientExportLenient` / `ReadPartialClientExportLenient` | any `--include` selection | `[]SchemaWarning` |

The partial readers validate the data sections exactly like the strict reader
and keep media (`tanks[].icons`, `tanks[].model`) and `crewSkillTraining` as raw
JSON, listed in `UnvalidatedSections`. Use the strict readers in CI and when
checking the schema, and the lenient reader in importers that must keep
working after a game patch:

```go
snapshot, warnings, err := export.ReadPartialClientExportLenient("/path/to/export")
if err != nil {
	return err // malformed JSON, duplicate key, unsupported version, invalid root
}
for _, w := range warnings {
	// w.Kind: export.SchemaUnknownField or export.SchemaUnexpectedValue
	// w.Path: "$.tanks[12].configuration.hull.newField"
	// w.SchemaPath: "$.tanks[].configuration.hull.newField" (for grouping)
	log.Printf("export schema: %s", w)
}
for _, tank := range snapshot.Tanks {
	config, ok := tank.Configuration.Get()
	if !ok {
		continue // absent or null
	}
	guns := config.Turrets0 // Optional[[]Optional[...Turrets0Entry]], source order
	_ = guns
}
```

Lenient decoding skips unknown fields and drops a value whose shape or scalar
spelling does not match: the field becomes absent, a dictionary entry or array
element is removed, and a required field that is dropped drops its enclosing
value. `SchemaWarning.Dropped` names the removed value. Everything else is
decoded exactly as in strict mode. Strict errors are `*SchemaError` with the
JSON path of the offending value.

Guarantees of the typed readers:

- Every value in `tanks[].metadata/parameters/configuration`, `localization`,
  `maps`, `components`, `equipment`, `provisions`, all 23 `settings`
  documents, `hangarParameters`, `crewSkills`, `modes` and `issues` is decoded into a
  named type; the data sections hold no `any` or `json.RawMessage`.
  Dictionaries keyed by game identifiers are `map[string]Optional[T]`.
- `Optional[T]` distinguishes an absent field (`!Present()`), explicit `null`
  (`Null()`) and a value (`Get()`), and encodes the same state back. Optional
  fields use the `omitzero` tag (Go 1.24+); the module requires Go 1.27.
- JSON numbers are `StrictNumber`/`StrictInteger` (string-backed, original
  spelling such as `9007199254740993`); nothing passes through `float64`.
- XML element text is a JSON string typed by its spelling: `XMLInteger`,
  `XMLNumber`, `XMLBool`, `XMLLocalizationKey` (`#namespace:key`) or
  `XMLNumberVector` (whitespace-separated numbers). They are string types that
  keep and re-encode the original text, validate it on decode and offer
  `Int64`, `Float64`, `Bool`, `Namespace`/`Key` and `Numbers`/`Float64s`.
  Plain `string` remains only for fields whose values include free text,
  names, paths or mixed spellings.
- XML elements observed in several shapes are explicit unions:
  `XMLScalarObject[S, T]`, `XMLScalarList[S, T]`, `XMLObjectList[O, A]`,
  `XMLScalarObjectList[S, O, A]`, where `S` is the scalar type of the text
  variant. Exactly one variant is set.
- Arrays, including configuration selection lists, keep their order;
  selection entries must have a non-null `name` and a `properties` field.
- Unknown fields, unknown settings documents, duplicate keys, case-mismatched
  keys and trailing JSON are errors in strict mode.
- `json.Marshal(snapshot)` reproduces `data.json` as an equal JSON value.
  `json.Unmarshal` into the section types applies the strict schema.

Limits: the schema describes the shapes observed in the client versions it was
built from, not game semantics or cross-references. Scalar types come from the
values seen; a field seen with only integers is `XMLInteger` even if a later
version writes `1.5` (lenient mode then drops that value with a warning).
`crewSkillTraining` and media have no schema.

### Updating the schema after a game patch

Export the new client with `--include data`, then compare it with the schema,
ideally together with exports of earlier versions:

```sh
go run ./internal/tools/exportschema new/data.json old/data.json
```

The command prints lenient warnings grouped by schema path (new fields, new
shapes) and scalar drift (fields whose declared type differs from the most
specific type all values fit), and exits with status 1 if there are any.
`-write` rewrites drifted scalar types in `export/*.go`; new fields and shapes
are added by hand to the struct named in the warning path. `-text` lists the
fields that stay `string` with value examples; `-stats` counts schema fields by
scalar type. The tests validate real exports listed in `BLITZ_EXPORT_DATA`
(separated by `:`); with two or more exports they also require the declared
scalar types to equal the inferred ones:

```sh
BLITZ_EXPORT_DATA=new/data.json:old/data.json go test ./export
```

`--game-dir` accepts an install or Data directory. Plain and DVPL resources share
the existing reader. Within a root, plain files win over DVPL siblings; packs
wins over Data by unpacked relative path. Default packs is used when it exists;
`--cache-dir` overrides it, `--no-cache` disables it. These two options conflict.
Symlinks in source trees are skipped. Model conversion sees the same overlay as
other sections. Resources are never modified.

Model encoding options are `--max-texture-size original|N|WIDTHxHEIGHT` (1024x1024),
`--texture-format webp|png` (webp), `--webp-quality 0..100` (90), `--no-compress`, and `--no-pbr`.
The pre-existing model codecs remain required for corresponding assets: ffmpeg
for texture decoding, cwebp for WebP when ffmpeg lacks a built-in WebP encoder,
brotli for compressed model payloads. Data
and icon exports require none of them. There are no new external dependencies.
Textures shared by several vehicles are converted once per export and linked
into the other bundles.
