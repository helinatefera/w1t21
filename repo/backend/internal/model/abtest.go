package model

import (
	"time"

	"github.com/google/uuid"
)

type ABTestStatus string

const (
	ABTestStatusDraft      ABTestStatus = "draft"
	ABTestStatusRunning    ABTestStatus = "running"
	ABTestStatusRolledBack ABTestStatus = "rolled_back"
	ABTestStatusCompleted  ABTestStatus = "completed"
)

type ABTest struct {
	ID                   uuid.UUID    `json:"id"`
	Name                 string       `json:"name"`
	Description          string       `json:"description"`
	Status               ABTestStatus `json:"status"`
	TrafficPct           int          `json:"traffic_pct"`
	StartDate            time.Time    `json:"start_date"`
	EndDate              time.Time    `json:"end_date"`
	ControlVariant       string       `json:"control_variant"`
	TestVariant          string       `json:"test_variant"`
	RollbackThresholdPct int          `json:"rollback_threshold_pct"`
	CreatedBy            uuid.UUID    `json:"created_by"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

type ABTestResult struct {
	ID             uuid.UUID `json:"id"`
	ABTestID       uuid.UUID `json:"ab_test_id"`
	Variant        string    `json:"variant"`
	Views          int64     `json:"views"`
	Orders         int64     `json:"orders"`
	ConversionRate float64   `json:"conversion_rate"`
	ComputedAt     time.Time `json:"computed_at"`
}
