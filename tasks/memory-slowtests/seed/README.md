# shopbill

Small invoicing service.

| Path | Purpose |
|---|---|
| `cmd/app` | prints a sample invoice |
| `internal/billing` | invoice lines, tax and totals |
| `internal/payments` | charges an invoice total through a payment gateway |
| `internal/ledger` | books charged invoices and reconciles them |
| `internal/notify` | formats customer messages |
| `internal/testlock` | test helper shared by the integration tests |

## Billing rules

- Amounts are integer cents.
- The tax of a line is `quantity * unit price * rate`, rounded to the nearest cent, halves away from zero (0.525 becomes 0.53, -0.525 becomes -0.53). The rate is in basis points (825 = 8.25%).
- Refund lines have a negative quantity and follow the same rule, so a refund returns exactly the tax that the sale charged.
- The invoice total is the sum of the line amounts plus the sum of the line taxes.

## Tests

Run everything:

    make test

Run one package:

    make test-pkg PKG=./internal/notify
