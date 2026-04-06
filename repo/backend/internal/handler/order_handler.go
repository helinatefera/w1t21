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

type OrderHandler struct {
	orderService *service.OrderService
	auditStore   *store.AuditStore
}

func NewOrderHandler(os *service.OrderService, audit *store.AuditStore) *OrderHandler {
	return &OrderHandler{orderService: os, auditStore: audit}
}

func (h *OrderHandler) Create(c echo.Context) error {
	var req dto.CreateOrderRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	idempotencyKey := c.Request().Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "Idempotency-Key header is required")
	}

	buyerID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	order, err := h.orderService.Create(c.Request().Context(), req, buyerID, idempotencyKey)
	if err != nil {
		return mapError(c, err)
	}
	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &buyerID, model.AuditActionOrderCreate, "order", &order.ID,
			map[string]interface{}{"collectible_id": req.CollectibleID.String()}, c.RealIP())
	}
	return c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid order ID")
	}

	actorID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	order, err := h.orderService.GetByID(c.Request().Context(), id, actorID)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) List(c echo.Context) error {
	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	page, pageSize := pagination(c)
	role := c.QueryParam("role")

	var orders []model.Order
	var total int

	if role == "seller" {
		orders, total, err = h.orderService.ListBySeller(c.Request().Context(), userID, page, pageSize)
	} else {
		orders, total, err = h.orderService.ListByBuyer(c.Request().Context(), userID, page, pageSize)
	}
	if err != nil {
		return mapError(c, err)
	}
	return paginatedResponse(c, orders, page, pageSize, total)
}

func (h *OrderHandler) Confirm(c echo.Context) error {
	return h.transition(c, model.OrderStatusConfirmed)
}

func (h *OrderHandler) Process(c echo.Context) error {
	return h.transition(c, model.OrderStatusProcessing)
}

func (h *OrderHandler) Complete(c echo.Context) error {
	return h.transition(c, model.OrderStatusCompleted)
}

func (h *OrderHandler) Cancel(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid order ID")
	}

	var req dto.CancelOrderRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	actorID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	order, err := h.orderService.TransitionStatus(c.Request().Context(), id, model.OrderStatusCancelled, actorID, req.Reason)
	if err != nil {
		return mapError(c, err)
	}
	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &actorID, model.AuditActionOrderTransit, "order", &id,
			map[string]interface{}{"to_status": "cancelled", "reason": req.Reason}, c.RealIP())
	}
	return c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) UpdateFulfillment(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid order ID")
	}

	var req dto.UpdateFulfillmentRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	actorID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	if err := h.orderService.UpdateFulfillment(c.Request().Context(), id, req, actorID); err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *OrderHandler) ApproveRefund(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid order ID")
	}

	var req dto.ApproveRefundRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	actorID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	order, err := h.orderService.ApproveRefund(c.Request().Context(), id, actorID, req.Reason)
	if err != nil {
		return mapError(c, err)
	}
	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &actorID, "order.refund_approved", "order", &id,
			map[string]interface{}{"reason": req.Reason}, c.RealIP())
	}
	return c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) OpenArbitration(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid order ID")
	}

	var req dto.OpenArbitrationRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	actorID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	order, err := h.orderService.OpenArbitration(c.Request().Context(), id, actorID, req.Reason)
	if err != nil {
		return mapError(c, err)
	}
	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &actorID, "order.arbitration_opened", "order", &id,
			map[string]interface{}{"reason": req.Reason}, c.RealIP())
	}
	return c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) transition(c echo.Context, status model.OrderStatus) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid order ID")
	}

	actorID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	order, err := h.orderService.TransitionStatus(c.Request().Context(), id, status, actorID, "")
	if err != nil {
		return mapError(c, err)
	}
	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &actorID, model.AuditActionOrderTransit, "order", &id,
			map[string]interface{}{"to_status": string(status)}, c.RealIP())
	}
	return c.JSON(http.StatusOK, order)
}
