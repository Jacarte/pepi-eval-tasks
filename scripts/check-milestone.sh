#!/usr/bin/env bash
# Usage: check-milestone.sh <task-dir> <workdir> <milestone-id>
#
# Grades one milestone the way the grading harness must: <workdir> is the agent
# workspace with hidden/ already overlaid (and owned_dirs already deleted before
# the overlay). The script
#   1. reads the milestone command from <task-dir>/milestones.yaml;
#   2. derives the expected test names from <task-dir>/hidden/**/*_test.go
#      (functions TestM<N>..., N from the milestone id "m<N>...") without running them;
#   3. if task.yaml has restore_seed_tests: true, rejects the workspace unless its
#      *_test.go files are exactly the seed's plus hidden/ (see prepare-workspace.sh);
#      then rejects it if a Go file the agent wrote (not from hidden/, not identical to
#      the seed) carries a known exit-code trick, found with scancheat (go/parser): TestMain
#      in a directory with seed or hidden tests; os.Args assigned or flag.Set/flag.Parse
#      called in an init() body or package var initializer; string literals that are
#      go test flags ("-test.run=...");
#   4. runs the command with `go test -json -count=1` and requires rc 0, a pass
#      event for every expected test and no skip/fail event for them.
# A milestone with no TestM<N> tests (for example a "full suite" milestone) only
# needs rc 0 and at least one passing test.
# Exit codes: 0 milestone passed, 1 milestone failed or rejected, 2 bad usage.
set -euo pipefail
# shellcheck source=taskyaml.sh
. "$(dirname "$0")/taskyaml.sh"

if [ $# -ne 3 ]; then
  echo "usage: $0 <task-dir> <workdir> <milestone-id>" >&2
  exit 2
fi
HERE=$(cd "$(dirname "$0")" && pwd)
TASK=$(cd "$1" && pwd)
WORKDIR=$(cd "$2" && pwd)
MID=$3
HIDDEN="$TASK/hidden"
[ -f "$TASK/milestones.yaml" ] && [ -d "$HIDDEN" ] || { echo "check-milestone: $TASK is not a task dir" >&2; exit 2; }

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
reject() { echo "REJECT: $*"; exit 1; }

# ---- milestone command (line-based, same schema as verify-task.sh) ----
CMD=$(awk -v want="$MID" '
  function clean(s) {
    sub(/[[:space:]]+#.*$/, "", s); sub(/^[[:space:]]+/, "", s); sub(/[[:space:]]+$/, "", s)
    if (s ~ /^".*"$/ || s ~ /^\x27.*\x27$/) s = substr(s, 2, length(s) - 2)
    return s
  }
  /^[[:space:]]*-[[:space:]]+id:/ { v = $0; sub(/^[^:]*:/, "", v); cur = clean(v); next }
  /^[[:space:]]*(-[[:space:]]+)?command:/ && cur == want { v = $0; sub(/^[^:]*:/, "", v); print clean(v); exit }
' "$TASK/milestones.yaml")
[ -n "$CMD" ] || { echo "check-milestone: milestone '$MID' or its command not found" >&2; exit 2; }
[[ "$CMD" == "go test"* ]] || reject "command must start with 'go test' (got '$CMD')"
[[ "$MID" =~ ^m([0-9]+)($|[^0-9]) ]] || { echo "check-milestone: milestone id '$MID' must start with m<number>" >&2; exit 2; }
NUM=${BASH_REMATCH[1]}

# ---- expected tests, from the hidden sources only ----
: > "$TMP/expected"
while IFS= read -r f; do
  rel=${f#"$HIDDEN"/}
  dir=$(dirname "$rel")
  awk -v dir="$dir" -v n="$NUM" '
    match($0, /^func TestM[0-9]+[A-Za-z0-9_]*\(/) {
      name = substr($0, 6, RLENGTH - 6)      # TestM<digits>...
      digits = name; sub(/^TestM/, "", digits); sub(/[^0-9].*$/, "", digits)
      if (digits == n) print dir "\t" name
    }' "$f" >> "$TMP/expected"
done < <(find "$HIDDEN" -name '*_test.go' -type f | sort)
sort -u "$TMP/expected" -o "$TMP/expected"

# ---- restore_seed_tests: the seed's visible tests must be intact, nothing else visible may test ----
RESTORE=$(task_restore_seed_tests "$TASK/task.yaml") || { echo "check-milestone: restore_seed_tests must be true or false" >&2; exit 2; }
if [ "$RESTORE" = true ]; then
  while IFS= read -r rel; do
    [ -f "$HIDDEN/$rel" ] && continue
    if [ ! -f "$TASK/seed/$rel" ] || ! cmp -s "$TASK/seed/$rel" "$WORKDIR/$rel"; then
      reject "$rel is not the seed's test file (restore_seed_tests: true; run prepare-workspace.sh first)"
    fi
  done < <(cd "$WORKDIR" && find . -name '*_test.go' -type f -not -path '*/.git/*' | sed 's|^\./||' | sort)
  while IFS= read -r rel; do
    [ -f "$HIDDEN/$rel" ] && continue
    cmp -s "$TASK/seed/$rel" "$WORKDIR/$rel" 2>/dev/null || reject "seed test $rel is missing or modified (restore_seed_tests: true; run prepare-workspace.sh first)"
  done < <(cd "$TASK/seed" && find . -name '*_test.go' -type f | sed 's|^\./||' | sort)
fi

# ---- static scan of agent-written Go files ----
# Agent-written = not overlaid from hidden/ and not byte-identical to the seed file.
(cd "$HERE/scancheat" && GOTOOLCHAIN=local go build -o "$TMP/scancheat" main.go)
# Directories that hold seed tests or hidden tests: a TestMain there would replace the graded run.
{
  (cd "$HIDDEN" && find . -name '*_test.go' -type f)
  if [ -d "$TASK/seed" ]; then (cd "$TASK/seed" && find . -name '*_test.go' -type f); fi
} | sed 's|^\./||' | while IFS= read -r rel; do dirname "$rel"; done | sort -u > "$TMP/testdirs"
: > "$TMP/agentfiles"
while IFS= read -r rel; do
  [ -f "$HIDDEN/$rel" ] && continue
  if [ -f "$TASK/seed/$rel" ] && cmp -s "$TASK/seed/$rel" "$WORKDIR/$rel"; then continue; fi
  echo "$rel" >> "$TMP/agentfiles"
done < <(cd "$WORKDIR" && find . -name '*.go' -type f -not -path '*/.git/*' | sed 's|^\./||' | sort)
"$TMP/scancheat" -root "$WORKDIR" < "$TMP/agentfiles" > "$TMP/findings"
while IFS=$'\t' read -r rel kind detail; do
  case "$kind" in
    testmain)
      grep -qxF "$(dirname "$rel")" "$TMP/testdirs" && reject "$rel defines TestMain in a package directory with seed or hidden tests" ;;
    init-args) reject "$rel has init() touching os.Args or flags ($detail)" ;;
    test-flag-literal) reject "$rel mentions go test flags ($detail)" ;;
  esac
done < "$TMP/findings"

# ---- run ----
(cd "$HERE/checkjson" && GOTOOLCHAIN=local go build -o "$TMP/checkjson" main.go)
RUNCMD="go test -json -count=1${CMD#go test}"
set +e
(cd "$WORKDIR" && GOTOOLCHAIN=local bash -c "$RUNCMD") > "$TMP/events" 2> "$TMP/stderr"
RC=$?
set -e
if [ "$RC" -ne 0 ]; then
  grep -hE '"Action":"output"' "$TMP/events" | sed -E 's/.*"Output":"(.*)"\}$/\1/' | tail -n 20 || true
  tail -n 20 "$TMP/stderr"
  reject "$MID: command exited $RC"
fi
if [ -s "$TMP/expected" ]; then
  "$TMP/checkjson" -expected "$TMP/expected" < "$TMP/events" || reject "$MID: expected tests not all passing"
else
  "$TMP/checkjson" < "$TMP/events" || reject "$MID: no tests passed"
fi
echo "PASS: $MID ($(wc -l < "$TMP/expected" | tr -d ' ') expected tests)"
