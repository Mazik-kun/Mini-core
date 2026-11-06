package grpcserver

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	commonv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/common/v1"
	ledgerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/ledger/v1"
	"github.com/Mazik-kun/mini-core/services/ledger/internal/core/domain"
)

func toProtoTransaction(t domain.Transaction) *ledgerv1.Transaction {
	return &ledgerv1.Transaction{
		TransactionId:  t.ID.String(),
		FromAccountId:  t.FromAccountID.String(),
		ToAccountId:    t.ToAccountID.String(),
		Amount: &commonv1.Money{
			Amount:   t.Amount,
			Currency: t.Currency,
		},
		Status:         toProtoStatus(t.Status),
		Type:           toProtoType(t.Type),
		IdempotencyKey: t.IdempotencyKey,
		CreatedAt:      timestamppb.New(t.CreatedAt),
	}
}

func toProtoStatus(s domain.Status) ledgerv1.TransactionStatus {
	switch s {
	case domain.StatusCompleted: return ledgerv1.TransactionStatus_TRANSACTION_STATUS_COMPLETED
	case domain.StatusPending: return ledgerv1.TransactionStatus_TRANSACTION_STATUS_PENDING
	case domain.StatusDeclined: return ledgerv1.TransactionStatus_TRANSACTION_STATUS_DECLINED
	default: return ledgerv1.TransactionStatus_TRANSACTION_STATUS_UNSPECIFIED
	}
}

func toProtoType(t domain.TransactionType) ledgerv1.TransactionType {
	switch t {
	case domain.TypeTransfer: return ledgerv1.TransactionType_TRANSACTION_TYPE_TRANSFER
	case domain.TypeDeposit: return ledgerv1.TransactionType_TRANSACTION_TYPE_DEPOSIT
	default: return ledgerv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED
	}
}