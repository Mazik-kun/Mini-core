package grpcserver

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	accountv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/account/v1"
	commonv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/common/v1"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/app"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
	"github.com/google/uuid"
)

type Server struct {
	accountv1.UnimplementedAccountServiceServer
	openUC       *app.OpenAccountUseCase
	getUC        *app.GetAccountUseCase
	listUC       *app.ListAccountsUseCase
	blockUC      *app.BlockAccountUseCase   // <-- Новое
	closeUC      *app.CloseAccountUseCase   // <-- Новое
	balanceUC    *app.GetBalanceUseCase     // <-- Новое
	log          *slog.Logger
}

func NewServer(
	open *app.OpenAccountUseCase, 
	get *app.GetAccountUseCase, 
	list *app.ListAccountsUseCase, 
	block *app.BlockAccountUseCase, 
	closeUC *app.CloseAccountUseCase, 
	balance *app.GetBalanceUseCase, 
	log *slog.Logger,
) *Server {
	return &Server{
		openUC:    open, 
		getUC:     get, 
		listUC:    list, 
		blockUC:   block, 
		closeUC:   closeUC, 
		balanceUC: balance, 
		log:       log,
	}
}

func (s *Server) OpenAccount(ctx context.Context, req *accountv1.OpenAccountRequest) (*accountv1.Account, error) {
	cid, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id format")
	}

	// Пока маппим тип счета жестко в CHECKING, если в proto enum не передан
	accType := domain.TypeChecking
	if req.AccountType == accountv1.AccountType_ACCOUNT_TYPE_SAVINGS {
		accType = domain.TypeSavings
	}

	acc, err := s.openUC.OpenAccount(ctx, cid, req.Currency, accType)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return toProto(acc), nil
}

func (s *Server) GetAccount(ctx context.Context, req *accountv1.GetAccountRequest) (*accountv1.Account, error) {
	id, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id format")
	}

	acc, err := s.getUC.GetByID(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return toProto(acc), nil
}
func (s *Server) ListAccounts(ctx context.Context, req *accountv1.ListAccountsRequest) (*accountv1.ListAccountsResponse, error) {
	cid, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id format")
	}

	accs, err := s.listUC.ListByCustomer(ctx, cid)
	if err != nil {
		return nil, toGRPCError(err)
	}

	pbAccs := make([]*accountv1.Account, 0, len(accs))
	for _, a := range accs {
		pbAccs = append(pbAccs, toProto(a))
	}

	return &accountv1.ListAccountsResponse{
		Accounts: pbAccs,
		// Page: ... (пока пагинацию не делаем, ТЗ позволяет)
	}, nil
}

func (s *Server) GetBalance(ctx context.Context, req *accountv1.GetBalanceRequest) (*accountv1.GetBalanceResponse, error) {
	id, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id format")
	}

	bal, err := s.balanceUC.GetBalance(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &accountv1.GetBalanceResponse{
		Balance: &commonv1.Money{
			Amount:   bal.Amount,
			Currency: bal.Currency,
		},
	}, nil
}

func (s *Server) BlockAccount(ctx context.Context, req *accountv1.BlockAccountRequest)( *accountv1.Account, error){
	id, err := uuid.Parse(req.AccountId)

	if err != nil{
		return nil, status.Error(codes.InvalidArgument, "invalid account_id format")
	}
	acc, err := s.blockUC.BlockAccount(ctx, id)
	if err != nil{
		return nil, toGRPCError(err)
	}
	return toProto(acc), nil
}

func (s *Server) CloseAccount(ctx context.Context, req *accountv1.CloseAccountRequest) (*accountv1.Account, error){
	id, err := uuid.Parse(req.AccountId)
	if err != nil{
		return nil, status.Error(codes.InvalidArgument, "invalid account_id format")
	}
	acc, err := s.closeUC.CloseAccount(ctx, id)
	if err != nil{
		return nil, toGRPCError(err)
	}
	return toProto(acc), nil
}

// toGRPCError — переводит ошибки домена в стандартные gRPC коды.
// Это критично для шлюза, чтобы он вернул правильный HTTP-статус (404, 409, 412).
func toGRPCError(err error) error {
	switch {
	case errors.Is(err, domain.ErrAccountNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrLimitExceeded):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrCustomerNotActive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrInvalidStatus):
		return status.Error(codes.Aborted, err.Error())
	default:
		// Не отдаем внутренние ошибки БД наружу
		return status.Error(codes.Internal, "internal server error")
	}
}