package suppliers

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"
)

// SupplierSummary aggregates a batch of suppliers for the reports package.
type SupplierSummary struct {
	Count             int
	ByStatus          map[SupplierStatus]int
	TotalLeadTimeDays int64
	Earliest          time.Time
	Latest            time.Time
}

// SummarizeSuppliers folds items into a SupplierSummary. Nil entries are skipped.
func SummarizeSuppliers(items []*Supplier) SupplierSummary {
	sum := SupplierSummary{ByStatus: make(map[SupplierStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalLeadTimeDays += int64(v.LeadTimeDays)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopSuppliersByLeadTimeDays returns the n items with the largest lead time days, largest first.
// Ties keep their input order.
func TopSuppliersByLeadTimeDays(items []*Supplier, n int) []*Supplier {
	out := append([]*Supplier(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].LeadTimeDays > out[j].LeadTimeDays })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupSuppliersByName buckets items by name.
func GroupSuppliersByName(items []*Supplier) map[string][]*Supplier {
	out := make(map[string][]*Supplier)
	for _, v := range items {
		out[v.Name] = append(out[v.Name], v)
	}
	return out
}

// SuppliersCreatedBetween keeps items created in [from, to).
func SuppliersCreatedBetween(items []*Supplier, from, to time.Time) []*Supplier {
	var out []*Supplier
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteSuppliersCSV writes items as CSV with a header row.
func WriteSuppliersCSV(out io.Writer, items []*Supplier) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "name", "country", "contact_email", "lead_time_days", "preferred", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.Name,
			v.Country,
			v.ContactEmail,
			strconv.Itoa(v.LeadTimeDays),
			strconv.FormatBool(v.Preferred),
			v.Status.String(),
			formatTime(v.CreatedAt),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// DiffSupplier lists the fields that differ between a and b.
func DiffSupplier(a, b *Supplier) []FieldChange {
	var out []FieldChange
	if a.Name != b.Name {
		out = append(out, FieldChange{"name", fmt.Sprint(a.Name), fmt.Sprint(b.Name)})
	}
	if a.Country != b.Country {
		out = append(out, FieldChange{"country", fmt.Sprint(a.Country), fmt.Sprint(b.Country)})
	}
	if a.ContactEmail != b.ContactEmail {
		out = append(out, FieldChange{"contact_email", fmt.Sprint(a.ContactEmail), fmt.Sprint(b.ContactEmail)})
	}
	if a.LeadTimeDays != b.LeadTimeDays {
		out = append(out, FieldChange{"lead_time_days", fmt.Sprint(a.LeadTimeDays), fmt.Sprint(b.LeadTimeDays)})
	}
	if a.Preferred != b.Preferred {
		out = append(out, FieldChange{"preferred", fmt.Sprint(a.Preferred), fmt.Sprint(b.Preferred)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
