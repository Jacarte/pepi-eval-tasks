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
#
# The workspace must hold only regular files and directories: a symlink (or fifo, socket,
# device) is rejected, because it would slip past every check that selects regular files
# while go still compiles it.
#
# The agent's code runs as an unprivileged user (setpriv, uid/gid 65534, no groups) on a copy
# of the workspace, with its own HOME, GOCACHE, GOPATH and TMPDIR. The checker's files
# (events, expected tests, helper binaries) live in a root-owned 0700 directory it cannot
# reach, and the helper binaries are hashed when built and compared before they are used.
# This needs root and setpriv (util-linux). Without them the script fails closed with exit 2.
# CHECK_MILESTONE_ALLOW_ROOT=1 turns the privilege drop off: ONLY for local self-tests on a
# machine without setpriv (macOS); the agent's code then runs with the script's privileges and
# can tamper with the checker.
# GRADE_GOCACHE=<dir> (optional) is a Go build cache shared between runs, handed to the
# unprivileged user; the grading container pre-warms it.
# Exit codes: 0 milestone passed, 1 milestone failed or rejected, 2 bad usage or harness error.
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

# ---- privilege separation (fail closed) ----
DROP=()
if [ "${CHECK_MILESTONE_ALLOW_ROOT:-0}" = 1 ]; then
  echo "check-milestone: WARNING: CHECK_MILESTONE_ALLOW_ROOT=1, agent code runs with this script's privileges (local self-tests only)" >&2
else
  if ! command -v setpriv >/dev/null 2>&1; then
    echo "check-milestone: setpriv (util-linux) not found: refusing to run agent code with the grader's privileges (CHECK_MILESTONE_ALLOW_ROOT=1 is for local self-tests only)" >&2
    exit 2
  fi
  if [ "$(id -u)" -ne 0 ]; then
    echo "check-milestone: must run as root to drop privileges for the agent's code (CHECK_MILESTONE_ALLOW_ROOT=1 is for local self-tests only)" >&2
    exit 2
  fi
  DROP=(setpriv --reuid=65534 --regid=65534 --clear-groups)
  if ! "${DROP[@]}" true 2>/dev/null; then
    echo "check-milestone: cannot run commands as uid 65534 with setpriv" >&2
    exit 2
  fi
fi

# ---- workspace: regular files and directories only ----
SPECIAL=$(workspace_special_files "$WORKDIR")
[ -z "$SPECIAL" ] || reject "workspace contains symlinks or special files: $(printf '%s' "$SPECIAL" | tr '\n' ' ')"

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
done < <(find "$HIDDEN" -name '*_test.go' ! -type d | sort)
sort -u "$TMP/expected" -o "$TMP/expected"

# ---- restore_seed_tests: the seed's visible tests must be intact, nothing else visible may test ----
RESTORE=$(task_restore_seed_tests "$TASK/task.yaml") || { echo "check-milestone: restore_seed_tests must be true or false" >&2; exit 2; }
if [ "$RESTORE" = true ]; then
  while IFS= read -r rel; do
    [ -f "$HIDDEN/$rel" ] && continue
    if [ ! -f "$TASK/seed/$rel" ] || ! cmp -s "$TASK/seed/$rel" "$WORKDIR/$rel"; then
      reject "$rel is not the seed's test file (restore_seed_tests: true; run prepare-workspace.sh first)"
    fi
  done < <(cd "$WORKDIR" && find . -name '*_test.go' ! -type d -not -path '*/.git/*' | sed 's|^\./||' | sort)
  while IFS= read -r rel; do
    [ -f "$HIDDEN/$rel" ] && continue
    cmp -s "$TASK/seed/$rel" "$WORKDIR/$rel" 2>/dev/null || reject "seed test $rel is missing or modified (restore_seed_tests: true; run prepare-workspace.sh first)"
  done < <(cd "$TASK/seed" && find . -name '*_test.go' ! -type d | sed 's|^\./||' | sort)
fi

# ---- static scan of agent-written Go files ----
# Agent-written = not overlaid from hidden/ and not byte-identical to the seed file.
(cd "$HERE/scancheat" && GOTOOLCHAIN=local go build -o "$TMP/scancheat" main.go)
# Directories that hold seed tests or hidden tests: a TestMain there would replace the graded run.
{
  (cd "$HIDDEN" && find . -name '*_test.go' ! -type d)
  if [ -d "$TASK/seed" ]; then (cd "$TASK/seed" && find . -name '*_test.go' ! -type d); fi
} | sed 's|^\./||' | while IFS= read -r rel; do dirname "$rel"; done | sort -u > "$TMP/testdirs"
: > "$TMP/agentfiles"
while IFS= read -r rel; do
  [ -f "$HIDDEN/$rel" ] && continue
  if [ -f "$TASK/seed/$rel" ] && cmp -s "$TASK/seed/$rel" "$WORKDIR/$rel"; then continue; fi
  echo "$rel" >> "$TMP/agentfiles"
done < <(cd "$WORKDIR" && find . -name '*.go' ! -type d -not -path '*/.git/*' | sed 's|^\./||' | sort)
"$TMP/scancheat" -root "$WORKDIR" < "$TMP/agentfiles" > "$TMP/findings"
while IFS=$'\t' read -r rel kind detail; do
  case "$kind" in
    testmain)
      grep -qxF "$(dirname "$rel")" "$TMP/testdirs" && reject "$rel defines TestMain in a package directory with seed or hidden tests" ;;
    init-args) reject "$rel has init() touching os.Args or flags ($detail)" ;;
    test-flag-literal) reject "$rel mentions go test flags ($detail)" ;;
  esac
done < "$TMP/findings"

# ---- helper binaries: built and hashed before any agent code runs ----
sha256_of() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi; }
(cd "$HERE/checkjson" && GOTOOLCHAIN=local go build -o "$TMP/checkjson" main.go)
CHECKJSON_SUM=$(sha256_of "$TMP/checkjson")

# ---- run the agent's code on a copy, unprivileged ----
# AGENT is traversable by the unprivileged user and holds everything it may write.
AGENT=$(mktemp -d)
trap 'rm -rf "$TMP" "$AGENT"' EXIT
chmod 755 "$AGENT"
mkdir "$AGENT/work" "$AGENT/home" "$AGENT/gopath" "$AGENT/tmp" "$AGENT/gotmp"
cp -R "$WORKDIR"/. "$AGENT/work"/
RUN_ENV=(env GOTOOLCHAIN=local)
if [ "${#DROP[@]}" -gt 0 ]; then
  GOCACHE_DIR=${GRADE_GOCACHE:-$AGENT/gocache}
  mkdir -p "$GOCACHE_DIR"
  chown -R 65534:65534 "$AGENT/work" "$AGENT/home" "$AGENT/gopath" "$AGENT/tmp" "$AGENT/gotmp"
  # A shared cache is chowned once, not on every run.
  if [ "$(stat -c %u "$GOCACHE_DIR" 2>/dev/null || true)" != 65534 ]; then chown -R 65534:65534 "$GOCACHE_DIR"; fi
  RUN_ENV+=(HOME="$AGENT/home" GOPATH="$AGENT/gopath" GOCACHE="$GOCACHE_DIR" GOTMPDIR="$AGENT/gotmp" TMPDIR="$AGENT/tmp")
fi
RUNCMD="go test -json -count=1${CMD#go test}"
set +e
(cd "$AGENT/work" && ${DROP[@]+"${DROP[@]}"} "${RUN_ENV[@]}" bash -c "$RUNCMD") > "$TMP/events" 2> "$TMP/stderr"
RC=$?
set -e
# Nothing the agent started may outlive the run or touch the results below.
if [ "${#DROP[@]}" -gt 0 ]; then "${DROP[@]}" kill -KILL -1 2>/dev/null || true; fi
[ "$(sha256_of "$TMP/checkjson")" = "$CHECKJSON_SUM" ] || reject "$MID: the result checker was modified while the tests ran"
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
