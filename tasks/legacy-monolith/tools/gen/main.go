// Command gen writes the shopcore seed workspace.
//
// Usage: go run . -out <dir>
//
// Output is byte-for-byte deterministic: every random choice comes from a
// fixed per-package seed and files are written in sorted order.
package main

import (
	"embed"
	"flag"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

//go:embed all:static
var static embed.FS

func main() {
	out := flag.String("out", "", "output directory (replaced)")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "usage: gen -out <dir>")
		os.Exit(2)
	}
	files, err := build()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	if err := os.RemoveAll(*out); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		full := filepath.Join(*out, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "gen:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(full, []byte(files[p]), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "gen:", err)
			os.Exit(1)
		}
	}
}

func build() (map[string]string, error) {
	files := map[string]string{}
	add := func(path, content string) error {
		if _, dup := files[path]; dup {
			return fmt.Errorf("duplicate output file %s", path)
		}
		files[path] = content
		return nil
	}
	for _, p := range pkgs {
		r := rngFor(p.Name)
		st := styleFor(p, r)
		if err := add(p.Name+"/errors.go", genErrors(p, st)); err != nil {
			return nil, err
		}
		utilName := []string{"util.go", "wire.go", "helpers.go"}[r.Intn(3)]
		if err := add(p.Name+"/"+utilName, genUtil(p, st, r)); err != nil {
			return nil, err
		}
		for _, e := range p.Entities {
			x := &ectx{p: p, e: e, st: st, sn: storeFor(p.Style), r: r, low: lowerFirst(e.Name), hum: words(e.Name)}
			base := snake(e.Name)
			rf := reportFlags{csv: r.Intn(3) == 0, diff: r.Intn(2) == 0}
			storeFile := []string{base + "_store.go", base + "_repo.go", "store_" + base + ".go"}[r.Intn(3)]
			for _, f := range []struct{ name, body string }{
				{base + ".go", genModel(x)},
				{storeFile, genStore(x)},
				{base + "_codec.go", genCodec(x)},
				{base + "_report.go", genReport(x, rf)},
				{base + "_test.go", genTest(x, rf)},
			} {
				if err := add(p.Name+"/"+f.name, f.body); err != nil {
					return nil, err
				}
			}
		}
	}
	for path, body := range tables() {
		if err := add(path, body); err != nil {
			return nil, err
		}
	}
	err := fs.WalkDir(static, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := static.ReadFile(path)
		if err != nil {
			return err
		}
		rel := strings.TrimSuffix(strings.TrimPrefix(path, "static/"), ".tmpl")
		return add(rel, string(data))
	})
	if err != nil {
		return nil, err
	}
	for path, body := range files {
		if !strings.HasSuffix(path, ".go") {
			continue
		}
		body = finalizeImports(body)
		src, err := format.Source([]byte(body))
		if err != nil {
			return nil, fmt.Errorf("%s: %v\n%s", path, err, numbered(body))
		}
		files[path] = string(src)
	}
	return files, nil
}

var importCandidates = []struct{ path, ident string }{
	{"bytes", "bytes"}, {"context", "context"}, {"encoding/csv", "csv"}, {"errors", "errors"},
	{"fmt", "fmt"}, {"io", "io"}, {"sort", "sort"}, {"strconv", "strconv"}, {"strings", "strings"},
	{"sync", "sync"}, {"testing", "testing"}, {"time", "time"},
	{"shopcore/money", "money"},
}

var lineComment = regexp.MustCompile(`//[^\n]*`)

// finalizeImports replaces the @@IMPORTS@@ marker with the packages the file
// actually references.
func finalizeImports(src string) string {
	if !strings.Contains(src, "@@IMPORTS@@") {
		return src
	}
	code := lineComment.ReplaceAllString(src, "")
	var std, local []string
	for _, c := range importCandidates {
		if regexp.MustCompile(`\b` + c.ident + `\.`).MatchString(code) {
			if strings.HasPrefix(c.path, "shopcore/") {
				local = append(local, c.path)
			} else {
				std = append(std, c.path)
			}
		}
	}
	var b strings.Builder
	if len(std)+len(local) > 0 {
		b.WriteString("import (\n")
		for _, s := range std {
			fmt.Fprintf(&b, "\t%q\n", s)
		}
		if len(std) > 0 && len(local) > 0 {
			b.WriteString("\n")
		}
		for _, s := range local {
			fmt.Fprintf(&b, "\t%q\n", s)
		}
		b.WriteString(")\n")
	}
	return strings.Replace(src, "@@IMPORTS@@\n", b.String(), 1)
}

func numbered(s string) string {
	var b strings.Builder
	for n, l := range strings.Split(s, "\n") {
		fmt.Fprintf(&b, "%4d %s\n", n+1, l)
	}
	return b.String()
}
