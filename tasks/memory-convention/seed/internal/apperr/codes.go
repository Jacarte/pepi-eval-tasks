// Package apperr holds the error codes sent to API clients.
package apperr

// Code is the machine-readable identifier of a failure reason.
type Code string

const (
	ErrInternal             Code = "ERR-PAY-0001"
	ErrMalformedBody        Code = "ERR-PAY-0002"
	ErrMissingField         Code = "ERR-PAY-0003"
	ErrPaymentNotFound      Code = "ERR-PAY-0004"
	ErrCustomerNotFound     Code = "ERR-PAY-0005"
	ErrPaymentNotAuthorized Code = "ERR-PAY-0006"
	ErrUnsupportedCurrency  Code = "ERR-PAY-0007"
	ErrAmountNotPositive    Code = "ERR-PAY-0008"
	ErrCustomerEmailTaken   Code = "ERR-PAY-0009"
)
