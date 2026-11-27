package ledgergrpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	ledgerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/ledger/v1"
)

type Client struct {
	cli ledgerv1.LedgerServiceClient
}

func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("ledger grpc dial: %w", err)
	}
	return &Client{cli: ledgerv1.NewLedgerServiceClient(conn)}, nil
}

func (c *Client) GetBalance(ctx context.Context, accountID uuid.UUID) (int64, error) {
	resp, err := c.cli.GetBalance(ctx, &ledgerv1.GetBalanceRequest{
		AccountId: accountID.String(),
	})
	if err != nil {
		return 0, err
	}

	if resp.GetBalance() == nil {
		return 0, fmt.Errorf("ledger returned empty balance for account %s", accountID)
	}
	return resp.GetBalance().GetAmount(), nil
}