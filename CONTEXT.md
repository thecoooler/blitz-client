# Client catalog vocabulary

- **Client resources**: installed files under the client's `Data` directory.
  Resources may be plain files or DVPL containers.
- **Showroom model**: a paired SC2 scene and SCG geometry resource for a tank.
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
