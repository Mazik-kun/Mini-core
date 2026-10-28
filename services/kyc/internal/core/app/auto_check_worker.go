package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/kyc/internal/core/domain"
)

type AutoCheckWorker struct {
	repo         domain.ApplicationRepository
	checkRepo    domain.CheckRepository
	kafka        domain.EventPublisher
	kafkaTopic   string
	interval     time.Duration
	batchSize    int
	maxAttempts  int
	minAge       int
	log          *slog.Logger
}

func NewAutoCheckWorker(
	repo domain.ApplicationRepository,
	checkRepo domain.CheckRepository,
	kafka domain.EventPublisher,
	topic string,
	interval time.Duration,
	batchSize, maxAttempts, minAge int,
	log *slog.Logger,
) *AutoCheckWorker {
	return &AutoCheckWorker{
		repo: repo, checkRepo: checkRepo, kafka: kafka,
		kafkaTopic: topic, interval: interval, batchSize: batchSize,
		maxAttempts: maxAttempts, minAge: minAge, log: log,
	}
}

func (w *AutoCheckWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.log.Info("auto-check worker started", "interval", w.interval)
	for {
		select {
		case <-ctx.Done():
			w.log.Info("auto-check worker stopped")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *AutoCheckWorker) processBatch(ctx context.Context) {
	apps, err := w.checkRepo.ListForAutoChecks(ctx, w.batchSize)
	if err != nil {
		w.log.Error("failed to list applications for auto checks", "err", err)
		return
	}
	if len(apps) == 0 {
		return
	}

	sanctions, err := w.checkRepo.ListSanctions(ctx)
	if err != nil {
		w.log.Error("failed to load sanctions", "err", err)
		return
	}

	for _, app := range apps {
		w.processSingle(ctx, app, sanctions)
	}
}

func (w *AutoCheckWorker) processSingle(ctx context.Context, app domain.Application, sanctions []domain.SanctionsEntry) {
	// 1. Проверка по санкциям
	sanctionsRes := domain.SanctionsCheck(app.FullNameSnapshot, app.BirthDateSnapshot, sanctions)
	_ = w.checkRepo.SaveCheck(ctx, app.ID, "SANCTIONS", sanctionsRes)

	// 2. Проверка возраста
	ageRes := domain.AgeCheck(app.BirthDateSnapshot, w.minAge)
	_ = w.checkRepo.SaveCheck(ctx, app.ID, "AGE", ageRes)

	// 3. Определяем итоговый статус
	var nextStatus domain.Status
	var reason string

	if !sanctionsRes.Passed || !ageRes.Passed {
		nextStatus = domain.StatusAutoRejected
		if !sanctionsRes.Passed {
			reason = sanctionsRes.Details
		} else {
			reason = ageRes.Details
		}
	} else {
		nextStatus = domain.StatusInReview
	}

	// 4. Обновляем статус
	// (В аудите пишем, что это сделала система, actorID = uuid.Nil)
	err := w.repo.UpdateStatus(ctx, app.ID, domain.StatusAutoChecks, nextStatus, uuid.Nil, "SYSTEM", reason)
	if err != nil {
		w.log.Error("failed to update status after checks", "app_id", app.ID, "err", err)
		return
	}

	// 5. Публикуем событие, если отклонили автоматически
	if nextStatus == domain.StatusAutoRejected {
		_ = w.kafka.Publish(ctx, w.kafkaTopic, "kyc.application_rejected", map[string]interface{}{
			"application_id": app.ID,
			"customer_id":    app.CustomerID,
			"reason":         reason,
		})
	}
}