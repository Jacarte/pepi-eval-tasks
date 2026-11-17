package main

import (
	"fmt"
	"strings"
)

// tables emits the large data files every legacy monolith accumulates.
func tables() map[string]string {
	r := rngFor("tables")
	out := map[string]string{}

	// ---- tax jurisdictions ----
	w := &writer{}
	w.p("package tax")
	w.p("")
	w.p("// regionTable is synced by hand from the finance spreadsheet (tab \"rates\").")
	w.p("// Rates are basis points of the taxable amount. Keep codes upper-case.")
	w.p("var regionTable = []regionRate{")
	fixed := []struct {
		code, name string
		bps        int
	}{
		{"US-CA", "California", 725}, {"US-NY", "New York", 400}, {"US-TX", "Texas", 625},
		{"US-WA", "Washington", 650}, {"US-OR", "Oregon", 0}, {"DE", "Germany", 1900},
		{"FR", "France", 2000}, {"GB", "United Kingdom", 2000}, {"NL", "Netherlands", 2100},
		{"ES", "Spain", 2100}, {"IT", "Italy", 2200}, {"SE", "Sweden", 2500},
		{"CA-ON", "Ontario", 1300}, {"CA-QC", "Quebec", 1498}, {"AU", "Australia", 1000}, {"JP", "Japan", 1000},
	}
	for _, f := range fixed {
		w.p("\t{Code: %q, Name: %q, BPS: %d, Since: %q},", f.code, f.name, f.bps, fmt.Sprintf("20%02d-01-01", 14+r.Intn(8)))
	}
	states := []string{"AL Alabama", "AK Alaska", "AZ Arizona", "AR Arkansas", "CO Colorado", "CT Connecticut",
		"DE Delaware", "FL Florida", "GA Georgia", "HI Hawaii", "ID Idaho", "IL Illinois", "IN Indiana", "IA Iowa",
		"KS Kansas", "KY Kentucky", "LA Louisiana", "ME Maine", "MD Maryland", "MA Massachusetts", "MI Michigan",
		"MN Minnesota", "MS Mississippi", "MO Missouri", "MT Montana", "NE Nebraska", "NV Nevada", "NH New Hampshire",
		"NJ New Jersey", "NM New Mexico", "NC North Carolina", "ND North Dakota", "OH Ohio", "OK Oklahoma",
		"PA Pennsylvania", "RI Rhode Island", "SC South Carolina", "SD South Dakota", "TN Tennessee", "UT Utah",
		"VT Vermont", "VA Virginia", "WV West Virginia", "WI Wisconsin", "WY Wyoming"}
	for _, st := range states {
		code, name, _ := strings.Cut(st, " ")
		bps := 0
		if r.Intn(8) != 0 {
			bps = 250 + r.Intn(26)*25
		}
		note := ""
		if r.Intn(9) == 0 {
			note = fmt.Sprintf(", Notes: %q", []string{"local surcharges not modelled", "clothing exempt under $110", "check with finance before changing", "rate changed mid-year"}[r.Intn(4)])
		}
		w.p("\t{Code: %q, Name: %q, BPS: %d, Since: %q%s},", "US-"+code, name, bps, fmt.Sprintf("20%02d-%02d-01", 12+r.Intn(10), 1+r.Intn(12)), note)
	}
	eu := []string{"AT Austria 2000", "BE Belgium 2100", "BG Bulgaria 2000", "HR Croatia 2500", "CY Cyprus 1900",
		"CZ Czechia 2100", "DK Denmark 2500", "EE Estonia 2200", "FI Finland 2400", "GR Greece 2400", "HU Hungary 2700",
		"IE Ireland 2300", "LV Latvia 2100", "LT Lithuania 2100", "LU Luxembourg 1700", "MT Malta 1800", "PL Poland 2300",
		"PT Portugal 2300", "RO Romania 1900", "SK Slovakia 2000", "SI Slovenia 2200", "NO Norway 2500", "CH Switzerland 810"}
	for _, c := range eu {
		f := strings.Fields(c)
		var bps int
		fmt.Sscan(f[2], &bps)
		w.p("\t{Code: %q, Name: %q, BPS: %d, Since: %q},", f[0], f[1], bps, fmt.Sprintf("20%02d-01-01", 15+r.Intn(7)))
	}
	for _, c := range []string{"AB Alberta 500", "BC British_Columbia 1200", "MB Manitoba 1200", "NB New_Brunswick 1500", "NS Nova_Scotia 1500", "SK Saskatchewan 1100"} {
		f := strings.Fields(c)
		var bps int
		fmt.Sscan(f[2], &bps)
		w.p("\t{Code: %q, Name: %q, BPS: %d, Since: \"2019-04-01\"},", "CA-"+f[0], strings.ReplaceAll(f[1], "_", " "), bps)
	}
	w.p("}")
	out["tax/rates_table.go"] = w.String()

	// ---- config defaults ----
	w = &writer{}
	w.p("package config")
	w.p("")
	w.p("// defaults is the full list of recognised settings. Anything not listed")
	w.p("// here is rejected by Load, which has caught many typos in deploy files.")
	w.p("var defaults = []setting{")
	areas := []string{"audit", "catalog", "coupons", "customers", "fraud", "httpapi", "inventory", "ledger", "loyalty",
		"notifications", "orders", "payments", "pricing", "refunds", "reports", "returns", "scheduling", "searchindex",
		"shipping", "suppliers", "tax", "warehouse"}
	knobs := []struct{ name, kind, doc string }{
		{"enabled", "bool", "feature switch"}, {"max_batch", "int", "upper bound for batch jobs"},
		{"timeout_ms", "int", "per call timeout"}, {"retry_attempts", "int", "retries before giving up"},
		{"page_size", "int", "default page size for list endpoints"}, {"cache_ttl_s", "int", "in-process cache ttl"},
		{"owner", "string", "team that gets paged"}, {"export_bucket", "string", "nightly export destination"},
		{"strict_mode", "bool", "reject instead of warn"}, {"region", "string", "primary region"},
	}
	teams := []string{"checkout", "fulfilment", "growth", "finance-eng", "platform", "catalog-team"}
	for _, a := range areas {
		for _, k := range r.Perm(len(knobs))[:3+r.Intn(3)] {
			kn := knobs[k]
			var val string
			switch kn.kind {
			case "bool":
				val = []string{"true", "false"}[r.Intn(2)]
			case "int":
				val = fmt.Sprint([]int{5, 10, 25, 50, 100, 200, 250, 500, 1000, 3000}[r.Intn(10)])
			default:
				switch kn.name {
				case "owner":
					val = teams[r.Intn(len(teams))]
				case "region":
					val = []string{"us-east-1", "eu-west-1"}[r.Intn(2)]
				default:
					val = "exports-" + a
				}
			}
			w.p("\t{Key: %q, Kind: %q, Value: %q, Doc: %q},", a+"."+kn.name, kn.kind, val, kn.doc)
		}
	}
	w.p("}")
	out["config/defaults_table.go"] = w.String()

	// ---- notification templates ----
	w = &writer{}
	w.p("package notifications")
	w.p("")
	w.p("// Template ids are part of the contract with the email vendor; do not")
	w.p("// rename them without migrating the vendor side first.")
	w.p("const (")
	ids := []string{"order_placed", "order_shipped", "order_delivered", "order_cancelled", "payment_failed",
		"payment_captured", "refund_issued", "refund_rejected", "return_authorized", "return_received",
		"loyalty_points_earned", "loyalty_points_redeemed", "loyalty_tier_changed", "coupon_expiring", "cart_abandoned",
		"password_reset", "account_suspended", "back_in_stock", "price_drop", "review_request", "supplier_po_approved",
		"invoice_ready", "invoice_overdue", "delivery_slot_booked", "delivery_slot_changed"}
	for _, id := range ids {
		w.p("\tTemplate%s = %q", pascal(id), id)
	}
	w.p(")")
	w.p("")
	w.p("var templates = []tmpl{")
	greet := []string{"Hi {{name}},", "Hello {{name}},", "Dear {{name}},", "{{name}},"}
	sign := []string{"The Shop team", "Customer care", "Thanks for shopping with us"}
	for _, id := range ids {
		ch := []string{"email", "email", "sms", "push"}[r.Intn(4)]
		subj := strings.ToUpper(id[:1]) + strings.ReplaceAll(id[1:], "_", " ")
		body := fmt.Sprintf("%s\\n\\nYour %s update for {{ref}} is ready.\\n\\n%s", greet[r.Intn(len(greet))], strings.ReplaceAll(id, "_", " "), sign[r.Intn(len(sign))])
		w.p("\t{ID: Template%s, Channel: %q, Subject: %q, Body: \"%s\"},", pascal(id), ch, subj, body)
	}
	w.p("}")
	out["notifications/templates_table.go"] = w.String()

	// ---- shipping zone rates ----
	w = &writer{}
	w.p("package shipping")
	w.p("")
	w.p("// zoneRates is the carrier rate card. MaxGrams is inclusive; rows are")
	w.p("// matched in order so keep each carrier/zone sorted by weight.")
	w.p("var zoneRates = []zoneRate{")
	for _, carrier := range []string{"ups", "fedex", "dhl", "usps"} {
		for zone := 1; zone <= 4; zone++ {
			base := 400 + r.Intn(300) + zone*120
			for _, g := range []int{500, 2000, 5000, 10000, 30000} {
				w.p("\t{Carrier: %q, Zone: %d, MaxGrams: %d, Cents: %d},", carrier, zone, g, base)
				base += 150 + r.Intn(400)
			}
		}
	}
	w.p("}")
	out["shipping/zones_table.go"] = w.String()

	// ---- search stopwords ----
	w = &writer{}
	w.p("package searchindex")
	w.p("")
	w.p("// stopwords are dropped by the tokenizer. English only for now.")
	w.p("var stopwords = map[string]bool{")
	sw := strings.Fields("a an and are as at be but by for from has have he in is it its of on or that the this to was were will with you your our we they them their than then there these those not no yes")
	for k := 0; k < len(sw); k += 6 {
		end := k + 6
		if end > len(sw) {
			end = len(sw)
		}
		parts := []string{}
		for _, s := range sw[k:end] {
			parts = append(parts, fmt.Sprintf("%q: true", s))
		}
		w.p("\t%s,", strings.Join(parts, ", "))
	}
	w.p("}")
	out["searchindex/stopwords.go"] = w.String()

	// ---- fraud country risk ----
	w = &writer{}
	w.p("package fraud")
	w.p("")
	w.p("// countryRisk is the base score contribution per billing country, from the")
	w.p("// 2022 chargeback analysis. Unlisted countries score defaultCountryRisk.")
	w.p("var countryRisk = map[string]int{")
	for _, c := range strings.Fields("US CA GB DE FR NL ES IT SE NO DK FI IE AT BE CH PL PT CZ GR HU RO BG HR SK SI LT LV EE AU NZ JP KR SG HK TW BR MX AR CL CO IN ID MY PH TH VN ZA NG EG TR") {
		w.p("\t%q: %d,", c, 5+r.Intn(30)*5)
	}
	w.p("}")
	out["fraud/country_risk.go"] = w.String()
	return out
}
