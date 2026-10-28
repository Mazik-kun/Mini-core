package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Application struct {
	ID               uuid.UUID
	CustomerID       uuid.UUID
	Status           Status
	OfficerID        *uuid.UUID
	FullNameSnapshot string
	BirthDateSnapshot time.Time
	Attempts         int
	Reason           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DecidedAt        *time.Time
}

type CheckResult struct {
	Passed  bool
	Details string
}

// SanctionsCheck проверяет ФИО и дату рождения по списку.
func SanctionsCheck(fullName string, birthDate time.Time, sanctions []SanctionsEntry) CheckResult {
	normName := NormalizeName(fullName)
	for _, s := range sanctions {
		if s.FullNameNormalized == normName && s.BirthDate.Equal(birthDate) {
			return CheckResult{Passed: false, Details: "Matched sanctions list: " + s.Reason}
		}
	}
	return CheckResult{Passed: true, Details: "No sanctions match"}
}

// AgeCheck проверяет, что клиенту больше MinAge лет.
func AgeCheck(birthDate time.Time, minAge int) CheckResult {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		age--
	}
	if age < minAge {
		return CheckResult{Passed: false, Details: "Underage"}
	}
	return CheckResult{Passed: true, Details: "Age OK"}
}

func NormalizeName(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

type SanctionsEntry struct {
	FullNameNormalized string
	BirthDate          time.Time
	Reason             string
}