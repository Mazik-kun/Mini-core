package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

// requireAccess: CLIENT — только свой customer_id, OFFICER/ADMIN — любой.
func requireAccess(ctx context.Context, customerID uuid.UUID) error {
	if err := authjwt.RequireCustomerAccess(ctx, customerID, roles.RoleOfficer, roles.RoleAdmin); err != nil {
		return domain.ErrAccessDenied
	}
	return nil
}
