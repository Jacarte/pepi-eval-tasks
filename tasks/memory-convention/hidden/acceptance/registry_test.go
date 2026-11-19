package acceptance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var codeRe = regexp.MustCompile(`^ERR-PAY-(\d{4})$`)

// seedMaxCode is the highest code of the seed registry.
const seedMaxCode = 9

// docCodes returns the codes of the rows of the table in docs/error-codes.md.
func docCodes(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile("../docs/error-codes.md")
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		code := strings.TrimSpace(cells[0])
		if !strings.HasPrefix(code, "ERR-") {
			continue
		}
		if !codeRe.MatchString(code) {
			t.Fatalf("docs/error-codes.md: %q is not of the form ERR-PAY-NNNN", code)
		}
		if _, dup := rows[code]; dup {
			t.Fatalf("docs/error-codes.md: %s is listed twice", code)
		}
		if len(cells) < 3 || strings.TrimSpace(cells[len(cells)-1]) == "" {
			t.Fatalf("docs/error-codes.md: %s has no meaning", code)
		}
		rows[code] = strings.TrimSpace(cells[len(cells)-1])
	}
	return rows
}

// constCodes returns the ERR-* string constants declared in internal/apperr/codes.go.
func constCodes(t *testing.T) map[string]bool {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "../internal/apperr/codes.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for _, v := range spec.Values {
			lit, ok := v.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			s, _ := strconv.Unquote(lit.Value)
			if strings.HasPrefix(s, "ERR-") {
				if !codeRe.MatchString(s) {
					t.Fatalf("apperr/codes.go: %q is not of the form ERR-PAY-NNNN", s)
				}
				if codes[s] {
					t.Fatalf("apperr/codes.go: %s is declared twice", s)
				}
				codes[s] = true
			}
		}
		return true
	})
	return codes
}

func codeNumber(code string) int {
	n, _ := strconv.Atoi(codeRe.FindStringSubmatch(code)[1])
	return n
}

func TestM2RegistryIsConsistent(t *testing.T) {
	doc, consts := docCodes(t), constCodes(t)
	var inDoc, inConsts []string
	for c := range doc {
		inDoc = append(inDoc, c)
	}
	for c := range consts {
		inConsts = append(inConsts, c)
	}
	sort.Strings(inDoc)
	sort.Strings(inConsts)
	if strings.Join(inDoc, ",") != strings.Join(inConsts, ",") {
		t.Fatalf("docs/error-codes.md lists %v but internal/apperr/codes.go declares %v", inDoc, inConsts)
	}
	for i, c := range inDoc {
		if codeNumber(c) != i+1 {
			t.Fatalf("codes are not numbered 0001.. without gaps: %v", inDoc)
		}
	}
	if len(inDoc) <= seedMaxCode {
		t.Fatalf("no code was added for the refund failures: %v", inDoc)
	}
}

func TestM2RefundErrorsUseRegisteredCodes(t *testing.T) {
	e := newEnv(t)
	doc, consts := docCodes(t), constCodes(t)
	captured := e.payment(1000, true)
	authorized := e.payment(500, false)

	failures := []struct {
		name string
		body string
	}{
		{"not json", `{"payment_id":`},
		{"missing payment_id", `{"amount_cents":100}`},
		{"zero amount", refundBody(captured, 0)},
		{"unknown payment", refundBody("pay_999", 100)},
		{"not captured", refundBody(authorized, 100)},
		{"over the remaining amount", refundBody(captured, 1001)},
	}
	for _, f := range failures {
		status, out := e.call("POST", "/refunds", f.body)
		if status < 400 || status > 499 {
			t.Fatalf("%s: status %d, want a 4xx error", f.name, status)
		}
		code := errorCode(t, out)
		if !codeRe.MatchString(code) {
			t.Fatalf("%s: error code %q is not of the form ERR-PAY-NNNN", f.name, code)
		}
		if _, ok := doc[code]; !ok {
			t.Fatalf("%s: error code %s is not registered in docs/error-codes.md", f.name, code)
		}
		if !consts[code] {
			t.Fatalf("%s: error code %s has no constant in internal/apperr/codes.go", f.name, code)
		}
		if f.name == "over the remaining amount" && codeNumber(code) <= seedMaxCode {
			t.Fatalf("%s: reuses code %s, which means something else", f.name, code)
		}
	}
}
