package mail

import (
	"context"
	"time"
)

type Address struct {
	Name    string
	Mailbox string
	Host    string
}

type Envelope struct {
	Date    time.Time
	Subject string
	From    Address
	UID     uint32
}

type Letter struct {
	Envelope Envelope
	Body     string
}

type FolderState struct {
	Folder      string
	UIDNext     uint32
	UIDValidity uint32
}

type Reader interface {
	GetNewLetters(ctx context.Context, folder string, uid uint32) ([]Letter, *FolderState, error)
	GetFolders(ctx context.Context) ([]string, error)
}
