# scripts

## verify-task.sh

```
scripts/verify-task.sh tasks/<name> <reference-dir>
```

`<reference-dir>` is the task's reference solution, normally `../pepi-eval-tasks-reference/<name>` next to this repo. It is not stored here and holds only the files that add or replace seed files.

Checks:

- `task.yaml`/`milestones.yaml` against the README schema, including `scoring` (`prefix|count`, required) and `owned_dirs` (each entry must be a directory under `hidden/`) and the optional `restore_seed_tests` (`true|false`);
- milestone ids start with `m<N>` and commands start with `go test`;
- `seed/` has a `go.mod`; `seed/` and `prompts/` never match, case-insensitively, `pepi-eval-tasks`, `Jacarte`, `AGENTS`, `subagent` or `delegat`;
- `prompts/` holds exactly `01.md` .. `NN.md` for `sessions: NN`.

Then every milestone is judged by `check-milestone.sh` twice. The workspace is built by `prepare-workspace.sh` (see below), as the grading harness does: `owned_dirs` deleted, seed tests restored when `restore_seed_tests: true`, then `hidden/` overlaid:

- seed + `hidden/`: each milestone must fail;
- seed + reference + `hidden/`: each milestone must pass.

Runs with `GOTOOLCHAIN=local`; a `go_version` mismatch only warns. YAML is read with PyYAML when available, otherwise a strict line-based parser for the README schema (the output says which).

Exit codes: `0` all checks pass, `1` a check failed, `2` bad usage.

## prepare-workspace.sh

```
scripts/prepare-workspace.sh <task-dir> <workdir>
```

Turns an agent workspace into a grading workspace, in place: deletes each `owned_dirs` entry; if `restore_seed_tests: true`, deletes every `*_test.go` (edited, deleted-and-recreated and agent-added) and copies the seed's `*_test.go` files back; then copies `hidden/` over it. With the flag `false` or absent, the agent's test files are left alone. Exit codes: `0` prepared, `2` bad usage or bad `task.yaml`.

## check-milestone.sh

```
scripts/check-milestone.sh <task-dir> <workdir> <milestone-id>
```

Grades one milestone in `<workdir>`, which must already be prepared with `prepare-workspace.sh`. It runs the milestone command with `go test -json -count=1`, derives the expected `TestM<N>...` names from `hidden/` without running them, and requires a pass event for each, no skip or fail event, and exit code 0. With `restore_seed_tests: true` it also rejects a workspace whose `*_test.go` files differ from the seed's plus `hidden/`.

It rejects agent-written Go files (not from `hidden/`, not identical to the seed) that carry an exit-code trick, found with `scancheat/main.go` (go/parser, stdlib only): `TestMain` in any directory with seed or hidden tests; `os.Args` assigned or `flag.Set`/`flag.Parse` called in an `init()` body or package var initializer (or a same-directory function they call); string literals that are go test flags (`"-test.run=..."`). Literals like `"test.recorded"` and ordinary `init()` functions in test files are accepted. `jq` is not needed: the event stream is read by `checkjson/main.go`. Both helpers are built on each run.

Exit codes: `0` milestone passed, `1` failed or rejected, `2` bad usage.

## selftest.sh

`scripts/selftest.sh` builds throwaway tasks (one good, several broken) in a temp dir and asserts the exit codes of verify-task.sh and check-milestone.sh, including the exit-code gaming cases (`TestMain` with `os.Exit(0)` in the hidden-test package or in another package with seed tests, `init()` or var initializers rewriting `os.Args` or calling `flag.Set`, test-flag literals, a skipped expected test, an expected test that never ran), the benign cases that must be accepted, and the `restore_seed_tests` regression gate (deleting or weakening a visible test fails with the flag, passes without). Cleans up after itself.
