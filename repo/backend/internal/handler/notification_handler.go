package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/service"
)

type NotificationHandler struct {
	notifService *service.NotificationService
}

func NewNotificationHandler(ns *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: ns}
}

func (h *NotificationHandler) List(c echo.Context) error {
	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	unreadOnly := c.QueryParam("unread") == "true"
	countOnly := c.QueryParam("count") == "true"

	if countOnly {
		count, err := h.notifService.CountUnread(c.Request().Context(), userID)
		if err != nil {
			return mapError(c, err)
		}
		return c.JSON(http.StatusOK, map[string]int{"unread_count": count})
	}

	page, pageSize := pagination(c)
	notifications, total, err := h.notifService.List(c.Request().Context(), userID, unreadOnly, page, pageSize)
	if err != nil {
		return mapError(c, err)
	}
	return paginatedResponse(c, notifications, page, pageSize, total)
}

func (h *NotificationHandler) MarkRead(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid notification ID")
	}

	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	if err := h.notifService.MarkRead(c.Request().Context(), id, userID); err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *NotificationHandler) MarkAllRead(c echo.Context) error {
	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	if err := h.notifService.MarkAllRead(c.Request().Context(), userID); err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *NotificationHandler) Retry(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid notification ID")
	}

	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	if err := h.notifService.Retry(c.Request().Context(), id, userID); err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *NotificationHandler) GetPreferences(c echo.Context) error {
	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	prefs, err := h.notifService.GetPreferences(c.Request().Context(), userID)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, prefs)
}

func (h *NotificationHandler) UpdatePreferences(c echo.Context) error {
	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	var req dto.UpdateNotificationPrefsRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.notifService.UpdatePreferences(c.Request().Context(), userID, req.Preferences, req.SubscriptionMode); err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
