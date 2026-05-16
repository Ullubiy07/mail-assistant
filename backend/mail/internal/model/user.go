package model

import "github.com/google/uuid"

type UserClaims struct {
	Sub       uuid.UUID `json:"sub"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IssuedAt  int64     `json:"iat"`
	ExpiredAt int64     `json:"exp"`
}
