package domain

import "errors"

var (
	ErrStockNotFound    = errors.New("stock not found in bank")
	ErrBankOutOfStock   = errors.New("bank has no stock available")
	ErrWalletOutOfStock = errors.New("wallet has no stock to sell")
	ErrInvalidTradeType = errors.New("invalid trade type: must be buy or sell")
)

// ValidationError signals a client-provided value that failed validation.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// NewValidationError creates a ValidationError.
func NewValidationError(msg string) *ValidationError {
	return &ValidationError{Message: msg}
}
