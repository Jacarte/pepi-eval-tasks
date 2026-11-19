# Architecture notes

The service keeps everything in memory (`internal/store`). Handlers live in
`internal/api` as methods of `Server`; each handler group has its own file.
Money is always integer cents. IDs are `<prefix>_<n>` strings.

Payment lifecycle: `authorized` -> `captured` or `voided`. Nothing leaves
`captured` or `voided`.
