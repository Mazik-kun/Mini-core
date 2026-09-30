package hasher

import "github.com/Mazik-kun/mini-core/services/auth/internal/domain/password"

type Argon2 struct{}

func NewArgon2() *Argon2 { return &Argon2{} }

func (Argon2) Hash(plain string) (string, error) {
	return password.HashPassword(plain)
}

func (Argon2) Verify(plain, hash string) (bool, error) {
	return password.VerifyPassword(plain, hash)
}
