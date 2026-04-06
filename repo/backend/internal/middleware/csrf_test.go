package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCSRF_SkipsGetRequests(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatalf("GET should be allowed without CSRF: %v", err)
	}
}

func TestCSRF_SkipsLogin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatalf("login should skip CSRF: %v", err)
	}
}

func TestCSRF_SkipsSetupAdmin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/setup/admin", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatalf("POST /api/setup/admin should skip CSRF: %v", err)
	}
}

func TestCSRF_EnforcedOnProtectedPost(t *testing.T) {
	// Verify that a non-exempt POST endpoint still requires CSRF.
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/read-all", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	if err == nil {
		t.Fatal("protected POST without CSRF token should be rejected")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got: %v", err)
	}
}

func TestCSRF_SkipsRefresh(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatalf("refresh should skip CSRF (uses its own cookie auth): %v", err)
	}
}

func TestCSRF_ValidToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/orders", nil)
	req.Header.Set("X-CSRF-Token", "test-token-123")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-token-123"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if err := handler(c); err != nil {
		t.Fatalf("valid CSRF should pass: %v", err)
	}
}

func TestCSRF_MismatchToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/orders", nil)
	req.Header.Set("X-CSRF-Token", "wrong-token")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "correct-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	if err == nil {
		t.Fatal("mismatched CSRF token should be rejected")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got: %v", err)
	}
}

// TestCSRF_AllStateChangingEndpoints exhaustively verifies that every
// state-changing endpoint (POST, PUT, PATCH, DELETE) is rejected when
// no CSRF token is provided — except for the two explicitly justified
// exemptions (login and bootstrap). This prevents regressions where a
// new route silently bypasses CSRF protection.
func TestCSRF_AllStateChangingEndpoints(t *testing.T) {
	// Every state-changing route registered in router.Setup, grouped by
	// whether CSRF enforcement is expected.
	type route struct {
		method string
		path   string
	}

	// These endpoints are exempt by design (pre-session or cookie-auth).
	exempted := []route{
		{http.MethodPost, "/api/auth/login"},
		{http.MethodPost, "/api/auth/refresh"},
		{http.MethodPost, "/api/setup/admin"},
	}

	// Every other POST/PUT/PATCH/DELETE endpoint MUST enforce CSRF.
	protected := []route{
		// Auth
		{http.MethodPost, "/api/auth/logout"},

		// Users (admin)
		{http.MethodPost, "/api/users"},
		{http.MethodPatch, "/api/users/1"},
		{http.MethodPost, "/api/users/1/roles"},
		{http.MethodDelete, "/api/users/1/roles/1"},
		{http.MethodPost, "/api/users/1/unlock"},

		// Collectibles
		{http.MethodPost, "/api/collectibles"},
		{http.MethodPatch, "/api/collectibles/1"},
		{http.MethodPost, "/api/collectibles/1/reviews"},
		{http.MethodPatch, "/api/collectibles/1/hide"},
		{http.MethodPatch, "/api/collectibles/1/publish"},

		// Orders
		{http.MethodPost, "/api/orders"},
		{http.MethodPost, "/api/orders/1/confirm"},
		{http.MethodPost, "/api/orders/1/process"},
		{http.MethodPost, "/api/orders/1/complete"},
		{http.MethodPost, "/api/orders/1/cancel"},
		{http.MethodPost, "/api/orders/1/refund"},
		{http.MethodPost, "/api/orders/1/arbitration"},
		{http.MethodPatch, "/api/orders/1/fulfillment"},

		// Messages
		{http.MethodPost, "/api/orders/1/messages"},

		// Notifications
		{http.MethodPatch, "/api/notifications/1/read"},
		{http.MethodPost, "/api/notifications/read-all"},
		{http.MethodPost, "/api/notifications/1/retry"},
		{http.MethodPut, "/api/notifications/preferences"},

		// A/B Tests
		{http.MethodPost, "/api/ab-tests"},
		{http.MethodPatch, "/api/ab-tests/1"},
		{http.MethodPost, "/api/ab-tests/1/complete"},
		{http.MethodPost, "/api/ab-tests/1/rollback"},

		// Admin
		{http.MethodPost, "/api/admin/ip-rules"},
		{http.MethodDelete, "/api/admin/ip-rules/1"},
		{http.MethodPatch, "/api/admin/anomalies/1/acknowledge"},
	}

	csrfMiddleware := CSRF()

	// Exempted endpoints must pass without any CSRF token.
	for _, r := range exempted {
		t.Run("exempt_"+r.method+"_"+r.path, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(r.method, r.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := csrfMiddleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})
			if err := h(c); err != nil {
				t.Fatalf("%s %s should be CSRF-exempt but was rejected: %v", r.method, r.path, err)
			}
		})
	}

	// Protected endpoints must be rejected (403) without a CSRF token.
	for _, r := range protected {
		t.Run("enforced_"+r.method+"_"+r.path, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(r.method, r.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := csrfMiddleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})
			err := h(c)
			if err == nil {
				t.Fatalf("%s %s should require CSRF but was allowed without token", r.method, r.path)
			}
			he, ok := err.(*echo.HTTPError)
			if !ok || he.Code != http.StatusForbidden {
				t.Fatalf("%s %s: expected 403 Forbidden, got: %v", r.method, r.path, err)
			}
		})
	}

	// Protected endpoints must pass when a valid CSRF token is provided.
	for _, r := range protected {
		t.Run("valid_"+r.method+"_"+r.path, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(r.method, r.path, nil)
			req.Header.Set("X-CSRF-Token", "valid-token")
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-token"})
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := csrfMiddleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})
			if err := h(c); err != nil {
				t.Fatalf("%s %s should pass with valid CSRF token: %v", r.method, r.path, err)
			}
		})
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	token, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token) != 64 { // 32 bytes → 64 hex chars
		t.Fatalf("expected 64 char hex token, got %d chars", len(token))
	}

	// Tokens should be unique
	token2, _ := GenerateCSRFToken()
	if token == token2 {
		t.Fatal("tokens should be unique")
	}
}
