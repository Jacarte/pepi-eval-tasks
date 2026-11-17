package customers

import (
	"fmt"
	"time"
)

// AddressSummary aggregates a batch of addresses for the reports package.
type AddressSummary struct {
	Count    int
	ByStatus map[AddressStatus]int
	Earliest time.Time
	Latest   time.Time
}

// SummarizeAddresses folds items into a AddressSummary. Nil entries are skipped.
func SummarizeAddresses(items []*Address) AddressSummary {
	sum := AddressSummary{ByStatus: make(map[AddressStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// GroupAddressesByCustomerID buckets items by customer id.
func GroupAddressesByCustomerID(items []*Address) map[string][]*Address {
	out := make(map[string][]*Address)
	for _, v := range items {
		out[v.CustomerID] = append(out[v.CustomerID], v)
	}
	return out
}

// AddressesCreatedBetween keeps items created in [from, to).
func AddressesCreatedBetween(items []*Address, from, to time.Time) []*Address {
	var out []*Address
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// DiffAddress lists the fields that differ between a and b.
func DiffAddress(a, b *Address) []FieldChange {
	var out []FieldChange
	if a.CustomerID != b.CustomerID {
		out = append(out, FieldChange{"customer_id", fmt.Sprint(a.CustomerID), fmt.Sprint(b.CustomerID)})
	}
	if a.Line1 != b.Line1 {
		out = append(out, FieldChange{"line1", fmt.Sprint(a.Line1), fmt.Sprint(b.Line1)})
	}
	if a.City != b.City {
		out = append(out, FieldChange{"city", fmt.Sprint(a.City), fmt.Sprint(b.City)})
	}
	if a.Postcode != b.Postcode {
		out = append(out, FieldChange{"postcode", fmt.Sprint(a.Postcode), fmt.Sprint(b.Postcode)})
	}
	if a.Country != b.Country {
		out = append(out, FieldChange{"country", fmt.Sprint(a.Country), fmt.Sprint(b.Country)})
	}
	if a.Primary != b.Primary {
		out = append(out, FieldChange{"primary", fmt.Sprint(a.Primary), fmt.Sprint(b.Primary)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
