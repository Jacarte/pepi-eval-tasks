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
```

### milestones.yaml

```yaml
milestones:
  - id: m1-storage
    title: In-memory key-value store
    command: go test ./... -run '^TestM1'
  - id: m2-query
    title: Text query layer
    command: go test ./... -run '^TestM2'
```

The score is the number of milestones that pass, in order.

## Rules

- Outcome only: a task passes by its milestones, never by whether the agent delegated.
- `hidden/` is copied into a fresh grading container, never into the agent's container.
- The seed must not mention this repo or its URL.
- Reference solutions are not stored here (they would give the answer away). They live outside the public repo and are only used to check that every milestone fails on the seed and passes on the reference.
- Go only, one pinned version per task.

## Verify a task

`scripts/verify-task.sh tasks/<name> <reference-dir>` checks that a task is well-formed, that every milestone fails on the seed and that every milestone passes with the reference solution. See `scripts/README.md`.
