package authjwt

import (
	"context"

	"github.com/google/uuid"
)

type claimsContextKey struct{}

var claimsKey claimsContextKey

// WithClaims кладёт Claims в context.
func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext достаёт Claims из context.
func ClaimsFromContext(ctx context.Context) (Claims, error) {
	claims, ok := ctx.Value(claimsKey).(Claims)
	if !ok {
		return Claims{}, ErrTokenInvalid
	}
	return claims, nil
}

// RequireRole проверяет, что роль пользователя входит в allowedRoles.
func RequireRole(ctx context.Context, allowedRoles ...string) error {
	claims, err := ClaimsFromContext(ctx)
	if err != nil {
		return err
	}

	for _, role := range allowedRoles {
		if claims.Role == role {
			return nil
		}
	}

	return ErrPermissionDenied
}

// RequireSelfOrRole разрешает доступ, если пользователь — сам targetUserID
// или его роль входит в allowedRoles.
func RequireSelfOrRole(ctx context.Context, targetUserID uuid.UUID, allowedRoles ...string) error {
	claims, err := ClaimsFromContext(ctx)
	if err != nil {
		return err
	}

	if claims.Subject == targetUserID {
		return nil
	}

	for _, role := range allowedRoles {
		if claims.Role == role {
			return nil
		}
	}

	return ErrPermissionDenied
}