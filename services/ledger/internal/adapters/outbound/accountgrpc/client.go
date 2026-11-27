package accountgrpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	accountv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/account/v1"
	"github.com/google/uuid"
)

type Client struct {
	cli accountv1.AccountServiceClient
}

func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("account grpc dial: %w", err)
	}
	return &Client{cli: accountv1.NewAccountServiceClient(conn)}, nil
}

func (c *Client) GetAccount(ctx context.Context, accountID uuid.UUID) (string, string, error) {
	resp, err := c.cli.GetAccount(ctx, &accountv1.GetAccountRequest{
		AccountId: accountID.String(),
	})
	if err != nil {
		return "", "", err
	}
	// Возвращаем валюту и статус (например, "GEL", "ACCOUNT_STATUS_ACTIVE")
	return resp.Currency, resp.Status.String(), nil
}