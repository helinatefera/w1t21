package middleware

import (
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var sensitiveFields = map[string]bool{
	"password":      true,
	"token":         true,
	"authorization": true,
	"cookie":        true,
	"secret":        true,
}

func StructuredLogger(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start)

			req := c.Request()

			// Prefer the request-scoped logger (already has request_id),
			// fall back to the base logger with an explicit field.
			log := Logger(c)
			if log == nil {
				log = logger.With(zap.String("request_id", GetRequestID(c)))
			}

			fields := []zap.Field{
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.Int("status", c.Response().Status),
				zap.Duration("latency", latency),
				zap.String("remote_ip", c.RealIP()),
			}

			if userID, ok := c.Get("user_id").(string); ok {
				fields = append(fields, zap.String("user_id", userID))
			}

			if err != nil {
				fields = append(fields, zap.Error(err))
			}

			status := c.Response().Status
			switch {
			case status >= 500:
				log.Error("request", fields...)
			case status >= 400:
				log.Warn("request", fields...)
			default:
				log.Info("request", fields...)
			}

			return err
		}
	}
}

func RedactSensitive(key, value string) string {
	if sensitiveFields[strings.ToLower(key)] {
		return "[REDACTED]"
	}
	return value
}
