package grpcserver

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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

func RequesterInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req any,
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (any, error) {
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return handler(ctx, req)
        }

        var id uuid.UUID
        if v := md.Get("x-requester-id"); len(v) > 0 {
            id, _ = uuid.Parse(v[0])
        }
        var role string
        if v := md.Get("x-requester-role"); len(v) > 0 {
            role = v[0]
        }

        ctx = context.WithValue(ctx, ctxKeyCustomerID, id)
        ctx = context.WithValue(ctx, ctxKeyRole, role)
        return handler(ctx, req)
    }
}