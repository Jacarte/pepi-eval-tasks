package tmpl_test

import (
	"errors"
	"testing"

	"twelvemods/tmpl"
)

func m12Data() map[string]any {
	return map[string]any{
		"name":    "World",
		"html":    `<a href="x">Tom & 'Jerry'</a>`,
		"n":       42,
		"big":     int64(-7),
		"pi":      3.5,
		"zero":    0,
		"fzero":   0.0,
		"empty":   "",
		"yes":     true,
		"no":      false,
		"nothing": nil,
		"list":    []any{"a", "b", "c"},
		"nums":    []any{1, int64(2), 3.25, true, nil},
		"none":    []any{},
		"nested":  []any{[]any{"x"}},
		"m":       map[string]any{"b": 2, "a": 1, "c": 3},
		"emptym":  map[string]any{},
		"user":    map[string]any{"first": "Ada", "tags": []any{"x", "y"}, "addr": map[string]any{"city": "Paris"}},
		"pad":     "  padded  ",
		"uni":     "héllo wörld",
		"sep":     "-",
	}
}

type m12Case struct {
	src    string
	escape bool
	want   string
}

func TestM12Execute(t *testing.T) {
	cases := []m12Case{
		{"plain text", false, "plain text"},
		{"", false, ""},
		{"Hello, {{ .name }}!", false, "Hello, World!"},
		{"{{.name}}", false, "World"},
		{"{{ .n }} {{ .big }} {{ .pi }} {{ .yes }} {{ .no }}", false, "42 -7 3.5 true false"},
		{"[{{ .nothing }}][{{ .missing }}][{{ .missing.deeper }}]", false, "[][][]"},
		{"{{ .user.first }} in {{ .user.addr.city }}", false, "Ada in Paris"},
		{"{{ $.name }}", false, "World"},
		{"{{ \"lit\" }} {{ 7 }} {{ -3 }} {{-3}} {{ true }}", false, "lit 7 -3 -3 true"},
		{`{{ "a\"b\\c\td" }}`, false, "a\"b\\c\td"},
		{`{{ "}}" }}`, false, "}}"},
		{"{{ .html }}", false, `<a href="x">Tom & 'Jerry'</a>`},
		{"{{ .html }}", true, "&lt;a href=&quot;x&quot;&gt;Tom &amp; &#39;Jerry&#39;&lt;/a&gt;"},
		{"<b>{{ .name }}</b>", true, "<b>World</b>"},
		{"{{ .html | raw }}", true, `<a href="x">Tom & 'Jerry'</a>`},
		{"{{ .html | escape }}", false, "&lt;a href=&quot;x&quot;&gt;Tom &amp; &#39;Jerry&#39;&lt;/a&gt;"},
		{"{{ .html | escape }}", true, "&lt;a href=&quot;x&quot;&gt;Tom &amp; &#39;Jerry&#39;&lt;/a&gt;"},
		{"{{ \"<i>\" | raw | upper }}", true, "&lt;I&gt;"},
		{"{{ .n | raw }}", true, "42"},
		// trim markers
		{"a  \n {{- .name -}} \n\t b", false, "aWorldb"},
		{"a  {{- .name }}  b", false, "aWorld  b"},
		{"a  {{ .name -}}  b", false, "a  Worldb"},
		{"x {{- /* note */ -}} y", false, "xy"},
		{"x{{/* a }} \"b */}}y", false, "xy"},
		{"x {{ /* c */ }} y", false, "x  y"},
		{"5 {{-3}} 5", false, "5 -3 5"},
		// if
		{"{{ if .yes }}Y{{ end }}", false, "Y"},
		{"{{ if .no }}Y{{ else }}N{{ end }}", false, "N"},
		{"{{ if .zero }}1{{ else if .fzero }}2{{ else if .empty }}3{{ else if .none }}4{{ else if .emptym }}5{{ else if .nothing }}6{{ else }}7{{ end }}", false, "7"},
		{"{{ if .n }}a{{ end }}{{ if .list }}b{{ end }}{{ if .m }}c{{ end }}{{ if .name }}d{{ end }}{{ if .pi }}e{{ end }}", false, "abcde"},
		{"{{ if .no }}1{{ else if .yes }}2{{ else }}3{{ end }}", false, "2"},
		{"{{ if .yes }}{{ if .no }}a{{ else }}b{{ end }}{{ end }}", false, "b"},
		{"{{ if .yes }}ok{{ else if .name.x }}bad{{ end }}", false, "ok"},
		{"{{ if .no }}{{ .name.x }}{{ end }}fine", false, "fine"},
		// range
		{"{{ range .list }}[{{ $index }}:{{ . }}]{{ end }}", false, "[0:a][1:b][2:c]"},
		{"{{ range .m }}{{ $key }}={{ . }};{{ end }}", false, "a=1;b=2;c=3;"},
		{"{{ range .m }}{{ $index }}{{ end }}", false, "012"},
		{"{{ range .list }}{{ $key }}{{ end }}", false, "012"},
		{"{{ range .none }}x{{ else }}empty{{ end }}", false, "empty"},
		{"{{ range .nothing }}x{{ else }}nil{{ end }}", false, "nil"},
		{"{{ range .emptym }}x{{ end }}done", false, "done"},
		{"{{ range .list }}{{ $.name }}{{ end }}", false, "WorldWorldWorld"},
		{"{{ range .nested }}{{ range . }}{{ $index }}{{ . }}{{ end }}{{ $index }}{{ end }}", false, "0x0"},
		{"{{ range .list }}{{ . }}{{ end }}{{ .name }}", false, "abcWorld"},
		{"{{ range .user.tags }}<{{ . }}>{{ end }}", true, "<x><y>"},
		{"{{ range .list -}}\n  {{ . }}\n{{- end }}", false, "abc"},
		{"{{ range $.list }}{{ if $index }},{{ end }}{{ . }}{{ end }}", false, "a,b,c"},
		// filters
		{"{{ .name | upper }} {{ .name | lower }} [{{ .pad | trim }}]", false, "WORLD world [padded]"},
		{"{{ .uni | len }} {{ .list | len }} {{ .m | len }} {{ .nothing | len }} {{ .none | len }}", false, "11 3 3 0 0"},
		{"{{ .empty | default \"none\" }} {{ .name | default \"none\" }} {{ .zero | default 5 }} {{ .missing | default .name }}", false, "none World 5 World"},
		{"{{ .list | join \", \" }}|{{ .nums | join $.sep }}|{{ .nothing | join \"x\" }}|{{ .none | join \"x\" }}", false, "a, b, c|1-2-3.25-true-||"},
		{"{{ .uni | truncate 4 }}|{{ .uni | truncate 0 }}|{{ .name | truncate 99 }}|{{ .name | truncate .n }}", false, "héll||World|World"},
		{"{{ .name | lower | truncate 3 | upper }}", false, "WOR"},
		{"{{ .html | raw | len }}", false, "29"},
		{"{{ .list | len | default 9 }}", false, "3"},
		{"{{ .missing | default \"<x>\" }}", true, "&lt;x&gt;"},
		{"line1\n{{ .name }}\nline3", false, "line1\nWorld\nline3"},
	}
	data := m12Data()
	for _, c := range cases {
		tp, err := tmpl.Parse(c.src, tmpl.Options{EscapeHTML: c.escape})
		if err != nil {
			t.Errorf("Parse(%q): %v", c.src, err)
			continue
		}
		got, err := tp.Execute(data)
		if err != nil {
			t.Errorf("Execute(%q): %v", c.src, err)
			continue
		}
		if got != c.want {
			t.Errorf("Execute(%q, escape=%v)\n got: %q\nwant: %q", c.src, c.escape, got, c.want)
		}
	}
}

func TestM12RootAndReuse(t *testing.T) {
	tp, err := tmpl.Parse("{{ . }}/{{ $ }}", tmpl.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		data any
		want string
	}{{"s", "s/s"}, {7, "7/7"}, {nil, "/"}, {2.5, "2.5/2.5"}} {
		if got, err := tp.Execute(c.data); err != nil || got != c.want {
			t.Errorf("Execute(%v) = %q, %v; want %q", c.data, got, err, c.want)
		}
	}
}

type m12Err struct {
	src       string
	line, col int
	msg       string
}

func m12Check(t *testing.T, what string, err error, c m12Err) {
	t.Helper()
	var te *tmpl.Error
	if !errors.As(err, &te) {
		t.Errorf("%s(%q) error = %v, want *tmpl.Error", what, c.src, err)
		return
	}
	if te.Line != c.line || te.Column != c.col || te.Msg != c.msg {
		t.Errorf("%s(%q) = %d:%d %q, want %d:%d %q", what, c.src, te.Line, te.Column, te.Msg, c.line, c.col, c.msg)
	}
}

func TestM12ParseErrors(t *testing.T) {
	cases := []m12Err{
		{"abc {{ .x", 1, 5, "unclosed action"},
		{"{{ .x }} {{", 1, 10, "unclosed action"},
		{"{{/* never closed }}", 1, 1, "unclosed action"},
		{"{{/* x */-}}", 1, 1, "unclosed action"},
		{"{{ }}", 1, 1, "empty action"},
		{"a\n  {{- -}}", 2, 3, "empty action"},
		{`{{ "abc }}`, 1, 1, "unterminated string"},
		{"{{ a }}", 1, 1, "invalid action"},
		{"{{ .a. }}", 1, 1, "invalid action"},
		{"{{ .1 }}", 1, 1, "invalid action"},
		{"{{ $x }}", 1, 1, "invalid action"},
		{"{{ $index.x }}", 1, 1, "invalid action"},
		{`{{ "\q" }}`, 1, 1, "invalid action"},
		{"{{ 99999999999999999999 }}", 1, 1, "invalid action"},
		{"{{ if }}x{{ end }}", 1, 1, "invalid action"},
		{"{{ .x | }}", 1, 1, "invalid action"},
		{`{{ .x | "s" }}`, 1, 1, "invalid action"},
		{"{{ .x .y }}", 1, 1, "invalid action"},
		{"{{ | upper }}", 1, 1, "invalid action"},
		{"{{ if .a }}{{ end x }}", 1, 12, "invalid action"},
		{"{{ if .a }}{{ else x }}{{ end }}", 1, 12, "invalid action"},
		{"{{ .x | upper .y | lower }}", 1, 1, `wrong number of arguments for filter "upper"`},
		{"{{ .x | default }}", 1, 1, `wrong number of arguments for filter "default"`},
		{"{{ .x | truncate 1 2 }}", 1, 1, `wrong number of arguments for filter "truncate"`},
		{"{{ .x | title }}", 1, 1, `unknown filter "title"`},
		{"{{ .x | if }}", 1, 1, `unknown filter "if"`},
		{"x\n{{ $index }}", 2, 1, "$index outside range"},
		{"{{ range $key }}{{ end }}", 1, 1, "$key outside range"},
		{"{{ range .l }}{{ else }}{{ $index }}{{ end }}", 1, 25, "$index outside range"},
		{"{{ range .l }}{{ end }}{{ .x | default $key }}", 1, 24, "$key outside range"},
		{"{{ else }}", 1, 1, "unexpected {{else}}"},
		{"{{ if .a }}{{ else }}{{ else }}{{ end }}", 1, 22, "unexpected {{else}}"},
		{"{{ if .a }}{{ else }}{{ else if .b }}{{ end }}", 1, 22, "unexpected {{else}}"},
		{"{{ range .a }}{{ else if .b }}{{ end }}", 1, 15, "unexpected {{else}}"},
		{"{{ range .a }}{{ else }}{{ else }}{{ end }}", 1, 25, "unexpected {{else}}"},
		{"{{ end }}", 1, 1, "unexpected {{end}}"},
		{"{{ if .a }}{{ end }}{{ end }}", 1, 21, "unexpected {{end}}"},
		{"{{ if .a }}x", 1, 1, "unclosed {{if}}"},
		{"{{ if .a }}\n {{ range .b }}x{{ end }}", 1, 1, "unclosed {{if}}"},
		{"{{ if .a }}\n {{ range .b }}x", 2, 2, "unclosed {{range}}"},
		{"é{{ .x | nope }}", 1, 2, `unknown filter "nope"`},
	}
	for _, c := range cases {
		tp, err := tmpl.Parse(c.src, tmpl.Options{})
		if tp != nil {
			t.Errorf("Parse(%q) returned a template with an error", c.src)
		}
		m12Check(t, "Parse", err, c)
	}
	_, err := tmpl.Parse("{{ }}", tmpl.Options{})
	if err == nil || err.Error() != "1:1: empty action" {
		t.Errorf("Error() = %v", err)
	}
}

func TestM12ExecErrors(t *testing.T) {
	cases := []m12Err{
		{"{{ .list }}", 1, 1, "cannot print list"},
		{"a {{ .m }}", 1, 3, "cannot print map"},
		{"{{ .nested | join \",\" }}", 1, 1, "cannot print list"},
		{"{{ .m | raw }}", 1, 1, "cannot print map"},
		{"{{ .list | escape }}", 1, 1, "cannot print list"},
		{"{{ .name.first }}", 1, 1, `cannot access field "first" on string`},
		{"{{ .list.x }}", 1, 1, `cannot access field "x" on list`},
		{"{{ .n.x }}", 1, 1, `cannot access field "x" on int`},
		{"{{ if .pi.x }}{{ end }}", 1, 1, `cannot access field "x" on float`},
		{"\n {{ range .name }}{{ end }}", 2, 2, "cannot range over string"},
		{"{{ range .yes }}{{ end }}", 1, 1, "cannot range over bool"},
		{"{{ .n | upper }}", 1, 1, `invalid input for filter "upper": int`},
		{"{{ .nothing | trim }}", 1, 1, `invalid input for filter "trim": nil`},
		{"{{ .yes | len }}", 1, 1, `invalid input for filter "len": bool`},
		{"{{ .name | join \",\" }}", 1, 1, `invalid input for filter "join": string`},
		{"{{ .list | truncate 2 }}", 1, 1, `invalid input for filter "truncate": list`},
		{"{{ .n | truncate \"x\" }}", 1, 1, `invalid input for filter "truncate": int`},
		{"{{ .name | truncate \"x\" }}", 1, 1, `invalid argument for filter "truncate"`},
		{"{{ .name | truncate -1 }}", 1, 1, `invalid argument for filter "truncate"`},
		{"{{ .list | join 1 }}", 1, 1, `invalid argument for filter "join"`},
		{"{{ .name | default .name.x }}", 1, 1, `cannot access field "x" on string`},
		{"ok {{ range .list }}{{ if $index }}{{ $.m }}{{ end }}{{ end }}", 1, 36, "cannot print map"},
	}
	data := m12Data()
	for _, c := range cases {
		tp, err := tmpl.Parse(c.src, tmpl.Options{})
		if err != nil {
			t.Errorf("Parse(%q): %v", c.src, err)
			continue
		}
		out, err := tp.Execute(data)
		if out != "" {
			t.Errorf("Execute(%q) returned %q with an error", c.src, out)
		}
		m12Check(t, "Execute", err, c)
	}
}
