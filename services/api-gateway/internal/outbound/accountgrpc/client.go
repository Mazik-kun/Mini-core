package accountgrpc

import (
	accountv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/account/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/grpcconn"
	"google.golang.org/grpc"
)

type Client struct {
	accountv1.AccountServiceClient
	conn *grpc.ClientConn
}

func New(o grpcconn.Options)(*Client, error){
	o.Name = "account"
	conn, err := grpcconn.Dial(o)
	if err != nil {
		return nil, err
	}

	return &Client{AccountServiceClient: accountv1.NewAccountServiceClient(conn), conn: conn}, nil
}

func (c *Client) Conn() *grpc.ClientConn {return c.conn}
func (c *Client) Close() error {return c.conn.Close()}