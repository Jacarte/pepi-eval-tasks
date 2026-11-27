# corpusstats

Package `corpusstats` (directory `corpusstats/`, import path `corpustokens/corpusstats`) reports
statistics over a directory tree of text files. The input is `corpus/` in this module, passed
to every function as `root`. Use the standard library only.

## Counted files

A counted file is a regular file anywhere under `root`, at any depth, whose name ends in
exactly `.txt`, `.md` or `.go` (case-sensitive: `NOTES.TXT` and `draft.txt.bak` do not count).
Every other file is ignored. The file's extension is the suffix above, including the dot.

## Lines

The lines of a file are its bytes split at `\n`:

- the number of `\n` bytes, plus one if the file is non-empty and its last byte is not `\n`;
- a `\r` before `\n` belongs to the line break, so `\r\n` is one line break;
- an empty file has zero lines; a lone `\r` is not a line break;
- blank and whitespace-only lines count as lines.

## Comment lines

A line is a comment line when, after removing leading spaces and tabs, it starts with:

- `#` in `.txt` and `.md` files (this includes markdown headings);
- `//` in `.go` files.

Nothing else is a comment: `/* */` lines, and code followed by a `#` or `//`, are not comments.

## Tokens

A token is a maximal non-empty run of ASCII letters (`A`-`Z`, `a`-`z`) and apostrophes (`'`)
within one line. Non-ASCII letters are not part of a token, so they split the run. Tokens are
lowercased (ASCII only). A token never spans a line break.

The tokens of a file are the tokens of its lines that are not comment lines.

## Functions

```go
// FileTokens is one counted file: its path relative to root, slash-separated, and its token count.
type FileTokens struct {
	Path   string
	Tokens int
}

// Files returns the number of counted files per extension (keys ".txt", ".md", ".go").
// An extension appears in the map only when at least one counted file has it.
func Files(root string) (map[string]int, error)

// Lines returns the total number of lines of the counted files per extension, using the
// Lines rule above. Only extensions that appear in the map are present, as for Files.
func Lines(root string) (map[string]int, error)

// Tokens returns the total number of tokens over all counted files, using the comment and
// token rules above.
func Tokens(root string) (int, error)

// Top returns up to n counted files with the most tokens, sorted by Tokens descending, then
// by Path ascending (byte order). Counted files with zero tokens are included. n <= 0 returns
// an empty slice.
func Top(root string, n int) ([]FileTokens, error)
```

Errors: an unreadable `root` or file returns a non-nil error. Results are deterministic for a
fixed tree.
