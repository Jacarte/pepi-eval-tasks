# kvdb specification

`kvdb` is a small key-value database written in Go 1.22 using only the
standard library (no third-party modules). It has four layers:

| Package          | Purpose                                                    |
|------------------|------------------------------------------------------------|
| `kvdb/store`     | ordered key-value store, in-memory or durable (WAL + snapshot) |
| `kvdb/query`     | text query language, sessions and transactions             |
| `kvdb/server`    | TCP line-protocol server                                   |
| `kvdb/cmd/kvdb`  | command-line tool: `serve`, `get`, `set`, `del`            |

Every exported name, signature, error value, response string, file format and
wire format below is part of the contract and must be implemented exactly as
written. The packages in this repository already declare the full API as
compiling stubs; replace the stub bodies with real implementations. You may add
unexported code and extra files freely. Do not change exported signatures.

Work proceeds in six milestones, in order. Each one builds on the previous ones.

1. **M1** ordered in-memory store (`store.New` and the `Store` methods)
2. **M2** query language without transactions (`query`)
3. **M3** transactions (`BEGIN` / `COMMIT` / `ROLLBACK`)
4. **M4** durability (`store.Open`, WAL, recovery, snapshots/compaction)
5. **M5** TCP server (`server`)
6. **M6** command-line tool (`cmd/kvdb`)

Acceptance tests for milestone *N* are named `TestM<N>...` and live in the
package directory `acceptance/`. That directory is reserved: do not create it
or put files in it. Put your own tests next to the code they test and do not
start their names with `TestM1`...`TestM6`. The acceptance tests are run with
`go test ./... -run '^TestM<N>'` from the module root, so the whole module must
build.

---

## M1 — ordered store (`package store`)

```go
package store

var (
	ErrNotFound = errors.New("store: key not found")
	ErrEmptyKey = errors.New("store: empty key")
	ErrClosed   = errors.New("store: closed")
)

// KV is one key/value pair returned by a scan.
type KV struct {
	Key   string
	Value []byte
}

// Op is one write in a batch passed to Apply.
// Delete == true means delete Key (Value is ignored); otherwise put Key=Value.
type Op struct {
	Key    string
	Value  []byte
	Delete bool
}

type Store struct{ /* unexported */ }

func New() *Store

func (s *Store) Put(key string, value []byte) error
func (s *Store) Get(key string) ([]byte, error)
func (s *Store) Delete(key string) error
func (s *Store) Apply(ops []Op) error
func (s *Store) Scan(start, end string, limit int) ([]KV, error)
func (s *Store) ScanPrefix(prefix string, limit int) ([]KV, error)
func (s *Store) Len() int
func (s *Store) Compact() error
func (s *Store) Close() error
```

Semantics:

- Keys are arbitrary non-empty strings (any bytes). Values are arbitrary byte
  slices; a nil value is stored as an empty value.
- Keys are ordered by plain byte-wise comparison (Go `<` on strings).
- `Put` inserts or overwrites. `Get` returns the value or `ErrNotFound`.
- `Delete` removes an existing key; if the key is absent it returns
  `ErrNotFound` and changes nothing.
- `Put`, `Get`, `Delete` with an empty key return `ErrEmptyKey`.
- `Apply` applies a batch of ops in slice order, **atomically**: concurrent
  readers (including scans) see either none or all of the batch. If any op has
  an empty key, `Apply` returns `ErrEmptyKey` and applies nothing. Deleting an
  absent key inside a batch is not an error. An empty or nil batch is a no-op
  that returns nil.
- `Scan(start, end, limit)` returns pairs with `start <= key < end`, ascending
  by key. `start == ""` means "from the first key"; `end == ""` means "no upper
  bound". If `end != ""` and `end <= start` the result is empty. `limit <= 0`
  means unlimited; otherwise at most `limit` pairs (the first ones in order).
- `ScanPrefix(prefix, limit)` returns the pairs whose key starts with `prefix`,
  ascending, with the same `limit` rule. `prefix == ""` matches every key.
- Scans return a consistent view: the result reflects the store at a single
  point in time.
- An empty scan result is an empty or nil slice with a nil error.
- Values are copied on the way in and on the way out: mutating a slice passed
  to `Put`/`Apply` or returned by `Get`/`Scan`/`ScanPrefix` never changes the
  stored data.
- `Len` returns the number of keys.
- All methods are safe for concurrent use by multiple goroutines.
- After `Close`, `Put`, `Get`, `Delete`, `Apply`, `Scan`, `ScanPrefix` and
  `Compact` return `ErrClosed`. Calling `Close` again returns nil.
- For a store made by `New` (in memory), `Compact` does nothing and returns nil.

---

## M2 — query language (`package query`)

```go
package query

var ErrSyntax = errors.New("query: syntax error")

type Engine struct{ /* unexported */ }
type Session struct{ /* unexported */ }

func NewEngine(st *store.Store) *Engine
func (e *Engine) NewSession() *Session

func (s *Session) Exec(line string) string
func (s *Session) InTransaction() bool
func (s *Session) Close()

func Quote(b []byte) string
func Unquote(s string) ([]byte, error)
```

An `Engine` wraps one store and is safe for concurrent use. A `Session` is one
client's state (used by a single goroutine at a time); many sessions of one
engine may run concurrently. `Exec` takes one command line (without a trailing
newline) and returns the full response with no trailing newline. Multi-line
responses separate lines with `"\n"`.

### Tokens

- A line is split into tokens separated by one or more spaces (0x20) or tabs
  (0x09). Leading and trailing separators are ignored.
- A **bare** token is a run of bytes that are not separators. Bytes are taken
  literally (a backslash in a bare token is a plain backslash). A bare token
  must not contain `"`.
- A **quoted** token starts with `"` and ends at the next unescaped `"`. Inside,
  spaces and tabs are ordinary bytes and these escapes are recognised:

  | escape  | byte                         |
  |---------|------------------------------|
  | `\\`    | `\`                          |
  | `\"`    | `"`                          |
  | `\n`    | 0x0A                         |
  | `\r`    | 0x0D                         |
  | `\t`    | 0x09                         |
  | `\xHH`  | the byte 0xHH (two hex digits, upper or lower case) |

  Any other byte after `\` is an error. The closing `"` must be followed by a
  separator or the end of the line. `""` is the empty string.
- Any tokenizer error (unterminated quote, bad escape, `"` inside a bare token,
  text glued to a closing quote) yields the response `(error) ERR syntax error`.
  Tokenizing happens before anything else, so a syntax error anywhere in the
  line wins over every other error.

### Quoting values in responses

`Quote(b)` returns `"` + escaped bytes + `"`, where each byte is written as:

- `\` → `\\`, `"` → `\"`, 0x0A → `\n`, 0x0D → `\r`, 0x09 → `\t`;
- any other byte below 0x20, and every byte 0x7F or above → `\x` followed by
  two **lowercase** hex digits (so UTF-8 text is escaped byte by byte);
- every other byte (0x20–0x7E) as itself.

`Unquote(s)` is the inverse for exactly one quoted token: `s` must start and
end with `"` and contain a valid quoted token per the tokenizer rules (any
escape from the table, either hex case). Anything else returns `ErrSyntax`.
`Unquote(Quote(b))` returns `b` for every byte slice.

### Commands

The first token is the command name, matched case-insensitively (ASCII). The
`LIMIT` keyword is also case-insensitive. Keys and values are case-sensitive.

| Command                     | Response                                        |
|-----------------------------|-------------------------------------------------|
| `SET key value`             | `OK`                                            |
| `GET key`                   | `Quote(value)` if present, else `(nil)`         |
| `DEL key`                   | `(integer) 1` if the key existed and was deleted, `(integer) 0` if absent |
| `SCAN prefix`               | array of all pairs whose key starts with prefix |
| `SCAN prefix LIMIT n`       | array of at most n such pairs (the first n)     |
| `COUNT prefix`              | `(integer) N`, number of keys starting with prefix |
| `BEGIN`, `COMMIT`, `ROLLBACK` | see M3                                        |

An **array** response is the header line `(array) N` followed by N lines, one
per pair in ascending key order, each `Quote(key) + " " + Quote(value)`. Zero
results is the single line `(array) 0`. Example:

```
> SCAN user:
(array) 2
"user:1" "alice"
"user:2" "bob smith"
```

The prefix may be `""` (all keys).

### Errors

All errors are a single line starting with `(error) ERR `. They are checked in
this order:

1. syntax error (see Tokens): `(error) ERR syntax error`
2. no tokens (empty or blank line): `(error) ERR empty command`
3. unknown command: `(error) ERR unknown command 'NAME'`, where NAME is the
   first token exactly as typed (after unquoting), e.g. `(error) ERR unknown command 'fetch'`
4. wrong argument count: `(error) ERR wrong number of arguments for 'CMD'`,
   where CMD is the canonical upper-case command name. Argument counts (not
   counting the command): SET 2, GET 1, DEL 1, COUNT 1, SCAN 1 or 3,
   BEGIN/COMMIT/ROLLBACK 0.
5. empty key for SET, GET or DEL: `(error) ERR empty key`
6. SCAN with 3 arguments whose second argument is not `LIMIT`:
   `(error) ERR syntax error`
7. LIMIT value that is not a positive decimal integer made only of ASCII
   digits (no sign) that fits in an `int`: `(error) ERR invalid limit`
8. transaction errors (M3)
9. a store error: `(error) ERR ` followed by the error's `Error()` text.

---

## M3 — transactions

| Command    | Response                                                     |
|------------|--------------------------------------------------------------|
| `BEGIN`    | `OK`; error `(error) ERR transaction already in progress` if one is open (the open transaction is left unchanged) |
| `COMMIT`   | `OK`; error `(error) ERR no transaction in progress` if none; error `(error) ERR transaction conflict` on conflict |
| `ROLLBACK` | `OK` (discards the transaction); error `(error) ERR no transaction in progress` if none |

Rules:

- Transactions belong to a session. There is no nesting.
- Between `BEGIN` and `COMMIT`, `SET` and `DEL` are buffered in the session and
  are invisible to other sessions and to the store.
- Reads inside a transaction (`GET`, `SCAN`, `COUNT`, and the existence check
  of `DEL`) see the latest committed data **merged with the transaction's own
  buffered writes** (read-your-writes; buffered deletes hide keys). `LIMIT`
  applies after merging.
- The transaction's **write set** is every key named by a `SET` or `DEL` it
  executed (a `DEL` that returned `(integer) 0` still counts).
- A key is **modified** when a `SET`, or a `DEL` that removed it, runs outside
  a transaction in any session of the engine, or when a transaction whose
  write set contains it commits.
- `COMMIT` fails with `(error) ERR transaction conflict` if any key of the
  write set was modified after this transaction's `BEGIN`. Keys that were only
  read never cause a conflict. On conflict the whole transaction is discarded:
  nothing is applied and the session is no longer in a transaction.
- A successful `COMMIT` applies all buffered writes with a single
  `store.Apply` call, so the commit is atomic (and is one WAL record, see M4).
  The conflict check and the apply happen atomically with respect to other
  commits and non-transactional writes of the same engine.
- Errors inside a transaction (syntax, arity, ...) do not end the transaction.
- `InTransaction` reports whether the session has an open transaction.
- `Close` discards any open transaction. The session must not be used after.
- Conflict tracking only covers writes made through the engine; writes made
  directly on the `*store.Store` are not tracked.

---

## M4 — durability (`package store`)

```go
type SyncPolicy int

const (
	SyncAlways SyncPolicy = iota // fsync the WAL before every write returns
	SyncNever                    // never fsync the WAL; the OS flushes eventually
)

type Options struct {
	Sync SyncPolicy
	// CompactThreshold: when the WAL file grows beyond this many bytes after
	// a write, the store compacts automatically before the write returns.
	// 0 means DefaultCompactThreshold; a negative value disables
	// automatic compaction (Compact can still be called).
	CompactThreshold int64
}

const (
	WALFileName             = "wal.log"
	SnapshotFileName        = "snapshot.db"
	DefaultCompactThreshold = 4 << 20
)

func Open(dir string, opts Options) (*Store, error)
```

`Open` creates `dir` (and parents) if needed, recovers the state from the
files in it, and returns a durable store with the same semantics as M1. Files
in `dir`:

- `wal.log` — the write-ahead log.
- `snapshot.db` — the latest snapshot (may be absent).
- `snapshot.db.tmp` — a snapshot being written; if present at `Open` it is
  leftover from a crash and must be ignored (delete it).

### Record format (shared by `wal.log` and `snapshot.db`)

A file is a sequence of records. All integers are little-endian `uint32`.

```
record  := length:u32  crc:u32  payload[length]
payload := count:u32  op{count}
op      := kind:u8  keylen:u32  key[keylen]  ( valuelen:u32 value[valuelen] )?
```

- `crc` is the CRC-32 (IEEE polynomial, Go `hash/crc32.ChecksumIEEE`) of the
  payload bytes.
- `kind` 1 = put (followed by the value), 2 = delete (no value).
- A payload is valid only if it is consumed exactly by `count` ops, every
  `kind` is 1 or 2, and every key is non-empty. `count` may be 0.
- Records are applied like `Apply`: ops in order; a delete of an absent key
  is a no-op.

### Writing

- Every successful mutating call appends exactly one record to `wal.log`
  **before** it returns, written to the file with a write system call (no
  user-space buffering that survives the call):
  - `Put(k, v)` → one record with one put op.
  - `Delete(k)` of an existing key → one record with one delete op. A `Delete`
    that returns `ErrNotFound` writes nothing.
  - `Apply(ops)` (non-empty) → one record with all ops, in order. This is what
    makes a transaction commit atomic on disk.
  - Calls that fail validation (`ErrEmptyKey`, `ErrClosed`) write nothing.
- The order of records in the WAL is the order in which the writes took
  effect in memory, even with concurrent writers.
- With `SyncAlways` the WAL is fsynced before the call returns. With
  `SyncNever` it is not.
- If writing the record fails, the call returns the error and the in-memory
  state is unchanged.

### Recovery

1. Delete `snapshot.db.tmp` if present.
2. If `snapshot.db` exists, read it: it must be a sequence of complete, valid
   records (any grouping of ops into records); apply them in order. If it is
   truncated or any record is invalid, `Open` returns an error.
3. Read `wal.log` from the start and apply valid records in order. Stop at the
   first record that is incomplete (file ends inside the header or payload),
   has a CRC mismatch, or has an invalid payload. Everything from that point on
   is a torn or corrupt tail: it is discarded and `wal.log` is truncated to the
   end of the last valid record, so later appends follow valid data.
4. A record is applied all-or-nothing, so a torn last transaction leaves no
   trace.

### Snapshots and compaction

`Compact()` writes the complete current state to `snapshot.db.tmp` as records
containing only put ops (grouping is up to you; an empty store gives an empty
file), fsyncs it, renames it to `snapshot.db` (replacing any previous one),
and then truncates `wal.log` to zero length. After `Compact` returns, the WAL
is empty and the snapshot alone describes the state. Automatic compaction
(see `CompactThreshold`) does the same; it keeps `wal.log` no larger than the
threshold plus the size of one record. Compaction is atomic with respect to
other operations on the store.

`Close` flushes and closes the files. A store must never require `Close` to
have been called to recover data: a process may crash at any time, and every
write that returned nil must be recovered as long as the operating system
wrote the file data (with `SyncAlways`, even across power loss).

---

## M5 — TCP server (`package server`)

```go
package server

const MaxLineLength = 1 << 20

type Server struct{ /* unexported */ }

func New(e *query.Engine) *Server
func (s *Server) Start(addr string) error
func (s *Server) Addr() string
func (s *Server) Shutdown(ctx context.Context) error
```

- `Start` listens on TCP `addr` (e.g. `"127.0.0.1:0"`), starts serving in
  background goroutines and returns once the listener is bound. It returns an
  error if listening fails, or if the server was already started.
- `Addr` returns the bound address as `host:port` with the real port (e.g.
  `"127.0.0.1:53124"`), or `""` before a successful `Start`.
- Each connection gets its own `query.Session` (so transactions are
  per-connection) and is served concurrently with all other connections.

### Wire protocol

- The client sends command lines terminated by `"\n"`; a `"\r"` right before
  the `"\n"` is removed. Lines of up to `MaxLineLength` bytes (excluding the
  terminator) must be accepted; longer lines may close the connection.
- For every line the server writes the `Exec` response followed by `"\n"`
  (for arrays, every line of the response ends with `"\n"`). Responses come in
  request order. Clients may pipeline: send many lines before reading.
- The line `QUIT` (case-insensitive, no arguments) is handled by the server:
  it replies `OK` and closes the connection.
- Bytes after the last `"\n"` when the client closes are discarded.
- When a connection ends for any reason its session is closed (an open
  transaction is discarded).

### Graceful shutdown

`Shutdown(ctx)`:

1. closes the listener (new connections are refused);
2. lets every connection finish the command it is currently executing and
   write its response — a command is either executed and answered, or not
   executed at all; then closes every connection, including idle ones (idle
   connections must not delay shutdown);
3. discards open transactions (their sessions are closed);
4. returns nil once all connection goroutines have exited, or, if `ctx` ends
   first, force-closes everything and returns `ctx.Err()`.

`Shutdown` does not close the store. Calling `Shutdown` again returns nil.

---

## M6 — command-line tool (`cmd/kvdb`)

```
kvdb serve --dir DIR [--addr ADDR] [--sync always|never]
kvdb get   [--addr ADDR] KEY
kvdb set   [--addr ADDR] KEY VALUE
kvdb del   [--addr ADDR] KEY
```

- Flags come after the subcommand and before positional arguments; both
  `-addr` and `--addr` spellings work (Go `flag` package). `ADDR` defaults to
  `127.0.0.1:7379`.
- `serve` opens the durable store in `DIR` (`--dir` is required) with the
  chosen sync policy (default `always`) and the default compaction threshold,
  starts the server, and then prints exactly one line to stdout:
  `listening on HOST:PORT` (the real bound address, i.e. `Server.Addr()`).
  On SIGINT or SIGTERM it shuts the server down gracefully, closes the store
  and exits 0.
- `get`, `set`, `del` connect to the server at `ADDR`, send one command, read
  the response and exit. Keys and values are taken verbatim from the
  arguments (any bytes, including spaces, quotes, backslashes and newlines),
  so the client must send them as quoted tokens.
  - `set`: on `OK` prints `OK` and a newline.
  - `get`: if found, prints the raw value bytes (unquoted) followed by one
    newline; if `(nil)`, prints nothing to stdout.
  - `del`: on `(integer) 1` prints `deleted` and a newline; on `(integer) 0`
    prints nothing to stdout.
- Diagnostics go to stderr, prefixed with `kvdb: `. Nothing but the outputs
  above is ever written to stdout.

Exit codes:

| code | meaning |
|------|---------|
| 0 | success |
| 1 | key not found (`get` got `(nil)`, `del` got `(integer) 0`) |
| 2 | usage error: no or unknown subcommand, unknown flag, wrong number of positional arguments, missing `--dir`, invalid `--sync` value |
| 3 | I/O error: cannot connect, connection lost, unparseable response, cannot open the store, cannot listen |
| 4 | the server answered with an `(error) ...` response (printed to stderr) |
