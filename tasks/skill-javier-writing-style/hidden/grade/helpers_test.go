package grade

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// readPost returns blog/post.md of the workspace.
func readPost(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../blog/post.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// fenceLine reports whether the line opens or closes a fenced code block.
func fenceLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "```")
}

// proseLines returns the lines of the post outside fenced code blocks, so that a comment in a
// config is not read as a heading or as prose.
func proseLines(post string) []string {
	var out []string
	in := false
	for _, line := range strings.Split(post, "\n") {
		if fenceLine(line) {
			in = !in
			continue
		}
		if !in {
			out = append(out, strings.TrimRight(line, " \t"))
		}
	}
	return out
}

// fencedBlocks returns the body lines of each fenced code block of the post.
func fencedBlocks(post string) [][]string {
	var blocks [][]string
	var cur []string
	in := false
	for _, line := range strings.Split(post, "\n") {
		if fenceLine(line) {
			if in {
				blocks = append(blocks, cur)
				cur = nil
			}
			in = !in
			continue
		}
		if in {
			cur = append(cur, strings.TrimRight(line, " \t"))
		}
	}
	return blocks
}

var headingLine = regexp.MustCompile(`^#{1,6}\s+(.*)$`)

// headings returns the text of each markdown heading outside code blocks.
func headings(post string) []string {
	var out []string
	for _, line := range proseLines(post) {
		if m := headingLine.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
		}
	}
	return out
}

// firstParagraph returns the lines of the first paragraph of the post: the first run of
// non-blank lines that is not a heading. It fails the test when that run opens with a code block.
func firstParagraph(t *testing.T, post string) []string {
	t.Helper()
	var para []string
	for _, line := range strings.Split(post, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(para) == 0 {
			if trimmed == "" || headingLine.MatchString(trimmed) {
				continue
			}
			if fenceLine(trimmed) {
				t.Fatal("the post opens with a code block, not with the problem")
			}
		}
		if trimmed == "" || headingLine.MatchString(trimmed) {
			break
		}
		para = append(para, trimmed)
	}
	if len(para) == 0 {
		t.Fatal("the post has no opening paragraph")
	}
	return para
}
