package imap

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
	Name        string `db:"name"`
	NumMessages uint32 `db:"num_messages"`
	UIDNext     uint32 `db:"uid_next"`
	UIDValidity uint32 `db:"uid_validity"`
}

type Auth struct {
	Email    string
	Password string
	Token    string
	Address  string
	Method   string
}

type FetcherFactory interface {
	NewFetcher(ctx context.Context, auth Auth) (Fetcher, error)
}

type Fetcher interface {
	FetchNewLetters(ctx context.Context, folder string, uid uint32) ([]Letter, error)
	FetchFolders(ctx context.Context) ([]Folder, error)
	Close() error
}
