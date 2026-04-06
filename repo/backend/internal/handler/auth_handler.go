package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/service"
	"github.com/ledgermint/platform/internal/store"
)

type AuthHandler struct {
	authService *service.AuthService
	auditStore  *store.AuditStore
}

func NewAuthHandler(as *service.AuthService, audit *store.AuditStore) *AuthHandler {
	return &AuthHandler{authService: as, auditStore: audit}
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	resp, cookies, err := h.authService.Login(c.Request().Context(), req, c.RealIP())
	if err != nil {
		if h.auditStore != nil {
			h.auditStore.LogEvent(c.Request().Context(), nil, model.AuditActionLoginFailed, "session", nil,
				map[string]interface{}{"username": req.Username}, c.RealIP())
		}
		return mapError(c, err)
	}

	for _, cookie := range cookies {
		c.SetCookie(cookie)
	}

	if h.auditStore != nil {
		uid := resp.User.ID
		h.auditStore.LogEvent(c.Request().Context(), &uid, model.AuditActionLogin, "session", nil,
			map[string]interface{}{"username": req.Username}, c.RealIP())
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "missing refresh token")
	}

	cookies, err := h.authService.Refresh(c.Request().Context(), cookie.Value)
	if err != nil {
		return mapError(c, err)
	}

	for _, cookie := range cookies {
		c.SetCookie(cookie)
	}

	if h.auditStore != nil {
		if uid, parseErr := uuid.Parse(middleware.GetUserID(c)); parseErr == nil {
			h.auditStore.LogEvent(c.Request().Context(), &uid, model.AuditActionTokenRefresh, "session", nil, nil, c.RealIP())
		}
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Me(c echo.Context) error {
	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	resp, err := h.authService.GetCurrentUser(c.Request().Context(), userID)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c echo.Context) error {
	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	cookies := h.authService.Logout(c.Request().Context(), userID)
	for _, cookie := range cookies {
		c.SetCookie(cookie)
	}

	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &userID, model.AuditActionLogout, "session", nil, nil, c.RealIP())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
