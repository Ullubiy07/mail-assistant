package token

import "backend/auth/internal/model"

type Generator interface {
	Generate(user model.UserClaims) (string, error)
}
