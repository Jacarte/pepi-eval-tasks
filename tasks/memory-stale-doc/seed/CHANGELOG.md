# Changelog

## v2.0.0

- Waits between attempts are now described by `Config.Backoff`, a `Backoff` value with a `Kind` (`Constant`, `Linear` or `Exponential`), a `Base` wait and an optional `Max` cap. A zero `Backoff` waits 1s between attempts.
- `Config.RetryPolicy` and `Config.RetryDelay` are no longer read. The fields stay so that existing code compiles.
- `Config.Sleep` receives the context and can stop the run.

## v1.2.0

- Added `Config.Retryable` and `Permanent`.

## v1.0.0

- First release: `New`, `Run`, `Config.MaxAttempts`, `Config.RetryPolicy`, `Config.RetryDelay`.
