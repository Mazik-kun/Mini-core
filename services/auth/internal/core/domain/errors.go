package domain

import "errors"

var (
	ErrInvalidEmail         = errors.New("invalid email")
	ErrWeakPassword         = errors.New("weak password")
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrInvalidToken         = errors.New("invalid token")
	ErrTokenExpired         = errors.New("token expired")
	ErrRefreshTokenNotFound = errors.New("token not found")
	ErrPermissionDenied     = errors.New("permission denied")
)