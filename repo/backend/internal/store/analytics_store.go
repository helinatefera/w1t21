package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/model"
)

type AnalyticsStore struct {
	pool *pgxpool.Pool
}

func NewAnalyticsStore(pool *pgxpool.Pool) *AnalyticsStore {
	return &AnalyticsStore{pool: pool}
}

func (s *AnalyticsStore) RecordEvent(ctx context.Context, e *model.AnalyticsEvent) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO analytics_events (user_id, event_type, collectible_id, session_id, ab_variant, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		e.UserID, e.EventType, e.CollectibleID, e.SessionID, e.ABVariant, e.Metadata)
	return err
}

func (s *AnalyticsStore) GetFunnel(ctx context.Context, days int) (*dto.FunnelResponse, error) {
	since := time.Now().AddDate(0, 0, -days)
	var views, orders int64
	err := s.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN event_type IN ('item_view', 'catalog_view') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN event_type = 'order_created' THEN 1 ELSE 0 END), 0)
		 FROM analytics_events WHERE created_at >= $1`, since).Scan(&views, &orders)
	if err != nil {
		return nil, err
	}

	rate := float64(0)
	if views > 0 {
		rate = float64(orders) / float64(views)
	}
	return &dto.FunnelResponse{Views: views, Orders: orders, Rate: rate, Days: days}, nil
}

func (s *AnalyticsStore) GetRetention(ctx context.Context, days int) ([]dto.RetentionCohort, error) {
	rows, err := s.pool.Query(ctx,
		`WITH cohorts AS (
			SELECT user_id, DATE(MIN(created_at)) AS cohort_date
			FROM analytics_events
			WHERE user_id IS NOT NULL AND created_at >= NOW() - make_interval(days => $1)
			GROUP BY user_id
		),
		activity AS (
			SELECT DISTINCT user_id, DATE(created_at) AS activity_date
			FROM analytics_events
			WHERE user_id IS NOT NULL AND created_at >= NOW() - make_interval(days => $1)
		)
		SELECT c.cohort_date::text, COUNT(DISTINCT c.user_id) AS cohort_size,
		       COUNT(DISTINCT CASE WHEN a.activity_date > c.cohort_date THEN a.user_id END) AS retained
		FROM cohorts c
		LEFT JOIN activity a ON c.user_id = a.user_id
		GROUP BY c.cohort_date
		ORDER BY c.cohort_date`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cohorts []dto.RetentionCohort
	for rows.Next() {
		var c dto.RetentionCohort
		if err := rows.Scan(&c.CohortDate, &c.CohortSize, &c.RetainedCount); err != nil {
			return nil, err
		}
		if c.CohortSize > 0 {
			c.RetentionRate = float64(c.RetainedCount) / float64(c.CohortSize)
		}
		cohorts = append(cohorts, c)
	}
	return cohorts, nil
}

func (s *AnalyticsStore) GetContentPerformance(ctx context.Context, limit int) ([]dto.ContentPerformance, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT c.id, c.title,
			COALESCE(SUM(CASE WHEN ae.event_type IN ('item_view', 'catalog_view') THEN 1 ELSE 0 END), 0) AS views,
			COALESCE(SUM(CASE WHEN ae.event_type = 'order_created' THEN 1 ELSE 0 END), 0) AS orders
		 FROM collectibles c
		 LEFT JOIN analytics_events ae ON ae.collectible_id = c.id
		 GROUP BY c.id, c.title
		 ORDER BY views DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.ContentPerformance
	for rows.Next() {
		var cp dto.ContentPerformance
		if err := rows.Scan(&cp.CollectibleID, &cp.Title, &cp.Views, &cp.Orders); err != nil {
			return nil, err
		}
		if cp.Views > 0 {
			cp.ConversionRate = float64(cp.Orders) / float64(cp.Views)
		}
		items = append(items, cp)
	}
	return items, nil
}

// A/B Tests

func (s *AnalyticsStore) CreateABTest(ctx context.Context, t *model.ABTest) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO ab_tests (name, description, status, traffic_pct, start_date, end_date,
		        control_variant, test_variant, rollback_threshold_pct, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, created_at, updated_at`,
		t.Name, t.Description, t.Status, t.TrafficPct, t.StartDate, t.EndDate,
		t.ControlVariant, t.TestVariant, t.RollbackThresholdPct, t.CreatedBy,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (s *AnalyticsStore) GetABTestByID(ctx context.Context, id uuid.UUID) (*model.ABTest, error) {
	var t model.ABTest
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, description, status, traffic_pct, start_date, end_date,
		        control_variant, test_variant, rollback_threshold_pct, created_by, created_at, updated_at
		 FROM ab_tests WHERE id = $1`, id).Scan(
		&t.ID, &t.Name, &t.Description, &t.Status, &t.TrafficPct, &t.StartDate, &t.EndDate,
		&t.ControlVariant, &t.TestVariant, &t.RollbackThresholdPct, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

func (s *AnalyticsStore) ListABTests(ctx context.Context) ([]model.ABTest, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, description, status, traffic_pct, start_date, end_date,
		        control_variant, test_variant, rollback_threshold_pct, created_by, created_at, updated_at
		 FROM ab_tests ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tests []model.ABTest
	for rows.Next() {
		var t model.ABTest
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Status, &t.TrafficPct,
			&t.StartDate, &t.EndDate, &t.ControlVariant, &t.TestVariant,
			&t.RollbackThresholdPct, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tests = append(tests, t)
	}
	return tests, nil
}

func (s *AnalyticsStore) ListRunningABTests(ctx context.Context) ([]model.ABTest, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, description, status, traffic_pct, start_date, end_date,
		        control_variant, test_variant, rollback_threshold_pct, created_by, created_at, updated_at
		 FROM ab_tests WHERE status = 'running'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tests []model.ABTest
	for rows.Next() {
		var t model.ABTest
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Status, &t.TrafficPct,
			&t.StartDate, &t.EndDate, &t.ControlVariant, &t.TestVariant,
			&t.RollbackThresholdPct, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tests = append(tests, t)
	}
	return tests, nil
}

func (s *AnalyticsStore) UpdateABTest(ctx context.Context, t *model.ABTest) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE ab_tests SET description = $2, traffic_pct = $3, end_date = $4,
		        rollback_threshold_pct = $5, updated_at = NOW()
		 WHERE id = $1`,
		t.ID, t.Description, t.TrafficPct, t.EndDate, t.RollbackThresholdPct)
	return err
}

func (s *AnalyticsStore) UpdateABTestStatus(ctx context.Context, id uuid.UUID, status model.ABTestStatus) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE ab_tests SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	return err
}

func (s *AnalyticsStore) GetABTestConversion(ctx context.Context, testName, variant string) (views int64, orders int64, err error) {
	tag := testName + ":" + variant
	err = s.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN event_type IN ('item_view', 'catalog_view') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN event_type IN ('order_created', 'order_completed') THEN 1 ELSE 0 END), 0)
		 FROM analytics_events
		 WHERE ab_variant = $1
		    OR ab_variant LIKE $1 || ',%'
		    OR ab_variant LIKE '%,' || $1
		    OR ab_variant LIKE '%,' || $1 || ',%'`,
		tag).Scan(&views, &orders)
	return
}

func (s *AnalyticsStore) SaveABTestResult(ctx context.Context, r *model.ABTestResult) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO ab_test_results (ab_test_id, variant, views, orders, conversion_rate)
		 VALUES ($1, $2, $3, $4, $5)`,
		r.ABTestID, r.Variant, r.Views, r.Orders, r.ConversionRate)
	return err
}

func (s *AnalyticsStore) GetLatestABTestResults(ctx context.Context, testID uuid.UUID) ([]model.ABTestResult, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT ON (variant) id, ab_test_id, variant, views, orders, conversion_rate, computed_at
		 FROM ab_test_results WHERE ab_test_id = $1
		 ORDER BY variant, computed_at DESC`, testID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ABTestResult
	for rows.Next() {
		var r model.ABTestResult
		if err := rows.Scan(&r.ID, &r.ABTestID, &r.Variant, &r.Views, &r.Orders, &r.ConversionRate, &r.ComputedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

// Anomaly events

func (s *AnalyticsStore) CreateAnomalyEvent(ctx context.Context, a *model.AnomalyEvent) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO anomaly_events (user_id, anomaly_type, details)
		 VALUES ($1, $2, $3) RETURNING id, created_at`,
		a.UserID, a.AnomalyType, a.Details).Scan(&a.ID, &a.CreatedAt)
}

func (s *AnalyticsStore) ListAnomalyEvents(ctx context.Context, acknowledgedFilter *bool, page, pageSize int) ([]model.AnomalyEvent, int, error) {
	filter := ""
	args := []interface{}{}
	argIdx := 1

	if acknowledgedFilter != nil {
		filter = fmt.Sprintf(" WHERE acknowledged = $%d", argIdx)
		args = append(args, *acknowledgedFilter)
		argIdx++
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM anomaly_events` + filter
	err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := fmt.Sprintf(
		`SELECT id, user_id, anomaly_type, details, acknowledged, created_at
		 FROM anomaly_events%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		filter, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []model.AnomalyEvent
	for rows.Next() {
		var e model.AnomalyEvent
		if err := rows.Scan(&e.ID, &e.UserID, &e.AnomalyType, &e.Details, &e.Acknowledged, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, nil
}

func (s *AnalyticsStore) AcknowledgeAnomaly(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE anomaly_events SET acknowledged = TRUE WHERE id = $1`, id)
	return err
}

func (s *AnalyticsStore) CountCheckoutFailures(ctx context.Context, userID uuid.UUID, hours int) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM analytics_events
		 WHERE user_id = $1 AND event_type = 'checkout_failed'
		   AND created_at > NOW() - make_interval(hours => $2)`,
		userID, hours).Scan(&count)
	return count, err
}

func (s *AnalyticsStore) GetDistinctUsersWithEvents(ctx context.Context, eventType string, hours int) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT user_id FROM analytics_events
		 WHERE event_type = $1 AND user_id IS NOT NULL
		   AND created_at > NOW() - make_interval(hours => $2)`,
		eventType, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *AnalyticsStore) CountActiveUsers(ctx context.Context, hours int) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM analytics_events
		 WHERE user_id IS NOT NULL AND created_at > NOW() - make_interval(hours => $1)`,
		hours).Scan(&count)
	return count, err
}
