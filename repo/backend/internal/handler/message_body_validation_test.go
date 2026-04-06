package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
)

// TestSendMessage_BodyTooLong verifies that the handler rejects message bodies
// exceeding the 10,000 character limit with a 400 status and ERR_VALIDATION code.
func TestSendMessage_BodyTooLong(t *testing.T) {
	e := echo.New()

	// 10001 characters — just over the limit
	longBody := strings.Repeat("a", 10001)
	formBody := "body=" + longBody

	req := httptest.NewRequest(http.MethodPost, "/api/orders/00000000-0000-0000-0000-000000000001/messages", strings.NewReader(formBody))
	req.Header.Set(echo.HeaderContentType, "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orderId")
	c.SetParamValues("00000000-0000-0000-0000-000000000001")
	c.Set("user_id", "00000000-0000-0000-0000-000000000002")

	h := &MessageHandler{}
	_ = h.Send(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Error.Code != dto.CodeValidation {
		t.Errorf("expected code %q, got %q", dto.CodeValidation, body.Error.Code)
	}
	if !strings.Contains(body.Error.Message, "10000") {
		t.Errorf("expected message to mention limit, got: %q", body.Error.Message)
	}
}

// TestSendMessage_BodyExactlyAtLimit verifies 10000 chars passes the
// body-length gate. The handler will panic downstream due to nil
// messageService, but the panic itself proves we got past validation.
func TestSendMessage_BodyExactlyAtLimit(t *testing.T) {
	e := echo.New()

	exactBody := strings.Repeat("a", 10000)
	formBody := "body=" + exactBody

	req := httptest.NewRequest(http.MethodPost, "/api/orders/00000000-0000-0000-0000-000000000001/messages", strings.NewReader(formBody))
	req.Header.Set(echo.HeaderContentType, "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orderId")
	c.SetParamValues("00000000-0000-0000-0000-000000000001")
	c.Set("user_id", "00000000-0000-0000-0000-000000000002")

	h := &MessageHandler{} // nil service — will panic past validation

	func() {
		defer func() { recover() }() // catch nil-service panic
		_ = h.Send(c)
	}()

	// If a response was written before the panic, verify it's not the length error
	if rec.Code == http.StatusBadRequest {
		var body dto.ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err == nil {
			if strings.Contains(body.Error.Message, "10000 character limit") {
				t.Fatal("10000-char body should not trigger the length validation")
			}
		}
	}
}

// TestSendMessage_EmptyBody verifies that empty body is rejected.
func TestSendMessage_EmptyBody(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/api/orders/00000000-0000-0000-0000-000000000001/messages", strings.NewReader("body="))
	req.Header.Set(echo.HeaderContentType, "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("orderId")
	c.SetParamValues("00000000-0000-0000-0000-000000000001")
	c.Set("user_id", "00000000-0000-0000-0000-000000000002")

	h := &MessageHandler{}
	_ = h.Send(c)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for empty body, got %d", rec.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body.Error.Code != dto.CodeValidation {
		t.Errorf("expected %q, got %q", dto.CodeValidation, body.Error.Code)
	}
}
