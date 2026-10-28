package customergrpc

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
)

type Client struct {
	cli customerv1.CustomerServiceClient
}

func forwardAuth(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if authHeader, exists := md["authorization"]; exists && len(authHeader) > 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader[0])
		}
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(forwardAuth), 
	)
	if err != nil {
		return nil, err
	}
	return &Client{cli: customerv1.NewCustomerServiceClient(conn)}, nil
}

func (c *Client) GetCustomerProfile(ctx context.Context, customerID uuid.UUID) (string, time.Time, string, error) {
	resp, err := c.cli.GetCustomer(ctx, &customerv1.GetCustomerRequest{
		CustomerId: customerID.String(),
	})
	if err != nil {
		return "", time.Time{}, "", err
	}

	birthDate := time.Time{}
	if resp.GetDateOfBirth() != nil {
		birthDate = resp.GetDateOfBirth().AsTime()
	}
	
	status := strings.TrimPrefix(resp.GetStatus().String(), "CUSTOMER_STATUS_")
	return resp.GetFullName(), birthDate, status, nil
}