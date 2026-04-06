package dto

import "errors"

// Sentinel errors mapped to HTTP status codes by the central error handler.
var (
	ErrNotFound           = errors.New("not found")
	ErrForbidden          = errors.New("forbidden")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrConflict           = errors.New("conflict")
	ErrRateLimited        = errors.New("rate limited")
	ErrValidation         = errors.New("validation failed")
	ErrAccountLocked      = errors.New("account locked")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAttachmentTooLarge = errors.New("attachment too large")
	ErrDuplicateOrder     = errors.New("duplicate order")
	ErrInvalidTransition  = errors.New("invalid state transition")
	ErrOversold           = errors.New("collectible already has an active order")
)

// Error codes returned in API responses.
const (
	CodeNotFound           = "ERR_NOT_FOUND"
	CodeForbidden          = "ERR_FORBIDDEN"
	CodeUnauthorized       = "ERR_UNAUTHORIZED"
	CodeConflict           = "ERR_CONFLICT"
	CodeRateLimited        = "ERR_RATE_LIMITED"
	CodeValidation         = "ERR_VALIDATION"
	CodeAccountLocked      = "ERR_ACCOUNT_LOCKED"
	CodeInvalidCredentials = "ERR_INVALID_CREDENTIALS"
	CodeAttachmentTooLarge = "ERR_ATTACHMENT_TOO_LARGE"
	CodeDuplicateOrder     = "ERR_ORDER_DUPLICATE"
	CodeInvalidTransition  = "ERR_INVALID_TRANSITION"
	CodeOversold           = "ERR_OVERSOLD"
	CodeInternal           = "ERR_INTERNAL"
	CodeSetupRequired      = "ERR_SETUP_REQUIRED"
)

// ErrorResponse is the unified API error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}
