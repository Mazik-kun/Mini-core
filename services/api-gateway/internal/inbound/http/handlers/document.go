package handlers

import (
	"net/http"

	"google.golang.org/grpc/codes"

	documentv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/document/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
)

type Document struct {
	client port.DocumentClient
}

func NewDocument(c port.DocumentClient) *Document { return &Document{client: c} }

// InitUpload — POST /v1/documents/init-upload.
// customer_id перетираем из JWT: клиент может инициировать загрузку только себе (IDOR).
func (h *Document) InitUpload(w http.ResponseWriter, r *http.Request) {
	customerID, ok := customerIDFromClaims(w, r)
	if !ok {
		return
	}
	var req documentv1.InitUploadRequest
	if !parseProtoJSON(w, r, &req) {
		return
	}
	req.CustomerId = customerID

	resp, err := h.client.InitUpload(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}

// ConfirmUpload — POST /v1/documents/confirm-upload.
// customer_id не передаётся в теле; ownership проверяет document-service по JWT из metadata.
func (h *Document) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	var req documentv1.ConfirmUploadRequest
	if !parseProtoJSON(w, r, &req) {
		return
	}
	resp, err := h.client.ConfirmUpload(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}

// GetDownloadURL — GET /v1/documents/{id}/url.
// Ownership (клиент/офицер) — на стороне document-service по JWT.
func (h *Document) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierr.WriteStatus(w, r, codes.InvalidArgument, "missing document id")
		return
	}
	resp, err := h.client.GetDownloadUrl(r.Context(), &documentv1.GetDownloadUrlRequest{
		DocumentId: id,
	})
	respondProtoJSON(w, r, resp, err)
}