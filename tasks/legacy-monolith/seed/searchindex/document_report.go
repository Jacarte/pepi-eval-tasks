package searchindex

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DocumentSummary aggregates a batch of documents for the reports package.
type DocumentSummary struct {
	Count      int
	ByStatus   map[DocumentStatus]int
	TotalBoost int64
	Earliest   time.Time
	Latest     time.Time
}

// SummarizeDocuments folds items into a DocumentSummary. Nil entries are skipped.
func SummarizeDocuments(items []*Document) DocumentSummary {
	sum := DocumentSummary{ByStatus: make(map[DocumentStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalBoost += int64(v.Boost)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopDocumentsByBoost returns the n items with the largest boost, largest first.
// Ties keep their input order.
func TopDocumentsByBoost(items []*Document, n int) []*Document {
	out := append([]*Document(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Boost > out[j].Boost })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupDocumentsByKind buckets items by kind.
func GroupDocumentsByKind(items []*Document) map[string][]*Document {
	out := make(map[string][]*Document)
	for _, v := range items {
		out[v.Kind] = append(out[v.Kind], v)
	}
	return out
}

// DocumentsCreatedBetween keeps items created in [from, to).
func DocumentsCreatedBetween(items []*Document, from, to time.Time) []*Document {
	var out []*Document
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteDocumentsCSV writes items as CSV with a header row.
// TODO(mk): add metrics here (OPS-1423).
func WriteDocumentsCSV(out io.Writer, items []*Document) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "kind", "title", "body", "boost", "tags", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.Kind,
			v.Title,
			v.Body,
			strconv.Itoa(v.Boost),
			csvJoin(v.Tags),
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

func csvJoin(vals []string) string {
	out := ""
	for i, v := range vals {
		if i > 0 {
			out += "|"
		}
		out += v
	}
	return out
}

// DiffDocument lists the fields that differ between a and b.
func DiffDocument(a, b *Document) []FieldChange {
	var out []FieldChange
	if a.Kind != b.Kind {
		out = append(out, FieldChange{"kind", fmt.Sprint(a.Kind), fmt.Sprint(b.Kind)})
	}
	if a.Title != b.Title {
		out = append(out, FieldChange{"title", fmt.Sprint(a.Title), fmt.Sprint(b.Title)})
	}
	if a.Body != b.Body {
		out = append(out, FieldChange{"body", fmt.Sprint(a.Body), fmt.Sprint(b.Body)})
	}
	if a.Boost != b.Boost {
		out = append(out, FieldChange{"boost", fmt.Sprint(a.Boost), fmt.Sprint(b.Boost)})
	}
	if strings.Join(a.Tags, ",") != strings.Join(b.Tags, ",") {
		out = append(out, FieldChange{"tags", strings.Join(a.Tags, ","), strings.Join(b.Tags, ",")})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
