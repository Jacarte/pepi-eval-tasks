package invoice

import "testing"

func TestParseLegacyString(t *testing.T) {
	m, err := ParseLegacy("1.50 EUR")
	if err != nil {
		t.Fatal(err)
	}
	if m.Cents != 150 || m.Currency != "EUR" {
		t.Fatalf("ParseLegacy = %+v", m)
	}
}
