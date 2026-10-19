package app

import (
    "context"

    "github.com/Mazik-kun/mini-core/services/auth/internal/core/domain"
)

type ValidateTokenUseCase struct {
    verifier domain.TokenVerifier
}

func NewValidateTokenUseCase(verifier domain.TokenVerifier) *ValidateTokenUseCase {
    return &ValidateTokenUseCase{verifier: verifier}
}

type ValidateTokenInput struct {
    Token string
}

type ValidateTokenOutput struct {
    Claims domain.Claims
}

func (uc *ValidateTokenUseCase) ValidateToken(ctx context.Context, in ValidateTokenInput) (ValidateTokenOutput, error) {
    claims, err := uc.verifier.Verify(in.Token)
    if err != nil {
        return ValidateTokenOutput{}, err
    }
    return ValidateTokenOutput{Claims: claims}, nil
}
