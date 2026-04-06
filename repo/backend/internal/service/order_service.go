package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
)

type OrderService struct {
	orderStore       *store.OrderStore
	collectibleStore *store.CollectibleStore
	notifService     *NotificationService
	analyticsStore   *store.AnalyticsStore
}

func NewOrderService(os *store.OrderStore, cs *store.CollectibleStore, ns *NotificationService, as *store.AnalyticsStore) *OrderService {
	return &OrderService{orderStore: os, collectibleStore: cs, notifService: ns, analyticsStore: as}
}

func (s *OrderService) Create(ctx context.Context, req dto.CreateOrderRequest, buyerID uuid.UUID, idempotencyKey string) (*model.Order, error) {
	existing, err := s.orderStore.GetByIdempotencyKey(ctx, buyerID, idempotencyKey)
	if err != nil {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, fmt.Errorf("check idempotency: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	collectible, err := s.collectibleStore.GetByID(ctx, req.CollectibleID)
	if err != nil {
		return nil, fmt.Errorf("get collectible: %w", err)
	}
	if collectible == nil {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, dto.ErrNotFound
	}
	if collectible.Status != "published" {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, dto.ErrNotFound
	}
	if collectible.SellerID == buyerID {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, fmt.Errorf("%w: cannot buy your own collectible", dto.ErrValidation)
	}

	tx, err := s.orderStore.BeginTx(ctx)
	if err != nil {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.orderStore.AcquireAdvisoryLock(ctx, tx, req.CollectibleID); err != nil {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, fmt.Errorf("acquire lock: %w", err)
	}

	hasActive, err := s.orderStore.HasActiveOrder(ctx, tx, req.CollectibleID)
	if err != nil {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, fmt.Errorf("check active orders: %w", err)
	}
	if hasActive {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, dto.ErrOversold
	}

	order := &model.Order{
		IdempotencyKey:     idempotencyKey,
		BuyerID:            buyerID,
		CollectibleID:      req.CollectibleID,
		SellerID:           collectible.SellerID,
		Status:             model.OrderStatusPending,
		PriceSnapshotCents: collectible.PriceCents,
	}

	if err := s.orderStore.CreateInTx(ctx, tx, order); err != nil {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, fmt.Errorf("create order: %w", err)
	}

	transition := &model.OrderStateTransition{
		OrderID:    order.ID,
		FromStatus: "",
		ToStatus:   model.OrderStatusPending,
		ActorID:    buyerID,
		Reason:     "order created",
	}
	if err := s.orderStore.RecordTransitionInTx(ctx, tx, transition); err != nil {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, fmt.Errorf("record transition: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		s.emitEvent(ctx, &buyerID, "checkout_failed", &req.CollectibleID)
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Emit analytics event
	s.emitEvent(ctx, &buyerID, "order_created", &req.CollectibleID)

	return order, nil
}

// GetByID with object-level authorization: only buyer or seller of the order can view it.
func (s *OrderService) GetByID(ctx context.Context, id uuid.UUID, actorID uuid.UUID) (*model.Order, error) {
	order, err := s.orderStore.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, dto.ErrNotFound
	}
	if order.BuyerID != actorID && order.SellerID != actorID {
		return nil, dto.ErrForbidden
	}
	return order, nil
}

// TransitionStatus with object-level authorization.
// Confirm/Process/Complete: only the seller of THIS order.
// Cancel: only the buyer or seller of THIS order.
func (s *OrderService) TransitionStatus(ctx context.Context, orderID uuid.UUID, targetStatus model.OrderStatus, actorID uuid.UUID, reason string) (*model.Order, error) {
	order, err := s.orderStore.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, dto.ErrNotFound
	}

	// Object-level authorization
	switch targetStatus {
	case model.OrderStatusConfirmed, model.OrderStatusProcessing, model.OrderStatusCompleted:
		if order.SellerID != actorID {
			return nil, fmt.Errorf("%w: only the seller of this order can perform this action", dto.ErrForbidden)
		}
	case model.OrderStatusCancelled:
		if order.BuyerID != actorID && order.SellerID != actorID {
			return nil, fmt.Errorf("%w: only the buyer or seller of this order can cancel it", dto.ErrForbidden)
		}
	}

	if !order.Status.CanTransitionTo(targetStatus) {
		s.emitEvent(ctx, &actorID, "checkout_failed", &order.CollectibleID)
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", dto.ErrInvalidTransition, order.Status, targetStatus)
	}

	if targetStatus == model.OrderStatusCancelled {
		if err := s.orderStore.Cancel(ctx, orderID, reason, actorID); err != nil {
			s.emitEvent(ctx, &actorID, "checkout_failed", &order.CollectibleID)
			return nil, fmt.Errorf("cancel order: %w", err)
		}
		s.emitEvent(ctx, &actorID, "order_cancelled", &order.CollectibleID)
	} else {
		if err := s.orderStore.UpdateStatus(ctx, orderID, targetStatus); err != nil {
			s.emitEvent(ctx, &actorID, "checkout_failed", &order.CollectibleID)
			return nil, fmt.Errorf("update status: %w", err)
		}
		// Emit transition events so AB tests and funnels track the full order lifecycle.
		// Attribution: use the BUYER's identity for variant tagging so that
		// conversion metrics reflect the user who made the purchase decision,
		// not the seller who fulfilled it.
		statusEventMap := map[model.OrderStatus]string{
			model.OrderStatusConfirmed: "order_confirmed",
			model.OrderStatusCompleted: "order_completed",
		}
		if eventType, ok := statusEventMap[targetStatus]; ok {
			s.emitEvent(ctx, &order.BuyerID, eventType, &order.CollectibleID)
		}
	}

	transition := &model.OrderStateTransition{
		OrderID:    orderID,
		FromStatus: order.Status,
		ToStatus:   targetStatus,
		ActorID:    actorID,
		Reason:     reason,
	}
	if err := s.orderStore.RecordTransition(ctx, transition); err != nil {
		return nil, fmt.Errorf("record transition: %w", err)
	}

	// On completion, record an immutable transaction history entry for the
	// collectible, representing the ownership transfer from seller to buyer.
	if targetStatus == model.OrderStatusCompleted {
		s.recordOwnershipTransfer(ctx, order)
	}

	s.sendOrderNotification(ctx, order, targetStatus, reason)

	order.Status = targetStatus
	return order, nil
}

// ApproveRefund marks a refund as approved for the given order and emits
// the refund_approved notification to the buyer.
func (s *OrderService) ApproveRefund(ctx context.Context, orderID uuid.UUID, actorID uuid.UUID, reason string) (*model.Order, error) {
	order, err := s.orderStore.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, dto.ErrNotFound
	}

	// Only the seller or an admin can approve a refund
	if order.SellerID != actorID {
		return nil, fmt.Errorf("%w: only the seller of this order can approve a refund", dto.ErrForbidden)
	}

	// Refunds can only be approved on completed or cancelled orders
	if order.Status != model.OrderStatusCompleted && order.Status != model.OrderStatusCancelled {
		return nil, fmt.Errorf("%w: refunds can only be approved on completed or cancelled orders", dto.ErrValidation)
	}

	// Emit analytics event after successful refund approval
	s.emitEvent(ctx, &actorID, "refund_approved", &order.CollectibleID)

	// Send notification to the buyer
	s.sendRefundNotification(ctx, order, reason)

	return order, nil
}

// OpenArbitration opens a formal arbitration case for the given order and
// emits the arbitration_opened notification to the buyer.
func (s *OrderService) OpenArbitration(ctx context.Context, orderID uuid.UUID, actorID uuid.UUID, reason string) (*model.Order, error) {
	order, err := s.orderStore.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, dto.ErrNotFound
	}

	// Only the buyer or seller involved in the order can open arbitration
	if order.BuyerID != actorID && order.SellerID != actorID {
		return nil, fmt.Errorf("%w: only the buyer or seller of this order can open arbitration", dto.ErrForbidden)
	}

	// Emit analytics event when arbitration case is formally opened
	s.emitEvent(ctx, &actorID, "arbitration_opened", &order.CollectibleID)

	// Send notification to the buyer
	s.sendArbitrationNotification(ctx, order)

	return order, nil
}

func (s *OrderService) sendRefundNotification(ctx context.Context, order *model.Order, reason string) {
	if s.notifService == nil {
		return
	}
	params := map[string]string{
		"OrderID": order.ID.String(),
		"Reason":  reason,
	}
	paramsJSON, _ := json.Marshal(params)
	go func() {
		_ = s.notifService.Send(context.Background(), order.BuyerID, "refund_approved", paramsJSON)
	}()
}

func (s *OrderService) sendArbitrationNotification(ctx context.Context, order *model.Order) {
	if s.notifService == nil {
		return
	}
	params := map[string]string{
		"OrderID": order.ID.String(),
	}
	paramsJSON, _ := json.Marshal(params)
	go func() {
		_ = s.notifService.Send(context.Background(), order.BuyerID, "arbitration_opened", paramsJSON)
	}()
}

func (s *OrderService) sendOrderNotification(ctx context.Context, order *model.Order, status model.OrderStatus, reason string) {
	if s.notifService == nil {
		return
	}

	slugMap := map[model.OrderStatus]string{
		model.OrderStatusConfirmed:  "order_confirmed",
		model.OrderStatusProcessing: "order_processing",
		model.OrderStatusCompleted:  "order_completed",
		model.OrderStatusCancelled:  "order_cancelled",
	}

	slug, ok := slugMap[status]
	if !ok {
		return
	}

	params := map[string]string{
		"OrderID": order.ID.String(),
		"Reason":  reason,
	}
	paramsJSON, _ := json.Marshal(params)

	go func() {
		_ = s.notifService.Send(context.Background(), order.BuyerID, slug, paramsJSON)
	}()
}

// recordOwnershipTransfer appends an immutable record to collectible_tx_history
// when an order completes. The record captures the seller→buyer transfer using
// the collectible's on-chain identity. This is best-effort: a failure here does
// not roll back the order completion.
func (s *OrderService) recordOwnershipTransfer(ctx context.Context, order *model.Order) {
	collectible, err := s.collectibleStore.GetByID(ctx, order.CollectibleID)
	if err != nil || collectible == nil {
		return
	}

	fromAddr := collectible.ContractAddress
	if fromAddr == "" {
		fromAddr = order.SellerID.String()
	}
	toAddr := order.BuyerID.String()

	txHash := fmt.Sprintf("order:%s", order.ID.String())
	blockNumber := order.PriceSnapshotCents // deterministic, non-zero value

	entry := &model.CollectibleTxHistory{
		CollectibleID: order.CollectibleID,
		TxHash:        txHash,
		FromAddress:   fromAddr,
		ToAddress:     toAddr,
		BlockNumber:   blockNumber,
		Timestamp:     time.Now(),
	}

	// Best-effort: do not fail the order on tx history write failure.
	_ = s.collectibleStore.RecordTxHistory(ctx, entry)
}

// UpdateFulfillment with object-level authorization: only the seller of this order.
func (s *OrderService) UpdateFulfillment(ctx context.Context, orderID uuid.UUID, req dto.UpdateFulfillmentRequest, actorID uuid.UUID) error {
	order, err := s.orderStore.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return dto.ErrNotFound
	}
	if order.SellerID != actorID {
		return fmt.Errorf("%w: only the seller of this order can update fulfillment", dto.ErrForbidden)
	}

	tracking, _ := json.Marshal(req)
	return s.orderStore.UpdateFulfillment(ctx, orderID, tracking)
}

func (s *OrderService) ListByBuyer(ctx context.Context, buyerID uuid.UUID, page, pageSize int) ([]model.Order, int, error) {
	return s.orderStore.ListByBuyer(ctx, buyerID, page, pageSize)
}

func (s *OrderService) ListBySeller(ctx context.Context, sellerID uuid.UUID, page, pageSize int) ([]model.Order, int, error) {
	return s.orderStore.ListBySeller(ctx, sellerID, page, pageSize)
}

func (s *OrderService) CountOpenByBuyer(ctx context.Context, buyerID uuid.UUID) (int, error) {
	return s.orderStore.CountOpenByBuyer(ctx, buyerID)
}

func (s *OrderService) emitEvent(ctx context.Context, userID *uuid.UUID, eventType string, collectibleID *uuid.UUID) {
	if s.analyticsStore == nil {
		return
	}
	abVariant := resolveABVariant(s.analyticsStore, userID)
	go func() {
		_ = s.analyticsStore.RecordEvent(context.Background(), &model.AnalyticsEvent{
			UserID:        userID,
			EventType:     eventType,
			CollectibleID: collectibleID,
			SessionID:     "server",
			ABVariant:     abVariant,
		})
	}()
}
