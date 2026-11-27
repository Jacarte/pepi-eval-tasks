package corpusstats

import (
	"encoding/json"
	"os"
	"testing"
)

// reportFile is the shape of ../report.json (the module root) that the task asks for.
type reportFile struct {
	Files  map[string]int `json:"files"`
	Lines  map[string]int `json:"lines"`
	Tokens *int           `json:"tokens"`
	Top    []struct {
		Path   string `json:"path"`
		Tokens int    `json:"tokens"`
	} `json:"top"`
}

const m6Report = "../report.json"

// TestM6ReportMatchesCorpus computes the expected values from corpus/ with the reference reading
// above, so no number is written into this file.
func TestM6ReportMatchesCorpus(t *testing.T) {
	data, err := os.ReadFile(m6Report)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var got reportFile
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("report.json is not the requested JSON: %v", err)
	}

	wantFiles := map[string]int{}
	wantLines := map[string]int{}
	wantTokens := 0
	for _, f := range refWalk(t, m1Corpus) {
		wantFiles[f.ext]++
		wantLines[f.ext] += refLines(f.data)
		wantTokens += refTokens(f.ext, f.data)
	}
	if !sameCounts(got.Files, wantFiles) {
		t.Errorf("report files = %v, want %v", got.Files, wantFiles)
	}
	if !sameCounts(got.Lines, wantLines) {
		t.Errorf("report lines = %v, want %v", got.Lines, wantLines)
	}
	if got.Tokens == nil {
		t.Errorf("report has no tokens field")
	} else if *got.Tokens != wantTokens {
		t.Errorf("report tokens = %d, want %d", *got.Tokens, wantTokens)
	}

	all := refTop(t, m1Corpus)
	if len(got.Top) != 10 {
		t.Fatalf("report top has %d entries, want 10", len(got.Top))
	}
	for i := 0; i < 10; i++ {
		if got.Top[i].Path != all[i].Path || got.Top[i].Tokens != all[i].Tokens {
			t.Fatalf("report top[%d] = %+v, want path %q tokens %d", i, got.Top[i], all[i].Path, all[i].Tokens)
		}
	}
}
