package app

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakeCustomerLinker struct {
	updatedUserID     uuid.UUID
	updatedCustomerID uuid.UUID
	callCount         int
	err               error
}

func (f *fakeCustomerLinker) UpdateCustomerID(ctx context.Context, userID, customerID uuid.UUID) error {
	f.updatedUserID = userID
	f.updatedCustomerID = customerID
	f.callCount++
	return f.err
}

func TestApplyCustomerCreated_Success(t *testing.T) {
	linker := &fakeCustomerLinker{}
	uc := NewApplyCustomerCreatedUseCase(linker)

	userID := uuid.Must(uuid.NewV7())
	customerID := uuid.Must(uuid.NewV7())

	err := uc.ApplyCustomerCreated(context.Background(), ApplyCustomerCreatedInput{
		UserID:     userID,
		CustomerID: customerID,
	})
	if err != nil {
		t.Fatalf("ApplyCustomerCreated: %v", err)
	}
	if linker.callCount != 1 {
		t.Errorf("ожидался 1 вызов, получили %d", linker.callCount)
	}
	if linker.updatedUserID != userID || linker.updatedCustomerID != customerID {
		t.Errorf("неверные аргументы: user_id=%s customer_id=%s",
			linker.updatedUserID, linker.updatedCustomerID)
	}
}

func TestApplyCustomerCreated_EmptyUserID(t *testing.T) {
	linker := &fakeCustomerLinker{}
	uc := NewApplyCustomerCreatedUseCase(linker)

	err := uc.ApplyCustomerCreated(context.Background(), ApplyCustomerCreatedInput{
		CustomerID: uuid.Must(uuid.NewV7()),
	})
	if err == nil {
		t.Fatalf("ожидалась ошибка при пустом user_id")
	}
	if linker.callCount != 0 {
		t.Errorf("репозиторий не должен вызываться")
	}
}

func TestApplyCustomerCreated_EmptyCustomerID(t *testing.T) {
	linker := &fakeCustomerLinker{}
	uc := NewApplyCustomerCreatedUseCase(linker)

	err := uc.ApplyCustomerCreated(context.Background(), ApplyCustomerCreatedInput{
		UserID: uuid.Must(uuid.NewV7()),
	})
	if err == nil {
		t.Fatalf("ожидалась ошибка при пустом customer_id")
	}
	if linker.callCount != 0 {
		t.Errorf("репозиторий не должен вызываться")
	}
}