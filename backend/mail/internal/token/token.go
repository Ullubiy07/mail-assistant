package token

import (
	"errors"

	"backend/mail/internal/model"
)

var (
	ErrTokenExpired     = errors.New("token has expired")
	ErrInvalidSignature = errors.New("signature mismatch")
)

type Verifier interface {
	Verify(token string) error
}

type Extractor interface {
	Extract(token string) (model.UserClaims, error)
}
