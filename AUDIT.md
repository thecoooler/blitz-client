# Audit: open findings

Known defects found in review and deliberately deferred. Remove an entry when it
is fixed.

## `client export`

### Concurrent exports into one directory can delete each other's output

- Where: `export/client.go`, `ExportClient` (empty-directory check and the
  deferred `removeExportOutput`).
- Problem: the "new or empty output directory" check does not reserve the
  directory. Two runs can both pass it. When one of them fails, its cleanup
  removes everything in the directory, including media and `data.json` written
  by the other run.
- Scenario: two `client export --output-dir out` runs start at the same time.
  Run A finishes. Run B then fails (for example, brotli is missing) and empties
  `out`, so A's completed export is gone.
- Suggested fix: write into a unique temporary sibling directory and rename it
  to the output path when done, refusing if the target became nonempty. Cleanup
  then only touches the run's own temporary directory.

### Resource lookup is case-sensitive on case-insensitive filesystems

- Where: `export/tanks.go` (`filesResolver`) and `export/data.go`
  (`snapshotReader`). Both look paths up in the `collectFiles` map, which uses
  exact-case keys.
- Problem: on NTFS (Windows) and default APFS (macOS), the game finds files
  whatever their case, but map lookups do not. Before the resolver, model and
  texture paths went through `os.Stat` on the Data directory, so for models
  without a cache this is a regression.
- Scenario: the client ships `images/Hull.dx11.dds.dvpl` and the material
  references `images/hull.tex`. The texture is not found and a solid-color
  placeholder is exported. A directory whose case differs from the expected
  path can likewise hide `list.xml` or component documents.
- Suggested fix: normalize keys the same way when indexing, applying the cache
  overlay and looking up, following the platform's case rules. This changes
  `collectFiles`, which `client files unpack` also uses, so check that command
  too: two files differing only in case must not silently overwrite each other.
