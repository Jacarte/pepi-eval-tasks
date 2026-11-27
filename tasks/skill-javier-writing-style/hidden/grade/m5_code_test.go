package grade

import (
	"os"
	"strings"
	"testing"
)

// The post shows the whole workflow file: its non-blank lines appear verbatim, in order, inside
// one fenced code block.
func TestM5Code(t *testing.T) {
	data, err := os.ReadFile("testdata/workflow.yml")
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			want = append(want, strings.TrimRight(line, " \t"))
		}
	}
	for _, block := range fencedBlocks(readPost(t)) {
		next := 0
		for _, line := range block {
			if next < len(want) && line == want[next] {
				next++
			}
		}
		if next == len(want) {
			return
		}
	}
	t.Fatalf("no fenced code block holds the whole workflow file (%d lines, verbatim and in order)", len(want))
}
