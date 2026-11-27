package fx_test

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

// TestM4BranchClean: the branch's own code is clean. It fails on the seed while the slop is there.
func TestM4BranchClean(t *testing.T) {
	checkNarration(t)
	checkCasts(t)
	checkStyle(t)
}

// The branch does not touch money/money.go, nor ParseLegacy in invoice.go, which were on main.
// Their bytes must not change, comments included.
const (
	moneyFileSHA256      = "277bfbc32396a08656a261412c0e1d7e6f2fc3ed9d1870f1d120cc38923acfaa"
	parseLegacySourceSHA = "04e445c8a0316797792676ea3939b8bcee889e85b371656957d15a82fbaccb08"
)

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// TestM4LegacyUntouched: code the branch did not change is byte-identical to main.
func TestM4LegacyUntouched(t *testing.T) {
	money, err := os.ReadFile("../money/money.go")
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256Hex(money); got != moneyFileSHA256 {
		t.Errorf("money/money.go changed: sha256 %s", got)
	}

	src, err := os.ReadFile("../invoice/invoice.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "invoice.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "ParseLegacy" {
			continue
		}
		start := fn.Pos()
		if fn.Doc != nil {
			start = fn.Doc.Pos()
		}
		text := src[fset.Position(start).Offset:fset.Position(fn.End()).Offset]
		if got := sha256Hex(text); got != parseLegacySourceSHA {
			t.Errorf("ParseLegacy changed: sha256 %s", got)
		}
		return
	}
	t.Fatal("invoice.go has no ParseLegacy function")
}
