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

// TestMapError_SentinelToHTTPStatus exercises the real mapError function
// from helpers.go, verifying that each sentinel error produces the correct
// HTTP status code. This test replaces the JS constant-map tests.
func TestMapError_SentinelToHTTPStatus(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{dto.ErrNotFound, 404, dto.CodeNotFound},
		{dto.ErrForbidden, 403, dto.CodeForbidden},
		{dto.ErrUnauthorized, 401, dto.CodeUnauthorized},
		{dto.ErrConflict, 409, dto.CodeConflict},
		{dto.ErrRateLimited, 429, dto.CodeRateLimited},
		{dto.ErrValidation, 422, dto.CodeValidation},
		{dto.ErrAccountLocked, 423, dto.CodeAccountLocked},
		{dto.ErrInvalidCredentials, 401, dto.CodeInvalidCredentials},
		{dto.ErrAttachmentTooLarge, 413, dto.CodeAttachmentTooLarge},
		{dto.ErrDuplicateOrder, 409, dto.CodeDuplicateOrder},
		{dto.ErrInvalidTransition, 422, dto.CodeInvalidTransition},
		{dto.ErrOversold, 409, dto.CodeOversold},
	}

	for _, tc := range tests {
		t.Run(tc.wantCode, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = mapError(c, tc.err)

			if rec.Code != tc.wantStatus {
				t.Errorf("mapError(%v): got status %d, want %d", tc.err, rec.Code, tc.wantStatus)
			}

			var body dto.ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}
			if body.Error.Code != tc.wantCode {
				t.Errorf("mapError(%v): got code %q, want %q", tc.err, body.Error.Code, tc.wantCode)
			}
		})
	}
}

// TestMapError_UnknownError_Returns500 verifies the catch-all path.
func TestMapError_UnknownError_Returns500(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = mapError(c, &customError{msg: "something unexpected"})

	if rec.Code != 500 {
		t.Errorf("unknown error: got status %d, want 500", rec.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body.Error.Code != dto.CodeInternal {
		t.Errorf("unknown error: got code %q, want %q", body.Error.Code, dto.CodeInternal)
	}
}

type customError struct{ msg string }

func (e *customError) Error() string { return e.msg }

// TestErrorResponse_JSONShape validates the actual JSON shape returned by
// errorResponse — the production function from helpers.go.
func TestErrorResponse_JSONShape(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("request_id", "test-req-123")

	_ = errorResponse(c, http.StatusNotFound, dto.CodeNotFound, "user not found")

	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var body dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body.Error.Code != dto.CodeNotFound {
		t.Errorf("code: got %q, want %q", body.Error.Code, dto.CodeNotFound)
	}
	if body.Error.Message != "user not found" {
		t.Errorf("message: got %q, want %q", body.Error.Message, "user not found")
	}
}

// TestBindAndValidate_RejectsInvalidBody tests the real bindAndValidate
// function against a production DTO.
func TestBindAndValidate_RejectsInvalidBody(t *testing.T) {
	e := echo.New()
	// Missing required fields
	body := `{"username":"ab"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var dtoReq dto.CreateUserRequest
	err := bindAndValidate(c, &dtoReq)

	// Should have returned an error response (422)
	if err == nil && rec.Code != http.StatusUnprocessableEntity {
		t.Fatal("expected validation error for incomplete request")
	}
}

// TestBindAndValidate_AcceptsValidBody tests happy path.
func TestBindAndValidate_AcceptsValidBody(t *testing.T) {
	e := echo.New()
	body := `{"username":"alice","password":"password123","display_name":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var dtoReq dto.CreateUserRequest
	err := bindAndValidate(c, &dtoReq)

	if err != nil {
		t.Fatalf("expected valid, got error; response status: %d", rec.Code)
	}
	if dtoReq.Username != "alice" {
		t.Errorf("username: got %q, want %q", dtoReq.Username, "alice")
	}
}

// TestPagination_Defaults exercises the real pagination() helper.
func TestPagination_Defaults(t *testing.T) {
	tests := []struct {
		page, pageSize         string
		wantPage, wantPageSize int
	}{
		{"", "", 1, 20},
		{"0", "0", 1, 20},
		{"-5", "10", 1, 10},
		{"1", "101", 1, 20},
		{"3", "50", 3, 50},
		{"1", "100", 1, 100},
	}

	for _, tc := range tests {
		e := echo.New()
		q := "?"
		if tc.page != "" {
			q += "page=" + tc.page + "&"
		}
		if tc.pageSize != "" {
			q += "page_size=" + tc.pageSize
		}
		req := httptest.NewRequest(http.MethodGet, "/"+q, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		page, pageSize := pagination(c)
		if page != tc.wantPage || pageSize != tc.wantPageSize {
			t.Errorf("pagination(%q, %q) = (%d, %d), want (%d, %d)",
				tc.page, tc.pageSize, page, pageSize, tc.wantPage, tc.wantPageSize)
		}
	}
}

// TestPaginatedResponse_TotalPages exercises the real paginatedResponse helper.
func TestPaginatedResponse_TotalPages(t *testing.T) {
	tests := []struct {
		total, size, wantPages int
	}{
		{100, 20, 5},
		{101, 20, 6},
		{5, 20, 1},
		{0, 20, 0},
	}
	for _, tc := range tests {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = paginatedResponse(c, []string{}, 1, tc.size, tc.total)

		var body dto.PaginatedResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode failed for total=%d size=%d: %v", tc.total, tc.size, err)
		}
		if body.TotalPages != tc.wantPages {
			t.Errorf("total=%d size=%d: got %d pages, want %d", tc.total, tc.size, body.TotalPages, tc.wantPages)
		}
	}
}
