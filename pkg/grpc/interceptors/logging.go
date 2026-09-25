package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Logging(log *slog.Logger) grpc.UnaryServerInterceptor{
	return func (
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	)(resp any, err error){
		start :=time.Now()

		resp, err = handler(ctx, req)
		
		duration := time.Since(start)
		code := status.Code(err)
	
		logFunc := log.Info
		switch code {
		case codes.Internal, codes.Unknown:
			logFunc = log.Error
		case codes.InvalidArgument, codes.NotFound:
			logFunc = log.Warn
		}

		attributes := []any{
			"method", info.FullMethod,
			"duration", duration,
			"code", code.String(),
		}

		if err != nil{
			attributes = append(attributes, "err", err)
		}
		logFunc("rpc", attributes...)
		return resp, err
	}
}