package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
)

// statusOnlySlugs defines the template slugs emitted under "status_only" subscription mode.
// These correspond to order status changes and refund outcomes.
var statusOnlySlugs = map[string]bool{
	"order_confirmed":  true,
	"order_processing": true,
	"order_completed":  true,
	"order_cancelled":  true,
	"refund_approved":  true,
}

type NotificationService struct {
	notifStore *store.NotificationStore
}

func NewNotificationService(ns *store.NotificationStore) *NotificationService {
	return &NotificationService{notifStore: ns}
}

func (s *NotificationService) Send(ctx context.Context, userID uuid.UUID, templateSlug string, paramsJSON json.RawMessage) error {
	// Check user preferences
	prefs, err := s.notifStore.GetPreferences(ctx, userID)
	if err != nil {
		return fmt.Errorf("get preferences: %w", err)
	}

	var prefMap map[string]bool
	if err := json.Unmarshal(prefs.Preferences, &prefMap); err != nil {
		prefMap = make(map[string]bool)
	}
	if enabled, exists := prefMap[templateSlug]; exists && !enabled {
		return nil // User opted out
	}

	// Filter by subscription mode. Default to all_events for backward compatibility.
	mode := prefs.SubscriptionMode
	if mode == "" {
		mode = "all_events"
	}
	if mode == "status_only" && !statusOnlySlugs[templateSlug] {
		return nil // Event excluded by subscription mode
	}

	template, err := s.notifStore.GetTemplateBySlug(ctx, templateSlug)
	if err != nil {
		return fmt.Errorf("get template: %w", err)
	}
	if template == nil {
		return fmt.Errorf("template not found: %s", templateSlug)
	}

	// Render template
	var params map[string]string
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		params = make(map[string]string)
	}

	title := renderTemplate(template.TitleTemplate, params)
	body := renderTemplate(template.BodyTemplate, params)

	notif := &model.Notification{
		UserID:        userID,
		TemplateID:    template.ID,
		Params:        paramsJSON,
		RenderedTitle: title,
		RenderedBody:  body,
		Status:        "pending",
	}

	if err := s.notifStore.Create(ctx, notif); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func (s *NotificationService) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, page, pageSize int) ([]model.Notification, int, error) {
	return s.notifStore.ListByUser(ctx, userID, unreadOnly, page, pageSize)
}

func (s *NotificationService) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	notif, err := s.notifStore.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if notif == nil {
		return dto.ErrNotFound
	}
	if notif.UserID != userID {
		return dto.ErrForbidden
	}
	return s.notifStore.MarkRead(ctx, id)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.notifStore.MarkAllRead(ctx, userID)
}

func (s *NotificationService) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.notifStore.CountUnread(ctx, userID)
}

func (s *NotificationService) Retry(ctx context.Context, id, userID uuid.UUID) error {
	notif, err := s.notifStore.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if notif == nil {
		return dto.ErrNotFound
	}
	if notif.UserID != userID {
		return dto.ErrForbidden
	}
	if notif.Status != "failed" && notif.Status != "permanently_failed" {
		return fmt.Errorf("%w: notification is not in a failed state", dto.ErrValidation)
	}
	// Manual retry resets to pending so the worker picks it up on the next tick.
	return s.notifStore.ResetToPending(ctx, id)
}

func (s *NotificationService) GetPreferences(ctx context.Context, userID uuid.UUID) (*model.NotificationPreferences, error) {
	return s.notifStore.GetPreferences(ctx, userID)
}

func (s *NotificationService) UpdatePreferences(ctx context.Context, userID uuid.UUID, prefs map[string]bool, subscriptionMode string) error {
	// Default to all_events for backward compatibility when no mode is provided.
	if subscriptionMode == "" {
		subscriptionMode = "all_events"
	}
	data, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("marshal preferences: %w", err)
	}
	return s.notifStore.UpsertPreferences(ctx, userID, data, subscriptionMode)
}

// IsStatusOnlySlug reports whether the given template slug is classified as a
// status-change event for subscription mode filtering.
func IsStatusOnlySlug(slug string) bool {
	return statusOnlySlugs[slug]
}

func (s *NotificationService) ListTemplates(ctx context.Context) ([]model.NotificationTemplate, error) {
	return s.notifStore.ListTemplates(ctx)
}

func renderTemplate(tmpl string, params map[string]string) string {
	result := tmpl
	for k, v := range params {
		result = strings.ReplaceAll(result, "{{."+k+"}}", v)
	}
	return result
}
