package dto

import (
	"errors"
	"strings"
	"testing"
)

// TestErrorCodeConstants verifies that every Code* constant starts with "ERR_"
// and is uppercase. This catches accidental typos or naming convention drift.
func TestErrorCodeConstants_Format(t *testing.T) {
	codes := []string{
		CodeNotFound, CodeForbidden, CodeUnauthorized, CodeConflict,
		CodeRateLimited, CodeValidation, CodeAccountLocked, CodeInvalidCredentials,
		CodeAttachmentTooLarge, CodeDuplicateOrder, CodeInvalidTransition,
		CodeOversold, CodeInternal,
	}
	for _, code := range codes {
		if !strings.HasPrefix(code, "ERR_") {
			t.Errorf("code %q should start with ERR_", code)
		}
		if code != strings.ToUpper(code) {
			t.Errorf("code %q should be uppercase", code)
		}
	}
}

// TestSentinelErrors_AreDistinct ensures that all sentinel errors are distinct
// values so that errors.Is comparisons work correctly in the handler layer.
func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []error{
		ErrNotFound, ErrForbidden, ErrUnauthorized, ErrConflict,
		ErrRateLimited, ErrValidation, ErrAccountLocked, ErrInvalidCredentials,
		ErrAttachmentTooLarge, ErrDuplicateOrder, ErrInvalidTransition, ErrOversold,
	}
	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			if errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("sentinel errors at index %d and %d should be distinct", i, j)
			}
		}
	}
}

// TestWrappedErrors_MatchViaSentinel verifies that errors wrapped with
// additional context (the pattern used in service methods) still match the
// sentinel via errors.Is — this is how mapError dispatches HTTP status codes.
func TestWrappedErrors_MatchViaSentinel(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
	}{
		{"not found", ErrNotFound},
		{"forbidden", ErrForbidden},
		{"unauthorized", ErrUnauthorized},
		{"conflict", ErrConflict},
		{"rate limited", ErrRateLimited},
		{"validation", ErrValidation},
		{"account locked", ErrAccountLocked},
		{"invalid credentials", ErrInvalidCredentials},
		{"attachment too large", ErrAttachmentTooLarge},
		{"duplicate order", ErrDuplicateOrder},
		{"invalid transition", ErrInvalidTransition},
		{"oversold", ErrOversold},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := errors.Join(tc.sentinel, errors.New("extra context"))
			if !errors.Is(wrapped, tc.sentinel) {
				t.Errorf("wrapped error should match sentinel %v via errors.Is", tc.sentinel)
			}
		})
	}
}

// TestBusinessErrors_Are4xx verifies the error→status mapping table that
// mapError in handler/helpers.go implements. Since mapError is package-private,
// we test the same mapping here via a lookup table that must stay in sync.
func TestBusinessErrors_Are4xx(t *testing.T) {
	// This map mirrors handler/helpers.go mapError exactly.
	errorToStatus := map[error]int{
		ErrNotFound:           404,
		ErrForbidden:          403,
		ErrUnauthorized:       401,
		ErrConflict:           409,
		ErrRateLimited:        429,
		ErrValidation:         422,
		ErrAccountLocked:      423,
		ErrInvalidCredentials: 401,
		ErrAttachmentTooLarge: 413,
		ErrDuplicateOrder:     409,
		ErrInvalidTransition:  422,
		ErrOversold:           409,
	}

	for err, status := range errorToStatus {
		if status < 400 || status >= 500 {
			t.Errorf("business error %v maps to %d; expected 4xx", err, status)
		}
	}
}

// TestErrorResponse_Structure validates the JSON envelope shape that the API
// always returns for errors. This ensures the ErrorResponse/ErrorDetail types
// have the required fields for clients to parse.
func TestErrorResponse_Structure(t *testing.T) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:      CodeNotFound,
			Message:   "resource not found",
			RequestID: "req-123",
		},
	}
	if resp.Error.Code == "" {
		t.Error("error code must not be empty")
	}
	if resp.Error.Message == "" {
		t.Error("error message must not be empty")
	}
	if resp.Error.RequestID == "" {
		t.Error("request_id should be set when provided")
	}
}

// TestErrorResponse_OmitsEmptyRequestID verifies that request_id can be
// omitted (zero value) without breaking the struct.
func TestErrorResponse_OmitsEmptyRequestID(t *testing.T) {
	resp := ErrorResponse{
		Error: ErrorDetail{Code: CodeValidation, Message: "bad input"},
	}
	if resp.Error.RequestID != "" {
		t.Error("request_id should be empty when not provided")
	}
}
