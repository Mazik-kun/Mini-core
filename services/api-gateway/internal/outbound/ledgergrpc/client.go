package ledgergrpc

import (
	ledgerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/ledger/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/grpcconn"
	"google.golang.org/grpc"
)

type Client struct {
	ledgerv1.LedgerServiceClient
	conn *grpc.ClientConn
}

func New(o grpcconn.Options) (*Client, error){
	o.Name = "ledger"
	conn, err := grpcconn.Dial(o)
	if err != nil{
		return nil, err
	}

	return &Client{LedgerServiceClient: ledgerv1.NewLedgerServiceClient(conn), conn: conn}, nil
}

func (c *Client) Conn() *grpc.ClientConn {return c.conn}
func (c *Client) Close() error {return c.conn.Close()}