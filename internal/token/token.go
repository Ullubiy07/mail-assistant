package token

import "mail-assistant/internal/model"

type TokenProducer interface {
	Generate(user *model.UserToken) string
	Verify(token string) bool
}
