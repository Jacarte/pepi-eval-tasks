package httpapi

import (
	"net/http"

	"shopcore/money"
	"shopcore/orders"
	"shopcore/refunds"
)

type lineJSON struct {
	SKU       string      `json:"sku"`
	Qty       int         `json:"qty"`
	UnitPrice money.Cents `json:"unit_price"`
}

type placeOrderJSON struct {
	CustomerID string     `json:"customer_id"`
	Region     string     `json:"region"`
	Lines      []lineJSON `json:"lines"`
	CouponCode string     `json:"coupon_code"`
	PercentOff int        `json:"percent_off"`
}

type orderJSON struct {
	ID       string      `json:"id"`
	Status   string      `json:"status"`
	Subtotal money.Cents `json:"subtotal"`
	Discount money.Cents `json:"discount"`
	Tax      money.Cents `json:"tax"`
	Total    money.Cents `json:"total"`
}

func toOrderJSON(o *orders.Order) orderJSON {
	return orderJSON{ID: o.ID, Status: string(o.Status), Subtotal: o.Subtotal, Discount: o.Discount, Tax: o.Tax, Total: o.Total}
}

func (s *Server) placeOrder(w http.ResponseWriter, r *http.Request) {
	var in placeOrderJSON
	if err := decode(r, &in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req := orders.PlaceRequest{CustomerID: in.CustomerID, Region: in.Region, CouponCode: in.CouponCode, PercentOff: in.PercentOff}
	for _, l := range in.Lines {
		req.Lines = append(req.Lines, orders.Line{SKU: l.SKU, Qty: l.Qty, UnitPrice: l.UnitPrice})
	}
	o, err := s.svc.Orders.Place(r.Context(), req)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toOrderJSON(o))
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	o, err := s.svc.Orders.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toOrderJSON(o))
}

func (s *Server) capturePayment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Amount money.Cents `json:"amount"`
	}
	if err := decode(r, &in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	p, err := s.svc.Payments.Capture(r.Context(), r.PathValue("id"), in.Amount)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": p.ID, "status": p.Status, "captured": p.Captured})
}

func (s *Server) issueRefund(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PaymentID string      `json:"payment_id"`
		OrderID   string      `json:"order_id"`
		Amount    money.Cents `json:"amount"`
		Reason    string      `json:"reason"`
		Note      string      `json:"note"`
	}
	if err := decode(r, &in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	rf, err := s.svc.Refunds.Issue(r.Context(), refunds.IssueRequest{PaymentID: in.PaymentID, OrderID: in.OrderID, Amount: in.Amount, Reason: refunds.Reason(in.Reason), Note: in.Note})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": rf.ID, "status": rf.Status, "amount": rf.Amount})
}

func (s *Server) reserveStock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Qty int `json:"qty"`
	}
	if err := decode(r, &in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	it, err := s.svc.Inventory.Reserve(r.Context(), r.PathValue("sku"), in.Qty)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sku": it.ID, "on_hand": it.OnHand, "reserved": it.Reserved, "available": it.Available()})
}

func (s *Server) redeemPoints(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Points int64 `json:"points"`
	}
	if err := decode(r, &in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	a, err := s.svc.Loyalty.Redeem(r.Context(), r.PathValue("id"), in.Points)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": a.ID, "balance": a.Balance})
}

func (s *Server) dispatchShipment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TrackingNo string `json:"tracking_no"`
	}
	if err := decode(r, &in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	sh, err := s.svc.Shipping.Dispatch(r.Context(), r.PathValue("id"), in.TrackingNo)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": sh.ID, "status": sh.Status.String(), "tracking_no": sh.TrackingNo})
}
