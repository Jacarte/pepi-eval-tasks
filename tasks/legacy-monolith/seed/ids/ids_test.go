package ids

import "testing"

func TestSequence(t *testing.T) {
	s := NewSequence()
	if a, b := s.Next("ord"), s.Next("ord"); a != "ord-000001" || b != "ord-000002" {
		t.Fatalf("got %s %s", a, b)
	}
	if s.Next("pay") != "pay-000001" {
		t.Fatal("prefixes must count independently")
	}
}

func TestRandom(t *testing.T) {
	a, b := Random{}.Next("x"), Random{}.Next("x")
	if a == b || !HasPrefix(a, "x") || len(a) != 18 {
		t.Fatalf("got %s %s", a, b)
	}
}
