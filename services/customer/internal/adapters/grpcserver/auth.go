package grpcserver

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	ctxKeyUserID     contextKey = "auth.user_id"
	ctxKeyCustomerID contextKey = "auth.customer_id"
	ctxKeyRole       contextKey = "auth.role"
)

// RequesterFromContext достаёт идентификаторы и роль, положенные интерсептором auth.
// Если что-то не заполнено — вернётся нулевое значение, проверка прав произойдёт в use case.
func RequesterFromContext(ctx context.Context) (customerID uuid.UUID, role string) {
	customerID, _ = ctx.Value(ctxKeyCustomerID).(uuid.UUID)
	role, _ = ctx.Value(ctxKeyRole).(string)
	return customerID, role
}

// SetRequester используется интерсептором, чтобы положить claims в контекст.
func SetRequester(ctx context.Context, userID, customerID uuid.UUID, role string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	ctx = context.WithValue(ctx, ctxKeyCustomerID, customerID)
	ctx = context.WithValue(ctx, ctxKeyRole, role)
	return ctx
}