package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// setTestLogger stores the observer-backed logger in the Echo context so that
// StructuredLogger picks it up via Logger(c) instead of creating a production logger.
func setTestLogger(c echo.Context, logger *zap.Logger) {
	c.Set(loggerKey, logger)
}

// ---------------------------------------------------------------------------
// Sensitive field redaction tests
// ---------------------------------------------------------------------------

func TestRedactSensitive_PasswordRedacted(t *testing.T) {
	got := RedactSensitive("password", "super-secret-123")
	if got != "[REDACTED]" {
		t.Errorf("password should be redacted, got %q", got)
	}
}

func TestRedactSensitive_TokenRedacted(t *testing.T) {
	got := RedactSensitive("token", "eyJhbGciOiJIUzI1NiJ9.xxxx")
	if got != "[REDACTED]" {
		t.Errorf("token should be redacted, got %q", got)
	}
}

func TestRedactSensitive_AuthorizationRedacted(t *testing.T) {
	got := RedactSensitive("Authorization", "Bearer eyJhbGciOiJIUzI1NiJ9.xxxx")
	if got != "[REDACTED]" {
		t.Errorf("authorization should be redacted, got %q", got)
	}
}

func TestRedactSensitive_CookieRedacted(t *testing.T) {
	got := RedactSensitive("Cookie", "access_token=eyJ...")
	if got != "[REDACTED]" {
		t.Errorf("cookie should be redacted, got %q", got)
	}
}

func TestRedactSensitive_SecretRedacted(t *testing.T) {
	got := RedactSensitive("secret", "my-api-secret")
	if got != "[REDACTED]" {
		t.Errorf("secret should be redacted, got %q", got)
	}
}

func TestRedactSensitive_CaseInsensitive(t *testing.T) {
	tests := []string{"PASSWORD", "Password", "pAssWoRd"}
	for _, key := range tests {
		got := RedactSensitive(key, "value")
		if got != "[REDACTED]" {
			t.Errorf("RedactSensitive(%q) should be redacted, got %q", key, got)
		}
	}
}

func TestRedactSensitive_SafeFieldNotRedacted(t *testing.T) {
	safeFields := []string{"user_id", "method", "path", "status", "username", "email"}
	for _, key := range safeFields {
		got := RedactSensitive(key, "some-value")
		if got == "[REDACTED]" {
			t.Errorf("safe field %q should not be redacted", key)
		}
		if got != "some-value" {
			t.Errorf("RedactSensitive(%q) = %q, want %q", key, got, "some-value")
		}
	}
}

// ---------------------------------------------------------------------------
// Structured logger tests — verify no sensitive data leaks to logs
// ---------------------------------------------------------------------------

func TestStructuredLogger_DoesNotLogRequestBody(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)

	e := echo.New()
	mw := StructuredLogger(testLogger)

	// Simulate a login request with sensitive body
	body := `{"username":"alice","password":"super-secret-password-123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("request_id", "test-123")
	setTestLogger(c, testLogger)

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = handler(c)

	// Verify the log entry exists
	if logs.Len() == 0 {
		t.Fatal("expected at least one log entry")
	}

	// Verify no sensitive data leaked into log fields
	for _, entry := range logs.All() {
		msg := entry.Message
		if strings.Contains(msg, "super-secret-password-123") {
			t.Error("password leaked into log message")
		}

		for _, field := range entry.Context {
			val := field.String
			if strings.Contains(val, "super-secret-password-123") {
				t.Errorf("password leaked into log field %q", field.Key)
			}
		}
	}
}

func TestStructuredLogger_LogsExpectedFields(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)

	e := echo.New()
	mw := StructuredLogger(testLogger)

	req := httptest.NewRequest(http.MethodGet, "/api/collectibles", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "user-123")
	setTestLogger(c, testLogger)

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	_ = handler(c)

	if logs.Len() == 0 {
		t.Fatal("expected log entry")
	}

	entry := logs.All()[0]
	fieldMap := map[string]bool{}
	for _, f := range entry.Context {
		fieldMap[f.Key] = true
	}

	required := []string{"method", "path", "status", "latency", "remote_ip"}
	for _, name := range required {
		if !fieldMap[name] {
			t.Errorf("missing expected log field: %q", name)
		}
	}
}

func TestStructuredLogger_ErrorStatusLogsAtErrorLevel(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core)

	e := echo.New()
	mw := StructuredLogger(testLogger)

	req := httptest.NewRequest(http.MethodGet, "/api/fail", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setTestLogger(c, testLogger)

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusInternalServerError, "error")
	})
	_ = handler(c)

	if logs.Len() == 0 {
		t.Fatal("expected log entry")
	}

	entry := logs.All()[0]
	if entry.Level != zapcore.ErrorLevel {
		t.Errorf("5xx should log at ERROR level, got %v", entry.Level)
	}
}

func TestStructuredLogger_ClientErrorLogsAtWarnLevel(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core)

	e := echo.New()
	mw := StructuredLogger(testLogger)

	req := httptest.NewRequest(http.MethodGet, "/api/missing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setTestLogger(c, testLogger)

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusNotFound, "not found")
	})
	_ = handler(c)

	if logs.Len() == 0 {
		t.Fatal("expected log entry")
	}

	entry := logs.All()[0]
	if entry.Level != zapcore.WarnLevel {
		t.Errorf("4xx should log at WARN level, got %v", entry.Level)
	}
}

// TestSensitiveFieldsExhaustive verifies all declared sensitive fields
// are properly redacted.
func TestSensitiveFieldsExhaustive(t *testing.T) {
	for field := range sensitiveFields {
		got := RedactSensitive(field, "value-that-should-not-appear")
		if got != "[REDACTED]" {
			t.Errorf("sensitive field %q was not redacted", field)
		}
	}
}
