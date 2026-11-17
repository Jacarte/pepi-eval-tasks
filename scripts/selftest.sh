#!/usr/bin/env bash
# Self-test for verify-task.sh using throwaway tasks in a temp dir.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
T=$(mktemp -d)
FAILS=0

# Real mode: root on Linux with setpriv, as in the grading container. check-milestone.sh then
# drops privileges for the agent's code and the cases below run against that. Otherwise (macOS,
# a non-root user) the privilege drop is switched off with CHECK_MILESTONE_ALLOW_ROOT=1, which is
# only legitimate for this self-test, and the privilege cases are skipped.
if [ "$(uname)" = Linux ] && [ "$(id -u)" -eq 0 ] && command -v setpriv >/dev/null 2>&1; then
  PRIVILEGED=1
  GRADE_GOCACHE=$(mktemp -d)
  chmod 755 "$GRADE_GOCACHE"
  export GRADE_GOCACHE
  unset CHECK_MILESTONE_ALLOW_ROOT
  echo "selftest mode: privileged (agent code runs as uid 65534)"
else
  PRIVILEGED=0
  export CHECK_MILESTONE_ALLOW_ROOT=1
  echo "selftest mode: CHECK_MILESTONE_ALLOW_ROOT=1 (no privilege drop; privilege cases skipped)"
fi
trap 'rm -rf "$T" "${GRADE_GOCACHE:-}"' EXIT

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

# ---- TestMain in any package with seed or hidden tests, not only the one hidden tests target ----
# mk_audit <case>: the seed also has a package audit/ with its own visible passing test
mk_audit() {
  mk "$1" "$GOOD"
  mkdir -p "$T/$1/task/seed/audit"
  printf 'package audit\n\nfunc Name() string { return "audit" }\n' > "$T/$1/task/seed/audit/audit.go"
  cat > "$T/$1/task/seed/audit/audit_test.go" <<'GO'
package audit

import "testing"

func TestName(t *testing.T) { if Name() != "audit" { t.Fatal("name") } }
GO
}
mk_audit auditmain
mkdir -p "$T/auditmain/ref/audit"
cat > "$T/auditmain/ref/audit/zz_test.go" <<'GO'
package audit

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) { os.Exit(0) }
GO
expect "TestMain os.Exit(0) in a package with seed tests (not hidden target)" 1 auditmain "audit/zz_test.go defines TestMain"

mk_audit auditmainedit
mkdir -p "$T/auditmainedit/ref/audit"
cat > "$T/auditmainedit/ref/audit/audit_test.go" <<'GO'
package audit

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) { os.Exit(0) }
func TestName(t *testing.T) {}
GO
expect "TestMain added to an edited seed test file" 1 auditmainedit "audit/audit_test.go defines TestMain"

# ---- benign code must not be rejected ----
mk_audit benign
cat > "$T/benign/ref/events.go" <<'GO'
package demo

import "os"

// Event names are "<noun>.<verb>" strings.
const Recorded = "test.recorded"

var Files = []string{"report-test.json", "run-test.log"}

func init() { _ = len(os.Args) }
GO
cat > "$T/benign/ref/events_test.go" <<'GO'
package demo

import "testing"

var table map[string]int

func init() {
	table = map[string]int{"test.recorded": 1, "test.run": 2}
}

func TestRecorded(t *testing.T) {
	if table[Recorded] != 1 {
		t.Fatal("recorded")
	}
}
GO
cat > "$T/benign/ref/cli_test.go" <<'GO'
package demo

import (
	"os"
	"testing"
)

func TestArgsInTest(t *testing.T) {
	old := os.Args
	os.Args = []string{"cmd", "-v"}
	defer func() { os.Args = old }()
}
GO
expect "benign event-name literal, init() in test file, os.Args in a test are accepted" 0 benign

# ---- exit-code tricks stay rejected: check-milestone on workspaces with one agent file each ----
trick() { # trick <case> <label> <file> <expected-substring> : file body on stdin
  mk "$1" "$GOOD"
  ws "$1"
  cat > "$T/$1/work/$3"
  check "$2" 1 "$1" m1 "$4"
}
trick tflagset "flag.Set(test.run) in init()" hack.go "touching os.Args or flags" <<'GO'
package demo

import "flag"

func init() { _ = flag.Set("test.run", "^$") }
GO
trick tflagparse "flag.Parse in a var initializer" hack.go "touching os.Args or flags" <<'GO'
package demo

import "flag"

var _ = func() int { flag.Parse(); return 0 }()
GO
trick tflaglookup "flag.Lookup(...).Value.Set in init()" hack.go "touching os.Args or flags" <<'GO'
package demo

import f "flag"

func init() { f.Lookup("test.run").Value.Set("^$") }
GO
trick targsvar "os.Args assigned in a var initializer" hack.go "touching os.Args or flags" <<'GO'
package demo

import "os"

var _ = func() bool { os.Args = append(os.Args, "-x"); return true }()
GO
trick targshelper "os.Args assigned by a helper that init() calls" hack.go "touching os.Args or flags" <<'GO'
package demo

import "os"

func quiet() { os.Args[1] = "-skip" }

func init() { quiet() }
GO
trick tliteral "go test flag string literal" hack.go "mentions go test flags" <<'GO'
package demo

var args = []string{"-test.run=^$"}
GO
trick ttestmain "TestMain in the package hidden tests target" hack_test.go "defines TestMain" <<'GO'
package demo

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) { os.Exit(0) }
GO

# ---- restore_seed_tests ----
# mk_restore <case> <flag>: the seed has a visible test TestVisible that the reference satisfies;
# m1 is a hidden test, m3 is a full-suite regression milestone (no TestM3 tests).
mk_restore() {
  mk "$1" "$GOOD  - id: m3
    title: No regressions
    command: go test ./...
"
  local d=$T/$1/task
  cat > "$d/hidden/demo_test.go" <<'GO'
package demo

import "testing"

func TestM1(t *testing.T) { if One() != 1 { t.Fatal("one") } }
func TestM2(t *testing.T) { if One() != 1 { t.Fatal("one") } }
func TestAlways(t *testing.T) {}
GO
  cat > "$d/seed/visible_test.go" <<'GO'
package demo

import "testing"

func TestVisible(t *testing.T) { if Two() != 2 { t.Fatal("two") } }
GO
  [ "$2" = none ] || echo "restore_seed_tests: $2" >> "$d/task.yaml"
}
# agent_ws <case>: seed + reference, regressed so that TestVisible fails (Two() returns 0)
agent_ws() {
  mkdir -p "$T/$1/work"
  cp -R "$T/$1/task/seed"/. "$T/$1/work"/
  cp -R "$T/$1/ref"/. "$T/$1/work"/
  printf 'package demo\n\nfunc One() int { return 1 }\n\nfunc Two() int { return 0 }\n' > "$T/$1/work/demo.go"
}
prepare() { "$HERE/prepare-workspace.sh" "$T/$1/task" "$T/$1/work"; }

mk_restore restoretrue true
expect "restore_seed_tests: true task verifies (reference passes, seed fails)" 0 restoretrue
mk_restore restorebad maybe
expect "invalid restore_seed_tests value" 1 restorebad "restore_seed_tests must be"

mk_restore delvisible true
agent_ws delvisible
rm "$T/delvisible/work/visible_test.go"
prepare delvisible
check "deleting the failing visible test cannot pass the regression milestone" 1 delvisible m3 "FAIL: TestVisible\|command exited"
if [ -f "$T/delvisible/work/visible_test.go" ]; then echo "ok   deleted visible test was restored"; else echo "FAIL deleted visible test was not restored"; FAILS=$((FAILS + 1)); fi

mk_restore editvisible true
agent_ws editvisible
printf 'package demo\n\nimport "testing"\n\nfunc TestVisible(t *testing.T) {}\n' > "$T/editvisible/work/visible_test.go"
printf 'package demo\n\nimport "testing"\n\nfunc TestExtra(t *testing.T) { t.Skip("x") }\n' > "$T/editvisible/work/extra_test.go"
prepare editvisible
check "weakening the visible test cannot pass the regression milestone" 1 editvisible m3 "command exited"
if [ ! -f "$T/editvisible/work/extra_test.go" ]; then echo "ok   agent-added test file was deleted"; else echo "FAIL agent-added test file survived"; FAILS=$((FAILS + 1)); fi

mk_restore unprepared true
agent_ws unprepared
rm "$T/unprepared/work/visible_test.go"
cp -R "$T/unprepared/task/hidden"/. "$T/unprepared/work"/
check "unprepared workspace is rejected when restore_seed_tests is true" 1 unprepared m3 "seed test visible_test.go is missing or modified"

mk_restore flagfalse false
agent_ws flagfalse
rm "$T/flagfalse/work/visible_test.go"
prepare flagfalse
check "restore_seed_tests: false keeps today's behavior (deleted test goes unnoticed)" 0 flagfalse m3 "PASS: m3"

mk_restore flagabsent none
agent_ws flagabsent
rm "$T/flagabsent/work/visible_test.go"
prepare flagabsent
check "restore_seed_tests absent defaults to false" 0 flagabsent m3 "PASS: m3"

# ---- symlinks and special files in the workspace ----
# The attack: a symlinked zz_test.go whose target is a text file holding a TestMain that fakes
# passing output. Every find that selects regular files misses it (the seed-test restore, the
# cheat scan) while go compiles it. check-milestone.sh and prepare-workspace.sh must reject it.
mk_restore symlinkmain true
agent_ws symlinkmain
mkdir -p "$T/symlinkmain/work/notes"
cat > "$T/symlinkmain/work/notes/x.txt" <<'GO'
package demo

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	fmt.Println("--- PASS: TestM1 (0.00s)")
	fmt.Println("PASS")
	os.Exit(0)
}
GO
ln -s notes/x.txt "$T/symlinkmain/work/zz_test.go"
rc=0
"$HERE/prepare-workspace.sh" "$T/symlinkmain/task" "$T/symlinkmain/work" > "$T/symlinkmain/prep.out" 2>&1 || rc=$?
if [ "$rc" -eq 1 ] && grep -q "workspace contains symlinks or special files: ./zz_test.go" "$T/symlinkmain/prep.out"; then echo "ok   prepare-workspace rejects a symlinked test file (exit 1)"; else echo "FAIL prepare-workspace symlink: got $rc"; cat "$T/symlinkmain/prep.out"; FAILS=$((FAILS + 1)); fi
# a workspace that skipped prepare-workspace.sh is rejected by check-milestone.sh as well
cp -R "$T/symlinkmain/task/hidden"/. "$T/symlinkmain/work"/
check "check-milestone rejects the symlinked TestMain file" 1 symlinkmain m1 "REJECT: workspace contains symlinks or special files: ./zz_test.go"

mk_restore symlinkdangling true
agent_ws symlinkdangling
ln -s /nonexistent/target "$T/symlinkdangling/work/dangling_test.go"
cp -R "$T/symlinkdangling/task/hidden"/. "$T/symlinkdangling/work"/
check "dangling symlink is rejected" 1 symlinkdangling m1 "workspace contains symlinks or special files"

mk_restore symlinkdir true
agent_ws symlinkdir
mkdir -p "$T/symlinkdir/real"
ln -s ../real "$T/symlinkdir/work/linkdir"
cp -R "$T/symlinkdir/task/hidden"/. "$T/symlinkdir/work"/
check "symlinked directory is rejected" 1 symlinkdir m1 "workspace contains symlinks or special files: ./linkdir"

if command -v mkfifo >/dev/null 2>&1; then
  mk_restore fifofile true
  agent_ws fifofile
  mkfifo "$T/fifofile/work/pipe_test.go"
  cp -R "$T/fifofile/task/hidden"/. "$T/fifofile/work"/
  check "fifo named like a test file is rejected (it would hang the scan)" 1 fifofile m1 "workspace contains symlinks or special files: ./pipe_test.go"
fi

# ---- the privilege drop ----
mk_restore noprivs true
agent_ws noprivs
cp -R "$T/noprivs/task/hidden"/. "$T/noprivs/work"/
# the unprivileged user must be able to reach the task, so it lives outside the 0700 $T
NP=$(mktemp -d)
chmod 755 "$NP"
cp -R "$T/noprivs/task" "$T/noprivs/work" "$NP"/
chmod -R a+rX "$NP"
rc=0
if [ "$PRIVILEGED" -eq 1 ]; then
  setpriv --reuid=65534 --regid=65534 --clear-groups env -u CHECK_MILESTONE_ALLOW_ROOT "$HERE/check-milestone.sh" "$NP/task" "$NP/work" m1 > "$T/noprivs/out" 2>&1 || rc=$?
else
  env -u CHECK_MILESTONE_ALLOW_ROOT "$HERE/check-milestone.sh" "$NP/task" "$NP/work" m1 > "$T/noprivs/out" 2>&1 || rc=$?
fi
rm -rf "$NP"
if [ "$rc" -eq 2 ] && grep -q "refusing to run agent code\|must run as root" "$T/noprivs/out"; then echo "ok   without root and setpriv the grader fails closed (exit 2)"; else echo "FAIL fail-closed: got $rc"; cat "$T/noprivs/out"; FAILS=$((FAILS + 1)); fi

# The attack: an init() in the agent's package overwrites the checker's files (found by glob in
# the temp dir) and exits 0. The workspace is the seed, so every milestone really fails: only a
# tampered verdict could turn it into a pass. scancheat does not flag it (no os.Args or flag use).
mk tamper "$GOOD"
ws_seed() { mkdir -p "$T/$1/work"; cp -R "$T/$1/task/seed"/. "$T/$1/work"/; cp -R "$T/$1/task/hidden"/. "$T/$1/work"/; }
ws_seed tamper
cat > "$T/tamper/work/tamper.go" <<'GO'
package demo

import (
	"os"
	"path/filepath"
)

func init() {
	fakeEvents := "{\"Action\":\"pass\",\"Package\":\"demo\",\"Test\":\"TestM1\"}\n{\"Action\":\"pass\",\"Package\":\"demo\",\"Test\":\"TestM2\"}\n"
	for _, dir := range []string{"/tmp", os.TempDir(), "/var/tmp", "/dev/shm"} {
		for _, p := range glob(dir + "/tmp.*/checkjson") {
			_ = os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755)
		}
		for _, p := range glob(dir + "/tmp.*/events") {
			_ = os.WriteFile(p, []byte(fakeEvents), 0o644)
		}
		for _, p := range glob(dir + "/tmp.*/expected") {
			_ = os.WriteFile(p, nil, 0o644)
		}
	}
	os.Exit(0)
}

func glob(pattern string) []string {
	m, _ := filepath.Glob(pattern)
	return m
}
GO
check "init() overwriting the checker, events and expected files does not change the verdict" 1 tamper m1 "REJECT"
if [ "$PRIVILEGED" -eq 1 ]; then
  if grep -q "the result checker was modified" "$T/tamper/check.out"; then echo "FAIL tamper reached the checker binary despite the privilege drop"; FAILS=$((FAILS + 1)); else echo "ok   the privilege drop kept the checker out of reach (rejected by the real result)"; fi
  # Control: with the drop switched off the same attack does reach the checker, and only the
  # hash comparison stops it. This shows the attack above is real.
  rc=0
  CHECK_MILESTONE_ALLOW_ROOT=1 "$HERE/check-milestone.sh" "$T/tamper/task" "$T/tamper/work" m1 > "$T/tamper/control.out" 2>&1 || rc=$?
  if [ "$rc" -eq 1 ] && grep -q "the result checker was modified" "$T/tamper/control.out"; then echo "ok   control: without the privilege drop the attack modifies the checker (caught by its hash)"; else echo "FAIL control: got $rc"; cat "$T/tamper/control.out"; FAILS=$((FAILS + 1)); fi
else
  echo "skip privilege drop proof (needs root + setpriv); the hash check caught the tamper: $(grep -c 'the result checker was modified' "$T/tamper/check.out") match"
fi

if [ "$FAILS" -eq 0 ]; then echo "selftest passed"; else echo "selftest: $FAILS failure(s)"; exit 1; fi
