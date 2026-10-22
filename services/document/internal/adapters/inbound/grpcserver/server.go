package grpcserver

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/common/v1"
	documentv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/document/v1"
	"github.com/Mazik-kun/mini-core/services/document/internal/core/app"
	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

type DocumentServer struct {
	documentv1.UnimplementedDocumentServiceServer

	initUpload           *app.InitUploadUseCase
	confirmUpload        *app.ConfirmUploadUseCase
	getDownloadURL       *app.GetDownloadURLUseCase
	listDocuments        *app.ListDocumentsUseCase
	hasRequiredDocuments *app.HasRequiredDocumentsUseCase

	log *slog.Logger
}

func NewDocumentServer(
	initUpload *app.InitUploadUseCase,
	confirmUpload *app.ConfirmUploadUseCase,
	getDownloadURL *app.GetDownloadURLUseCase,
	listDocuments *app.ListDocumentsUseCase,
	hasRequiredDocuments *app.HasRequiredDocumentsUseCase,
	log *slog.Logger,
) *DocumentServer {
	return &DocumentServer{
		initUpload:           initUpload,
		confirmUpload:        confirmUpload,
		getDownloadURL:       getDownloadURL,
		listDocuments:        listDocuments,
		hasRequiredDocuments: hasRequiredDocuments,
		log:                  log,
	}
}

func (s *DocumentServer) InitUpload(ctx context.Context, req *documentv1.InitUploadRequest) (*documentv1.InitUploadResponse, error) {
	customerID, err := uuid.Parse(req.GetCustomerId())
	if err != nil {
		return nil, mapError(domain.ErrInvalidCustomerID)
	}
	docType, ok := typeFromProto(req.GetDocumentType())
	if !ok {
		return nil, mapError(domain.ErrInvalidDocumentType)
	}

	out, err := s.initUpload.InitUpload(ctx, app.InitUploadInput{CustomerID: customerID, Type: docType})
	if err != nil {
		return nil, mapError(err)
	}
	return &documentv1.InitUploadResponse{DocumentId: out.DocumentID.String(), UploadUrl: out.UploadURL}, nil
}

func (s *DocumentServer) ConfirmUpload(ctx context.Context, req *documentv1.ConfirmUploadRequest) (*documentv1.Document, error) {
	documentID, err := uuid.Parse(req.GetDocumentId())
	if err != nil {
		return nil, mapError(domain.ErrInvalidDocumentID)
	}

	doc, err := s.confirmUpload.ConfirmUpload(ctx, app.ConfirmUploadInput{DocumentID: documentID})
	if err != nil {
		return nil, mapError(err)
	}
	return documentToProto(doc), nil
}

func (s *DocumentServer) GetDownloadUrl(ctx context.Context, req *documentv1.GetDownloadUrlRequest) (*documentv1.GetDownloadUrlResponse, error) {
	documentID, err := uuid.Parse(req.GetDocumentId())
	if err != nil {
		return nil, mapError(domain.ErrInvalidDocumentID)
	}

	out, err := s.getDownloadURL.GetDownloadURL(ctx, app.GetDownloadURLInput{DocumentID: documentID})
	if err != nil {
		return nil, mapError(err)
	}
	return &documentv1.GetDownloadUrlResponse{
		DownloadUrl: out.URL,
		ExpiresAt:   timestamppb.New(out.ExpiresAt),
	}, nil
}

func (s *DocumentServer) ListDocuments(ctx context.Context, req *documentv1.ListDocumentsRequest) (*documentv1.ListDocumentsResponse, error) {
	customerID, err := uuid.Parse(req.GetCustomerId())
	if err != nil {
		return nil, mapError(domain.ErrInvalidCustomerID)
	}
	statusFilter, ok := statusFilterFromProto(req.GetStatusFilter())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid status_filter")
	}

	var pageSize int
	var pageToken string
	if page := req.GetPage(); page != nil {
		pageSize = int(page.GetPageSize())
		pageToken = page.GetPageToken()
	}

	out, err := s.listDocuments.ListDocuments(ctx, app.ListDocumentsInput{
		CustomerID: customerID,
		Status:     statusFilter,
		PageSize:   pageSize,
		PageToken:  pageToken,
	})
	if err != nil {
		return nil, mapError(err)
	}

	docs := make([]*documentv1.Document, len(out.Documents))
	for i, d := range out.Documents {
		docs[i] = documentToProto(d)
	}
	return &documentv1.ListDocumentsResponse{
		Documents: docs,
		Page:      &commonv1.PageResponse{NextPageToken: out.NextPageToken},
	}, nil
}

func (s *DocumentServer) HasRequiredDocuments(ctx context.Context, req *documentv1.HasRequiredDocumentsRequest) (*documentv1.HasRequiredDocumentsResponse, error) {
	customerID, err := uuid.Parse(req.GetCustomerId())
	if err != nil {
		return nil, mapError(domain.ErrInvalidCustomerID)
	}

	required := make([]domain.Type, 0, len(req.GetRequiredTypes()))
	for _, t := range req.GetRequiredTypes() {
		docType, ok := typeFromProto(t)
		if !ok {
			return nil, mapError(domain.ErrInvalidDocumentType)
		}
		required = append(required, docType)
	}

	out, err := s.hasRequiredDocuments.HasRequiredDocuments(ctx, app.HasRequiredDocumentsInput{
		CustomerID: customerID,
		Required:   required,
	})
	if err != nil {
		return nil, mapError(err)
	}

	missing := make([]documentv1.DocumentType, len(out.Missing))
	for i, t := range out.Missing {
		missing[i] = typeToProto(t)
	}
	return &documentv1.HasRequiredDocumentsResponse{HasAll: out.HasAll, MissingTypes: missing}, nil
}
