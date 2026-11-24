package grpcserver

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/common/v1"
	kycv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/kyc/v1"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/app"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/domain"
)

type Server struct {
	kycv1.UnimplementedKycServiceServer
	submitUC  *app.SubmitApplicationUseCase
	officerUC *app.OfficerActionsUseCase
	getUC     *app.GetApplicationUseCase
	listUC    *app.ListApplicationsUseCase
	resubmitUC *app.ResubmitApplicationUseCase
}

func NewServer(submitUC *app.SubmitApplicationUseCase, officerUC *app.OfficerActionsUseCase, getUC *app.GetApplicationUseCase, listUC *app.ListApplicationsUseCase, resubmitUC *app.ResubmitApplicationUseCase) *Server {
	return &Server{submitUC: submitUC, officerUC: officerUC, getUC: getUC, listUC: listUC, resubmitUC: resubmitUC}
}

func (s *Server) SubmitApplication(ctx context.Context, req *kycv1.SubmitApplicationRequest) (*kycv1.Application, error) {
	custID, err := uuid.Parse(req.GetCustomerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id")
	}

	out, err := s.submitUC.SubmitApplication(ctx, app.SubmitApplicationInput{CustomerID: custID})
	if err != nil {
		return nil, mapError(err)
	}
	return toProto(out), nil
}

func (s *Server) ClaimApplication(ctx context.Context, req *kycv1.ClaimApplicationRequest) (*kycv1.Application, error) {
	appID, err := uuid.Parse(req.GetApplicationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid application_id")
	}

	out, err := s.officerUC.ClaimApplication(ctx, appID)
	if err != nil {
		return nil, mapError(err)
	}
	return toProto(out), nil
}

func (s *Server) ApproveApplication(ctx context.Context, req *kycv1.ApproveApplicationRequest) (*kycv1.Application, error) {
	appID, err := uuid.Parse(req.GetApplicationId())
	if err !=nil {
		return nil, status.Error(codes.InvalidArgument, "invalid application_id")
	}
	
	out, err := s.officerUC.ApproveApplication(ctx, appID)
	if err != nil {
		return nil, mapError(err)
	}
	return toProto(out), nil
}

func (s *Server) RejectApplication(ctx context.Context, req *kycv1.RejectApplicationRequest) (*kycv1.Application, error) {
	appID, err := uuid.Parse(req.GetApplicationId())
	if err !=nil {
		return nil, status.Error(codes.InvalidArgument, "invalid application_id")
	}

	out, err := s.officerUC.RejectApplication(ctx, appID, req.GetReason())
	if err != nil {
		return nil, mapError(err)
	}
	return toProto(out), nil
}

// mapError переводит ошибки домена в gRPC-коды
func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrApplicationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidStatusTransition),
		errors.Is(err, domain.ErrActiveApplicationExists),
		errors.Is(err, domain.ErrProfileNotFilled),
		errors.Is(err, domain.ErrDocumentsMissing):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func (s *Server) RequestMoreDocuments(ctx context.Context, req *kycv1.RequestMoreDocumentsRequest) (*kycv1.Application, error) {
	appID, err := uuid.Parse(req.GetApplicationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid application_id")
	}

	out, err := s.officerUC.RequestMoreDocuments(ctx, appID, req.GetMessage())
	if err != nil {
		return nil, mapError(err)
	}
	return toProto(out), nil
}

func (s *Server) ResubmitApplication(ctx context.Context, req *kycv1.ResubmitApplicationRequest) (*kycv1.Application, error) {
	appID, err := uuid.Parse(req.GetApplicationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid application_id")
	}

	out, err := s.resubmitUC.ResubmitApplication(ctx, appID)
	if err != nil {
		return nil, mapError(err)
	}
	return toProto(out), nil
}

func (s *Server) GetApplication(ctx context.Context, req *kycv1.GetApplicationRequest) (*kycv1.Application, error) {
	appID, err := uuid.Parse(req.GetApplicationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid application_id")
	}

	out, err := s.getUC.GetApplication(ctx, appID)
	if err != nil {
		return nil, mapError(err)
	}
	return toProto(out), nil
}

func (s *Server) ListApplications(ctx context.Context, req *kycv1.ListApplicationsRequest) (*kycv1.ListApplicationsResponse, error) {
	input := app.ListApplicationsInput{
		PageSize:  50,
		PageToken: "",
	}

	if req.Page != nil {
		if req.Page.PageSize > 0 {
			input.PageSize = int(req.Page.PageSize)
		}
		input.PageToken = req.Page.PageToken
	}

	if req.StatusFilter != kycv1.KycStatus_KYC_STATUS_UNSPECIFIED {
		st, ok := statusFromProto(req.StatusFilter)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "invalid status filter")
		}
		input.Status = &st
	}

	apps, err := s.listUC.ListApplications(ctx, input)
	if err != nil {
		return nil, mapError(err)
	}

	pbApps := make([]*kycv1.Application, 0, len(apps))
	for _, a := range apps {
		pbApps = append(pbApps, toProto(a))
	}

	return &kycv1.ListApplicationsResponse{
		Applications: pbApps,
		Page: &commonv1.PageResponse{},
	}, nil
}