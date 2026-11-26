#!/usr/bin/env bash
# Usage: verify-task.sh <task-dir> <reference-dir>
# Checks task metadata, then proves every milestone fails on the seed and passes on the reference,
# each judged by check-milestone.sh (go test -json, expected tests by name).
set -euo pipefail

if [ $# -ne 2 ]; then
  echo "usage: $0 <task-dir> <reference-dir>" >&2
  exit 2
fi
HERE=$(cd "$(dirname "$0")" && pwd)
TASK=$(cd "$1" && pwd)
REF=$(cd "$2" && pwd)

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

ERRORS=0
fail() { echo "FAIL: $*"; ERRORS=$((ERRORS + 1)); }

# A skill milestone (skill_loaded:<name> or skill_not_loaded:<name>) is graded only in evals, from
# the skills the parent loaded: it is not run in the workspace, so this script cannot prove it.
SKILL_COMMAND='^skill_(not_)?loaded:[a-z0-9][a-z0-9._-]*$'

# ---- YAML parsing: emits "T<TAB>key<TAB>value" and "M<TAB>id<TAB>title<TAB>command" ----
parse_python() {
  python3 - "$1" "$2" <<'PY'
import sys, yaml
t = yaml.safe_load(open(sys.argv[1])) or {}
m = yaml.safe_load(open(sys.argv[2])) or {}
if not isinstance(t, dict) or not isinstance(m, dict):
    sys.exit("yaml root must be a mapping")
for k, v in t.items():
    if isinstance(v, list):
        v = ",".join(str(x) for x in v)
    elif isinstance(v, bool):
        v = "true" if v else "false"
    print("T\t%s\t%s" % (k, "null" if v is None else v))
ms = m.get("milestones")
if not isinstance(ms, list):
    sys.exit("milestones must be a list")
for e in ms:
    e = e if isinstance(e, dict) else {}
    print("M\t%s\t%s\t%s" % (e.get("id", ""), e.get("title", ""), e.get("command", "")))
PY
}

# Strict line-based parser for exactly the README schema: flat "key: value" in task.yaml
# (lists only in flow style, e.g. owned_dirs: [acceptance]),
# and a "milestones:" list of "- id:" items with title/command in milestones.yaml.
# Limitations: no multi-line values; " #" starts a comment even inside quotes.
parse_lines() {
  awk -v OFS='\t' '
    function clean(s) {
      sub(/[[:space:]]+#.*$/, "", s); sub(/^[[:space:]]+/, "", s); sub(/[[:space:]]+$/, "", s)
      if (s ~ /^".*"$/ || s ~ /^\x27.*\x27$/) s = substr(s, 2, length(s) - 2)
      return s
    }
    function flush() { if (have) print "M", id, title, cmd; have = 0; id = title = cmd = "" }
    function kv(line,   i, k) {
      i = index(line, ":"); if (i == 0) return 0
      k = substr(line, 1, i - 1); gsub(/[[:space:]]/, "", k)
      KEY = k; VAL = clean(substr(line, i + 1)); return 1
    }
    FNR == 1 { inm = 0 }
    /^[[:space:]]*(#.*)?$/ { next }
    FILENAME == ARGV[1] { if (kv($0)) print "T", KEY, VAL; next }
    /^milestones:/ { inm = 1; next }
    inm {
      line = $0
      if (line ~ /^[[:space:]]*-[[:space:]]+/) { flush(); have = 1; sub(/^[[:space:]]*-[[:space:]]+/, "", line) }
      if (kv(line)) {
        if (KEY == "id") id = VAL; else if (KEY == "title") title = VAL; else if (KEY == "command") cmd = VAL
      }
    }
    END { flush() }
  ' "$1" "$2"
}

for f in task.yaml milestones.yaml; do
  [ -f "$TASK/$f" ] || { echo "FAIL: missing $TASK/$f"; exit 1; }
done

if python3 -c 'import yaml' 2>/dev/null; then
  METHOD="PyYAML"
  PARSED=$(parse_python "$TASK/task.yaml" "$TASK/milestones.yaml") || { echo "FAIL: YAML parse error"; exit 1; }
else
  METHOD="line-based fallback (PyYAML not installed)"
  PARSED=$(parse_lines "$TASK/task.yaml" "$TASK/milestones.yaml")
fi
echo "YAML parser: $METHOD"

tget() { printf '%s\n' "$PARSED" | awk -F'\t' -v k="$1" '$1=="T" && $2==k {print $3; exit}'; }

# ---- task.yaml fields ----
NAME=$(tget name); KIND=$(tget kind); LANG_=$(tget language); GOV=$(tget go_version)
SESS=$(tget sessions); STO=$(tget session_timeout_minutes); BUD=$(tget budget_usd); DL=$(tget deadline_minutes)
SCORING=$(tget scoring); OWNED_RAW=$(tget owned_dirs); RESTORE=$(tget restore_seed_tests)
[ -n "$NAME" ] || fail "task.yaml: name missing"
case "$KIND" in long|needs-subagents|parallel|memory) ;; *) fail "task.yaml: kind must be long|needs-subagents|parallel|memory (got '$KIND')" ;; esac
[ "$LANG_" = go ] || fail "task.yaml: language must be go (got '$LANG_')"
[ -n "$GOV" ] || fail "task.yaml: go_version missing"
[[ "$SESS" =~ ^[0-9]+$ ]] && [ "$SESS" -ge 1 ] || fail "task.yaml: sessions must be an integer >= 1 (got '$SESS')"
[[ "$STO" =~ ^[0-9]+$ ]] && [ "$STO" -ge 1 ] || fail "task.yaml: session_timeout_minutes must be a positive integer (got '$STO')"
[[ "$BUD" =~ ^[0-9]+(\.[0-9]+)?$ ]] || fail "task.yaml: budget_usd must be a number (got '$BUD')"
case "$SCORING" in prefix|count) ;; *) fail "task.yaml: scoring must be prefix|count (got '$SCORING')" ;; esac
case "$RESTORE" in ""|true|false) ;; *) fail "task.yaml: restore_seed_tests must be true|false (got '$RESTORE')" ;; esac
# owned_dirs: optional flow list ("[a, b]" from the line parser, "a,b" from PyYAML)
OWNED=()
OWNED_LIST=${OWNED_RAW#\[}; OWNED_LIST=${OWNED_LIST%\]}
OWNED_LIST=${OWNED_LIST//[[:space:]\"\']/}
if [ -n "$OWNED_LIST" ] && [ "$OWNED_LIST" != null ]; then
  IFS=',' read -r -a OWNED <<<"$OWNED_LIST"
fi
for d in ${OWNED[@]+"${OWNED[@]}"}; do
  if [ -z "$d" ] || [[ "$d" == /* ]] || [[ "$d" == . ]] || [[ "/$d/" == *"/../"* ]]; then
    fail "task.yaml: owned_dirs entry '$d' must be a relative path inside the workspace"
  elif [ ! -d "$TASK/hidden/$d" ]; then
    fail "task.yaml: owned_dirs entry '$d' is not a directory under hidden/"
  fi
done
if [ "$KIND" = parallel ]; then
  [[ "$DL" =~ ^[0-9]+$ ]] || fail "task.yaml: deadline_minutes required for kind=parallel"
elif ! [ -z "$DL" ] && ! [ "$DL" = null ] && ! [[ "$DL" =~ ^[0-9]+$ ]]; then
  fail "task.yaml: deadline_minutes must be a number or null (got '$DL')"
fi

# ---- milestones ----
IDS=(); TITLES=(); CMDS=()
while IFS=$'\t' read -r _ id title cmd; do
  IDS+=("$id"); TITLES+=("$title"); CMDS+=("$cmd")
done < <(printf '%s\n' "$PARSED" | awk -F'\t' '$1=="M"')
[ "${#IDS[@]}" -ge 1 ] || fail "milestones.yaml: no milestones"
for i in "${!IDS[@]}"; do
  [ -n "${IDS[$i]}" ] || fail "milestone #$((i + 1)): id missing"
  [ -n "${TITLES[$i]}" ] || fail "milestone #$((i + 1)): title missing"
  [ -n "${CMDS[$i]}" ] || fail "milestone #$((i + 1)): command missing"
  [[ "${IDS[$i]}" =~ ^m[0-9]+($|[^0-9]) ]] || fail "milestone #$((i + 1)): id '${IDS[$i]}' must start with m<number> (it selects the TestM<number> tests)"
  if [[ "${CMDS[$i]}" == skill_* ]]; then
    [[ "${CMDS[$i]}" =~ $SKILL_COMMAND ]] || fail "milestone #$((i + 1)): skill command must be skill_loaded:<name> or skill_not_loaded:<name> (got '${CMDS[$i]}')"
  else
    [[ "${CMDS[$i]}" == "go test"* ]] || fail "milestone #$((i + 1)): command must start with 'go test', skill_loaded: or skill_not_loaded:"
  fi
done
DUPES=$(printf '%s\n' "${IDS[@]}" | sort | uniq -d || true)
[ -z "$DUPES" ] || fail "milestones.yaml: duplicate ids: $DUPES"

# ---- layout ----
FORBIDDEN='pepi-eval-tasks|Jacarte|AGENTS|subagent|delegat'
if [[ "$SESS" =~ ^[0-9]+$ ]]; then
  [ -d "$TASK/prompts" ] || fail "prompts/ missing"
  EXPECT_PROMPTS=$(for n in $(seq 1 "$SESS"); do printf 'prompts/%02d.md\n' "$n"; done)
  FOUND_PROMPTS=$(cd "$TASK" && find prompts -type f 2>/dev/null | sort || true)
  [ "$FOUND_PROMPTS" = "$EXPECT_PROMPTS" ] || fail "prompts/ must hold exactly 01.md..$(printf '%02d' "$SESS").md for sessions: $SESS (found: $(echo "$FOUND_PROMPTS" | tr '\n' ' '))"
fi
SEED="$TASK/seed"
[ -d "$SEED" ] || fail "seed/ missing"
if [ -d "$SEED" ]; then
  [ -f "$SEED/go.mod" ] || fail "seed/go.mod missing"
  HITS=$(grep -rliE "$FORBIDDEN" "$SEED" "$TASK/prompts" 2>/dev/null || true)
  [ -z "$HITS" ] || fail "seed/prompts mention a forbidden word (pepi-eval-tasks|Jacarte|AGENTS|subagent|delegat) in: $HITS"
  [ -z "$(find "$SEED" -name hidden -type d)" ] || fail "hidden/ must not be inside seed/"
fi
[ -d "$TASK/hidden" ] || fail "hidden/ missing"
if [ -f "$TASK/memory/fixture.json" ]; then
  (cd "$HERE/checkfixture" && GOTOOLCHAIN=local go build -o "$WORK/checkfixture" main.go)
  while IFS= read -r p; do fail "$p"; done < <("$WORK/checkfixture" "$TASK/memory/fixture.json" "$SEED" "$TASK/prompts" 2>&1)
elif [ -e "$TASK/memory" ] || [ "$KIND" = memory ]; then
  fail "memory/fixture.json missing (required by kind=memory and by a memory/ dir)"
fi

if [ "$ERRORS" -gt 0 ]; then
  echo "Static checks failed ($ERRORS); not running milestones."
  exit 1
fi

HAVE_GO=$(go env GOVERSION | sed 's/^go//')
[ "$HAVE_GO" = "$GOV" ] || echo "WARN: task go_version $GOV, local go $HAVE_GO"

# ---- run milestones ----
export GOTOOLCHAIN=local
overlay() { cp -R "$1"/. "$2"/; }

# prepare <dir>: owned_dirs deleted, seed tests restored if restore_seed_tests, hidden/ overlaid
# (what the grading harness does)
overlay_hidden() { "$HERE/prepare-workspace.sh" "$TASK" "$1"; }

mkdir "$WORK/seed" "$WORK/ref"
cp -R "$SEED"/. "$WORK/seed"/
overlay_hidden "$WORK/seed"
cp -R "$SEED"/. "$WORK/ref"/
overlay "$REF" "$WORK/ref"
overlay_hidden "$WORK/ref"

SEED_RES=(); REF_RES=()
for i in "${!IDS[@]}"; do
  if [[ "${CMDS[$i]}" == skill_* ]]; then
    SEED_RES+=("skill-eval"); REF_RES+=("skill-eval")
    continue
  fi
  if OUT=$("$HERE/check-milestone.sh" "$TASK" "$WORK/seed" "${IDS[$i]}" 2>&1); then
    SEED_RES+=("PASSES(bad)"); fail "${IDS[$i]}: passes on the seed"
  else
    SEED_RES+=("fails-ok")
  fi

  if OUT=$("$HERE/check-milestone.sh" "$TASK" "$WORK/ref" "${IDS[$i]}" 2>&1); then
    REF_RES+=("passes-ok")
  else
    REF_RES+=("FAILS(bad)"); fail "${IDS[$i]}: fails on the reference"; printf '%s\n' "$OUT" | tail -n 20
  fi
done

echo
printf '%-24s %-14s %-14s\n' MILESTONE SEED REFERENCE
for i in "${!IDS[@]}"; do
  printf '%-24s %-14s %-14s\n' "${IDS[$i]}" "${SEED_RES[$i]}" "${REF_RES[$i]}"
done
echo
echo "skill-eval: graded only in evals (not run here)"
if [ "$ERRORS" -gt 0 ]; then echo "RESULT: FAILED ($ERRORS problem(s))"; exit 1; fi
echo "RESULT: OK ($NAME)"
