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
	if got := strings.TrimSpace(string(b)); got != "217" {
		t.Fatalf("ANSWER.txt holds %q, want %q", got, "217")
	}
}
