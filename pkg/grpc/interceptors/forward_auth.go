package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ForwardAuthUnary - клиентский interceptor: берёт заголовок authorization из ВХОДЯЩЕГО
// запроса и передаёт его в ИСХОДЯЩИЙ вызов. Без него сервис, вызываемый из другого сервиса,
// отвечает Unauthenticated: "missing authorization header".
// Если входящего заголовка нет (например, обработка события из Kafka), вызов уходит как есть.
func ForwardAuthUnary() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get("authorization"); len(v) > 0 {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", v[0])
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}