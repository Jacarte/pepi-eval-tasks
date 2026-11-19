package main

import (
	"os"
	"strings"
	"testing"
)

func TestM1Answer(t *testing.T) {
	b, err := os.ReadFile("ANSWER.txt")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSpace(string(b)), "\n")
	for i := range got {
		got[i] = strings.TrimSpace(got[i])
	}
	want := []string{"Heron", "orbit-night"}
	if !strings.EqualFold(strings.Join(got, "\n"), strings.Join(want, "\n")) {
		t.Fatalf("ANSWER.txt holds lines %q, want %q", got, want)
	}
}
