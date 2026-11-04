package domain

import "errors"

var (
	// ErrAccountNotFound — счет не найден в базе данных.
	ErrAccountNotFound = errors.New("account not found")

	// ErrLimitExceeded — превышен лимит счетов (например, больше 3 в одной валюте).
	ErrLimitExceeded = errors.New("limit of accounts per currency exceeded")

	// ErrCustomerNotActive — клиент не прошел KYC или заблокирован.
	ErrCustomerNotActive = errors.New("customer is not active")

	// ErrInvalidStatus — попытка сделать недопустимый переход статуса
	// (например, закрыть уже закрытый счет).
	ErrInvalidStatus = errors.New("invalid status transition")

	// ErrBalanceNotZero — попытка закрыть счет, на котором остались деньги.
	ErrBalanceNotZero = errors.New("account balance is not zero")
)