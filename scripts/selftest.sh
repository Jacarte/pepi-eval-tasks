#!/usr/bin/env bash
# Self-test for verify-task.sh using throwaway tasks in a temp dir.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT
FAILS=0

# mk <name> <milestone-yaml-body> : creates task + reference with 2 good milestones' files
mk() {
  local d=$T/$1/task ref=$T/$1/ref
  mkdir -p "$d/seed" "$d/hidden" "$d/prompts" "$ref"
  echo 'Implement the package.' > "$d/prompts/01.md"
  printf 'module demo\n\ngo 1.22\n' > "$d/seed/go.mod"
  printf 'package demo\n\nfunc One() int { return 0 }\n\nfunc Two() int { return 0 }\n' > "$d/seed/demo.go"
  printf 'package demo\n\nfunc One() int { return 1 }\n\nfunc Two() int { return 2 }\n' > "$ref/demo.go"
  cat > "$d/hidden/demo_test.go" <<'GO'
package demo

import "testing"

func TestM1(t *testing.T) { if One() != 1 { t.Fatal("one") } }
func TestM2(t *testing.T) { if Two() != 2 { t.Fatal("two") } }
func TestAlways(t *testing.T) {}
GO
  cat > "$d/task.yaml" <<'Y'
name: demo
kind: long
language: go
go_version: 1.22.5
sessions: 1
session_timeout_minutes: 5
budget_usd: 1
deadline_minutes: null
scoring: count
Y
  printf 'milestones:\n%s' "$2" > "$d/milestones.yaml"
}

# expect <label> <want-exit> <case> [output-substring]
expect() {
  local rc=0
  "$HERE/verify-task.sh" "$T/$3/task" "$T/$3/ref" > "$T/$3/out" 2>&1 || rc=$?
  if [ "$rc" -eq "$2" ] && { [ $# -lt 4 ] || grep -q -- "$4" "$T/$3/out"; }; then echo "ok   $1 (exit $rc)"; else echo "FAIL $1: want $2${4:+ with <$4>} got $rc"; sed 's/^/    /' "$T/$3/out"; FAILS=$((FAILS + 1)); fi
}

GOOD="  - id: m1
    title: One
    command: go test ./... -run '^TestM1\$'
  - id: m2
    title: Two
    command: go test ./... -run '^TestM2\$' -v
"
mk good "$GOOD"
expect "good task" 0 good

mk passes_seed "$GOOD  - id: m3
    title: Trivial
    command: go test ./... -run '^TestAlways\$'
"
expect "milestone passes on seed" 1 passes_seed "passes on the seed"

mk vacuous "$GOOD  - id: m3
    title: Nothing
    command: go test ./... -run '^TestNope\$'
"
expect "vacuous milestone" 1 vacuous "m3: fails on the reference"

mk nofield "$GOOD"
sed -i.bak '/^budget_usd/d' "$T/nofield/task/task.yaml"
expect "missing field" 1 nofield

mk dupid "$GOOD  - id: m1
    title: Dup
    command: go test ./...
"
expect "duplicate id" 1 dupid

mk leak "$GOOD"
echo '// see pepi-eval-tasks' >> "$T/leak/task/seed/demo.go"
expect "seed mentions repo" 1 leak

mk badscoring "$GOOD"
sed -i.bak 's/^scoring: .*/scoring: order/' "$T/badscoring/task/task.yaml"
expect "invalid scoring value" 1 badscoring "scoring must be"

mk noscoring "$GOOD"
sed -i.bak '/^scoring/d' "$T/noscoring/task/task.yaml"
expect "missing scoring" 1 noscoring "scoring must be"

mk leakprompt "$GOOD"
echo 'Ask a Subagent for help.' >> "$T/leakprompt/task/prompts/01.md"
expect "prompt mentions subagent" 1 leakprompt "forbidden word"

mk leakword "$GOOD"
echo '// work is delegated' >> "$T/leakword/task/seed/demo.go"
expect "seed mentions delegat" 1 leakword "forbidden word"

mk extraprompt "$GOOD"
echo 'More.' > "$T/extraprompt/task/prompts/02.md"
expect "prompt count differs from sessions" 1 extraprompt "prompts/ must hold"

# owned_dirs: hidden/acc owns acc/. The reference leaves a broken file there, which must be deleted.
mk_owned() {
  mk "$1" "  - id: m1
    title: One
    command: go test ./acc/ -run '^TestM1\$'
  - id: m2
    title: Two
    command: go test ./acc/ -run '^TestM2\$'
"
  local d=$T/$1/task
  rm "$d/hidden/demo_test.go"
  mkdir -p "$d/hidden/acc" "$T/$1/ref/acc"
  cat > "$d/hidden/acc/acc_test.go" <<'GO'
package acc

import (
	"testing"

	"demo"
)

func TestM1(t *testing.T) { if demo.One() != 1 { t.Fatal("one") } }
func TestM2(t *testing.T) { if demo.Two() != 2 { t.Fatal("two") } }
GO
  echo 'package otherpkg' > "$T/$1/ref/acc/junk_test.go"
}
mk_owned owned
echo 'owned_dirs: [acc]' >> "$T/owned/task/task.yaml"
expect "owned_dirs removes agent files before overlay" 0 owned

mk_owned notowned
expect "agent file in hidden dir without owned_dirs breaks milestones" 1 notowned "fails on the reference"

mk_owned ownedmissing
echo 'owned_dirs: [nope]' >> "$T/ownedmissing/task/task.yaml"
expect "owned_dirs entry missing from hidden/" 1 ownedmissing "not a directory under hidden"

# Gaming attempts: each must make verify-task.sh reject the reference.
mk testmain "$GOOD"
cat > "$T/testmain/ref/agent_test.go" <<'GO'
package demo

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) { os.Exit(0) }
GO
expect "TestMain os.Exit(0) trick" 1 testmain "defines TestMain"

mk initargs "$GOOD"
cat > "$T/initargs/ref/hack.go" <<'GO'
package demo

import "os"

func init() { os.Args = append(os.Args, "-test.run=^$") }
GO
expect "init() rewriting os.Args" 1 initargs "init() touching os.Args"

# Direct check-milestone cases on a workspace (seed + reference + hidden).
ws() { # ws <case> : copy of the good task's reference workspace
  mkdir -p "$T/$1/work"
  cp -R "$T/good/task/seed"/. "$T/$1/work"/
  cp -R "$T/good/ref"/. "$T/$1/work"/
  cp -R "$T/$1/task/hidden"/. "$T/$1/work"/
}
check() { # check <label> <want-exit> <case> <milestone> <expected-output-substring>
  local rc=0
  "$HERE/check-milestone.sh" "$T/$3/task" "$T/$3/work" "$4" > "$T/$3/check.out" 2>&1 || rc=$?
  if [ "$rc" -eq "$2" ] && grep -q -- "$5" "$T/$3/check.out"; then echo "ok   $1 (exit $rc)"; else echo "FAIL $1: want $2 with '$5', got $rc"; sed 's/^/    /' "$T/$3/check.out"; FAILS=$((FAILS + 1)); fi
}

mk skipped "$GOOD"
cat > "$T/skipped/task/hidden/demo_test.go" <<'GO'
package demo

import "testing"

func TestM1(t *testing.T) { t.Skip("not today") }
func TestM2(t *testing.T) { if Two() != 2 { t.Fatal("two") } }
GO
ws skipped
check "skipped expected test is rejected" 1 skipped m1 "skip event"
check "other milestone of the same workspace passes" 0 skipped m2 "PASS: m2"

mk missing "$GOOD"
cat > "$T/missing/task/hidden/demo_test.go" <<'GO'
package demo

import "testing"

func TestM1A(t *testing.T) { if One() != 1 { t.Fatal("one") } }
func TestM1B(t *testing.T) { if One() != 1 { t.Fatal("one") } }
GO
ws missing
sed -i.bak "s/'^TestM1\\\$'/'^TestM1A\\\$'/" "$T/missing/task/milestones.yaml"
check "expected test that never ran is rejected" 1 missing m1 "TestM1B: no pass event"

mk goodcheck "$GOOD"
ws goodcheck
check "check-milestone accepts a clean workspace" 0 goodcheck m1 "PASS: m1 (1 expected tests)"

if [ "$FAILS" -eq 0 ]; then echo "selftest passed"; else echo "selftest: $FAILS failure(s)"; exit 1; fi
