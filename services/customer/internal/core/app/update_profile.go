package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/pkg/roles"
	"github.com/Mazik-kun/mini-core/services/customer/internal/core/domain"
)

type UpdateProfileUseCase struct {
	repo domain.CustomerRepository
}

func NewUpdateProfileUseCase(repo domain.CustomerRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{repo: repo}
}

type UpdateProfileInput struct {
	CustomerID  uuid.UUID
	FullName    string
	BirthDate   pgtype.Date
	Address     string
	PhoneNumber string
	Citizenship string
}

func (uc *UpdateProfileUseCase) UpdateProfile(ctx context.Context, in UpdateProfileInput) (domain.Customer, error) {
	if err := authjwt.RequireCustomerAccess(ctx, in.CustomerID, roles.RoleOfficer, roles.RoleAdmin); err != nil {
		return domain.Customer{}, domain.ErrAccessDenied
	}

	if err := validateAddress(in.Address); err != nil {
		return domain.Customer{}, err
	}
	if err := validatePhoneNumber(in.PhoneNumber); err != nil {
		return domain.Customer{}, err
	}
	if err := validateBirthDate(in.BirthDate); err != nil {
		return domain.Customer{}, err
	}
	if err := validateFullName(in.FullName); err != nil {
		return domain.Customer{}, err
	}
	if err := validateCitizenship(in.Citizenship); err != nil {
		return domain.Customer{}, err
	}

	customer, err := uc.repo.GetByID(ctx, in.CustomerID)
	if err != nil {
		return domain.Customer{}, err
	}

	if !customer.Status.CanEditProfile() {
		return domain.Customer{}, domain.ErrProfileLocked
	}

	customer.Address = in.Address
	customer.BirthDate = in.BirthDate
	customer.PhoneNumber = in.PhoneNumber
	customer.Citizenship = in.Citizenship
	customer.FullName = in.FullName

	if customer.Status.CanTransitionTo(domain.StatusProfileFilled) {
		customer.Status = domain.StatusProfileFilled
	}

	return uc.repo.Update(ctx, customer)
}
