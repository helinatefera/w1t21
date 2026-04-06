package service

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
)

type ABTestService struct {
	analyticsStore *store.AnalyticsStore
}

func NewABTestService(as *store.AnalyticsStore) *ABTestService {
	return &ABTestService{analyticsStore: as}
}

func (s *ABTestService) Create(ctx context.Context, req dto.CreateABTestRequest, createdBy uuid.UUID) (*model.ABTest, error) {
	// Enforce experiment-to-component mapping: every experiment must reference
	// a registered UI component with known variant names.
	if msg := ValidateExperiment(req.Name, req.ControlVariant, req.TestVariant); msg != "" {
		return nil, fmt.Errorf("%w: %s", dto.ErrValidation, msg)
	}

	startDate, err := parseFlexibleDate(req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid start_date format, use RFC3339 or YYYY-MM-DDTHH:MM", dto.ErrValidation)
	}
	endDate, err := parseFlexibleDate(req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid end_date format, use RFC3339 or YYYY-MM-DDTHH:MM", dto.ErrValidation)
	}

	test := &model.ABTest{
		Name:                 req.Name,
		Description:          req.Description,
		Status:               model.ABTestStatusDraft,
		TrafficPct:           req.TrafficPct,
		StartDate:            startDate,
		EndDate:              endDate,
		ControlVariant:       req.ControlVariant,
		TestVariant:          req.TestVariant,
		RollbackThresholdPct: req.RollbackThresholdPct,
		CreatedBy:            createdBy,
	}

	// Auto-start if start date is in the past or now
	if !startDate.After(time.Now()) {
		test.Status = model.ABTestStatusRunning
	}

	if err := s.analyticsStore.CreateABTest(ctx, test); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "ab_tests_active_name_unique") {
			return nil, fmt.Errorf("%w: an active A/B test with this name already exists", dto.ErrConflict)
		}
		return nil, fmt.Errorf("create A/B test: %w", err)
	}
	return test, nil
}

func (s *ABTestService) GetByID(ctx context.Context, id uuid.UUID) (*model.ABTest, error) {
	test, err := s.analyticsStore.GetABTestByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if test == nil {
		return nil, dto.ErrNotFound
	}
	return test, nil
}

func (s *ABTestService) List(ctx context.Context) ([]model.ABTest, error) {
	return s.analyticsStore.ListABTests(ctx)
}

func (s *ABTestService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateABTestRequest) (*model.ABTest, error) {
	test, err := s.analyticsStore.GetABTestByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if test == nil {
		return nil, dto.ErrNotFound
	}
	if test.Status != model.ABTestStatusDraft && test.Status != model.ABTestStatusRunning {
		return nil, fmt.Errorf("%w: cannot update a %s test", dto.ErrValidation, test.Status)
	}

	if req.Description != nil {
		test.Description = *req.Description
	}
	if req.TrafficPct != nil {
		test.TrafficPct = *req.TrafficPct
	}
	if req.EndDate != nil {
		endDate, err := parseFlexibleDate(*req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid end_date format", dto.ErrValidation)
		}
		test.EndDate = endDate
	}
	if req.RollbackThresholdPct != nil {
		test.RollbackThresholdPct = *req.RollbackThresholdPct
	}

	if err := s.analyticsStore.UpdateABTest(ctx, test); err != nil {
		return nil, fmt.Errorf("update A/B test: %w", err)
	}
	return test, nil
}

func (s *ABTestService) Rollback(ctx context.Context, id uuid.UUID) error {
	test, err := s.analyticsStore.GetABTestByID(ctx, id)
	if err != nil {
		return err
	}
	if test == nil {
		return dto.ErrNotFound
	}
	if test.Status != model.ABTestStatusRunning {
		return fmt.Errorf("%w: test is not running", dto.ErrValidation)
	}
	return s.analyticsStore.UpdateABTestStatus(ctx, id, model.ABTestStatusRolledBack)
}

func (s *ABTestService) Complete(ctx context.Context, id uuid.UUID) error {
	test, err := s.analyticsStore.GetABTestByID(ctx, id)
	if err != nil {
		return err
	}
	if test == nil {
		return dto.ErrNotFound
	}
	if test.Status != model.ABTestStatusRunning {
		return fmt.Errorf("%w: test is not running", dto.ErrValidation)
	}
	return s.analyticsStore.UpdateABTestStatus(ctx, id, model.ABTestStatusCompleted)
}

func (s *ABTestService) GetResults(ctx context.Context, testID uuid.UUID) ([]model.ABTestResult, error) {
	return s.analyticsStore.GetLatestABTestResults(ctx, testID)
}

func (s *ABTestService) GetAssignments(ctx context.Context, userID string) ([]dto.ABTestAssignment, error) {
	tests, err := s.analyticsStore.ListRunningABTests(ctx)
	if err != nil {
		return nil, err
	}

	assignments := make([]dto.ABTestAssignment, 0, len(tests))
	for _, test := range tests {
		variant := AssignVariant(userID, test.Name, test.TrafficPct)
		variantName := test.ControlVariant
		if variant == "test" {
			variantName = test.TestVariant
		}
		assignments = append(assignments, dto.ABTestAssignment{
			TestName: test.Name,
			Variant:  variantName,
		})
	}
	return assignments, nil
}

// parseFlexibleDate accepts datetime-local (2006-01-02T15:04), RFC3339, and US format.
func parseFlexibleDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04",       // HTML datetime-local
		time.RFC3339,             // Standard API format
		"2006-01-02T15:04:05",   // ISO without timezone
		"01/02/2006 3:04 PM",    // US human-readable
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}

// AssignVariant uses deterministic hashing: fnv32(userID + testName) % 100.
func AssignVariant(userID, testName string, trafficPct int) string {
	h := fnv.New32a()
	h.Write([]byte(userID + testName))
	bucket := int(h.Sum32() % 100)
	if bucket < trafficPct {
		return "test"
	}
	return "control"
}
