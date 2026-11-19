package main

import "testing"

func TestSampleHasTwoLines(t *testing.T) {
	if n := len(sample().Lines); n != 2 {
		t.Fatalf("sample has %d lines, want 2", n)
	}
}
