package documentgrpc

import (
	"google.golang.org/grpc"

	documentv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/document/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/outbound/grpcconn"
)

type Client struct {
	documentv1.DocumentServiceClient
	conn *grpc.ClientConn
}

func New(o grpcconn.Options) (*Client, error) {
	o.Name = "document"
	conn, err := grpcconn.Dial(o)
	if err != nil {
		return nil, err
	}
	return &Client{DocumentServiceClient: documentv1.NewDocumentServiceClient(conn), conn: conn}, nil
}

func (c *Client) Conn() *grpc.ClientConn { return c.conn }
func (c *Client) Close() error           { return c.conn.Close() }