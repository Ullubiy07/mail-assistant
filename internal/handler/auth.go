package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"mail-assistant/internal/model"
	"mail-assistant/internal/storage"
	"mail-assistant/internal/token"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	storage   storage.UserStorage
	generator token.Generator
}

func NewAuthHandler(storage storage.UserStorage, generator token.Generator) AuthHandler {
	return AuthHandler{
		storage:   storage,
		generator: generator,
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

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("generate hashed password", "error", err)
		sendResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}
	user.Password = string(hash)

	if err := h.storage.CreateUser(r.Context(), user); err != nil {
		if errors.Is(err, storage.ErrDublicateUser) {
			sendResponse(w, http.StatusBadRequest, err.Error())
		} else {
			slog.Error("create user", "error", err)
			sendResponse(w, http.StatusInternalServerError, "internal server error")
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
	} else if bcrypt.CompareHashAndPassword([]byte(userDB.Password), []byte(user.Password)) != nil {
		sendResponse(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	now := time.Now().Unix()

	jwt := h.generator.Generate(&model.UserToken{
		ID:        userDB.ID,
		Username:  userDB.Username,
		Email:     userDB.Email,
		IssuedAt:  now,
		ExpiredAt: now + 900,
	})

	json.NewEncoder(w).Encode(LoginResponse{Token: jwt})
}
