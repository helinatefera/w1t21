package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AnalyticsEvent struct {
	ID            uuid.UUID       `json:"id"`
	UserID        *uuid.UUID      `json:"user_id,omitempty"`
	EventType     string          `json:"event_type"`
	CollectibleID *uuid.UUID      `json:"collectible_id,omitempty"`
	SessionID     string          `json:"session_id"`
	ABVariant     string          `json:"ab_variant,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type IPRule struct {
	ID        uuid.UUID `json:"id"`
	CIDR      string    `json:"cidr"`
	Action    string    `json:"action"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type AnomalyEvent struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"user_id"`
	AnomalyType  string          `json:"anomaly_type"`
	Details      json.RawMessage `json:"details"`
	Acknowledged bool            `json:"acknowledged"`
	CreatedAt    time.Time       `json:"created_at"`
}
