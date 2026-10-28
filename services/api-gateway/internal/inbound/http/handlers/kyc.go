package handlers

import (
	"net/http"

	"google.golang.org/grpc/codes"

	kycv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/kyc/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
)

type KYC struct {
	client port.KYCClient
}

func NewKYC(c port.KYCClient) *KYC { return &KYC{client: c} }

// Submit — POST /v1/kyc/applications.
// customer_id перетираем из claims (IDOR). Валидность document_ids (что это его документы,
// обязательные типы загружены) — ответственность kyc-service по ТЗ п.5.4.
func (h *KYC) Submit(w http.ResponseWriter, r *http.Request) {
	customerID, ok := customerIDFromClaims(w, r)
	if !ok {
		return
	}
	var req kycv1.SubmitApplicationRequest
	if !parseProtoJSON(w, r, &req) {
		return
	}
	req.CustomerId = customerID

	resp, err := h.client.SubmitApplication(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}

// Claim — POST /v1/kyc/applications/{id}/claim (OFFICER).
// reviewer_id берём из Subject; роль OFFICER проверяет kyc-service по форварднутому JWT.
func (h *KYC) Claim(w http.ResponseWriter, r *http.Request) {
	reviewerID, ok := reviewerIDFromClaims(w, r)
	if !ok {
		return
	}
	appID := r.PathValue("id")
	if appID == "" {
		apierr.WriteStatus(w, r, codes.InvalidArgument, "missing application id")
		return
	}
	resp, err := h.client.ClaimApplication(r.Context(), &kycv1.ClaimApplicationRequest{
		ApplicationId: appID,
		ReviewerId:    reviewerID,
	})
	respondProtoJSON(w, r, resp, err)
}

// Approve — POST /v1/kyc/applications/{id}/approve (OFFICER).
func (h *KYC) Approve(w http.ResponseWriter, r *http.Request) {
	reviewerID, ok := reviewerIDFromClaims(w, r)
	if !ok {
		return
	}
	appID := r.PathValue("id")
	if appID == "" {
		apierr.WriteStatus(w, r, codes.InvalidArgument, "missing application id")
		return
	}
	resp, err := h.client.ApproveApplication(r.Context(), &kycv1.ApproveApplicationRequest{
		ApplicationId: appID,
		ReviewerId:    reviewerID,
	})
	respondProtoJSON(w, r, resp, err)
}