package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ledgermint/platform/internal/model"
)

type OrderStore struct {
	pool *pgxpool.Pool
}

func NewOrderStore(pool *pgxpool.Pool) *OrderStore {
	return &OrderStore{pool: pool}
}

func (s *OrderStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := s.pool.QueryRow(ctx,
		`SELECT id, idempotency_key, buyer_id, collectible_id, seller_id, status,
		        price_snapshot_cents, COALESCE(cancellation_reason, ''), cancelled_by,
		        COALESCE(fulfillment_tracking, '{}'::jsonb),
		        created_at, updated_at
		 FROM orders WHERE id = $1`, id).Scan(
		&o.ID, &o.IdempotencyKey, &o.BuyerID, &o.CollectibleID, &o.SellerID, &o.Status,
		&o.PriceSnapshotCents, &o.CancellationReason, &o.CancelledBy, &o.FulfillmentTracking,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &o, err
}

func (s *OrderStore) GetByIdempotencyKey(ctx context.Context, buyerID uuid.UUID, key string) (*model.Order, error) {
	var o model.Order
	err := s.pool.QueryRow(ctx,
		`SELECT id, idempotency_key, buyer_id, collectible_id, seller_id, status,
		        price_snapshot_cents, COALESCE(cancellation_reason, ''), cancelled_by,
		        COALESCE(fulfillment_tracking, '{}'::jsonb),
		        created_at, updated_at
		 FROM orders WHERE buyer_id = $1 AND idempotency_key = $2`, buyerID, key).Scan(
		&o.ID, &o.IdempotencyKey, &o.BuyerID, &o.CollectibleID, &o.SellerID, &o.Status,
		&o.PriceSnapshotCents, &o.CancellationReason, &o.CancelledBy, &o.FulfillmentTracking,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &o, err
}

func (s *OrderStore) HasActiveOrder(ctx context.Context, tx pgx.Tx, collectibleID uuid.UUID) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM orders WHERE collectible_id = $1 AND status NOT IN ('cancelled', 'completed'))`,
		collectibleID).Scan(&exists)
	return exists, err
}

func (s *OrderStore) AcquireAdvisoryLock(ctx context.Context, tx pgx.Tx, collectibleID uuid.UUID) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1::text))`, collectibleID.String())
	return err
}

func (s *OrderStore) CreateInTx(ctx context.Context, tx pgx.Tx, o *model.Order) error {
	return tx.QueryRow(ctx,
		`INSERT INTO orders (idempotency_key, buyer_id, collectible_id, seller_id, status, price_snapshot_cents)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		o.IdempotencyKey, o.BuyerID, o.CollectibleID, o.SellerID, o.Status, o.PriceSnapshotCents,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
}

func (s *OrderStore) UpdateStatus(ctx context.Context, id uuid.UUID, status model.OrderStatus) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE orders SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	return err
}

func (s *OrderStore) Cancel(ctx context.Context, id uuid.UUID, reason string, cancelledBy uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE orders SET status = 'cancelled', cancellation_reason = $2, cancelled_by = $3, updated_at = NOW()
		 WHERE id = $1`, id, reason, cancelledBy)
	return err
}

func (s *OrderStore) UpdateFulfillment(ctx context.Context, id uuid.UUID, tracking json.RawMessage) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE orders SET fulfillment_tracking = $2, updated_at = NOW() WHERE id = $1`, id, tracking)
	return err
}

func (s *OrderStore) RecordTransition(ctx context.Context, t *model.OrderStateTransition) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO order_state_transitions (order_id, from_status, to_status, actor_id, reason)
		 VALUES ($1, $2, $3, $4, $5)`,
		t.OrderID, t.FromStatus, t.ToStatus, t.ActorID, t.Reason)
	return err
}

func (s *OrderStore) RecordTransitionInTx(ctx context.Context, tx pgx.Tx, t *model.OrderStateTransition) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO order_state_transitions (order_id, from_status, to_status, actor_id, reason)
		 VALUES ($1, $2, $3, $4, $5)`,
		t.OrderID, t.FromStatus, t.ToStatus, t.ActorID, t.Reason)
	return err
}

func (s *OrderStore) ListByBuyer(ctx context.Context, buyerID uuid.UUID, page, pageSize int) ([]model.Order, int, error) {
	return s.listByField(ctx, "buyer_id", buyerID, page, pageSize)
}

func (s *OrderStore) ListBySeller(ctx context.Context, sellerID uuid.UUID, page, pageSize int) ([]model.Order, int, error) {
	return s.listByField(ctx, "seller_id", sellerID, page, pageSize)
}

func (s *OrderStore) listByField(ctx context.Context, field string, id uuid.UUID, page, pageSize int) ([]model.Order, int, error) {
	var total int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM orders WHERE `+field+` = $1`, id).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := s.pool.Query(ctx,
		`SELECT id, idempotency_key, buyer_id, collectible_id, seller_id, status,
		        price_snapshot_cents, COALESCE(cancellation_reason, ''), cancelled_by,
		        COALESCE(fulfillment_tracking, '{}'::jsonb),
		        created_at, updated_at
		 FROM orders WHERE `+field+` = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, id, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.IdempotencyKey, &o.BuyerID, &o.CollectibleID, &o.SellerID,
			&o.Status, &o.PriceSnapshotCents, &o.CancellationReason, &o.CancelledBy,
			&o.FulfillmentTracking, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, nil
}

func (s *OrderStore) CountOpenByBuyer(ctx context.Context, buyerID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM orders WHERE buyer_id = $1 AND status NOT IN ('cancelled', 'completed')`,
		buyerID).Scan(&count)
	return count, err
}

func (s *OrderStore) CountOpenBySeller(ctx context.Context, sellerID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM orders WHERE seller_id = $1 AND status NOT IN ('cancelled', 'completed')`,
		sellerID).Scan(&count)
	return count, err
}

func (s *OrderStore) CountCompletedByBuyer(ctx context.Context, buyerID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM orders WHERE buyer_id = $1 AND status = 'completed'`,
		buyerID).Scan(&count)
	return count, err
}

func (s *OrderStore) CountCancelledInPeriod(ctx context.Context, userID uuid.UUID, hours int) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM orders
		 WHERE (buyer_id = $1 OR seller_id = $1)
		   AND status = 'cancelled'
		   AND updated_at > NOW() - make_interval(hours => $2)`,
		userID, hours).Scan(&count)
	return count, err
}

func (s *OrderStore) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}

func (s *OrderStore) CountByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM orders GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}
	return result, nil
}
