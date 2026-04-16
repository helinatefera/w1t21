package model

import "testing"

func TestOrderStatus_AllTransitions(t *testing.T) {
	tests := []struct {
		from    OrderStatus
		to      OrderStatus
		allowed bool
	}{
		// Pending transitions
		{OrderStatusPending, OrderStatusConfirmed, true},
		{OrderStatusPending, OrderStatusCancelled, true},
		{OrderStatusPending, OrderStatusProcessing, false},
		{OrderStatusPending, OrderStatusCompleted, false},

		// Confirmed transitions
		{OrderStatusConfirmed, OrderStatusProcessing, true},
		{OrderStatusConfirmed, OrderStatusCancelled, true},
		{OrderStatusConfirmed, OrderStatusCompleted, false},
		{OrderStatusConfirmed, OrderStatusPending, false},

		// Processing transitions
		{OrderStatusProcessing, OrderStatusCompleted, true},
		{OrderStatusProcessing, OrderStatusCancelled, false},
		{OrderStatusProcessing, OrderStatusPending, false},
		{OrderStatusProcessing, OrderStatusConfirmed, false},

		// Terminal states - no transitions
		{OrderStatusCompleted, OrderStatusPending, false},
		{OrderStatusCompleted, OrderStatusConfirmed, false},
		{OrderStatusCompleted, OrderStatusProcessing, false},
		{OrderStatusCompleted, OrderStatusCancelled, false},
		{OrderStatusCancelled, OrderStatusPending, false},
		{OrderStatusCancelled, OrderStatusConfirmed, false},
		{OrderStatusCancelled, OrderStatusProcessing, false},
		{OrderStatusCancelled, OrderStatusCompleted, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			got := tc.from.CanTransitionTo(tc.to)
			if got != tc.allowed {
				t.Errorf("%s.CanTransitionTo(%s) = %v, want %v",
					tc.from, tc.to, got, tc.allowed)
			}
		})
	}
}

func TestAllowedTransitions_Coverage(t *testing.T) {
	// Verify every status has an entry in AllowedTransitions
	statuses := []OrderStatus{
		OrderStatusPending,
		OrderStatusConfirmed,
		OrderStatusProcessing,
		OrderStatusCompleted,
		OrderStatusCancelled,
	}

	for _, s := range statuses {
		if _, exists := AllowedTransitions[s]; !exists {
			t.Errorf("AllowedTransitions missing entry for %s", s)
		}
	}
}

func TestTerminalStates_HaveNoTransitions(t *testing.T) {
	terminals := []OrderStatus{OrderStatusCompleted, OrderStatusCancelled}
	for _, s := range terminals {
		transitions := AllowedTransitions[s]
		if len(transitions) != 0 {
			t.Errorf("terminal state %s should have no transitions, has %v", s, transitions)
		}
	}
}
