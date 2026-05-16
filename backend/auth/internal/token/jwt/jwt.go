package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"

	"encoding/base64"
	"encoding/json"

	"backend/auth/internal/config"
	"backend/auth/internal/model"
)

type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type JWTGenerator struct {
	SecretKey string
}

func NewGenerator(config config.Token) JWTGenerator {
	return JWTGenerator{
		SecretKey: config.SecretKey,
	}
}

func generateHMAC(msg []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(msg)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func (g JWTGenerator) Generate(user model.UserClaims) (string, error) {
	headerJson, err := json.Marshal(Header{
		Alg: "HS256",
		Typ: "JWT",
	})
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}

	payloadJson, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	header := base64.RawURLEncoding.EncodeToString(headerJson)
	payload := base64.RawURLEncoding.EncodeToString(payloadJson)

	to_sign := string(header) + "." + string(payload)

	signature := generateHMAC([]byte(to_sign), g.SecretKey)

	return to_sign + "." + signature, nil
}
