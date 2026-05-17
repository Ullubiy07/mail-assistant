package model

import "backend/mail/internal/client/imap"

type QuestionRequst struct {
	Address  string   `json:"address"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Folders  []string `json:"folders"`
	Question string   `json:"question"`
}

type QuestionResponse struct {
	Content string `json:"content"`
}

type GetFoldersRequest struct {
	Address  string `json:"address"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GetFoldersResponse struct {
	Folders []imap.Folder `json:"folders"`
}
