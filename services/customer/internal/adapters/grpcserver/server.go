package grpcserver

import (
	"context"
	"log/slog"

	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/services/customer/internal/app"
)

type CustomerServer struct {
	customerv1.UnimplementedCustomerServiceServer
	getCustomer *app.GetCustomerUseCase
	log *slog.Logger
}

func NewCustomerServer(getCustomer *app.GetCustomerUseCase) *CustomerServer{
	return &CustomerServer{getCustomer: getCustomer}
}

func (s * CustomerServer) GetCustomer(ctx context.Context, req *customerv1.GetCustomerRequest)(*customerv1.Customer, error){
	s.log.Info("GetCustomer called", "customerID", req.GetCustomerId())

	out, err := s.getCustomer.GetCustomer(ctx, app.GetCustomerInput{
		CustomerID: req.GetCustomerId(),
		RequesterRole: req.GetCus
		RequesterID uuid.UUID
	})
	return &customerv1.Customer{}, nil
}