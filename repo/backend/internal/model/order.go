package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

var AllowedTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusPending:    {OrderStatusConfirmed, OrderStatusCancelled},
	OrderStatusConfirmed:  {OrderStatusProcessing, OrderStatusCancelled},
	OrderStatusProcessing: {OrderStatusCompleted},
	OrderStatusCompleted:  {},
	OrderStatusCancelled:  {},
}

func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	for _, allowed := range AllowedTransitions[s] {
		if allowed == target {
			return true
		}
	}
	return false
}

type Order struct {
	ID                  uuid.UUID       `json:"id"`
	IdempotencyKey      string          `json:"idempotency_key"`
	BuyerID             uuid.UUID       `json:"buyer_id"`
	CollectibleID       uuid.UUID       `json:"collectible_id"`
	SellerID            uuid.UUID       `json:"seller_id"`
	Status              OrderStatus     `json:"status"`
	PriceSnapshotCents  int64           `json:"price_snapshot_cents"`
	CancellationReason  string          `json:"cancellation_reason,omitempty"`
	CancelledBy         *uuid.UUID      `json:"cancelled_by,omitempty"`
	FulfillmentTracking json.RawMessage `json:"fulfillment_tracking,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type OrderStateTransition struct {
	ID         uuid.UUID   `json:"id"`
	OrderID    uuid.UUID   `json:"order_id"`
	FromStatus OrderStatus `json:"from_status"`
	ToStatus   OrderStatus `json:"to_status"`
	ActorID    uuid.UUID   `json:"actor_id"`
	Reason     string      `json:"reason,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}
