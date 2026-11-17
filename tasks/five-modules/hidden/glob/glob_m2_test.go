package glob_test

import (
	"testing"

	"fivemods/glob"
)

type m2Case struct {
	patterns []string
	path     string
	isDir    bool
	want     bool
}

func m2Run(t *testing.T, cases []m2Case) {
	t.Helper()
	for _, c := range cases {
		if got := glob.Match(c.patterns, c.path, c.isDir); got != c.want {
			t.Errorf("Match(%q, %q, isDir=%v) = %v, want %v", c.patterns, c.path, c.isDir, got, c.want)
		}
	}
}

func TestM2Wildcards(t *testing.T) {
	m2Run(t, []m2Case{
		{[]string{"*.log"}, "debug.log", false, true},
		{[]string{"*.log"}, "logs/debug.log", false, true},
		{[]string{"*.log"}, "a/b/c/x.log", false, true},
		{[]string{"*.log"}, "debug.log.txt", false, false},
		{[]string{"*.log"}, ".log", false, true},
		{[]string{"debug?.log"}, "debug1.log", false, true},
		{[]string{"debug?.log"}, "debug10.log", false, false},
		{[]string{"debug?.log"}, "debug.log", false, false},
		{[]string{"?"}, "é", false, true},
		{[]string{"a?c"}, "aéc", false, true},
		{[]string{"a*b*c"}, "aXXbYYbZZc", false, true},
		{[]string{"a*b*c"}, "aXXbYYbZZ", false, false},
		{[]string{"a**b"}, "aXYZb", false, true},
		{[]string{"*"}, "anything", false, true},
		{[]string{"Makefile"}, "makefile", false, false},
		{[]string{"Makefile"}, "src/Makefile", false, true},
		{[]string{"foo"}, "foobar", false, false},
		{[]string{"foo"}, "barfoo", false, false},
		{[]string{"*.[ch]"}, "main.c", false, true},
		{[]string{"*.[ch]"}, "main.h", false, true},
		{[]string{"*.[ch]"}, "main.o", false, false},
		{[]string{"file[0-9].txt"}, "file7.txt", false, true},
		{[]string{"file[0-9].txt"}, "filex.txt", false, false},
		{[]string{"file[!0-9].txt"}, "filex.txt", false, true},
		{[]string{"file[!0-9].txt"}, "file7.txt", false, false},
		{[]string{"file[^0-9].txt"}, "file7.txt", false, false},
		{[]string{"file[^0-9].txt"}, "fileQ.txt", false, true},
		{[]string{"[]]x"}, "]x", false, true},
		{[]string{"[!]]x"}, "]x", false, false},
		{[]string{"[!]]x"}, "ax", false, true},
		{[]string{"[a-]x"}, "-x", false, true},
		{[]string{"[-a]x"}, "-x", false, true},
		{[]string{"[a-c-]x"}, "bx", false, true},
		{[]string{"[a-cx-z]"}, "y", false, true},
		{[]string{"[a-cx-z]"}, "m", false, false},
		{[]string{`[\]]`}, "]", false, true},
		{[]string{`[a\-z]`}, "b", false, false},
		{[]string{`[a\-z]`}, "-", false, true},
		{[]string{"[abc"}, "[abc", false, true},
		{[]string{"[abc"}, "a", false, false},
		{[]string{"x[ab"}, "x[ab", false, true},
		{[]string{`\*.txt`}, "*.txt", false, true},
		{[]string{`\*.txt`}, "a.txt", false, false},
		{[]string{`a\?`}, "a?", false, true},
		{[]string{`a\?`}, "ab", false, false},
		{[]string{`\[x]`}, "[x]", false, true},
		{[]string{`\[x]`}, "x", false, false},
		{[]string{`foo\`}, "foo", false, false},
		{[]string{`foo\`}, `foo\`, false, false},
		{[]string{`foo\\`}, `foo\`, false, true},
	})
}

func TestM2LinePreprocessing(t *testing.T) {
	m2Run(t, []m2Case{
		{[]string{"", "   ", "# comment"}, "comment", false, false},
		{[]string{"#foo"}, "#foo", false, false},
		{[]string{`\#foo`}, "#foo", false, true},
		{[]string{"foo   "}, "foo", false, true},
		{[]string{`foo\ `}, "foo ", false, true},
		{[]string{`foo\ `}, "foo", false, false},
		{[]string{`foo\  `}, "foo ", false, true},
		{[]string{`\!important`}, "!important", false, true},
		{[]string{`\!important`}, "important", false, false},
		{[]string{"!"}, "x", false, false},
		{[]string{"/"}, "x", true, false},
		{[]string{"*", "!"}, "x", false, true},
		{[]string{"x y"}, "x y", false, true},
	})
}

func TestM2Anchoring(t *testing.T) {
	m2Run(t, []m2Case{
		{[]string{"/foo"}, "foo", false, true},
		{[]string{"/foo"}, "a/foo", false, false},
		{[]string{"doc/frotz"}, "doc/frotz", false, true},
		{[]string{"doc/frotz"}, "a/doc/frotz", false, false},
		{[]string{"/doc/frotz"}, "doc/frotz", false, true},
		{[]string{"doc/*.txt"}, "doc/notes.txt", false, true},
		{[]string{"doc/*.txt"}, "doc/server/arch.txt", false, false},
		{[]string{"/*.c"}, "cat-file.c", false, true},
		{[]string{"/*.c"}, "mozilla-sha1/sha1.c", false, false},
		{[]string{"a/b"}, "a/b/c.txt", false, true},
		{[]string{"a/b"}, "a/bc", false, false},
		{[]string{"src/*/test"}, "src/x/test", true, true},
		{[]string{"src/*/test"}, "src/x/y/test", true, false},
	})
}

func TestM2DoubleStar(t *testing.T) {
	m2Run(t, []m2Case{
		{[]string{"**/foo"}, "foo", false, true},
		{[]string{"**/foo"}, "a/b/foo", false, true},
		{[]string{"**/foo/bar"}, "x/foo/bar", false, true},
		{[]string{"**/foo/bar"}, "foo/bar", false, true},
		{[]string{"**/foo/bar"}, "foo/x/bar", false, false},
		{[]string{"abc/**"}, "abc/x", false, true},
		{[]string{"abc/**"}, "abc/x/y", false, true},
		{[]string{"abc/**"}, "abc", true, false},
		{[]string{"abc/**"}, "abc", false, false},
		{[]string{"abc/**"}, "x/abc/y", false, false},
		{[]string{"a/**/b"}, "a/b", false, true},
		{[]string{"a/**/b"}, "a/x/b", false, true},
		{[]string{"a/**/b"}, "a/x/y/b", false, true},
		{[]string{"a/**/b"}, "a/x/y/c", false, false},
		{[]string{"a/**/b"}, "z/a/b", false, false},
		{[]string{"a/**/*.go"}, "a/main.go", false, true},
		{[]string{"a/**/*.go"}, "a/p/q/main.go", false, true},
		{[]string{"a/**/*.go"}, "a/p/q/main.c", false, false},
		{[]string{"a/**b/c"}, "a/xyzb/c", false, true},
		{[]string{"a/**b/c"}, "a/x/b/c", false, false},
		{[]string{"**"}, "deep/nested/file", false, true},
		{[]string{"/**"}, "x", false, true},
		{[]string{"**/"}, "x", false, false},
		{[]string{"**/"}, "x", true, true},
		{[]string{"**/"}, "x/y", false, true},
	})
}

func TestM2DirOnly(t *testing.T) {
	m2Run(t, []m2Case{
		{[]string{"build/"}, "build", true, true},
		{[]string{"build/"}, "build", false, false},
		{[]string{"build/"}, "build/out.o", false, true},
		{[]string{"build/"}, "src/build", true, true},
		{[]string{"build/"}, "src/build", false, false},
		{[]string{"build/"}, "src/build/x/y.o", false, true},
		{[]string{"/build/"}, "src/build/y.o", false, false},
		{[]string{"/build/"}, "build/y.o", false, true},
		{[]string{"doc/frotz/"}, "doc/frotz", true, true},
		{[]string{"doc/frotz/"}, "doc/frotz", false, false},
		{[]string{"doc/frotz/"}, "a/doc/frotz", true, false},
		{[]string{"*.d/"}, "x/conf.d/a", false, true},
		{[]string{"*.d/"}, "x/conf.d", false, false},
	})
}

func TestM2Negation(t *testing.T) {
	m2Run(t, []m2Case{
		{[]string{"*.log", "!keep.log"}, "a/x.log", false, true},
		{[]string{"*.log", "!keep.log"}, "a/keep.log", false, false},
		{[]string{"*.log", "!keep.log"}, "keep.log", false, false},
		{[]string{"!keep.log", "*.log"}, "keep.log", false, true},
		{[]string{"*.log", "!keep.log", "keep.log"}, "keep.log", false, true},
		{[]string{"build/", "!build/keep.txt"}, "build/keep.txt", false, true},
		{[]string{"build/*", "!build/keep.txt"}, "build/keep.txt", false, false},
		{[]string{"build/*", "!build/keep.txt"}, "build/other.txt", false, true},
		{[]string{"/*", "!/src"}, "README", false, true},
		{[]string{"/*", "!/src"}, "docs/a.md", false, true},
		{[]string{"/*", "!/src"}, "src/main.go", false, false},
		{[]string{"/*", "!/src"}, "src", true, false},
		{[]string{"/*", "!/foo", "/foo/*", "!/foo/bar"}, "foo/bar/baz", false, false},
		{[]string{"/*", "!/foo", "/foo/*", "!/foo/bar"}, "foo/qux", false, true},
		{[]string{"/*", "!/foo", "/foo/*", "!/foo/bar"}, "foo/bar", true, false},
		{[]string{"/*", "!/foo", "/foo/*", "!/foo/bar"}, "other/x", false, true},
		{[]string{"logs/", "!logs/"}, "logs/a.txt", false, false},
		{[]string{"logs", "!logs/"}, "logs", false, true},
		{[]string{"logs", "!logs/"}, "logs", true, false},
		{[]string{"a/**", "!a/b/**"}, "a/b/c", false, true},
		{[]string{"a/**", "!a/b"}, "a/b/c", false, true},
		{[]string{"a/**", "!a/b"}, "a/b", true, false},
		{[]string{"*", "!*/"}, "dir/file", false, true},
		{[]string{"*", "!*/", "!*.go"}, "main.go", false, false},
		{[]string{"*", "!*/", "!*.go"}, "main.c", false, true},
		{[]string{"*", "!*/", "!*.go"}, "pkg/main.go", false, false},
		{[]string{"*", "!*/", "!*.go"}, "pkg/main.c", false, true},
		{nil, "anything", false, false},
	})
}
