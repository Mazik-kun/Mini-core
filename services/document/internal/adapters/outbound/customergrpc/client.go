package customergrpc

import (
	"context"
	"fmt"

	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/pkg/grpc/interceptors"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	cli customerv1.CustomerServiceClient
}

func New(addr string) (*Client, error) {
    conn, err := grpc.NewClient(addr,grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(interceptors.ForwardAuthUnary()))
    if err != nil {
        return nil, fmt.Errorf("customer grpc dial: %w", err)
    }
    return &Client{cli: customerv1.NewCustomerServiceClient(conn)}, nil
}

func (c *Client) GetCustomerStatus(ctx context.Context, customerID uuid.UUID) (string, error) {
	resp, err := c.cli.GetCustomer(ctx, &customerv1.GetCustomerRequest{CustomerId: customerID.String()})
	if err != nil {
		return "", err
	}
	return resp.GetStatus().String(), nil
}