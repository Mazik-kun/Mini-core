package grpcserver

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	kycv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/kyc/v1"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/domain"
)

func statusToProto(s domain.Status) kycv1.KycStatus {
	switch s {
	case domain.StatusDraft:
		return kycv1.KycStatus_KYC_STATUS_DRAFT
	case domain.StatusSubmitted:
		return kycv1.KycStatus_KYC_STATUS_SUBMITTED
	case domain.StatusAutoChecks:
		return kycv1.KycStatus_KYC_STATUS_AUTO_CHECKS
	case domain.StatusInReview:
		return kycv1.KycStatus_KYC_STATUS_IN_REVIEW
	case domain.StatusApproved:
		return kycv1.KycStatus_KYC_STATUS_APPROVED
	case domain.StatusRejected:
		return kycv1.KycStatus_KYC_STATUS_REJECTED
	case domain.StatusAutoRejected:
		return kycv1.KycStatus_KYC_STATUS_AUTO_REJECTED
	case domain.StatusDocsRequested:
		return kycv1.KycStatus_KYC_STATUS_DOCS_REQUESTED
	default:
		return kycv1.KycStatus_KYC_STATUS_UNSPECIFIED
	}
}

func toProto(app domain.Application) *kycv1.Application {
	reviewerID := ""
	if app.OfficerID != nil {
		reviewerID = app.OfficerID.String()
	}

	return &kycv1.Application{
		ApplicationId:   app.ID.String(),
		CustomerId:      app.CustomerID.String(),
		Status:          statusToProto(app.Status),
		ReviewerId:      reviewerID,
		RejectionReason: app.Reason,
		CreatedAt:       timestamppb.New(app.CreatedAt),
		UpdatedAt:       timestamppb.New(app.UpdatedAt),
	}
}