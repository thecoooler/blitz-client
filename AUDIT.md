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
