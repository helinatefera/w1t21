package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
	"go.uber.org/zap"
)

func AnomalyDetectorJob(orderStore *store.OrderStore, analyticsStore *store.AnalyticsStore, logger *zap.Logger) Job {
	return Job{
		Name:     "anomaly_detector",
		Interval: 5 * time.Minute,
		Fn: func(ctx context.Context) error {
			// Rule 1: More than 6 cancellations in 24 hours
			if err := detectExcessiveCancellations(ctx, orderStore, analyticsStore, logger); err != nil {
				logger.Error("excessive cancellations check failed", zap.Error(err))
			}

			// Rule 2: More than 10 failed checkout attempts in 1 hour
			if err := detectCheckoutFailures(ctx, analyticsStore, logger); err != nil {
				logger.Error("checkout failures check failed", zap.Error(err))
			}

			return nil
		},
	}
}

func detectExcessiveCancellations(ctx context.Context, orderStore *store.OrderStore, analyticsStore *store.AnalyticsStore, logger *zap.Logger) error {
	// Get users who have cancelled orders in the last 24 hours
	// We check all users with recent cancellations
	rows, err := orderStore.CountByStatus(ctx)
	if err != nil {
		return err
	}
	_ = rows // Using this as a health check

	// Query users with excessive cancellations directly
	type userCancel struct {
		UserID string
		Count  int
	}

	// This is done via a direct query through the order store
	// For each user who has cancelled, check their count
	// In practice, we'd add a specific store method, but for now we rely on the
	// analytics events approach - each cancellation is also an analytics event
	users, err := analyticsStore.GetDistinctUsersWithEvents(ctx, "order_cancelled", 24)
	if err != nil {
		return fmt.Errorf("get users with cancellations: %w", err)
	}

	for _, userID := range users {
		count, err := orderStore.CountCancelledInPeriod(ctx, userID, 24)
		if err != nil {
			continue
		}
		if count > 6 {
			details, _ := json.Marshal(map[string]interface{}{
				"cancellations_24h": count,
				"threshold":         6,
			})
			anomaly := &model.AnomalyEvent{
				UserID:      userID,
				AnomalyType: "excessive_cancellations",
				Details:     details,
			}
			if err := analyticsStore.CreateAnomalyEvent(ctx, anomaly); err != nil {
				logger.Error("create anomaly event failed", zap.Error(err))
			} else {
				logger.Warn("anomaly detected: excessive cancellations",
					zap.String("user_id", userID.String()),
					zap.Int("count", count))
			}
		}
	}
	return nil
}

func detectCheckoutFailures(ctx context.Context, analyticsStore *store.AnalyticsStore, logger *zap.Logger) error {
	users, err := analyticsStore.GetDistinctUsersWithEvents(ctx, "checkout_failed", 1)
	if err != nil {
		return fmt.Errorf("get users with checkout failures: %w", err)
	}

	for _, userID := range users {
		count, err := analyticsStore.CountCheckoutFailures(ctx, userID, 1)
		if err != nil {
			continue
		}
		if count > 10 {
			details, _ := json.Marshal(map[string]interface{}{
				"checkout_failures_1h": count,
				"threshold":            10,
			})
			anomaly := &model.AnomalyEvent{
				UserID:      userID,
				AnomalyType: "repeated_checkout_failures",
				Details:     details,
			}
			if err := analyticsStore.CreateAnomalyEvent(ctx, anomaly); err != nil {
				logger.Error("create anomaly event failed", zap.Error(err))
			} else {
				logger.Warn("anomaly detected: repeated checkout failures",
					zap.String("user_id", userID.String()),
					zap.Int("count", count))
			}
		}
	}
	return nil
}
