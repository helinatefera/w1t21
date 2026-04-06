package model

import "testing"

func TestOrderStatus_AllStatesRecognized(t *testing.T) {
	expected := []OrderStatus{
		OrderStatusPending, OrderStatusConfirmed, OrderStatusProcessing,
		OrderStatusCompleted, OrderStatusCancelled,
	}
	for _, s := range expected {
		if _, ok := AllowedTransitions[s]; !ok {
			t.Errorf("state %q not in AllowedTransitions map", s)
		}
	}
	if len(AllowedTransitions) != len(expected) {
		t.Errorf("AllowedTransitions has %d entries, expected %d", len(AllowedTransitions), len(expected))
	}
}

func TestOrderStatus_ValidTransitions(t *testing.T) {
	tests := []struct {
		from, to OrderStatus
		ok       bool
	}{
		{OrderStatusPending, OrderStatusConfirmed, true},
		{OrderStatusPending, OrderStatusCancelled, true},
		{OrderStatusConfirmed, OrderStatusProcessing, true},
		{OrderStatusConfirmed, OrderStatusCancelled, true},
		{OrderStatusProcessing, OrderStatusCompleted, true},

		// Invalid transitions
		{OrderStatusPending, OrderStatusProcessing, false},
		{OrderStatusPending, OrderStatusCompleted, false},
		{OrderStatusConfirmed, OrderStatusCompleted, false},
		{OrderStatusProcessing, OrderStatusCancelled, false},
		{OrderStatusCompleted, OrderStatusCancelled, false},
		{OrderStatusCompleted, OrderStatusPending, false},
		{OrderStatusCancelled, OrderStatusPending, false},
	}

	for _, tc := range tests {
		name := string(tc.from) + " -> " + string(tc.to)
		t.Run(name, func(t *testing.T) {
			got := tc.from.CanTransitionTo(tc.to)
			if got != tc.ok {
				t.Errorf("CanTransitionTo(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.ok)
			}
		})
	}
}

func TestOrderStatus_TerminalStates(t *testing.T) {
	terminals := []OrderStatus{OrderStatusCompleted, OrderStatusCancelled}
	all := []OrderStatus{
		OrderStatusPending, OrderStatusConfirmed, OrderStatusProcessing,
		OrderStatusCompleted, OrderStatusCancelled,
	}
	for _, terminal := range terminals {
		for _, target := range all {
			if terminal.CanTransitionTo(target) {
				t.Errorf("terminal state %q should not transition to %q", terminal, target)
			}
		}
	}
}
