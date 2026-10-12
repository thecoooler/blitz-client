#!/usr/bin/env bash
# Run the same command with the Rust reference binary and the Go binary and
# compare exit codes, stdout, and stderr byte for byte.
# Usage: scripts/compare.sh <rust-binary> <go-binary> <args...>
set -u
rust=$1 go=$2
shift 2
tmp=$(mktemp -d)
"$rust" "$@" >"$tmp/rust.out" 2>"$tmp/rust.err"; rust_code=$?
"$go" "$@" >"$tmp/go.out" 2>"$tmp/go.err"; go_code=$?
status=0
if [ "$rust_code" != "$go_code" ]; then
  echo "EXIT DIFF ($rust_code vs $go_code): $*"; status=1
fi
for stream in out err; do
  if ! cmp -s "$tmp/rust.$stream" "$tmp/go.$stream"; then
    echo "STD$stream DIFF: $*"
    diff "$tmp/rust.$stream" "$tmp/go.$stream" | head -${DIFF_LINES:-20}
    status=1
  fi
done
[ $status = 0 ] && echo "same ($rust_code): $*"
rm -rf "$tmp"
exit $status
