package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/ledgermint/platform/internal/cache"
	"github.com/ledgermint/platform/internal/config"
	"github.com/ledgermint/platform/internal/crypto"
	"github.com/ledgermint/platform/internal/handler"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/router"
	"github.com/ledgermint/platform/internal/service"
	"github.com/ledgermint/platform/internal/store"
	"github.com/ledgermint/platform/internal/worker"
)

// Log configuration — paths and retention.
//
//	Primary log path : /var/log/ledgermint/app.log
//	Fallback         : ./logs/app.log  (when /var/log is not writable)
//	Max file size    : 100 MB per file
//	Max backups      : 10 rotated files
//	Max age          : 30 days
//	Compression      : enabled (gzip)
const (
	logMaxSizeMB  = 100
	logMaxBackups = 10
	logMaxAgeDays = 30
)

func main() {
	logger := buildLogger()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load config", zap.Error(err))
	}

	// Database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("connect to database", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("ping database", zap.Error(err))
	}
	logger.Info("connected to database")

	// Run migrations
	if err := runMigrations(ctx, pool, logger); err != nil {
		logger.Fatal("run migrations", zap.Error(err))
	}

	// Encryption
	encryptor, err := crypto.NewAESEncryptor(cfg.AESMasterKey)
	if err != nil {
		logger.Fatal("init AES encryptor", zap.Error(err))
	}

	// Cache
	hotCache := cache.New()

	// Stores
	userStore := store.NewUserStore(pool)
	collectibleStore := store.NewCollectibleStore(pool)
	orderStore := store.NewOrderStore(pool)
	messageStore := store.NewMessageStore(pool)
	notifStore := store.NewNotificationStore(pool)
	analyticsStore := store.NewAnalyticsStore(pool)
	auditStore := store.NewAuditStore(pool)

	// Services
	authService := service.NewAuthService(userStore, cfg.JWTSigningKey)
	userService := service.NewUserService(userStore, encryptor)
	notifService := service.NewNotificationService(notifStore)
	collectibleService := service.NewCollectibleService(collectibleStore, analyticsStore, hotCache, notifService)
	orderService := service.NewOrderService(orderStore, collectibleStore, notifService, analyticsStore)
	messageService := service.NewMessageService(messageStore, orderStore)
	analyticsService := service.NewAnalyticsService(analyticsStore)
	abtestService := service.NewABTestService(analyticsStore)

	// Handlers
	handlers := router.Handlers{
		Auth:         handler.NewAuthHandler(authService, auditStore),
		User:         handler.NewUserHandler(userService, auditStore),
		Collectible:  handler.NewCollectibleHandler(collectibleService, auditStore),
		Order:        handler.NewOrderHandler(orderService, auditStore),
		Message:      handler.NewMessageHandler(messageService),
		Notification: handler.NewNotificationHandler(notifService),
		Analytics:    handler.NewAnalyticsHandler(analyticsService, orderStore, collectibleService, notifStore),
		ABTest:       handler.NewABTestHandler(abtestService),
		Admin:        handler.NewAdminHandler(userStore, analyticsStore, hotCache),
		Setup:        handler.NewSetupHandler(userStore),
		Audit:        auditStore,
	}

	// Echo server
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = middleware.GlobalErrorHandler

	// Body limit for message attachments (10MB)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if strings.HasSuffix(c.Path(), "/messages") && c.Request().Method == "POST" {
				c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, 10<<20)
			}
			return next(c)
		}
	})

	router.Setup(e, handlers, cfg.JWTSigningKey, userStore, hotCache, logger)

	// Background workers
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	scheduler := worker.NewScheduler(logger)
	scheduler.Register(worker.NotificationRetryJob(notifStore, logger))
	scheduler.Register(worker.AnomalyDetectorJob(orderStore, analyticsStore, logger))
	scheduler.Register(worker.ABTestEvaluatorJob(analyticsStore, logger))
	scheduler.Register(worker.MetricsWriterJob(analyticsStore, orderStore, logger))
	scheduler.Register(worker.AnalyticsRollupJob(pool, logger))
	scheduler.Register(worker.Job{
		Name:     "token_cleanup",
		Interval: 1 * time.Hour,
		Fn: func(ctx context.Context) error {
			return userStore.CleanupExpiredTokens(ctx)
		},
	})
	scheduler.Register(worker.Job{
		Name:     "login_attempt_cleanup",
		Interval: 1 * time.Hour,
		Fn: func(ctx context.Context) error {
			return userStore.CleanupOldLoginAttempts(ctx)
		},
	})
	scheduler.Start(workerCtx)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutting down...")
		workerCancel()
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		e.Shutdown(shutdownCtx)
	}()

	logger.Info("starting server", zap.String("addr", cfg.ListenAddr))
	if err := e.Start(cfg.ListenAddr); err != nil {
		logger.Info("server stopped", zap.Error(err))
	}
}

// buildLogger creates a production JSON logger that writes to both stderr and
// a rotating log file. The file sink uses lumberjack for automatic rotation
// and retention.
func buildLogger() *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	jsonEncoder := zapcore.NewJSONEncoder(encoderCfg)

	// Always write to stderr
	stderrSink := zapcore.Lock(os.Stderr)
	stderrCore := zapcore.NewCore(jsonEncoder, stderrSink, zap.InfoLevel)

	// Try to set up a file sink with rotation
	fileCore := buildFileCore(jsonEncoder)
	if fileCore != nil {
		return zap.New(zapcore.NewTee(stderrCore, fileCore), zap.AddCaller())
	}
	return zap.New(stderrCore, zap.AddCaller())
}

func buildFileCore(encoder zapcore.Encoder) zapcore.Core {
	logPath := resolveLogPath()
	if logPath == "" {
		return nil
	}

	writer := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    logMaxSizeMB,
		MaxBackups: logMaxBackups,
		MaxAge:     logMaxAgeDays,
		Compress:   true,
	}
	return zapcore.NewCore(encoder, zapcore.AddSync(writer), zap.InfoLevel)
}

// resolveLogPath returns the first writable log path, preferring
// /var/log/ledgermint/app.log, falling back to ./logs/app.log.
func resolveLogPath() string {
	candidates := []string{
		"/var/log/ledgermint",
		"./logs",
	}
	for _, dir := range candidates {
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return filepath.Join(dir, "app.log")
		}
	}
	return ""
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool, logger *zap.Logger) error {
	migrationDir := "/app/migrations"
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		migrationDir = "migrations"
	}

	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		logger.Warn("no migration directory found", zap.String("dir", migrationDir))
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		var version int
		fmt.Sscanf(entry.Name(), "%d_", &version)
		if version == 0 {
			continue
		}

		var applied bool
		pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&applied)
		if applied {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		_, err = pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("execute migration %s: %w", entry.Name(), err)
		}

		_, err = pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version)
		if err != nil {
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}

		logger.Info("applied migration", zap.String("file", entry.Name()))
	}

	return nil
}
