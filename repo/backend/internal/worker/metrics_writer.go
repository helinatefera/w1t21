package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ledgermint/platform/internal/store"
	"go.uber.org/zap"
)

func MetricsWriterJob(analyticsStore *store.AnalyticsStore, orderStore *store.OrderStore, logger *zap.Logger) Job {
	return Job{
		Name:     "metrics_writer",
		Interval: 60 * time.Second,
		Fn: func(ctx context.Context) error {
			logDir := resolveMetricsDir()
			if logDir == "" {
				logger.Warn("metrics_writer: no writable directory found, skipping")
				return nil
			}

			// Use date-based filename for rotation
			filename := fmt.Sprintf("metrics-%s.log", time.Now().Format("2006-01-02"))
			logPath := filepath.Join(logDir, filename)

			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return fmt.Errorf("open metrics log: %w", err)
			}
			defer f.Close()

			// Collect metrics
			activeUsers, _ := analyticsStore.CountActiveUsers(ctx, 24)
			ordersByStatus, _ := orderStore.CountByStatus(ctx)

			timestamp := time.Now().Format(time.RFC3339)

			// Write Prometheus-style text
			fmt.Fprintf(f, "# HELP ledgermint_active_users_24h Number of active users in last 24 hours\n")
			fmt.Fprintf(f, "ledgermint_active_users_24h %d %s\n", activeUsers, timestamp)

			for status, count := range ordersByStatus {
				fmt.Fprintf(f, "ledgermint_orders_total{status=\"%s\"} %d %s\n", status, count, timestamp)
			}
			fmt.Fprintf(f, "\n")

			// Clean up old log files (>7 days)
			entries, err := os.ReadDir(logDir)
			if err == nil {
				cutoff := time.Now().AddDate(0, 0, -7)
				for _, entry := range entries {
					if info, err := entry.Info(); err == nil {
						if info.ModTime().Before(cutoff) {
							os.Remove(filepath.Join(logDir, entry.Name()))
						}
					}
				}
			}

			return nil
		},
	}
}

// resolveMetricsDir returns the first writable directory for metrics logs,
// mirroring the fallback behaviour of the main application logger.
func resolveMetricsDir() string {
	candidates := []string{
		os.Getenv("METRICS_LOG_DIR"),
		"/var/log/ledgermint",
		"./logs",
	}
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return dir
		}
	}
	return ""
}
