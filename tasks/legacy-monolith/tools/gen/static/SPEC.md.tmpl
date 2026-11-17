# Audit trail for checkout operations

Compliance has asked for an audit trail on five operations that change data
but currently record nothing:

| Operation                   | Package     |
|-----------------------------|-------------|
| `(*orders.Service).Place`        | `orders`    |
| `(*payments.Service).Capture`    | `payments`  |
| `(*refunds.Service).Issue`       | `refunds`   |
| `(*inventory.Service).Reserve`   | `inventory` |
| `(*loyalty.Service).Redeem`      | `loyalty`   |

## Requirements

1. Every successful call to one of these operations records exactly one audit
   event through the `audit.Recorder` the service was constructed with (the
   `shopcore/audit` package).
2. These operations were written before the rest of the codebase settled on
   its conventions. While adding the audit trail, bring them in line: the
   audit events, how they are correlated with the request, and how the
   operations report errors must follow the conventions the other services in
   this repository already use. There is no separate style guide; the existing
   code is the reference.
3. Do not change the exported signatures of these operations, of their
   services' constructors, or of the repository interfaces they use.
4. Existing behaviour (validation rules, pricing, state changes) must not
   change, and `go test ./...` must keep passing.
