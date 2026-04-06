package dto

import (
	"time"

	"github.com/google/uuid"
)

type AuthResponse struct {
	User  UserResponse `json:"user"`
	Roles []string     `json:"roles"`
}

type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email,omitempty"`
	IsLocked    bool      `json:"is_locked"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalCount int         `json:"total_count"`
	TotalPages int         `json:"total_pages"`
}

type DashboardResponse struct {
	OwnedCollectibles   int `json:"owned_collectibles"`
	OpenOrders          int `json:"open_orders"`
	UnreadNotifications int `json:"unread_notifications"`
	SellerOpenOrders    int `json:"seller_open_orders"`
	ListedItems         int `json:"listed_items"`
}

type FunnelResponse struct {
	Views  int64   `json:"views"`
	Orders int64   `json:"orders"`
	Rate   float64 `json:"rate"`
	Days   int     `json:"days"`
}

type RetentionCohort struct {
	CohortDate    string  `json:"cohort_date"`
	CohortSize    int     `json:"cohort_size"`
	RetainedCount int     `json:"retained_count"`
	RetentionRate float64 `json:"retention_rate"`
}

type ContentPerformance struct {
	CollectibleID uuid.UUID `json:"collectible_id"`
	Title         string    `json:"title"`
	Views         int64     `json:"views"`
	Orders        int64     `json:"orders"`
	ConversionRate float64  `json:"conversion_rate"`
}

type ABTestAssignment struct {
	TestName string `json:"test_name"`
	Variant  string `json:"variant"`
}

type MetricsResponse struct {
	RequestCount    int64            `json:"request_count"`
	ErrorCount      int64            `json:"error_count"`
	ActiveUsers     int              `json:"active_users"`
	OrdersByStatus  map[string]int64 `json:"orders_by_status"`
	CollectedAt     time.Time        `json:"collected_at"`
}
