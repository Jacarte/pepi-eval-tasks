package httpapi

import (
	"errors"
	"net/http"

	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/inventory"
	"shopcore/loyalty"
	"shopcore/orders"
	"shopcore/payments"
	"shopcore/refunds"
	"shopcore/shipping"
	"shopcore/tax"
)

type errorBody struct {
	Error     string `json:"error"`
	Op        string `json:"op,omitempty"`
	RequestID string `json:"request_id"`
}

var notFound = []error{orders.ErrNotFound, payments.ErrNotFound, refunds.ErrNotFound, inventory.ErrNotFound, loyalty.ErrNotFound, shipping.ErrNotFound}

var unprocessable = []error{
	orders.ErrEmptyOrder, orders.ErrInvalidQuantity, orders.ErrMissingCustomer, orders.ErrTooManyLines, tax.ErrUnknownRegion,
	payments.ErrInvalidAmount, payments.ErrExceedsAuthorization,
	refunds.ErrInvalidAmount, refunds.ErrUnknownReason, refunds.ErrMissingPayment, refunds.ErrOverLimit,
	inventory.ErrInvalidQuantity, loyalty.ErrInvalidPoints,
}

var conflict = []error{
	orders.ErrNotCancellable, payments.ErrInvalidState, inventory.ErrInsufficientStock, loyalty.ErrInsufficientPoints,
	shipping.ErrBadTransition, refunds.ErrConflict,
}

func statusFor(err error) int {
	for _, group := range []struct {
		errs   []error
		status int
	}{{notFound, http.StatusNotFound}, {unprocessable, http.StatusUnprocessableEntity}, {conflict, http.StatusConflict}} {
		for _, target := range group.errs {
			if errors.Is(err, target) {
				return group.status
			}
		}
	}
	return http.StatusInternalServerError
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status := statusFor(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "internal error" // details are in the logs, keyed by request id
	}
	writeJSON(w, status, errorBody{Error: msg, Op: errs.Op(err), RequestID: ctxkeys.RequestIDFrom(r.Context())})
}
