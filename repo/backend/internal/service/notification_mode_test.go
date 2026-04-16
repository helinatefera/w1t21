package service

import "testing"

func TestIsStatusOnlySlug_Included(t *testing.T) {
	included := []string{
		"order_confirmed",
		"order_processing",
		"order_completed",
		"order_cancelled",
		"refund_approved",
	}
	for _, slug := range included {
		if !IsStatusOnlySlug(slug) {
			t.Errorf("expected %q to be a status-only slug", slug)
		}
	}
}

func TestIsStatusOnlySlug_Excluded(t *testing.T) {
	excluded := []string{
		"arbitration_opened",
		"review_posted",
		"some_unknown_slug",
		"",
	}
	for _, slug := range excluded {
		if IsStatusOnlySlug(slug) {
			t.Errorf("expected %q to NOT be a status-only slug", slug)
		}
	}
}

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     string
		params   map[string]string
		expected string
	}{
		{
			name:     "simple substitution",
			tmpl:     "Order {{.OrderID}} confirmed",
			params:   map[string]string{"OrderID": "abc-123"},
			expected: "Order abc-123 confirmed",
		},
		{
			name:     "multiple substitutions",
			tmpl:     "{{.OrderID}} by {{.Reason}}",
			params:   map[string]string{"OrderID": "X", "Reason": "Y"},
			expected: "X by Y",
		},
		{
			name:     "no params",
			tmpl:     "No placeholders here",
			params:   map[string]string{},
			expected: "No placeholders here",
		},
		{
			name:     "missing param left as-is",
			tmpl:     "Order {{.MissingField}} status",
			params:   map[string]string{"OrderID": "abc"},
			expected: "Order {{.MissingField}} status",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderTemplate(tc.tmpl, tc.params)
			if got != tc.expected {
				t.Errorf("renderTemplate(%q, %v) = %q, want %q", tc.tmpl, tc.params, got, tc.expected)
			}
		})
	}
}
