# Client export, format v1

There is one entity export command and one interchange format:

```sh
# All sections (default): data + tank icons + 3D/PBR models
go run ./cmd/blitz-client client export --game-dir /path/to/Data --output-dir /tmp/blitz-all

# Fast offline data export: no image reads, no model conversion
go run ./cmd/blitz-client client export --game-dir /path/to/Data --output-dir /tmp/blitz-data --include data --no-cache

# One option selects one or several sections
go run ./cmd/blitz-client client export --output-dir /tmp/blitz-icons --include data,icons
go run ./cmd/blitz-client client export --output-dir /tmp/blitz-models --include models
```

`--include` accepts `all` (default) or any comma-separated subset of `data`,
`icons`, `models`; the option may be repeated. Order is immaterial; duplicates collapse. Unknown/empty values
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
  images/tanks/<nation>/<techName>/big.webp
  images/tanks/<nation>/<techName>/small.webp
  assets/tanks/<nation>/<techName>/model.json[.br]
  assets/tanks/<nation>/<techName>/model.bin[.br]
  assets/tanks/<nation>/<techName>/model.pbr.bin[.br]
  assets/tanks/<nation>/<techName>/armor.json[.br]
  assets/tanks/<nation>/<techName>/armor-mesh.json[.br]
  assets/tanks/<nation>/<techName>/<texture files>
```

No separate manifest, maps/tanks/parameters/configuration/component files, model
catalog or duplicated raw documents are written. A tank record contains its own
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
  included: ("data" | "icons" | "models")[],
  nations: string[],
  tanks: Tank[],
  localization?: { ru: { [localizationKey: string]: string } },
  maps?: Object,
  components?: { [nation: string]: { [kind: string]: XMLValue } },
  equipment?: { devices: XMLValue, slots: XMLValue },
  provisions?: { [mode: string]: XMLValue },
  settings?: { [name: string]: XMLValue | YAMLValue },
  issues: { tank: string, section: "icons" | "models", message: string }[]
}
```

Require `format == "blitz-client"` and `version == 1`. Unknown versions must be
rejected; additional fields may be introduced without changing the version.
`included` is always canonical: data, icons, models, filtered to the selection.
`nations` comes from available_nations.yaml, in source order. Tanks are in nation
order, then list.xml order. Map/object key order never determines component order.
All media paths are relative to this directory and use `/` (a `./` prefix is valid).
Media references contain no source-machine paths or timestamps. Diagnostic messages
may mention source paths when conversion fails.

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

## Media, resources and consumers

Icon sources come from parameters.resourcesPath.bigIconPath/smallIconPath.
`~res:/` addresses the merged resource tree. Candidate order: exact path,
`.packed.webp`, `@2x.packed.webp`, `.webp`, `.png`. DVPL is decoded; images are not
transcoded/resized and retain their real extension. Icons never select models,
collision meshes, textures or photos. Map atlas cropping/preview images are outside
v1. Missing icons are null plus an issue when icons were requested.

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

`--game-dir` accepts an install or Data directory. Plain and DVPL resources share
the existing reader. Within a root, plain files win over DVPL siblings; packs
wins over Data by unpacked relative path. Default packs is used when it exists;
`--cache-dir` overrides it, `--no-cache` disables it. These two options conflict.
Symlinks in source trees are skipped. Model conversion sees the same overlay as
other sections. Resources are never modified.

Model encoding options are `--max-texture-size original|N|WIDTHxHEIGHT` (1024x1024),
`--texture-format webp|png` (webp), `--webp-quality 0..100` (90), `--no-compress`, and `--no-pbr`.
The pre-existing model codecs remain required for corresponding assets: ffmpeg
for texture decoding, cwebp for WebP, brotli for compressed model payloads. Data
and icon exports require none of them. There are no new external dependencies.
