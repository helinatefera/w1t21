package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/ledgermint/platform/internal/store"
	"go.uber.org/zap"
)

// DeliveryFunc is the hook called to actually deliver a notification.
// Return nil on success or an error to mark the notification as failed.
// When nil (the default in production), delivery always succeeds — the
// platform is LAN-only, so marking as delivered *is* the delivery action.
type DeliveryFunc func(title, body string) error

func NotificationRetryJob(notifStore *store.NotificationStore, logger *zap.Logger) Job {
	return NotificationRetryJobWithDelivery(notifStore, logger, nil)
}

func NotificationRetryJobWithDelivery(notifStore *store.NotificationStore, logger *zap.Logger, deliver DeliveryFunc) Job {
	return Job{
		Name:     "notification_retry",
		Interval: 60 * time.Second,
		Fn: func(ctx context.Context) error {
			// 1. Attempt delivery for pending notifications.
			pending, err := notifStore.GetPendingForDelivery(ctx, 100)
			if err != nil {
				return fmt.Errorf("get pending notifications: %w", err)
			}
			for _, n := range pending {
				if deliver != nil {
					if deliveryErr := deliver(n.RenderedTitle, n.RenderedBody); deliveryErr != nil {
						// Delivery failed — move to "failed" so the retry path picks it up.
						backoff := time.Duration(1<<uint(1)) * time.Minute // first retry in 2 min
						nextRetry := time.Now().Add(backoff).Format(time.RFC3339)
						if err := notifStore.UpdateRetryState(ctx, n.ID, "failed", 0, &nextRetry); err != nil {
							logger.Error("mark notification failed", zap.String("id", n.ID.String()), zap.Error(err))
						}
						continue
					}
				}
				// Delivery succeeded (or no delivery hook — LAN mode).
				if err := notifStore.UpdateRetryState(ctx, n.ID, "delivered", 0, nil); err != nil {
					logger.Error("deliver notification failed", zap.String("id", n.ID.String()), zap.Error(err))
				}
			}

			// 2. Retry failed notifications with exponential backoff.
			failed, err := notifStore.GetFailedForRetry(ctx, 50)
			if err != nil {
				return fmt.Errorf("get failed notifications: %w", err)
			}

			for _, n := range failed {
				newRetryCount := n.RetryCount + 1

				// Attempt redelivery.
				deliveryOK := true
				if deliver != nil {
					if deliveryErr := deliver(n.RenderedTitle, n.RenderedBody); deliveryErr != nil {
						deliveryOK = false
					}
				}

				if deliveryOK {
					// Retry succeeded — mark delivered.
					if err := notifStore.UpdateRetryState(ctx, n.ID, "delivered", newRetryCount, nil); err != nil {
						logger.Error("retry notification failed", zap.String("id", n.ID.String()), zap.Error(err))
					}
					continue
				}

				// Retry still failing.
				if newRetryCount >= n.MaxRetries {
					// Exhausted all retries — terminal failure.
					if err := notifStore.UpdateRetryState(ctx, n.ID, "permanently_failed", newRetryCount, nil); err != nil {
						logger.Error("permanently fail notification", zap.String("id", n.ID.String()), zap.Error(err))
					}
				} else {
					// Schedule next retry with exponential backoff.
					backoff := time.Duration(1<<uint(newRetryCount+1)) * time.Minute
					nextRetry := time.Now().Add(backoff).Format(time.RFC3339)
					if err := notifStore.UpdateRetryState(ctx, n.ID, "failed", newRetryCount, &nextRetry); err != nil {
						logger.Error("schedule notification retry", zap.String("id", n.ID.String()), zap.Error(err))
					}
				}
			}
			return nil
		},
	}
}
