package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"mail-assistant/internal/config"
	"mail-assistant/internal/model"
	"mail-assistant/internal/storage"
	"mail-assistant/internal/token"
	"mail-assistant/internal/token/jwt"
)

type AuthHandler struct {
	storage storage.UserStorer
	token   token.TokenProducer

	cfg *config.Token
}

func NewAuthHandler(storage storage.UserStorer, cfg *config.Token) AuthHandler {
	return AuthHandler{
		storage: storage,
		token:   jwt.New(cfg),
	}
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (h AuthHandler) Register(w http.ResponseWriter, r *http.Request) {

	user := model.UserRegister{}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		sendResponse(w, http.StatusBadRequest, "wrong request structure")
		return
	}

	if err := h.storage.CreateUser(r.Context(), user); err != nil {
		if errors.Is(err, storage.ErrDublicateUser) {
			sendResponse(w, http.StatusBadRequest, err.Error())
		} else {
			sendResponse(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	sendResponse(w, http.StatusOK, "user created successfully")
}

func (h AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	user := model.UserLogin{}
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		sendResponse(w, http.StatusBadRequest, "wrong request structure")
		return
	}

	userDB, err := h.storage.FindUserByUsername(r.Context(), user.Username)

	if err != nil {
		sendResponse(w, http.StatusNotFound, "invalid username or password")
		return
	} else if userDB.Password != user.Password {
		sendResponse(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	now := time.Now().Unix()

	jwt := h.token.Generate(&model.UserToken{
		ID:        userDB.ID,
		Username:  userDB.Username,
		Email:     userDB.Email,
		IssuedAt:  now,
		ExpiredAt: now + 900,
	})

	json.NewEncoder(w).Encode(LoginResponse{Token: jwt})
}
