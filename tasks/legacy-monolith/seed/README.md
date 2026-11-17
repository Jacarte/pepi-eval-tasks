# shopcore

The shop backend: one Go module, one package per business area. The HTTP
layer (`httpapi`) is thin; all business logic lives in the domain packages.

```
go build ./...
go test ./...
```

## Layout

| Package         | What it does                                              |
|-----------------|-----------------------------------------------------------|
| `audit`         | business event trail                                      |
| `billing`       | v1 invoices (wholesale portal only)                       |
| `catalog`       | product listings and categories                           |
| `clock`, `ids`  | injectable time and id generation                         |
| `config`        | deploy settings                                           |
| `coupons`       | discount codes                                            |
| `ctxkeys`       | context keys shared across layers                         |
| `customers`     | customer master and addresses                             |
| `errs`          | error annotation                                          |
| `fraud`         | risk scoring and the manual review desk                   |
| `httpapi`       | v2 JSON API                                               |
| `inventory`     | stock positions and reservations                          |
| `ledger`        | general ledger                                            |
| `loyalty`       | points programme                                          |
| `money`         | integer amounts                                           |
| `notifications` | customer messages                                         |
| `orders`        | order placement and lifecycle                             |
| `payments`      | authorizations and captures                               |
| `pricing`       | price rules                                               |
| `refunds`       | refunds                                                   |
| `reports`       | finance and ops reports                                   |
| `returns`       | return authorizations                                     |
| `scheduling`    | background jobs and delivery slots                        |
| `searchindex`   | storefront search                                         |
| `shipping`      | carriers, rate card and shipment lifecycle                |
| `suppliers`     | vendors and purchase orders                               |
| `tax`           | tax rates and exemptions                                  |
| `warehouse`     | bins and internal transfers                               |

Every store has an in-memory implementation; production adapters live in the
deploy repository.
