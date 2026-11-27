// Package corpusstats reports statistics over a tree of text files. See ../SPEC.md.
package corpusstats

// FileTokens is one counted file: its path relative to root, slash-separated, and its token count.
type FileTokens struct {
	Path   string
	Tokens int
}

// Files returns the number of counted files per extension. See ../SPEC.md.
func Files(root string) (map[string]int, error) {
	return nil, nil
}

// Lines returns the total number of lines of the counted files per extension. See ../SPEC.md.
func Lines(root string) (map[string]int, error) {
	return nil, nil
}

// Tokens returns the total number of tokens over all counted files. See ../SPEC.md.
func Tokens(root string) (int, error) {
	return 0, nil
}

// Top returns up to n counted files with the most tokens. See ../SPEC.md.
func Top(root string, n int) ([]FileTokens, error) {
	return nil, nil
}
