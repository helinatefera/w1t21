package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ledgermint/platform/internal/model"
)

type AuditStore struct {
	pool *pgxpool.Pool
}

func NewAuditStore(pool *pgxpool.Pool) *AuditStore {
	return &AuditStore{pool: pool}
}

func (s *AuditStore) Log(ctx context.Context, entry *model.AuditLog) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, details, ip_address)
		 VALUES ($1, $2, $3, $4, $5, $6::inet)`,
		entry.ActorID, entry.Action, entry.ResourceType, entry.ResourceID, entry.Details, nullableIP(entry.IPAddress))
	return err
}

// LogEvent is a convenience wrapper that builds an AuditLog from args.
func (s *AuditStore) LogEvent(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, details map[string]interface{}, ip string) {
	detailsJSON, _ := json.Marshal(details)
	// Best-effort: audit writes must never block or fail callers.
	_ = s.Log(ctx, &model.AuditLog{
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      detailsJSON,
		IPAddress:    ip,
	})
}

func nullableIP(ip string) *string {
	if ip == "" {
		return nil
	}
	return &ip
}
