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

type CollectibleHandler struct {
	collectibleService *service.CollectibleService
	auditStore         *store.AuditStore
}

func NewCollectibleHandler(cs *service.CollectibleService, audit *store.AuditStore) *CollectibleHandler {
	return &CollectibleHandler{collectibleService: cs, auditStore: audit}
}

func (h *CollectibleHandler) List(c echo.Context) error {
	status := c.QueryParam("status")
	if status == "" {
		status = "published"
	}
	page, pageSize := pagination(c)
	roles := middleware.GetUserRoles(c)

	var userID *uuid.UUID
	if uid, err := uuid.Parse(middleware.GetUserID(c)); err == nil {
		userID = &uid
	}

	collectibles, total, err := h.collectibleService.List(c.Request().Context(), status, page, pageSize, roles, userID)
	if err != nil {
		return mapError(c, err)
	}
	return paginatedResponse(c, collectibles, page, pageSize, total)
}

func (h *CollectibleHandler) ListMine(c echo.Context) error {
	sellerID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}
	page, pageSize := pagination(c)

	collectibles, total, err := h.collectibleService.ListBySeller(c.Request().Context(), sellerID, page, pageSize)
	if err != nil {
		return mapError(c, err)
	}
	return paginatedResponse(c, collectibles, page, pageSize, total)
}

func (h *CollectibleHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid collectible ID")
	}

	roles := middleware.GetUserRoles(c)
	collectible, err := h.collectibleService.GetByID(c.Request().Context(), id, roles)
	if err != nil {
		return mapError(c, err)
	}

	// Emit item_view analytics event
	userIDStr := middleware.GetUserID(c)
	if uid, parseErr := uuid.Parse(userIDStr); parseErr == nil {
		h.collectibleService.EmitViewEvent(c.Request().Context(), &uid, id)
	}

	txHistory, _ := h.collectibleService.GetTxHistory(c.Request().Context(), id)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"collectible":         collectible,
		"transaction_history": txHistory,
	})
}

func (h *CollectibleHandler) Create(c echo.Context) error {
	var req dto.CreateCollectibleRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	sellerID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	collectible, err := h.collectibleService.Create(c.Request().Context(), req, sellerID)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusCreated, collectible)
}

func (h *CollectibleHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid collectible ID")
	}

	var req dto.UpdateCollectibleRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	actorID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	collectible, err := h.collectibleService.Update(c.Request().Context(), id, req, actorID)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, collectible)
}

func (h *CollectibleHandler) Hide(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid collectible ID")
	}

	var req dto.HideCollectibleRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	adminID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	if err := h.collectibleService.Hide(c.Request().Context(), id, req.Reason, adminID); err != nil {
		return mapError(c, err)
	}
	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &adminID, model.AuditActionCollectHide, "collectible", &id,
			map[string]interface{}{"reason": req.Reason}, c.RealIP())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *CollectibleHandler) PostReview(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid collectible ID")
	}

	var req dto.PostReviewRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	reviewerID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	if err := h.collectibleService.PostReview(c.Request().Context(), id, reviewerID, req.Rating, req.Body); err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": "ok"})
}

func (h *CollectibleHandler) Publish(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid collectible ID")
	}

	if err := h.collectibleService.Publish(c.Request().Context(), id); err != nil {
		return mapError(c, err)
	}
	if h.auditStore != nil {
		actorID, _ := uuid.Parse(middleware.GetUserID(c))
		h.auditStore.LogEvent(c.Request().Context(), &actorID, model.AuditActionCollectPublish, "collectible", &id, nil, c.RealIP())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
