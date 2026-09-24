package interceptors

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Recovery(log *slog.Logger) grpc.UnaryServerInterceptor{
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error){
		defer func(){
		if r:=recover(); r != nil{
			log.Error("panic recovered",
			"method", info.FullMethod,
			"panic", r,
			"stack", string(debug.Stack()),		
		)
		err = status.Error(codes.Internal,"internal error")
		}}()
		resp, err = handler(ctx, req)
		return 
	}
}