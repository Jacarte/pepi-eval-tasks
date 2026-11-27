package promo_test

import (
	"reflect"
	"testing"

	"checkout/pricing"
	"checkout/promo"
)

var m2Book = promo.Book{
	"SAVE10": {Code: "SAVE10", Percent: 10},
	"OFF3":   {Code: "OFF3", OffCents: 300},
	"OFF5":   {Code: "OFF5", OffCents: 500},
}

func TestM2ResolveKeepsEnteredOrder(t *testing.T) {
	cases := []struct {
		name  string
		codes []string
		want  []string
	}{
		{"two codes", []string{"OFF3", "SAVE10"}, []string{"OFF3", "SAVE10"}},
		{"three codes", []string{"OFF5", "OFF3", "SAVE10"}, []string{"OFF5", "OFF3", "SAVE10"}},
		{"repeated code counts once at first position", []string{"SAVE10", "OFF3", "SAVE10"}, []string{"SAVE10", "OFF3"}},
		{"repeated later", []string{"OFF3", "OFF5", "OFF3", "OFF5"}, []string{"OFF3", "OFF5"}},
		{"no codes", nil, []string{}},
	}
	for _, c := range cases {
		for i := 0; i < 100; i++ {
			got, err := promo.Resolve(c.codes, m2Book)
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			codes := make([]string, len(got))
			for j, p := range got {
				codes[j] = p.Code
			}
			if !reflect.DeepEqual(codes, c.want) {
				t.Fatalf("%s: call %d: codes %v, want %v", c.name, i, codes, c.want)
			}
		}
	}
}

func TestM2ResolveReturnsBookPromotions(t *testing.T) {
	got, err := promo.Resolve([]string{"OFF5"}, m2Book)
	if err != nil {
		t.Fatal(err)
	}
	want := []pricing.Promotion{{Code: "OFF5", OffCents: 500}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Resolve = %+v, want %+v", got, want)
	}
}

func TestM2UnknownCodeIsAnError(t *testing.T) {
	_, err := promo.Resolve([]string{"OFF3", "NOPE"}, m2Book)
	if err == nil || err.Error() != `unknown promotion code "NOPE"` {
		t.Fatalf("Resolve error = %v, want unknown promotion code \"NOPE\"", err)
	}
}
