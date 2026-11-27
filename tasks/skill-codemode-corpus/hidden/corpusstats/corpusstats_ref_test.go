package corpusstats

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// refFile is one counted file, read by the reference implementation below.
type refFile struct {
	rel  string // slash-separated, relative to the walked root
	ext  string
	data []byte
}

var refExts = []string{".txt", ".md", ".go"}

// refWalk is the reference reading of the spec's "counted file": a regular file at any depth
// whose name ends in one of refExts (case-sensitive).
func refWalk(t *testing.T, root string) []refFile {
	t.Helper()
	var out []refFile
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		ext := ""
		for _, e := range refExts {
			if strings.HasSuffix(d.Name(), e) {
				ext = e
			}
		}
		if ext == "" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, refFile{rel: filepath.ToSlash(rel), ext: ext, data: data})
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return out
}

func refLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	n := bytes.Count(data, []byte("\n"))
	if data[len(data)-1] != '\n' {
		n++
	}
	return n
}

func refTokens(ext string, data []byte) int {
	n := 0
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if ext == ".go" && strings.HasPrefix(trimmed, "//") {
			continue
		}
		if ext != ".go" && strings.HasPrefix(trimmed, "#") {
			continue
		}
		inRun := false
		for i := 0; i < len(line); i++ {
			c := line[i]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '\'' {
				if !inRun {
					n++
					inRun = true
				}
			} else {
				inRun = false
			}
		}
	}
	return n
}

func refTop(t *testing.T, root string) []FileTokens {
	t.Helper()
	var all []FileTokens
	for _, f := range refWalk(t, root) {
		all = append(all, FileTokens{Path: f.rel, Tokens: refTokens(f.ext, f.data)})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Tokens != all[j].Tokens {
			return all[i].Tokens > all[j].Tokens
		}
		return all[i].Path < all[j].Path
	})
	return all
}

// sameCounts compares two per-extension maps by their keys and values only, so a nil map and
// an empty map are equal.
func sameCounts(got, want map[string]int) bool {
	if len(got) != len(want) {
		return false
	}
	for k, v := range want {
		g, ok := got[k]
		if !ok || g != v {
			return false
		}
	}
	return true
}

// writeFixture writes files (slash-separated relative paths) with the given contents under a
// new temporary root and returns the root.
func writeFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
