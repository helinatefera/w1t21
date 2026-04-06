package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/service"
	"github.com/ledgermint/platform/internal/store"
)

type AnalyticsHandler struct {
	analyticsService   *service.AnalyticsService
	orderStore         *store.OrderStore
	collectibleService *service.CollectibleService
	notificationStore  *store.NotificationStore
}

func NewAnalyticsHandler(as *service.AnalyticsService, os *store.OrderStore, cs *service.CollectibleService, ns *store.NotificationStore) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: as, orderStore: os, collectibleService: cs, notificationStore: ns}
}

func (h *AnalyticsHandler) GetFunnel(c echo.Context) error {
	days, _ := strconv.Atoi(c.QueryParam("days"))
	if days <= 0 {
		days = 7
	}

	funnel, err := h.analyticsService.GetFunnel(c.Request().Context(), days)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, funnel)
}

func (h *AnalyticsHandler) GetRetention(c echo.Context) error {
	days, _ := strconv.Atoi(c.QueryParam("days"))
	if days <= 0 {
		days = 30
	}

	cohorts, err := h.analyticsService.GetRetention(c.Request().Context(), days)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, cohorts)
}

func (h *AnalyticsHandler) GetContentPerformance(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}

	items, err := h.analyticsService.GetContentPerformance(c.Request().Context(), limit)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, items)
}

func (h *AnalyticsHandler) GetMetrics(c echo.Context) error {
	ctx := c.Request().Context()

	activeUsers, _ := h.analyticsService.CountActiveUsers(ctx, 24)
	ordersByStatus, _ := h.orderStore.CountByStatus(ctx)

	return c.JSON(http.StatusOK, dto.MetricsResponse{
		ActiveUsers:    activeUsers,
		OrdersByStatus: ordersByStatus,
		CollectedAt:    time.Now(),
	})
}

func (h *AnalyticsHandler) GetDashboard(c echo.Context) error {
	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	ctx := c.Request().Context()

	// Buyer-perspective counters
	openOrders, _ := h.orderStore.CountOpenByBuyer(ctx, userID)
	ownedCollectibles, _ := h.orderStore.CountCompletedByBuyer(ctx, userID)
	unreadNotifications, _ := h.notificationStore.CountUnread(ctx, userID)

	// Seller-perspective counters
	sellerOpenOrders, _ := h.orderStore.CountOpenBySeller(ctx, userID)
	listedItems, _ := h.collectibleService.CountBySeller(ctx, userID)

	return c.JSON(http.StatusOK, dto.DashboardResponse{
		OwnedCollectibles:   ownedCollectibles,
		OpenOrders:          openOrders,
		UnreadNotifications: unreadNotifications,
		SellerOpenOrders:    sellerOpenOrders,
		ListedItems:         listedItems,
	})
}
