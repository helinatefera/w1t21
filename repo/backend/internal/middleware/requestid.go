package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const (
	RequestIDKey = "request_id"
	loggerKey    = "ctx_logger"
)

// RequestID generates or propagates a request ID and, when a base logger is
// provided, stores a child logger enriched with that ID in the context.
func RequestID(baseLogger ...*zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get("X-Request-ID")
			if id == "" {
				id = uuid.New().String()
			}
			c.Set(RequestIDKey, id)
			c.Response().Header().Set("X-Request-ID", id)

			if len(baseLogger) > 0 && baseLogger[0] != nil {
				c.Set(loggerKey, baseLogger[0].With(zap.String("request_id", id)))
			}
			return next(c)
		}
	}
}

func GetRequestID(c echo.Context) string {
	if id, ok := c.Get(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// Logger returns the request-scoped logger (with request_id) stored in the
// context, or a no-op production logger if none was set.
func Logger(c echo.Context) *zap.Logger {
	if l, ok := c.Get(loggerKey).(*zap.Logger); ok {
		return l
	}
	l, _ := zap.NewProduction()
	return l
}
