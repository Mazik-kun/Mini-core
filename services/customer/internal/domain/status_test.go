package domain_test

import (
	"testing"

	"github.com/Mazik-kun/mini-core/services/customer/internal/domain"
)

func TestStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name   string
		from   domain.Status
		to     domain.Status
		want   bool
	}{
		// Разрешённые переходы
		{"NEW → PROFILE_FILLED", domain.StatusNew, domain.StatusProfileFilled, true},
		{"PROFILE_FILLED → ON_KYC", domain.StatusProfileFilled, domain.StatusOnKYC, true},
		{"ON_KYC → ACTIVE", domain.StatusOnKYC, domain.StatusActive, true},
		{"ON_KYC → REJECTED", domain.StatusOnKYC, domain.StatusRejected, true},
		{"ACTIVE → BLOCKED", domain.StatusActive, domain.StatusBlocked, true},
		{"BLOCKED → ACTIVE", domain.StatusBlocked, domain.StatusActive, true},

		// Запрещённые переходы
		{"NEW → ACTIVE", domain.StatusNew, domain.StatusActive, false},
		{"NEW → REJECTED", domain.StatusNew, domain.StatusRejected, false},
		{"NEW → BLOCKED", domain.StatusNew, domain.StatusBlocked, false},
		{"PROFILE_FILLED → ACTIVE", domain.StatusProfileFilled, domain.StatusActive, false},
		{"ON_KYC → PROFILE_FILLED", domain.StatusOnKYC, domain.StatusProfileFilled, false},
		{"REJECTED → ACTIVE", domain.StatusRejected, domain.StatusActive, false},
		{"BLOCKED → ON_KYC", domain.StatusBlocked, domain.StatusOnKYC, false},
		{"ACTIVE → NEW", domain.StatusActive, domain.StatusNew, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.from.CanTransitionTo(tt.to)
			if got != tt.want {
				t.Errorf("CanTransitionTo(%q → %q) = %v, want %v",
					tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestStatus_CanEditProfile(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		want   bool
	}{
		{"NEW — можно", domain.StatusNew, true},
		{"PROFILE_FILLED — можно", domain.StatusProfileFilled, true},
		{"ON_KYC — нельзя", domain.StatusOnKYC, false},
		{"ACTIVE — нельзя", domain.StatusActive, false},
		{"REJECTED — нельзя", domain.StatusRejected, false},
		{"BLOCKED — нельзя", domain.StatusBlocked, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.CanEditProfile()
			if got != tt.want {
				t.Errorf("CanEditProfile(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		want   bool
	}{
		{"NEW", domain.StatusNew, true},
		{"PROFILE_FILLED", domain.StatusProfileFilled, true},
		{"ON_KYC", domain.StatusOnKYC, true},
		{"ACTIVE", domain.StatusActive, true},
		{"REJECTED", domain.StatusRejected, true},
		{"BLOCKED", domain.StatusBlocked, true},
		{"пустая строка", domain.Status(""), false},
		{"мусор", domain.Status("FOO"), false},
		{"нижний регистр", domain.Status("active"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsValid()
			if got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}