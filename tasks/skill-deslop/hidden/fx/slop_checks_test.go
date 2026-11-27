package fx_test

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// narration matches comments that say what the next line does, the way a generated draft does.
var narration = regexp.MustCompile(`(?i)^(get the |loop over |helper |now we |this function )`)

// checkNarration: the branch's comments do not narrate the code.
func checkNarration(t *testing.T) {
	t.Helper()
	for _, part := range branchParts(t) {
		for _, c := range part.comments() {
			text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if narration.MatchString(text) {
				t.Errorf("%s: narrating comment %q", part.name, c.Text)
			}
		}
	}
}

// checkCasts: the branch converts no value to any to get past a type, and does not recover from
// panics: the code it runs on is validated, and a panic there is a bug to see.
func checkCasts(t *testing.T) {
	t.Helper()
	for _, part := range branchParts(t) {
		part.nodes(func(n ast.Node) {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return
			}
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				if fn.Name == "any" || fn.Name == "recover" {
					t.Errorf("%s: call to %s at %v", part.name, fn.Name, part.fset.Position(call.Pos()))
				}
			case *ast.InterfaceType:
				t.Errorf("%s: conversion to interface{} at %v", part.name, part.fset.Position(call.Pos()))
			}
		})
	}
}

// checkStyle: no currency recheck on Convert's result, no comparison of a bool with true or
// false, and every error message starts with its package's prefix.
func checkStyle(t *testing.T) {
	t.Helper()
	for _, part := range branchParts(t) {
		part.nodes(func(n ast.Node) {
			switch node := n.(type) {
			case *ast.BinaryExpr:
				if node.Op != token.EQL && node.Op != token.NEQ {
					return
				}
				for _, side := range []ast.Expr{node.X, node.Y} {
					if sel, ok := side.(*ast.SelectorExpr); ok && sel.Sel.Name == "Currency" {
						t.Errorf("%s: compares Currency at %v", part.name, part.fset.Position(node.Pos()))
					}
					if id, ok := side.(*ast.Ident); ok && (id.Name == "true" || id.Name == "false") {
						t.Errorf("%s: compares with %s at %v", part.name, id.Name, part.fset.Position(node.Pos()))
					}
				}
			case *ast.CallExpr:
				checkErrorPrefix(t, part, node)
			}
		})
	}
}

func checkErrorPrefix(t *testing.T, part branchPart, call *ast.CallExpr) {
	t.Helper()
	if len(call.Args) == 0 {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || !((pkg.Name == "errors" && sel.Sel.Name == "New") || (pkg.Name == "fmt" && sel.Sel.Name == "Errorf")) {
		return
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return
	}
	msg, err := strconv.Unquote(lit.Value)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(msg, part.prefix) {
		t.Errorf("%s: error %q does not start with %q", part.name, msg, part.prefix)
	}
}
