package model

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID             uuid.UUID `json:"id"`
	OrderID        uuid.UUID `json:"order_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Body           string    `json:"body"`
	AttachmentID   string    `json:"attachment_id,omitempty"`
	AttachmentSize int       `json:"attachment_size,omitempty"`
	AttachmentMime string    `json:"attachment_mime,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// MessageAttachment stores attachment binary data in PostgreSQL,
// making the database the sole system of record for all core entities.
type MessageAttachment struct {
	ID        uuid.UUID `json:"id"`
	MessageID uuid.UUID `json:"message_id"`
	Data      []byte    `json:"-"`
	Size      int       `json:"size"`
	Mime      string    `json:"mime"`
	CreatedAt time.Time `json:"created_at"`
}
