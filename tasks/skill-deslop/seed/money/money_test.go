package money

import "testing"

func TestParseAndString(t *testing.T) {
	m, err := Parse("12.34 USD")
	if err != nil {
		t.Fatal(err)
	}
	if m != (Money{Cents: 1234, Currency: "USD"}) {
		t.Fatalf("Parse = %+v", m)
	}
	if got := m.String(); got != "12.34 USD" {
		t.Fatalf("String = %q", got)
	}
}

func TestParseRejectsOneDecimal(t *testing.T) {
	if _, err := Parse("12.3 USD"); err == nil {
		t.Fatal("want an error for one decimal place")
	}
}
