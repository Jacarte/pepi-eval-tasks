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
	if got := strings.TrimSpace(string(b)); got != "rel-a39e7bd69d94" {
		t.Fatalf("ANSWER.txt holds %q, want %q", got, "rel-a39e7bd69d94")
	}
}
