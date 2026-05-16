package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"encoding/base64"
	"encoding/json"

	"backend/mail/internal/config"
	"backend/mail/internal/model"
	tokengen "backend/mail/internal/token"
)

type JWTVerifier struct {
	SecretKey string
}

type JWTExtractor struct{}

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
