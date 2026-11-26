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

mk skill "$GOOD  - id: m3
    title: Skill loaded
    command: skill_loaded:demo-skill
  - id: m4
    title: Skill not loaded
    command: skill_not_loaded:other-skill
"
expect "skill milestones are not run" 0 skill "skill-eval"

mk badskill "$GOOD  - id: m3
    title: Bad skill name
    command: skill_loaded:Bad_Name
"
expect "skill milestone with a bad name" 1 badskill "skill command must be"

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

# mk_memory <case> <fixture-json>: a kind=memory task with memory/fixture.json
mk_memory() {
  mk "$1" "$GOOD"
  sed -i.bak 's/^kind: .*/kind: memory/' "$T/$1/task/task.yaml"
  mkdir -p "$T/$1/task/memory"
  printf '%s\n' "$2" > "$T/$1/task/memory/fixture.json"
}
mk_memory memkind '{"memories": [{"text": "One and Two live in demo.go."}]}'
expect "memory kind with a fixture" 0 memkind

mk_memory badfixture '{"memories": [{"text": " "}]}'
expect "fixture with an empty memory text" 1 badfixture "text is empty"

mk_memory fixtureinseed '{"memories": [{"text": "One and Two live in demo.go."}]}'
cp "$T/fixtureinseed/task/memory/fixture.json" "$T/fixtureinseed/task/seed/notes.json"
expect "seed holding a copy of the fixture" 1 fixtureinseed "contains memories"

mk_memory fixtureinprompt '{"memories": [{"text": "One and Two live in demo.go."}]}'
echo 'One and Two live in demo.go.' >> "$T/fixtureinprompt/task/prompts/01.md"
expect "prompt holding a fixture text" 1 fixtureinprompt "contains memories"

mk_memory fixturekey '{"memories": [{"text": "x", "infer": true}]}'
expect "fixture with an unknown key" 1 fixturekey "unknown field"

mk_memory fixturemany "{\"memories\": [$(printf '{"text": "m%d"},' $(seq 20)){\"text\": \"m21\"}]}"
expect "fixture with more than 20 memories" 1 fixturemany "1..20 entries (got 21)"

mk_memory fixturetrailing '{"memories": [{"text": "x"}]}]'
expect "fixture with trailing data" 1 fixturetrailing "trailing data"

mk_memory nofixture '{}'
rm -r "$T/nofixture/task/memory"
expect "kind memory without a fixture" 1 nofixture "fixture.json missing"

mk memdir "$GOOD"
mkdir "$T/memdir/task/memory"
expect "memory/ dir without fixture.json" 1 memdir "fixture.json missing"

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
check "restore_seed_tests: false, a deleted seed test is still noticed by the full-suite milestone" 1 flagfalse m3 "TestVisible: no pass event"

mk_restore flagabsent none
agent_ws flagabsent
rm "$T/flagabsent/work/visible_test.go"
prepare flagabsent
check "restore_seed_tests absent defaults to false, full-suite milestone still expects the seed's tests" 1 flagabsent m3 "TestVisible: no pass event"

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


# ---- file names with control characters ----
# The agent-file list is one name per line: "zz<newline>main_test.go" would read as two names that
# do not exist, and the cheat scan would never look at the TestMain inside.
mk_restore newlinename true
agent_ws newlinename
NLNAME=$(printf 'zz\nmain_test.go')
cat > "$T/newlinename/work/$NLNAME" <<'GO'
package demo

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) { os.Exit(0) }
GO
rc=0
"$HERE/prepare-workspace.sh" "$T/newlinename/task" "$T/newlinename/work" > "$T/newlinename/prep.out" 2>&1 || rc=$?
if [ "$rc" -eq 1 ] && grep -q "REJECT: workspace contains file names with control characters" "$T/newlinename/prep.out"; then echo "ok   prepare-workspace rejects a newline-named file (exit 1)"; else echo "FAIL prepare newline name: got $rc"; cat "$T/newlinename/prep.out"; FAILS=$((FAILS + 1)); fi
cp -R "$T/newlinename/task/hidden"/. "$T/newlinename/work"/
check "check-milestone rejects the newline-named TestMain file" 1 newlinename m3 "REJECT: workspace contains file names with control characters"

# scancheat never skips a listed file it cannot open
(cd "$HERE/scancheat" && go build -o "$T/scancheat" main.go)
rc=0
printf 'does/not/exist.go\n' | "$T/scancheat" -root "$T" > "$T/scancheat.out" 2>&1 || rc=$?
if [ "$rc" -eq 2 ]; then echo "ok   scancheat exits 2 for a listed file it cannot open"; else echo "FAIL scancheat missing file: got $rc"; cat "$T/scancheat.out"; FAILS=$((FAILS + 1)); fi

# ---- processes that end early, forged output, hidden-test tampering ----
# trick3 <case> <label> <file> <expected-substring>: like trick, on the restore task, milestone m3 (go test ./...)
trick3() {
  mk_restore "$1" true
  agent_ws "$1"
  cat > "$T/$1/work/$3"
  cp -R "$T/$1/task/hidden"/. "$T/$1/work"/
  check "$2" 1 "$1" m3 "$4"
}
trick3 initexit "os.Exit(0) in init() at the full-suite milestone" exit.go "can end or hijack the test process" <<'GO'
package demo

import "os"

func init() { os.Exit(0) }
GO
trick3 initexitref "os.Exit taken as a value in a package var initializer" exit.go "can end or hijack the test process" <<'GO'
package demo

import "os"

var quit = os.Exit

func init() { quit(0) }
GO
trick3 initlogfatal "log.Fatal reached from init() through a helper" exit.go "can end or hijack the test process" <<'GO'
package demo

import "log"

func bail() { log.Fatal("done") }

func init() { bail() }
GO
trick3 syscallexit "syscall.Exit in a library function a hidden test calls, printing a fake PASS" exit.go "can end or hijack the test process" <<'GO'
package demo

import (
	"fmt"
	"syscall"
)

func Fake() {
	fmt.Println("---", "PASS: TestM1 (0.00s)")
	syscall.Exit(0)
}
GO
trick3 procfd "string literal naming /proc/ (raw JSON written to the parent's output fd)" exit.go "marker of forged test output" <<'GO'
package demo

import "os"

func init() {
	f, err := os.OpenFile("/proc/"+"self/stat", os.O_WRONLY, 0)
	if err == nil {
		f.Close()
	}
}
GO
trick3 passlit "string literal with --- PASS" exit.go "marker of forged test output" <<'GO'
package demo

var fake = "--- PASS: TestM1 (0.00s)"
GO
trick3 actionlit "string literal with a go test -json Action key" exit.go "marker of forged test output" <<'GO'
package demo

var fake = `{"Action":"pass","Package":"demo","Test":"TestM1"}`
GO

# An exit hidden in another directory is not reached by the scan; the full-suite milestone then
# misses the pass events of the seed's tests in the package that exited.
mk_audit crossexit
cat >> "$T/crossexit/task/milestones.yaml" <<'Y'
  - id: m3
    title: Full suite
    command: go test ./...
Y
mkdir -p "$T/crossexit/work/helper" "$T/crossexit/work/audit"
ws crossexit
cp -R "$T/crossexit/task/seed/audit"/. "$T/crossexit/work/audit"/
cat > "$T/crossexit/work/helper/helper.go" <<'GO'
package helper

import "os"

func Quit() { os.Exit(0) }
GO
cat > "$T/crossexit/work/audit/early.go" <<'GO'
package audit

import "demo/helper"

func init() { helper.Quit() }
GO
check "exit hidden in another package is caught by the expected seed tests" 1 crossexit m3 "TestName: no pass event\|expected tests not all passing"

# Benign: an exit in package main's main(), log.Fatal there, and a struct tag with "Action"
mk_audit benignmain
cat >> "$T/benignmain/task/milestones.yaml" <<'Y'
  - id: m3
    title: Full suite
    command: go test ./...
Y
ws benignmain
mkdir -p "$T/benignmain/work/audit" "$T/benignmain/work/cmd"
cp -R "$T/benignmain/task/seed/audit"/. "$T/benignmain/work/audit"/
cat > "$T/benignmain/work/cmd/main.go" <<'GO'
package main

import (
	"log"
	"os"
	"syscall"
)

type Event struct {
	Action string `json:"Action"`
}

func main() {
	if len(os.Args) > 5 {
		log.Fatal("too many arguments")
	}
	if len(os.Args) > 9 {
		syscall.Exit(3)
	}
	os.Exit(0)
}
GO
check "os.Exit, log.Fatal and syscall.Exit in package main's main() are accepted" 0 benignmain m3 "PASS: m3"

# checkjson on canned streams
(cd "$HERE/checkjson" && go build -o "$T/checkjson" main.go)
printf 'demo\tTestA\n' > "$T/expected-a"
cj() { # cj <label> <want-exit> <flags...> ; stream on stdin
  local label=$1 want=$2 rc=0
  shift 2
  "$T/checkjson" "$@" > "$T/cj.out" 2>&1 || rc=$?
  if [ "$rc" -eq "$want" ]; then echo "ok   checkjson: $label (exit $rc)"; else echo "FAIL checkjson: $label: want $want got $rc"; cat "$T/cj.out"; FAILS=$((FAILS + 1)); fi
}
cj "run then pass is accepted" 0 -expected "$T/expected-a" -strict-packages <<'JSON'
{"Action":"run","Package":"m/demo","Test":"TestA"}
{"Action":"pass","Package":"m/demo","Test":"TestA"}
{"Action":"pass","Package":"m/demo"}
JSON
cj "a pass with no run event before it is rejected" 1 -expected "$T/expected-a" <<'JSON'
{"Action":"pass","Package":"m/demo","Test":"TestA"}
{"Action":"pass","Package":"m/demo"}
JSON
cj "a pass before the run event is rejected" 1 -expected "$T/expected-a" <<'JSON'
{"Action":"pass","Package":"m/demo","Test":"TestA"}
{"Action":"run","Package":"m/demo","Test":"TestA"}
JSON
cj "package pass with no test event is rejected under -strict-packages" 1 -expected "$T/expected-a" -strict-packages <<'JSON'
{"Action":"pass","Package":"m/demo"}
JSON
cj "without -strict-packages a bare package pass still lacks the expected test" 1 -expected "$T/expected-a" <<'JSON'
{"Action":"pass","Package":"m/demo"}
JSON

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
# The exit is taken in another package, and the literals are split, so the static scan lets this
# through: what stops it is the privilege drop (and, without it, the checker's hash).
mkdir -p "$T/tamper/work/quit"
cat > "$T/tamper/work/quit/quit.go" <<'GO'
package quit

import "os"

func Now() { os.Exit(0) }
GO
cat > "$T/tamper/work/tamper.go" <<'GO'
package demo

import (
	"os"
	"path/filepath"

	"demo/quit"
)

func init() {
	fakeEvents := "{\"Act" + "ion\":\"pass\",\"Package\":\"demo\",\"Test\":\"TestM1\"}\n{\"Act" + "ion\":\"pass\",\"Package\":\"demo\",\"Test\":\"TestM2\"}\n"
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
	quit.Now()
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

# An agent package rewrites another package's hidden test while it is being built or run. The
# workspace copy is read-only for the unprivileged user, so the write fails and TestM1 stays red.
if [ "$PRIVILEGED" -eq 1 ]; then
  mk rewrite "$GOOD"
  rm "$T/rewrite/task/hidden/demo_test.go"
  mkdir -p "$T/rewrite/task/hidden/z"
  cat > "$T/rewrite/task/hidden/z/z_test.go" <<'GO'
package z

import (
	"testing"

	"demo"
)

func TestM1(t *testing.T) { if demo.One() != 1 { t.Fatal("one") } }
GO
  printf 'package z\n' > "$T/rewrite/task/seed/z.go"
  mkdir -p "$T/rewrite/task/seed/z" "$T/rewrite/work/a" "$T/rewrite/work/z"
  mv "$T/rewrite/task/seed/z.go" "$T/rewrite/task/seed/z/z.go"
  cp -R "$T/rewrite/task/seed"/. "$T/rewrite/work"/
  cp -R "$T/rewrite/task/hidden"/. "$T/rewrite/work"/
  cat > "$T/rewrite/work/a/a.go" <<'GO'
package a

import "os"

func init() {
	// the working directory of this package's test binary is its own directory
	_ = os.WriteFile("../z/z_test.go", []byte("package z\n\nimport \"testing\"\n\nfunc TestM1(t *testing.T) {}\n"), 0o644)
}
GO
  cat > "$T/rewrite/work/a/a_test.go" <<'GO'
package a

import "testing"

func TestA(t *testing.T) {}
GO
  printf 'milestones:\n  - id: m1\n    title: One\n    command: go test ./a/ ./z/\n' > "$T/rewrite/task/milestones.yaml"
  rc=0
  GOFLAGS=-p=1 "$HERE/check-milestone.sh" "$T/rewrite/task" "$T/rewrite/work" m1 > "$T/rewrite/check.out" 2>&1 || rc=$?
  if [ "$rc" -eq 1 ] && grep -q "TestM1" "$T/rewrite/check.out" && ! grep -q "^PASS: m1" "$T/rewrite/check.out"; then echo "ok   a package rewriting another package's hidden test cannot change the verdict (exit 1)"; else echo "FAIL hidden test rewrite: got $rc"; cat "$T/rewrite/check.out"; FAILS=$((FAILS + 1)); fi
  # Control: the same attack with the privilege drop off rewrites the test and passes.
  rc=0
  GOFLAGS=-p=1 CHECK_MILESTONE_ALLOW_ROOT=1 "$HERE/check-milestone.sh" "$T/rewrite/task" "$T/rewrite/work" m1 > "$T/rewrite/control.out" 2>&1 || rc=$?
  if [ "$rc" -eq 0 ] && grep -q "^PASS: m1" "$T/rewrite/control.out"; then echo "ok   control: without the drop the same attack rewrites the hidden test and passes"; else echo "FAIL control hidden test rewrite: got $rc"; cat "$T/rewrite/control.out"; FAILS=$((FAILS + 1)); fi
fi

if [ "$FAILS" -eq 0 ]; then echo "selftest passed"; else echo "selftest: $FAILS failure(s)"; exit 1; fi
