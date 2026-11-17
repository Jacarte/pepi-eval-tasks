#!/usr/bin/env bash
# Usage: prepare-workspace.sh <task-dir> <workdir>
#
# Turns an agent workspace into a grading workspace, in the order the grading rules require:
#   1. delete every owned_dirs entry of task.yaml;
#   2. if task.yaml has restore_seed_tests: true, delete every *_test.go in the workspace
#      (edited, deleted-and-recreated and agent-added alike) and copy the seed's *_test.go
#      files back, so visible tests cannot be removed or weakened to pass a regression milestone;
#   3. copy hidden/ over the workspace.
# Exit codes: 0 prepared, 2 bad usage or bad task.yaml.
set -euo pipefail

if [ $# -ne 2 ]; then
  echo "usage: $0 <task-dir> <workdir>" >&2
  exit 2
fi
HERE=$(cd "$(dirname "$0")" && pwd)
TASK=$(cd "$1" && pwd)
WORKDIR=$(cd "$2" && pwd)
# shellcheck source=taskyaml.sh
. "$HERE/taskyaml.sh"
[ -f "$TASK/task.yaml" ] && [ -d "$TASK/hidden" ] || { echo "prepare-workspace: $TASK is not a task dir" >&2; exit 2; }

RESTORE=$(task_restore_seed_tests "$TASK/task.yaml") || { echo "prepare-workspace: restore_seed_tests must be true or false" >&2; exit 2; }

while IFS= read -r d; do
  [ -n "$d" ] || continue
  if [[ "$d" == /* ]] || [ "$d" = . ] || [[ "/$d/" == *"/../"* ]]; then
    echo "prepare-workspace: owned_dirs entry '$d' must be a relative path inside the workspace" >&2
    exit 2
  fi
  rm -rf "${WORKDIR:?}/$d"
done < <(task_owned_dirs "$TASK/task.yaml")

if [ "$RESTORE" = true ]; then
  find "$WORKDIR" -name '*_test.go' -type f -not -path '*/.git/*' -delete
  while IFS= read -r rel; do
    mkdir -p "$WORKDIR/$(dirname "$rel")"
    cp "$TASK/seed/$rel" "$WORKDIR/$rel"
  done < <(cd "$TASK/seed" && find . -name '*_test.go' -type f | sed 's|^\./||' | sort)
fi

cp -R "$TASK/hidden"/. "$WORKDIR"/
