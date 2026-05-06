package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// type MailHandler struct {
// 	storage storage.UserStorage
// }

type QuestionRequst struct {
	Message string `json:"msg"`
}

type QuestionResponse struct {
	Message string `json:"msg"`
}

func Question(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	req := QuestionRequst{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, http.StatusBadRequest, "wrong request structure")
		return
	}

	resp := QuestionResponse{}
	rawResp, err := json.Marshal(resp)
	if err != nil {
		slog.Error("marshal question response", "error", err)
		sendResponse(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.Write(rawResp)
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
