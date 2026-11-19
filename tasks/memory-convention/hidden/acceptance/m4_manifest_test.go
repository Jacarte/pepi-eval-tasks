package acceptance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"payments/internal/api"
)

// isHandlerSignature matches func(http.ResponseWriter, *http.Request).
func isHandlerSignature(ft *ast.FuncType) bool {
	if ft.Params == nil || len(ft.Params.List) != 2 || ft.Results != nil && len(ft.Results.List) > 0 {
		return false
	}
	w, ok := ft.Params.List[0].Type.(*ast.SelectorExpr)
	if !ok || w.Sel.Name != "ResponseWriter" {
		return false
	}
	star, ok := ft.Params.List[1].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	r, ok := star.X.(*ast.SelectorExpr)
	return ok && r.Sel.Name == "Request"
}

// exportedHandlers lists the exported functions and methods of internal/api that look like handlers. (ServeHTTP is the http.Handler method, not a route).
func exportedHandlers(t *testing.T) []string {
	t.Helper()
	var names []string
	fset := token.NewFileSet()
	err := filepath.WalkDir("../internal/api", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.IsExported() && fn.Name.Name != "ServeHTTP" && isHandlerSignature(fn.Type) {
				names = append(names, fn.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	return names
}

func TestM4ManifestListsEveryHandler(t *testing.T) {
	listed := map[string]bool{}
	routes := map[string]bool{}
	for _, r := range api.Manifest {
		if r.Method == "" || r.Path == "" || r.Handler == "" {
			t.Fatalf("incomplete manifest entry: %+v", r)
		}
		if key := r.Method + " " + r.Path; routes[key] {
			t.Fatalf("route %s is listed twice", key)
		} else {
			routes[key] = true
		}
		if listed[r.Handler] {
			t.Fatalf("handler %s is listed twice", r.Handler)
		}
		listed[r.Handler] = true
	}
	handlers := exportedHandlers(t)
	for _, h := range handlers {
		if !listed[h] {
			t.Errorf("exported handler %s is not in the routes manifest", h)
		}
	}
	for h := range listed {
		found := false
		for _, name := range handlers {
			found = found || name == h
		}
		if !found {
			t.Errorf("manifest lists %s, which is not an exported handler of internal/api", h)
		}
	}
}

func TestM4ManifestHasRefunds(t *testing.T) {
	handlers := map[string]bool{}
	for _, h := range exportedHandlers(t) {
		handlers[h] = true
	}
	for _, r := range api.Manifest {
		if r.Method == "POST" && r.Path == "/refunds" {
			if !handlers[r.Handler] {
				t.Fatalf("POST /refunds names %s, which is not an exported handler", r.Handler)
			}
			return
		}
	}
	t.Fatal("POST /refunds is not in the routes manifest")
}
