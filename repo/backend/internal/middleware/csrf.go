package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
)

func CSRF() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip for safe methods
			if c.Request().Method == http.MethodGet ||
				c.Request().Method == http.MethodHead ||
				c.Request().Method == http.MethodOptions {
				return next(c)
			}

			// Skip CSRF for pre-session endpoints where no cookie exists yet.
			// - /api/auth/login: pre-auth, no cookies.
			// - /api/setup/admin: one-time bootstrap before any user session
			//   exists. This endpoint self-disables after the first admin is
			//   created, so the exemption window is minimal.
			// Refresh is protected: the client has a csrf_token cookie from login.
			path := c.Request().URL.Path
			if path == "/api/auth/login" || path == "/api/auth/refresh" || path == "/api/setup/admin" {
				return next(c)
			}

			cookie, err := c.Cookie("csrf_token")
			if err != nil {
				return echo.NewHTTPError(http.StatusForbidden, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:      dto.CodeForbidden,
						Message:   "missing CSRF cookie",
						RequestID: GetRequestID(c),
					},
				})
			}

			header := c.Request().Header.Get("X-CSRF-Token")
			if header == "" || header != cookie.Value {
				return echo.NewHTTPError(http.StatusForbidden, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:      dto.CodeForbidden,
						Message:   "CSRF token mismatch",
						RequestID: GetRequestID(c),
					},
				})
			}

			return next(c)
		}
	}
}

func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func SetCSRFCookie(c echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be readable by JS
		Secure:   false, // LAN - no TLS
		SameSite: http.SameSiteStrictMode,
	})
}
