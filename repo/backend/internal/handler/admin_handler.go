package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/cache"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
)

type AdminHandler struct {
	userStore      *store.UserStore
	analyticsStore *store.AnalyticsStore
	cache          *cache.HotCache
}

func NewAdminHandler(us *store.UserStore, as *store.AnalyticsStore, c *cache.HotCache) *AdminHandler {
	return &AdminHandler{userStore: us, analyticsStore: as, cache: c}
}

func (h *AdminHandler) ListIPRules(c echo.Context) error {
	rules, err := h.userStore.GetAllIPRules()
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, rules)
}

func (h *AdminHandler) CreateIPRule(c echo.Context) error {
	var req dto.CreateIPRuleRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	createdBy, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	rule := &model.IPRule{
		CIDR:      req.CIDR,
		Action:    req.Action,
		CreatedBy: createdBy,
	}

	if err := h.userStore.CreateIPRule(c.Request().Context(), rule); err != nil {
		return mapError(c, err)
	}

	// Invalidate IP rules cache
	h.cache.Delete("ip_rules")

	return c.JSON(http.StatusCreated, rule)
}

func (h *AdminHandler) DeleteIPRule(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid IP rule ID")
	}

	if err := h.userStore.DeleteIPRule(c.Request().Context(), id); err != nil {
		return mapError(c, err)
	}

	h.cache.Delete("ip_rules")
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminHandler) ListAnomalies(c echo.Context) error {
	page, pageSize := pagination(c)

	var ackFilter *bool
	if ack := c.QueryParam("acknowledged"); ack != "" {
		v := ack == "true"
		ackFilter = &v
	}

	events, total, err := h.analyticsStore.ListAnomalyEvents(c.Request().Context(), ackFilter, page, pageSize)
	if err != nil {
		return mapError(c, err)
	}
	return paginatedResponse(c, events, page, pageSize, total)
}

func (h *AdminHandler) AcknowledgeAnomaly(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid anomaly ID")
	}

	if err := h.analyticsStore.AcknowledgeAnomaly(c.Request().Context(), id); err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
