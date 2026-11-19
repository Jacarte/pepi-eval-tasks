# jobq

A small job runner. It runs a function, retries it when it fails, and waits between attempts.

## Running a job

```go
r := jobq.New(jobq.Config{Name: "export", MaxAttempts: 4})
err := r.Run(ctx, func(ctx context.Context) error {
	return export(ctx)
})
```

`MaxAttempts` counts every try, the first one included. When it is 0, the runner makes 3 attempts. `Run` returns nil as soon as one attempt succeeds. After the last failed attempt it returns an error that wraps the last failure.

## Configuring retries

Pick a schedule with `RetryPolicy` and set the first wait with `RetryDelay`:

```go
cfg := jobq.Config{
	Name:        "export",
	MaxAttempts: 4,
	RetryPolicy: "linear", // "constant", "linear" or "exponential"
	RetryDelay:  250 * time.Millisecond,
}
```

With `"linear"` the waits are 250ms, 500ms, 750ms. With `"exponential"` they double each time: 250ms, 500ms, 1s. With `"constant"` every wait is 250ms.

`go run ./cmd/demo` runs a flaky job with this configuration and prints each wait.

## Permanent failures

An error that wraps `jobq.ErrPermanent` is never retried. Mark one with `jobq.Permanent(err)`. To use your own rule, set `Config.Retryable`; it receives the error and reports whether another attempt is allowed.

## Testing

`Config.Sleep` replaces the real wait. It receives the context and the wait, and returns an error to stop the run. Tests set it to record the waits instead of sleeping.
