package regexlite_test

import (
	"errors"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"twelvemods/regexlite"
)

func m6Compile(t *testing.T, pattern string) *regexlite.Regexp {
	t.Helper()
	re, err := regexlite.Compile(pattern)
	if err != nil {
		t.Fatalf("Compile(%q): %v", pattern, err)
	}
	if re == nil {
		t.Fatalf("Compile(%q) returned a nil *Regexp and no error", pattern)
	}
	return re
}

func TestM6SubmatchIndex(t *testing.T) {
	cases := []struct {
		pattern, s string
		want       []int
	}{
		{`abc`, "xxabcxx", []int{2, 5}},
		{`a.c`, "a\nc abc", []int{4, 7}},
		{`a|ab`, "ab", []int{0, 1}},
		{`ab|a`, "ab", []int{0, 2}},
		{`a*`, "baaa", []int{0, 0}},
		{`a+`, "baaa", []int{1, 4}},
		{`a+?`, "baaa", []int{1, 2}},
		{`a*?b`, "aaab", []int{0, 4}},
		{`<.+>`, "<a><b>", []int{0, 6}},
		{`<.+?>`, "<a><b>", []int{0, 3}},
		{`a{2}`, "aaaa", []int{0, 2}},
		{`a{2,}`, "aaaa", []int{0, 4}},
		{`a{2,3}`, "aaaa", []int{0, 3}},
		{`a{2,3}?`, "aaaa", []int{0, 2}},
		{`a{0}b`, "ab", []int{1, 2}},
		{`x{1,2}y`, "xxxy", []int{1, 4}},
		{`a{3}`, "aa", nil},
		{`^abc`, "abcabc", []int{0, 3}},
		{`^b`, "ab", nil},
		{`c$`, "abcabc", []int{5, 6}},
		{`^$`, "", []int{0, 0}},
		{`^$`, "\n", nil},
		{`a$`, "a\n", nil},
		{`[a-c]+`, "xxbcaz", []int{2, 5}},
		{`[^a-c]+`, "abxyzc", []int{2, 5}},
		{`[^a]`, "a\n", []int{1, 2}},
		{`[]a]+`, "x]a]", []int{1, 4}},
		{`[a-]+`, "x-a-", []int{1, 4}},
		{`[-a]+`, "x-a-", []int{1, 4}},
		{`[\]]`, "a]", []int{1, 2}},
		{`[a\-z]+`, "b-az", []int{1, 4}},
		{`[\d.]+`, "v1.25", []int{1, 5}},
		{`[^\d\s]+`, "12 ab3", []int{3, 5}},
		{`[\w-]+`, "--a-b c", []int{0, 5}},
		{`[a-c-e]+`, "xd-eab", []int{2, 6}},
		{`\d+`, "abc123def", []int{3, 6}},
		{`\D+`, "12ab34", []int{2, 4}},
		{`\w+`, "  foo_bar9!", []int{2, 10}},
		{`\W+`, "ab, cd", []int{2, 4}},
		{`\s+`, "a \t\n\r\f\vb", []int{1, 7}},
		{`\S+`, "  xy ", []int{2, 4}},
		{`\.`, "a.b", []int{1, 2}},
		{`\*\+\?\(\)\[\]\{\}\|\^\$\\`, `*+?()[]{}|^$\`, []int{0, 13}},
		{`a\nb\tc`, "a\nb\tc", []int{0, 5}},
		{`a]b}`, "a]b}", []int{0, 4}},
		{`colou?r`, "my color", []int{3, 8}},
		{`(?:ab){2}`, "abababab", []int{0, 4}},
		{`a.*b`, "axxbyyb", []int{0, 7}},
		{`a.*?b`, "axxbyyb", []int{0, 4}},
		{`^(?:a|ab)c`, "abc", []int{0, 3}},
		{`x*`, "", []int{0, 0}},
		{``, "abc", []int{0, 0}},
		{`a|`, "b", []int{0, 0}},
		{`é+`, "caféé!", []int{3, 7}},
		{`.`, "日本", []int{0, 3}},
		{`[à-ÿ]+`, "naïve", []int{2, 4}},
		{`[^a]b`, "日b", []int{0, 4}},
		// groups
		{`()`, "x", []int{0, 0, 0, 0}},
		{`(a)(b)?`, "a", []int{0, 1, 0, 1, -1, -1}},
		{`(a)|b`, "b", []int{0, 1, -1, -1}},
		{`(a|b)*`, "abba", []int{0, 4, 3, 4}},
		{`(?:a|b)*c`, "ababc", []int{0, 5}},
		{`(a(b(c)))`, "abc", []int{0, 3, 0, 3, 1, 3, 2, 3}},
		{`(\w+)@(\w+)\.com`, "mail bob@example.com now", []int{5, 20, 5, 8, 9, 16}},
		{`(a+)(a+)`, "aaaa", []int{0, 4, 0, 3, 3, 4}},
		{`(a+?)(a+)`, "aaaa", []int{0, 4, 0, 1, 1, 4}},
		{`(a|ab)(c|bcd)(d*)`, "abcd", []int{0, 4, 0, 1, 1, 4, 4, 4}},
		{`\$\d+(\.\d\d)?`, "cost $15.5", []int{5, 8, -1, -1}},
		{`(ab){2,}?c`, "ababc", []int{0, 5, 2, 4}},
		{`(?:(a)|b)+`, "ab", []int{0, 2, 0, 1}},
		{`(a)+`, "aaa", []int{0, 3, 2, 3}},
		// empty iterations
		{`(a*)+`, "b", []int{0, 0, 0, 0}},
		{`(a*)*`, "b", []int{0, 0, -1, -1}},
		{`(a|)+`, "aab", []int{0, 2, 1, 2}},
		{`(a*){2}`, "b", []int{0, 0, 0, 0}},
		{`(?:a?)*b`, "aab", []int{0, 3}},
	}
	for _, c := range cases {
		re, err := regexlite.Compile(c.pattern)
		if err != nil {
			t.Errorf("Compile(%q): %v", c.pattern, err)
			continue
		}
		if got := re.FindStringSubmatchIndex(c.s); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q.FindStringSubmatchIndex(%q) = %v, want %v", c.pattern, c.s, got, c.want)
		}
		if got := re.MatchString(c.s); got != (c.want != nil) {
			t.Errorf("%q.MatchString(%q) = %v, want %v", c.pattern, c.s, got, c.want != nil)
		}
	}
}

func TestM6Submatch(t *testing.T) {
	cases := []struct {
		pattern, s string
		want       []string
	}{
		{`(a)|(b)`, "b", []string{"b", "", "b"}},
		{`(x)?y`, "y", []string{"y", ""}},
		{`(\d+)-(\d+)`, "on 10-20!", []string{"10-20", "10", "20"}},
		{`z`, "abc", nil},
		{`(é)(.)`, "aéb", []string{"éb", "é", "b"}},
	}
	for _, c := range cases {
		re := m6Compile(t, c.pattern)
		if got := re.FindStringSubmatch(c.s); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q.FindStringSubmatch(%q) = %q, want %q", c.pattern, c.s, got, c.want)
		}
	}
}

func TestM6NumGroups(t *testing.T) {
	cases := []struct {
		pattern string
		want    int
	}{
		{`a(b)(?:c)(d(e))`, 3},
		{``, 0},
		{`\(`, 0},
		{`[(]`, 0},
		{`((((a))))`, 4},
		{`(?:(?:a))`, 0},
	}
	for _, c := range cases {
		if got := m6Compile(t, c.pattern).NumGroups(); got != c.want {
			t.Errorf("%q.NumGroups() = %d, want %d", c.pattern, got, c.want)
		}
	}
}

func TestM6SyntaxErrors(t *testing.T) {
	cases := []struct {
		pattern string
		pos     int
		msg     string
	}{
		{`\`, 0, "trailing backslash"},
		{`ab\`, 2, "trailing backslash"},
		{`[a\`, 2, "trailing backslash"},
		{`\q`, 0, "invalid escape"},
		{`a\1`, 1, "invalid escape"},
		{`[\b]`, 1, "invalid escape"},
		{`\é`, 0, "invalid escape"},
		{`[abc`, 0, "missing closing ]"},
		{`x[]`, 1, "missing closing ]"},
		{`[^]`, 0, "missing closing ]"},
		{`[z-a]`, 1, "invalid character class range"},
		{`[a-\d]`, 1, "invalid character class range"},
		{`[x\d-z]`, 2, "invalid character class range"},
		{`(ab`, 0, "missing closing )"},
		{`a(b(c)`, 1, "missing closing )"},
		{`((a)`, 0, "missing closing )"},
		{`((a`, 1, "missing closing )"},
		{`a)`, 1, "unexpected )"},
		{`(a))`, 3, "unexpected )"},
		{`)`, 0, "unexpected )"},
		{`(?i)a`, 0, "invalid group"},
		{`x(?`, 1, "invalid group"},
		{`*a`, 0, "missing argument to repetition operator"},
		{`a|*`, 2, "missing argument to repetition operator"},
		{`(+)`, 1, "missing argument to repetition operator"},
		{`^*`, 1, "missing argument to repetition operator"},
		{`a$?`, 2, "missing argument to repetition operator"},
		{`{2}`, 0, "missing argument to repetition operator"},
		{`(?:*)`, 3, "missing argument to repetition operator"},
		{`a**`, 2, "invalid nested repetition operator"},
		{`a*??`, 3, "invalid nested repetition operator"},
		{`a+*`, 2, "invalid nested repetition operator"},
		{`a{2}{3}`, 4, "invalid nested repetition operator"},
		{`a?{1}`, 2, "invalid nested repetition operator"},
		{`a{`, 1, "invalid repetition"},
		{`a{x}`, 1, "invalid repetition"},
		{`a{,3}`, 1, "invalid repetition"},
		{`a{2`, 1, "invalid repetition"},
		{`a{2,`, 1, "invalid repetition"},
		{`a{2,3`, 1, "invalid repetition"},
		{`a{3,2}`, 1, "invalid repetition"},
		{`a{1001}`, 1, "invalid repetition"},
		{`a{1,1001}`, 1, "invalid repetition"},
		{`a{99999999999999999999}`, 1, "invalid repetition"},
		{`ab{1 }`, 2, "invalid repetition"},
		{`é{`, 2, "invalid repetition"},
	}
	for _, c := range cases {
		re, err := regexlite.Compile(c.pattern)
		var se *regexlite.SyntaxError
		if !errors.As(err, &se) {
			t.Errorf("Compile(%q) error = %v, want *SyntaxError", c.pattern, err)
			continue
		}
		if re != nil {
			t.Errorf("Compile(%q) returned a non-nil *Regexp with an error", c.pattern)
		}
		if se.Pos != c.pos || se.Msg != c.msg {
			t.Errorf("Compile(%q) = {Pos %d, Msg %q}, want {Pos %d, Msg %q}", c.pattern, se.Pos, se.Msg, c.pos, c.msg)
		}
		want := "regexlite: " + c.msg + " at position " + strconv.Itoa(c.pos)
		if se.Error() != want {
			t.Errorf("Compile(%q).Error() = %q, want %q", c.pattern, se.Error(), want)
		}
	}
	for _, ok := range []string{`a{1000}`, `a{0,1000}`, `]`, `}`, `a{0}`, `a{2,}?`, `[]]`, `[\]-a]`, `(|)`, `$^`} {
		if _, err := regexlite.Compile(ok); err != nil {
			t.Errorf("Compile(%q) = %v, want no error", ok, err)
		}
	}
}

func TestM6FindAll(t *testing.T) {
	cases := []struct {
		pattern, s string
		n          int
		want       []string
	}{
		{`a*`, "baaac", -1, []string{"", "aaa", "", ""}},
		{`\d+`, "a1b22c333", -1, []string{"1", "22", "333"}},
		{`\d+`, "a1b22c333", 2, []string{"1", "22"}},
		{`\d+`, "a1b22c333", 0, nil},
		{`x`, "abc", -1, nil},
		{``, "日本", -1, []string{"", "", ""}},
		{`a|b`, "abc", -1, []string{"a", "b"}},
		{`\w+?`, "ab", -1, []string{"a", "b"}},
		{`^a`, "aaa", -1, []string{"a"}},
		{`a$`, "aaa", -1, []string{"a"}},
		{`.`, "a\nb", -1, []string{"a", "b"}},
		{`(a)(b)`, "abab", -1, []string{"ab", "ab"}},
		{`b*`, "abb", -1, []string{"", "bb", ""}},
	}
	for _, c := range cases {
		re := m6Compile(t, c.pattern)
		if got := re.FindAllString(c.s, c.n); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q.FindAllString(%q, %d) = %q, want %q", c.pattern, c.s, c.n, got, c.want)
		}
	}
}

func TestM6ReplaceAll(t *testing.T) {
	cases := []struct {
		pattern, src, repl, want string
	}{
		{`a(\d)`, "a1 a2 b3", "<$1>", "<1> <2> b3"},
		{`(\w+)@(\w+)`, "x bob@host y", "$2 at $1", "x host at bob y"},
		{`a`, "banana", "$$", "b$n$n$"},
		{`(a)`, "ab", "$1x", "axb"},
		{`(a)`, "ab", "${1}1", "a1b"},
		{`(a)`, "ab", "$11", "b"},
		{`(a)`, "ab", "$0$0", "aab"},
		{`(a)|(b)`, "ab", "[$2]", "[][b]"},
		{`a`, "a", "${x}", "${x}"},
		{`a`, "a", "$", "$"},
		{`a`, "a", "${1", "${1"},
		{`a`, "a", "x$-y", "x$-y"},
		{`a*`, "baaac", "-", "-b--c-"},
		{`x`, "abc", "y", "abc"},
		{`\s+`, "a  b\t\tc", " ", "a b c"},
		{`(\d+)-(\d+)`, "10-20", "$2-$1", "20-10"},
		{`é`, "café", "e", "cafe"},
		{`(o)`, "foo", "${0}${1}", "foooo"},
	}
	for _, c := range cases {
		re := m6Compile(t, c.pattern)
		if got := re.ReplaceAllString(c.src, c.repl); got != c.want {
			t.Errorf("%q.ReplaceAllString(%q, %q) = %q, want %q", c.pattern, c.src, c.repl, got, c.want)
		}
	}
}

func TestM6LongInputs(t *testing.T) {
	s := strings.Repeat("ab", 5000) + "c"
	re := m6Compile(t, `(?:ab)*c`)
	if got := re.FindStringSubmatchIndex(s); !reflect.DeepEqual(got, []int{0, len(s)}) {
		t.Errorf("long match = %v, want [0 %d]", got, len(s))
	}
	re = m6Compile(t, `a.*?z`)
	if re.MatchString(strings.Repeat("a", 2000)) {
		t.Errorf("a.*?z matched a string without z")
	}
}

// No Go file under regexlite/ may import regexp, regexp/syntax or another
// package of this module.
func TestM6NoRegexpImport(t *testing.T) {
	fset := token.NewFileSet()
	files := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		files++
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return nil
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			switch {
			case p == "regexp" || strings.HasPrefix(p, "regexp/"):
				t.Errorf("%s imports %s", path, p)
			case (p == "twelvemods" || strings.HasPrefix(p, "twelvemods/")) &&
				p != "twelvemods/regexlite" && !strings.HasPrefix(p, "twelvemods/regexlite/"):
				t.Errorf("%s imports another package of the module: %s", path, p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files == 0 {
		t.Fatal("no Go files found under regexlite/")
	}
}
