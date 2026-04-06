package worker

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Job struct {
	Name     string
	Interval time.Duration
	Fn       func(ctx context.Context) error
}

type Scheduler struct {
	logger *zap.Logger
	jobs   []Job
}

func NewScheduler(logger *zap.Logger) *Scheduler {
	return &Scheduler{logger: logger}
}

func (s *Scheduler) Register(job Job) {
	s.jobs = append(s.jobs, job)
}

func (s *Scheduler) Start(ctx context.Context) {
	for _, job := range s.jobs {
		go s.runJob(ctx, job)
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job) {
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	s.logger.Info("worker started", zap.String("job", job.Name), zap.Duration("interval", job.Interval))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("worker stopped", zap.String("job", job.Name))
			return
		case <-ticker.C:
			if err := job.Fn(ctx); err != nil {
				s.logger.Error("worker job failed", zap.String("job", job.Name), zap.Error(err))
			}
		}
	}
}
