package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/common/v1"
	ledgerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/ledger/v1"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
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

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// ListByAccount — GET /v1/accounts/{id}/transactions
// Query: page_size (default 20, max 100), page_token (offset), from, to (RFC3339).
func (h *Ledger) ListByAccount(w http.ResponseWriter, r *http.Request) {
	customerID, ok := customerIDFromClaims(w, r)
	if !ok {
		return
	}

	accID := r.PathValue("id")
	if _, err := uuid.Parse(accID); err != nil {
		apierr.WriteStatus(w, r, codes.InvalidArgument, "invalid account id")
		return
	}

	q := r.URL.Query()

	pageSize := int32(defaultPageSize)
	if v := q.Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			apierr.WriteStatus(w, r, codes.InvalidArgument, "page_size must be a positive integer")
			return
		}
		if n > maxPageSize {
			n = maxPageSize
		}
		pageSize = int32(n)
	}

	// page_token — это offset; ledger-сервис разбирает его сам, здесь только отсекаем мусор
	pageToken := q.Get("page_token")
	if pageToken != "" {
		if n, err := strconv.Atoi(pageToken); err != nil || n < 0 {
			apierr.WriteStatus(w, r, codes.InvalidArgument, "invalid page_token")
			return
		}
	}

	from, ok := parseTimeQuery(w, r, "from")
	if !ok {
		return
	}
	to, ok := parseTimeQuery(w, r, "to")
	if !ok {
		return
	}
	if from != nil && to != nil && from.AsTime().After(to.AsTime()) {
		apierr.WriteStatus(w, r, codes.InvalidArgument, "from must not be after to")
		return
	}

	// IDOR: счёт должен принадлежать клиенту
	isOwner, err := checkAccountOwnership(r.Context(), h.accountClient, customerID, accID)
	if err != nil {
		respondProtoJSON(w, r, nil, err)
		return
	}
	if !isOwner {
		apierr.WriteStatus(w, r, codes.PermissionDenied, "this is not your account")
		return
	}

	resp, err := h.client.ListTransactions(r.Context(), &ledgerv1.ListTransactionsRequest{
		AccountId: accID,
		Page: &commonv1.PageRequest{
			PageSize:  pageSize,
			PageToken: pageToken,
		},
		From: from,
		To:   to,
	})
	respondProtoJSON(w, r, resp, err)
}

// GetTransaction — GET /v1/transactions/{id}
// Чужую транзакцию отдаём как 404, чтобы не раскрывать её существование.
func (h *Ledger) GetTransaction(w http.ResponseWriter, r *http.Request) {
	customerID, ok := customerIDFromClaims(w, r)
	if !ok {
		return
	}

	txID := r.PathValue("id")
	if _, err := uuid.Parse(txID); err != nil {
		apierr.WriteStatus(w, r, codes.InvalidArgument, "invalid transaction id")
		return
	}

	tx, err := h.client.GetTransaction(r.Context(), &ledgerv1.GetTransactionRequest{TransactionId: txID})
	if err != nil {
		respondProtoJSON(w, r, nil, err)
		return
	}

	owned, err := h.ownsAnyAccount(r.Context(), customerID, tx.GetFromAccountId(), tx.GetToAccountId())
	if err != nil {
		respondProtoJSON(w, r, nil, err)
		return
	}
	if !owned {
		apierr.WriteStatus(w, r, codes.NotFound, "transaction not found")
		return
	}

	respondProtoJSON(w, r, tx, nil)
}

// ownsAnyAccount — true, если клиенту принадлежит хотя бы один из счетов.
// Пустой/нулевой id (у пополнения нет счёта-отправителя) и счета, которых
// account-service не знает (NotFound), считаются "не его".
func (h *Ledger) ownsAnyAccount(ctx context.Context, customerID string, accountIDs ...string) (bool, error) {
	for _, id := range accountIDs {
		if id == "" || id == uuid.Nil.String() {
			continue
		}
		owned, err := checkAccountOwnership(ctx, h.accountClient, customerID, id)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			return false, err
		}
		if owned {
			return true, nil
		}
	}
	return false, nil
}

// parseTimeQuery читает необязательный RFC3339-параметр. Пустой -> (nil, true).
func parseTimeQuery(w http.ResponseWriter, r *http.Request, name string) (*timestamppb.Timestamp, bool) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		apierr.WriteStatus(w, r, codes.InvalidArgument, name+" must be an RFC3339 timestamp, e.g. 2026-11-01T00:00:00Z")
		return nil, false
	}
	return timestamppb.New(t), true
}