package money

import (
	"errors"
	"testing"
)

func TestStringParse(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Cents
	}{{"12", 1200}, {"12.3", 1230}, {"0.05", 5}, {"-1.50", -150}} {
		got, err := Parse(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("Parse(%q) = %v, %v", tc.in, got, err)
		}
	}
	if Cents(-150).String() != "-1.50" || Cents(7).String() != "0.07" {
		t.Fatal("String formatting changed")
	}
	if _, err := Parse("1.234"); !errors.Is(err, ErrSyntax) {
		t.Fatalf("err = %v", err)
	}
}

func TestMulBPS(t *testing.T) {
	if got := Cents(1000).MulBPS(725); got != 73 {
		t.Fatalf("MulBPS = %d", got)
	}
	if got := Cents(-1000).MulBPS(725); got != -73 {
		t.Fatalf("negative MulBPS = %d", got)
	}
	if got := Cents(999).Percent(10); got != 100 {
		t.Fatalf("Percent = %d", got)
	}
}

func TestAllocate(t *testing.T) {
	parts := Allocate(1001, 3)
	if len(parts) != 3 || parts[0] != 334 || parts[2] != 333 || Sum(parts...) != 1001 {
		t.Fatalf("Allocate = %v", parts)
	}
	if Allocate(5, 0) != nil {
		t.Fatal("n=0 should give nil")
	}
}
