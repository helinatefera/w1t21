package worker

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func AnalyticsRollupJob(pool *pgxpool.Pool, logger *zap.Logger) Job {
	return Job{
		Name:     "analytics_rollup",
		Interval: 1 * time.Hour,
		Fn: func(ctx context.Context) error {
			// Pre-aggregate funnel data (views and orders) for 7-day and 30-day windows
			_, err := pool.Exec(ctx, `
				INSERT INTO analytics_rollups (period_days, views, orders, conversion_rate, computed_at)
				SELECT
					7,
					COALESCE(SUM(CASE WHEN event_type IN ('item_view', 'catalog_view') THEN 1 ELSE 0 END), 0),
					COALESCE(SUM(CASE WHEN event_type = 'order_created' THEN 1 ELSE 0 END), 0),
					CASE
						WHEN COALESCE(SUM(CASE WHEN event_type IN ('item_view', 'catalog_view') THEN 1 ELSE 0 END), 0) > 0
						THEN COALESCE(SUM(CASE WHEN event_type = 'order_created' THEN 1 ELSE 0 END), 0)::numeric
							/ COALESCE(SUM(CASE WHEN event_type IN ('item_view', 'catalog_view') THEN 1 ELSE 0 END), 0)::numeric
						ELSE 0
					END,
					NOW()
				FROM analytics_events
				WHERE created_at >= NOW() - INTERVAL '7 days'
				ON CONFLICT (period_days) DO UPDATE
					SET views = EXCLUDED.views,
						orders = EXCLUDED.orders,
						conversion_rate = EXCLUDED.conversion_rate,
						computed_at = EXCLUDED.computed_at
			`)
			if err != nil {
				logger.Error("rollup 7-day funnel failed", zap.Error(err))
			}

			_, err = pool.Exec(ctx, `
				INSERT INTO analytics_rollups (period_days, views, orders, conversion_rate, computed_at)
				SELECT
					30,
					COALESCE(SUM(CASE WHEN event_type IN ('item_view', 'catalog_view') THEN 1 ELSE 0 END), 0),
					COALESCE(SUM(CASE WHEN event_type = 'order_created' THEN 1 ELSE 0 END), 0),
					CASE
						WHEN COALESCE(SUM(CASE WHEN event_type IN ('item_view', 'catalog_view') THEN 1 ELSE 0 END), 0) > 0
						THEN COALESCE(SUM(CASE WHEN event_type = 'order_created' THEN 1 ELSE 0 END), 0)::numeric
							/ COALESCE(SUM(CASE WHEN event_type IN ('item_view', 'catalog_view') THEN 1 ELSE 0 END), 0)::numeric
						ELSE 0
					END,
					NOW()
				FROM analytics_events
				WHERE created_at >= NOW() - INTERVAL '30 days'
				ON CONFLICT (period_days) DO UPDATE
					SET views = EXCLUDED.views,
						orders = EXCLUDED.orders,
						conversion_rate = EXCLUDED.conversion_rate,
						computed_at = EXCLUDED.computed_at
			`)
			if err != nil {
				logger.Error("rollup 30-day funnel failed", zap.Error(err))
			}

			// Pre-aggregate retention cohorts for the last 30 days
			_, err = pool.Exec(ctx, `
				INSERT INTO retention_rollups (cohort_date, cohort_size, retained_count, retention_rate, computed_at)
				WITH cohorts AS (
					SELECT user_id, DATE(MIN(created_at)) AS cohort_date
					FROM analytics_events
					WHERE user_id IS NOT NULL AND created_at >= NOW() - INTERVAL '30 days'
					GROUP BY user_id
				),
				activity AS (
					SELECT DISTINCT user_id, DATE(created_at) AS activity_date
					FROM analytics_events
					WHERE user_id IS NOT NULL AND created_at >= NOW() - INTERVAL '30 days'
				)
				SELECT
					c.cohort_date,
					COUNT(DISTINCT c.user_id)::int AS cohort_size,
					COUNT(DISTINCT CASE WHEN a.activity_date > c.cohort_date THEN a.user_id END)::int AS retained,
					CASE
						WHEN COUNT(DISTINCT c.user_id) > 0
						THEN COUNT(DISTINCT CASE WHEN a.activity_date > c.cohort_date THEN a.user_id END)::numeric
							/ COUNT(DISTINCT c.user_id)::numeric
						ELSE 0
					END AS retention_rate,
					NOW()
				FROM cohorts c
				LEFT JOIN activity a ON c.user_id = a.user_id
				GROUP BY c.cohort_date
				ON CONFLICT (cohort_date) DO UPDATE
					SET cohort_size = EXCLUDED.cohort_size,
						retained_count = EXCLUDED.retained_count,
						retention_rate = EXCLUDED.retention_rate,
						computed_at = EXCLUDED.computed_at
			`)
			if err != nil {
				logger.Error("rollup retention failed", zap.Error(err))
			}

			logger.Info("analytics rollup completed")
			return nil
		},
	}
}
