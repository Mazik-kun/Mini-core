package domain

import "errors"

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrSameAccount       = errors.New("cannot transfer to the same account")
	ErrCurrencyMismatch  = errors.New("currency mismatch")
	ErrTransactionNotFound = errors.New("transaction not found")
)