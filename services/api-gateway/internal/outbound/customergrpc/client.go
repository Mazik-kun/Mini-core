// Package customergrpc — исходящий клиент customer-service.
package customergrpc

import (
	"google.golang.org/grpc"

	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/grpcconn"
)

type Client struct {
	customerv1.CustomerServiceClient
	conn *grpc.ClientConn
}

func New(o grpcconn.Options) (*Client, error) {
	o.Name = "customer"
	conn, err := grpcconn.Dial(o)
	if err != nil {
		return nil, err
	}
	return &Client{CustomerServiceClient: customerv1.NewCustomerServiceClient(conn), conn: conn}, nil
}

func (c *Client) Conn() *grpc.ClientConn { return c.conn }
func (c *Client) Close() error           { return c.conn.Close() }
