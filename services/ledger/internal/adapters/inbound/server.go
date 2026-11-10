package grpcserver

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/common/v1"
	ledgerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/ledger/v1"
	"github.com/Mazik-kun/mini-core/services/ledger/internal/core/app"
	"github.com/Mazik-kun/mini-core/services/ledger/internal/core/domain"
)

type Server struct {
	ledgerv1.UnimplementedLedgerServiceServer
	transferUC   *app.TransferUseCase
	depositUC    *app.DepositUseCase
	balanceUC    *app.GetBalanceUseCase
	listUC *app.ListTransactionsUseCase
	log          *slog.Logger
}

func NewServer(transferUC *app.TransferUseCase, depositUC *app.DepositUseCase, balanceUC *app.GetBalanceUseCase, listUC *app.ListTransactionsUseCase, log *slog.Logger) *Server {
	return &Server{
		transferUC: transferUC,
		depositUC:  depositUC,
		balanceUC:  balanceUC,
		listUC: listUC,
		log:        log,
	}
}

func (s *Server) Transfer(ctx context.Context, req *ledgerv1.TransferRequest) (*ledgerv1.Transaction, error) {
	fromID, err := uuid.Parse(req.FromAccountId)
	if err != nil { return nil, status.Error(codes.InvalidArgument, "invalid from_account_id") }
	
	toID, err := uuid.Parse(req.ToAccountId)
	if err != nil { return nil, status.Error(codes.InvalidArgument, "invalid to_account_id") }

	in := app.TransferInput{
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         req.Amount.Amount,
		Currency:       req.Amount.Currency,
		IdempotencyKey: req.IdempotencyKey,
	}

	tx, err := s.transferUC.Transfer(ctx, in)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoTransaction(tx), nil
}

func (s *Server) Deposit(ctx context.Context, req *ledgerv1.DepositRequest) (*ledgerv1.Transaction, error) {
	accID, err := uuid.Parse(req.AccountId)
	if err != nil { return nil, status.Error(codes.InvalidArgument, "invalid account_id") }

	in := app.DepositInput{
		AccountID:      accID,
		Amount:         req.Amount.Amount,
		Currency:       req.Amount.Currency,
		IdempotencyKey: req.IdempotencyKey,
	}

	tx, err := s.depositUC.Deposit(ctx, in)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoTransaction(tx), nil
}

func (s *Server) GetBalance(ctx context.Context, req *ledgerv1.GetBalanceRequest) (*ledgerv1.GetBalanceResponse, error) {
	accID, err := uuid.Parse(req.AccountId)
	if err != nil { return nil, status.Error(codes.InvalidArgument, "invalid account_id") }

	bal, err := s.balanceUC.GetBalance(ctx, accID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &ledgerv1.GetBalanceResponse{
		Balance: &commonv1.Money{
			Amount:   bal.Amount,
			Currency: bal.Currency,
		},
	}, nil
}

// GetTransaction пока оставим Unimplemented для простоты
func (s *Server) GetTransaction(ctx context.Context, req *ledgerv1.GetTransactionRequest) (*ledgerv1.Transaction, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented yet")
}
func (s *Server) ListTransactions(ctx context.Context, req *ledgerv1.ListTransactionsRequest) (*ledgerv1.ListTransactionsResponse, error){
	accID, err := uuid.Parse(req.AccountId)
	if err != nil{
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	var fromTime, toTime pgtype.Timestamptz
	if req.From != nil {
		fromTime = pgtype.Timestamptz{Time: req.From.AsTime(), Valid: true}
	}
	if req.To != nil{
		toTime = pgtype.Timestamptz{Time: req.To.AsTime(), Valid: true}
	}

	limit := int32(20)
	offset := int32(0)
	if req.Page != nil && req.Page.PageSize > 0 {
		limit = req.Page.PageSize
	}

	txs, err := s.listUC.ListTransactions(ctx, accID, fromTime, toTime, limit, offset)

	if err != nil{
		return nil, toGRPCError(err)
	}

	pbTxs := make([]*ledgerv1.Transaction, 0, len(txs))

	for _, tx := range txs{
		pbTxs = append(pbTxs, toProtoTransaction(tx))
	}

	return &ledgerv1.ListTransactionsResponse{
		Transactions: pbTxs,
	}, nil
}
func toGRPCError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInsufficientFunds):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrSameAccount), errors.Is(err, domain.ErrCurrencyMismatch):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrTransactionNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}