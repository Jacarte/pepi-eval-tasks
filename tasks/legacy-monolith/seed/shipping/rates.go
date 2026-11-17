package shipping

import (
	"errors"
	"strings"

	"shopcore/money"
)

type zoneRate struct {
	Carrier  string
	Zone     int
	MaxGrams int
	Cents    money.Cents
}

// ErrNoRate is returned when the rate card has no row for the parcel.
var ErrNoRate = errors.New("shipping: no rate for parcel")

func knownCarrier(c string) bool {
	c = strings.ToLower(c)
	for _, r := range zoneRates {
		if r.Carrier == c {
			return true
		}
	}
	return false
}

// Quote returns the cheapest-matching rate for a parcel. The rate card is
// ordered by weight so the first row that fits wins.
func Quote(carrier string, zone, grams int) (money.Cents, error) {
	carrier = strings.ToLower(carrier)
	for _, r := range zoneRates {
		if r.Carrier == carrier && r.Zone == zone && grams <= r.MaxGrams {
			return r.Cents, nil
		}
	}
	return 0, ErrNoRate
}

// ZoneFor maps a destination country to a rate zone. Anything we do not know
// is zone 4; ops prefer overcharging to eating the difference.
func ZoneFor(country string) int {
	switch strings.ToUpper(country) {
	case "US":
		return 1
	case "CA", "MX":
		return 2
	case "GB", "DE", "FR", "NL", "ES", "IT", "IE", "BE", "AT":
		return 3
	}
	return 4
}
