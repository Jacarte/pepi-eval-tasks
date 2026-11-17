# Audit trail for back-office operations

Compliance has asked for an audit trail on twelve operations that change data
but currently record nothing:

| Operation                           | Package     |
|-------------------------------------|-------------|
| `(*orders.Service).Place`           | `orders`    |
| `(*payments.Service).Capture`       | `payments`  |
| `(*refunds.Service).Issue`          | `refunds`   |
| `(*inventory.Service).Reserve`      | `inventory` |
| `(*loyalty.Service).Redeem`         | `loyalty`   |
| `(*returns.Service).Request`        | `returns`   |
| `(*warehouse.Service).PlanTransfer` | `warehouse` |
| `(*customers.Service).AddAddress`   | `customers` |
| `(*suppliers.Service).Cancel`       | `suppliers` |
| `(*coupons.Service).Expire`         | `coupons`   |
| `(*ledger.Service).Amend`           | `ledger`    |
| `(*catalog.Service).Reprice`        | `catalog`   |

## Requirements

1. Every successful call to one of these operations records exactly one audit
   event through the `audit.Recorder` the service was constructed with (the
   `shopcore/audit` package).
2. Several of these operations were written before the rest of the codebase
   settled on its conventions. While adding the audit trail, bring them in
   line: the audit events, how they are correlated with the request, and how
   the operations report errors must follow the conventions of this
   repository. There is no separate style guide; the existing code and its
   tests are the reference.
3. Do not change the exported signatures of these operations, of their
   services' constructors, or of the repository interfaces they use.
4. Existing behaviour (validation rules, pricing, state changes) must not
   change, and `go test ./...` must keep passing. The error values and
   messages returned by the twelve operations are governed by requirement 2
   and are excepted from this rule: the underlying causes must stay
   detectable with `errors.Is`, but how they are wrapped follows the codebase
   convention.
