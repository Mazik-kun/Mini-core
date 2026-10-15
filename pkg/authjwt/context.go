package authjwt

import (
	"context"

	"github.com/google/uuid"
)

type claimsContextKey struct{}

var claimsKey claimsContextKey

func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, error) {
	claims, ok := ctx.Value(claimsKey).(Claims)
	if !ok {
		return Claims{}, ErrTokenInvalid
	}
	return claims, nil
}

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
// или его роль входит в allowedRoles. Сравнение по Subject (user_id).
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

// RequireCustomerAccess разрешает доступ, если customer_id из claims совпадает
// с targetCustomerID, или роль пользователя входит в allowedRoles.
// Используется в customer-сервисе, где profile привязан к customer_id, а не к user_id.
func RequireCustomerAccess(ctx context.Context, targetCustomerID uuid.UUID, allowedRoles ...string) error {
	claims, err := ClaimsFromContext(ctx)
	if err != nil {
		return err
	}
	if claims.CustomerID != nil && *claims.CustomerID == targetCustomerID {
		return nil
	}
	for _, role := range allowedRoles {
		if claims.Role == role {
			return nil
		}
	}
	return ErrPermissionDenied
}