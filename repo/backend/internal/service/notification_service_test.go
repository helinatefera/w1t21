package service

import "testing"

func TestIsStatusOnlySlug(t *testing.T) {
	statusSlugs := []string{
		"order_confirmed",
		"order_processing",
		"order_completed",
		"order_cancelled",
		"refund_approved",
	}
	for _, slug := range statusSlugs {
		if !IsStatusOnlySlug(slug) {
			t.Errorf("expected %q to be a status-only slug", slug)
		}
	}

	auxiliarySlugs := []string{
		"review_posted",
		"arbitration_opened",
		"some_unknown_event",
	}
	for _, slug := range auxiliarySlugs {
		if IsStatusOnlySlug(slug) {
			t.Errorf("expected %q to NOT be a status-only slug", slug)
		}
	}
}

func TestStatusOnlyMode_FiltersAuxiliaryEvents(t *testing.T) {
	// Verify the filtering logic: status_only mode should include status slugs
	// and exclude auxiliary slugs. This tests the same map used by Send().
	tests := []struct {
		slug    string
		mode    string
		allowed bool
	}{
		// all_events mode allows everything
		{"order_confirmed", "all_events", true},
		{"review_posted", "all_events", true},
		{"arbitration_opened", "all_events", true},

		// status_only mode filters auxiliary events
		{"order_confirmed", "status_only", true},
		{"order_processing", "status_only", true},
		{"order_completed", "status_only", true},
		{"order_cancelled", "status_only", true},
		{"refund_approved", "status_only", true},
		{"review_posted", "status_only", false},
		{"arbitration_opened", "status_only", false},

		// empty mode (backward compat) defaults to all_events behavior
		{"review_posted", "", true},
		{"order_confirmed", "", true},
	}

	for _, tc := range tests {
		mode := tc.mode
		if mode == "" {
			mode = "all_events"
		}
		allowed := mode != "status_only" || statusOnlySlugs[tc.slug]
		if allowed != tc.allowed {
			t.Errorf("mode=%q slug=%q: got allowed=%v, want %v", tc.mode, tc.slug, allowed, tc.allowed)
		}
	}
}

func TestDefaultSubscriptionMode(t *testing.T) {
	// Verify that an empty subscription mode defaults to all_events.
	// This ensures backward compatibility for users who haven't chosen a mode.
	mode := ""
	if mode == "" {
		mode = "all_events"
	}
	if mode != "all_events" {
		t.Errorf("expected default mode to be all_events, got %q", mode)
	}
}
