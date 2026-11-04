package grpcserver

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	accountv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/account/v1"
	"github.com/Mazik-kun/mini-core/services/account/internal/core/domain"
)

func toProto(a domain.Account) *accountv1.Account {
	return &accountv1.Account{
		AccountId:     a.ID.String(),
		CustomerId:    a.CustomerID.String(),
		AccountNumber: a.AccountNumber,
		Currency:      a.Currency,
		AccountType:   accountv1.AccountType(accountv1.AccountType_value["ACCOUNT_TYPE_"+string(a.Type)]),
		Status:        accountv1.AccountStatus(accountv1.AccountStatus_value["ACCOUNT_STATUS_"+string(a.Status)]),
		CreatedAt:     timestamppb.New(a.CreatedAt),
	}
}