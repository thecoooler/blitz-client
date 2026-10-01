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
