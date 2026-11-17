# legacy-monolith: notes for reviewers

Not copied into the agent workspace. The agent sees only `seed/` and `prompts/`.

## Layout

- `tools/gen/` deterministic generator (stdlib only, fixed per-package seeds,
  files written in sorted order, `go/format`-ed). Bulk model/store/codec/report
  code and data tables are procedural (`filler.go`, `tables.go`, `specs.go`);
  services that carry the conventions are handwritten templates in
  `tools/gen/static/` (copied verbatim, `.tmpl` suffix stripped).
  Regenerate: `cd tools/gen && go run . -out ../../seed`.
- `tools/check-gen.sh` regenerates into a temp dir and diffs against `seed/`.
- `seed/` the generated `shopcore` module (29 packages, ~18.3k non-test lines,
  ~4.1k visible test lines) plus `SPEC.md`.
- `hidden/<pkg>/*_m<N>_test.go` external test packages (`<pkg>_test`) using
  only exported seed API.
- Reference solution: outside the repo, replaces only `orders/place.go`,
  `payments/capture.go`, `refunds/issue.go`, `inventory/reserve.go`,
  `loyalty/redeem.go`.

## The undocumented convention

SPEC.md only says "follow the conventions the other services use". The
evidence is ~25 call sites in shipping (6), suppliers (4), customers (3),
warehouse, fraud, catalog, coupons, returns, ledger (2 each), their visible
tests, and `errs`/`scheduling`/`httpapi`:

1. Event name = `<snake_case of the persisted Go type>.<past tense of the
   method>`: `shipment.dispatched`, `purchase_order.approved`,
   `bin_slot.allocated`, `product_listing.published`, `coupon.redeemed`, ...
   Expected: `order.placed`, `payment.captured`, `refund.issued`,
   `stock_item.reserved`, `account.redeemed`.
2. `Event.RequestID` = `ctxkeys.RequestIDFrom(ctx)` (or
   `ctx.Value(ctxkeys.RequestID)`), `Event.Subject` = the record's `ID`.
3. `audit.Record` is called after the repository write succeeds and before
   returning; never on error paths; a Record error is returned wrapped.
4. Every returned error is `errs.Wrap(op, err)` with `op` = lower-case
   `pkg.func` (`orders.place`, `payments.capture`, `refunds.issue`,
   `inventory.reserve`, `loyalty.redeem`), wrapped once, no extra prefix.

The seed versions of the five operations return raw sentinels or
`fmt.Errorf` prefixes; `loyalty.Redeem` uses `errs.Wrap("Redeem", ...)`.

Decoy: `billing` (package doc says Deprecated, sorts first in grep) records
before the write, records on error paths, uses `ctxkeys.TraceID` as request id,
names events `charge_captured`/`invoice_created`, and wraps with
`billing.CaptureCharge`. Copying it fails every hidden test category. Other
traps: notification template ids like `order_placed`, `TraceID` set to a
different value than `RequestID` in every hidden test context.

## Which hidden test catches which mistake

Per milestone N (1 orders.Place, 2 payments.Capture, 3 refunds.Issue,
4 inventory.Reserve, 5 loyalty.Redeem), five top-level tests:

| Mistake | Test |
|---|---|
| Wrong / inconsistent event name, no event, duplicate events | `TestMN<Op>EventName` |
| Request id missing, taken from TraceID/Actor, constant; wrong Subject | `TestMN<Op>RequestIDAndSubject` |
| Record before the repository write, Record in a goroutine/after return, Record twice | `TestMN<Op>RecordsAfterWrite` (journal must be exactly `[write, record]`) |
| Audit event on validation failure or repository failure | `TestMN<Op>NoAuditOnFailure` (also requires exactly 1 event for a later success, so it is not vacuous) |
| Errors not `errs.Wrap`, wrong op (`orders.Place`, `Place`, `Redeem`), extra `fmt.Errorf` prefix, double wrap, Record error swallowed | `TestMN<Op>ErrorWrapping` (outermost `*errs.Error` op, `errors.Is` cause, exact message for sentinel/repo/audit failures, write already stored when Record fails) |

Request ids are derived from the test name and call index, so they cannot be
hard-coded; the repository decorators embed the seed interface and journal
writes, so the ordering checks do not depend on the memory store.

## M6

`go test ./...` on seed + agent changes + hidden tests: the visible suite and
all hidden tests. It fails on the seed because M1-M5 fail, so in practice it
is only reachable after M1-M5 (scores count milestones in order), and it also
fails if the agent breaks any visible test (e.g. by changing behaviour or
signatures). A plain `-skip '^TestM[1-5]'` command was rejected because it
passes on the seed, which verify-task.sh forbids.

## Known risks

- Event names for inventory/loyalty (`stock_item.reserved`,
  `account.redeemed`) follow the mechanical type-name rule; agents that pick a
  "natural" noun (`stock.reserved`, `points.redeemed`) fail EventName only.
- Generator output depends on `go/format`; a future gofmt change could make
  `check-gen.sh` report a diff without any semantic change.
