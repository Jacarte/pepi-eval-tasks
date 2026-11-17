package coupons

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"
)

// CouponSummary aggregates a batch of coupons for the reports package.
type CouponSummary struct {
	Count           int
	ByStatus        map[CouponStatus]int
	TotalPercentOff int64
	TotalMaxUses    int64
	TotalUses       int64
	Earliest        time.Time
	Latest          time.Time
}

// SummarizeCoupons folds items into a CouponSummary. Nil entries are skipped.
func SummarizeCoupons(items []*Coupon) CouponSummary {
	sum := CouponSummary{ByStatus: make(map[CouponStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalPercentOff += int64(v.PercentOff)
		sum.TotalMaxUses += int64(v.MaxUses)
		sum.TotalUses += int64(v.Uses)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopCouponsByPercentOff returns the n items with the largest percent off, largest first.
// Ties keep their input order.
func TopCouponsByPercentOff(items []*Coupon, n int) []*Coupon {
	out := append([]*Coupon(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].PercentOff > out[j].PercentOff })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupCouponsByCode buckets items by code.
func GroupCouponsByCode(items []*Coupon) map[string][]*Coupon {
	out := make(map[string][]*Coupon)
	for _, v := range items {
		out[v.Code] = append(out[v.Code], v)
	}
	return out
}

// CouponsCreatedBetween keeps items created in [from, to).
func CouponsCreatedBetween(items []*Coupon, from, to time.Time) []*Coupon {
	var out []*Coupon
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteCouponsCSV writes items as CSV with a header row.
func WriteCouponsCSV(out io.Writer, items []*Coupon) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "code", "percent_off", "max_uses", "uses", "expires_at", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.Code,
			strconv.Itoa(v.PercentOff),
			strconv.Itoa(v.MaxUses),
			strconv.Itoa(v.Uses),
			fmtTime(v.ExpiresAt),
			v.Status.String(),
			fmtTime(v.CreatedAt),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// DiffCoupon lists the fields that differ between a and b.
func DiffCoupon(a, b *Coupon) []FieldChange {
	var out []FieldChange
	if a.Code != b.Code {
		out = append(out, FieldChange{"code", fmt.Sprint(a.Code), fmt.Sprint(b.Code)})
	}
	if a.PercentOff != b.PercentOff {
		out = append(out, FieldChange{"percent_off", fmt.Sprint(a.PercentOff), fmt.Sprint(b.PercentOff)})
	}
	if a.MaxUses != b.MaxUses {
		out = append(out, FieldChange{"max_uses", fmt.Sprint(a.MaxUses), fmt.Sprint(b.MaxUses)})
	}
	if a.Uses != b.Uses {
		out = append(out, FieldChange{"uses", fmt.Sprint(a.Uses), fmt.Sprint(b.Uses)})
	}
	if !a.ExpiresAt.Equal(b.ExpiresAt) {
		out = append(out, FieldChange{"expires_at", fmtTime(a.ExpiresAt), fmtTime(b.ExpiresAt)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
