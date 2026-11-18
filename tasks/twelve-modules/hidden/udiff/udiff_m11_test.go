package udiff_test

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"twelvemods/udiff"
)

// m11Lines turns "abc" into ["a\n", "b\n", "c\n"].
func m11Lines(s string) []string {
	var out []string
	for _, r := range s {
		out = append(out, string(r)+"\n")
	}
	return out
}

// m11Script renders an edit script as "=a -b +c" (line text without "\n").
func m11Script(edits []udiff.Edit) string {
	var parts []string
	for _, e := range edits {
		mark := map[udiff.OpKind]string{udiff.OpEqual: "=", udiff.OpDelete: "-", udiff.OpInsert: "+"}[e.Kind]
		parts = append(parts, mark+strings.TrimSuffix(e.Line, "\n"))
	}
	return strings.Join(parts, " ")
}

func TestM11SplitLines(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a\n", []string{"a\n"}},
		{"a\nb", []string{"a\n", "b"}},
		{"\n\n", []string{"\n", "\n"}},
		{"a\r\nb\r\n", []string{"a\r\n", "b\r\n"}},
	}
	for _, c := range cases {
		got := udiff.SplitLines(c.in)
		if len(got) != len(c.want) || (len(got) > 0 && !reflect.DeepEqual(got, c.want)) {
			t.Errorf("SplitLines(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestM11Diff(t *testing.T) {
	cases := []struct{ a, b, want string }{
		{"", "", ""},
		{"abc", "abc", "=a =b =c"},
		{"", "ab", "+a +b"},
		{"ab", "", "-a -b"},
		{"a", "b", "-a +b"},
		{"ab", "ba", "-a =b +a"},
		{"ABCABBA", "CBABAC", "-A -B =C +B =A =B -B =A +C"},
		{"abcd", "abxd", "=a =b -c +x =d"},
		{"abc", "xabc", "+x =a =b =c"},
		{"abc", "abcx", "=a =b =c +x"},
		{"aaa", "aa", "=a =a -a"},
		{"aa", "aaa", "=a =a +a"},
		{"abab", "baba", "-a =b =a =b +a"},
		{"xyz", "zyx", "-x -y =z +y +x"},
	}
	for _, c := range cases {
		got := m11Script(udiff.Diff(m11Lines(c.a), m11Lines(c.b)))
		if got != c.want {
			t.Errorf("Diff(%q, %q) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
	// Lines differing only in the final newline are different lines.
	got := m11Script(udiff.Diff([]string{"a\n", "b"}, []string{"a\n", "b\n"}))
	if got != "=a -b +b" {
		t.Errorf("newline-sensitive Diff = %q", got)
	}
}

func m11LCS(a, b []string) int {
	dp := make([][]int, len(a)+1)
	for i := range dp {
		dp[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] > dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	return dp[0][0]
}

func m11Random(r *rand.Rand) string {
	var b strings.Builder
	n := r.Intn(12)
	for i := 0; i < n; i++ {
		b.WriteString([]string{"a", "b", "c", "d", ""}[r.Intn(5)])
		b.WriteString("\n")
	}
	if r.Intn(4) == 0 {
		b.WriteString("tail")
	}
	return b.String()
}

func TestM11DiffProperties(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	for iter := 0; iter < 1500; iter++ {
		a, b := udiff.SplitLines(m11Random(r)), udiff.SplitLines(m11Random(r))
		edits := udiff.Diff(a, b)
		var gotA, gotB []string
		changes := 0
		for _, e := range edits {
			if e.Kind != udiff.OpInsert {
				gotA = append(gotA, e.Line)
			}
			if e.Kind != udiff.OpDelete {
				gotB = append(gotB, e.Line)
			}
			if e.Kind != udiff.OpEqual {
				changes++
			}
		}
		if strings.Join(gotA, "") != strings.Join(a, "") || strings.Join(gotB, "") != strings.Join(b, "") || len(gotA) != len(a) || len(gotB) != len(b) {
			t.Fatalf("Diff(%q, %q) does not reconstruct its inputs: %v", a, b, edits)
		}
		if want := len(a) + len(b) - 2*m11LCS(a, b); changes != want {
			t.Fatalf("Diff(%q, %q) has %d changes, shortest is %d", a, b, changes, want)
		}
	}
}

func TestM11Unified(t *testing.T) {
	ten := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n"
	tenChanged := "1\nX\n3\n4\n5\n6\n7\n8\nY\n10\n"
	cases := []struct {
		name, a, b string
		context    int
		want       string
	}{
		{"spec example", "x\ny\n", "x\nz\n", 1, "--- a\n+++ b\n@@ -1,2 +1,2 @@\n x\n-y\n+z\n"},
		{"equal", "x\ny\n", "x\ny\n", 3, ""},
		{"both empty", "", "", 3, ""},
		{"from empty", "", "hello\n", 3, "--- a\n+++ b\n@@ -0,0 +1 @@\n+hello\n"},
		{"to empty", "hello\n", "", 3, "--- a\n+++ b\n@@ -1 +0,0 @@\n-hello\n"},
		{"no newline", "a\nb", "a\nc", 1, "--- a\n+++ b\n@@ -1,2 +1,2 @@\n a\n-b\n\\ No newline at end of file\n+c\n\\ No newline at end of file\n"},
		{"newline added", "a", "a\n", 3, "--- a\n+++ b\n@@ -1 +1 @@\n-a\n\\ No newline at end of file\n+a\n"},
		{"two hunks", ten, tenChanged, 1, "--- a\n+++ b\n@@ -1,3 +1,3 @@\n 1\n-2\n+X\n 3\n@@ -8,3 +8,3 @@\n 8\n-9\n+Y\n 10\n"},
		{"two hunks ctx 2", ten, tenChanged, 2, "--- a\n+++ b\n@@ -1,4 +1,4 @@\n 1\n-2\n+X\n 3\n 4\n@@ -7,4 +7,4 @@\n 7\n 8\n-9\n+Y\n 10\n"},
		{"merged hunk", ten, tenChanged, 3, "--- a\n+++ b\n@@ -1,10 +1,10 @@\n 1\n-2\n+X\n 3\n 4\n 5\n 6\n 7\n 8\n-9\n+Y\n 10\n"},
		{"ctx 0 change", "a\nb\nc\n", "a\nB\nc\n", 0, "--- a\n+++ b\n@@ -2 +2 @@\n-b\n+B\n"},
		{"ctx 0 insert", "a\nc\n", "a\nb\nc\n", 0, "--- a\n+++ b\n@@ -1,0 +2 @@\n+b\n"},
		{"ctx 0 delete", "a\nb\nc\n", "a\nc\n", 0, "--- a\n+++ b\n@@ -2 +1,0 @@\n-b\n"},
		{"negative ctx", "a\nb\nc\n", "a\nB\nc\n", -5, "--- a\n+++ b\n@@ -2 +2 @@\n-b\n+B\n"},
		{"ctx 0 adjacent", "a\nb\nc\nd\n", "a\nB\nC\nd\n", 0, "--- a\n+++ b\n@@ -2,2 +2,2 @@\n-b\n-c\n+B\n+C\n"},
		{"ctx 0 one apart", "a\nb\nc\nd\n", "B\nb\nC\nd\n", 0, "--- a\n+++ b\n@@ -1 +1 @@\n-a\n+B\n@@ -3 +3 @@\n-c\n+C\n"},
		{"big ctx", "a\nb\n", "a\nc\n", 10, "--- a\n+++ b\n@@ -1,2 +1,2 @@\n a\n-b\n+c\n"},
		{"myers", "A\nB\nC\nA\nB\nB\nA\n", "C\nB\nA\nB\nA\nC\n", 3, "--- a\n+++ b\n@@ -1,7 +1,6 @@\n-A\n-B\n C\n+B\n A\n B\n-B\n A\n+C\n"},
		{"insert at start ctx 1", "b\nc\nd\n", "a\nb\nc\nd\n", 1, "--- a\n+++ b\n@@ -1 +1,2 @@\n+a\n b\n"},
		{"append ctx 2", "a\nb\nc\n", "a\nb\nc\nd\n", 2, "--- a\n+++ b\n@@ -2,2 +2,3 @@\n b\n c\n+d\n"},
		{"names", "a\n", "b\n", 0, "--- old/file.txt\n+++ new/file.txt\n@@ -1 +1 @@\n-a\n+b\n"},
	}
	for _, c := range cases {
		aName, bName := "a", "b"
		if c.name == "names" {
			aName, bName = "old/file.txt", "new/file.txt"
		}
		if got := udiff.Unified(aName, bName, c.a, c.b, c.context); got != c.want {
			t.Errorf("%s: Unified\n got: %q\nwant: %q", c.name, got, c.want)
		}
	}
}

func TestM11ApplyRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	for iter := 0; iter < 1500; iter++ {
		a, b := m11Random(r), m11Random(r)
		for ctx := 0; ctx <= 3; ctx++ {
			patch := udiff.Unified("a", "b", a, b, ctx)
			got, err := udiff.Apply(a, patch)
			if err != nil || got != b {
				t.Fatalf("Apply(%q, Unified(%q, %q, %d)) = %q, %v\npatch:\n%s", a, a, b, ctx, got, err, patch)
			}
		}
	}
	big := strings.Repeat("line\nother\n", 1000)
	changed := strings.Replace(big, "other\n", "changed\n", 3)
	patch := udiff.Unified("a", "b", big, changed+"end\n", 3)
	if got, err := udiff.Apply(big, patch); err != nil || got != changed+"end\n" {
		t.Errorf("large round trip failed: %v", err)
	}
	crlf, crlf2 := "a\r\nb\r\n", "a\r\nc\r\n"
	if got, err := udiff.Apply(crlf, udiff.Unified("a", "b", crlf, crlf2, 1)); err != nil || got != crlf2 {
		t.Errorf("CRLF round trip = %q, %v", got, err)
	}
}

func TestM11ApplyPatches(t *testing.T) {
	cases := []struct{ name, original, patch, want string }{
		{"empty patch", "a\nb\n", "", "a\nb\n"},
		{"headers only", "a\nb\n", "--- a\n+++ b\n", "a\nb\n"},
		{"no file headers", "a\nb\nc\n", "@@ -2 +2 @@\n-b\n+B\n", "a\nB\nc\n"},
		{"section text", "a\nb\n", "--- x\n+++ y\n@@ -1,2 +1,2 @@ func main()\n a\n-b\n+c\n", "a\nc\n"},
		{"no trailing newline in patch", "a\n", "@@ -1 +1 @@\n-a\n+b", "b\n"},
		{"insert at start", "a\n", "@@ -0,0 +1 @@\n+x\n", "x\na\n"},
		{"into empty", "", "@@ -0,0 +1 @@\n+x\n", "x\n"},
		{"insert at end", "a\n", "@@ -1,0 +2 @@\n+x\n", "a\nx\n"},
		{"two inserts same point", "a\nb\n", "@@ -1,0 +2 @@\n+x\n@@ -1,0 +3 @@\n+y\n", "a\nx\ny\nb\n"},
		{"delete all", "a\nb\n", "@@ -1,2 +0,0 @@\n-a\n-b\n", ""},
		{"marker on old line", "a", "@@ -1 +1 @@\n-a\n\\ No newline at end of file\n+b\n", "b\n"},
		{"marker on new line", "a\n", "@@ -1 +1 @@\n-a\n+b\n\\ No newline at end of file\n", "b"},
		{"adjacent hunks", "a\nb\nc\n", "@@ -1 +1 @@\n-a\n+A\n@@ -2 +2 @@\n-b\n+B\n", "A\nB\nc\n"},
		{"s2 not checked", "a\nb\n", "@@ -2 +77 @@\n-b\n+c\n", "a\nc\n"},
	}
	for _, c := range cases {
		got, err := udiff.Apply(c.original, c.patch)
		if err != nil || got != c.want {
			t.Errorf("%s: Apply = %q, %v; want %q", c.name, got, err, c.want)
		}
	}
}

func TestM11ApplyErrors(t *testing.T) {
	abc := "a\nb\nc\n"
	cases := []struct {
		name, original, patch string
		line                  int
		msg                   string
	}{
		{"garbage", abc, "garbage\n", 1, "expected hunk header"},
		{"garbage after headers", abc, "--- a\n+++ b\nfoo\n", 3, "expected hunk header"},
		{"trailing junk", "a\n", "@@ -1 +1 @@\n-a\n+b\nextra\n", 4, "expected hunk header"},
		{"plus header", abc, "@@ +1 @@\n", 1, "expected hunk header"},
		{"bad number", abc, "@@ -x +1 @@\n", 1, "invalid hunk header"},
		{"no closing", abc, "@@ -1 +1\n", 1, "invalid hunk header"},
		{"glued text", abc, "@@ -1 +1 @@x\n", 1, "invalid hunk header"},
		{"zero start", abc, "@@ -0 +1 @@\n-a\n", 1, "invalid hunk header"},
		{"bad count", abc, "@@ -1,2, +1 @@\n", 1, "invalid hunk header"},
		{"empty count", abc, "@@ -1, +1 @@\n", 1, "invalid hunk header"},
		{"overflow", abc, "@@ -99999999999999999999999 +1 @@\n", 1, "invalid hunk header"},
		{"out of order", abc, "@@ -3 +3 @@\n-c\n+C\n@@ -2 +2 @@\n-b\n+B\n", 4, "hunk out of order"},
		{"overlap", abc, "@@ -1,2 +1,2 @@\n a\n-b\n+B\n@@ -2 +2 @@\n-b\n+X\n", 5, "hunk out of order"},
		{"insert inside previous", abc, "@@ -1,2 +1,2 @@\n a\n-b\n+B\n@@ -1,0 +2 @@\n+x\n", 5, "hunk out of order"},
		{"beyond end", "a\n", "@@ -3 +3 @@\n-c\n+d\n", 1, "hunk out of order"},
		{"range beyond end", "a\n", "@@ -1,2 +1,2 @@\n a\n-b\n+c\n", 1, "hunk out of order"},
		{"insert beyond end", "a\n", "@@ -2,0 +2 @@\n+x\n", 1, "hunk out of order"},
		{"bad prefix", "a\n", "@@ -1 +1 @@\n*a\n", 2, "unexpected line in hunk"},
		{"empty body line", "a\n", "@@ -1 +1 @@\n\n", 2, "unexpected line in hunk"},
		{"marker first", "a\n", "@@ -1 +1 @@\n\\ No newline at end of file\n-a\n+b\n", 2, "unexpected line in hunk"},
		{"double marker", "a", "@@ -1 +1 @@\n-a\n\\ x\n\\ y\n+b\n", 4, "unexpected line in hunk"},
		{"marker after full hunk", "a", "@@ -1 +1 @@\n-a\n+b\n\\ x\n\\ y\n", 5, "unexpected line in hunk"},
		{"exceeds count", "a\nb\n", "@@ -1,2 +1 @@\n a\n b\n", 3, "unexpected line in hunk"},
		{"truncated", abc, "@@ -1,3 +1,3 @@\n a\n b\n", 4, "unexpected end of patch"},
		{"truncated no newline", abc, "@@ -1,2 +1,2 @@\n a", 3, "unexpected end of patch"},
		{"mismatch", abc, "@@ -2 +2 @@\n-x\n+y\n", 2, "does not match original line 2"},
		{"context mismatch", abc, "@@ -1,3 +1,3 @@\n a\n-b\n+B\n d\n", 5, "does not match original line 3"},
		{"second hunk mismatch", abc, "@@ -1 +1 @@\n-a\n+A\n@@ -3 +3 @@\n z\n", 5, "does not match original line 3"},
		{"structure before content", "a\n", "@@ -1 +1 @@\n-zzz\n+b\nbad\n", 4, "expected hunk header"},
		{"newline mismatch", "a", "@@ -1 +1 @@\n-a\n+b\n", 2, "does not match original line 1"},
		{"marker mismatch", "a\n", "@@ -1 +1 @@\n-a\n\\ No newline at end of file\n+b\n", 2, "does not match original line 1"},
	}
	for _, c := range cases {
		got, err := udiff.Apply(c.original, c.patch)
		var ae *udiff.ApplyError
		if !errors.As(err, &ae) {
			t.Errorf("%s: Apply error = %v (result %q), want *ApplyError", c.name, err, got)
			continue
		}
		if got != "" {
			t.Errorf("%s: Apply returned %q with an error", c.name, got)
		}
		if ae.Line != c.line || ae.Msg != c.msg {
			t.Errorf("%s: ApplyError{%d, %q}, want {%d, %q}", c.name, ae.Line, ae.Msg, c.line, c.msg)
		}
		if want := fmt.Sprintf("patch line %d: %s", c.line, c.msg); ae.Error() != want {
			t.Errorf("%s: Error() = %q, want %q", c.name, ae.Error(), want)
		}
	}
}
