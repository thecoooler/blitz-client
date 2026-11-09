# Client catalog vocabulary

- **Client resources**: installed files under the client's `Data` directory.
  Resources may be plain files or DVPL containers.
- **Showroom model**: a paired SC2 scene and SCG geometry resource for a tank.
- **Battle mode**: a hangar battle type identified by the client's mode string,
  such as `regular` or `bossmode`. Its name, description, and short description
  are localization keys; availability, level limits, and event schedules are
  server data and are not inferred. The client names the mode artwork by the
  lower-case mode string in `Gfx/Lobby/backgrounds/GameMode`: `icon_<mode>` is
  the logo, `bg_<mode>_card` the card banner, and `bg_<mode>_layer1..3` the
  parallax banner layers.
- **Model discovery**: finding tanks with both showroom model files. The tank
  identifier comes from the vehicle XML node name. Localization, `userString`,
  and level are optional; without a localized name, the identifier supplies the
  display name using the existing underscore-to-space convention. Price, shop,
  and configuration fields are not prerequisites for model discovery.
- **Full client catalog**: map metadata and complete tank metadata, together
  with the tanks that have showroom model files. Required tank metadata remains
  strict: an incomplete record fails full catalog discovery rather than being
  silently omitted.

The catalog module owns these distinct validation rules. Combined catalog
discovery shares vehicle-list and localization reads between metadata and
showroom-model results.

# Item and icon vocabulary

- **Item**: a regular consumable, provision, or equipment definition with an
  `<id>`. Event-specific item files are not read.
- **Shell type**: an entry of the `<icons>` table in the nations' shells
  lists, such as `ap_cr_premium`. A shell's icon depends only on its type, so
  types are exported instead of individual shells.
- **Crew skill**: an entry of `<skills>` in `XML/item_defs/tankmen/avatar.xml`,
  with the tank class that lists it in `<skillsByClasses>`.
- **Icon key**: an item's `<icon>`, or a shell type's or crew skill's ID. UI
  resources map keys to sprites; the item never names its sprite. Selectors
  with state or event classes are ignored.
- **Sprite descriptor**: a DAVA `.txt` resource that places frames from texture
  atlases onto a canvas of fixed size, or the same text embedded in a packed
  WebP's `extr` chunk, where it refers to the image itself. `resources/sprite`
  parses both and rejects canvases over 16,777,216 pixels and overflowing frame
  coordinates.
- **Icon set**: a named group of client directories exported to one folder,
  plus the sprites its items reference from other directories. Every
  descriptor and image in the directories is an icon; atlases referenced by
  descriptors are not. The icon ID is the file name without `@2x` and
  extension, and must be unique within its set. Different image formats do not
  count as resolution variants. Atlas exclusion compares full resource paths
  across discovered directories. References must stay inside `Data`; icon path
  operations reject symlinks below the chosen resource and output roots.
- **Icon catalog**: `assets/icons/catalog.json`, listing sets and icons with
  size, resolution scale, and source resource. Adding a set means adding a row
  to `iconSets` in the export module; items link to icons through
  `items.json`.

# Geometry vocabulary

- **SCG polygon group**: indexed vertex geometry referenced by an SC2 scene.
  The archive module owns group identity, packed attribute layout, counts,
  triangle-list validation, and 16/32-bit index decoding. Duplicate or invalid
  group identities fail archive loading; geometry validation is applied only
  when a group is used. Used groups with unknown layout bits or overflowing
  counts fail with a diagnostic instead of truncating or guessing the layout.
- **Showroom geometry**: rendered tank geometry. Its module generates missing
  UVs and repairs missing or nonfinite normals and PBR directions. Position and
  UV values must be finite.
- **Collision geometry**: geometry whose per-vertex armor-zone labels join to
  vehicle XML thickness values. Its module requires normals and integer zone
  labels, repairs nonfinite face normals, and skips degenerate faces that cannot
  be repaired. Position and zone values must be finite.

The SCG decoding interface preserves optional attributes and nonfinite floats;
showroom and collision policy remain inside their respective modules.
