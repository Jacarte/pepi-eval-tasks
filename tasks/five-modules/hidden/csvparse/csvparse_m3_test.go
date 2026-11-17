package csvparse_test

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"fivemods/csvparse"
)

func m3ReadAll(t *testing.T, in string, opts csvparse.Options) ([][]string, error) {
	t.Helper()
	return csvparse.NewReader(strings.NewReader(in), opts).ReadAll()
}

func TestM3ReadRecords(t *testing.T) {
	cases := []struct {
		name string
		in   string
		opts csvparse.Options
		want [][]string
	}{
		{"basic", "a,b,c\n1,2,3\n", csvparse.Options{}, [][]string{{"a", "b", "c"}, {"1", "2", "3"}}},
		{"no final newline", "a,b\nc,d", csvparse.Options{}, [][]string{{"a", "b"}, {"c", "d"}}},
		{"crlf", "a,b\r\nc,d\r\n", csvparse.Options{}, [][]string{{"a", "b"}, {"c", "d"}}},
		{"quoted delimiter", "\"a,b\",c\n", csvparse.Options{}, [][]string{{"a,b", "c"}}},
		{"embedded newline", "\"line1\nline2\",x\ny,z\n", csvparse.Options{}, [][]string{{"line1\nline2", "x"}, {"y", "z"}}},
		{"embedded crlf normalized", "\"l1\r\nl2\",x\r\n", csvparse.Options{}, [][]string{{"l1\nl2", "x"}}},
		{"embedded lone cr kept", "\"a\rb\",x\n", csvparse.Options{}, [][]string{{"a\rb", "x"}}},
		{"doubled quotes", "\"he said \"\"hi\"\"\",x\n", csvparse.Options{}, [][]string{{"he said \"hi\"", "x"}}},
		{"empty fields", ",,\n", csvparse.Options{}, [][]string{{"", "", ""}}},
		{"empty quoted", "\"\",x\n", csvparse.Options{}, [][]string{{"", "x"}}},
		{"single empty quoted", "\"\"\n", csvparse.Options{}, [][]string{{""}}},
		{"blank lines skipped", "a\n\n\nb\n\n", csvparse.Options{}, [][]string{{"a"}, {"b"}}},
		{"blank crlf lines skipped", "\r\na\r\n\r\nb", csvparse.Options{}, [][]string{{"a"}, {"b"}}},
		{"whitespace kept", " a , b \n \n", csvparse.Options{Ragged: csvparse.RaggedAllow}, [][]string{{" a ", " b "}, {" "}}},
		{"lone cr is data", "a\rb,c\n", csvparse.Options{}, [][]string{{"a\rb", "c"}}},
		{"bom dropped", "\uFEFFa,b\n", csvparse.Options{}, [][]string{{"a", "b"}}},
		{"bom elsewhere kept", "a,\uFEFFb\n", csvparse.Options{}, [][]string{{"a", "\uFEFFb"}}},
		{"semicolon", "a;b,c\n", csvparse.Options{Delimiter: ';'}, [][]string{{"a", "b,c"}}},
		{"tab", "a\tb c\n", csvparse.Options{Delimiter: '\t'}, [][]string{{"a", "b c"}}},
		{"multibyte delimiter", "aü\"b\"üc\n", csvparse.Options{Delimiter: 'ü'}, [][]string{{"a", "b", "c"}}},
		{"custom quote", "'a,b','it''s',\"x\"\n", csvparse.Options{Quote: '\''}, [][]string{{"a,b", "it's", "\"x\""}}},
		{"comments", "#c\na,b\n#x,y\n", csvparse.Options{Comment: '#'}, [][]string{{"a", "b"}}},
		{"comment after bom", "\uFEFF#c\na\n", csvparse.Options{Comment: '#'}, [][]string{{"a"}}},
		{"comment char mid line", "a,#b\n", csvparse.Options{Comment: '#'}, [][]string{{"a", "#b"}}},
		{"comment char inside quoted field", "\"a\n#b\",c\n", csvparse.Options{Comment: '#'}, [][]string{{"a\n#b", "c"}}},
		{"comments disabled", "#a,b\n", csvparse.Options{}, [][]string{{"#a", "b"}}},
		{"comment last line no newline", "a\n;x", csvparse.Options{Comment: ';'}, [][]string{{"a"}}},
		{"lazy bare quote", "a\"b,c\n", csvparse.Options{LazyQuotes: true}, [][]string{{"a\"b", "c"}}},
		{"lazy inner quote", "\"a\"b\",c\n", csvparse.Options{LazyQuotes: true}, [][]string{{"a\"b", "c"}}},
		{"lazy eof in quotes", "x,\"abc", csvparse.Options{LazyQuotes: true}, [][]string{{"x", "abc"}}},
		{"quoted at eof", "\"a\"", csvparse.Options{}, [][]string{{"a"}}},
		{"trailing delimiter at eof", "a,", csvparse.Options{}, [][]string{{"a", ""}}},
		{"quoted then delimiter at eof", "\"a\",", csvparse.Options{}, [][]string{{"a", ""}}},
		{"unicode", "héllo,wörld,日本\n", csvparse.Options{}, [][]string{{"héllo", "wörld", "日本"}}},
		{"only newlines", "\n\n\r\n", csvparse.Options{}, nil},
		{"empty input", "", csvparse.Options{}, nil},
		{"bom only", "\uFEFF", csvparse.Options{}, nil},
	}
	for _, c := range cases {
		got, err := m3ReadAll(t, c.in, c.opts)
		if err != nil {
			t.Errorf("%s: ReadAll error: %v", c.name, err)
			continue
		}
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: ReadAll(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestM3ParseErrors(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		opts      csvparse.Options
		line, col int
		err       error
	}{
		{"bare quote", "a,b\"c\n", csvparse.Options{}, 1, 4, csvparse.ErrBareQuote},
		{"bare quote after space", "x, \"a\"\n", csvparse.Options{}, 1, 4, csvparse.ErrBareQuote},
		{"char after closing quote", "a,\"b\"c\n", csvparse.Options{}, 1, 6, csvparse.ErrQuote},
		{"space after closing quote", "a,\"b\" ,c\n", csvparse.Options{}, 1, 6, csvparse.ErrQuote},
		{"eof in quoted field", "a,\"bc", csvparse.Options{}, 1, 3, csvparse.ErrQuote},
		{"eof in multiline quoted field", "x\n\"abc\ndef\n", csvparse.Options{}, 2, 1, csvparse.ErrQuote},
		{"error on continuation line", "x\ny,\"multi\nline\"z\n", csvparse.Options{}, 3, 6, csvparse.ErrQuote},
		{"bom not counted", "\uFEFFa\"b\n", csvparse.Options{}, 1, 2, csvparse.ErrBareQuote},
		{"rune columns", "héé,ü\"\n", csvparse.Options{}, 1, 6, csvparse.ErrBareQuote},
		{"rune columns after quote", "\"日本\"語\n", csvparse.Options{}, 1, 5, csvparse.ErrQuote},
		{"comment lines counted", "#c\n#d\na\"\n", csvparse.Options{Comment: '#'}, 3, 2, csvparse.ErrBareQuote},
		{"blank lines counted", "a\n\n\nb\"\n", csvparse.Options{}, 4, 2, csvparse.ErrBareQuote},
		{"crlf lines", "a\r\n\"b\"x\r\n", csvparse.Options{}, 2, 4, csvparse.ErrQuote},
		{"custom quote closing", "'a'b\n", csvparse.Options{Quote: '\''}, 1, 4, csvparse.ErrQuote},
		{"custom quote bare", "ab'c\n", csvparse.Options{Quote: '\''}, 1, 3, csvparse.ErrBareQuote},
		{"semicolon delimiter quote check", "\"a\",\"b\"\n", csvparse.Options{Delimiter: ';'}, 1, 4, csvparse.ErrQuote},
	}
	for _, c := range cases {
		r := csvparse.NewReader(strings.NewReader(c.in), c.opts)
		var err error
		var rec []string
		for err == nil {
			rec, err = r.Read()
		}
		if err == io.EOF {
			t.Errorf("%s: no error for %q", c.name, c.in)
			continue
		}
		var pe *csvparse.ParseError
		if !errors.As(err, &pe) {
			t.Errorf("%s: error %v (%T) is not *ParseError", c.name, err, err)
			continue
		}
		if !errors.Is(err, c.err) || pe.Line != c.line || pe.Column != c.col {
			t.Errorf("%s: got line %d col %d err %v, want line %d col %d err %v", c.name, pe.Line, pe.Column, pe.Err, c.line, c.col, c.err)
		}
		if rec != nil {
			t.Errorf("%s: record %q returned with syntax error", c.name, rec)
		}
		want := "line " + m3Itoa(c.line) + ", column " + m3Itoa(c.col) + ": " + c.err.Error()
		if pe.Error() != want {
			t.Errorf("%s: Error() = %q, want %q", c.name, pe.Error(), want)
		}
	}
	// Quote handling is not affected by a non-default quote rune.
	got, err := m3ReadAll(t, "a'b,\"c\"\n", csvparse.Options{})
	if err != nil || !reflect.DeepEqual(got, [][]string{{"a'b", "c"}}) {
		t.Errorf("default quote: got %q, %v", got, err)
	}
}

func m3Itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestM3Ragged(t *testing.T) {
	in := "a,b\nc\n\n\"d\ne\",f,g\nh,i\n"
	for _, c := range []struct {
		policy csvparse.RaggedPolicy
		want   [][]string
	}{
		{csvparse.RaggedAllow, [][]string{{"a", "b"}, {"c"}, {"d\ne", "f", "g"}, {"h", "i"}}},
		{csvparse.RaggedPad, [][]string{{"a", "b"}, {"c", ""}, {"d\ne", "f"}, {"h", "i"}}},
	} {
		got, err := m3ReadAll(t, in, csvparse.Options{Ragged: c.policy})
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("policy %d: got %q, %v want %q", c.policy, got, err, c.want)
		}
	}

	r := csvparse.NewReader(strings.NewReader(in), csvparse.Options{})
	type step struct {
		rec  []string
		line int // 0 = no error
	}
	steps := []step{{[]string{"a", "b"}, 0}, {[]string{"c"}, 2}, {[]string{"d\ne", "f", "g"}, 4}, {[]string{"h", "i"}, 0}}
	for i, s := range steps {
		rec, err := r.Read()
		if !reflect.DeepEqual(rec, s.rec) {
			t.Errorf("step %d: record %q, want %q", i, rec, s.rec)
		}
		if s.line == 0 {
			if err != nil {
				t.Errorf("step %d: unexpected error %v", i, err)
			}
			continue
		}
		var pe *csvparse.ParseError
		if !errors.As(err, &pe) || !errors.Is(err, csvparse.ErrFieldCount) || pe.Line != s.line || pe.Column != 1 {
			t.Errorf("step %d: err %v, want ErrFieldCount at line %d col 1", i, err, s.line)
		}
	}
	if rec, err := r.Read(); rec != nil || err != io.EOF {
		t.Errorf("after last record: %q, %v want nil, EOF", rec, err)
	}
	if rec, err := r.Read(); rec != nil || err != io.EOF {
		t.Errorf("Read after EOF: %q, %v want nil, EOF", rec, err)
	}

	got, err := m3ReadAll(t, "a,b\nc,d\ne\nf,g\n", csvparse.Options{})
	if !errors.Is(err, csvparse.ErrFieldCount) || !reflect.DeepEqual(got, [][]string{{"a", "b"}, {"c", "d"}}) {
		t.Errorf("ReadAll ragged: %q, %v", got, err)
	}
	got, err = m3ReadAll(t, "a,b\nc,\"d\n", csvparse.Options{})
	var pe *csvparse.ParseError
	if !errors.As(err, &pe) || !errors.Is(err, csvparse.ErrQuote) || pe.Line != 2 || pe.Column != 3 ||
		!reflect.DeepEqual(got, [][]string{{"a", "b"}}) {
		t.Errorf("ReadAll quote error: %q, %v", got, err)
	}
}

func TestM3InvalidOptions(t *testing.T) {
	bad := []csvparse.Options{
		{Delimiter: '"'},
		{Delimiter: ';', Quote: ';'},
		{Delimiter: '\n'},
		{Delimiter: '\r'},
		{Quote: '\n'},
		{Delimiter: utf8.RuneError},
		{Comment: ','},
		{Comment: '"'},
		{Comment: '\n'},
		{Delimiter: ';', Comment: ';'},
	}
	for _, o := range bad {
		_, err := csvparse.NewReader(strings.NewReader("a,b\n"), o).Read()
		if !errors.Is(err, csvparse.ErrInvalidOptions) {
			t.Errorf("Read with %+v: err %v, want ErrInvalidOptions", o, err)
		}
		_, err = csvparse.NewReader(strings.NewReader("a,b\n"), o).ReadAll()
		if !errors.Is(err, csvparse.ErrInvalidOptions) {
			t.Errorf("ReadAll with %+v: err %v, want ErrInvalidOptions", o, err)
		}
	}
	for _, o := range []csvparse.WriterOptions{{Delimiter: '"'}, {Quote: '\r'}, {Delimiter: '\n'}, {Delimiter: 'x', Quote: 'x'}} {
		var buf bytes.Buffer
		w := csvparse.NewWriter(&buf, o)
		if err := w.Write([]string{"a"}); !errors.Is(err, csvparse.ErrInvalidOptions) {
			t.Errorf("Write with %+v: err %v, want ErrInvalidOptions", o, err)
		}
		if err := w.WriteAll([][]string{{"a"}}); !errors.Is(err, csvparse.ErrInvalidOptions) {
			t.Errorf("WriteAll with %+v: err %v, want ErrInvalidOptions", o, err)
		}
	}
}

func TestM3Streaming(t *testing.T) {
	pr, pw := io.Pipe()
	r := csvparse.NewReader(pr, csvparse.Options{})
	release := make(chan struct{})
	go func() {
		pw.Write([]byte("a,\"b\nc\"\n"))
		<-release
		pw.Write([]byte("d,e\n"))
		pw.Close()
	}()
	type result struct {
		rec []string
		err error
	}
	got := make(chan result, 1)
	go func() {
		rec, err := r.Read()
		got <- result{rec, err}
	}()
	select {
	case res := <-got:
		if res.err != nil || !reflect.DeepEqual(res.rec, []string{"a", "b\nc"}) {
			t.Fatalf("first record %q, %v", res.rec, res.err)
		}
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("Read did not return the first record before more input arrived")
	}
	close(release)
	rec, err := r.Read()
	if err != nil || !reflect.DeepEqual(rec, []string{"d", "e"}) {
		t.Fatalf("second record %q, %v", rec, err)
	}
	if _, err := r.Read(); err != io.EOF {
		t.Fatalf("want EOF, got %v", err)
	}
}

func TestM3FreshSlices(t *testing.T) {
	r := csvparse.NewReader(strings.NewReader("a,b\nc,d\n"), csvparse.Options{})
	first, _ := r.Read()
	second, _ := r.Read()
	if !reflect.DeepEqual(first, []string{"a", "b"}) || !reflect.DeepEqual(second, []string{"c", "d"}) {
		t.Errorf("records overwritten: %q %q", first, second)
	}
}

func TestM3Writer(t *testing.T) {
	cases := []struct {
		name    string
		opts    csvparse.WriterOptions
		records [][]string
		want    string
	}{
		{"plain", csvparse.WriterOptions{}, [][]string{{"a", "b"}, {"c", "d"}}, "a,b\nc,d\n"},
		{"quoting rules", csvparse.WriterOptions{}, [][]string{{"c,d", "e\"f", "plain"}}, "\"c,d\",\"e\"\"f\",plain\n"},
		{"leading space and tab", csvparse.WriterOptions{}, [][]string{{" lead", "trail ", "\tx", "in side"}}, "\" lead\",trail ,\"\tx\",in side\n"},
		{"newlines", csvparse.WriterOptions{}, [][]string{{"x\ny", "\r", "z"}}, "\"x\ny\",\"\r\",z\n"},
		{"single empty field", csvparse.WriterOptions{}, [][]string{{""}}, "\"\"\n"},
		{"two empty fields", csvparse.WriterOptions{}, [][]string{{"", ""}}, ",\n"},
		{"no fields", csvparse.WriterOptions{}, [][]string{{}}, "\n"},
		{"crlf", csvparse.WriterOptions{UseCRLF: true}, [][]string{{"a", "b\nc"}, {"d"}}, "a,\"b\nc\"\r\nd\r\n"},
		{"quote all", csvparse.WriterOptions{QuoteAll: true}, [][]string{{"a", "", "b\"c"}}, "\"a\",\"\",\"b\"\"c\"\n"},
		{"semicolon", csvparse.WriterOptions{Delimiter: ';'}, [][]string{{"a,b", "c;d"}}, "a,b;\"c;d\"\n"},
		{"single quote", csvparse.WriterOptions{Quote: '\''}, [][]string{{"it's", "say \"hi\"", "a,b"}}, "'it''s',say \"hi\",'a,b'\n"},
		{"multibyte delimiter", csvparse.WriterOptions{Delimiter: '¦'}, [][]string{{"a", "b¦c", "日本"}}, "a¦\"b¦c\"¦日本\n"},
	}
	for _, c := range cases {
		var buf bytes.Buffer
		w := csvparse.NewWriter(&buf, c.opts)
		for _, rec := range c.records {
			if err := w.Write(rec); err != nil {
				t.Fatalf("%s: Write: %v", c.name, err)
			}
		}
		if err := w.Flush(); err != nil {
			t.Fatalf("%s: Flush: %v", c.name, err)
		}
		if buf.String() != c.want {
			t.Errorf("%s: wrote %q, want %q", c.name, buf.String(), c.want)
		}
		var buf2 bytes.Buffer
		if err := csvparse.NewWriter(&buf2, c.opts).WriteAll(c.records); err != nil || buf2.String() != c.want {
			t.Errorf("%s: WriteAll wrote %q, %v", c.name, buf2.String(), err)
		}
	}
}

type m3FailingWriter struct{}

func (m3FailingWriter) Write(p []byte) (int, error) { return 0, errors.New("disk full") }

func TestM3WriterError(t *testing.T) {
	err := csvparse.NewWriter(m3FailingWriter{}, csvparse.WriterOptions{}).WriteAll([][]string{{"a", "b"}})
	if err == nil {
		t.Fatal("WriteAll to a failing writer returned nil")
	}
}

func TestM3RoundTrip(t *testing.T) {
	alphabet := []string{"a", "b", " ", ",", ";", "\"", "'", "\n", "é", "\t", "#", "x"}
	rng := rand.New(rand.NewSource(42))
	for _, o := range []struct {
		w csvparse.WriterOptions
		r csvparse.Options
	}{
		{csvparse.WriterOptions{}, csvparse.Options{Ragged: csvparse.RaggedAllow}},
		{csvparse.WriterOptions{Delimiter: ';', UseCRLF: true}, csvparse.Options{Delimiter: ';', Ragged: csvparse.RaggedAllow}},
		{csvparse.WriterOptions{Quote: '\'', QuoteAll: true}, csvparse.Options{Quote: '\'', Ragged: csvparse.RaggedAllow}},
	} {
		var records [][]string
		for i := 0; i < 200; i++ {
			n := 1 + rng.Intn(5)
			rec := make([]string, n)
			for j := range rec {
				var sb strings.Builder
				for k := rng.Intn(6); k > 0; k-- {
					sb.WriteString(alphabet[rng.Intn(len(alphabet))])
				}
				rec[j] = sb.String()
			}
			records = append(records, rec)
		}
		var buf bytes.Buffer
		if err := csvparse.NewWriter(&buf, o.w).WriteAll(records); err != nil {
			t.Fatalf("WriteAll: %v", err)
		}
		got, err := csvparse.NewReader(&buf, o.r).ReadAll()
		if err != nil {
			t.Fatalf("ReadAll(%+v): %v", o.r, err)
		}
		if !reflect.DeepEqual(got, records) {
			t.Errorf("round trip with %+v differs", o.w)
			for i := range records {
				if i >= len(got) || !reflect.DeepEqual(got[i], records[i]) {
					t.Errorf("first difference at record %d: wrote %q", i, records[i])
					break
				}
			}
		}
	}
}
