package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type NotificationTemplate struct {
	ID           uuid.UUID `json:"id"`
	Slug         string    `json:"slug"`
	TitleTemplate string   `json:"title_template"`
	BodyTemplate  string   `json:"body_template"`
	CreatedAt    time.Time `json:"created_at"`
}

type Notification struct {
	ID            uuid.UUID       `json:"id"`
	UserID        uuid.UUID       `json:"user_id"`
	TemplateID    uuid.UUID       `json:"template_id"`
	TemplateSlug  string          `json:"template_slug,omitempty"`
	Params        json.RawMessage `json:"params,omitempty"`
	RenderedTitle string          `json:"rendered_title"`
	RenderedBody  string          `json:"rendered_body"`
	IsRead        bool            `json:"is_read"`
	Status        string          `json:"status"`
	RetryCount    int             `json:"retry_count"`
	MaxRetries    int             `json:"max_retries"`
	NextRetryAt   *time.Time      `json:"next_retry_at,omitempty"`
	DeliveredAt   *time.Time      `json:"delivered_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type NotificationPreferences struct {
	UserID           uuid.UUID       `json:"user_id"`
	Preferences      json.RawMessage `json:"preferences"`
	SubscriptionMode string          `json:"subscription_mode"`
}
