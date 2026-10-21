// Package authgrpc — исходящий клиент auth-service.
package authgrpc

import (
	"google.golang.org/grpc"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/grpcconn"
)

type Client struct {
	authv1.AuthServiceClient
	conn *grpc.ClientConn
}

func New(o grpcconn.Options) (*Client, error) {
	o.Name = "auth"
	conn, err := grpcconn.Dial(o)
	if err != nil {
		return nil, err
	}
	return &Client{AuthServiceClient: authv1.NewAuthServiceClient(conn), conn: conn}, nil
}

func (c *Client) Conn() *grpc.ClientConn { return c.conn }
func (c *Client) Close() error           { return c.conn.Close() }
