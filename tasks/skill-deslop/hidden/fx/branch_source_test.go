package fx_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// branchPart is source the branch added: a whole file of this package, or the Print function of
// ../invoice. Checks see only the nodes and comments inside [lo, hi).
type branchPart struct {
	name   string
	file   *ast.File
	fset   *token.FileSet
	lo, hi token.Pos
	// prefix is what every error message of the part starts with.
	prefix string
}

// branchParts returns the non-test files of this package (prefix "fx: ") and the Print function
// of ../invoice (prefix "invoice: "), the code the branch wrote.
func branchParts(t *testing.T) []branchPart {
	t.Helper()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var parts []branchPart
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, branchPart{name: path, file: file, fset: fset, lo: file.Pos(), hi: file.End(), prefix: "fx: "})
	}
	return append(parts, printPart(t))
}

func printPart(t *testing.T) branchPart {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "../invoice/invoice.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "Print" {
			return branchPart{name: "invoice.go Print", file: file, fset: fset, lo: fn.Pos(), hi: fn.End(), prefix: "invoice: "}
		}
	}
	t.Fatal("invoice.go has no Print function")
	return branchPart{}
}

// nodes calls f for every node of the part that lies inside its range.
func (p branchPart) nodes(f func(ast.Node)) {
	ast.Inspect(p.file, func(n ast.Node) bool {
		if n != nil && n.Pos() >= p.lo && n.End() <= p.hi {
			f(n)
		}
		return true
	})
}

// comments returns the comments of the part, with their "//" markers.
func (p branchPart) comments() []*ast.Comment {
	var out []*ast.Comment
	for _, group := range p.file.Comments {
		for _, c := range group.List {
			if c.Pos() >= p.lo && c.End() <= p.hi {
				out = append(out, c)
			}
		}
	}
	return out
}
