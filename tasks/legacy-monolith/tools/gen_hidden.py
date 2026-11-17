#!/usr/bin/env python3
"""Writes hidden/<pkg>/*_m<N>_test.go for the twelve audit milestones.

Usage: python3 tools/gen_hidden.py   (from tasks/legacy-monolith)

Every milestone gets the same six checks; the per-package parts (fixtures,
the repository write to journal, invalid calls, expected subject/attrs) are
the Go snippets below.
"""
import os

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "..", "hidden")

M = []

def m(**kw):
    M.append(kw)

m(n=1, pkg="orders", file="place_audit", fn="Place", op="orders.place", event="order.placed",
  iface="orders.Repository", field="Repository", write="Save", vtype="*orders.Order",
  base="orders.NewMemoryRepo()", basetype="*orders.MemoryRepo", extra_fields="", extra_init="",
  construct="orders.NewService(orders.Config{Repo: repo, Audit: rec, Clock: clk, IDs: ids.NewSequence()})",
  rtype="*orders.Order",
  prepare="",
  invoke='return env.svc.Place(ctx, orders.PlaceRequest{CustomerID: fmt.Sprintf("cus-%d", n), Region: "US-CA", Lines: []orders.Line{{SKU: "SKU-1", Qty: 2, UnitPrice: 1999}, {SKU: "SKU-2", Qty: n, UnitPrice: 500}}})',
  subject="res.ID", attrs='map[string]string{"customer_id": fmt.Sprintf("cus-%d", n)}',
  bad='''lines := []orders.Line{{SKU: "SKU-1", Qty: 1, UnitPrice: 10}}
	_, e1 := env.svc.Place(ctx, orders.PlaceRequest{Region: "US-CA", Lines: lines})
	_, e2 := env.svc.Place(ctx, orders.PlaceRequest{CustomerID: "cus-x", Region: "US-CA"})
	_, e3 := env.svc.Place(ctx, orders.PlaceRequest{CustomerID: "cus-x", Region: "US-CA", Lines: []orders.Line{{SKU: "S", Qty: -1}}})
	_, e4 := env.svc.Place(ctx, orders.PlaceRequest{CustomerID: "cus-x", Region: "XX-NOWHERE", Lines: lines})
	return []m{N}Bad{
		{"missing customer", e1, orders.ErrMissingCustomer, true},
		{"empty order", e2, orders.ErrEmptyOrder, true},
		{"invalid quantity", e3, orders.ErrInvalidQuantity, false},
		{"unknown tax region", e4, nil, false},
	}''',
  stored='list, _ := env.base.ListByCustomer(ctx, fmt.Sprintf("cus-%d", n))\n\treturn len(list) == 1')

m(n=2, pkg="payments", file="capture_audit", fn="Capture", op="payments.capture", event="payment.captured",
  iface="payments.Repository", field="Repository", write="Save", vtype="*payments.Payment",
  base="payments.NewMemoryRepo()", basetype="*payments.MemoryRepo", extra_fields="", extra_init="",
  construct="payments.NewService(repo, rec, clk)", rtype="*payments.Payment",
  prepare='_ = env.base.Save(ctx, &payments.Payment{ID: fmt.Sprintf("pay-%d", n), OrderID: fmt.Sprintf("ord-%d", n), Processor: "stripe", Amount: 5000, Status: payments.StatusAuthorized})',
  invoke='return env.svc.Capture(ctx, fmt.Sprintf("pay-%d", n), 3000)',
  subject='fmt.Sprintf("pay-%d", n)', attrs='map[string]string{"order_id": fmt.Sprintf("ord-%d", n), "processor": "stripe"}',
  bad='''m{N}Prepare(env, ctx, 70)
	_ = env.base.Save(ctx, &payments.Payment{ID: "pay-done", OrderID: "ord-done", Processor: "adyen", Amount: 100, Captured: 100, Status: payments.StatusCaptured})
	_, e1 := env.svc.Capture(ctx, "pay-70", 0)
	_, e2 := env.svc.Capture(ctx, "pay-70", 5001)
	_, e3 := env.svc.Capture(ctx, "pay-done", 10)
	_, e4 := env.svc.Capture(ctx, "pay-missing", 10)
	return []m{N}Bad{
		{"invalid amount", e1, payments.ErrInvalidAmount, true},
		{"exceeds authorization", e2, payments.ErrExceedsAuthorization, true},
		{"already captured", e3, payments.ErrInvalidState, true},
		{"unknown payment", e4, payments.ErrNotFound, false},
	}''',
  stored='p, err := env.base.Get(ctx, fmt.Sprintf("pay-%d", n))\n\treturn err == nil && p.Status == payments.StatusCaptured')

m(n=3, pkg="refunds", file="issue_audit", fn="Issue", op="refunds.issue", event="refund.issued",
  iface="refunds.Repository", field="Repository", write="Insert", vtype="*refunds.Refund",
  base="refunds.NewMemoryRepo()", basetype="*refunds.MemoryRepo", extra_fields="", extra_init="",
  construct="refunds.New(refunds.Deps{Repo: repo, Audit: rec, Clock: clk, IDs: ids.NewSequence(), MaxAmount: 100000})",
  rtype="*refunds.Refund", prepare="",
  invoke='return env.svc.Issue(ctx, refunds.IssueRequest{PaymentID: fmt.Sprintf("pay-%d", n), OrderID: fmt.Sprintf("ord-%d", n), Amount: 1500, Reason: refunds.ReasonDamaged})',
  subject="res.ID", attrs='map[string]string{"payment_id": fmt.Sprintf("pay-%d", n), "reason": "damaged"}',
  bad='''_, e1 := env.svc.Issue(ctx, refunds.IssueRequest{Amount: 100, Reason: refunds.ReasonDamaged})
	_, e2 := env.svc.Issue(ctx, refunds.IssueRequest{PaymentID: "pay-x", Reason: refunds.ReasonDamaged})
	_, e3 := env.svc.Issue(ctx, refunds.IssueRequest{PaymentID: "pay-x", Amount: 100, Reason: "because"})
	_, e4 := env.svc.Issue(ctx, refunds.IssueRequest{PaymentID: "pay-x", Amount: 100001, Reason: refunds.ReasonGoodwill})
	return []m{N}Bad{
		{"missing payment", e1, refunds.ErrMissingPayment, true},
		{"invalid amount", e2, refunds.ErrInvalidAmount, true},
		{"unknown reason", e3, refunds.ErrUnknownReason, false},
		{"over limit", e4, refunds.ErrOverLimit, true},
	}''',
  stored='list, _ := env.base.ForPayment(ctx, fmt.Sprintf("pay-%d", n))\n\treturn len(list) == 1')

m(n=4, pkg="inventory", file="reserve_audit", fn="Reserve", op="inventory.reserve", event="stock_item.reserved",
  iface="inventory.Repository", field="Repository", write="Put", vtype="*inventory.StockItem",
  base="inventory.NewMemoryRepo()", basetype="*inventory.MemoryRepo", extra_fields="", extra_init="",
  construct="inventory.NewService(repo, inventory.WithAudit(rec), inventory.WithClock(clk))",
  rtype="*inventory.StockItem",
  prepare='_ = env.base.Put(ctx, &inventory.StockItem{ID: fmt.Sprintf("SKU-%d", n), Warehouse: fmt.Sprintf("wh-%d", n), OnHand: 10})',
  invoke='return env.svc.Reserve(ctx, fmt.Sprintf("SKU-%d", n), 2+n)',
  subject='fmt.Sprintf("SKU-%d", n)', attrs='map[string]string{"warehouse": fmt.Sprintf("wh-%d", n), "qty": strconv.Itoa(2 + n)}',
  bad='''_ = env.base.Put(ctx, &inventory.StockItem{ID: "SKU-low", Warehouse: "ams", OnHand: 3, Reserved: 2})
	_, e1 := env.svc.Reserve(ctx, "SKU-low", 0)
	_, e2 := env.svc.Reserve(ctx, "SKU-low", 2)
	_, e3 := env.svc.Reserve(ctx, "SKU-missing", 1)
	return []m{N}Bad{
		{"invalid quantity", e1, inventory.ErrInvalidQuantity, true},
		{"insufficient stock", e2, inventory.ErrInsufficientStock, true},
		{"unknown sku", e3, inventory.ErrNotFound, false},
	}''',
  stored='it, err := env.base.Get(ctx, fmt.Sprintf("SKU-%d", n))\n\treturn err == nil && it.Reserved == 2+n')

m(n=5, pkg="loyalty", file="redeem_audit", fn="Redeem", op="loyalty.redeem", event="account.redeemed",
  iface="loyalty.Repository", field="Repository", write="Store", vtype="*loyalty.Account",
  base="loyalty.NewMemoryRepo()", basetype="*loyalty.MemoryRepo", extra_fields="", extra_init="",
  construct="loyalty.NewService(repo, rec, clk)", rtype="*loyalty.Account",
  prepare='_ = env.base.Store(ctx, &loyalty.Account{ID: fmt.Sprintf("acct-%d", n), CustomerID: fmt.Sprintf("cus-%d", n), Balance: 1000, Lifetime: 1000, TierName: "bronze"})',
  invoke='return env.svc.Redeem(ctx, fmt.Sprintf("acct-%d", n), int64(100*n))',
  subject='fmt.Sprintf("cus-%d", n)', attrs='map[string]string{"account_id": fmt.Sprintf("acct-%d", n), "points": strconv.Itoa(100 * n)}',
  bad='''m{N}Prepare(env, ctx, 60)
	_, e1 := env.svc.Redeem(ctx, "acct-60", 0)
	_, e2 := env.svc.Redeem(ctx, "acct-60", 1001)
	_, e3 := env.svc.Redeem(ctx, "acct-missing", 1)
	return []m{N}Bad{
		{"invalid points", e1, loyalty.ErrInvalidPoints, true},
		{"insufficient points", e2, loyalty.ErrInsufficientPoints, true},
		{"unknown account", e3, loyalty.ErrNotFound, false},
	}''',
  stored='a, err := env.base.Load(ctx, fmt.Sprintf("acct-%d", n))\n\treturn err == nil && a.Balance == int64(1000-100*n)')

m(n=6, pkg="returns", file="request_audit", fn="Request", op="returns.request", event="return_request.requested",
  iface="returns.ReturnRequestStore", field="ReturnRequestStore", write="Save", vtype="*returns.ReturnRequest",
  base="returns.NewMemoryReturnRequestStore()", basetype="*returns.MemoryReturnRequestStore", extra_fields="", extra_init="",
  construct="returns.NewService(repo, rec, clk, ids.NewSequence())", rtype="*returns.ReturnRequest", prepare="",
  invoke='return env.svc.Request(ctx, fmt.Sprintf("ord-%d", n), fmt.Sprintf("SKU-%d", n), n, "too small")',
  subject="res.ID", attrs="nil",
  bad='''_, e1 := env.svc.Request(ctx, "ord-x", "SKU-x", 0, "none")
	_, e2 := env.svc.Request(ctx, "ord-x", "SKU-x", -3, "none")
	return []m{N}Bad{
		{"zero quantity", e1, returns.ErrBadQuantity, true},
		{"negative quantity", e2, returns.ErrBadQuantity, true},
	}''',
  stored='return env.base.Len() == 1')

m(n=7, pkg="warehouse", file="transfer_audit", fn="PlanTransfer", op="warehouse.plantransfer", event="transfer.planned",
  iface="warehouse.TransferStore", field="TransferStore", write="Store", vtype="*warehouse.Transfer",
  base="warehouse.NewMemoryTransferStore()", basetype="*warehouse.MemoryTransferStore",
  extra_fields="bins *warehouse.MemoryBinSlotStore", extra_init="bins := warehouse.NewMemoryBinSlotStore()",
  construct="warehouse.NewService(bins, repo, rec, clk, ids.NewSequence())", rtype="*warehouse.Transfer",
  prepare='''for _, id := range []string{fmt.Sprintf("A-%d", n), fmt.Sprintf("B-%d", n)} {
		_ = env.bins.Store(ctx, &warehouse.BinSlot{ID: id, Zone: "A", Aisle: 1, Level: 1, Capacity: 20, Status: warehouse.BinSlotStatusFree})
	}''',
  invoke='return env.svc.PlanTransfer(ctx, fmt.Sprintf("A-%d", n), fmt.Sprintf("B-%d", n), fmt.Sprintf("SKU-%d", n), 5)',
  subject="res.ID", attrs="nil",
  bad='''m{N}Prepare(env, ctx, 80)
	_ = env.bins.Store(ctx, &warehouse.BinSlot{ID: "B-blocked", Zone: "B", Aisle: 2, Level: 1, Capacity: 20, Status: warehouse.BinSlotStatusBlocked})
	_, e1 := env.svc.PlanTransfer(ctx, "A-80", "A-80", "SKU-1", 1)
	_, e2 := env.svc.PlanTransfer(ctx, "A-80", "B-80", "SKU-1", 21)
	_, e3 := env.svc.PlanTransfer(ctx, "A-80", "B-blocked", "SKU-1", 1)
	_, e4 := env.svc.PlanTransfer(ctx, "A-80", "B-missing", "SKU-1", 1)
	return []m{N}Bad{
		{"same bin", e1, warehouse.ErrSameBin, true},
		{"over capacity", e2, warehouse.ErrOverCapacity, true},
		{"blocked bin", e3, warehouse.ErrBinBlocked, true},
		{"unknown bin", e4, warehouse.ErrNotFound, false},
	}''',
  stored='return env.base.Len() == 1')

m(n=8, pkg="customers", file="address_audit", fn="AddAddress", op="customers.addaddress", event="address.added",
  iface="customers.AddressStore", field="AddressStore", write="Put", vtype="*customers.Address",
  base="customers.NewMemoryAddressStore()", basetype="*customers.MemoryAddressStore",
  extra_fields="people *customers.MemoryCustomerStore", extra_init="people := customers.NewMemoryCustomerStore()",
  construct="customers.NewService(people, repo, rec, clk, ids.NewSequence())", rtype="*customers.Address",
  prepare='_ = env.people.Put(ctx, &customers.Customer{ID: fmt.Sprintf("cus-%d", n), Email: fmt.Sprintf("c%d@example.com", n), DisplayName: "C", Status: customers.CustomerStatusActive})',
  invoke='''a := &customers.Address{CustomerID: fmt.Sprintf("cus-%d", n), Line1: "1 Main St", City: "Utrecht", Postcode: "3511", Country: "NL"}
	return a, env.svc.AddAddress(ctx, a)''',
  subject="res.ID", attrs="nil",
  bad='''_, e1 := m{N}Invoke(env, ctx, 404)
	return []m{N}Bad{
		{"unknown customer", e1, customers.ErrNotFound, false},
	}''',
  stored='return env.base.Len() == 1')

m(n=9, pkg="suppliers", file="cancel_audit", fn="Cancel", op="suppliers.cancel", event="purchase_order.cancelled",
  iface="suppliers.PurchaseOrderStore", field="PurchaseOrderStore", write="Save", vtype="*suppliers.PurchaseOrder",
  base="suppliers.NewMemoryPurchaseOrderStore()", basetype="*suppliers.MemoryPurchaseOrderStore", extra_fields="", extra_init="",
  construct="suppliers.NewService(suppliers.NewMemorySupplierStore(), repo, rec, clk)", rtype="*suppliers.PurchaseOrder",
  prepare='_ = env.base.Save(ctx, &suppliers.PurchaseOrder{ID: fmt.Sprintf("po-%d", n), SupplierID: fmt.Sprintf("sup-%d", n), Total: 1000, Lines: 1, Status: suppliers.PurchaseOrderStatusDraft})',
  invoke='return env.svc.Cancel(ctx, fmt.Sprintf("po-%d", n))',
  subject='fmt.Sprintf("po-%d", n)', attrs="nil",
  bad='''_ = env.base.Save(ctx, &suppliers.PurchaseOrder{ID: "po-in", SupplierID: "sup-1", Total: 1000, Lines: 1, Status: suppliers.PurchaseOrderStatusReceived})
	_, e1 := env.svc.Cancel(ctx, "po-in")
	_, e2 := env.svc.Cancel(ctx, "po-missing")
	return []m{N}Bad{
		{"already received", e1, suppliers.ErrNotCancellable, true},
		{"unknown purchase order", e2, suppliers.ErrNotFound, false},
	}''',
  stored='po, err := env.base.Find(ctx, fmt.Sprintf("po-%d", n))\n\treturn err == nil && po.Status == suppliers.PurchaseOrderStatusCancelled')

m(n=10, pkg="coupons", file="expire_audit", fn="Expire", op="coupons.expire", event="coupon.expired",
  iface="coupons.CouponStore", field="CouponStore", write="Store", vtype="*coupons.Coupon",
  base="coupons.NewMemoryCouponStore()", basetype="*coupons.MemoryCouponStore", extra_fields="", extra_init="",
  construct="coupons.NewService(repo, rec, clk)", rtype="*coupons.Coupon",
  prepare='_ = env.base.Store(ctx, &coupons.Coupon{ID: fmt.Sprintf("cpn-%d", n), Code: fmt.Sprintf("SAVE%d", n), PercentOff: 10, Status: coupons.CouponStatusActive})',
  invoke='return env.svc.Expire(ctx, fmt.Sprintf("cpn-%d", n))',
  subject='fmt.Sprintf("cpn-%d", n)', attrs="nil",
  bad='''_ = env.base.Store(ctx, &coupons.Coupon{ID: "cpn-used", Code: "USED", PercentOff: 5, MaxUses: 1, Uses: 1, Status: coupons.CouponStatusExhausted})
	_, e1 := env.svc.Expire(ctx, "cpn-used")
	_, e2 := env.svc.Expire(ctx, "cpn-missing")
	return []m{N}Bad{
		{"exhausted coupon", e1, coupons.ErrNotActive, true},
		{"unknown coupon", e2, coupons.ErrNotFound, false},
	}''',
  stored='c, err := env.base.Load(ctx, fmt.Sprintf("cpn-%d", n))\n\treturn err == nil && c.Status == coupons.CouponStatusExpired')

m(n=11, pkg="ledger", file="amend_audit", fn="Amend", op="ledger.amend", event="journal_entry.amended",
  iface="ledger.JournalEntryStore", field="JournalEntryStore", write="Put", vtype="*ledger.JournalEntry",
  base="ledger.NewMemoryJournalEntryStore()", basetype="*ledger.MemoryJournalEntryStore", extra_fields="", extra_init="",
  construct="ledger.NewService(repo, ledger.NewMemoryAccountStore(), rec, clk)", rtype="*ledger.JournalEntry",
  prepare='_ = env.base.Put(ctx, &ledger.JournalEntry{ID: fmt.Sprintf("je-%d", n), Account: fmt.Sprintf("%d000", n), Debit: 100, Status: ledger.JournalEntryStatusDraft})',
  invoke='return env.svc.Amend(ctx, fmt.Sprintf("je-%d", n), fmt.Sprintf("memo %d", n))',
  subject='fmt.Sprintf("je-%d", n)', attrs='map[string]string{"account": fmt.Sprintf("%d000", n)}',
  bad='''m{N}Prepare(env, ctx, 7)
	_ = env.base.Put(ctx, &ledger.JournalEntry{ID: "je-posted", Account: "1000", Debit: 100, Status: ledger.JournalEntryStatusPosted})
	_, e1 := env.svc.Amend(ctx, "je-7", "   ")
	_, e2 := env.svc.Amend(ctx, "je-posted", "late")
	_, e3 := env.svc.Amend(ctx, "je-missing", "memo")
	return []m{N}Bad{
		{"blank memo", e1, ledger.ErrEmptyMemo, true},
		{"posted entry", e2, ledger.ErrNotDraft, true},
		{"unknown entry", e3, ledger.ErrNotFound, false},
	}''',
  stored='e, err := env.base.Get(ctx, fmt.Sprintf("je-%d", n))\n\treturn err == nil && e.Memo == fmt.Sprintf("memo %d", n)')

m(n=12, pkg="catalog", file="reprice_audit", fn="Reprice", op="catalog.reprice", event="product_listing.repriced",
  iface="catalog.ProductListingStore", field="ProductListingStore", write="Put", vtype="*catalog.ProductListing",
  base="catalog.NewMemoryProductListingStore()", basetype="*catalog.MemoryProductListingStore", extra_fields="", extra_init="",
  construct="catalog.NewService(repo, rec, clk)", rtype="*catalog.ProductListing",
  prepare='_ = env.base.Put(ctx, &catalog.ProductListing{ID: fmt.Sprintf("pl-%d", n), SKU: fmt.Sprintf("SKU-%d", n), Title: "Mug", Price: 1000, Status: catalog.ProductListingStatusPublished})',
  invoke='return env.svc.Reprice(ctx, fmt.Sprintf("pl-%d", n), 1500)',
  subject='fmt.Sprintf("pl-%d", n)', attrs='map[string]string{"sku": fmt.Sprintf("SKU-%d", n)}',
  bad='''m{N}Prepare(env, ctx, 9)
	_ = env.base.Put(ctx, &catalog.ProductListing{ID: "pl-old", SKU: "SKU-old", Title: "Old", Price: 1000, Status: catalog.ProductListingStatusRetired})
	_, e1 := env.svc.Reprice(ctx, "pl-9", 0)
	_, e2 := env.svc.Reprice(ctx, "pl-old", 900)
	_, e3 := env.svc.Reprice(ctx, "pl-missing", 900)
	return []m{N}Bad{
		{"no price", e1, catalog.ErrNoPrice, true},
		{"retired listing", e2, catalog.ErrRetired, true},
		{"unknown listing", e3, catalog.ErrNotFound, false},
	}''',
  stored='l, err := env.base.Get(ctx, fmt.Sprintf("pl-%d", n))\n\treturn err == nil && l.Price == 1500')

TEMPLATE = '''// Code generated by tools/gen_hidden.py; DO NOT EDIT.

package {pkg}_test

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
	"shopcore/{pkg}"
)

var (
	_ = strconv.Itoa
	_ = ids.NewSequence
)

const (
	m{N}Op    = "{op}"
	m{N}Event = "{event}"
)

type m{N}Journal struct {{
	mu    sync.Mutex
	steps []string
}}

func (j *m{N}Journal) add(s string) {{
	j.mu.Lock()
	j.steps = append(j.steps, s)
	j.mu.Unlock()
}}

func (j *m{N}Journal) snapshot() []string {{
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.steps...)
}}

// m{N}Repo journals successful writes and can be told to fail them.
type m{N}Repo struct {{
	{iface}
	j       *m{N}Journal
	failErr error
}}

func (r *m{N}Repo) {write}(ctx context.Context, v {vtype}) error {{
	if r.failErr != nil {{
		r.j.add("write-failed")
		return r.failErr
	}}
	if err := r.{field}.{write}(ctx, v); err != nil {{
		return err
	}}
	r.j.add("write")
	return nil
}}

type m{N}Recorder struct {{
	mu      sync.Mutex
	j       *m{N}Journal
	events  []audit.Event
	failErr error
}}

func (r *m{N}Recorder) Record(ctx context.Context, ev audit.Event) error {{
	r.j.add("record")
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
	return r.failErr
}}

func (r *m{N}Recorder) all() []audit.Event {{
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]audit.Event(nil), r.events...)
}}

type m{N}Env struct {{
	svc  *{pkg}.Service
	base {basetype}
	repo *m{N}Repo
	rec  *m{N}Recorder
	j    *m{N}Journal
	{extra_fields}
}}

func m{N}Setup() *m{N}Env {{
	j := &m{N}Journal{{}}
	base := {base}
	repo := &m{N}Repo{{{field}: base, j: j}}
	rec := &m{N}Recorder{{j: j}}
	clk := clock.NewFixed(time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC))
	{extra_init}
	svc := {construct}
	return &m{N}Env{{svc: svc, base: base, repo: repo, rec: rec, j: j{extra_assign}}}
}}

// m{N}Ctx builds a request context like httpapi does, with ids unique to the
// test and call so they cannot be hard-coded. The trace id always differs.
// It also returns the request id, so tests compare events against this value
// and never read it back through ctxkeys.
func m{N}Ctx(t *testing.T, n int) (context.Context, string) {{
	h := fnv.New32a()
	h.Write([]byte(t.Name()))
	reqID := fmt.Sprintf("req-%08x-%d", h.Sum32(), n)
	ctx := ctxkeys.WithRequestID(context.Background(), reqID)
	ctx = ctxkeys.WithTraceID(ctx, fmt.Sprintf("trace-%08x-%d", h.Sum32()^0xabcdef, n))
	return ctxkeys.WithActor(ctx, "u-m{N}"), reqID
}}

// m{N}Prepare stores the fixtures call n needs directly in the base store, so
// they are not journaled.
func m{N}Prepare(env *m{N}Env, ctx context.Context, n int) {{
	{prepare}
}}

func m{N}Invoke(env *m{N}Env, ctx context.Context, n int) ({rtype}, error) {{
	{invoke}
}}

func m{N}Succeed(t *testing.T, env *m{N}Env, ctx context.Context, n int) {rtype} {{
	t.Helper()
	m{N}Prepare(env, ctx, n)
	res, err := m{N}Invoke(env, ctx, n)
	if err != nil {{
		t.Fatalf("{fn} #%d: %v", n, err)
	}}
	return res
}}

func m{N}Expect(res {rtype}, n int) (string, map[string]string) {{
	return {subject}, {attrs}
}}

// m{N}Stored reports whether the write of call n reached the store.
func m{N}Stored(env *m{N}Env, ctx context.Context, n int) bool {{
	{stored}
}}

type m{N}Bad struct {{
	label string
	err   error
	cause error // nil: only the op is checked
	exact bool  // false: op prefix and cause text only, other context allowed
}}

// m{N}Invalid runs calls that must fail before anything is written.
func m{N}Invalid(env *m{N}Env, ctx context.Context) []m{N}Bad {{
	{bad}
}}

// m{N}Chain lists err and every error it wraps, outermost first and depth
// first like errors.Is. It follows *errs.Error through its Err field and other
// errors through their Unwrap methods, so no method or helper of package errs
// takes part in the checks.
func m{N}Chain(err error) []error {{
	var chain []error
	var walk func(error)
	walk = func(err error) {{
		if err == nil || len(chain) >= 100 {{
			return
		}}
		chain = append(chain, err)
		switch e := err.(type) {{
		case *errs.Error:
			walk(e.Err)
		case interface{{ Unwrap() []error }}:
			for _, inner := range e.Unwrap() {{
				walk(inner)
			}}
		default:
			walk(errors.Unwrap(err))
		}}
	}}
	walk(err)
	return chain
}}

// m{N}Message renders err as "op: cause" from the *errs.Error fields.
func m{N}Message(err error) string {{
	if e, ok := err.(*errs.Error); ok {{
		if e.Err == nil {{
			return e.Op
		}}
		return e.Op + ": " + m{N}Message(e.Err)
	}}
	return err.Error()
}}

func m{N}CheckWrapped(t *testing.T, label string, err, cause error, exact bool) {{
	t.Helper()
	if err == nil {{
		t.Fatalf("%s: expected an error", label)
	}}
	chain := m{N}Chain(err)
	if cause != nil {{
		found := false
		for _, e := range chain {{
			found = found || reflect.TypeOf(cause).Comparable() && e == cause
		}}
		if !found {{
			t.Fatalf("%s: error %q does not wrap %q", label, err, cause)
		}}
	}}
	var ops []string
	for _, e := range chain {{
		if w, ok := e.(*errs.Error); ok {{
			ops = append(ops, w.Op)
		}}
	}}
	if len(ops) == 0 {{
		t.Fatalf("%s: error %q is not annotated with errs.Wrap", label, err)
	}}
	if ops[0] != m{N}Op {{
		t.Errorf("%s: op = %q, want %q", label, ops[0], m{N}Op)
	}}
	n := 0
	for _, op := range ops {{
		if op == m{N}Op {{
			n++
		}}
	}}
	if n > 1 {{
		t.Errorf("%s: %q wrapped %d times: %q", label, m{N}Op, n, err)
	}}
	if cause == nil {{
		return
	}}
	msg := m{N}Message(err)
	if exact {{
		if want := m{N}Op + ": " + cause.Error(); msg != want {{
			t.Errorf("%s: err.Error() = %q, want %q", label, msg, want)
		}}
	}} else if !strings.HasPrefix(msg, m{N}Op+": ") || !strings.Contains(msg, cause.Error()) {{
		t.Errorf("%s: err.Error() = %q, want %q + ...%q", label, msg, m{N}Op+": ", cause.Error())
	}}
}}

// Catches: wrong or inconsistent event names (including names that break the
// audit schema), missing or duplicate events.
func TestM{N}{fn}EventName(t *testing.T) {{
	env := m{N}Setup()
	for n := 1; n <= 2; n++ {{
		ctx, _ := m{N}Ctx(t, n)
		m{N}Succeed(t, env, ctx, n)
	}}
	evs := env.rec.all()
	if len(evs) != 2 {{
		t.Fatalf("want exactly one audit event per successful {fn}, got %d: %+v", len(evs), evs)
	}}
	for i, ev := range evs {{
		if ev.Name != m{N}Event {{
			t.Errorf("event %d name = %q, want %q", i, ev.Name, m{N}Event)
		}}
		if err := audit.Validate(ev); err != nil {{
			t.Errorf("event %d does not pass audit.Validate: %v", i, err)
		}}
	}}
}}

// Catches: request id not taken from ctxkeys.RequestID (missing, trace id,
// actor, constant), wrong subject.
func TestM{N}{fn}RequestIDAndSubject(t *testing.T) {{
	env := m{N}Setup()
	var reqIDs, subjects []string
	for n := 1; n <= 2; n++ {{
		ctx, reqID := m{N}Ctx(t, n)
		res := m{N}Succeed(t, env, ctx, n)
		s, _ := m{N}Expect(res, n)
		reqIDs = append(reqIDs, reqID)
		subjects = append(subjects, s)
	}}
	evs := env.rec.all()
	if len(evs) != 2 {{
		t.Fatalf("want 2 audit events, got %d: %+v", len(evs), evs)
	}}
	for i, want := range reqIDs {{
		if evs[i].RequestID != want {{
			t.Errorf("event %d RequestID = %q, want %q (the request id the context carries)", i, evs[i].RequestID, want)
		}}
		if evs[i].Subject != subjects[i] {{
			t.Errorf("event %d Subject = %q, want %q", i, evs[i].Subject, subjects[i])
		}}
	}}
}}

// Catches: attributes this package's events always carry missing or wrong.
func TestM{N}{fn}Attrs(t *testing.T) {{
	env := m{N}Setup()
	ctx, _ := m{N}Ctx(t, 1)
	res := m{N}Succeed(t, env, ctx, 3)
	_, want := m{N}Expect(res, 3)
	evs := env.rec.all()
	if len(evs) != 1 {{
		t.Fatalf("want 1 audit event, got %d: %+v", len(evs), evs)
	}}
	for k, v := range want {{
		if got, ok := evs[0].Attrs[k]; !ok || got != v {{
			t.Errorf("Attrs[%q] = %q (present %v), want %q; attrs = %v", k, got, ok, v, evs[0].Attrs)
		}}
	}}
}}

// Catches: recording before the repository write, recording asynchronously
// (after {fn} returns), or recording more than once.
func TestM{N}{fn}RecordsAfterWrite(t *testing.T) {{
	env := m{N}Setup()
	ctx, _ := m{N}Ctx(t, 1)
	m{N}Succeed(t, env, ctx, 1)
	steps := env.j.snapshot()
	if len(steps) != 2 || steps[0] != "write" || steps[1] != "record" {{
		t.Fatalf("calls during {fn} = %v, want [write record]: the event must be recorded after the write and before {fn} returns", steps)
	}}
}}

// Catches: audit events emitted on validation or repository failures.
func TestM{N}{fn}NoAuditOnFailure(t *testing.T) {{
	env := m{N}Setup()
	ctx, _ := m{N}Ctx(t, 1)
	for _, b := range m{N}Invalid(env, ctx) {{
		if b.err == nil {{
			t.Fatalf("%s: accepted", b.label)
		}}
	}}
	m{N}Prepare(env, ctx, 1)
	env.repo.failErr = errors.New("m{N}: repository unavailable")
	if _, err := m{N}Invoke(env, ctx, 1); err == nil {{
		t.Fatal("{fn} succeeded although the repository failed")
	}}
	if evs := env.rec.all(); len(evs) != 0 {{
		t.Fatalf("failed {fn} calls must not be audited, got %d events: %+v", len(evs), evs)
	}}
	env.repo.failErr = nil
	m{N}Succeed(t, env, ctx, 2)
	if evs := env.rec.all(); len(evs) != 1 {{
		t.Fatalf("successful {fn} after failures: want 1 audit event, got %d: %+v", len(evs), evs)
	}}
}}

// Catches: errors not wrapped with errs.Wrap("{op}", err), wrong op
// spelling, extra prefixes, double wrapping, swallowed audit errors. A
// repository failure may carry extra context below the op (as in
// errs.Wrap(op, fmt.Errorf("save: %w", err))); its cause is checked by identity.
func TestM{N}{fn}ErrorWrapping(t *testing.T) {{
	env := m{N}Setup()
	ctx, _ := m{N}Ctx(t, 1)
	for _, b := range m{N}Invalid(env, ctx) {{
		m{N}CheckWrapped(t, b.label, b.err, b.cause, b.exact)
	}}

	m{N}Prepare(env, ctx, 1)
	writeErr := errors.New("m{N}: write timeout")
	env.repo.failErr = writeErr
	_, err := m{N}Invoke(env, ctx, 1)
	m{N}CheckWrapped(t, "repository failure", err, writeErr, false)
	env.repo.failErr = nil

	recErr := errors.New("m{N}: audit sink unavailable")
	env.rec.failErr = recErr
	m{N}Prepare(env, ctx, 2)
	_, err = m{N}Invoke(env, ctx, 2)
	m{N}CheckWrapped(t, "audit failure", err, recErr, true)
	if !m{N}Stored(env, ctx, 2) {{
		t.Fatal("audit failure: the write must already have happened")
	}}
}}
'''

for c in M:
    N = str(c["n"])
    c = dict(c)
    for k in ("bad", "prepare", "invoke", "stored"):
        c[k] = c[k].replace("{N}", N)
    extra_assign = ""
    if c["extra_fields"]:
        nm = c["extra_fields"].split()[0]
        extra_assign = ", %s: %s" % (nm, nm)
    c["extra_assign"] = extra_assign
    c["N"] = N
    src = TEMPLATE.format(**c)
    d = os.path.join(OUT, c["pkg"])
    os.makedirs(d, exist_ok=True)
    for f in os.listdir(d):
        if f.endswith("_test.go"):
            os.remove(os.path.join(d, f))
    with open(os.path.join(d, "%s_m%s_test.go" % (c["file"], N)), "w") as fh:
        fh.write(src)

import subprocess
subprocess.run(["gofmt", "-w", OUT], check=True)
