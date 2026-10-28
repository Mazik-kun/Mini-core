package domain

import "errors"

type Status string

const (
	StatusDraft         Status = "DRAFT"
	StatusSubmitted     Status = "SUBMITTED"
	StatusAutoChecks    Status = "AUTO_CHECKS"
	StatusInReview      Status = "IN_REVIEW"
	StatusApproved      Status = "APPROVED"
	StatusRejected      Status = "REJECTED"
	StatusAutoRejected  Status = "AUTO_REJECTED"
	StatusDocsRequested Status = "DOCS_REQUESTED"
)

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrApplicationNotFound     = errors.New("application not found")
	ErrActiveApplicationExists = errors.New("active application already exists")
	ErrAccessDenied            = errors.New("access denied")
	ErrProfileNotFilled        = errors.New("customer profile is not filled")
	ErrDocumentsMissing        = errors.New("required documents are missing")
)

var allowedTransitions = map[Status][]Status{
	StatusDraft:         {StatusSubmitted},
	StatusSubmitted:     {StatusAutoChecks},
	StatusAutoChecks:    {StatusInReview, StatusAutoRejected},
	StatusInReview:      {StatusApproved, StatusRejected, StatusDocsRequested},
	StatusDocsRequested: {StatusSubmitted},
	// APPROVED, REJECTED, AUTO_REJECTED - терминальные
}

func (s Status) CanTransitionTo(next Status) bool {
	for _, a := range allowedTransitions[s] {
		if a == next {
			return true
		}
	}
	return false
}