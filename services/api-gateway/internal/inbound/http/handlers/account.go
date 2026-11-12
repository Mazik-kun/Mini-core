package handlers

import (
	"net/http"

	accountv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/account/v1"
	ledgerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/ledger/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

type Account struct {
	client port.AccountClient
	ledgerClient port.LedgerClient
}
func NewAccount (c port.AccountClient, lc port.LedgerClient) *Account {return &Account{client: c, ledgerClient: lc}}

func (h *Account) ListMyAccounts(w http.ResponseWriter, r *http.Request){
	customerID, ok := customerIDFromClaims(w,r)
	if !ok {return}

	resp, err := h.client.ListAccounts(r.Context(), &accountv1.ListAccountsRequest{
		CustomerId:customerID,
	})
	respondProtoJSON(w, r, resp, err)
}

func (h *Account) GetBalance(w http.ResponseWriter, r *http.Request){
	customerID, ok := customerIDFromClaims(w, r)
	if !ok{
		return
	}

	accID := r.PathValue("id")
	if _, err := uuid.Parse(accID); err != nil{
		apierr.WriteStatus(w, r, codes.InvalidArgument, "invalid account id")
		return
	}
	isOwner, err := checkAccountOwnership(r.Context(), h.client, customerID, accID)
	if err != nil{
		respondProtoJSON(w, r, nil, err)
		return
	}
	if !isOwner{
		apierr.WriteStatus(w, r, codes.PermissionDenied, "this is not your account")
		return
	}
	resp, err := h.ledgerClient.GetBalance(r.Context(), &ledgerv1.GetBalanceRequest{
		AccountId: accID,
	})

	respondProtoJSON(w, r, resp, err)
}
	