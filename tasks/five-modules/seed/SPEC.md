# fivemods — specification

This module (`module fivemods`, Go 1.22, standard library only) contains five
**independent** packages. Each package is graded on its own, so a finished
package counts even if another one is unfinished. Packages must not import each
other.

| Package | Directory | Summary |
|---|---|---|
| `semver` | `semver/` | Semantic Versioning 2.0.0: parse, compare, increment, ranges |
| `glob` | `glob/` | gitignore-style path matching |
| `csvparse` | `csvparse/` | streaming RFC 4180 CSV reader and writer |
| `inireader` | `inireader/` | INI/TOML-lite configuration parser with typed getters |
| `lrucache` | `lrucache/` | generic LRU cache with TTL, weights, callbacks, stats |

Every exported identifier listed below already exists as a compiling stub.
Keep the exact names and signatures; you may add unexported helpers and files.
`go build ./...` and `go vet ./...` must keep passing.

Anything not specified here is not tested.

---

## 1. `semver`

```go
type Version struct {
    Major, Minor, Patch uint64
    Prerelease          []string // dot-separated pre-release identifiers; nil when absent
    Build               []string // dot-separated build identifiers; nil when absent
}

func Parse(s string) (Version, error)
func MustParse(s string) Version            // panics if Parse fails
func (v Version) String() string
func Compare(a, b Version) int              // -1, 0 or +1
func (v Version) IncMajor() Version
func (v Version) IncMinor() Version
func (v Version) IncPatch() Version
func (v Version) IncPrerelease(id string) Version
func Sort(vs []Version)

type Range struct{ /* unexported */ }
func ParseRange(s string) (Range, error)
func (r Range) Contains(v Version) bool
func MaxSatisfying(vs []Version, r Range) (Version, bool)
```

### Parse

Accepts exactly `MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]` (strict SemVer 2.0.0):

- No leading `v`, no surrounding or inner whitespace, nothing else.
- `MAJOR`, `MINOR`, `PATCH`: non-empty ASCII decimal digits, no leading zeros
  (`0` is fine, `01` is an error), value must fit in `uint64` (otherwise error).
- `PRERELEASE` (after the first `-`): one or more identifiers separated by `.`.
  Each identifier is non-empty and made of `[0-9A-Za-z-]`. An identifier made
  only of digits is *numeric* and must not have leading zeros (`0` is fine,
  `01` is an error). Numeric identifiers may be arbitrarily long.
- `BUILD` (after the first `+`): one or more identifiers separated by `.`, each
  non-empty and made of `[0-9A-Za-z-]`; leading zeros are allowed.
- Any violation returns a non-nil error and the zero `Version`.

`String()` returns the canonical text: `M.m.p`, plus `-` and the pre-release
identifiers joined by `.` if any, plus `+` and the build identifiers joined by
`.` if any. `String(Parse(s)) == s` for every valid `s`.

### Compare (precedence, semver.org §11)

1. Compare Major, Minor, Patch numerically.
2. A version without pre-release has higher precedence than the same
   Major.Minor.Patch with a pre-release (`1.0.0-alpha < 1.0.0`).
3. Pre-release identifiers are compared left to right: two numeric identifiers
   compare numerically; two alphanumeric identifiers compare lexically in ASCII
   byte order; a numeric identifier is lower than an alphanumeric one. If all
   compared identifiers are equal, the version with more identifiers is higher.
4. Build metadata is ignored: `Compare(1.0.0+a, 1.0.0+b) == 0`.

### Increments

All increments return a new `Version` with **no** build metadata (`Build` is
nil) and never modify the receiver (including its slices).

- `IncMajor`: if `v` has a pre-release and Minor == 0 and Patch == 0, the result
  is `Major.0.0` (pre-release dropped). Otherwise `(Major+1).0.0`.
- `IncMinor`: if `v` has a pre-release and Patch == 0, the result is
  `Major.Minor.0`. Otherwise `Major.(Minor+1).0`.
- `IncPatch`: if `v` has a pre-release, the result is `Major.Minor.Patch`
  (pre-release dropped). Otherwise `Major.Minor.(Patch+1)`.
- `IncPrerelease(id)`:
  1. If `v` has no pre-release: Patch+1 and pre-release `[id, "0"]`
     (or `["0"]` when `id == ""`). `1.2.3` → `1.2.4-0`; with `id="rc"` → `1.2.4-rc.0`.
  2. Else if `id != ""` and the first pre-release identifier is not `id`:
     same Major.Minor.Patch, pre-release `[id, "0"]`. `1.2.3-alpha.5` with `"beta"` → `1.2.3-beta.0`.
  3. Else: increment the **rightmost** numeric identifier by one (decimal,
     arbitrary length: `9` → `10`); if there is no numeric identifier, append
     `"0"`. `1.0.0-rc.1.x` → `1.0.0-rc.2.x`; `1.0.0-alpha` → `1.0.0-alpha.0`.

### Sort

Sorts ascending by precedence. The sort is **stable**: versions with equal
precedence (e.g. differing only in build metadata) keep their input order.

### Ranges

Grammar:

```
range       = set *( "||" set )
set         = term *( whitespace term )
term        = comparator | hyphen
hyphen      = version whitespace "-" whitespace version
comparator  = "*" | [ op ] version          ; no whitespace between op and version
op          = ">=" | "<=" | ">" | "<" | "=" | "^" | "~"
version     = a full version accepted by Parse (pre-release and build allowed)
```

- Whitespace is one or more spaces or tabs. Leading/trailing whitespace of the
  whole range and around `||` is allowed.
- Errors: empty or all-whitespace input; an empty set (e.g. `"1.2.3 ||"`,
  `"|| 1.2.3"`); an operator without a version or separated from it by
  whitespace (`">= 1.2.3"`); a hyphen whose either side is missing or is not a
  bare version (`">=1.0.0 - 2.0.0"`); any token that is not a valid term
  (`"1.2"`, `"x"`, `"~>1.2.3"`).
- Meaning of each term for a version `v` (comparisons use `Compare`):
  - `*`: any version.
  - `V` or `=V`: `v == V`.
  - `>V`, `>=V`, `<V`, `<=V`: as written.
  - `~M.m.p[-pre]`: `>=M.m.p[-pre]` and `<M.(m+1).0`.
  - `^M.m.p[-pre]`: `>=M.m.p[-pre]` and `<` the upper bound
    `(M+1).0.0` if M > 0; `0.(m+1).0` if M == 0 and m > 0; `0.0.(p+1)` if M == 0 and m == 0.
  - `A - B`: `>=A` and `<=B`.
- A set matches `v` when every term matches **and** the pre-release rule holds:
  if `v` has a pre-release, the set matches only if at least one version
  **written** in the set (the operand of a comparator, or either side of a
  hyphen; never a computed upper bound, and `*` has none) has a pre-release
  and the same Major.Minor.Patch as `v`. Versions without a pre-release are
  not affected by this rule.
  - `^1.2.3-beta.1` contains `1.2.3-beta.2` and `1.2.5`, but not `1.2.4-alpha`.
  - `<2.0.0` does not contain `2.0.0-alpha`; `*` does not contain `1.0.0-rc.1`.
- `Contains` is true when any set matches.

`MaxSatisfying` returns the version with the highest precedence among those in
`vs` that `r` contains, and true; if several have that precedence, the earliest
in `vs`. Returns the zero Version and false if none match.

---

## 2. `glob`

```go
func Match(patterns []string, path string, isDir bool) bool
```

`patterns` are gitignore lines in file order. `path` is relative, uses `/`,
has no leading or trailing `/`, no empty, `.` or `..` components, and is
non-empty. `isDir` says whether `path` itself is a directory. `Match` returns
true when the path is **ignored**.

### Line preprocessing (in this order)

1. Trailing spaces (U+0020) are removed, except a space preceded by an odd
   number of backslashes (`foo\ ` keeps the escaped space and matches the
   name `foo `; the backslash still escapes it).
2. A line that is now empty is ignored. A line starting with `#` is a comment
   and ignored (`\#foo` is a pattern matching the literal name `#foo`).
3. A leading `!` makes the rule a *negation* and is removed (`\!foo` is not a
   negation; it matches the literal name `!foo`).
4. A trailing `/` makes the rule *directory-only* and is removed.
5. If the remaining pattern contains a `/` (anywhere), the rule is *anchored*:
   one leading `/` is removed and the pattern is matched against the whole
   path. Otherwise the rule is *unanchored* and is matched against the last
   component of the path only (so it matches at any depth).
6. If the pattern is now empty (e.g. `!`, `/`), the line is ignored.
7. A pattern ending in an unescaped (odd-count) backslash never matches.

### Pattern syntax

The pattern is split on `/` into segments; each path component is matched
against one segment. Within a segment:

- `*` matches any sequence of characters (including empty), never `/`.
  Consecutive `*` inside a segment behave like one `*` (e.g. `a**b` == `a*b`).
- `?` matches exactly one character (rune), never `/`.
- `[...]` matches one character from the class. `!` or `^` right after `[`
  negates the class. A `]` immediately after `[` (or after the negation mark)
  is a literal member. `a-z` is an inclusive range of code points; a `-` at
  the start or end of the class is literal. `\x` inside a class is the literal
  `x`. A `[` with no closing `]` is a literal `[` character.
- `\x` outside a class matches the literal character `x`.
- Every other character matches itself; matching is case-sensitive.

A segment that is exactly `**` in an anchored pattern is special:

- `**` as the **last** segment matches one or more remaining components
  (`abc/**` matches `abc/x` and `abc/x/y`, not `abc`).
- `**` anywhere else matches zero or more components (`**/foo` matches `foo`
  and `a/b/foo`; `a/**/b` matches `a/b`, `a/x/b`, `a/x/y/b`).

A rule *matches* a path when the pattern matches it and, for a
directory-only rule, the path is a directory.

### Evaluation

To decide one path with a given `isDir`: the **last** rule (in list order)
that matches decides — non-negated means ignored, negated means not ignored.
If no rule matches, the path is not ignored.

`Match(patterns, path, isDir)`:

1. For each proper ancestor directory of `path`, from the top down (for
   `a/b/c`: `a`, then `a/b`), decide it as a directory (`isDir = true`). If any
   ancestor is ignored, return true — a file inside an ignored directory can
   never be re-included by a negation.
2. Otherwise return the decision for `path` itself with the given `isDir`.

Examples: `["*.log", "!keep.log"]` ignores `a/x.log` but not `a/keep.log`.
`["build/", "!build/keep.txt"]` ignores `build/keep.txt` (its parent is
ignored). `["/*", "!/src"]` ignores `README` and `docs/a.md` but not
`src/main.go`.

---

## 3. `csvparse`

```go
var (
    ErrBareQuote      = errors.New("bare quote in unquoted field")
    ErrQuote          = errors.New("extraneous or missing quote in quoted field")
    ErrFieldCount     = errors.New("wrong number of fields")
    ErrInvalidOptions = errors.New("invalid options")
)

type ParseError struct {
    Line   int   // 1-based line
    Column int   // 1-based column, counted in runes
    Err    error // ErrBareQuote, ErrQuote or ErrFieldCount
}
func (e *ParseError) Error() string // fmt.Sprintf("line %d, column %d: %v", Line, Column, Err)
func (e *ParseError) Unwrap() error // returns Err

type RaggedPolicy int
const (
    RaggedError RaggedPolicy = iota // default
    RaggedAllow
    RaggedPad
)

type Options struct {
    Delimiter  rune         // 0 means ','
    Quote      rune         // 0 means '"'
    Comment    rune         // 0 disables comment lines
    LazyQuotes bool
    Ragged     RaggedPolicy
}

type Reader struct{ /* unexported */ }
func NewReader(r io.Reader, opts Options) *Reader
func (r *Reader) Read() ([]string, error)
func (r *Reader) ReadAll() ([][]string, error)

type WriterOptions struct {
    Delimiter rune // 0 means ','
    Quote     rune // 0 means '"'
    UseCRLF   bool // terminate records with "\r\n" instead of "\n"
    QuoteAll  bool // quote every field
}

type Writer struct{ /* unexported */ }
func NewWriter(w io.Writer, opts WriterOptions) *Writer
func (w *Writer) Write(record []string) error
func (w *Writer) Flush() error
func (w *Writer) WriteAll(records [][]string) error
```

### Options validation

After applying defaults, `Delimiter` and `Quote` (and `Comment` if non-zero)
must be pairwise different, valid runes, and none of them may be `'\r'`,
`'\n'` or `utf8.RuneError`. Otherwise every `Read`/`ReadAll` returns an error
for which `errors.Is(err, ErrInvalidOptions)` is true (likewise `Write`,
`WriteAll` and `Flush` for invalid `WriterOptions`).

### Reading

- Input is UTF-8. A UTF-8 BOM (U+FEFF) at the very start of the input is
  dropped and does not count as a column. A BOM anywhere else is data.
- A record ends at `"\n"` or `"\r\n"` (outside quotes), or at end of input.
  A lone `"\r"` not followed by `"\n"` is ordinary data. A final line break
  does not create an extra record.
- Physical lines with no characters at all (a line consisting only of its
  line break, `"\n"` or `"\r\n"`, including at the start of the input) are
  skipped. A line containing only whitespace or only
  delimiters is a record (`","` → `["", ""]`).
- If `Comment != 0`, a line whose first character is `Comment` is skipped
  entirely (only when the line starts a new record, never inside a quoted
  field).
- Fields are split on `Delimiter`. Whitespace is never trimmed.
- A field is *quoted* only if its first character is `Quote`. Inside a quoted
  field, two consecutive `Quote` characters mean one literal `Quote`;
  delimiters and line breaks are data; `"\r\n"` inside a quoted field becomes
  `"\n"`. The closing `Quote` must be followed by `Delimiter`, a line break, or
  end of input.
- Read returns a new slice on each call. At end of input it returns
  `nil, io.EOF` (also on later calls).
- **Streaming:** `Read` must return a record as soon as its terminating line
  break (or end of input) has been read; it must not wait for more input.
  Never read the whole input up front.

Errors (`*ParseError`; `Line` counts `\n` line breaks from 1, including those
inside quoted fields, skipped lines and comment lines; `Column` is the rune
position within that line, 1-based):

| Case (when `LazyQuotes` is false) | `Err` | Position |
|---|---|---|
| `Quote` appears in an unquoted field | `ErrBareQuote` | that quote character |
| closing quote followed by anything other than delimiter, line break or end of input | `ErrQuote` | the character after the closing quote |
| end of input inside a quoted field | `ErrQuote` | the opening quote of that field |
| field count mismatch (see Ragged) | `ErrFieldCount` | line where the record starts, column 1 |

For `ErrBareQuote`/`ErrQuote`, `Read` returns a nil record.

With `LazyQuotes` true: a `Quote` in an unquoted field is ordinary data; in a
quoted field, a `Quote` followed by something other than `Quote`, delimiter or
line break is kept as a literal `Quote` and the field continues; end of input
inside a quoted field ends the field and the record without error.

Ragged rows: the *expected* field count is the field count of the first record
returned. For a later record with a different count:

- `RaggedError`: `Read` returns the record **and** a `*ParseError` with
  `ErrFieldCount`; the next `Read` continues with the following record.
- `RaggedAllow`: the record is returned unchanged.
- `RaggedPad`: short records are padded with `""`, long records are
  truncated, to exactly the expected count.

`ReadAll` reads until end of input and returns all records with a nil error
(no records → nil or empty slice, nil error). On the first error of any kind
it stops and returns the records read before the failing record, plus the
error.

### Writing

- Fields are joined with `Delimiter`; each record ends with `"\n"` (or
  `"\r\n"` with `UseCRLF`). Field contents are written verbatim (embedded
  `"\n"` is not converted).
- A field is enclosed in `Quote` if `QuoteAll` is set, or it contains
  `Delimiter`, `Quote`, `'\r'` or `'\n'`, or it starts with a space or tab, or
  it is the only field of the record and is empty (so `[""]` is written as
  `""` plus the terminator). Otherwise it is written bare.
- Inside a quoted field every `Quote` is doubled.
- `Write` may buffer; `Flush` writes buffered data to the underlying writer
  and returns any write error. `WriteAll` writes every record, then flushes.

---

## 4. `inireader`

```go
type Kind int
const (
    KindString Kind = iota + 1
    KindInt
    KindFloat
    KindBool
    KindArray
)

type Value struct {
    Kind  Kind
    Str   string  // KindString
    Int   int64   // KindInt
    Float float64 // KindFloat
    Bool  bool    // KindBool
    Array []Value // KindArray (non-nil, possibly empty)
}

type ParseError struct {
    Line int    // 1-based physical line number
    Msg  string // exact messages listed below
}
func (e *ParseError) Error() string // fmt.Sprintf("line %d: %s", Line, Msg)

type File struct{ /* unexported */ }
func Parse(r io.Reader) (*File, error)
func ParseString(s string) (*File, error)

func (f *File) Sections() []string
func (f *File) Subsections(parent string) []string
func (f *File) Keys(section string) []string
func (f *File) Lookup(section, key string) (Value, bool)
func (f *File) GetString(section, key, def string) string
func (f *File) GetInt(section, key string, def int64) int64
func (f *File) GetFloat(section, key string, def float64) float64
func (f *File) GetBool(section, key string, def bool) bool
func (f *File) GetStrings(section, key string, def []string) []string
func (f *File) GetInts(section, key string, def []int64) []int64
```

On any syntax error, `Parse`/`ParseString` return a nil `*File` and a
`*ParseError` for the **first** offending line.

### Lines

Input is split on `"\n"`; a trailing `"\r"` on a line is removed. Each line is
then trimmed of spaces and tabs. "Whitespace" below means spaces and tabs.

- Empty lines are ignored.
- A line whose first character is `#` or `;` is a comment.
- A line starting with `[` is a **section header** (see below).
- Any other line is a **key/value** line.

Keys before the first header belong to the root section, named `""`.

### Section headers

`[name]` optionally followed by whitespace and a comment (`#` or `;` to end of
line). `name` (trimmed of whitespace inside the brackets) is one or more parts
joined by `.`; each part is non-empty and made of `[A-Za-z0-9_-]`. `[a.b]` is
the section named `"a.b"`. Errors:

- missing `]`, empty/invalid name (`[]`, `[a..b]`, `[a b]`), or anything other
  than whitespace/comment after `]` → `invalid section header`
- a name declared earlier → `duplicate section "NAME"`

### Key/value lines

`key = value`, split at the **first** `=`. Checked in this order:

1. no `=` on the line → `missing '='`
2. key (trimmed) empty or not made of `[A-Za-z0-9_-]` → `invalid key`
3. key already defined in the current section → `duplicate key "KEY"`
4. value text (trimmed) empty or starting with `#`/`;` → `missing value`
5. value parsing errors below.

Keys and section names are case-sensitive.

### Values

The value text is classified by its first character:

- `"` — **basic string**. Ends at the next unescaped `"`. Escapes: `\"`, `\\`,
  `\n`, `\t`, `\r`, `\uXXXX` (exactly four hex digits, either case). Errors:
  no closing quote (including a backslash at end of line) →
  `unterminated string`; `\u` not followed by four hex digits →
  `invalid unicode escape`; any other escape `\c` →
  `invalid escape sequence \c` (that is: the text `invalid escape sequence `
  followed by a backslash and the character, e.g. `invalid escape sequence \q`).
- `'` — **literal string**: everything up to the next `'`, no escapes.
  No closing quote → `unterminated string`.
- `[` — **array** (single line): elements separated by `,`, optional
  whitespace anywhere between tokens, an optional trailing comma before `]`;
  `[]` is an empty array. Elements are basic/literal strings, or bare tokens
  (a run of characters other than `,`, `]` and whitespace) that are a bool, int
  or float as defined below. Errors (in the order encountered, scanning left
  to right):
  - element starting with `[` → `nested arrays are not supported`
  - an empty element (`[,]`, `[1,,2]`) or a bare token that is not bool/int/float → `invalid array element`
  - after an element, something other than `,` or `]` → `expected ',' or ']' in array`
  - end of line before the closing `]` → `unterminated array`
  - an element whose Kind differs from the first element's → `mixed types in array`
  - string, int and float errors from elements are reported as for scalars.
- anything else — **bare value**. An inline comment starts at a `#` or `;`
  that is preceded by whitespace; it and the whitespace are removed
  (`a#b` stays `a#b`). The remaining text is then:
  - `true` / `false` (lowercase only) → bool;
  - an int: `[+-]?[0-9]+` (decimal; leading zeros allowed) or `0x[0-9a-fA-F]+`
    (hex, lowercase `0x`, no sign). Does not fit int64 → `integer out of range`;
  - a float: `[+-]?[0-9]+` followed by a fraction `.[0-9]+` and/or an exponent
    `[eE][+-]?[0-9]+` (at least one of the two). Overflows to ±Inf →
    `float out of range`;
  - otherwise a string with that exact text (e.g. `hello world`, `1.2.3`,
    `True`, `1.`, `0x`, `-0x10`).

After a quoted string or an array, the rest of the line must be empty,
whitespace, or a comment (`#`/`;`, whitespace before it optional) — otherwise
`trailing characters after value`.

### Accessors

- `Sections()` — declared section names in order of appearance (the root
  `""` is never included). Returns nil/empty if none.
- `Subsections(parent)` — distinct immediate child part names of `parent`
  among declared section names, in order of first appearance. A declared
  `[a.b.c]` makes `b` a child of `a` even if `[a.b]` is not declared.
  `Subsections("")` returns the distinct first parts of all section names.
- `Keys(section)` — keys in definition order; nil/empty for an unknown or
  empty section.
- `Lookup` — the raw value and true, or zero Value and false.
- `GetString`/`GetInt`/`GetBool` — the value if the key exists and has
  exactly that Kind; otherwise `def`.
- `GetFloat` — the value for KindFloat, `float64(v.Int)` for KindInt,
  otherwise `def`.
- `GetStrings`/`GetInts` — for a KindArray whose elements are all strings
  (resp. ints) the element values as a new non-nil slice (an empty array gives
  an empty non-nil slice); otherwise `def`.

---

## 5. `lrucache`

```go
type Clock interface{ Now() time.Time }

type EvictionReason int
const (
    ReasonCapacity EvictionReason = iota + 1 // removed to satisfy MaxEntries/MaxBytes
    ReasonExpired                            // TTL elapsed
    ReasonDeleted                            // Delete or Purge
    ReasonReplaced                           // overwritten by Set
)

type Options[K comparable, V any] struct {
    MaxEntries int                        // <= 0: unlimited
    MaxBytes   int64                      // <= 0: unlimited
    Weigher    func(key K, value V) int64 // nil: every entry weighs 1; negative results count as 0
    TTL        time.Duration              // default TTL for Set; <= 0: no expiry
    Clock      Clock                      // nil: the system clock
    OnEvict    func(key K, value V, reason EvictionReason)
}

type Stats struct {
    Hits, Misses, Evictions, Expirations uint64
}

type Cache[K comparable, V any] struct{ /* unexported */ }
func New[K comparable, V any](opts Options[K, V]) *Cache[K, V]
func (c *Cache[K, V]) Set(key K, value V)
func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration)
func (c *Cache[K, V]) Get(key K) (V, bool)
func (c *Cache[K, V]) Peek(key K) (V, bool)
func (c *Cache[K, V]) Delete(key K) bool
func (c *Cache[K, V]) Len() int
func (c *Cache[K, V]) Bytes() int64
func (c *Cache[K, V]) Keys() []K
func (c *Cache[K, V]) RemoveExpired() int
func (c *Cache[K, V]) Purge()
func (c *Cache[K, V]) Stats() Stats
```

### Time and expiry

All time comes from `Options.Clock` (never call `time.Now` directly when a
Clock is given). An entry stored at time `t` with TTL `d > 0` expires at
`t + d` and is **expired when `now >= t + d`**. `Set` uses `Options.TTL`;
`SetWithTTL` uses its argument (`ttl <= 0` means never expires, even if
`Options.TTL` is set). Expired entries are removed lazily (by the operations
below), never by a background goroutine.

### Operations

- `Set`/`SetWithTTL(key, value)`: computes the weight (`Weigher`, called once
  per call), then:
  1. If `key` is stored: the old entry is removed; reason `ReasonExpired`
     (counts as an expiration) if it had expired, else `ReasonReplaced`
     (counts as nothing).
  2. If `MaxBytes > 0` and the new weight > `MaxBytes`: the new value is not
     stored; `OnEvict(key, newValue, ReasonCapacity)`; counts as an eviction.
  3. Otherwise the entry is stored as the most recently used, then while
     `Len() > MaxEntries` (if > 0) or `Bytes() > MaxBytes` (if > 0), the least
     recently used **other** entry is removed: reason `ReasonExpired` (counts
     as an expiration) if it has expired, else `ReasonCapacity` (counts as an
     eviction). Expiry alone never makes `Set` remove other entries.
- `Get(key)`: absent → miss. Stored but expired → removed with
  `ReasonExpired`, counts as a miss **and** an expiration, returns false.
  Otherwise a hit; the entry becomes most recently used.
- `Peek(key)`: like `Get` but never changes recency, stats or contents; an
  expired entry returns false and stays stored.
- `Delete(key)`: absent → false. Stored and expired → removed with
  `ReasonExpired` (counts as an expiration), returns false. Otherwise removed
  with `ReasonDeleted`, returns true.
- `Len()`/`Bytes()`: number / total weight of **stored** entries, including
  expired entries not removed yet.
- `Keys()`: keys of unexpired entries, most recently used first. Does not
  remove anything.
- `RemoveExpired()`: removes every expired entry (`ReasonExpired`, each counts
  as an expiration), from least to most recently used; returns the count.
- `Purge()`: removes every entry with `ReasonDeleted`, from least to most
  recently used. Stats are unchanged.
- `Stats()`: cumulative counters. Hits/Misses are counted only by `Get`.
  Evictions count only `ReasonCapacity` removals; Expirations only
  `ReasonExpired` removals.

### Callbacks and concurrency

- All methods are safe for concurrent use.
- `OnEvict` (if non-nil) is called exactly once per removed entry,
  synchronously before the triggering method returns, in removal order, and
  **without holding the cache's internal lock** — a callback may call any
  method of the same cache without deadlocking.
- Callbacks run only after the triggering method has finished all of its
  changes to the cache, so a callback observes the final state (for example,
  during the `ReasonReplaced` callback of `Set`, the new value is already
  stored and `Len()` already reflects any capacity evictions of that `Set`).
