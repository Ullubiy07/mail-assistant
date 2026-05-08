package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"mail-assistant/internal/client/embed"
	"mail-assistant/internal/client/mail"
	"mail-assistant/internal/storage"
	"mail-assistant/internal/token"
)

type MailHandler struct {
	storage  storage.MailStorage
	vector   storage.VectorStorage
	factory  mail.FetcherFactory
	model    embed.Embedder
	verifier token.Verifier
}

func NewMailHandler(storage storage.MailStorage, vector storage.VectorStorage, factory mail.FetcherFactory, model embed.Embedder, verifier token.Verifier) MailHandler {
	return MailHandler{
		storage:  storage,
		vector:   vector,
		factory:  factory,
		model:    model,
		verifier: verifier,
	}
}

type QuestionRequst struct {
	Address  string   `json:"address"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Folders  []string `json:"folder"`
	Question string   `json:"question"`
	UID      uint32   `json:"uid"`
}

type QuestionResponse struct {
	Message string `json:"msg"`
}

func (h MailHandler) Question(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	req := QuestionRequst{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "wrong request structure")
		return
	}

	creds := mail.Creds{
		Email:    req.Email,
		Password: req.Password,
		Token:    "",
	}
	auth := mail.Auth{
		Address: req.Address,
		Method:  "PLAIN",
	}
	fetcher := h.factory.NewFetcher(creds, auth)

	letter, err := fetcher.FetchNewLetters(r.Context(), "INBOX", req.UID)
	if err != nil {
		slog.Error("fetch new letters", "error", err)
		sendResponse(w, http.StatusInternalServerError, "internal error")
		return
	}

	var chunks []string
	for _, item := range letter {
		chunks = append(chunks, item.Body)
	}

	questionEmbedding, err := h.model.Embed(r.Context(), []string{req.Question})
	if err != nil {
		slog.Error("embed question chunk", "error", err)
		sendResponse(w, http.StatusInternalServerError, "internal error")
		return
	}

	embeddings, err := h.model.Embed(r.Context(), chunks)
	if err != nil {
		slog.Error("embed letters chunks", "error", err)
		sendResponse(w, http.StatusInternalServerError, "internal error")
		return
	}

	var points []storage.Point
	for i := range letter {
		points = append(points, storage.Point{Embedding: embeddings[i], Payload: &letter[i]})
	}

	if err = h.vector.CreateCollection(r.Context(), "letters"); err != nil {
		slog.Error("create collection", "error", err)
		sendResponse(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err = h.vector.Upsert(r.Context(), "letters", points); err != nil {
		slog.Error("upsert points", "error", err)
		sendResponse(w, http.StatusInternalServerError, "internal error")
		return
	}

	score, err := h.vector.Search(r.Context(), "letters", questionEmbedding[0])
	if err != nil {
		slog.Error("search question embedding in vector storage", "error", err)
		sendResponse(w, http.StatusInternalServerError, "internal error")
		return
	}

	for _, item := range score {
		fmt.Println("Score: ", item.Score)
		fmt.Println("Subject: ", item.Payload.Envelope.Subject)
		fmt.Println(item.Payload.Body)
		fmt.Println()
		fmt.Println()
	}

	// resp := QuestionResponse{}
	// rawResp, err := json.Marshal(resp)
	// if err != nil {
	// 	slog.Error("marshal question response", "error", err)
	// 	sendResponse(w, http.StatusInternalServerError, "internal error")
	// 	return
	// }

	// w.Write(rawResp)
}

func sendResponse(w http.ResponseWriter, statusCode int, msg string) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Add("Content-Type", "application/json")
	}
	w.WriteHeader(statusCode)

	raw, _ := json.Marshal(struct {
		Msg string `json:"msg"`
	}{Msg: msg})

	w.Write(raw)
}
