package app

import (
	"context"
	"time"
	"uuid"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

type UpdateProfileUseCase struct {
	repo domain.CustomerRepository
}

func NewUpdateProfileUseCase(repo domain.CustomerRepository) *UpdateProfileUseCase{
	return &UpdateProfileUseCase{repo:repo}
}

type UpdateProfileInput struct{
	CustomerID uuid.UUID
	RequesterID uuid.UUID
	RequesterRole string
	FullName string
	BirthDate time.Time
	Address string
	PhoneNumber string
	Citizenship string
}

func(uc *UpdateProfileUseCase)UpdateProfile(ctx context.Context, in UpdateProfileInput)(domain.Customer, error){
	if in.RequesterRole == domain.RoleClient && in.CustomerID != in.RequesterID {
		return domain.Customer{}, domain.ErrAccessDenied
	}
	err := validateAddress(in.Address)
	if err != nil{
		return domain.Customer{}, err
	}
	err = validatePhoneNumber(in.PhoneNumber)
	if err != nil{
		return domain.Customer{}, err
	}
	err = validateBirthDate(in.BirthDate)
	if err != nil{
		return domain.Customer{}, err
	}
	err = validateFullName(in.FullName)
	if err != nil{
		return domain.Customer{}, err
	}
	err = validateCitizenship(in.Citizenship)
	if err != nil{
		return domain.Customer{}, err
	}
	
	
	customer, err := uc.repo.GetByID(ctx, in.CustomerID)
	
	if err!= nil{
		return domain.Customer{}, err 
	}

	if !customer.Status.CanEditProfile(){
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

	customer, err = uc.repo.Update(ctx,customer)
	if err != nil{
		return domain.Customer{}, err
	}	
	return customer, nil
}