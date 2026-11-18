// Command checkfixture validates a task's memory/fixture.json and checks that no file under the
// given directories contains one of its memory texts, so the agent cannot read them in the workspace.
//
// Usage: checkfixture <fixture.json> <dir>...
// Prints one line per problem; exit 1 on any problem, 2 on bad usage.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const maxMemories = 20

type fixture struct {
	Memories []struct {
		Text string `json:"text"`
	} `json:"memories"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: checkfixture <fixture.json> <dir>...")
		os.Exit(2)
	}
	problems := check(os.Args[1], os.Args[2:])
	for _, p := range problems {
		fmt.Println(p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
}

func check(path string, dirs []string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{err.Error()}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var f fixture
	if err := dec.Decode(&f); err != nil {
		return []string{"fixture.json: " + err.Error()}
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return []string{"fixture.json: trailing data after the object"}
	}
	var problems []string
	if len(f.Memories) == 0 || len(f.Memories) > maxMemories {
		problems = append(problems, fmt.Sprintf("fixture.json: memories must hold 1..%d entries (got %d)", maxMemories, len(f.Memories)))
	}
	for i, m := range f.Memories {
		if strings.TrimSpace(m.Text) == "" {
			problems = append(problems, fmt.Sprintf("fixture.json: memories[%d].text is empty", i))
		}
	}
	if len(problems) > 0 {
		return problems
	}
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			content, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			for i, m := range f.Memories {
				if bytes.Contains(content, []byte(m.Text)) {
					problems = append(problems, fmt.Sprintf("%s contains memories[%d].text", p, i))
				}
			}
			return nil
		})
		if err != nil {
			problems = append(problems, err.Error())
		}
	}
	return problems
}
