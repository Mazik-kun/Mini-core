package app

import (
	"strings"

	"github.com/Mazik-kun/mini-core/services/auth/internal/domain"
)

func validateEmail(email string) error {
	if email == "" || len(email) > 255 {
		return domain.ErrInvalidEmail
	}
	at := strings.Index(email, "@")
	if at < 1 || at == len(email)-1 {
		return domain.ErrInvalidEmail
	}
	if !strings.Contains(email[at+1:], ".") {
		return domain.ErrInvalidEmail
	}
	return nil
}

func validatePassword(pass string) error {
	if len(pass) < 8 || len(pass) > 72 {
		return domain.ErrWeakPassword
	}
	return nil
}