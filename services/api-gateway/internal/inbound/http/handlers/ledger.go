package handlers

import (
	"net/http"

	ledgerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/ledger/v1"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
	"google.golang.org/grpc/codes"
)

type Ledger struct {
	client port.LedgerClient
	accountClient port.AccountClient
}

func NewLedger(c port.LedgerClient, ac port.AccountClient) *Ledger{
	return &Ledger{client: c, accountClient: ac}
}

func (h *Ledger) Transfer(w http.ResponseWriter, r *http.Request){
	customerID, ok := customerIDFromClaims(w, r)
	if !ok {return}

	var req ledgerv1.TransferRequest
	if !parseProtoJSON(w, r, &req){
		return
	}

	isOwner, err := checkAccountOwnership(r.Context(), h.accountClient, customerID, req.FromAccountId)
	if err != nil{
		respondProtoJSON(w, r, nil, err)
		return
	}
	if !isOwner{
		apierr.WriteStatus(w, r, codes.PermissionDenied, "this is not your account")
		return
	}

	resp, err := h.client.Transfer(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}

func (h *Ledger) Deposit(w http.ResponseWriter, r *http.Request) {
	claims, ok := reqctx.Claims(r.Context())
	if !ok {
		apierr.WriteStatus(w, r, codes.Unauthenticated, "authentication required")
		return
	}

	if claims.Role != roles.RoleOfficer && claims.Role != roles.RoleAdmin {
		apierr.WriteStatus(w, r, codes.PermissionDenied, "only officers can make deposits")
		return
	}

	var req ledgerv1.DepositRequest
	if !parseProtoJSON(w, r, &req) {
		return
	}

	// 4. Идем в Ledger
	resp, err := h.client.Deposit(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}