package grpcserver

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/common/v1"
	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/services/customer/internal/app"
)

type CustomerServer struct {
	customerv1.UnimplementedCustomerServiceServer

	getCustomer       *app.GetCustomerUseCase
	getCustomerStatus *app.GetCustomerStatusUseCase
	listCustomers     *app.ListCustomersUseCase
	updateProfile     *app.UpdateProfileUseCase

	log *slog.Logger
}

func NewCustomerServer(
	getCustomer *app.GetCustomerUseCase,
	getCustomerStatus *app.GetCustomerStatusUseCase,
	listCustomers *app.ListCustomersUseCase,
	updateProfile *app.UpdateProfileUseCase,
	log *slog.Logger,
) *CustomerServer {
	return &CustomerServer{
		getCustomer:       getCustomer,
		getCustomerStatus: getCustomerStatus,
		listCustomers:     listCustomers,
		updateProfile:     updateProfile,
		log:               log,
	}
}

func (s *CustomerServer) GetCustomer(ctx context.Context, req *customerv1.GetCustomerRequest) (*customerv1.Customer, error) {
	customerID, err := uuid.Parse(req.GetCustomerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id")
	}

	requesterID, requesterRole := RequesterFromContext(ctx)

	customer, err := s.getCustomer.GetCustomer(ctx, app.GetCustomerInput{
		CustomerID:    customerID,
		RequesterID:   requesterID,
		RequesterRole: requesterRole,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return customerToProto(customer), nil
}

func (s *CustomerServer) GetCustomerStatus(ctx context.Context, req *customerv1.GetCustomerStatusRequest) (*customerv1.GetCustomerStatusResponse, error) {
	customerID, err := uuid.Parse(req.GetCustomerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id")
	}

	requesterID, requesterRole := RequesterFromContext(ctx)

	current, err := s.getCustomerStatus.GetCustomerStatus(ctx, app.GetCustomerStatusInput{
		CustomerID:    customerID,
		RequesterID:   requesterID,
		RequesterRole: requesterRole,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &customerv1.GetCustomerStatusResponse{Status: statusToProto(current)}, nil
}

func (s *CustomerServer) UpdateProfile(ctx context.Context, req *customerv1.UpdateProfileRequest) (*customerv1.Customer, error) {
	customerID, err := uuid.Parse(req.GetCustomerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id")
	}

	requesterID, requesterRole := RequesterFromContext(ctx)

	var birthDate pgtype.Date
	if ts := req.GetDateOfBirth(); ts != nil {
		birthDate = pgtype.Date{Time: ts.AsTime(), Valid: true}
	}

	updated, err := s.updateProfile.UpdateProfile(ctx, app.UpdateProfileInput{
		CustomerID:    customerID,
		RequesterID:   requesterID,
		RequesterRole: requesterRole,
		FullName:      req.GetFullName(),
		BirthDate:     birthDate,
		Address:       req.GetAddress(),
		PhoneNumber:   req.GetPhone(),
		Citizenship:   req.GetCitizenship(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return customerToProto(updated), nil
}

func (s *CustomerServer) ListCustomers(ctx context.Context, req *customerv1.ListCustomersRequest) (*customerv1.ListCustomersResponse, error) {
	requesterID, requesterRole := RequesterFromContext(ctx)
	_ = requesterID // use case сам проверит роль

	statuses, ok := statusFilterFromProto(req.GetStatusFilter())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid status_filter")
	}

	var pageSize int
	var pageToken string
	if page := req.GetPage(); page != nil {
		pageSize = int(page.GetPageSize())
		pageToken = page.GetPageToken()
	}

	out, err := s.listCustomers.ListCustomers(ctx, app.ListCustomersInput{
		RequesterRole: requesterRole,
		Statuses:      statuses,
		PageSize:      pageSize,
		PageToken:     pageToken,
	})
	if err != nil {
		return nil, mapError(err)
	}

	customers := make([]*customerv1.Customer, len(out.Customers))
	for i, c := range out.Customers {
		customers[i] = customerToProto(c)
	}

	return &customerv1.ListCustomersResponse{
		Customers: customers,
		Page:      &commonv1.PageResponse{NextPageToken: out.NextPageToken},
	}, nil
}