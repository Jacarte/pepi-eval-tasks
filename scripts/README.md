# scripts

## verify-task.sh

```
scripts/verify-task.sh tasks/<name> /Users/javier.cabrera/DEV/pepi-eval-tasks-reference/<name>
```

Checks `task.yaml`/`milestones.yaml` against the README schema, that `seed/` has a `go.mod` and never mentions the repo, then runs every milestone twice:

- seed + `hidden/`: each milestone must fail;
- seed + reference + `hidden/`: each milestone must pass, print `--- PASS` or `ok`, and not say `no tests to run`.

Plain `go test` commands get `-v` added. Runs with `GOTOOLCHAIN=local`; a `go_version` mismatch only warns. YAML is read with PyYAML when available, otherwise a strict line-based parser for the README schema (the output says which).

Reference solutions live in the sibling directory `/Users/javier.cabrera/DEV/pepi-eval-tasks-reference/<task-name>/`, not in this repo. They hold only the files that add or replace seed files.

Exit codes: `0` all checks pass, `1` a check failed, `2` bad usage.

## selftest.sh

`scripts/selftest.sh` builds throwaway tasks (one good, several broken) in a temp dir and asserts verify-task.sh's exit codes. Cleans up after itself.
