package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func TestUpdateNotificationPrefsRequest_SubscriptionModeValidation(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{"all_events is valid", "all_events", false},
		{"status_only is valid", "status_only", false},
		{"empty is valid (omitempty)", "", false},
		{"invalid value rejected", "weekly_digest", true},
		{"random string rejected", "foo", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := UpdateNotificationPrefsRequest{
				Preferences:      map[string]bool{"order_confirmed": true},
				SubscriptionMode: tc.mode,
			}
			err := validate.Struct(req)
			if tc.wantErr && err == nil {
				t.Errorf("expected validation error for mode %q", tc.mode)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected validation error for mode %q: %v", tc.mode, err)
			}
		})
	}
}

func TestUpdateNotificationPrefsRequest_PreferencesRequired(t *testing.T) {
	req := UpdateNotificationPrefsRequest{
		SubscriptionMode: "all_events",
	}
	err := validate.Struct(req)
	if err == nil {
		t.Error("expected validation error when preferences is nil")
	}
}
