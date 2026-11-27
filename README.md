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
  memory/           optional fixture.json: memories stored in the agent's memory service before it starts
  skills/           optional, skill-<name> tasks: the vendored skill folder (SKILL.md, scripts) that is the --global-skills source
```

### task.yaml

```yaml
name: kv-database
kind: long                 # long | needs-subagents | parallel | memory
language: go
go_version: 1.22.5
sessions: 3                # sequential `pi -p` runs in the same workspace
session_timeout_minutes: 30
budget_usd: 8
deadline_minutes: null     # parallel kind: time box shared by all arms
scoring: prefix            # prefix | count, see "Grading rules"
owned_dirs: [acceptance]   # optional: directories under hidden/ that the agent may not own
restore_seed_tests: false  # optional: true restores the seed's visible tests before grading
```

- `scoring` (required): `prefix` scores the number of consecutive milestones passed from the first one (for ordered tasks); `count` scores the number of milestones passed, in any order (for independent milestones).
- `owned_dirs` (optional, flow-style list of relative paths): directories that only `hidden/` may populate. Each must exist under `hidden/`. Before `hidden/` is overlaid, the grading harness deletes them from the agent workspace.
- `memory/fixture.json` (required for `kind: memory` with `sessions: 1`, optional otherwise: a `kind: memory` task with more sessions may start with empty memory and test what later sessions recall from earlier ones): `{"memories": [{"text": "..."}]}` with 1 to 20 non-empty texts. The harness stores them in the memory service of the run when memory is on and the fixture enabled; they are never copied into the workspace, and no seed or prompt file may contain one.
- `restore_seed_tests` (optional, `true` or `false`, default `false`): when `true`, before `hidden/` is overlaid the grading harness deletes every `*_test.go` in the agent workspace and copies the seed's `*_test.go` files back. Deleting, editing or adding visible tests then cannot make a regression milestone (for example a full-suite milestone with no `TestM<N>` tests) pass. When `false`, the agent's test files are left as they are, but a full-suite milestone still expects every test of the seed to pass, so a deleted or renamed seed test fails it.

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

Each milestone `id` starts with `m<N>` and each `command` starts with `go test`, or is a skill milestone (`skill_loaded:<name>` or `skill_not_loaded:<name>`, graded only in evals from the skills the parent loaded). The hidden tests of milestone N are the functions named `TestM<N>...`; that is how the grading harness knows which tests a milestone expects.

## Grading rules

The grading harness (built separately) must follow these rules. `scripts/prepare-workspace.sh` implements step 1, `scripts/check-milestone.sh` the per-milestone part, and `scripts/verify-task.sh` uses both, so the rules are exercised on every task.

1. Prepare the grading workspace with `scripts/prepare-workspace.sh <task-dir> <workdir>`. Starting from the agent workspace it:
   - deletes each `owned_dirs` entry;
   - if `restore_seed_tests: true`, deletes every `*_test.go` in the workspace (edited and agent-added ones alike) and copies the seed's `*_test.go` files back, so a regression milestone (M13-style) cannot be passed by deleting or editing visible tests;
   - copies `hidden/` over the workspace.
2. Score each milestone with `scripts/check-milestone.sh <task-dir> <workdir> <milestone-id>`. It:
   - runs the milestone command as `go test -json -count=1 ...`;
   - derives the expected test names from the hidden sources (functions matching `func (TestM<N>\w*)` in `hidden/**/*_test.go`, N taken from the milestone id) without running them;
   - requires exit code 0 and a `pass` event for each expected test in the package of its hidden file, and no `skip` or `fail` event for it or its subtests. A milestone with no `TestM<N>` tests and no `-run` filter (a full-suite milestone) expects every `func Test*` of the seed and of `hidden/` to have a pass event, so a package that exits before its tests run, or whose tests were deleted, fails it. Every pass event must follow a run event, and a package with expected tests that passes with no test event at all is rejected. With a `-run` filter and no `TestM<N>` tests it needs exit code 0 and one passing test;
   - with `restore_seed_tests: true`, rejects a workspace whose `*_test.go` files are not exactly the seed's plus `hidden/` (a workspace that skipped step 1);
   - rejects a workspace that holds a file or directory name with a control character (a newline splits a name in two in the scripts' lists and the cheat scan would never see the file);
   - rejects a workspace that holds anything but regular files and directories (a symlink, fifo, socket or device): a symlinked `zz_test.go` pointing at a text file would escape every `find` that selects regular files (the seed-test restore, the cheat scan) while `go` still compiles it. `prepare-workspace.sh` rejects it too (exit 1);
   - runs the agent's code (the `go test` command) as an unprivileged user (`setpriv --reuid=65534 --regid=65534 --clear-groups`) on a copy of the workspace, with its own `HOME`, `GOCACHE`, `GOPATH` and `TMPDIR`, so it cannot reach the checker's files (kept in a root-owned `0700` directory) or the helper binaries, whose hash is compared before the result is used. This needs root and `setpriv` (util-linux, present in `ubuntu:24.04`); without them the script fails closed with exit 2. `CHECK_MILESTONE_ALLOW_ROOT=1` switches the drop off and is meant **only for local self-tests on a machine without setpriv (macOS)**: the agent's code then runs with your privileges and can tamper with the checker;
   - rejects the workspace if a Go file the agent wrote (not overlaid from `hidden/`, not identical to the seed file) uses one of these exit-code tricks, found by parsing the Go source (`scripts/scancheat`):
     - it defines `TestMain` in a directory that holds seed tests or hidden tests (any package, not only the ones hidden tests target);
     - an `init()` body or a package-level `var` initializer, or a same-directory function called from one, assigns or appends to `os.Args`, or calls `flag.Set` / `flag.Parse` (also through `flag.CommandLine` or `flag.Lookup(...).Value`);
     - it contains a string literal that is a real go test flag (starting with `-test.` or `--test.` followed by a letter, such as `"-test.run=^$"`).

     Everything else is accepted: a literal such as `"test.recorded"`, `init()` functions in test files that build tables, and tests that set `os.Args` inside a test function. Exit code alone is never trusted: the rejected tricks make `go test` exit 0 without running the tests.
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
