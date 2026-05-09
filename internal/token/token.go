package token

import (
	"errors"
	"mail-assistant/internal/model"
)

var (
	ErrTokenExpired     = errors.New("token has expired")
	ErrInvalidSignature = errors.New("signature mismatch")
)

type Generator interface {
	Generate(user model.UserClaims) (string, error)
}

type Verifier interface {
	Verify(token string) error
}

type Extractor interface {
	Extract(token string) (model.UserClaims, error)
}
