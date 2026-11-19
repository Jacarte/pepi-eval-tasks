package inbox

import (
	"os"
	"path"
	"slices"
	"strings"
	"testing"
)

func TestM1DamagedList(t *testing.T) {
	b, err := os.ReadFile("../../DAMAGED.txt")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, line := range strings.Split(string(b), "\n") {
		if name := path.Base(strings.TrimSpace(line)); name != "." {
			if !slices.Contains(got, name) {
				got = append(got, name)
			}
		}
	}
	slices.Sort(got)
	want := []string{"in-02.rec", "in-06.rec", "in-07.rec", "in-10.rec"}
	if !slices.Equal(got, want) {
		t.Fatalf("DAMAGED.txt lists %q, want %q", got, want)
	}
}
