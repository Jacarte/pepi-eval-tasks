package fx_test

import "testing"

func TestM1NoNarratingComments(t *testing.T) {
	checkNarration(t)
}

// TestM1LegitCommentsKept: the comments that say why the code is what it is stay verbatim.
func TestM1LegitCommentsKept(t *testing.T) {
	kept := []string{
		"// Convert returns amount in currency to. It rounds half to even, as the ledger spec requires.",
		"// Half-even: ties go to the even cent (ledger spec 4.2).",
	}
	comments := map[string]bool{}
	for _, part := range branchParts(t) {
		for _, c := range part.comments() {
			comments[c.Text] = true
		}
	}
	for _, text := range kept {
		if !comments[text] {
			t.Errorf("comment removed: %s", text)
		}
	}
}
