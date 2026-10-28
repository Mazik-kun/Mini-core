package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Mazik-kun/mini-core/services/kyc/internal/adapters/outbound/postgres/db"
	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/domain"
)

type KycRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewKycRepository(pool *pgxpool.Pool) *KycRepository {
	return &KycRepository{pool: pool, q: db.New(pool)}
}


func (r *KycRepository) Create(ctx context.Context, app domain.Application, actorID uuid.UUID, actorRole string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)

	_, err = qtx.CreateApplication(ctx, db.CreateApplicationParams{
		ID:                app.ID,
		CustomerID:        app.CustomerID,
		Status:            string(domain.StatusSubmitted),
		FullNameSnapshot:  app.FullNameSnapshot,
		BirthDateSnapshot: pgtype.Date{Time: app.BirthDateSnapshot, Valid: true},
	})
	if err != nil {
		return err
	}

	_, err = qtx.UpdateApplicationStatus(ctx, db.UpdateApplicationStatusParams{
		ID:     app.ID,
		Status: string(domain.StatusAutoChecks),
		Status_2: string(domain.StatusSubmitted),
	})
	if err != nil {
		return err
	}

	// Пишем события в аудит (kyc_events)
	for _, s := range []struct{ from, to string }{
		{"", string(domain.StatusSubmitted)},
		{string(domain.StatusSubmitted), string(domain.StatusAutoChecks)},
	} {
		_, _ = qtx.InsertEvent(ctx, db.InsertEventParams{
			ID:            uuid.Must(uuid.NewV7()),
			ApplicationID: app.ID,
			ActorID:       pgtype.UUID{Bytes: actorID, Valid: actorID != uuid.Nil},
			ActorRole:     pgtype.Text{String: actorRole, Valid: actorRole != ""},
			FromStatus:    pgtype.Text{String: s.from, Valid: s.from != ""},
			ToStatus:      pgtype.Text{String: s.to},
		})
	}

	return tx.Commit(ctx)
}

func (r *KycRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Application, error) {
	row, err := r.q.GetApplicationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Application{}, domain.ErrApplicationNotFound
		}
		return domain.Application{}, err
	}
	return toDomain(row), nil
}

func (r *KycRepository) GetActiveByCustomerID(ctx context.Context, customerID uuid.UUID) (domain.Application, error) {
	row, err := r.q.GetActiveByCustomerID(ctx, customerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Application{}, domain.ErrApplicationNotFound
		}
		return domain.Application{}, err
	}
	return toDomain(row), nil
}

func (r *KycRepository) Claim(ctx context.Context, appID, officerID uuid.UUID) error {
	_, err := r.q.ClaimApplication(ctx, db.ClaimApplicationParams{
		ID:        appID,
		OfficerID: pgtype.UUID{Bytes: officerID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAccessDenied // Уже занята другим офицером
		}
		return err
	}
	return nil
}

func (r *KycRepository) UpdateStatus(ctx context.Context, appID uuid.UUID, from, to domain.Status, actorID uuid.UUID, actorRole, comment string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)

	_, err = qtx.UpdateApplicationStatus(ctx, db.UpdateApplicationStatusParams{
		ID:       appID,
		Status:   string(to),
		Reason:   pgtype.Text{String: comment, Valid: comment != ""},
		Status_2: string(from), 
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInvalidStatusTransition
		}
		return err
	}

	_, _ = qtx.InsertEvent(ctx, db.InsertEventParams{
		ID:            uuid.Must(uuid.NewV7()),
		ApplicationID: appID,
		ActorID:       pgtype.UUID{Bytes: actorID, Valid: actorID != uuid.Nil},
		ActorRole:     pgtype.Text{String: actorRole, Valid: actorRole != ""},
		FromStatus:    pgtype.Text{String: string(from), Valid: true},
		ToStatus:      pgtype.Text{String: string(to)},
		Comment:       pgtype.Text{String: comment, Valid: comment != ""},
	})

	return tx.Commit(ctx)
}

func (r *KycRepository) List(ctx context.Context, status *domain.Status, limit int, cursor uuid.UUID) ([]domain.Application, error) {
	params := db.ListApplicationsParams{Limit: int32(limit)}
	if status != nil {
		params.Status = pgtype.Text{String: string(*status), Valid: true}
	}
	if cursor != uuid.Nil {
		params.Cursor = pgtype.UUID{Bytes: cursor, Valid: true}
	}
	
	rows, err := r.q.ListApplications(ctx, params)
	if err != nil {
		return nil, err
	}
	
	out := make([]domain.Application, len(rows))
	for i, row := range rows {
		out[i] = toDomain(row)
	}
	return out, nil
}

// ==========================================
// CheckRepository
// ==========================================

func (r *KycRepository) SaveCheck(ctx context.Context, appID uuid.UUID, checkType string, res domain.CheckResult) error {
	status := "PASS"
	if !res.Passed {
		status = "FAIL"
	}
	
	_, err := r.q.SaveCheck(ctx, db.SaveCheckParams{
		ID:            uuid.Must(uuid.NewV7()),
		ApplicationID: appID,
		CheckType:     checkType,
		Status:        status,
		Details:       []byte(`{"details":"` + res.Details + `"}`),
	})
	return err
}

func (r *KycRepository) ListSanctions(ctx context.Context) ([]domain.SanctionsEntry, error) {
	rows, err := r.q.ListSanctions(ctx)
	if err != nil {
		return nil, err
	}
	
	out := make([]domain.SanctionsEntry, len(rows))
	for i, row := range rows {
		out[i] = domain.SanctionsEntry{
			FullNameNormalized: row.FullNameNormalized,
			BirthDate:          row.BirthDate.Time,
			Reason:             row.Reason,
		}
	}
	return out, nil
}

func (r *KycRepository) ListForAutoChecks(ctx context.Context, limit int) ([]domain.Application, error) {
	rows, err := r.q.ListForAutoChecks(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	
	out := make([]domain.Application, len(rows))
	for i, row := range rows {
		out[i] = toDomain(row)
	}
	return out, nil
}

func (r *KycRepository) IncrementAttempts(ctx context.Context, appID uuid.UUID, maxAttempts int) (bool, error) {
	attempts, err := r.q.IncrementAttempts(ctx, appID)
	if err != nil {
		return false, err
	}
	return attempts >= int32(maxAttempts), nil
}

// ==========================================
// Mappers
// ==========================================

// ВАЖНО: Если sqlc сгенерировал тип как db.Application вместо db.KycApplication,
// просто замени db.KycApplication на db.Application ниже.
func toDomain(row db.KycApplication) domain.Application {
	app := domain.Application{
		ID:                row.ID,
		CustomerID:        row.CustomerID,
		Status:            domain.Status(row.Status),
		FullNameSnapshot:  row.FullNameSnapshot,
		BirthDateSnapshot: row.BirthDateSnapshot.Time,
		Attempts:          int(row.Attempts),
		Reason:            row.Reason.String,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
	if row.OfficerID.Valid {
		id := uuid.UUID(row.OfficerID.Bytes)
		app.OfficerID = &id
	}
	if row.DecidedAt.Valid {
		t := row.DecidedAt.Time
		app.DecidedAt = &t
	}
	return app
}