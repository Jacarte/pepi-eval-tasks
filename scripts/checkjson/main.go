// Command checkjson judges the `go test -json` stream of one milestone run.
//
// Usage: checkjson [-expected FILE] < events.json
//
// FILE holds one expected test per line as "<dir>\t<TestName>", where <dir> is
// the directory of the hidden test file relative to hidden/ ("." for the module
// root). The run is accepted only if every expected test has a pass event in a
// package whose import path ends in /<dir> (any package for "."), and neither
// an expected test nor one of its subtests has a skip or fail event.
//
// Without -expected the run is accepted if it has at least one pass event for a
// test, and no fail event.
//
// Exit status: 0 accepted, 1 rejected (reasons on stderr), 2 usage or I/O error.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

type event struct {
	Action  string
	Package string
	Test    string
}

type expectedTest struct {
	dir  string
	name string
}

func (e expectedTest) matches(pkg string) bool {
	return e.dir == "." || pkg == e.dir || strings.HasSuffix(pkg, "/"+e.dir)
}

func main() {
	expectedPath := flag.String("expected", "", "file with expected tests, one \"<dir>\\t<TestName>\" per line")
	flag.Parse()

	var expected []expectedTest
	if *expectedPath != "" {
		f, err := os.Open(*expectedPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "checkjson:", err)
			os.Exit(2)
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			dir, name, ok := strings.Cut(line, "\t")
			if !ok {
				fmt.Fprintf(os.Stderr, "checkjson: bad expected line %q\n", line)
				os.Exit(2)
			}
			expected = append(expected, expectedTest{dir: dir, name: name})
		}
		f.Close()
	}

	var events []event
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	for sc.Scan() {
		var ev event
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue // build output and other non-JSON lines
		}
		events = append(events, ev)
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "checkjson:", err)
		os.Exit(2)
	}

	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	if len(expected) == 0 {
		passes := 0
		for _, ev := range events {
			switch {
			case ev.Test != "" && ev.Action == "pass":
				passes++
			case ev.Test != "" && ev.Action == "fail":
				add("test %s %s failed", ev.Package, ev.Test)
			}
		}
		if passes == 0 {
			add("no test passed (zero pass events)")
		}
	}

	for _, want := range expected {
		passed := false
		for _, ev := range events {
			if ev.Test == "" || !want.matches(ev.Package) {
				continue
			}
			if ev.Test != want.name && !strings.HasPrefix(ev.Test, want.name+"/") {
				continue
			}
			switch ev.Action {
			case "pass":
				if ev.Test == want.name {
					passed = true
				}
			case "skip":
				add("expected test %s/%s: skip event for %s", want.dir, want.name, ev.Test)
			case "fail":
				add("expected test %s/%s: fail event for %s", want.dir, want.name, ev.Test)
			}
		}
		if !passed {
			add("expected test %s/%s: no pass event", want.dir, want.name)
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, "checkjson:", p)
		}
		os.Exit(1)
	}
}
