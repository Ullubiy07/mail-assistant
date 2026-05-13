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

type Folder struct {
	Name        string
	UIDNext     uint32
	UIDValidity uint32
}

type Auth struct {
	Email    string
	Password string
	Token    string
	Address  string
	Method   string
}

type FetcherFactory interface {
	NewFetcher(auth Auth) Fetcher
}

type Fetcher interface {
	FetchNewLetters(ctx context.Context, folder string, uid uint32) ([]Letter, error)
	FetchFolders(ctx context.Context) ([]Folder, error)
}
