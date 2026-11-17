#!/usr/bin/env bash
# Regenerates the seed into a temp dir and diffs it against the committed seed/.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
TASK=$(cd "$HERE/.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
(cd "$HERE/gen" && GOTOOLCHAIN=local go run . -out "$TMP/seed")
if diff -r "$TMP/seed" "$TASK/seed"; then
  echo "check-gen: seed/ is up to date ($(find "$TMP/seed" -type f | wc -l | tr -d ' ') files)"
else
  echo "check-gen: seed/ differs from generator output; run: (cd tools/gen && go run . -out ../../seed)" >&2
  exit 1
fi
