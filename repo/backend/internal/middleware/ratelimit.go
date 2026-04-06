package middleware

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
	"golang.org/x/time/rate"
)

type RateLimiterConfig struct {
	Rate  rate.Limit
	Burst int
	KeyFn func(c echo.Context) string
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	config   RateLimiterConfig
}

func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		config:   cfg,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	e, ok := rl.limiters[key]
	if !ok {
		limiter := rate.NewLimiter(rl.config.Rate, rl.config.Burst)
		rl.limiters[key] = &rateLimiterEntry{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}
	e.lastSeen = time.Now()
	return e.limiter
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-1 * time.Hour)
		for k, e := range rl.limiters {
			if e.lastSeen.Before(cutoff) {
				delete(rl.limiters, k)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip rate limiting if disabled (e.g., during tests)
			if os.Getenv("DISABLE_RATE_LIMIT") == "true" {
				return next(c)
			}

			key := rl.config.KeyFn(c)
			limiter := rl.getLimiter(key)

			if !limiter.Allow() {
				c.Response().Header().Set("Retry-After", "60")
				return echo.NewHTTPError(http.StatusTooManyRequests, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:      dto.CodeRateLimited,
						Message:   "too many requests",
						RequestID: GetRequestID(c),
					},
				})
			}
			return next(c)
		}
	}
}

// Preconfigured rate limiters

func LoginRateLimiter() *RateLimiter {
	// 10 requests per 15 minutes per IP
	return NewRateLimiter(RateLimiterConfig{
		Rate:  rate.Every(90 * time.Second), // ~10 per 15 min
		Burst: 10,
		KeyFn: func(c echo.Context) string { return "login:" + c.RealIP() },
	})
}

func OrderRateLimiter() *RateLimiter {
	// 30 per minute per user
	return NewRateLimiter(RateLimiterConfig{
		Rate:  rate.Every(2 * time.Second),
		Burst: 30,
		KeyFn: func(c echo.Context) string { return "order:" + GetUserID(c) },
	})
}

func OrderIPRateLimiter() *RateLimiter {
	// 60 per minute per IP — catches credential-stuffing bots that rotate users.
	return NewRateLimiter(RateLimiterConfig{
		Rate:  rate.Every(1 * time.Second),
		Burst: 60,
		KeyFn: func(c echo.Context) string { return "order_ip:" + c.RealIP() },
	})
}

func MessageRateLimiter() *RateLimiter {
	// 20 per minute per user
	return NewRateLimiter(RateLimiterConfig{
		Rate:  rate.Every(3 * time.Second),
		Burst: 20,
		KeyFn: func(c echo.Context) string { return "message:" + GetUserID(c) },
	})
}

func MessageIPRateLimiter() *RateLimiter {
	// 40 per minute per IP — prevents spam from a single network.
	return NewRateLimiter(RateLimiterConfig{
		Rate:  rate.Every(1500 * time.Millisecond),
		Burst: 40,
		KeyFn: func(c echo.Context) string { return "message_ip:" + c.RealIP() },
	})
}

func ListingRateLimiter() *RateLimiter {
	// 10 per hour per user
	return NewRateLimiter(RateLimiterConfig{
		Rate:  rate.Every(6 * time.Minute),
		Burst: 10,
		KeyFn: func(c echo.Context) string { return "listing:" + GetUserID(c) },
	})
}

func ListingIPRateLimiter() *RateLimiter {
	// 20 per hour per IP — prevents bulk listing creation from one IP.
	return NewRateLimiter(RateLimiterConfig{
		Rate:  rate.Every(3 * time.Minute),
		Burst: 20,
		KeyFn: func(c echo.Context) string { return "listing_ip:" + c.RealIP() },
	})
}
