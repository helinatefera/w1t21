package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ledgermint/platform/internal/crypto"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
)

const (
	// Rolling-window abuse thresholds
	maxUserFailures = 5
	maxIPFailures   = 20
	failureWindow   = 15 * time.Minute
	lockoutDuration = 30 * time.Minute
)

type AuthService struct {
	userStore  *store.UserStore
	signingKey []byte
}

func NewAuthService(userStore *store.UserStore, signingKey []byte) *AuthService {
	return &AuthService{userStore: userStore, signingKey: signingKey}
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, clientIP string) (*dto.AuthResponse, []*http.Cookie, error) {
	windowStart := time.Now().Add(-failureWindow)

	// IP-based rolling-window check — blocks brute force across usernames
	ipCount, err := s.userStore.CountRecentFailuresByIP(ctx, clientIP, windowStart)
	if err != nil {
		return nil, nil, fmt.Errorf("count IP failures: %w", err)
	}
	if ipCount >= maxIPFailures {
		return nil, nil, dto.ErrAccountLocked
	}

	user, err := s.userStore.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		// Record attempt against IP even for unknown usernames
		_ = s.userStore.RecordFailedAttempt(ctx, nil, clientIP)
		return nil, nil, dto.ErrInvalidCredentials
	}

	// User-level lockout check: first check the is_locked flag, then fall
	// back to the rolling-window failure count so that a concurrent request
	// that missed the flag update still sees the lockout.
	if user.IsLocked {
		if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
			return nil, nil, dto.ErrAccountLocked
		}
		// Lock expired — release it
		if err := s.userStore.UnlockAccount(ctx, user.ID); err != nil {
			return nil, nil, fmt.Errorf("unlock account: %w", err)
		}
	} else {
		// Double-check rolling-window count in case is_locked flag was not
		// visible yet (e.g. read before a concurrent LockAccount committed).
		preCount, err := s.userStore.CountRecentFailuresByUser(ctx, user.ID, windowStart)
		if err != nil {
			return nil, nil, fmt.Errorf("count user failures: %w", err)
		}
		if preCount >= maxUserFailures {
			lockUntil := time.Now().Add(lockoutDuration).Format(time.RFC3339)
			_ = s.userStore.LockAccount(ctx, user.ID, lockUntil)
			return nil, nil, dto.ErrAccountLocked
		}
	}

	if !crypto.CheckPassword(user.PasswordHash, req.Password) {
		// Record the failed attempt
		if err := s.userStore.RecordFailedAttempt(ctx, &user.ID, clientIP); err != nil {
			return nil, nil, fmt.Errorf("record failed attempt: %w", err)
		}

		// Evaluate user threshold in the rolling window
		userCount, err := s.userStore.CountRecentFailuresByUser(ctx, user.ID, windowStart)
		if err != nil {
			return nil, nil, fmt.Errorf("count user failures: %w", err)
		}
		if userCount >= maxUserFailures {
			lockUntil := time.Now().Add(lockoutDuration).Format(time.RFC3339)
			if err := s.userStore.LockAccount(ctx, user.ID, lockUntil); err != nil {
				return nil, nil, fmt.Errorf("lock account: %w", err)
			}
		}

		return nil, nil, dto.ErrInvalidCredentials
	}

	// Successful login — clear rolling-window entries and legacy counter
	if err := s.userStore.ClearFailedAttempts(ctx, user.ID); err != nil {
		return nil, nil, fmt.Errorf("clear failed attempts: %w", err)
	}
	if err := s.userStore.ResetFailedLogin(ctx, user.ID); err != nil {
		return nil, nil, fmt.Errorf("reset failed login: %w", err)
	}

	userRoles, err := s.userStore.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("get roles: %w", err)
	}
	roleNames := make([]string, len(userRoles))
	for i, r := range userRoles {
		roleNames[i] = r.RoleName
	}

	accessToken, err := s.generateAccessToken(user.ID.String(), roleNames)
	if err != nil {
		return nil, nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshTokenStr := uuid.New().String()
	familyID := uuid.New()
	tokenHash := sha256.Sum256([]byte(refreshTokenStr))

	rt := &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash[:],
		FamilyID:  familyID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.userStore.CreateRefreshToken(ctx, rt); err != nil {
		return nil, nil, fmt.Errorf("create refresh token: %w", err)
	}

	csrfToken, err := middleware.GenerateCSRFToken()
	if err != nil {
		return nil, nil, fmt.Errorf("generate CSRF token: %w", err)
	}

	cookies := []*http.Cookie{
		{
			Name: "access_token", Value: accessToken, Path: "/",
			HttpOnly: true, Secure: false, SameSite: http.SameSiteStrictMode, MaxAge: 900,
		},
		{
			Name: "refresh_token", Value: refreshTokenStr, Path: "/api/auth",
			HttpOnly: true, Secure: false, SameSite: http.SameSiteStrictMode, MaxAge: 604800,
		},
		{
			Name: "csrf_token", Value: csrfToken, Path: "/",
			HttpOnly: false, Secure: false, SameSite: http.SameSiteStrictMode, MaxAge: 604800,
		},
	}

	resp := &dto.AuthResponse{
		User: dto.UserResponse{
			ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
			IsLocked: user.IsLocked, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
		},
		Roles: roleNames,
	}
	return resp, cookies, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshTokenStr string) ([]*http.Cookie, error) {
	tokenHash := sha256.Sum256([]byte(refreshTokenStr))
	rt, err := s.userStore.GetRefreshTokenByHash(ctx, tokenHash[:])
	if err != nil {
		return nil, fmt.Errorf("lookup refresh token: %w", err)
	}
	if rt == nil {
		return nil, dto.ErrUnauthorized
	}

	if rt.RevokedAt != nil {
		_ = s.userStore.RevokeRefreshTokenFamily(ctx, rt.FamilyID)
		return nil, dto.ErrUnauthorized
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, dto.ErrUnauthorized
	}

	if err := s.userStore.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("revoke token: %w", err)
	}

	user, err := s.userStore.GetByID(ctx, rt.UserID)
	if err != nil || user == nil {
		return nil, dto.ErrUnauthorized
	}

	userRoles, err := s.userStore.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get roles: %w", err)
	}
	roleNames := make([]string, len(userRoles))
	for i, r := range userRoles {
		roleNames[i] = r.RoleName
	}

	accessToken, err := s.generateAccessToken(user.ID.String(), roleNames)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	newRefreshStr := uuid.New().String()
	newHash := sha256.Sum256([]byte(newRefreshStr))
	newRT := &model.RefreshToken{
		ID: uuid.New(), UserID: rt.UserID, TokenHash: newHash[:],
		FamilyID: rt.FamilyID, ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.userStore.CreateRefreshToken(ctx, newRT); err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}

	csrfToken, err := middleware.GenerateCSRFToken()
	if err != nil {
		return nil, fmt.Errorf("generate CSRF token: %w", err)
	}

	cookies := []*http.Cookie{
		{Name: "access_token", Value: accessToken, Path: "/", HttpOnly: true, Secure: false, SameSite: http.SameSiteStrictMode, MaxAge: 900},
		{Name: "refresh_token", Value: newRefreshStr, Path: "/api/auth", HttpOnly: true, Secure: false, SameSite: http.SameSiteStrictMode, MaxAge: 604800},
		{Name: "csrf_token", Value: csrfToken, Path: "/", HttpOnly: false, Secure: false, SameSite: http.SameSiteStrictMode, MaxAge: 604800},
	}
	return cookies, nil
}

// GetCurrentUser returns the user profile and roles for the already-authenticated
// user. Used on app init to restore session state after page reload.
func (s *AuthService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*dto.AuthResponse, error) {
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, dto.ErrUnauthorized
	}
	userRoles, err := s.userStore.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get roles: %w", err)
	}
	roleNames := make([]string, len(userRoles))
	for i, r := range userRoles {
		roleNames[i] = r.RoleName
	}
	return &dto.AuthResponse{
		User: dto.UserResponse{
			ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
			IsLocked: user.IsLocked,
			CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
		},
		Roles: roleNames,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) []*http.Cookie {
	_ = s.userStore.RevokeAllUserTokens(ctx, userID)
	return []*http.Cookie{
		{Name: "access_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true},
		{Name: "refresh_token", Value: "", Path: "/api/auth", MaxAge: -1, HttpOnly: true},
		{Name: "csrf_token", Value: "", Path: "/", MaxAge: -1},
	}
}

func (s *AuthService) generateAccessToken(userID string, roles []string) (string, error) {
	claims := &middleware.UserClaims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ledgermint",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.signingKey)
}
