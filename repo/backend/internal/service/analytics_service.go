package service

import (
	"context"

	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
)

type AnalyticsService struct {
	analyticsStore *store.AnalyticsStore
}

func NewAnalyticsService(as *store.AnalyticsStore) *AnalyticsService {
	return &AnalyticsService{analyticsStore: as}
}

func (s *AnalyticsService) RecordEvent(ctx context.Context, event *model.AnalyticsEvent) error {
	return s.analyticsStore.RecordEvent(ctx, event)
}

func (s *AnalyticsService) GetFunnel(ctx context.Context, days int) (*dto.FunnelResponse, error) {
	return s.analyticsStore.GetFunnel(ctx, days)
}

func (s *AnalyticsService) GetRetention(ctx context.Context, days int) ([]dto.RetentionCohort, error) {
	return s.analyticsStore.GetRetention(ctx, days)
}

func (s *AnalyticsService) GetContentPerformance(ctx context.Context, limit int) ([]dto.ContentPerformance, error) {
	return s.analyticsStore.GetContentPerformance(ctx, limit)
}

func (s *AnalyticsService) CountActiveUsers(ctx context.Context, hours int) (int, error) {
	return s.analyticsStore.CountActiveUsers(ctx, hours)
}
