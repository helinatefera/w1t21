package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID           uuid.UUID       `json:"id"`
	ActorID      *uuid.UUID      `json:"actor_id"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *uuid.UUID      `json:"resource_id"`
	Details      json.RawMessage `json:"details,omitempty"`
	IPAddress    string          `json:"ip_address,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Canonical audit actions.
const (
	AuditActionLogin          = "auth.login"
	AuditActionLoginFailed    = "auth.login_failed"
	AuditActionLogout         = "auth.logout"
	AuditActionTokenRefresh   = "auth.token_refresh"
	AuditActionUserCreate     = "user.create"
	AuditActionUserUpdate     = "user.update"
	AuditActionUserUnlock     = "user.unlock"
	AuditActionRoleAdd        = "user.role_add"
	AuditActionRoleRemove     = "user.role_remove"
	AuditActionOrderCreate    = "order.create"
	AuditActionOrderTransit   = "order.transition"
	AuditActionCollectHide    = "collectible.hide"
	AuditActionCollectPublish = "collectible.publish"
)
