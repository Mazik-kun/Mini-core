package authjwt

import (
	"crypto/ed25519"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// KeyID достаёт kid из заголовка JWT БЕЗ проверки подписи. Нужен только чтобы выбрать
// публичный ключ из JWKS; доверять токену можно лишь после VerifyWithKey.
func KeyID(tokenString string) (string, error) {
	parser := jwtlib.NewParser(jwtlib.WithValidMethods([]string{jwtlib.SigningMethodEdDSA.Alg()}))
	token, _, err := parser.ParseUnverified(tokenString, jwtlib.MapClaims{})
	if err != nil {
		return "", ErrTokenInvalid
	}
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return "", ErrTokenInvalid
	}
	return kid, nil
}

// VerifyWithKey проверяет токен заданным публичным ключом (подпись EdDSA, exp, структура claims).
// Источник ключа (JWKS-кэш, kid) определяет вызывающий код.
func VerifyWithKey(tokenString string, publicKey ed25519.PublicKey) (Claims, error) {
	return NewVerifier(publicKey).VerifyToken(tokenString)
}
