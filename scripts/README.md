# scripts

## verify-task.sh

```
scripts/verify-task.sh tasks/<name> <reference-dir>
```

`<reference-dir>` is the task's reference solution, normally `../pepi-eval-tasks-reference/<name>` next to this repo. It is not stored here and holds only the files that add or replace seed files.

Checks:

- `task.yaml`/`milestones.yaml` against the README schema, including `scoring` (`prefix|count`, required) and `owned_dirs` (each entry must be a directory under `hidden/`) and the optional `restore_seed_tests` (`true|false`);
- milestone ids start with `m<N>` and commands start with `go test`, or are a skill milestone (`skill_loaded:<name>` or `skill_not_loaded:<name>`, graded only in evals and not run by this script);
- `seed/` has a `go.mod`; `seed/` and `prompts/` never match, case-insensitively, `pepi-eval-tasks`, `Jacarte`, `AGENTS`, `subagent` or `delegat`;
- `prompts/` holds exactly `01.md` .. `NN.md` for `sessions: NN`;
- when `memory/fixture.json` exists (it must for `kind: memory` or a `memory/` dir), its shape (`checkfixture`), and that no file of `seed/` or `prompts/` contains one of its texts.

Then every milestone is judged by `check-milestone.sh` twice. The workspace is built by `prepare-workspace.sh` (see below), as the grading harness does: `owned_dirs` deleted, seed tests restored when `restore_seed_tests: true`, then `hidden/` overlaid:

- seed + `hidden/`: each milestone must fail;
- seed + reference + `hidden/`: each milestone must pass.

Runs with `GOTOOLCHAIN=local`; a `go_version` mismatch only warns. `check-milestone.sh` needs root and `setpriv` (see below): on a machine without them, such as macOS, run it as `CHECK_MILESTONE_ALLOW_ROOT=1 scripts/verify-task.sh ...`. YAML is read with PyYAML when available, otherwise a strict line-based parser for the README schema (the output says which).

Exit codes: `0` all checks pass, `1` a check failed, `2` bad usage.

## prepare-workspace.sh

```
scripts/prepare-workspace.sh <task-dir> <workdir>
```

Turns an agent workspace into a grading workspace, in place: deletes each `owned_dirs` entry; if `restore_seed_tests: true`, deletes every `*_test.go` (edited, deleted-and-recreated and agent-added) and copies the seed's `*_test.go` files back; then copies `hidden/` over it. With the flag `false` or absent, the agent's test files are left alone. A workspace that holds anything but regular files and directories (symlinks, fifos, sockets, devices) is rejected before anything is changed. Exit codes: `0` prepared, `1` workspace rejected (`REJECT: workspace contains symlinks or special files: ...`), `2` bad usage or bad `task.yaml`.

## check-milestone.sh

```
scripts/check-milestone.sh <task-dir> <workdir> <milestone-id>
```

Grades one milestone in `<workdir>`, which must already be prepared with `prepare-workspace.sh`. It runs the milestone command with `go test -json -count=1`, derives the expected `TestM<N>...` names from `hidden/` without running them, and requires a pass event for each, no skip or fail event, and exit code 0. A milestone with no `TestM<N>` tests and no `-run` filter expects every `func Test*` of the seed (files not replaced by `hidden/`) and of `hidden/`, whatever the `restore_seed_tests` flag says; `checkjson` also requires a run event before every pass event, and with `-strict-packages` (used for those commands) rejects a package that holds expected tests and passes with no test event. With `restore_seed_tests: true` it also rejects a workspace whose `*_test.go` files differ from the seed's plus `hidden/`.

It rejects agent-written Go files (not from `hidden/`, not identical to the seed) that carry an exit-code trick, found with `scancheat/main.go` (go/parser, stdlib only): `TestMain` in any directory with seed or hidden tests; `os.Args` assigned or `flag.Set`/`flag.Parse` called in an `init()` body or package var initializer (or a same-directory function they call); string literals that are go test flags (`"-test.run=..."`). Literals like `"test.recorded"` and ordinary `init()` functions in test files are accepted. `jq` is not needed: the event stream is read by `checkjson/main.go`. Both helpers are built on each run.

### Isolation of the agent's code

- **Special files.** The workspace is rejected (exit 1) if it holds a symlink, fifo, socket or device anywhere. A symlink to a text file named `*_test.go` would be compiled by `go` but missed by every `find -type f`, so a fake `TestMain` could hide behind it. Every `find` of the scripts selects `! -type d`, so nothing but directories is skipped.
- **Unprivileged run.** The `go test` command runs as uid/gid 65534 with no groups (`setpriv --reuid=65534 --regid=65534 --clear-groups`), on a copy of the workspace. It gets its own `HOME`, `GOPATH`, `GOTMPDIR`, `TMPDIR` and `GOCACHE` (set `GRADE_GOCACHE=<dir>` to share a pre-warmed build cache between runs). The event stream, the expected tests and the `checkjson` binary live in a root-owned `0700` directory; `checkjson` is built before the run, its SHA-256 is recorded, and the result is only used if the hash is unchanged. Every process of that user is killed when the run ends.
- **Read-only workspace.** Only the scratch directories (`HOME`, `GOPATH`, `GOTMPDIR`, `TMPDIR`, the Go build cache) belong to the unprivileged user. The workspace copy, hidden tests included, stays root-owned and not writable by it, so agent code cannot rewrite another package's tests while they build or run. A test that must write inside its own package directory at run time (fixtures written under `testdata`, say) fails under this rule: write to `t.TempDir()` instead. `setpriv --no-new-privs` also keeps setuid binaries from raising privileges.
- **Early exits and forged output.** `scancheat` rejects `os.Exit`, `syscall.Exit`, `unix.Exit`, `log.Fatal*`, `runtime.Goexit` and raw syscalls reached from `init()` or a package var initializer (also through functions of the same directory), `syscall.Exit` or `unix.Exit` anywhere outside `package main`, and string literals containing `/proc/`, `--- PASS` or `"Action"` (struct tags excepted). An exit in `main()` of a `package main` is fine. An exit hidden in a function of *another* directory is not seen by the scan: a full-suite milestone catches it, because the package's seed tests then have no pass event.
- **Harness errors.** An exit 2 (or a script crash) is a grading harness error, never a score: the module returns it as an error and aborts the whole run, parallel tasks included, so nothing is cached as an agent failure.
- **Fail closed.** Without root or without `setpriv` the script exits 2 with a clear message instead of running the agent's code with the grader's privileges.
- **`CHECK_MILESTONE_ALLOW_ROOT=1`** turns the drop off. Use it **only for local self-tests on a machine without setpriv (macOS)**. With it the agent's code runs with your privileges; only the hash comparison still catches a replaced checker. Never set it in a grading container.
- **Known limits.** The test output is produced by a process of the same user as the code under test (the `go` command and the test binary run unprivileged together). Forging it needs code the scan above flags, but the scan is static: a forgery that builds its markers at run time from pieces, calls an exit function through a helper in another directory, and does so in a package without seed tests, is not detected by it. The full-suite and run-before-pass checks narrow this, they do not close it. Building as root and running only the test binaries unprivileged (`go test -exec`) would put the `go` command and the events file out of the agent's reach; it is not done, and text printed by the test binary itself would still be parsed.

Exit codes: `0` milestone passed, `1` failed or rejected, `2` bad usage or a harness problem (no root or `setpriv`, a bad milestone id).

## selftest.sh

`scripts/selftest.sh` builds throwaway tasks (one good, several broken) in a temp dir and asserts the exit codes of verify-task.sh and check-milestone.sh, including the symlink and fifo cases (a symlinked `zz_test.go` holding a fake `TestMain` must be rejected), file names with control characters, early exits and forged output (`os.Exit(0)` in `init()` at the full-suite milestone, `syscall.Exit` called from a library function, `/proc/` and `--- PASS` literals, an exit hidden in another package), canned `checkjson` streams, an agent package that rewrites another package's hidden test (Linux root run only), the privilege drop and an `init()` that overwrites the checker's files and exits 0 (run as root on Linux it exercises the real drop, with a control run without it; elsewhere it runs with `CHECK_MILESTONE_ALLOW_ROOT=1` and skips the privilege-only checks; run it in a container with `docker run --rm -v "$PWD":/r:ro golang:1.26 bash /r/scripts/selftest.sh`, installing `gawk` first), the exit-code gaming cases (`TestMain` with `os.Exit(0)` in the hidden-test package or in another package with seed tests, `init()` or var initializers rewriting `os.Args` or calling `flag.Set`, test-flag literals, a skipped expected test, an expected test that never ran), the benign cases that must be accepted, and the `restore_seed_tests` regression gate (deleting or weakening a visible test fails with the flag; without it a full-suite milestone still notices a deleted seed test). Cleans up after itself.
