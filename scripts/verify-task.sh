#!/usr/bin/env bash
# Usage: verify-task.sh <task-dir> <reference-dir>
# Checks task metadata, then proves every milestone fails on the seed and passes on the reference.
set -euo pipefail

if [ $# -ne 2 ]; then
  echo "usage: $0 <task-dir> <reference-dir>" >&2
  exit 2
fi
TASK=$(cd "$1" && pwd)
REF=$(cd "$2" && pwd)

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

ERRORS=0
fail() { echo "FAIL: $*"; ERRORS=$((ERRORS + 1)); }

# ---- YAML parsing: emits "T<TAB>key<TAB>value" and "M<TAB>id<TAB>title<TAB>command" ----
parse_python() {
  python3 - "$1" "$2" <<'PY'
import sys, yaml
t = yaml.safe_load(open(sys.argv[1])) or {}
m = yaml.safe_load(open(sys.argv[2])) or {}
if not isinstance(t, dict) or not isinstance(m, dict):
    sys.exit("yaml root must be a mapping")
for k, v in t.items():
    print("T\t%s\t%s" % (k, "null" if v is None else v))
ms = m.get("milestones")
if not isinstance(ms, list):
    sys.exit("milestones must be a list")
for e in ms:
    e = e if isinstance(e, dict) else {}
    print("M\t%s\t%s\t%s" % (e.get("id", ""), e.get("title", ""), e.get("command", "")))
PY
}

# Strict line-based parser for exactly the README schema: flat "key: value" in task.yaml,
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
[ -n "$NAME" ] || fail "task.yaml: name missing"
case "$KIND" in long|needs-subagents|parallel) ;; *) fail "task.yaml: kind must be long|needs-subagents|parallel (got '$KIND')" ;; esac
[ "$LANG_" = go ] || fail "task.yaml: language must be go (got '$LANG_')"
[ -n "$GOV" ] || fail "task.yaml: go_version missing"
[[ "$SESS" =~ ^[0-9]+$ ]] && [ "$SESS" -ge 1 ] || fail "task.yaml: sessions must be an integer >= 1 (got '$SESS')"
[[ "$STO" =~ ^[0-9]+$ ]] && [ "$STO" -ge 1 ] || fail "task.yaml: session_timeout_minutes must be a positive integer (got '$STO')"
[[ "$BUD" =~ ^[0-9]+(\.[0-9]+)?$ ]] || fail "task.yaml: budget_usd must be a number (got '$BUD')"
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
done
DUPES=$(printf '%s\n' "${IDS[@]}" | sort | uniq -d || true)
[ -z "$DUPES" ] || fail "milestones.yaml: duplicate ids: $DUPES"

# ---- layout ----
SEED="$TASK/seed"
[ -d "$SEED" ] || fail "seed/ missing"
if [ -d "$SEED" ]; then
  [ -f "$SEED/go.mod" ] || fail "seed/go.mod missing"
  HITS=$(grep -rlE 'pepi-eval-tasks|Jacarte' "$SEED" 2>/dev/null || true)
  [ -z "$HITS" ] || fail "seed mentions the repo (pepi-eval-tasks/Jacarte) in: $HITS"
  [ -z "$(find "$SEED" -name hidden -type d)" ] || fail "hidden/ must not be inside seed/"
fi
[ -d "$TASK/hidden" ] || fail "hidden/ missing"

if [ "$ERRORS" -gt 0 ]; then
  echo "Static checks failed ($ERRORS); not running milestones."
  exit 1
fi

HAVE_GO=$(go env GOVERSION | sed 's/^go//')
[ "$HAVE_GO" = "$GOV" ] || echo "WARN: task go_version $GOV, local go $HAVE_GO"

# ---- run milestones ----
export GOTOOLCHAIN=local
overlay() { cp -R "$1"/. "$2"/; }

# Append -v to a plain `go test` command that lacks it.
verbose_cmd() {
  local c=$1
  if [[ "$c" == "go test"* && ! "$c" =~ (^|[[:space:]])-v([[:space:]]|$) ]]; then
    c="go test -v${c#go test}"
  fi
  printf '%s' "$c"
}

# run_phase <dir> <index>: sets RC and OUT
run_phase() {
  local cmd; cmd=$(verbose_cmd "${CMDS[$2]}")
  set +e
  OUT=$(cd "$1" && bash -c "$cmd" 2>&1)
  RC=$?
  set -e
}

mkdir "$WORK/seed" "$WORK/ref"
cp -R "$SEED"/. "$WORK/seed"/
overlay "$TASK/hidden" "$WORK/seed"
cp -R "$SEED"/. "$WORK/ref"/
overlay "$REF" "$WORK/ref"
overlay "$TASK/hidden" "$WORK/ref"

SEED_RES=(); REF_RES=()
for i in "${!IDS[@]}"; do
  run_phase "$WORK/seed" "$i"
  if [ "$RC" -ne 0 ]; then SEED_RES+=("fails-ok"); else SEED_RES+=("PASSES(bad)"); fail "${IDS[$i]}: passes on the seed"; fi

  run_phase "$WORK/ref" "$i"
  if [ "$RC" -ne 0 ]; then
    REF_RES+=("FAILS(bad)"); fail "${IDS[$i]}: fails on the reference"; printf '%s\n' "$OUT" | tail -n 20
  elif grep -qE 'no tests to run' <<<"$OUT"; then
    REF_RES+=("VACUOUS(bad)"); fail "${IDS[$i]}: output says no tests to run"
  elif ! grep -qE -- '--- PASS|^ok' <<<"$OUT"; then
    REF_RES+=("VACUOUS(bad)"); fail "${IDS[$i]}: no '--- PASS' or 'ok' in output"
  else
    REF_RES+=("passes-ok")
  fi
done

echo
printf '%-24s %-14s %-14s\n' MILESTONE SEED REFERENCE
for i in "${!IDS[@]}"; do
  printf '%-24s %-14s %-14s\n' "${IDS[$i]}" "${SEED_RES[$i]}" "${REF_RES[$i]}"
done
echo
if [ "$ERRORS" -gt 0 ]; then echo "RESULT: FAILED ($ERRORS problem(s))"; exit 1; fi
echo "RESULT: OK ($NAME)"
