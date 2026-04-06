package middleware

import (
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/cache"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/model"
)

type IPRuleProvider interface {
	GetAllIPRules() ([]model.IPRule, error)
}

func IPFilter(provider IPRuleProvider, c *cache.HotCache) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			rules, err := getCachedIPRules(provider, c)
			if err != nil {
				// Fail closed: deny all traffic if rules cannot be loaded
				return echo.NewHTTPError(http.StatusServiceUnavailable, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:      dto.CodeInternal,
						Message:   "security rules unavailable",
						RequestID: GetRequestID(ctx),
					},
				})
			}

			if len(rules) == 0 {
				return next(ctx)
			}

			clientIP := net.ParseIP(ctx.RealIP())
			if clientIP == nil {
				return next(ctx)
			}

			// Check deny rules first
			for _, rule := range rules {
				_, network, err := net.ParseCIDR(rule.CIDR)
				if err != nil {
					continue
				}
				if rule.Action == "deny" && network.Contains(clientIP) {
					return echo.NewHTTPError(http.StatusForbidden, dto.ErrorResponse{
						Error: dto.ErrorDetail{
							Code:      dto.CodeForbidden,
							Message:   "IP address blocked",
							RequestID: GetRequestID(ctx),
						},
					})
				}
			}

			// If there are allow rules, client must match at least one
			hasAllowRules := false
			for _, rule := range rules {
				if rule.Action == "allow" {
					hasAllowRules = true
					break
				}
			}

			if hasAllowRules {
				allowed := false
				for _, rule := range rules {
					if rule.Action != "allow" {
						continue
					}
					_, network, err := net.ParseCIDR(rule.CIDR)
					if err != nil {
						continue
					}
					if network.Contains(clientIP) {
						allowed = true
						break
					}
				}
				if !allowed {
					return echo.NewHTTPError(http.StatusForbidden, dto.ErrorResponse{
						Error: dto.ErrorDetail{
							Code:      dto.CodeForbidden,
							Message:   "IP address not in allow list",
							RequestID: GetRequestID(ctx),
						},
					})
				}
			}

			return next(ctx)
		}
	}
}

func getCachedIPRules(provider IPRuleProvider, c *cache.HotCache) ([]model.IPRule, error) {
	if cached, ok := c.Get("ip_rules"); ok {
		return cached.([]model.IPRule), nil
	}
	rules, err := provider.GetAllIPRules()
	if err != nil {
		return nil, err
	}
	c.Set("ip_rules", rules, 60*time.Second)
	return rules, nil
}
