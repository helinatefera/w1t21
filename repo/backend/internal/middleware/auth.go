package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
)

type UserClaims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func JWTAuth(signingKey []byte) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("access_token")
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:      dto.CodeUnauthorized,
						Message:   "missing access token",
						RequestID: GetRequestID(c),
					},
				})
			}

			token, err := jwt.ParseWithClaims(cookie.Value, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
				return signingKey, nil
			})
			if err != nil || !token.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:      dto.CodeUnauthorized,
						Message:   "invalid or expired token",
						RequestID: GetRequestID(c),
					},
				})
			}

			claims, ok := token.Claims.(*UserClaims)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:      dto.CodeUnauthorized,
						Message:   "invalid token claims",
						RequestID: GetRequestID(c),
					},
				})
			}

			c.Set("user_id", claims.UserID)
			c.Set("user_roles", claims.Roles)
			return next(c)
		}
	}
}

func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRoles, ok := c.Get("user_roles").([]string)
			if !ok {
				return echo.NewHTTPError(http.StatusForbidden, dto.ErrorResponse{
					Error: dto.ErrorDetail{
						Code:      dto.CodeForbidden,
						Message:   "access denied",
						RequestID: GetRequestID(c),
					},
				})
			}

			for _, required := range roles {
				for _, has := range userRoles {
					if strings.EqualFold(required, has) {
						return next(c)
					}
				}
			}

			return echo.NewHTTPError(http.StatusForbidden, dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:      dto.CodeForbidden,
					Message:   "insufficient role",
					RequestID: GetRequestID(c),
				},
			})
		}
	}
}

func GetUserID(c echo.Context) string {
	if id, ok := c.Get("user_id").(string); ok {
		return id
	}
	return ""
}

func GetUserRoles(c echo.Context) []string {
	if roles, ok := c.Get("user_roles").([]string); ok {
		return roles
	}
	return nil
}
