# Code conventions (reviewed on every change)

1. Error codes. Every error returned to API clients carries a code of the
   form `ERR-PAY-<4 digits>`. A code names exactly one failure reason; never
   reuse a code for a different reason. New reasons take the next free number
   and are registered in both places, with the same value:
   - a row in `docs/error-codes.md` (code, HTTP status, meaning), and
   - a constant in `internal/apperr/codes.go`.

2. Logging. Code under `internal/` logs only through `internal/xlog`, with a
   logger obtained from `xlog.For("<component>")`. The component is the
   lowercase name of the handler group (`payments`, `customers`). The stdlib
   `log`/`slog` packages and `fmt.Print*` are not used there.

3. Route manifest. Every exported handler of `internal/api` is listed in
   `internal/api/routes_manifest.go` with its method and path. The manifest
   feeds `GET /debug/routes` and the on-call route inventory.
