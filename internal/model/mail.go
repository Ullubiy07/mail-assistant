package model

import (
	"github.com/emersion/go-imap/v2"
)

type Letter struct {
	Envelope *imap.Envelope
	Body     string
}
