package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"encoding/base64"
	"encoding/json"

	"mail-assistant/internal/config"
	"mail-assistant/internal/model"
	tokengen "mail-assistant/internal/token"
)

type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type JWTGenerator struct {
	SecretKey string
}

type JWTVerifier struct {
	SecretKey string
}

type JWTExtractor struct{}

func NewGenerator(config config.Token) JWTGenerator {
	return JWTGenerator{
		SecretKey: config.SecretKey,
	}
}

func NewVerifier(config config.Token) JWTVerifier {
	return JWTVerifier{
		SecretKey: config.SecretKey,
	}
}

func NewExtractor() JWTExtractor {
	return JWTExtractor{}
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

func (v JWTVerifier) Verify(token string) error {
	var section []string = strings.Split(token, ".")

	to_sign := section[0] + "." + section[1]
	user := model.UserClaims{}

	signature := generateHMAC([]byte(to_sign), v.SecretKey)
	if signature != section[2] {
		return tokengen.ErrInvalidSignature
	}

	payload, err := base64.RawURLEncoding.DecodeString(section[1])
	if err != nil {
		return fmt.Errorf("base64 decode payload: %w", err)
	}
	if err = json.Unmarshal(payload, &user); err != nil {
		return fmt.Errorf("unmarshal payload into user: %w", err)
	}

	if time.Now().After(time.Unix(user.ExpiredAt, 0)) {
		return tokengen.ErrTokenExpired
	}

	return nil
}

func (e JWTExtractor) Extract(token string) (model.UserClaims, error) {
	var section []string = strings.Split(token, ".")
	user := model.UserClaims{}

	payload, err := base64.RawURLEncoding.DecodeString(section[1])
	if err != nil {
		return user, fmt.Errorf("base64 decode payload: %w", err)
	}
	if err = json.Unmarshal(payload, &user); err != nil {
		return user, fmt.Errorf("unmarshal payload into user: %w", err)
	}
	return user, nil
}
