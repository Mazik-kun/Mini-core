package grpcserver

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
}

func NewServer(submitUC *app.SubmitApplicationUseCase, officerUC *app.OfficerActionsUseCase, getUC *app.GetApplicationUseCase, listUC *app.ListApplicationsUseCase) *Server {
	return &Server{submitUC: submitUC, officerUC: officerUC, getUC: getUC, listUC: listUC}
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
	appID, _ := uuid.Parse(req.GetApplicationId())
	out, err := s.officerUC.ApproveApplication(ctx, appID)
	if err != nil {
		return nil, mapError(err)
	}
	return toProto(out), nil
}

func (s *Server) RejectApplication(ctx context.Context, req *kycv1.RejectApplicationRequest) (*kycv1.Application, error) {
	appID, _ := uuid.Parse(req.GetApplicationId())
	out, err := s.officerUC.RejectApplication(ctx, appID, req.GetReason())
	if err != nil {
		return nil, mapError(err)
	}
	return toProto(out), nil
}

// mapError переводит ошибки домена в gRPC-коды
func mapError(err error) error {
	switch {
	case err == domain.ErrAccessDenied:
		return status.Error(codes.PermissionDenied, err.Error())
	case err == domain.ErrApplicationNotFound:
		return status.Error(codes.NotFound, err.Error())
	case err == domain.ErrInvalidStatusTransition, err == domain.ErrActiveApplicationExists, err == domain.ErrProfileNotFilled, err == domain.ErrDocumentsMissing:
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
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
	return nil, status.Error(codes.Unimplemented, "not implemented yet")
}