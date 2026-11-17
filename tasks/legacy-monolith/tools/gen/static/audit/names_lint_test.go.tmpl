package audit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// grandfathered are names that were written to audit_events before the
// importer enforced the schema. The importer keeps a mapping for exactly
// these; nothing may be added to this list.
var grandfathered = map[string]bool{
	"invoice_created": true, "charge_captured": true, "charge_capture_failed": true,
	"invoice_void_rejected": true, "invoice_voided": true,
	"warehouse.allocate": true, "warehouse.block": true,
	"suppliers.onboard": true, "suppliers.suspend": true, "suppliers.approve": true, "suppliers.receive": true,
}

// TestEventNamesFollowSchema scans the module for event names: string
// literals or constants used as audit.Event Name, and package-level
// constants named ev<Something>.
func TestEventNamesFollowSchema(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	byDir := map[string][]*ast.File{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		byDir[filepath.Dir(path)] = append(byDir[filepath.Dir(path)], f)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, files := range byDir {
		consts := map[string]string{}
		for _, f := range files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.CONST {
					continue
				}
				for _, spec := range gd.Specs {
					vs := spec.(*ast.ValueSpec)
					for i, n := range vs.Names {
						if i < len(vs.Values) {
							if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
								v, _ := strconv.Unquote(lit.Value)
								consts[n.Name] = v
								if len(n.Name) > 2 && strings.HasPrefix(n.Name, "ev") && n.Name[2] >= 'A' && n.Name[2] <= 'Z' {
									seen[v] = fset.Position(n.Pos()).String()
								}
							}
						}
					}
				}
			}
		}
		for _, f := range files {
			for _, decl := range f.Decls {
				scope := consts
				if fd, ok := decl.(*ast.FuncDecl); ok && fd.Body != nil {
					scope = withLocalConsts(consts, fd.Body)
				}
				ast.Inspect(decl, func(n ast.Node) bool {
					cl, ok := n.(*ast.CompositeLit)
					if !ok {
						return true
					}
					sel, ok := cl.Type.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "Event" {
						return true
					}
					for _, el := range cl.Elts {
						kv, ok := el.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						if k, ok := kv.Key.(*ast.Ident); !ok || k.Name != "Name" {
							continue
						}
						switch v := kv.Value.(type) {
						case *ast.BasicLit:
							s, _ := strconv.Unquote(v.Value)
							seen[s] = fset.Position(v.Pos()).String()
						case *ast.Ident:
							if s, ok := scope[v.Name]; ok {
								seen[s] = fset.Position(v.Pos()).String()
							}
						}
					}
					return true
				})
			}
		}
	}
	if len(seen) < 20 {
		t.Fatalf("found only %d event names; the scan is broken", len(seen))
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if grandfathered[n] {
			continue
		}
		if err := ValidateName(n); err != nil {
			t.Errorf("%s: %v", seen[n], err)
		}
	}
}

// withLocalConsts returns pkg overlaid with the string constants declared
// inside body (for example a function's op name).
func withLocalConsts(pkg map[string]string, body *ast.BlockStmt) map[string]string {
	out := make(map[string]string, len(pkg))
	for k, v := range pkg {
		out[k] = v
	}
	ast.Inspect(body, func(n ast.Node) bool {
		gd, ok := n.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			return true
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, name := range vs.Names {
				if i < len(vs.Values) {
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						out[name.Name], _ = strconv.Unquote(lit.Value)
					}
				}
			}
		}
		return true
	})
	return out
}
