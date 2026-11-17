# scripts

## verify-task.sh

```
scripts/verify-task.sh tasks/<name> <reference-dir>
```

`<reference-dir>` is the task's reference solution, normally `../pepi-eval-tasks-reference/<name>` next to this repo. It is not stored here and holds only the files that add or replace seed files.

Checks:

- `task.yaml`/`milestones.yaml` against the README schema, including `scoring` (`prefix|count`, required) and `owned_dirs` (each entry must be a directory under `hidden/`);
- milestone ids start with `m<N>` and commands start with `go test`;
- `seed/` has a `go.mod`; `seed/` and `prompts/` never match, case-insensitively, `pepi-eval-tasks`, `Jacarte`, `AGENTS`, `subagent` or `delegat`;
- `prompts/` holds exactly `01.md` .. `NN.md` for `sessions: NN`.

Then every milestone is judged by `check-milestone.sh` twice. Before `hidden/` is overlaid, each `owned_dirs` entry is deleted, as the grading harness does:

- seed + `hidden/`: each milestone must fail;
- seed + reference + `hidden/`: each milestone must pass.

Runs with `GOTOOLCHAIN=local`; a `go_version` mismatch only warns. YAML is read with PyYAML when available, otherwise a strict line-based parser for the README schema (the output says which).

Exit codes: `0` all checks pass, `1` a check failed, `2` bad usage.

## check-milestone.sh

```
scripts/check-milestone.sh <task-dir> <workdir> <milestone-id>
```

Grades one milestone in `<workdir>`, which must already contain the agent workspace with `hidden/` overlaid (see README, "Grading rules"). It runs the milestone command with `go test -json -count=1`, derives the expected `TestM<N>...` names from `hidden/` without running them, and requires a pass event for each, no skip or fail event, and exit code 0. It rejects agent-written Go files with a `TestMain` next to hidden tests or an `init()` that manipulates `os.Args` or test flags. `jq` is not needed: the event stream is read by `checkjson/main.go` (stdlib only, built on each run).

Exit codes: `0` milestone passed, `1` failed or rejected, `2` bad usage.

## selftest.sh

`scripts/selftest.sh` builds throwaway tasks (one good, several broken) in a temp dir and asserts the exit codes of verify-task.sh and check-milestone.sh, including the exit-code gaming cases (`TestMain` with `os.Exit(0)`, `init()` rewriting `os.Args`, a skipped expected test, an expected test that never ran). Cleans up after itself.
