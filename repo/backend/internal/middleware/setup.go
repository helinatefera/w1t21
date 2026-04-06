package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/store"
)

// RequireSetup returns middleware that blocks all API requests (except the
// setup endpoints themselves) until at least one administrator account exists.
// Once an admin is detected the check is cached so it does not hit the
// database on every request.
func RequireSetup(userStore *store.UserStore) echo.MiddlewareFunc {
	var (
		mu       sync.RWMutex
		complete bool
		checked  time.Time
	)

	isComplete := func(ctx context.Context) bool {
		mu.RLock()
		if complete {
			mu.RUnlock()
			return true
		}
		// Re-check at most once every 2 seconds to avoid hammering the DB.
		stale := time.Since(checked) > 2*time.Second
		mu.RUnlock()

		if !stale {
			return false
		}

		exists, err := userStore.AdminExists(ctx)
		if err != nil {
			return false
		}

		mu.Lock()
		checked = time.Now()
		if exists {
			complete = true
		}
		mu.Unlock()
		return exists
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if isComplete(c.Request().Context()) {
				return next(c)
			}
			return c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:      dto.CodeSetupRequired,
					Message:   "Initial setup required. Create an administrator account via POST /api/setup/admin before using the API.",
					RequestID: GetRequestID(c),
				},
			})
		}
	}
}
