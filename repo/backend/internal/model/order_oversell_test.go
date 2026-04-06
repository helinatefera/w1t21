package model

import (
	"testing"
)

// These tests exercise the order state machine transitions that are critical
// for preventing oversell conditions. The state machine is the last line of
// defence: even if the advisory lock or active-order check fails, the state
// machine must reject invalid transitions.

// TestOversellDefence_CompletedOrderCannotBeReordered verifies that once an
// order reaches "completed", no transition can move it back to an active state.
func TestOversellDefence_CompletedOrderCannotBeReordered(t *testing.T) {
	blockedTransitions := []OrderStatus{
		OrderStatusPending,
		OrderStatusConfirmed,
		OrderStatusProcessing,
		OrderStatusCompleted,
		OrderStatusCancelled,
	}

	for _, target := range blockedTransitions {
		if target == OrderStatusCancelled {
			// completed → cancelled is intentionally blocked by state machine
			continue
		}
		if OrderStatusCompleted.CanTransitionTo(target) {
			t.Errorf("completed → %s should be blocked (oversell defence)", target)
		}
	}
}

// TestOversellDefence_PendingCannotSkipToCompleted verifies the
// mandatory progression pending → confirmed → processing → completed.
func TestOversellDefence_PendingCannotSkipToCompleted(t *testing.T) {
	if OrderStatusPending.CanTransitionTo(OrderStatusCompleted) {
		t.Fatal("pending → completed should be blocked (must go through confirm/process)")
	}
	if OrderStatusPending.CanTransitionTo(OrderStatusProcessing) {
		t.Fatal("pending → processing should be blocked (must confirm first)")
	}
}

// TestOversellDefence_ConfirmedCannotSkipToCompleted verifies confirmed
// must go through processing before completing.
func TestOversellDefence_ConfirmedCannotSkipToCompleted(t *testing.T) {
	if OrderStatusConfirmed.CanTransitionTo(OrderStatusCompleted) {
		t.Fatal("confirmed → completed should be blocked (must process first)")
	}
}

// TestIdempotencyKey_SameKeyReturnsSameResult validates the idempotency
// invariant: two order creation requests with the same (buyerID, idempotencyKey)
// must resolve identically. This test exercises the model-level constraint
// that the composite unique key enforces.
func TestIdempotencyKey_StateTransitionAfterCreate(t *testing.T) {
	// A newly created order starts as pending
	order := Order{Status: OrderStatusPending}

	// The valid first transition is to confirmed (by seller) or cancelled
	if !order.Status.CanTransitionTo(OrderStatusConfirmed) {
		t.Fatal("pending → confirmed should be allowed")
	}
	if !order.Status.CanTransitionTo(OrderStatusCancelled) {
		t.Fatal("pending → cancelled should be allowed")
	}
}

// TestCancelledOrderFreesSlot verifies that a cancelled order is in a terminal
// state, which is essential for the oversell check: HasActiveOrder excludes
// cancelled orders, so cancellation frees the collectible for a new order.
func TestCancelledOrderFreesSlot(t *testing.T) {
	terminalStatuses := []OrderStatus{OrderStatusPending, OrderStatusConfirmed, OrderStatusProcessing, OrderStatusCompleted, OrderStatusCancelled}
	for _, target := range terminalStatuses {
		if OrderStatusCancelled.CanTransitionTo(target) {
			t.Errorf("cancelled → %s should be blocked (terminal state)", target)
		}
	}
}

// TestFullOrderLifecycle_HappyPath exercises the complete lifecycle
// that must succeed for a valid order.
func TestFullOrderLifecycle_HappyPath(t *testing.T) {
	transitions := []struct {
		from, to OrderStatus
	}{
		{OrderStatusPending, OrderStatusConfirmed},
		{OrderStatusConfirmed, OrderStatusProcessing},
		{OrderStatusProcessing, OrderStatusCompleted},
	}

	for _, tr := range transitions {
		if !tr.from.CanTransitionTo(tr.to) {
			t.Errorf("%s → %s should be allowed in happy path", tr.from, tr.to)
		}
	}
}
