package documentgrpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	documentv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/document/v1"
)

type Client struct {
	cli documentv1.DocumentServiceClient
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
	return &Client{cli: documentv1.NewDocumentServiceClient(conn)}, nil
}

func (c *Client) HasRequiredDocuments(ctx context.Context, customerID uuid.UUID, requiredTypes []string) (bool, []string, error) {
	var types []documentv1.DocumentType
	for _, t := range requiredTypes {
		if t == "PASSPORT" { types = append(types, documentv1.DocumentType_DOCUMENT_TYPE_PASSPORT) }
		if t == "SELFIE" { types = append(types, documentv1.DocumentType_DOCUMENT_TYPE_SELFIE) }
	}

	resp, err := c.cli.HasRequiredDocuments(ctx, &documentv1.HasRequiredDocumentsRequest{
		CustomerId:    customerID.String(),
		RequiredTypes: types,
	})
	if err != nil {
		return false, nil, err
	}

	var missing []string
	for _, m := range resp.GetMissingTypes() {
		missing = append(missing, m.String())
	}

	return resp.GetHasAll(), missing, nil
}