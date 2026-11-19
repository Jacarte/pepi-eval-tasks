package billing

// divRound returns n/d rounded to the nearest integer, halves away from zero. d must be positive.
func divRound(n, d int64) int64 {
	return (n + d/2 - 1) / d
}
