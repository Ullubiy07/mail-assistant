package imap

import (
	"time"

	"github.com/emersion/go-imap/v2"
)

type Envelope struct {
	Date    time.Time
	Subject string
	From    []imap.Address
	UID     uint32
}

type Letter struct {
	Envelope Envelope
	Body     string
}
