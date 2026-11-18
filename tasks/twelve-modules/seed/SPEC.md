# twelvemods — specification

This module (`module twelvemods`, Go 1.22, standard library only) contains twelve
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
| `regexlite` | `regexlite/` | backtracking regular expressions: groups, classes, lazy quantifiers, replace |
| `jsonpatch` | `jsonpatch/` | JSON Pointer (RFC 6901) and atomic JSON Patch (RFC 6902) |
| `cron` | `cron/` | five-field cron expressions and next run times |
| `mdlite` | `mdlite/` | Markdown subset to HTML |
| `expr` | `expr/` | expression tokenizer, parser and evaluator with positioned errors |
| `udiff` | `udiff/` | Myers line diff, unified diff output and patch application |
| `tmpl` | `tmpl/` | text templates with conditionals, loops, filters and escaping |

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

---

## 6. `regexlite`

```go
type SyntaxError struct {
    Pos int    // 0-based byte offset in the pattern
    Msg string // one of the messages listed below
}
func (e *SyntaxError) Error() string // fmt.Sprintf("regexlite: %s at position %d", Msg, Pos)

type Regexp struct{ /* unexported */ }
func Compile(pattern string) (*Regexp, error)
func MustCompile(pattern string) *Regexp // panics if Compile fails
func (re *Regexp) NumGroups() int
func (re *Regexp) MatchString(s string) bool
func (re *Regexp) FindStringSubmatchIndex(s string) []int
func (re *Regexp) FindStringSubmatch(s string) []string
func (re *Regexp) FindAllString(s string, n int) []string
func (re *Regexp) ReplaceAllString(src, repl string) string
```

A backtracking regular expression engine for the subset below. The package
`regexp` must not be used. Patterns and subjects are UTF-8; matching works on
runes, positions and indices are byte offsets.

### Syntax

```
regexp      = alternative *( "|" alternative )
alternative = *( atom [ quantifier ] | anchor )
atom        = literal | "." | class | escape | group
group       = "(" regexp ")" | "(?:" regexp ")"
quantifier  = ( "*" | "+" | "?" | "{" n "}" | "{" n ",}" | "{" n "," m "}" ) [ "?" ]
anchor      = "^" | "$"
```

- An alternative may be empty (`a|`, `(|b)`, `()` and the empty pattern are
  valid; an empty alternative matches the empty string).
- *Literal*: any rune other than the metacharacters `\ . [ ( ) | * + ? { ^ $`
  matches itself. `]` and `}` are ordinary literals outside a class.
- `.` matches any rune except `\n`.
- *Escape* (outside a class): `\d` = `[0-9]`, `\D` = not `\d`, `\w` =
  `[0-9A-Za-z_]`, `\W` = not `\w`, `\s` = one of space, `\t`, `\n`, `\r`,
  `\f`, `\v`, `\S` = not `\s`; `\n`, `\t`, `\r` are newline, tab and carriage
  return; `\` followed by any ASCII punctuation character (``!"#$%&'()*+,-./:;<=>?@[\]^_`{|}~``)
  is that character literally. `\` followed by anything else is an error.
- *Class*: `[` items `]`, or `[^` items `]` (negated: matches any rune,
  including `\n`, that no item matches). The first character after `[` or `[^`
  always begins an item, so a `]` there is a literal `]` (`[]a]` matches `]` or
  `a`; `[]` and `[^]` alone are unclosed). Otherwise an unescaped `]` closes
  the class. Items are read left to right; each starts with a *class atom*:
  - a *single character*: any rune other than `\`, or an escape `\n`, `\t`,
    `\r` or `\`+ASCII punctuation (`[\]]` is a class holding `]`);
  - a *class escape* `\d \D \w \W \s \S` (the sets above).

  If the atom is followed by `-` and the `-` is followed by a character other
  than `]`, the atom, the `-` and the next class atom form a *range* `x-y`:
  both endpoints must be single characters and `x <= y` by code point; the
  range matches the runes from `x` to `y` inclusive. Otherwise the atom is an
  item on its own and a following `-` begins the next item, so `[-a]`, `[a-]`
  and `[a-c-]` contain a literal `-`, and `[a-c-e]` is `a`..`c`, `-` and `e`.
- *Groups*: `( ... )` is capturing; groups are numbered 1, 2, ... by the
  position of their `(` from left to right. `(?: ... )` does not capture.
- *Quantifiers* apply to the atom right before them: `*` (0 or more), `+`
  (1 or more), `?` (0 or 1), `{n}` (exactly n), `{n,}` (n or more), `{n,m}`
  (n to m). `n` and `m` are decimal numbers of one or more digits with
  `n <= m <= 1000` (also `n <= 1000` for `{n}` and `{n,}`). A quantifier is
  *greedy* (prefers more repetitions); followed by `?` it is *lazy* (prefers
  fewer). A `{` always starts a quantifier.
- *Anchors*: `^` matches only at offset 0 of the subject, `$` only at its end
  (no multiline mode). Anchors cannot be quantified.

### Errors

`Compile` returns a `*SyntaxError` (and a nil `*Regexp`) for the first error,
scanning the pattern left to right:

| Message | Case | `Pos` |
|---|---|---|
| `trailing backslash` | `\` is the last character of the pattern (also inside a class) | the `\` |
| `invalid escape` | `\` followed by anything other than `d D w W s S n t r` or ASCII punctuation (inside or outside a class) | the `\` |
| `missing closing ]` | a class with no closing `]` | its `[` |
| `invalid character class range` | a range whose endpoint is a class escape (`[a-\d]`, `[\d-z]`) or whose start is greater than its end (`[z-a]`) | the first character of the range (the `\` for `[\d-z]`) |
| `missing closing )` | a group not closed before the end of the pattern | the `(` of the innermost unclosed group |
| `unexpected )` | a `)` with no open group | the `)` |
| `invalid group` | `(?` not followed by `:` | the `(` |
| `missing argument to repetition operator` | a quantifier (`*`, `+`, `?` or `{`) not preceded by an atom: at the start of the pattern or of a group, after `\|`, or after an anchor | the quantifier character |
| `invalid nested repetition operator` | a quantifier (including its optional lazy `?`) followed by another `*`, `+`, `?` or `{` | the second operator |
| `invalid repetition` | a `{` (preceded by an atom) not followed by a well-formed `n}`, `n,}` or `n,m}`, or with `n > m`, or with a count above 1000 | the `{` |

Examples: `a{,3}`, `a{x}`, `a{2` and `a{3,2}` are `invalid repetition` at 1;
`a**` and `a*??` are `invalid nested repetition operator` at 2 and 3; `(?:*)`
is `missing argument to repetition operator` at 3.

### Matching

The match at a given start offset is the first successful match found by a
backtracking search that tries alternatives from left to right, greedy
quantifiers with the most repetitions first and lazy quantifiers with the
fewest first (Perl semantics). A search tries start offsets from left to right
(rune by rune, including the end of the subject) and reports the match at the
first start offset where one exists ("leftmost-first"). This means `a|ab` on
`"ab"` matches `"a"`, and `a*?` matches the empty string.

Empty iterations: after a quantified atom has matched its minimum number of
repetitions, a further repetition that consumes no input is not accepted (this
prevents infinite loops: `(a*)*` and `(|a)+` terminate). Repetitions up to the
minimum are always attempted even when empty.

Captures: a group that took part in the match reports the span of the last
text it matched. A group inside a quantifier keeps the span from the last
iteration in which it matched, and a group that never matched reports `-1`.
Backtracking undoes captures made on the abandoned path.

### Methods

- `NumGroups()`: the number of capturing groups.
- `MatchString(s)`: whether there is a match anywhere in `s`.
- `FindStringSubmatchIndex(s)`: `nil` if no match; otherwise a slice of
  `2*(NumGroups()+1)` ints: the start and end offsets of the whole match, then
  of each group in order (`-1, -1` for a group that did not take part).
- `FindStringSubmatch(s)`: `nil` if no match; otherwise `NumGroups()+1`
  strings: the whole match and each group (`""` for a group that did not take
  part).
- `FindAllString(s, n)`: successive matches: the first search starts at offset
  0; after a match `[a,b)` with `b > a` the next search starts at `b`; after an
  empty match at `a` the next search starts one rune after `a` (and stops if
  `a` is the end of `s`). Returns at most `n` matches if `n >= 0`, all if
  `n < 0`. Returns `nil` when there is no match (also for `n == 0`).
- `ReplaceAllString(src, repl)`: replaces every match found as in
  `FindAllString(src, -1)` with `repl` expanded for that match; text between
  matches is copied unchanged. In `repl`, `$$` is a literal `$`; `$` followed by
  one or more decimal digits (the longest such run) or `${` digits `}` is the
  text of that group (group 0 is the whole match); a group number above
  `NumGroups()` or a group that did not take part expands to `""`. Any other
  `$` is a literal `$`.

---

## 7. `jsonpatch`

```go
var (
    ErrInvalidPointer   = errors.New("invalid JSON pointer")
    ErrPathNotFound     = errors.New("path not found")
    ErrInvalidIndex     = errors.New("invalid array index")
    ErrTestFailed       = errors.New("test operation failed")
    ErrInvalidOperation = errors.New("invalid operation")
    ErrInvalidDocument  = errors.New("invalid document")
)

type PatchError struct {
    Index int    // 0-based index of the failing operation; -1 when the patch as a whole is malformed
    Op    string // the operation's "op" ("" if absent or not a string)
    Path  string // the operation's "path" ("" if absent or not a string)
    Err   error  // one of the sentinel errors above
}
func (e *PatchError) Error() string // fmt.Sprintf("operation %d (%s %s): %v", Index, Op, Path, Err)
func (e *PatchError) Unwrap() error // returns Err

func ParsePointer(s string) ([]string, error)
func FormatPointer(tokens []string) string
func Get(doc any, pointer string) (any, error)
func Equal(a, b any) bool

type Operation struct {
    Op    string // "add", "remove", "replace", "move", "copy" or "test"
    Path  string
    From  string // move and copy only
    Value any    // add, replace and test only
}
func DecodePatch(data []byte) ([]Operation, error)
func Apply(doc any, patch []Operation) (any, error)
func ApplyJSON(doc, patch []byte) ([]byte, error)
```

JSON values are the Go values that `encoding/json` produces when decoding
into an `any`: `nil`, `bool`, `float64`, `string`, `[]any` and
`map[string]any`. No other types are passed to this package.

### JSON Pointer (RFC 6901)

- `ParsePointer("")` returns an empty (non-nil or nil) token list: the whole
  document. Any other pointer must start with `/`; it is split on `/` into
  tokens after the first `/` (`"/"` is one empty token `""`, `"/a//b"` is
  `a`, `""`, `b`). In each token `~1` becomes `/` and then `~0` becomes `~`
  (so `~01` is the literal `~1`). A `~` not followed by `0` or `1` (including
  at the end of a token) is an error; a pointer not starting with `/` is an
  error. Errors satisfy `errors.Is(err, ErrInvalidPointer)`.
- `FormatPointer(tokens)` is the inverse: `""` for no tokens, otherwise `/`
  before each token with `~` written as `~0` and `/` as `~1`.
- *Resolving* a token list against a value, token by token:
  - in a `map[string]any`, the token is a key; a missing key →
    `ErrPathNotFound`;
  - in a `[]any`, the token must be an *index*: `0` or a non-empty run of
    decimal digits not starting with `0` (any length); otherwise (`-`, `01`,
    `+1`, `-1`, `1.0`, `x`, `""`) → `ErrInvalidIndex`, except that the token
    `-` → `ErrPathNotFound` (it names the element after the last one, which
    never exists). An index `>= len` → `ErrPathNotFound`;
  - in any other value (a scalar or nil) → `ErrPathNotFound`.
- `Get(doc, pointer)` returns the resolved value (`ErrInvalidPointer` first
  if the pointer does not parse). It does not copy the value.

### Equal

`Equal(a, b)` is JSON equality: same type; equal booleans, strings and
numbers (`==` on `float64`); arrays of the same length with equal elements in
order; objects with the same key set and equal values. `nil` equals only `nil`.

### Operations (RFC 6902)

Operations run in order on a working document. For each operation:

1. `Op` must be one of the six names, otherwise `ErrInvalidOperation`.
2. `Path` must parse, otherwise `ErrInvalidPointer`; then, for `move` and
   `copy`, `From` must parse, otherwise `ErrInvalidPointer`.
3. The operation itself:
   - `add`: if `Path` is `""`, the whole document becomes `Value`. Otherwise
     the *parent* (all tokens but the last) is resolved (errors as above). If
     the parent is an object, the last token is the key that is set (added or
     replaced). If it is an array, the last token is `-` (append) or an index
     `<= len` (insert before that position, shifting later elements; an index
     equal to `len` appends); another token → `ErrInvalidIndex`, an index
     `> len` → `ErrPathNotFound`. A parent that is neither → `ErrPathNotFound`.
   - `remove`: `Path` `""` → `ErrInvalidOperation`. Otherwise the target must
     resolve (errors as above) and is removed (array elements after it shift
     down).
   - `replace`: the target must resolve; it is replaced by `Value` (`""`
     replaces the whole document).
   - `move`: if the tokens of `From` are a *proper prefix* of the tokens of
     `Path` (moving a value into one of its own children; compared token by
     token, so `/a` is not a prefix of `/ab`) → `ErrInvalidOperation`.
     Otherwise the value at `From` must resolve; it is removed as by `remove`,
     then added at `Path` as by `add` (`Path` is resolved after the removal).
     `From` `""` is a proper prefix of every non-empty path; `From == Path` is
     allowed and leaves the document unchanged (after checking that it
     resolves).
   - `copy`: the value at `From` must resolve; a deep copy of it is added at
     `Path` as by `add`.
   - `test`: the target must resolve, and must be `Equal` to `Value`;
     otherwise `ErrTestFailed`.

The first failing operation stops the patch: `Apply` returns `nil` and a
`*PatchError` with that operation's index, `Op`, `Path` and the sentinel.

### Apply

`Apply(doc, patch)` returns the patched document. It is atomic and
non-destructive: it never modifies `doc` or anything reachable from it (on
success or on failure), and the result shares no map or slice with `doc` or
with any operation's `Value` (values are deep-copied). An empty patch returns
a deep copy of `doc`.

### DecodePatch and ApplyJSON

`DecodePatch(data)` decodes a JSON patch document. Errors are `*PatchError`
with `Err == ErrInvalidOperation`:

- `data` is not valid JSON or its top level is not an array → `Index` `-1`,
  `Op` and `Path` `""`;
- element `i` is not an object; or its `"op"` member is missing or not a
  string or not one of the six names; or `"path"` is missing or not a string;
  or, for `move`/`copy`, `"from"` is missing or not a string; or, for
  `add`/`replace`/`test`, `"value"` is missing (`null` is a valid value) →
  `Index` `i`, with `Op` and `Path` set from the element when they are strings
  (`""` otherwise). Checks are made in that order: for an unknown op `Path`
  is still filled in.

Other members are ignored. Pointers are not checked by `DecodePatch` (only by
`Apply`). An empty array is a valid, empty patch.

`ApplyJSON(doc, patch)` decodes `doc` (invalid JSON → an error for which
`errors.Is(err, ErrInvalidDocument)`; it is not a `*PatchError`), decodes the
patch with `DecodePatch` (its error returned unchanged), applies it with
`Apply` (its error returned unchanged) and returns `json.Marshal` of the
result (compact, object keys sorted).

---

## 8. `cron`

```go
type ParseError struct {
    Field string // "minute", "hour", "day-of-month", "month", "day-of-week", or "" (see below)
    Msg   string // exact messages listed below
}
func (e *ParseError) Error() string // "cron: " + Msg if Field == "", else "cron: " + Field + " field: " + Msg

type Schedule struct{ /* unexported */ }
func Parse(expr string) (*Schedule, error)
func (s *Schedule) Matches(t time.Time) bool
func (s *Schedule) Next(after time.Time) (time.Time, bool)
func (s *Schedule) NextN(after time.Time, n int) []time.Time
```

### Expressions

An expression is either a macro or five fields separated by one or more
spaces or tabs (leading and trailing spaces/tabs are allowed):

| # | Field | Values | Names |
|---|---|---|---|
| 1 | `minute` | 0-59 | |
| 2 | `hour` | 0-23 | |
| 3 | `day-of-month` | 1-31 | |
| 4 | `month` | 1-12 | `JAN`-`DEC` = 1-12 |
| 5 | `day-of-week` | 0-7 (0 and 7 are both Sunday) | `SUN`-`SAT` = 0-6 |

Macros (the whole expression after trimming, lowercase only):
`@yearly` and `@annually` = `0 0 1 1 *`, `@monthly` = `0 0 1 * *`,
`@weekly` = `0 0 * * 0`, `@daily` and `@midnight` = `0 0 * * *`,
`@hourly` = `0 * * * *`.

A field is a comma-separated list of one or more *items*. An item is split at
its first `/` into a *base* and an optional *step*; the base is split at its
first `-` into a start and an optional end:

- `*`: every value of the field (for day-of-week `*` is 0-7).
- `?`: same as `*`, allowed only in day-of-month and day-of-week.
- `V`: the single value V.
- `A-B`: every value from A to B inclusive; `A > B` is an error (no
  wrap-around).
- `BASE/S`: from the values of BASE (`*`, `?` or `A-B`), take the first and
  every S-th after it (A, A+S, A+2S, ... up to B). A single value `V/S` means
  `V-MAX/S`, where MAX is the field's maximum (59, 23, 31, 12, or 7 for
  day-of-week).
- Values are decimal numbers (leading zeros allowed: `05` is 5) or, for month
  and day-of-week, a three-letter name in any letter case (`jan`, `Mon`).
  Names may be used wherever a value may (`MON-FRI`, `JAN/3`), but not as the
  step. A step is a decimal number >= 1.
- For day-of-week, 7 is turned into 0 after the set is computed, so
  `5-7` is Fri, Sat, Sun and `*/2` is 0, 2, 4, 6.

### Errors

`Parse` returns a nil `*Schedule` and a `*ParseError` for the first problem,
checking fields from left to right and the items of a field from left to
right:

| Case | `Field` | `Msg` |
|---|---|---|
| expression starts with `@` but is not one of the macros | `""` | `unknown macro "TEXT"` (TEXT is the trimmed expression) |
| not exactly five fields (including an empty expression) | `""` | `expected 5 fields, got N` |
| an empty item (`1,,2`, `1,`, `,1`) | field name | `empty list item` |
| a value that is neither a decimal number nor a name valid for the field (including `?` outside day-of-month/day-of-week, `*` used as a range end, an empty start or end such as `-5` or `5-`, and a second `-` as in `1-2-3` whose end is `2-3`) | field name | `invalid value "TEXT"` (TEXT is that value) |
| a decimal value outside the field's range | field name | `value "TEXT" out of range MIN-MAX` (TEXT as written; e.g. `value "60" out of range 0-59`) |
| a range with start > end | field name | `invalid range "TEXT"` (TEXT is the base, e.g. `"5-3"`) |
| a step that is not a decimal number >= 1 (`0`, `x`, empty, `2/3` when the item is `1/2/3`) | field name | `invalid step "TEXT"` (TEXT is everything after the first `/`) |

Within an item the start is checked first, then the end, then the range
order, then the step. `*` and `?` are only valid as the whole base, so `*-5`
reports `invalid value "*"` and `1-*` reports `invalid value "*"`.

### Matching

Times are interpreted in UTC (`t.UTC()`); seconds and smaller units are
ignored. A time matches when its minute, hour and month are in their sets and
its day matches:

- the day-of-month field is *unrestricted* if its text is exactly `*` or `?`;
  likewise the day-of-week field. Any other text (even `*/1` or `1-31`) is
  *restricted*;
- if both day fields are restricted, the day matches when the day of the month
  is in the day-of-month set **or** the weekday is in the day-of-week set;
- otherwise the day matches when the day of the month is in the day-of-month
  set **and** the weekday is in the day-of-week set (an unrestricted field
  contains every value, so only the other one matters).

Day-of-month values that a month does not have (31 in April, 30 in February)
simply never match.

- `Matches(t)` reports whether the minute containing `t` matches.
- `Next(after)` returns the earliest matching time that is strictly later than
  `after`, as a UTC `time.Time` with zero seconds and nanoseconds, and true.
  Only times up to the end of the year `after.UTC().Year() + 10` are
  considered; if none matches, it returns the zero `time.Time` and false (for
  example `0 0 30 2 *`). Example: after `2024-01-01 10:15:30` UTC, `* * * * *`
  gives `10:16:00`, and `15 10 * * *` gives `2024-01-02 10:15:00`.
- `NextN(after, n)` returns up to `n` successive `Next` times (each one
  strictly after the previous), stopping early when `Next` returns false;
  `n <= 0` returns an empty (nil or zero-length) slice.

---

## 9. `mdlite`

```go
func ToHTML(markdown string) string
```

Converts a Markdown subset to HTML. Every input is valid; there are no errors.

### Lines and output format

- `"\r\n"` is first replaced by `"\n"`; the input is then split on `"\n"`
  (a final `"\n"` does not create an extra line). Tabs are ordinary
  characters; only spaces (U+0020) count as indentation or blank space.
- A *blank* line is empty or contains only spaces.
- The output is the concatenation of the rendered blocks, each followed by
  `"\n"`. An input with no blocks gives `""`.
- *Escaping*: in all text output (inline text, code, link URLs, the language
  name) `&`, `<`, `>` and `"` are replaced by `&amp;`, `&lt;`, `&gt;` and
  `&quot;`. Nothing else is escaped; raw HTML is not supported.

### Blocks

Blocks are read line by line. Outside a fenced code block, a line is
classified by the first rule that applies:

1. **Blank**: ends the current paragraph or list; otherwise ignored.
2. **Fence opening**: starts at column 0 with three or more backticks
   (`` ` ``). The rest of the line after the backticks, trimmed of spaces, is
   the *info*; the language is its first space-separated word (if any).
3. **ATX heading**: starts at column 0 with 1 to 6 `#` followed by a space or
   the end of the line. Text = the rest of the line, trimmed of spaces.
   Trailing `#` characters are kept as text. (`#foo` and `####### x` are not
   headings.)
4. **Thematic break**: the line, with trailing spaces removed, is three or
   more `-` and nothing else, starting at column 0.
5. **Blockquote line**: starts at column 0 with `>`.
6. **List item line**: zero or more spaces (the item's *indent*), then a
   marker, then one space, then the item text, which trimmed of spaces must
   be non-empty. Markers: `-`, `*` or `+` (unordered), or 1 to 9 decimal
   digits followed by `.` (ordered). (`-`, `- ` + spaces only, `1.x`, `-x`,
   `1)` are not item lines.)
7. Otherwise a **paragraph line**.

Rendering:

- **Paragraph**: consecutive paragraph lines; any other kind of line ends it.
  Each line is trimmed of leading and trailing spaces; the lines are joined
  with `"\n"` and rendered as inline content: `<p>…</p>`.
- **Heading**: `<hN>…</hN>` with the inline-rendered text (`#` alone gives
  `<h1></h1>`).
- **Thematic break**: `<hr>`.
- **Fenced code block**: the lines after the opening fence, verbatim, up to a
  *closing fence* — a line starting at column 0 with at least as many
  backticks as the opening fence and nothing else but trailing spaces — or the
  end of the input (or of the enclosing blockquote). Lines inside are never
  classified. Output: `<pre><code>` (or `<pre><code class="language-LANG">`)
  followed by each content line escaped and followed by `"\n"`, then
  `</code></pre>`. No inline processing happens inside.
- **Blockquote**: consecutive blockquote lines (a non-`>` line ends it, even
  a paragraph line). From each line the `>` and, if present, one following
  space are removed; the resulting lines are parsed as a nested document with
  all these rules. Output: `<blockquote>\n` + the nested blocks (each followed
  by `"\n"`, as above) + `</blockquote>`.
- **Lists**: consecutive list item lines form a list structure; any other
  line (including a blank line) ends it. Items are placed with a stack of open
  lists, each with an indent and a type (ordered/unordered; the unordered
  marker characters are interchangeable). For an item line with indent `K`
  and type `T`:
  1. while the top list's indent is greater than `K`, close it;
  2. if the top list has indent `K` and type `T`, add the item to it;
  3. if the top list has indent `K` and the other type, close it, then
     continue with 4;
  4. otherwise open a new list (indent `K`, type `T`) holding the item: as a
     child of the last item of the top list if there is one, or else as a new
     top-level block.

  An item's text is rendered as inline content. Output of a list:
  `<ul>\n` (or `<ol>\n`, or `<ol start="N">\n` when the first item's number
  is not 1, N in decimal without leading zeros) + its items + `</ul>` /
  `</ol>`. An item is `<li>` + inline text + `</li>\n`, or, when it has child
  lists, `<li>` + inline text + `"\n"` + each child list followed by `"\n"` +
  `</li>\n`. The numbers of later items are ignored.

### Inline content

Inline content is scanned left to right. At each position, the first rule
that applies:

1. **Backslash escape**: `\` followed by an ASCII punctuation character
   produces that character literally (it never starts or ends anything). Any
   other `\` is a literal backslash.
2. **Code span**: a run of N backticks starts a code span that ends at the
   next run of exactly N backticks. The content between them has each `"\n"`
   replaced by a space; then, if it both starts and ends with a space and is
   not only spaces, one space is removed from each end. Output:
   `<code>` + escaped content + `</code>`. No other processing inside. If
   there is no closing run, the opening run is literal text.
3. **Link**: `[`, the link text, `](`, the url, `)`. The link text is found by
   scanning from the character after `[`: a `\` followed by an ASCII
   punctuation character skips both characters; an unskipped `[` means there
   is no link here; the first unskipped `]` ends the text, and it must be
   followed directly by `(`. The url is zero or more characters other than
   space, `"\n"`, `(` and `)`, followed by `)`. Output:
   `<a href="` + escaped url + `">` + inline-rendered link text + `</a>` (the
   url is taken verbatim: no backslash escapes). If any part is missing, the
   `[` is literal text and scanning goes on at the next character.
4. **Emphasis delimiters**: a maximal run of `*`. A run of length 1 or 2 is a
   delimiter; a longer run is literal text. A run *can open* if it is followed
   by a character that is not whitespace (space, tab or `"\n"`), and *can close*
   if it is preceded by a character that is not whitespace (the start or end of
   the inline content counts as whitespace). "Preceded" and "followed" refer
   to the source characters, so in `\**a*` the run `*` before `a` can close.
5. Any other character is literal text.

Delimiters are matched with a stack, in the order they are found: a run that
can close and finds, on the stack, an opener of the same length pops it (the
nearest such opener; any openers above it are removed from the stack and
become literal text) and the content between them becomes
`<em>…</em>` (length 1) or `<strong>…</strong>` (length 2). Otherwise, if
it can open, it is pushed. Otherwise it is literal text. Openers left on the
stack at the end are literal text. Links and code spans are atomic:
delimiters inside link text are matched within the link text only.

Examples: `*a **b** c*` → `<em>a <strong>b</strong> c</em>`;
`*a **b*` → `<em>a **b</em>`; `**a*` → `**a*`; `***a***` → `***a***`;
`a*b*c` → `a<em>b</em>c`; `* a *` → `* a *`.

---

## 10. `expr`

```go
type Error struct {
    Line, Column int    // 1-based; Column counts runes within the line
    Msg          string // exact messages listed below
}
func (e *Error) Error() string // fmt.Sprintf("%d:%d: %s", Line, Column, Msg)

type Expr struct{ /* unexported */ }
func Parse(src string) (*Expr, error)
func (e *Expr) String() string
func (e *Expr) Eval(env map[string]any) (any, error)
func Eval(src string, env map[string]any) (any, error) // Parse, then Eval
```

An expression language with four value types: **int** (`int64`), **float**
(`float64`), **bool** and **string**. `Eval` returns one of these Go types.
Variables in `env` hold `int64`, `float64`, `bool` or `string` values (a nil
`env` has no variables). All errors are `*Error`; `Parse` and `Eval` return
a nil `*Expr` / nil value with an error.

### Tokens

Whitespace (space, tab, `\r`, `\n`) separates tokens and is otherwise
ignored. Positions: line 1 column 1 is the first character; `\n` starts a new
line.

- **Int literal**: one or more decimal digits. A value above the int64
  maximum is the error `integer literal out of range` (at the literal), so
  `-9223372036854775808` cannot be written.
- **Float literal**: digits `.` digits, with an optional exponent, or digits
  with an exponent; the exponent is `e` or `E`, an optional sign, and one or
  more digits. If `e`/`E` is not followed by digits (after an optional sign)
  it is not part of the number (`1e` is the int `1` followed by the
  identifier `e`). `1.` and `.5` are not literals (`.` is an unexpected
  character). A value that overflows to infinity is the error
  `float literal out of range`.
- **String literal**: `"` … `"` with escapes `\"`, `\\`, `\n`, `\t`. A `\`
  followed by another character is the error `invalid escape \c` (the text
  `invalid escape ` followed by the backslash and that character, at the
  backslash). No closing quote (a `\` right before the end counts as none):
  `unterminated string` (at the opening quote). Strings may contain newlines.
- **Identifier**: `[A-Za-z_][A-Za-z0-9_]*`. `true` and `false` are bool
  literals.
- **Operators and punctuation**: `+ - * / % ^ == != < <= > >= && || ! ? : ( ) ,`
  (the longest match wins, so `<=` is one token).
- Any other character (including a lone `&`, `|` or `=`) is the error
  `unexpected character 'c'` (formatted with `fmt.Sprintf("unexpected character %q", r)`
  for the rune `r`), at that character.

The whole input is tokenized before parsing, so a lexical error anywhere is
reported even if a syntax error comes earlier.

### Grammar (lowest precedence first)

```
expr     = or [ "?" expr ":" expr ]      ; right-associative: a ? b : c ? d : e = a ? b : (c ? d : e)
or       = and { "||" and }
and      = equality { "&&" equality }
equality = compare { ("==" | "!=") compare }
compare  = sum { ("<" | "<=" | ">" | ">=") sum }
sum      = product { ("+" | "-") product }
product  = unary { ("*" | "/" | "%") unary }
unary    = ("-" | "!") unary | power
power    = primary [ "^" unary ]             ; right-associative, binds tighter than unary minus
primary  = int | float | string | "true" | "false"
         | identifier                        ; variable
         | identifier "(" [ expr { "," expr } ] ")"  ; function call
         | "(" expr ")"
```

All binary operators except `^` are left-associative. So `-2 ^ 2` is
`-(2 ^ 2)`, `2 ^ -1` is `2 ^ (-1)` and `2 ^ 3 ^ 2` is `2 ^ (3 ^ 2)`.

A syntax error is reported at the first token that does not fit the grammar,
as `unexpected token "T"` (`fmt.Sprintf("unexpected token %q", text)` with
the token's source text, e.g. `unexpected token ")"`), or as
`unexpected end of input` when the input ends too early, positioned just
after the last character of the input, trailing whitespace included (for
`1 +` that is 1:4, for `"1 +\n"` it is 2:1, for an empty input 1:1). Tokens left after a complete expression are also an
error (`1 2` → `unexpected token "2"` at 1:3).

### String

`String()` returns a canonical, fully parenthesized form:

- int: decimal; float: `strconv.FormatFloat(f, 'g', -1, 64)` with `.0`
  appended when the result consists only of digits (so `2.0` → `2.0`, `1e3`
  → `1000.0`, `1.5e300` → `1.5e+300`); string: `strconv.Quote`; bool: `true`
  / `false`; variable: its name;
- unary: `(-x)`, `(!x)`; binary: `(a OP b)` with single spaces; ternary:
  `(c ? a : b)`; call: `name(a, b)` (`name()` with no arguments).

Example: `1+2*3 == 7 && !ok ? max(a, 2) : -x^2` →
`((((1 + (2 * 3)) == 7) && (!ok)) ? max(a, 2) : (-(x ^ 2)))`.

### Evaluation

Operands are evaluated left to right. "Numeric" means int or float; in a
numeric operation with one float operand the int is converted to float.

- `+`: int + int → int; numeric with a float → float; string + string →
  concatenation. `-`, `*`: int with int → int, otherwise float. Int `+ - *`
  wrap around on overflow like Go's `int64`.
- `/`: int / int → int, truncated toward zero; otherwise float. A zero
  divisor (int `0` or float `0.0`) is the error `division by zero`.
- `%`: ints only (`invalid operands` for floats); result has the sign of the
  dividend, like Go; zero divisor → `division by zero`.
- `^`: int ^ int with exponent >= 0 → int, equal to repeated multiplication
  with wrap-around (`x ^ 0` is 1; large exponents must not take long); otherwise (a float operand or a negative int
  exponent) `math.Pow` on floats → float.
- `< <= > >=`: two ints (compared exactly), two numerics with a float
  (compared as floats) or two strings (byte-wise); → bool.
- `==`, `!=`: two ints compare exactly; an int and a float compare as
  floats (`1 == 1.0` is true); two floats as floats; two
  bools or two strings compare by value; values of different kinds (other
  than int/float) are simply not equal (no error).
- `&&`, `||`: the left operand must be bool; `&&` with a false left and `||`
  with a true left return it without evaluating the right operand. Otherwise
  the right operand must be bool and is the result.
- `!` needs a bool; unary `-` a numeric (int negation wraps: `-x` of the
  minimum int64 is itself).
- `c ? a : b`: `c` must be bool; only the chosen branch is evaluated.
- Variable: its value in `env`; absent → `undefined variable "x"`.
- Functions (all arguments are evaluated first, left to right, then checked):

| Function | Arguments | Result |
|---|---|---|
| `min(x, ...)`, `max(x, ...)` | one or more numerics | the smallest / largest; int if all arguments are ints, else float |
| `abs(x)` | one numeric | same type (int abs wraps for the minimum int64) |
| `len(s)` | one string | number of runes, as int |
| `upper(s)`, `lower(s)` | one string | `strings.ToUpper` / `strings.ToLower` |

Evaluation errors and their positions:

| Error | Msg | Position |
|---|---|---|
| binary operator with operand kinds it does not accept | `invalid operands for "OP": K1 and K2` | the operator |
| `!`, unary `-`, `&&`, `\|\|` with a wrong operand | `invalid operand for "OP": K` (the first wrong operand, left before right) | the operator |
| ternary condition not bool | `invalid operand for "?": K` | the `?` |
| zero divisor | `division by zero` | the operator |
| unknown variable | `undefined variable "NAME"` | the identifier |
| unknown function | `unknown function "NAME"` | the function name |
| wrong argument count | `wrong number of arguments for "NAME"` | the function name |
| argument of the wrong kind | `invalid argument for "NAME": K` (the first such argument) | the function name |

Kinds are written `int`, `float`, `bool` and `string`. The unknown-function
check happens before arguments are evaluated; the argument count is checked
after.

---

## 11. `udiff`

```go
type OpKind int
const (
    OpEqual OpKind = iota
    OpDelete
    OpInsert
)

type Edit struct {
    Kind OpKind
    Line string // the line, including its "\n" if it has one
}

func SplitLines(s string) []string
func Diff(a, b []string) []Edit
func Unified(aName, bName, a, b string, context int) string

type ApplyError struct {
    Line int    // 1-based line number in the patch text
    Msg  string // exact messages listed below
}
func (e *ApplyError) Error() string // fmt.Sprintf("patch line %d: %s", Line, Msg)

func Apply(original, patch string) (string, error)
```

### Lines

`SplitLines(s)` splits after every `"\n"`; each line keeps its `"\n"`, and
the last line has none if `s` does not end with `"\n"`. `""` gives no lines
(nil or empty); `"a\nb"` gives `["a\n", "b"]`; `"a\n"` gives `["a\n"]`.
Lines are compared as whole strings, so `"x"` and `"x\n"` differ.

### Diff (Myers)

`Diff(a, b)` returns a shortest edit script (fewest deletions + insertions)
turning `a` into `b`: every line of `a` appears once as `OpEqual` or
`OpDelete`, every line of `b` once as `OpEqual` or `OpInsert`, in order. When
several shortest scripts exist, the one chosen is the one produced by exactly
this algorithm (Myers' greedy forward search, then a backtrack):

```
N = len(a), M = len(b)
V = array indexed by k in [-(N+M)-1, N+M+1], all 0
trace = empty list
for D = 0, 1, ..., N+M:
    append a copy of V to trace            // trace[D] = V before round D
    for k = -D, -D+2, ..., D:
        if k == -D or (k != D and V[k-1] < V[k+1]):
            x = V[k+1]                     // down: insert
        else:
            x = V[k-1] + 1                 // right: delete
        y = x - k
        while x < N and y < M and a[x] == b[y]:
            x, y = x+1, y+1
        V[k] = x
        if x >= N and y >= M: stop both loops; the final D is this D

backtrack, starting at x = N, y = M and building the script from the end:
for d = D down to 1:
    V = trace[d]; k = x - y
    if k == -d or (k != d and V[k-1] < V[k+1]): prevK = k+1 else prevK = k-1
    prevX = V[prevK]; prevY = prevX - prevK
    while x > prevX and y > prevY: emit OpEqual a[x-1]; x, y = x-1, y-1
    if x == prevX: emit OpInsert b[y-1]; y = y-1
    else:          emit OpDelete a[x-1]; x = x-1
while x > 0: emit OpEqual a[x-1]; x, y = x-1, y-1
reverse the emitted edits
```

(`Diff(nil, nil)` is empty; `Diff(["a\n"], ["b\n"])` is delete `a`, insert
`b`.)

### Unified

`Unified(aName, bName, a, b, context)` diffs `SplitLines(a)` against
`SplitLines(b)` with `Diff` and formats the result (`context < 0` counts as
0). If there are no deletions or insertions, it returns `""`. Otherwise:

- Header: `"--- " + aName + "\n"` then `"+++ " + bName + "\n"`.
- *Changes* are the `OpDelete`/`OpInsert` edits. Two changes go in the same
  hunk when at most `2*context` `OpEqual` edits separate them. A hunk holds
  its changes, the equal lines between them, and up to `context` equal lines
  before its first change and after its last change.
- Hunk header: `@@ -S1,C1 +S2,C2 @@` then `"\n"`. `C1` is the number of hunk
  lines taken from `a` (equal + deleted), `C2` from `b` (equal + inserted).
  `S1` is the 1-based number in `a` of the hunk's first line from `a`; when
  `C1 == 0` it is the number of the line *after which* the hunk applies (0 at
  the start). `S2` likewise for `b`. `,C` is omitted when the count is 1
  (`@@ -3 +3,2 @@`).
- Hunk lines, in edit order: `" "`, `"-"` or `"+"` followed by the line. A
  line without a trailing `"\n"` is followed by `"\n\\ No newline at end of
  file\n"` (that is: a newline, a backslash, the text
  ` No newline at end of file`, a newline).

Example: `Unified("a", "b", "x\ny\n", "x\nz\n", 1)` is
`"--- a\n+++ b\n@@ -1,2 +1,2 @@\n x\n-y\n+z\n"`.

### Apply

`Apply(original, patch)` applies a unified diff to `original`
(`SplitLines(original)`) and returns the result. `Apply(a, Unified(n1, n2, a, b, c))`
must equal `b` for every `a`, `b`, `c`. A patch with no hunks (including
`""`) returns `original` unchanged.

The patch is split into lines on `"\n"` (a final `"\n"` does not create an
extra empty line; line numbers start at 1):

1. Leading lines that start with `"--- "` or `"+++ "` are skipped.
2. Then come zero or more hunks. Each starts with a header line
   `@@ -S1[,C1] +S2[,C2] @@`, optionally followed by a space and any text;
   numbers are one or more decimal digits that fit in an `int` (missing `,C`
   means 1). `S2` is never checked; `C2` is only used to count body lines.
3. Body lines follow: `" "` + text (context), `"-"` + text (delete), `"+"` +
   text (insert); the line's content is the text plus `"\n"`. A line starting
   with `\` is a *no-newline marker*: it removes the `"\n"` from the content of
   the body line right before it. The body ends when `C1` lines of context +
   delete and `C2` lines of context + insert have been read (a marker directly
   after the last body line still belongs to the hunk).
4. After a hunk, the next line must be another hunk header or the end of the
   patch.

A hunk with `C1 > 0` applies at line `S1` of the original: its context and
delete lines must equal the original lines `S1`, `S1+1`, ... exactly (no
fuzz, no offset search). A hunk with `C1 == 0` inserts after original line
`S1` (0 = before the first line). Hunks must be in order and not overlap:
with `P` = the last original line used by the previous hunk (its `S1+C1-1`,
or its `S1` when its `C1 == 0`; 0 before the first hunk), a hunk with
`C1 > 0` needs `S1 > P` and one with `C1 == 0` needs `S1 >= P`.

Errors (on error `Apply` returns `""`). The patch is first read from top to
bottom and the first *structural* error (every row but the last) is
reported; only if there is none are the context and delete lines compared
with the original, hunk by hunk in patch order, reporting the first mismatch:

| Msg | Case | `Line` |
|---|---|---|
| `expected hunk header` | a line where a hunk header must be (after the skipped header lines or after a complete hunk) that is not `@@ -` … | that line |
| `invalid hunk header` | a line starting with `@@ -` that does not have the form above, or has `C1 > 0` with `S1 == 0` | that line |
| `hunk out of order` | `S1` violates the order rule above, or `S1 + C1 - 1` (or `S1` when `C1 == 0`) is beyond the end of the original | the header line |
| `unexpected line in hunk` | a body line that does not start with `" "`, `"-"`, `"+"` or `\`, a marker not directly after a body line, or a body line that would exceed `C1` or `C2` | that line |
| `unexpected end of patch` | the patch ends before a hunk's counts are reached | the number of patch lines + 1 |
| `does not match original line N` | a context or delete line differing from original line `N` | that body line |

The order and range checks are made when the header is read.

---

## 12. `tmpl`

```go
type Error struct {
    Line, Column int    // position of the "{{" of the offending action; 1-based, Column in runes
    Msg          string // exact messages listed below
}
func (e *Error) Error() string // fmt.Sprintf("%d:%d: %s", Line, Column, Msg)

type Options struct {
    EscapeHTML bool // escape the output of value actions
}

type Template struct{ /* unexported */ }
func Parse(src string, opts Options) (*Template, error)
func (t *Template) Execute(data any) (string, error)
```

A small text template engine. All errors are `*Error`; `Parse` returns a nil
`*Template` and `Execute` returns `""` with an error.

### Data model

Data values are `nil`, `bool`, `int`, `int64`, `float64`, `string`, `[]any`
and `map[string]any` (nested freely). Their *kinds*, used in messages, are
`nil`, `bool`, `int` (both `int` and `int64`), `float`, `string`, `list` and
`map`.

- *Printing* a value: `nil` → `""`; bool → `true`/`false`; int → decimal;
  float → `strconv.FormatFloat(f, 'g', -1, 64)`; string → itself. A list or a
  map cannot be printed: `cannot print list` / `cannot print map`.
- *Truthiness*: `nil`, `false`, `0`, `0.0`, `""`, an empty list and an empty
  map are false; everything else is true.

### Syntax

Text outside actions is copied unchanged; there is no way to write a literal
`{{` in text. An action starts with `{{`; it is scanned from there: a `"`
starts a string literal that runs to the next `"` not escaped by a `\` (if
the input ends first: `unterminated string`), and the first `}}` outside a
string literal ends the action (if the input ends first: `unclosed action`).
Comment actions are scanned differently (below).

**Trim markers**: if an action starts with `{{-` followed by a space, tab,
`\r` or `\n`, all whitespace (space, tab, `\r`, `\n`) at the end of the text
before it is removed; if it ends with a whitespace character followed by
`-}}`, all whitespace at the start of the text after it is removed. Markers
work on every kind of action. `{{-3}}` is the int literal `-3` without trim.

**Comments**: when the text after `{{` (after a left trim marker, if any,
and whitespace) starts with `/*`, the action is a comment. It ends at the
first `*/` that is followed by optional whitespace and `}}`, or by at least
one whitespace character and `-}}` (a right trim marker); if there is none,
`unclosed action`. A comment produces nothing (its trim markers still apply)
and may contain `}}` and quotes.

Inside other actions, tokens are separated by spaces, tabs, `\r`, `\n`:

- *paths*: `.` (the current value, "dot"), `.name`, `.name.sub...` (field
  lookups starting at dot), `$` (the root data), `$.name...`; names are
  `[A-Za-z_][A-Za-z0-9_]*`. `$index` and `$key` are the loop variables (below);
- *literals*: strings `"..."` with escapes `\"`, `\\`, `\n`, `\t` (any other
  escape is `invalid action`); ints `-?[0-9]+` that fit in an int64; `true`,
  `false`;
- `|` separates pipeline stages;
- *words* `[A-Za-z_][A-Za-z0-9_]*` (other than `true`/`false`): the keywords
  `if`, `else`, `range`, `end` at the start of an action, filter names after
  `|`. A word anywhere else, or any other token, is `invalid action`.
  `$` followed by a name other than `index`/`key` is `invalid action`, and
  loop variables take no field lookups.

Action forms:

```
{{ pipeline }}                     value action: print the pipeline's value
{{ if pipeline }} ... {{ else if pipeline }} ... {{ else }} ... {{ end }}
{{ range pipeline }} ... {{ else }} ... {{ end }}
pipeline = operand { "|" filter { argument } }
operand, argument = path | literal
```

`if` may have any number of `else if` branches and at most one `else`, which
must come last. `range` may have one `else`. Blocks nest freely.

### Execution

- Field lookup `.a.b`: on `nil` the result is `nil`; on a map the value of
  that key (`nil` if absent); on anything else the error
  `cannot access field "b" on KIND` (KIND of the value being looked into).
- A value action prints its pipeline's value. With `EscapeHTML`, the printed
  text is HTML-escaped (`&` `<` `>` `"` `'` → `&amp;` `&lt;` `&gt;` `&quot;`
  `&#39;`) unless the value is *safe* (see `raw` and `escape`). Text outside
  actions is never escaped.
- `if`: runs the first branch whose pipeline is truthy, else the `else`
  branch if any. Pipelines of later branches are not evaluated.
- `range`: a list iterates its elements in order; a map iterates its values
  in ascending byte order of the keys; `nil` iterates zero times; any other
  kind is the error `cannot range over KIND`. Inside the body, dot is the
  element, `$index` is the 0-based iteration number (int) and `$key` the map
  key (string), or for a list the same int as `$index`. Zero iterations run
  the `else` branch if any. Dot and the loop variables are restored after the
  loop; `$` is always the root data.
- Filters receive the value of the previous stage as their input:

| Filter | Args | Input → result |
|---|---|---|
| `upper`, `lower`, `trim` | 0 | string → `strings.ToUpper` / `strings.ToLower` / `strings.TrimSpace` |
| `len` | 0 | string → number of runes; list, map → number of elements; nil → 0 |
| `default X` | 1 | if the input is falsy, X, else the input |
| `join SEP` | 1 (string) | list → the printed elements joined with SEP; nil → `""` |
| `truncate N` | 1 (int >= 0) | string → its first N runes |
| `raw` | 0 | any printable value → its printed text, marked safe |
| `escape` | 0 | any printable value → its printed text HTML-escaped, marked safe |

  A safe value passed to another filter is treated as a plain string and the
  result is not safe (`{{ .x | raw | upper }}` is escaped again). A filter's
  input is checked before its argument; arguments are evaluated (left to
  right) before the filter runs. Filters other than `default`, `raw` and
  `escape` reject inputs of kinds not listed (`invalid input`).

### Errors

All errors report the position of the `{{` of the action concerned.

Parse errors (the first one in the source):

| Msg | Case |
|---|---|
| `unclosed action` | `{{` with no matching `}}` |
| `empty action` | an action with no content (`{{ }}`, `{{- -}}`) |
| `unterminated string` | a string literal whose closing quote is missing (it runs to the end of the input) |
| `invalid action` | anything else that does not fit the grammar: a bad token (`{{ a }}`, `{{ .a. }}`, `{{ .1 }}`, `{{ $x }}`, `{{ "\q" }}`, `{{ 99999999999999999999 }}`), a missing pipeline or filter (`{{ if }}`, `{{ .x | }}`, `{{ .x | "s" }}`), extra tokens after `else` or `end`, an operand followed by something other than `\|` |
| `unknown filter "NAME"` | a filter name not in the table |
| `wrong number of arguments for filter "NAME"` | |
| `$index outside range` / `$key outside range` | a loop variable used where no `range` body is open (a `range`'s own pipeline and its `else` branch are not part of its body) |
| `unexpected {{else}}` | `else` (or `else if`) outside an `if`/`range`, after the `else` of its block, or `else if` in a `range` |
| `unexpected {{end}}` | `end` with no open block |
| `unclosed {{if}}` / `unclosed {{range}}` | end of input with an open block (position: its opening action; the innermost one) |

Execution errors (the first one in execution order):

| Msg | Case |
|---|---|
| `cannot print list` / `cannot print map` | a value action, `join` element, `raw` or `escape` on a list or map |
| `cannot access field "NAME" on KIND` | field lookup on a value that is not a map or nil |
| `cannot range over KIND` | |
| `invalid input for filter "NAME": KIND` | a filter given an input kind it does not accept |
| `invalid argument for filter "NAME"` | a `join` separator that is not a string, a `truncate` count that is not an int or is negative |
