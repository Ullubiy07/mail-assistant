package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"strings"

	"encoding/base64"
	"encoding/json"

	"mail-assistant/internal/config"
	"mail-assistant/internal/model"
)

type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type JWT struct {
	SecretKey string
}

func New(cfg *config.Token) JWT {
	return JWT{
		SecretKey: cfg.SecretKey,
	}
}

func (t JWT) generateHMAC(msg []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(msg)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func (t JWT) Generate(user *model.UserToken) string {
	headerJson, _ := json.Marshal(Header{
		Alg: "HS256",
		Typ: "JWT",
	})
	payloadJson, _ := json.Marshal(user)

	header := base64.RawURLEncoding.EncodeToString(headerJson)
	payload := base64.RawURLEncoding.EncodeToString(payloadJson)

	to_sign := string(header) + "." + string(payload)

	signature := t.generateHMAC([]byte(to_sign), t.SecretKey)

	return to_sign + "." + signature
}

func (t JWT) Verify(token string) bool {
	var section []string = strings.Split(token, ".")

	to_sign := section[0] + "." + section[1]
	signature := t.generateHMAC([]byte(to_sign), t.SecretKey)

	if signature != section[2] {
		return false
	}
	return true
}
