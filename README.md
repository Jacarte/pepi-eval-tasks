# pepi-eval-tasks

Task fixtures for the `pepi` Dagger evals (`daggervals`). They measure when delegating to subagents pays off: long-horizon projects, tasks that need subagents, and tasks where parallel agents can help.

The harness loads this repo at a pinned commit:

```
dagger call daggervals evals ... --suite tasks --tasks-repo https://github.com/Jacarte/pepi-eval-tasks --tasks-ref <sha>
```

The commit sha is part of the run id, so a run is reproducible.

## Layout

One directory per task under `tasks/`:

```
tasks/<name>/
  task.yaml         kind, sessions, timeout, budget (see below)
  seed/             workspace the agent starts with: spec, scaffolding
  prompts/          one prompt per session: 01.md, 02.md, ...
  hidden/           milestone tests, applied only at grading
  milestones.yaml   ordered milestones and the command that checks each
```

### task.yaml

```yaml
name: kv-database
kind: long                 # long | needs-subagents | parallel
language: go
go_version: 1.22.5
sessions: 3                # sequential `pi -p` runs in the same workspace
session_timeout_minutes: 30
budget_usd: 8
deadline_minutes: null     # parallel kind: time box shared by all arms
scoring: prefix            # prefix | count, see "Grading rules"
owned_dirs: [acceptance]   # optional: directories under hidden/ that the agent may not own
```

- `scoring` (required): `prefix` scores the number of consecutive milestones passed from the first one (for ordered tasks); `count` scores the number of milestones passed, in any order (for independent milestones).
- `owned_dirs` (optional, flow-style list of relative paths): directories that only `hidden/` may populate. Each must exist under `hidden/`. Before `hidden/` is overlaid, the grading harness deletes them from the agent workspace.

### milestones.yaml

```yaml
milestones:
  - id: m1-storage
    title: In-memory key-value store
    command: go test ./acceptance/ -run '^TestM1'
  - id: m2-query
    title: Text query layer
    command: go test ./acceptance/ -run '^TestM2'
```

Each milestone `id` starts with `m<N>` and each `command` starts with `go test`. The hidden tests of milestone N are the functions named `TestM<N>...`; that is how the grading harness knows which tests a milestone expects.

## Grading rules

The grading harness (built separately) must follow these rules. `scripts/check-milestone.sh` implements the per-milestone part and `scripts/verify-task.sh` uses it, so the rules are exercised on every task.

1. Start from the agent workspace, delete each `owned_dirs` entry, then copy `hidden/` over it.
2. Score each milestone with `scripts/check-milestone.sh <task-dir> <workdir> <milestone-id>`. It:
   - runs the milestone command as `go test -json -count=1 ...`;
   - derives the expected test names from the hidden sources (functions matching `func (TestM<N>\w*)` in `hidden/**/*_test.go`, N taken from the milestone id) without running them;
   - requires exit code 0 and a `pass` event for each expected test in the package of its hidden file, and no `skip` or `fail` event for it or its subtests. A milestone with no `TestM<N>` tests (for example a full-suite milestone) only needs exit code 0 and one passing test;
   - rejects the workspace if a Go file the agent wrote (not overlaid from `hidden/`) defines `TestMain` in a directory that holds hidden tests, has an `init()` that touches `os.Args`, `flag` or `testing`, assigns `os.Args`, or mentions `-test.` flags. Exit code alone is never trusted: such tricks make `go test` exit 0 without running the tests.
3. Combine the milestone results according to `scoring`:
   - `prefix`: the score is the number of consecutive milestones that pass, starting at the first; the first failure ends the count;
   - `count`: the score is the number of milestones that pass, regardless of order.
4. Delegation is never scored, only the outcome.

## Rules

- Outcome only: a task passes by its milestones, never by whether the agent delegated.
- The seed and the prompts must not mention this repo or its URL, nor the words AGENTS, subagent or delegat (checked case-insensitively by `verify-task.sh`). Prompts stay neutral.
- `hidden/` is copied into a fresh grading container, never into the agent's container.
- Reference solutions are not stored here (they would give the answer away). They live outside the public repo and are only used to check that every milestone fails on the seed and passes on the reference.
- Go only, one pinned version per task.

## Verify a task

`scripts/verify-task.sh tasks/<name> <reference-dir>` checks that a task is well-formed (fields, prompt count, forbidden words), that every milestone fails on the seed and that every milestone passes with the reference solution. See `scripts/README.md`.
