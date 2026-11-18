package inireader_test

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"twelvemods/inireader"
)

func m4MustParse(t *testing.T, src string) *inireader.File {
	t.Helper()
	f, err := inireader.ParseString(src)
	if err != nil {
		t.Fatalf("ParseString error: %v\n%s", err, src)
	}
	return f
}

func TestM4Values(t *testing.T) {
	src := strings.Join([]string{
		"# leading comment",
		"; another comment",
		"name = root value",
		"",
		"[server]",
		"host = \"example.com\"   # trailing comment",
		"port = 8080",
		"neg = -42",
		"plus = +7",
		"zeros = 007",
		"hex = 0xFF",
		"hexlow = 0x1f",
		"ratio = 0.75",
		"sci = 1e3",
		"sci2 = -2.5E-2",
		"on = true",
		"off = false",
		"cap = True",
		"version = 1.2.3",
		"dot = 1.",
		"hexonly = 0x",
		"neghex = -0x10",
		"bare = hello   world   ; note",
		"hash = a#b",
		"semi = a;b",
		"eqs = a = b",
		"\tindented\t=\t  spaced value\t",
		"lit = 'C:\\path\\n'   ; literal",
		"esc = \"tab\\there \\\"q\\\" back\\\\slash\\nnl\\rcr\"",
		"uni = \"\\u00e9\\u00C9\\u65e5\"",
		"empty = \"\"",
		"emptylit = ''",
		"hashstr = \"a # not comment\"#real comment",
		"maxint = 9223372036854775807",
		"minint = -9223372036854775808",
		"utf = héllo wörld",
		"crlf = yes\r",
	}, "\n")
	f := m4MustParse(t, src)
	strs := map[string]string{
		"host": "example.com", "cap": "True", "version": "1.2.3", "dot": "1.", "hexonly": "0x", "neghex": "-0x10",
		"bare": "hello   world", "hash": "a#b", "semi": "a;b", "eqs": "a = b", "indented": "spaced value",
		"lit": `C:\path\n`, "esc": "tab\there \"q\" back\\slash\nnl\rcr", "uni": "éÉ日", "empty": "", "emptylit": "",
		"hashstr": "a # not comment", "utf": "héllo wörld", "crlf": "yes",
	}
	for k, want := range strs {
		v, ok := f.Lookup("server", k)
		if !ok || v.Kind != inireader.KindString || v.Str != want {
			t.Errorf("server.%s = %+v, %v; want string %q", k, v, ok, want)
		}
		if got := f.GetString("server", k, "DEF"); got != want {
			t.Errorf("GetString(server.%s) = %q, want %q", k, got, want)
		}
	}
	ints := map[string]int64{"port": 8080, "neg": -42, "plus": 7, "zeros": 7, "hex": 255, "hexlow": 31,
		"maxint": math.MaxInt64, "minint": math.MinInt64}
	for k, want := range ints {
		v, ok := f.Lookup("server", k)
		if !ok || v.Kind != inireader.KindInt || v.Int != want {
			t.Errorf("server.%s = %+v, %v; want int %d", k, v, ok, want)
		}
		if got := f.GetInt("server", k, -1); got != want {
			t.Errorf("GetInt(server.%s) = %d, want %d", k, got, want)
		}
	}
	floats := map[string]float64{"ratio": 0.75, "sci": 1000, "sci2": -0.025}
	for k, want := range floats {
		v, ok := f.Lookup("server", k)
		if !ok || v.Kind != inireader.KindFloat || v.Float != want {
			t.Errorf("server.%s = %+v, %v; want float %v", k, v, ok, want)
		}
	}
	if v, ok := f.Lookup("server", "on"); !ok || v.Kind != inireader.KindBool || !v.Bool {
		t.Errorf("server.on = %+v", v)
	}
	if v, ok := f.Lookup("server", "off"); !ok || v.Kind != inireader.KindBool || v.Bool {
		t.Errorf("server.off = %+v", v)
	}
	if got := f.GetString("", "name", "DEF"); got != "root value" {
		t.Errorf("root name = %q", got)
	}
	if _, ok := f.Lookup("", "host"); ok {
		t.Errorf("host leaked into root section")
	}
	wantKeys := []string{"host", "port", "neg", "plus", "zeros", "hex", "hexlow", "ratio", "sci", "sci2", "on", "off",
		"cap", "version", "dot", "hexonly", "neghex", "bare", "hash", "semi", "eqs", "indented", "lit", "esc", "uni",
		"empty", "emptylit", "hashstr", "maxint", "minint", "utf", "crlf"}
	if got := f.Keys("server"); !reflect.DeepEqual(got, wantKeys) {
		t.Errorf("Keys(server) = %q\nwant %q", got, wantKeys)
	}
	if got := f.Keys(""); !reflect.DeepEqual(got, []string{"name"}) {
		t.Errorf("Keys(root) = %q", got)
	}
	if got := f.Keys("nope"); len(got) != 0 {
		t.Errorf("Keys(nope) = %q", got)
	}
}

func TestM4Getters(t *testing.T) {
	f := m4MustParse(t, `
s = "str"
i = 12
fl = 2.5
b = true
arr_s = ["a", 'b', "c,d", "e]"]
arr_i = [1, -2, 0x10,]
arr_f = [1.5, 2e2]
arr_e = []
arr_b = [true, false]
[Sec]
i = 1
`)
	checks := []struct {
		name      string
		got, want any
	}{
		{"GetString string", f.GetString("", "s", "d"), "str"},
		{"GetString int", f.GetString("", "i", "d"), "d"},
		{"GetString missing", f.GetString("", "zz", "d"), "d"},
		{"GetString missing section", f.GetString("nosec", "s", "d"), "d"},
		{"GetInt int", f.GetInt("", "i", -1), int64(12)},
		{"GetInt float", f.GetInt("", "fl", -1), int64(-1)},
		{"GetInt string", f.GetInt("", "s", -1), int64(-1)},
		{"GetInt case sensitive section", f.GetInt("sec", "i", -1), int64(-1)},
		{"GetInt section", f.GetInt("Sec", "i", -1), int64(1)},
		{"GetInt case sensitive key", f.GetInt("", "I", -1), int64(-1)},
		{"GetFloat float", f.GetFloat("", "fl", -1), 2.5},
		{"GetFloat int", f.GetFloat("", "i", -1), 12.0},
		{"GetFloat bool", f.GetFloat("", "b", -1), -1.0},
		{"GetBool bool", f.GetBool("", "b", false), true},
		{"GetBool string", f.GetBool("", "s", false), false},
		{"GetBool missing default", f.GetBool("", "zz", true), true},
		{"GetStrings", f.GetStrings("", "arr_s", nil), []string{"a", "b", "c,d", "e]"}},
		{"GetStrings ints", f.GetStrings("", "arr_i", []string{"def"}), []string{"def"}},
		{"GetStrings scalar", f.GetStrings("", "s", []string{"def"}), []string{"def"}},
		{"GetStrings empty", f.GetStrings("", "arr_e", nil), []string{}},
		{"GetInts", f.GetInts("", "arr_i", nil), []int64{1, -2, 16}},
		{"GetInts floats", f.GetInts("", "arr_f", []int64{9}), []int64{9}},
		{"GetInts empty", f.GetInts("", "arr_e", nil), []int64{}},
		{"GetInts missing", f.GetInts("", "none", []int64{3}), []int64{3}},
	}
	for _, c := range checks {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s: got %#v, want %#v", c.name, c.got, c.want)
		}
	}
	v, ok := f.Lookup("", "arr_f")
	if !ok || v.Kind != inireader.KindArray || len(v.Array) != 2 || v.Array[0].Kind != inireader.KindFloat || v.Array[1].Float != 200 {
		t.Errorf("arr_f = %+v", v)
	}
	v, ok = f.Lookup("", "arr_e")
	if !ok || v.Kind != inireader.KindArray || v.Array == nil || len(v.Array) != 0 {
		t.Errorf("arr_e = %#v", v)
	}
	v, ok = f.Lookup("", "arr_b")
	if !ok || v.Kind != inireader.KindArray || len(v.Array) != 2 || v.Array[0].Kind != inireader.KindBool || !v.Array[0].Bool || v.Array[1].Bool {
		t.Errorf("arr_b = %+v", v)
	}
	if _, ok := f.Lookup("", "missing"); ok {
		t.Errorf("Lookup(missing) ok")
	}
	got := f.GetStrings("", "arr_s", nil)
	got[0] = "mutated"
	if again := f.GetStrings("", "arr_s", nil); again[0] != "a" {
		t.Errorf("GetStrings returned shared storage")
	}
}

func TestM4Sections(t *testing.T) {
	f := m4MustParse(t, `
top = 1
[a]
x = 1
[ a.b ]  # spaced name
y = 2
[a.b.c] ; deep
[x-y_z.9]
[a.d.e]
[b]
[a.b2]
`)
	if got, want := f.Sections(), []string{"a", "a.b", "a.b.c", "x-y_z.9", "a.d.e", "b", "a.b2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Sections = %q, want %q", got, want)
	}
	if got, want := f.Subsections(""), []string{"a", "x-y_z", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Subsections(\"\") = %q, want %q", got, want)
	}
	if got, want := f.Subsections("a"), []string{"b", "d", "b2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Subsections(a) = %q, want %q", got, want)
	}
	if got, want := f.Subsections("a.b"), []string{"c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Subsections(a.b) = %q, want %q", got, want)
	}
	if got := f.Subsections("a.b.c"); len(got) != 0 {
		t.Errorf("Subsections(a.b.c) = %q", got)
	}
	if got := f.Subsections("zzz"); len(got) != 0 {
		t.Errorf("Subsections(zzz) = %q", got)
	}
	if got := f.GetInt("a.b", "y", 0); got != 2 {
		t.Errorf("a.b.y = %d", got)
	}
	if got := f.GetInt("a", "y", 0); got != 0 {
		t.Errorf("a.y leaked: %d", got)
	}
	if got := f.Keys("a.b.c"); len(got) != 0 {
		t.Errorf("Keys(a.b.c) = %q", got)
	}
	f2 := m4MustParse(t, "k = v\n")
	if got := f2.Sections(); len(got) != 0 {
		t.Errorf("Sections without headers = %q", got)
	}
}

func TestM4Errors(t *testing.T) {
	cases := []struct {
		src  string
		line int
		msg  string
	}{
		{"[a\nx=1", 1, "invalid section header"},
		{"x = 1\n[]", 2, "invalid section header"},
		{"[a..b]", 1, "invalid section header"},
		{"[.a]", 1, "invalid section header"},
		{"[a b]", 1, "invalid section header"},
		{"[a] x", 1, "invalid section header"},
		{"[a]]", 1, "invalid section header"},
		{"[a$]", 1, "invalid section header"},
		{"[a]\nk=1\n[b]\n[a]", 4, `duplicate section "a"`},
		{"just text", 1, "missing '='"},
		{"\n\n  # c\nfoo", 4, "missing '='"},
		{"= 5", 1, "invalid key"},
		{"a b = 5", 1, "invalid key"},
		{"a.b = 5", 1, "invalid key"},
		{"k\u00e9y = 1", 1, "invalid key"},
		{"a = 1\nb = 2\na = 3", 3, `duplicate key "a"`},
		{"[s]\nk = 1\n[t]\nk = 1\n[u]\nk = 1\nk = 2", 7, `duplicate key "k"`},
		{"a =", 1, "missing value"},
		{"a =    ", 1, "missing value"},
		{"a = # comment", 1, "missing value"},
		{"a = ;comment", 1, "missing value"},
		{"a = \"abc", 1, "unterminated string"},
		{"a = \"abc\\", 1, "unterminated string"},
		{"a = \"abc\\\"", 1, "unterminated string"},
		{"a = 'abc", 1, "unterminated string"},
		{`a = "bad \q escape"`, 1, `invalid escape sequence \q`},
		{`a = "bad \x41"`, 1, `invalid escape sequence \x`},
		{`a = "\u12"`, 1, "invalid unicode escape"},
		{`a = "\u12G4"`, 1, "invalid unicode escape"},
		{`a = "\u+123"`, 1, "invalid unicode escape"},
		{`a = "x" y`, 1, "trailing characters after value"},
		{`a = 'x''y'`, 1, "trailing characters after value"},
		{`a = [1, 2] 3`, 1, "trailing characters after value"},
		{"a = [1, 2", 1, "unterminated array"},
		{"a = [1,", 1, "unterminated array"},
		{"a = [", 1, "unterminated array"},
		{"a = [\"x\"", 1, "unterminated array"},
		{"a = [1, \"two\"]", 1, "mixed types in array"},
		{"a = [1, 2.0]", 1, "mixed types in array"},
		{"a = [true, 1]", 1, "mixed types in array"},
		{"a = [[1], [2]]", 1, "nested arrays are not supported"},
		{"a = [1, [2]]", 1, "nested arrays are not supported"},
		{"a = [,]", 1, "invalid array element"},
		{"a = [1,,2]", 1, "invalid array element"},
		{"a = [hello]", 1, "invalid array element"},
		{"a = [1.]", 1, "invalid array element"},
		{"a = [1 2]", 1, "expected ',' or ']' in array"},
		{"a = [\"a\" \"b\"]", 1, "expected ',' or ']' in array"},
		{"a = [\"bad \\q\"]", 1, `invalid escape sequence \q`},
		{"a = 9223372036854775808", 1, "integer out of range"},
		{"a = -9223372036854775809", 1, "integer out of range"},
		{"a = 0x8000000000000000", 1, "integer out of range"},
		{"a = [1, 99999999999999999999]", 1, "integer out of range"},
		{"a = 1e999", 1, "float out of range"},
		{"ok = 1\r\nbad\r\n", 2, "missing '='"},
		{"a = 1\nb = [1, \"x\"]\nc = \"unterminated", 2, "mixed types in array"},
	}
	for _, c := range cases {
		f, err := inireader.ParseString(c.src)
		if err == nil {
			t.Errorf("ParseString(%q) succeeded, want line %d %q", c.src, c.line, c.msg)
			continue
		}
		if f != nil {
			t.Errorf("ParseString(%q) returned non-nil File with error", c.src)
		}
		var pe *inireader.ParseError
		if !errors.As(err, &pe) {
			t.Errorf("ParseString(%q) error %v is not *ParseError", c.src, err)
			continue
		}
		if pe.Line != c.line || pe.Msg != c.msg {
			t.Errorf("ParseString(%q) = line %d %q, want line %d %q", c.src, pe.Line, pe.Msg, c.line, c.msg)
		}
		if want := "line " + strings.TrimSpace(m4Itoa(c.line)) + ": " + c.msg; err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	}
}

func m4Itoa(n int) string {
	s := ""
	for {
		s = string(rune('0'+n%10)) + s
		n /= 10
		if n == 0 {
			return s
		}
	}
}

func TestM4ParseReader(t *testing.T) {
	f, err := inireader.Parse(strings.NewReader("[db]\nport = 5432\nhosts = [\"a\", \"b\"]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if f.GetInt("db", "port", 0) != 5432 || !reflect.DeepEqual(f.GetStrings("db", "hosts", nil), []string{"a", "b"}) {
		t.Errorf("Parse(reader) values wrong")
	}
	_, err = inireader.Parse(strings.NewReader("x = 1\n[bad"))
	var pe *inireader.ParseError
	if !errors.As(err, &pe) || pe.Line != 2 {
		t.Errorf("Parse(reader) error = %v", err)
	}
}
