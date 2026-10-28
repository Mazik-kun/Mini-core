package kycgrpc

import (
	"google.golang.org/grpc"

	kycv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/kyc/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/grpcconn"
)

type Client struct {
	kycv1.KycServiceClient
	conn *grpc.ClientConn
}

func New(o grpcconn.Options) (*Client, error) {
	o.Name = "kyc"
	conn, err := grpcconn.Dial(o)
	if err != nil {
		return nil, err
	}
	return &Client{KycServiceClient: kycv1.NewKycServiceClient(conn), conn: conn}, nil
}

func (c *Client) Conn() *grpc.ClientConn { return c.conn }
func (c *Client) Close() error           { return c.conn.Close() }