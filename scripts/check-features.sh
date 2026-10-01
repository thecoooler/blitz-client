#!/usr/bin/env bash
# Pass extra Cargo flags, e.g. bash scripts/check-features.sh --offline.
set -euo pipefail
cd "$(dirname "$0")/.."
cargo test --all-features "$@"
cargo test --no-default-features "$@"
for feature in resources client export mods replay cli; do
    cargo test --no-default-features --features "$feature" "$@"
done
