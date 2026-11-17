package main

// Field kinds understood by the filler generator.
const (
	kString = "string"
	kInt    = "int"
	kInt64  = "int64"
	kCents  = "cents"
	kBool   = "bool"
	kTime   = "time"
	kTags   = "tags"
)

type Field struct {
	Name     string
	Kind     string
	Required bool // strings: non-blank; numbers: always checked non-negative unless Signed
	Signed   bool
	Doc      string
}

type Entity struct {
	Name     string
	Doc      string
	Fields   []Field
	Statuses []string
}

type Pkg struct {
	Name     string
	Doc      string
	Style    string // store method naming: crud | find | load
	Entities []Entity
}

func s(name string, req bool, doc string) Field {
	return Field{Name: name, Kind: kString, Required: req, Doc: doc}
}
func i(name, doc string) Field    { return Field{Name: name, Kind: kInt, Doc: doc} }
func is(name, doc string) Field   { return Field{Name: name, Kind: kInt, Signed: true, Doc: doc} }
func i64(name, doc string) Field  { return Field{Name: name, Kind: kInt64, Doc: doc} }
func c(name, doc string) Field    { return Field{Name: name, Kind: kCents, Doc: doc} }
func b(name, doc string) Field    { return Field{Name: name, Kind: kBool, Doc: doc} }
func t(name, doc string) Field    { return Field{Name: name, Kind: kTime, Doc: doc} }
func tags(name, doc string) Field { return Field{Name: name, Kind: kTags, Doc: doc} }

// pkgs lists every package that receives generated model/store/codec code.
// Handwritten files in static/ build on top of these names, so renaming a
// field or status here usually means touching static/ too.
var pkgs = []Pkg{
	{Name: "billing", Style: "crud", Entities: []Entity{
		{Name: "Invoice", Doc: "is a customer invoice issued through the v1 API.", Fields: []Field{
			s("CustomerID", true, "owning customer"), s("Number", true, "human readable invoice number"),
			c("AmountDue", ""), s("Currency", false, "ISO 4217; empty means USD"), t("DueAt", ""), i("Lines", "number of invoice lines"),
		}, Statuses: []string{"draft", "open", "paid", "void"}},
		{Name: "Charge", Doc: "is a processor charge attached to an invoice.", Fields: []Field{
			s("InvoiceID", true, ""), c("Amount", "authorized amount"), c("Captured", ""), s("Processor", false, "stripe, adyen or manual"),
		}, Statuses: []string{"pending", "authorized", "captured", "failed"}},
	}},
	{Name: "catalog", Style: "crud", Entities: []Entity{
		{Name: "ProductListing", Doc: "is the merchandised view of a SKU.", Fields: []Field{
			s("SKU", true, ""), s("Title", true, ""), s("Brand", false, ""), c("Price", "list price"), tags("Tags", "free-form merchandising tags"), b("Featured", ""),
		}, Statuses: []string{"draft", "published", "retired"}},
	}},
	{Name: "coupons", Style: "load", Entities: []Entity{
		{Name: "Coupon", Doc: "is a redeemable discount code.", Fields: []Field{
			s("Code", true, "case-insensitive code shown to shoppers"), i("PercentOff", ""), i("MaxUses", "0 means unlimited"), i("Uses", ""), t("ExpiresAt", ""),
		}, Statuses: []string{"draft", "active", "exhausted", "expired"}},
	}},
	{Name: "customers", Style: "crud", Entities: []Entity{
		{Name: "Customer", Doc: "is a registered shopper.", Fields: []Field{
			s("Email", true, ""), s("DisplayName", false, ""), s("Segment", false, "crm segment code"), c("LifetimeValue", ""), b("Marketing", "opted into marketing email"),
		}, Statuses: []string{"pending", "active", "suspended", "closed"}},
		{Name: "Address", Doc: "is a postal address on file for a customer.", Fields: []Field{
			s("CustomerID", true, ""), s("Line1", true, ""), s("City", false, ""), s("Postcode", false, ""), s("Country", true, "ISO 3166 alpha-2"), b("Primary", ""),
		}, Statuses: []string{"active", "archived"}},
	}},
	{Name: "fraud", Style: "find", Entities: []Entity{
		{Name: "Review", Doc: "is a manual fraud review opened for an order.", Fields: []Field{
			s("OrderID", true, ""), i("Score", "0-1000, higher is riskier"), s("Reason", false, ""), s("Analyst", false, "assigned analyst"),
		}, Statuses: []string{"open", "escalated", "cleared", "confirmed"}},
		{Name: "Rule", Doc: "is a weighted scoring rule.", Fields: []Field{
			s("Name", true, ""), i("Weight", ""), b("Enabled", ""), s("Expression", false, "legacy rule DSL, see scorer.go"),
		}, Statuses: []string{"draft", "live", "retired"}},
	}},
	{Name: "inventory", Style: "crud", Entities: []Entity{
		{Name: "Adjustment", Doc: "is a manual stock correction.", Fields: []Field{
			s("SKU", true, ""), is("Delta", "signed change in units"), s("Reason", false, ""), s("Operator", false, ""),
		}, Statuses: []string{"pending", "applied", "rejected"}},
	}},
	{Name: "ledger", Style: "crud", Entities: []Entity{
		{Name: "JournalEntry", Doc: "is a double-entry journal line pair.", Fields: []Field{
			s("Account", true, "account code"), c("Debit", ""), c("Credit", ""), s("Memo", false, ""), s("Reference", false, "external reference, e.g. an order id"),
		}, Statuses: []string{"draft", "posted", "reversed"}},
		{Name: "Account", Doc: "is a general ledger account.", Fields: []Field{
			s("Code", true, ""), s("Name", false, ""), s("Kind", false, "asset, liability, revenue or expense"), c("Balance", ""),
		}, Statuses: []string{"open", "frozen", "closed"}},
	}},
	{Name: "loyalty", Style: "find", Entities: []Entity{
		{Name: "Tier", Doc: "is a loyalty tier definition.", Fields: []Field{
			s("Name", true, ""), i64("Threshold", "lifetime points needed"), i("Multiplier", "accrual multiplier in percent"),
		}, Statuses: []string{"active", "retired"}},
	}},
	{Name: "notifications", Style: "load", Entities: []Entity{
		{Name: "Message", Doc: "is an outbound customer notification.", Fields: []Field{
			s("Recipient", true, ""), s("Template", true, ""), s("Channel", false, "email, sms or push"), i("Attempts", ""),
		}, Statuses: []string{"queued", "sent", "failed", "suppressed"}},
	}},
	{Name: "orders", Style: "find", Entities: []Entity{
		{Name: "Draft", Doc: "is an unsubmitted cart snapshot.", Fields: []Field{
			s("CustomerID", true, ""), i("Items", ""), s("Note", false, ""), c("Total", ""),
		}, Statuses: []string{"open", "abandoned", "converted"}},
	}},
	{Name: "payments", Style: "load", Entities: []Entity{
		{Name: "Settlement", Doc: "is a processor settlement batch.", Fields: []Field{
			s("Batch", true, ""), c("Amount", ""), s("Processor", false, ""), i("Items", ""),
		}, Statuses: []string{"open", "submitted", "settled"}},
	}},
	{Name: "pricing", Style: "crud", Entities: []Entity{
		{Name: "PriceRule", Doc: "is a price adjustment rule.", Fields: []Field{
			s("SKU", true, "SKU or * for all"), s("Kind", true, "percent or fixed"), i64("Value", "basis points for percent, cents for fixed"), i("Priority", "lower runs first"), b("Stackable", ""),
		}, Statuses: []string{"draft", "active", "expired"}},
	}},
	{Name: "refunds", Style: "crud", Entities: []Entity{
		{Name: "RefundPolicy", Doc: "describes the refund window for a product family.", Fields: []Field{
			s("Name", true, ""), i("WindowDays", ""), i("RestockFeePct", ""), b("RequiresReceipt", ""),
		}, Statuses: []string{"active", "archived"}},
	}},
	{Name: "reports", Style: "find", Entities: []Entity{
		{Name: "ReportRun", Doc: "is one execution of a scheduled report.", Fields: []Field{
			s("Kind", true, ""), s("RequestedBy", false, ""), i("Rows", ""), s("Format", false, "csv or json"),
		}, Statuses: []string{"queued", "running", "done", "failed"}},
	}},
	{Name: "returns", Style: "find", Entities: []Entity{
		{Name: "ReturnRequest", Doc: "is a customer request to send items back.", Fields: []Field{
			s("OrderID", true, ""), s("SKU", true, ""), i("Qty", ""), s("Reason", false, ""), s("Inspector", false, "warehouse inspector"),
		}, Statuses: []string{"requested", "authorized", "received", "rejected"}},
	}},
	{Name: "scheduling", Style: "load", Entities: []Entity{
		{Name: "Job", Doc: "is a recurring background job definition.", Fields: []Field{
			s("Name", true, ""), s("Spec", true, "schedule spec, see ParseSpec"), s("Owner", false, ""), t("LastRunAt", ""), i("Failures", "consecutive failures"),
		}, Statuses: []string{"idle", "running", "paused", "disabled"}},
		{Name: "Slot", Doc: "is a delivery booking window.", Fields: []Field{
			s("Warehouse", true, ""), s("Window", true, "e.g. 09:00-12:00"), i("Capacity", ""), i("Booked", ""),
		}, Statuses: []string{"open", "full", "closed"}},
	}},
	{Name: "searchindex", Style: "crud", Entities: []Entity{
		{Name: "Document", Doc: "is a unit of indexed content.", Fields: []Field{
			s("Kind", true, "product, category or page"), s("Title", true, ""), s("Body", false, ""), i("Boost", ""), tags("Tags", ""),
		}, Statuses: []string{"pending", "indexed", "stale"}},
	}},
	{Name: "shipping", Style: "crud", Entities: []Entity{
		{Name: "Shipment", Doc: "is a parcel handed to a carrier.", Fields: []Field{
			s("OrderID", true, ""), s("Carrier", true, ""), s("TrackingNo", false, ""), i("WeightGrams", ""), s("Service", false, "carrier service level"), s("HoldReason", false, ""),
		}, Statuses: []string{"created", "dispatched", "held", "delivered"}},
	}},
	{Name: "suppliers", Style: "find", Entities: []Entity{
		{Name: "Supplier", Doc: "is a vendor we buy stock from.", Fields: []Field{
			s("Name", true, ""), s("Country", true, ""), s("ContactEmail", false, ""), i("LeadTimeDays", ""), b("Preferred", ""),
		}, Statuses: []string{"prospect", "onboarded", "suspended"}},
		{Name: "PurchaseOrder", Doc: "is an order placed with a supplier.", Fields: []Field{
			s("SupplierID", true, ""), c("Total", ""), i("Lines", ""), t("ExpectedAt", ""),
		}, Statuses: []string{"draft", "approved", "received", "cancelled"}},
	}},
	{Name: "tax", Style: "load", Entities: []Entity{
		{Name: "Exemption", Doc: "is a tax exemption certificate.", Fields: []Field{
			s("CustomerID", true, ""), s("Certificate", true, ""), s("Region", false, ""), t("ExpiresAt", ""),
		}, Statuses: []string{"pending", "valid", "expired"}},
	}},
	{Name: "warehouse", Style: "load", Entities: []Entity{
		{Name: "BinSlot", Doc: "is a physical storage slot.", Fields: []Field{
			s("Zone", true, ""), i("Aisle", ""), i("Level", ""), s("SKU", false, "allocated SKU, if any"), i("Capacity", "units"),
		}, Statuses: []string{"free", "allocated", "blocked"}},
		{Name: "Transfer", Doc: "moves stock between two bins.", Fields: []Field{
			s("FromBin", true, ""), s("ToBin", true, ""), s("SKU", true, ""), i("Qty", ""),
		}, Statuses: []string{"planned", "moving", "done"}},
	}},
}
