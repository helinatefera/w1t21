package router

import (
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/cache"
	"github.com/ledgermint/platform/internal/handler"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/store"
	"go.uber.org/zap"
)

type Handlers struct {
	Auth         *handler.AuthHandler
	User         *handler.UserHandler
	Collectible  *handler.CollectibleHandler
	Order        *handler.OrderHandler
	Message      *handler.MessageHandler
	Notification *handler.NotificationHandler
	Analytics    *handler.AnalyticsHandler
	ABTest       *handler.ABTestHandler
	Admin        *handler.AdminHandler
	Setup        *handler.SetupHandler
	Audit        *store.AuditStore
}

func Setup(e *echo.Echo, h Handlers, signingKey []byte, userStore *store.UserStore, hotCache *cache.HotCache, logger *zap.Logger) {
	// Global middleware — applied to every route in declaration order.
	// CSRF runs first (globally) but exempts /api/auth/login and
	// /api/setup/admin. The setup guard is applied per-group below so
	// that the setup endpoints remain reachable before any admin exists.
	e.Use(middleware.RequestID(logger))
	e.Use(middleware.StructuredLogger(logger))
	e.Use(middleware.IPFilter(userStore, hotCache))
	e.Use(middleware.CSRF())

	api := e.Group("/api")

	// Initial setup (public, CSRF-exempt, self-disabling once an admin exists)
	setup := api.Group("/setup")
	setup.GET("/status", h.Setup.Status)
	setup.POST("/admin", h.Setup.Bootstrap)

	// Block all remaining API access until initial setup is complete.
	setupGuard := middleware.RequireSetup(userStore)

	// Auth (public, but gated behind setup)
	loginRL := middleware.LoginRateLimiter()
	auth := api.Group("/auth", setupGuard)
	auth.POST("/login", h.Auth.Login, loginRL.Middleware())
	auth.POST("/refresh", h.Auth.Refresh)

	// Protected routes
	protected := api.Group("", setupGuard, middleware.JWTAuth(signingKey))

	// Auth (protected)
	protected.GET("/auth/me", h.Auth.Me)
	protected.POST("/auth/logout", h.Auth.Logout)

	// Dashboard
	protected.GET("/dashboard", h.Analytics.GetDashboard)

	// Users (admin only)
	adminRole := middleware.RequireRole("administrator")
	users := protected.Group("/users", adminRole)
	users.POST("", h.User.Create)
	users.GET("", h.User.List)
	users.GET("/:id", h.User.Get)
	users.PATCH("/:id", h.User.Update)
	users.POST("/:id/roles", h.User.AddRole)
	users.DELETE("/:id/roles/:roleId", h.User.RemoveRole)
	users.POST("/:id/unlock", h.User.Unlock)

	// Collectibles
	listingRL := middleware.ListingRateLimiter()
	listingIPRL := middleware.ListingIPRateLimiter()
	collectibles := protected.Group("/collectibles")
	collectibles.GET("", h.Collectible.List)
	collectibles.GET("/mine", h.Collectible.ListMine, middleware.RequireRole("seller"))
	collectibles.GET("/:id", h.Collectible.Get)
	collectibles.POST("", h.Collectible.Create, middleware.RequireRole("seller"), listingRL.Middleware(), listingIPRL.Middleware())
	collectibles.PATCH("/:id", h.Collectible.Update, middleware.RequireRole("seller"))
	collectibles.POST("/:id/reviews", h.Collectible.PostReview)
	collectibles.PATCH("/:id/hide", h.Collectible.Hide, adminRole)
	collectibles.PATCH("/:id/publish", h.Collectible.Publish, adminRole)

	// Orders
	orderRL := middleware.OrderRateLimiter()
	orderIPRL := middleware.OrderIPRateLimiter()
	orders := protected.Group("/orders")
	orders.POST("", h.Order.Create, middleware.RequireRole("buyer"), orderRL.Middleware(), orderIPRL.Middleware())
	orders.GET("", h.Order.List)
	orders.GET("/:id", h.Order.Get)
	orders.POST("/:id/confirm", h.Order.Confirm, middleware.RequireRole("seller"))
	orders.POST("/:id/process", h.Order.Process, middleware.RequireRole("seller"))
	orders.POST("/:id/complete", h.Order.Complete, middleware.RequireRole("seller"))
	orders.POST("/:id/cancel", h.Order.Cancel)
	orders.POST("/:id/refund", h.Order.ApproveRefund, middleware.RequireRole("seller"))
	orders.POST("/:id/arbitration", h.Order.OpenArbitration)
	orders.PATCH("/:id/fulfillment", h.Order.UpdateFulfillment, middleware.RequireRole("seller"))

	// Messages
	messageRL := middleware.MessageRateLimiter()
	messageIPRL := middleware.MessageIPRateLimiter()
	orders.GET("/:orderId/messages", h.Message.List)
	orders.POST("/:orderId/messages", h.Message.Send, messageRL.Middleware(), messageIPRL.Middleware())
	protected.GET("/messages/:messageId/attachment", h.Message.DownloadAttachment)

	// Notifications
	notifications := protected.Group("/notifications")
	notifications.GET("", h.Notification.List)
	notifications.PATCH("/:id/read", h.Notification.MarkRead)
	notifications.POST("/read-all", h.Notification.MarkAllRead)
	notifications.POST("/:id/retry", h.Notification.Retry)
	notifications.GET("/preferences", h.Notification.GetPreferences)
	notifications.PUT("/preferences", h.Notification.UpdatePreferences)

	// Analytics (admin + compliance)
	analyticsRole := middleware.RequireRole("administrator", "compliance_analyst")
	analytics := protected.Group("/analytics", analyticsRole)
	analytics.GET("/funnel", h.Analytics.GetFunnel)
	analytics.GET("/retention", h.Analytics.GetRetention)
	analytics.GET("/content-performance", h.Analytics.GetContentPerformance)

	// A/B Tests (admin + compliance for read, run, and rollback; admin-only for other writes)
	// A/B test run and rollback operations are permitted for Administrators and Compliance Analysts.
	abtestReadRole := middleware.RequireRole("administrator", "compliance_analyst")
	abtestRunRole := middleware.RequireRole("administrator", "compliance_analyst")
	abtests := protected.Group("/ab-tests")
	abtests.POST("", h.ABTest.Create, abtestRunRole)
	abtests.GET("", h.ABTest.List, abtestReadRole)
	abtests.GET("/:id", h.ABTest.Get, abtestReadRole)
	abtests.PATCH("/:id", h.ABTest.Update, abtestRunRole)
	abtests.POST("/:id/complete", h.ABTest.Complete, abtestRunRole)
	abtests.POST("/:id/rollback", h.ABTest.Rollback, abtestRunRole)
	protected.GET("/ab-tests/assignments", h.ABTest.GetAssignments) // Available to all authenticated users
	protected.GET("/ab-tests/registry", h.ABTest.GetRegistry)      // Available to all authenticated users

	// Admin (administrator only)
	admin := protected.Group("/admin", adminRole)
	admin.GET("/ip-rules", h.Admin.ListIPRules)
	admin.POST("/ip-rules", h.Admin.CreateIPRule)
	admin.DELETE("/ip-rules/:id", h.Admin.DeleteIPRule)
	admin.GET("/metrics", h.Analytics.GetMetrics)

	// Admin anomalies (administrator + compliance_analyst)
	adminAnomalies := protected.Group("/admin", middleware.RequireRole("administrator", "compliance_analyst"))
	adminAnomalies.GET("/anomalies", h.Admin.ListAnomalies)
	adminAnomalies.PATCH("/anomalies/:id/acknowledge", h.Admin.AcknowledgeAnomaly)
}
