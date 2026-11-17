#!/usr/bin/env bash
# Usage: check-milestone.sh <task-dir> <workdir> <milestone-id>
#
# Grades one milestone the way the grading harness must: <workdir> is the agent
# workspace with hidden/ already overlaid (and owned_dirs already deleted before
# the overlay). The script
#   1. reads the milestone command from <task-dir>/milestones.yaml;
#   2. derives the expected test names from <task-dir>/hidden/**/*_test.go
#      (functions TestM<N>..., N from the milestone id "m<N>...") without running them;
#   3. rejects the workspace if a non-hidden Go file carries a known exit-code trick
#      (TestMain next to the hidden tests, os.Args or testing flags touched by
#      init(), os.Args assignment, "-test." flags in strings);
#   4. runs the command with `go test -json -count=1` and requires rc 0, a pass
#      event for every expected test and no skip/fail event for them.
# A milestone with no TestM<N> tests (for example a "full suite" milestone) only
# needs rc 0 and at least one passing test.
# Exit codes: 0 milestone passed, 1 milestone failed or rejected, 2 bad usage.
set -euo pipefail

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

# ---- static scan of agent-written Go files ----
HIDDEN_DIRS=$(cd "$HIDDEN" && find . -name '*_test.go' -type f -exec dirname {} \; | sort -u)
while IFS= read -r f; do
  rel=${f#"$WORKDIR"/}
  [ -f "$HIDDEN/$rel" ] && continue          # overlaid hidden file
  d=$(dirname "./$rel"); d=${d#./}; [ -n "$d" ] || d=.
  d="./$d"; [ "$d" = ./. ] && d=.
  if [[ "$rel" == *_test.go ]] && grep -qE '^func TestMain\(' "$f" && grep -qxF "$d" <<<"$HIDDEN_DIRS"; then
    reject "$rel defines TestMain in a package directory graded by hidden tests"
  fi
  if grep -qE '^func init\(' "$f" && grep -qE 'os\.Args|flag\.|testing\.' "$f"; then
    reject "$rel has init() touching os.Args or flags"
  fi
  if grep -qE 'os\.Args[[:space:]]*=[^=]' "$f"; then
    reject "$rel assigns os.Args"
  fi
  if grep -qE '"-?test\.' "$f"; then
    reject "$rel mentions test flags (-test.*)"
  fi
done < <(find "$WORKDIR" -name '*.go' -type f -not -path '*/.git/*' | sort)

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
