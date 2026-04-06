package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
)

// GlobalErrorHandler converts all Echo errors into consistent JSON format.
// This ensures no raw framework errors (HTML or plain text) leak to clients.
func GlobalErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	errCode := dto.CodeInternal
	message := "internal server error"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code

		// If the error message is already our ErrorResponse format, use it directly
		if resp, ok := he.Message.(dto.ErrorResponse); ok {
			c.JSON(code, resp)
			return
		}

		// Map common HTTP status codes to our error codes
		switch code {
		case http.StatusBadRequest:
			errCode = dto.CodeValidation
			message = "bad request"
		case http.StatusUnauthorized:
			errCode = dto.CodeUnauthorized
			message = "unauthorized"
		case http.StatusForbidden:
			errCode = dto.CodeForbidden
			message = "forbidden"
		case http.StatusNotFound:
			errCode = dto.CodeNotFound
			message = "not found"
		case http.StatusMethodNotAllowed:
			errCode = dto.CodeValidation
			message = "method not allowed"
		case http.StatusTooManyRequests:
			errCode = dto.CodeRateLimited
			message = "rate limited"
		case http.StatusServiceUnavailable:
			errCode = dto.CodeInternal
			message = "service unavailable"
		}

		// Use the echo error message if it's a string
		if msg, ok := he.Message.(string); ok && msg != "" {
			message = msg
		}
	}

	c.JSON(code, dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:      errCode,
			Message:   message,
			RequestID: GetRequestID(c),
		},
	})
}
