package app

import (
	"strings"
	"time"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxNameLen    = 200
	maxAddressLen = 500
	minAge        = 0
	maxAge        = 120
)

func validateFullName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxNameLen {
		return domain.ErrInvalidFullName
	}
	return nil
}

func validateBirthDate(d pgtype.Date) error {
	if !d.Valid {
		return domain.ErrInvalidBirthDate
	}
	now := time.Now()
	if d.Time.After(now) {
		return domain.ErrInvalidBirthDate
	}
	age := now.Year() - d.Time.Year()
	if age < minAge || age > maxAge {
		return domain.ErrInvalidBirthDate
	}
	return nil
}

func validateAddress(addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" || len(addr) > maxAddressLen {
		return domain.ErrInvalidAddress
	}
	return nil
}

func validatePhoneNumber(phone string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return domain.ErrInvalidPhoneNumber
	}
	// допустимы цифры, +, -, пробелы, скобки
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '+', '-', ' ', '(', ')':
			continue
		default:
			return domain.ErrInvalidPhoneNumber
		}
	}
	digits := 0
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	if digits < 7 || digits > 15 {
		return domain.ErrInvalidPhoneNumber
	}
	return nil
}

func validateCitizenship(c string) error {
	c = strings.TrimSpace(c)
	if c == "" || len(c) > 100 {
		return domain.ErrInvalidCitizenship
	}
	return nil
}