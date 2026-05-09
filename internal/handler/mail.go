package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"mail-assistant/internal/client/embed"
	"mail-assistant/internal/client/mail"
	"mail-assistant/internal/storage"
)

type MailHandler struct {
	storage storage.MailStorage
	vector  storage.VectorStorage
	factory mail.FetcherFactory
	model   embed.Embedder
}

func NewMailHandler(storage storage.MailStorage, vector storage.VectorStorage, factory mail.FetcherFactory, model embed.Embedder) MailHandler {
	return MailHandler{
		storage: storage,
		vector:  vector,
		factory: factory,
		model:   model,
	}
}

type QuestionRequst struct {
	Address  string   `json:"address"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Folders  []string `json:"folder"`
	Question string   `json:"question"`
}

type QuestionResponse struct {
	Message string `json:"msg"`
}

func (h MailHandler) Question(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	req := QuestionRequst{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Wrong request structure")
		return
	}

	_, err := h.runRAGPipeline(r.Context(), req)
	if err != nil {
		slog.Error("RAG pipeline", "error", err)
		sendResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}

	// for _, item := range score {
	// 	fmt.Println("Score: ", item.Score)
	// 	fmt.Println("Subject: ", item.Payload.Envelope.Subject)
	// 	fmt.Println(item.Payload.Body)
	// 	fmt.Println()
	// 	fmt.Println()
	// }

	// resp := QuestionResponse{}
	// rawResp, err := json.Marshal(resp)
	// if err != nil {
	// 	slog.Error("marshal question response", "error", err)
	// 	sendResponse(w, http.StatusInternalServerError, "internal error")
	// 	return
	// }

	// w.Write(rawResp)
}

func (h MailHandler) runRAGPipeline(ctx context.Context, req QuestionRequst) (string, error) {
	if err := h.dataIngestion(ctx, req); err != nil {
		return "", fmt.Errorf("stage - data ingestion: %w", err)
	}
	score, err := h.retrieval(ctx, req)
	if err != nil {
		return "", fmt.Errorf("stage - retrieval: %w", err)
	}
	answer, err := h.generation(score)
	if err != nil {
		return "", fmt.Errorf("stage - generation: %w", err)
	}
	return answer, nil
}

func (h MailHandler) dataIngestion(ctx context.Context, req QuestionRequst) error {
	// fetch data
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

	// for _, folder := range req.Folders {}
	letters, err := fetcher.FetchNewLetters(ctx, "INBOX", 5000)
	if err != nil {
		return fmt.Errorf("fetch new letters: %w", err)
	}

	// chunking
	chunks := make([]string, len(letters))
	for i, item := range letters {
		chunks[i] = item.Body
	}

	// embeddings
	vectors, err := h.model.Embed(ctx, chunks)
	if err != nil {
		return fmt.Errorf("embed letters chunks")
	}

	// store in vector storage
	var points []storage.Point
	for i := range letters {
		points = append(points, storage.Point{Embedding: vectors[i], Payload: &letters[i]})
	}

	if err = h.vector.CreateCollection(ctx, "letters"); err != nil {
		return fmt.Errorf("create collection for chunks: %w", err)
	}

	if err = h.vector.Upsert(ctx, "letters", points); err != nil {
		return fmt.Errorf("upsert points to collection: %w", err)
	}
	return nil
}

func (h MailHandler) retrieval(ctx context.Context, req QuestionRequst) ([]storage.ScoredPoint, error) {
	questionVector, err := h.model.Embed(ctx, []string{req.Question})
	if err != nil {
		return nil, fmt.Errorf("embed question chunk")
	}

	score, err := h.vector.Search(ctx, "letters", questionVector[0])
	if err != nil {
		return nil, fmt.Errorf("search question embedding in vector storage")
	}
	return score, nil
}

func (h MailHandler) generation(score []storage.ScoredPoint) (string, error) {
	return "", nil
}

func sendResponse(w http.ResponseWriter, statusCode int, msg string) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Add("Content-Type", "application/json")
	}
	w.WriteHeader(statusCode)

	raw, _ := json.Marshal(struct {
		Msg string `json:"message"`
	}{Msg: msg})

	w.Write(raw)
}
