package worker

import (
	"context"
	"time"

	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
	"go.uber.org/zap"
)

const minSampleSize int64 = 100

func ABTestEvaluatorJob(analyticsStore *store.AnalyticsStore, logger *zap.Logger) Job {
	return Job{
		Name:     "abtest_evaluator",
		Interval: 5 * time.Minute,
		Fn: func(ctx context.Context) error {
			tests, err := analyticsStore.ListRunningABTests(ctx)
			if err != nil {
				return err
			}

			for _, test := range tests {
				// Check if test has ended
				if time.Now().After(test.EndDate) {
					if err := analyticsStore.UpdateABTestStatus(ctx, test.ID, model.ABTestStatusCompleted); err != nil {
						logger.Error("failed to complete A/B test", zap.String("test", test.Name), zap.Error(err))
					}
					continue
				}

				// Compute conversion rates
				controlViews, controlOrders, err := analyticsStore.GetABTestConversion(ctx, test.Name, test.ControlVariant)
				if err != nil {
					logger.Error("get control conversion", zap.String("test", test.Name), zap.Error(err))
					continue
				}

				testViews, testOrders, err := analyticsStore.GetABTestConversion(ctx, test.Name, test.TestVariant)
				if err != nil {
					logger.Error("get test conversion", zap.String("test", test.Name), zap.Error(err))
					continue
				}

				// Calculate rates
				controlRate := float64(0)
				if controlViews > 0 {
					controlRate = float64(controlOrders) / float64(controlViews)
				}
				testRate := float64(0)
				if testViews > 0 {
					testRate = float64(testOrders) / float64(testViews)
				}

				// Save results
				for _, r := range []struct {
					variant string
					views   int64
					orders  int64
					rate    float64
				}{
					{test.ControlVariant, controlViews, controlOrders, controlRate},
					{test.TestVariant, testViews, testOrders, testRate},
				} {
					result := &model.ABTestResult{
						ABTestID:       test.ID,
						Variant:        r.variant,
						Views:          r.views,
						Orders:         r.orders,
						ConversionRate: r.rate,
					}
					if err := analyticsStore.SaveABTestResult(ctx, result); err != nil {
						logger.Error("save A/B result", zap.String("test", test.Name), zap.Error(err))
					}
				}

				// Auto-rollback check with relative drop and minimum sample size
				if controlViews < minSampleSize || testViews < minSampleSize {
					continue // Not enough data
				}

				if controlRate > 0 {
					relativeDrop := (controlRate - testRate) / controlRate * 100
					if relativeDrop > float64(test.RollbackThresholdPct) {
						logger.Warn("A/B test auto-rollback triggered",
							zap.String("test", test.Name),
							zap.Float64("control_rate", controlRate),
							zap.Float64("test_rate", testRate),
							zap.Float64("relative_drop_pct", relativeDrop),
							zap.Int("threshold_pct", test.RollbackThresholdPct))

						if err := analyticsStore.UpdateABTestStatus(ctx, test.ID, model.ABTestStatusRolledBack); err != nil {
							logger.Error("auto-rollback failed", zap.String("test", test.Name), zap.Error(err))
						}
					}
				}
			}
			return nil
		},
	}
}
