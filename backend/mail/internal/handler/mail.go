package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"backend/mail/internal/client/embed"
	"backend/mail/internal/client/imap"
	"backend/mail/internal/client/llm"
	"backend/mail/internal/config"
	"backend/mail/internal/model"
	"backend/mail/internal/storage"
)

type MailHandler struct {
	storage storage.MailStorage
	vector  storage.VectorStorage
	factory imap.FetcherFactory
	llm     llm.Generator
	model   embed.Embedder

	imapConfig config.IMAP
}

func NewMailHandler(storage storage.MailStorage, vector storage.VectorStorage, factory imap.FetcherFactory, llm llm.Generator, model embed.Embedder, config config.IMAP) MailHandler {
	return MailHandler{
		storage:    storage,
		vector:     vector,
		factory:    factory,
		llm:        llm,
		model:      model,
		imapConfig: config,
	}
}

func (h MailHandler) AnswerQuestion(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	if _, ok := r.Context().Value("user_id").(uuid.UUID); !ok {
		slog.Error("no user_id provided in context")
		sendResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}

	req := model.QuestionRequst{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Wrong request structure")
		return
	}

	auth := imap.Auth{Email: req.Email, Password: req.Password, Token: "", Address: req.Address, Method: "PLAIN"}
	fetcher, err := h.factory.NewFetcher(r.Context(), auth, nil)
	if err != nil {
		handleNewFetcherError(w, err)
		return
	}
	defer fetcher.Close()

	answer, err := h.runRAGPipeline(r.Context(), req, fetcher)
	if err != nil {
		slog.Error("run RAG pipeline", "error", err)
		sendResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}

	raw, err := json.Marshal(model.QuestionResponse{Content: answer})
	if err != nil {
		slog.Error("marshal question response", "error", err)
		sendResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}
	w.Write(raw)
}

func (h MailHandler) GetFolders(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	if _, ok := r.Context().Value("user_id").(uuid.UUID); !ok {
		slog.Error("no user_id provided in context")
		sendResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}

	req := model.GetFoldersRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "Wrong request structure")
		return
	}

	config := h.imapConfig
	config.MaxConnections = 1
	auth := imap.Auth{Email: req.Email, Password: req.Password, Token: "", Address: req.Address, Method: "PLAIN"}

	fetcher, err := h.factory.NewFetcher(r.Context(), auth, &config)
	if err != nil {
		handleNewFetcherError(w, err)
		return
	}
	defer fetcher.Close()

	folders, err := fetcher.FetchFolders(r.Context())
	if err != nil {
		slog.Error("fetch folders", "error", err)
		sendResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}

	raw, err := json.Marshal(model.GetFoldersResponse{Folders: folders})
	if err != nil {
		slog.Error("marshal get folders response", "error", err)
		sendResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}
	w.Write(raw)
}

func (h MailHandler) runRAGPipeline(ctx context.Context, req model.QuestionRequst, fetcher imap.Fetcher) (string, error) {
	if err := h.index(ctx, req, fetcher); err != nil {
		return "", fmt.Errorf("stage - indexing: %w", err)
	}

	scored, err := h.retrieve(ctx, req)
	if err != nil {
		return "", fmt.Errorf("stage - retrieval: %w", err)
	}

	answer, err := h.generate(ctx, req, scored)
	if err != nil {
		return "", fmt.Errorf("stage - generation: %w", err)
	}
	return answer, nil
}

func (h MailHandler) index(ctx context.Context, req model.QuestionRequst, fetcher imap.Fetcher) error {
	// fetch new letters from mail server
	defer fetcher.Close()

	userID := ctx.Value("user_id").(uuid.UUID)
	
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

	// to embeddings
	vectors, err := h.model.Embed(ctx, chunks)
	if err != nil {
		return fmt.Errorf("embed letters chunks: %w", err)
	}

	// store in vector storage
	var points []storage.Point
	for i := range letters {
		points = append(points, storage.Point{Embedding: vectors[i], Payload: &letters[i]})
	}

	if err = h.vector.Insert(ctx, userID, points); err != nil {
		return fmt.Errorf("upsert points to collection: %w", err)
	}
	return nil
}

func (h MailHandler) retrieve(ctx context.Context, req model.QuestionRequst) ([]storage.ScoredPoint, error) {
	questionVector, err := h.model.Embed(ctx, []string{req.Question})
	if err != nil {
		return nil, fmt.Errorf("embed question chunk: %w", err)
	}

	userID := ctx.Value("user_id").(uuid.UUID)
	scored, err := h.vector.Search(ctx, userID, questionVector[0])
	if err != nil {
		return nil, fmt.Errorf("search question embedding in vector storage: %w", err)
	}
	return scored, nil
}

func (h MailHandler) generate(ctx context.Context, req model.QuestionRequst, scored []storage.ScoredPoint) (string, error) {
	resp, err := h.llm.Generate(ctx, req.Question, scored)
	if err != nil {
		return "", fmt.Errorf("generate llm response: %w", err)
	}
	if len(resp) == 0 {
		return "", fmt.Errorf("empty llm response: %w", err)
	}
	return resp, nil
}

func (h MailHandler) collectNewLetters(ctx context.Context, req model.QuestionRequst, fetcher imap.Fetcher, folders []imap.Folder) ([]imap.Letter, error) {
	userID := ctx.Value("user_id").(uuid.UUID)

	foldersDB, err := h.storage.GetFolders(ctx, userID, req.Email)
	if err != nil && !errors.Is(err, storage.ErrNotFoundFolders) {
		return nil, fmt.Errorf("get folders from storage: %w", err)
	}

	foldersOldMap := make(map[string]imap.Folder)
	foldersNewMap := make(map[string]imap.Folder)
	for _, item := range foldersDB {
		foldersOldMap[item.Name] = item
	}
	for _, item := range folders {
		foldersNewMap[item.Name] = item
	}

	var letters []imap.Letter
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

func handleNewFetcherError(w http.ResponseWriter, err error) {
	if err != nil {
		if errors.Is(err, imap.ErrAppPasswordRequired) {
			sendResponse(w, http.StatusUnprocessableEntity, imap.ErrAppPasswordRequired.Error())
		} else if errors.Is(err, imap.ErrAuthenticationFailed) {
			sendResponse(w, http.StatusUnprocessableEntity, imap.ErrAuthenticationFailed.Error())
		} else {
			slog.Error("new fetcher", "error", err)
			sendResponse(w, http.StatusInternalServerError, "Internal error")
		}
	}
}
