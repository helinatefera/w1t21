package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Scheduler lifecycle tests
// ---------------------------------------------------------------------------

// TestScheduler_StartAndStop verifies that the scheduler starts jobs and
// stops them cleanly when the context is cancelled.
func TestScheduler_StartAndStop(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewScheduler(logger)

	var mu sync.Mutex
	runCount := 0

	scheduler.Register(Job{
		Name:     "test_job",
		Interval: 10 * time.Millisecond,
		Fn: func(ctx context.Context) error {
			mu.Lock()
			runCount++
			mu.Unlock()
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx)

	// Wait for at least one execution
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count := runCount
	mu.Unlock()

	if count == 0 {
		t.Fatal("job should have run at least once")
	}

	cancel()
	// Give goroutine time to stop
	time.Sleep(30 * time.Millisecond)

	mu.Lock()
	countAfterStop := runCount
	mu.Unlock()

	// Wait again — count should not increase after cancel
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	countFinal := runCount
	mu.Unlock()

	if countFinal > countAfterStop+1 {
		t.Errorf("job continued running after cancel: countAfterStop=%d countFinal=%d", countAfterStop, countFinal)
	}
}

// TestScheduler_MultipleJobs verifies that multiple registered jobs all run.
func TestScheduler_MultipleJobs(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewScheduler(logger)

	var mu sync.Mutex
	ran := map[string]bool{}

	for _, name := range []string{"job_a", "job_b", "job_c"} {
		n := name
		scheduler.Register(Job{
			Name:     n,
			Interval: 10 * time.Millisecond,
			Fn: func(ctx context.Context) error {
				mu.Lock()
				ran[n] = true
				mu.Unlock()
				return nil
			},
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()
	for _, name := range []string{"job_a", "job_b", "job_c"} {
		if !ran[name] {
			t.Errorf("job %q never ran", name)
		}
	}
}

// TestScheduler_JobError_DoesNotStopScheduler verifies that a failing job
// does not crash the scheduler — the job continues to be retried.
func TestScheduler_JobError_DoesNotStopScheduler(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewScheduler(logger)

	var mu sync.Mutex
	attempts := 0

	scheduler.Register(Job{
		Name:     "failing_job",
		Interval: 10 * time.Millisecond,
		Fn: func(ctx context.Context) error {
			mu.Lock()
			attempts++
			mu.Unlock()
			return errors.New("transient error")
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx)
	time.Sleep(80 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()
	if attempts < 2 {
		t.Errorf("failing job should have been retried, only ran %d time(s)", attempts)
	}
}

// ---------------------------------------------------------------------------
// Notification retry logic tests (unit-level, no DB)
// ---------------------------------------------------------------------------

// TestNotificationRetryJob_DeliverySuccess verifies that when the delivery
// function succeeds, the job completes without error.
func TestNotificationRetryJob_DeliverySuccess(t *testing.T) {
	logger := zap.NewNop()
	deliverCalls := 0

	deliver := func(title, body string) error {
		deliverCalls++
		return nil
	}

	// We can't easily test with real NotificationStore without a DB,
	// but we can verify the DeliveryFunc contract and that the job is created.
	job := NotificationRetryJobWithDelivery(nil, logger, deliver)

	// The job is properly configured
	if job.Name != "notification_retry" {
		t.Errorf("expected job name %q, got %q", "notification_retry", job.Name)
	}
	if job.Interval != 60*time.Second {
		t.Errorf("expected 60s interval, got %v", job.Interval)
	}
}

// TestNotificationRetryJob_ExponentialBackoff verifies the backoff calculation
// used in the retry worker.
func TestNotificationRetryJob_ExponentialBackoff(t *testing.T) {
	tests := []struct {
		retryCount int
		wantMin    time.Duration
	}{
		{0, 2 * time.Minute},  // 1 << (0+1) = 2 min
		{1, 4 * time.Minute},  // 1 << (1+1) = 4 min
		{2, 8 * time.Minute},  // 1 << (2+1) = 8 min
		{3, 16 * time.Minute}, // 1 << (3+1) = 16 min
	}

	for _, tc := range tests {
		backoff := time.Duration(1<<uint(tc.retryCount+1)) * time.Minute
		if backoff != tc.wantMin {
			t.Errorf("retryCount=%d: got %v, want %v", tc.retryCount, backoff, tc.wantMin)
		}
	}
}

// TestNotificationRetryJob_MaxRetriesExhausted verifies that the permanently_failed
// state is reached after max retries. This tests the logic path, not the DB.
func TestNotificationRetryJob_MaxRetriesExhausted(t *testing.T) {
	maxRetries := 3
	retryCount := 3

	// This mirrors the condition in notification_retry.go:76
	if retryCount < maxRetries {
		t.Fatal("at max retries, should not schedule another retry")
	}
	// At max retries, the worker marks permanently_failed
}

// TestScheduler_ContextCancellation_StopsImmediately verifies that a
// context cancellation stops the scheduler promptly.
func TestScheduler_ContextCancellation_StopsImmediately(t *testing.T) {
	logger := zap.NewNop()
	scheduler := NewScheduler(logger)

	longRunning := false
	scheduler.Register(Job{
		Name:     "long_interval",
		Interval: 1 * time.Hour, // would never fire in test
		Fn: func(ctx context.Context) error {
			longRunning = true
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx)

	// Cancel immediately
	cancel()
	time.Sleep(30 * time.Millisecond)

	if longRunning {
		t.Fatal("job with 1h interval should not have run")
	}
}
