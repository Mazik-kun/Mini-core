package grpcserver

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

func customerToProto(c domain.Customer) *customerv1.Customer {
	out := &customerv1.Customer{
		CustomerId:  c.ID.String(),
		FullName:    c.FullName,
		Address:     c.Address,
		Phone:       c.PhoneNumber,
		Citizenship: c.Citizenship,
		Status:      statusToProto(c.Status),
	}
	if c.BirthDate.Valid {
		out.DateOfBirth = timestamppb.New(c.BirthDate.Time)
	}
	if !c.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(c.CreatedAt)
	}
	if !c.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(c.UpdatedAt)
	}
	return out
}

func statusToProto(s domain.Status) customerv1.CustomerStatus {
	switch s {
	case domain.StatusNew:
		return customerv1.CustomerStatus_CUSTOMER_STATUS_NEW
	case domain.StatusProfileFilled:
		return customerv1.CustomerStatus_CUSTOMER_STATUS_PROFILE_FILLED
	case domain.StatusOnKYC:
		return customerv1.CustomerStatus_CUSTOMER_STATUS_ON_KYC
	case domain.StatusActive:
		return customerv1.CustomerStatus_CUSTOMER_STATUS_ACTIVE
	case domain.StatusRejected:
		return customerv1.CustomerStatus_CUSTOMER_STATUS_REJECTED
	case domain.StatusBlocked:
		return customerv1.CustomerStatus_CUSTOMER_STATUS_BLOCKED
	default:
		return customerv1.CustomerStatus_CUSTOMER_STATUS_UNSPECIFIED
	}
}

// statusFilterFromProto возвращает ([]Status, ok). UNSPECIFIED → пустой список (без фильтра).
func statusFilterFromProto(s customerv1.CustomerStatus) ([]domain.Status, bool) {
	switch s {
	case customerv1.CustomerStatus_CUSTOMER_STATUS_UNSPECIFIED:
		return nil, true
	case customerv1.CustomerStatus_CUSTOMER_STATUS_NEW:
		return []domain.Status{domain.StatusNew}, true
	case customerv1.CustomerStatus_CUSTOMER_STATUS_PROFILE_FILLED:
		return []domain.Status{domain.StatusProfileFilled}, true
	case customerv1.CustomerStatus_CUSTOMER_STATUS_ON_KYC:
		return []domain.Status{domain.StatusOnKYC}, true
	case customerv1.CustomerStatus_CUSTOMER_STATUS_ACTIVE:
		return []domain.Status{domain.StatusActive}, true
	case customerv1.CustomerStatus_CUSTOMER_STATUS_REJECTED:
		return []domain.Status{domain.StatusRejected}, true
	case customerv1.CustomerStatus_CUSTOMER_STATUS_BLOCKED:
		return []domain.Status{domain.StatusBlocked}, true
	default:
		return nil, false
	}
}