// Package promo resolves promotion codes against the promotion book.
package promo

// Resolve returns the codes that exist in book, once each, in the order they were entered.
func Resolve(codes []string, book map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, c := range codes {
		if book[c] && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}
