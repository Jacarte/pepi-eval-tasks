#!/usr/bin/env bash
# Regenerates the record files into a temp dir and diffs them against the committed seed/data/.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
TASK=$(cd "$HERE/.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
(cd "$HERE/gen" && GOTOOLCHAIN=local go run . -out "$TMP")
if diff -r "$TMP/data" "$TASK/seed/data"; then
  echo "check-gen: seed/data/ is up to date ($(find "$TMP/data" -type f | wc -l | tr -d ' ') files)"
else
  echo "check-gen: seed/data/ differs from generator output; run: (cd tools/gen && go run . -out ../../seed)" >&2
  exit 1
fi
