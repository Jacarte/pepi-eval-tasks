package acceptance

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"payments/internal/xlog"
)

var componentRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// logged runs fn and returns the xlog lines it wrote.
func logged(t *testing.T, fn func()) []map[string]any {
	t.Helper()
	var buf bytes.Buffer
	xlog.SetOutput(&buf)
	defer xlog.SetOutput(io.Discard)
	fn()
	var lines []map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if raw == "" {
			continue
		}
		var line map[string]any
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			t.Fatalf("xlog line is not JSON: %q", raw)
		}
		lines = append(lines, line)
	}
	return lines
}

func requireComponents(t *testing.T, what string, lines []map[string]any) {
	t.Helper()
	if len(lines) == 0 {
		t.Fatalf("%s: nothing was logged through xlog", what)
	}
	for _, l := range lines {
		if c, _ := l["component"].(string); !componentRe.MatchString(c) {
			t.Fatalf("%s: xlog line %v has no lowercase component", what, l)
		}
	}
}

func TestM3RefundOutcomesAreLogged(t *testing.T) {
	e := newEnv(t)
	id := e.payment(1000, true)
	requireComponents(t, "created refund", logged(t, func() {
		e.mustCall(201, "POST", "/refunds", refundBody(id, 400))
	}))
	requireComponents(t, "rejected refund (over the remaining amount)", logged(t, func() {
		e.mustCall(422, "POST", "/refunds", refundBody(id, 601))
	}))
	requireComponents(t, "rejected refund (unknown payment)", logged(t, func() {
		e.mustCall(404, "POST", "/refunds", refundBody("pay_999", 1))
	}))
}

func TestM3NoStdlibLoggingUnderInternal(t *testing.T) {
	fset := token.NewFileSet()
	err := filepath.WalkDir("../internal", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && filepath.Base(path) == "xlog" {
			return fs.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == "log" || p == "log/slog" {
				t.Errorf("%s imports %s; use internal/xlog", path, p)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				if fn.Name == "print" || fn.Name == "println" {
					t.Errorf("%s: builtin %s", fset.Position(call.Pos()), fn.Name)
				}
			case *ast.SelectorExpr:
				if pkg, ok := fn.X.(*ast.Ident); ok && pkg.Name == "fmt" {
					switch {
					case fn.Sel.Name == "Print" || fn.Sel.Name == "Printf" || fn.Sel.Name == "Println":
						t.Errorf("%s: fmt.%s; use internal/xlog", fset.Position(call.Pos()), fn.Sel.Name)
					case strings.HasPrefix(fn.Sel.Name, "Fprint") && len(call.Args) > 0 && isStdStream(call.Args[0]):
						t.Errorf("%s: fmt.%s to a std stream; use internal/xlog", fset.Position(call.Pos()), fn.Sel.Name)
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func isStdStream(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "os" && (sel.Sel.Name == "Stdout" || sel.Sel.Name == "Stderr")
}
