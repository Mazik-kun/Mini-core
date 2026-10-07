package domain

import "errors"

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrAccessDenied = errors.New("access denied")
	ErrProfileLocked = errors.New("cannot change because of status")
	ErrAlreadyExists = errors.New("profile already exists")
	ErrInvalidCustomerID = errors.New("invalid customer id")
	ErrInvalidFullName = errors.New("invalid FullName")
	ErrInvalidBirthDate = errors.New("invalid birth date")
	ErrInvalidAddress = errors.New("invalid address")
	ErrInvalidStatus = errors.New("invalid status")
	ErrInvalidPhoneNumber = errors.New("invalid phone number")
	ErrInvalidCitizenship = errors.New("invalid citizenship")
	ErrInvalidPageToken = errors.New("invalid page token")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)