package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"mail-assistant/internal/client/embed"
	"mail-assistant/internal/client/mail"
	"mail-assistant/internal/storage"

	"github.com/google/uuid"
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
	Folders  []string `json:"folders"`
	Question string   `json:"question"`
}

type QuestionResponse struct {
	Message string `json:"msg"`
}

func (h MailHandler) Question(w http.ResponseWriter, r *http.Request) {
	if _, ok := r.Context().Value("user_id").(uuid.UUID); !ok {
		slog.Error("no user_id provided in context")
		sendResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}

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
	
	for _, item := range score {
		fmt.Println("Score: ", item)
		fmt.Println(item.Payload.Envelope.Subject)
		fmt.Println(item.Payload.Body)
	}

	answer, err := h.generation(score)
	if err != nil {
		return "", fmt.Errorf("stage - generation: %w", err)
	}
	return answer, nil
}

func (h MailHandler) dataIngestion(ctx context.Context, req QuestionRequst) error {
	// fetch data
	auth := mail.Auth{
		Email:    req.Email,
		Password: req.Password,
		Token:    "",
		Address:  req.Address,
		Method:   "PLAIN",
	}
	userID := ctx.Value("user_id").(uuid.UUID)
	fetcher := h.factory.NewFetcher(auth)

	folders, err := fetcher.FetchFolders(ctx)
	if err != nil {
		return fmt.Errorf("get folders from mailbox: %w", err)
	}

	letters, err := h.collectNewLetters(ctx, req, fetcher, folders)
	if err != nil {
		return fmt.Errorf("collect new letters: %w", err)
	}

	if len(letters) != 0 {
		mailbox := storage.Mailbox{UserID: userID, Email: req.Email, Folders: folders}
		if err := h.storage.UpdateMailbox(ctx, mailbox); err != nil {
			return fmt.Errorf("update mailbox: %w", err)
		}
	}

	// chunking
	chunks := make([]string, len(letters))
	for i, item := range letters {
		chunks[i] = item.Body
	}

	// embeddings
	vectors, err := h.model.Embed(ctx, chunks)
	if err != nil {
		return fmt.Errorf("embed letters chunks: %w", err)
	}

	// store in vector storage
	var points []storage.Point
	for i := range letters {
		points = append(points, storage.Point{Embedding: vectors[i], Payload: &letters[i]})
	}

	if err = h.vector.CreateCollection(ctx, userID.String()); err != nil {
		return fmt.Errorf("create collection for chunks: %w", err)
	}

	if err = h.vector.Insert(ctx, userID.String(), points); err != nil {
		return fmt.Errorf("upsert points to collection: %w", err)
	}
	return nil
}

func (h MailHandler) retrieval(ctx context.Context, req QuestionRequst) ([]storage.ScoredPoint, error) {
	questionVector, err := h.model.Embed(ctx, []string{req.Question})
	if err != nil {
		return nil, fmt.Errorf("embed question chunk")
	}

	userID := ctx.Value("user_id").(uuid.UUID)
	score, err := h.vector.Search(ctx, userID.String(), questionVector[0])
	if err != nil {
		return nil, fmt.Errorf("search question embedding in vector storage")
	}
	return score, nil
}

func (h MailHandler) generation(score []storage.ScoredPoint) (string, error) {
	return "", nil
}

func (h MailHandler) collectNewLetters(ctx context.Context, req QuestionRequst, fetcher mail.Fetcher, folders []mail.Folder) ([]mail.Letter, error) {
	userID := ctx.Value("user_id").(uuid.UUID)

	foldersDB, err := h.storage.GetFolders(ctx, userID, req.Email)
	if err != nil && !errors.Is(err, storage.ErrNotFoundFolders) {
		return nil, fmt.Errorf("get folders from storage: %w", err)
	}

	foldersOldMap := make(map[string]mail.Folder)
	foldersNewMap := make(map[string]mail.Folder)
	for _, item := range foldersDB {
		foldersOldMap[item.Name] = item
	}
	for _, item := range folders {
		foldersNewMap[item.Name] = item
	}

	var letters []mail.Letter
	for _, folder := range req.Folders {
		old, oldExist := foldersOldMap[folder]
		new, newExist := foldersNewMap[folder]

		if !newExist {
			continue
		}

		if !oldExist || old.UIDValidity != new.UIDValidity {
			res, err := fetcher.FetchNewLetters(ctx, folder, 1)
			if err != nil {
				return nil, fmt.Errorf("fetch new letters (all): %w", err)
			}
			letters = append(letters, res...)
		} else if old.UIDValidity == new.UIDValidity && old.UIDNext != new.UIDNext {
			res, err := fetcher.FetchNewLetters(ctx, folder, old.UIDNext)
			if err != nil {
				return nil, fmt.Errorf("fetch new letters (new): %w", err)
			}
			letters = append(letters, res...)
		}
	}
	return letters, nil
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
