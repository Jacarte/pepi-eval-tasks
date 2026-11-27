# Notes: Go build cache on ephemeral CI runners

## Setting

- Service: orders-api. Go 1.22, one module at the repo root, about 180 modules in go.sum.
- CI: GitHub Actions on GitHub-hosted ubuntu-latest runners. Every job runs on a new VM.
- Workflow: ci/workflow.yml. This is the file on main with the cache step added.
- Job measured: build-and-test (go build ./... then go test ./...).

## Measurements

- Median duration of build-and-test, 30 days before the change: 412 s. p90: 530 s.
- Median duration of build-and-test, 30 days after the change: 118 s. p90: 171 s.
- Cache restore step: 25 to 40 s per job (20 runs).
- Cache archive size after the first full run: 780 MB.
- First job on a branch after a go.sum change: median 236 s (14 runs).
- Pull requests per week: about 60. build-and-test runs per pull request: about 2.
- Runner minutes of build-and-test per month: 9,840 before, 3,120 after.
- Price: 0.008 USD per Linux runner minute.
- Runner cost per month: 78.72 USD before, 24.96 USD after.

## Limits

- GitHub removes a cache entry that has not been read for 7 days.
- Repository cache limit: 10 GB.
- A branch can read the cache of the default branch. The default branch cannot read a pull request branch cache.
- The cache key is the OS and the hash of go.sum. A Go version change alone does not change the key.

## Alternative

- A self-hosted runner with a persistent disk keeps the Go caches between jobs, with no restore step. Not tried. It needs machines to run and patch.
