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
  mkdir -p "$d/seed" "$d/hidden" "$ref"
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
Y
  printf 'milestones:\n%s' "$2" > "$d/milestones.yaml"
}

# expect <label> <want-exit> <case>
expect() {
  local rc=0
  "$HERE/verify-task.sh" "$T/$3/task" "$T/$3/ref" > "$T/$3/out" 2>&1 || rc=$?
  if [ "$rc" -eq "$2" ]; then echo "ok   $1 (exit $rc)"; else echo "FAIL $1: want $2 got $rc"; sed 's/^/    /' "$T/$3/out"; FAILS=$((FAILS + 1)); fi
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
expect "milestone passes on seed" 1 passes_seed

mk vacuous "$GOOD  - id: m3
    title: Nothing
    command: go test ./... -run '^TestNope\$'
"
expect "vacuous milestone" 1 vacuous

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

if [ "$FAILS" -eq 0 ]; then echo "selftest passed"; else echo "selftest: $FAILS failure(s)"; exit 1; fi
