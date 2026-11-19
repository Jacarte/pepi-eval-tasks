// Command app prints a sample invoice summary.
package main

import (
	"fmt"

	"shopbill/internal/billing"
	"shopbill/internal/notify"
)

func sample() billing.Invoice {
	return billing.Invoice{ID: "S-1", Lines: []billing.Line{
		{Description: "widget", UnitCents: 1999, Quantity: 3, TaxBP: 825},
		{Description: "return", UnitCents: 1999, Quantity: -1, TaxBP: 825},
	}}
}

func main() {
	fmt.Println(notify.Format(sample()))
}
