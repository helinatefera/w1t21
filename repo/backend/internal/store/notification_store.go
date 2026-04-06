package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ledgermint/platform/internal/model"
)

type NotificationStore struct {
	pool *pgxpool.Pool
}

func NewNotificationStore(pool *pgxpool.Pool) *NotificationStore {
	return &NotificationStore{pool: pool}
}

func (s *NotificationStore) GetTemplateBySlug(ctx context.Context, slug string) (*model.NotificationTemplate, error) {
	var t model.NotificationTemplate
	err := s.pool.QueryRow(ctx,
		`SELECT id, slug, title_template, body_template, created_at
		 FROM notification_templates WHERE slug = $1`, slug).Scan(
		&t.ID, &t.Slug, &t.TitleTemplate, &t.BodyTemplate, &t.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

func (s *NotificationStore) ListTemplates(ctx context.Context) ([]model.NotificationTemplate, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, slug, title_template, body_template, created_at FROM notification_templates ORDER BY slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []model.NotificationTemplate
	for rows.Next() {
		var t model.NotificationTemplate
		if err := rows.Scan(&t.ID, &t.Slug, &t.TitleTemplate, &t.BodyTemplate, &t.CreatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, nil
}

func (s *NotificationStore) Create(ctx context.Context, n *model.Notification) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO notifications (user_id, template_id, params, rendered_title, rendered_body, status, delivered_at)
		 VALUES ($1, $2, $3, $4, $5, $6::varchar, CASE WHEN $6::varchar = 'delivered' THEN NOW() ELSE NULL END)
		 RETURNING id, created_at`,
		n.UserID, n.TemplateID, n.Params, n.RenderedTitle, n.RenderedBody, n.Status,
	).Scan(&n.ID, &n.CreatedAt)
}

func (s *NotificationStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	var n model.Notification
	err := s.pool.QueryRow(ctx,
		`SELECT n.id, n.user_id, n.template_id, t.slug, n.params, n.rendered_title, n.rendered_body,
		        n.is_read, n.status, n.retry_count, n.max_retries, n.next_retry_at, n.delivered_at, n.created_at
		 FROM notifications n JOIN notification_templates t ON n.template_id = t.id
		 WHERE n.id = $1`, id).Scan(
		&n.ID, &n.UserID, &n.TemplateID, &n.TemplateSlug, &n.Params, &n.RenderedTitle, &n.RenderedBody,
		&n.IsRead, &n.Status, &n.RetryCount, &n.MaxRetries, &n.NextRetryAt, &n.DeliveredAt, &n.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &n, err
}

func (s *NotificationStore) ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool, page, pageSize int) ([]model.Notification, int, error) {
	filter := ""
	if unreadOnly {
		filter = " AND n.is_read = FALSE"
	}

	var total int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications n WHERE n.user_id = $1`+filter, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := s.pool.Query(ctx,
		`SELECT n.id, n.user_id, n.template_id, t.slug, n.params, n.rendered_title, n.rendered_body,
		        n.is_read, n.status, n.retry_count, n.max_retries, n.next_retry_at, n.delivered_at, n.created_at
		 FROM notifications n JOIN notification_templates t ON n.template_id = t.id
		 WHERE n.user_id = $1`+filter+`
		 ORDER BY n.created_at DESC LIMIT $2 OFFSET $3`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.TemplateID, &n.TemplateSlug, &n.Params,
			&n.RenderedTitle, &n.RenderedBody, &n.IsRead, &n.Status, &n.RetryCount,
			&n.MaxRetries, &n.NextRetryAt, &n.DeliveredAt, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		notifications = append(notifications, n)
	}
	return notifications, total, nil
}

func (s *NotificationStore) MarkRead(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE notifications SET is_read = TRUE WHERE id = $1`, id)
	return err
}

func (s *NotificationStore) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE notifications SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE`, userID)
	return err
}

func (s *NotificationStore) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&count)
	return count, err
}

func (s *NotificationStore) RetryNotification(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE notifications SET status = 'delivered', delivered_at = NOW(), retry_count = retry_count + 1
		 WHERE id = $1`, id)
	return err
}

// ResetToPending moves a failed/permanently_failed notification back to pending
// so the worker picks it up for redelivery. retry_count is preserved for auditing.
func (s *NotificationStore) ResetToPending(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE notifications SET status = 'pending', next_retry_at = NULL
		 WHERE id = $1`, id)
	return err
}

func (s *NotificationStore) GetPendingForDelivery(ctx context.Context, limit int) ([]model.Notification, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, template_id, rendered_title, rendered_body, status, retry_count, max_retries
		 FROM notifications
		 WHERE status = 'pending'
		 ORDER BY created_at ASC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.TemplateID, &n.RenderedTitle, &n.RenderedBody,
			&n.Status, &n.RetryCount, &n.MaxRetries); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func (s *NotificationStore) GetFailedForRetry(ctx context.Context, limit int) ([]model.Notification, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, template_id, rendered_title, rendered_body, status, retry_count, max_retries
		 FROM notifications
		 WHERE status = 'failed' AND retry_count < max_retries AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.TemplateID, &n.RenderedTitle, &n.RenderedBody,
			&n.Status, &n.RetryCount, &n.MaxRetries); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func (s *NotificationStore) UpdateRetryState(ctx context.Context, id uuid.UUID, status string, retryCount int, nextRetryAt *string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE notifications SET status = $2, retry_count = $3, next_retry_at = $4::timestamptz,
		        delivered_at = CASE WHEN $2 = 'delivered' THEN NOW() ELSE delivered_at END
		 WHERE id = $1`, id, status, retryCount, nextRetryAt)
	return err
}

// Preferences

func (s *NotificationStore) GetPreferences(ctx context.Context, userID uuid.UUID) (*model.NotificationPreferences, error) {
	var p model.NotificationPreferences
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, preferences, subscription_mode FROM notification_preferences WHERE user_id = $1`, userID).Scan(
		&p.UserID, &p.Preferences, &p.SubscriptionMode)
	if err == pgx.ErrNoRows {
		return &model.NotificationPreferences{UserID: userID, Preferences: json.RawMessage(`{}`), SubscriptionMode: "all_events"}, nil
	}
	return &p, err
}

func (s *NotificationStore) UpsertPreferences(ctx context.Context, userID uuid.UUID, prefs json.RawMessage, subscriptionMode string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO notification_preferences (user_id, preferences, subscription_mode)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE SET preferences = $2, subscription_mode = $3`,
		userID, prefs, subscriptionMode)
	return err
}
