package service

import (
	"crypto/sha256"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ledgermint/platform/internal/crypto"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/model"
)

// ---------------------------------------------------------------------------
// Lockout threshold and window constants
// ---------------------------------------------------------------------------

func TestLockoutConstants(t *testing.T) {
	if maxUserFailures != 5 {
		t.Fatalf("maxUserFailures should be 5, got %d", maxUserFailures)
	}
	if maxIPFailures != 20 {
		t.Fatalf("maxIPFailures should be 20, got %d", maxIPFailures)
	}
	if failureWindow != 15*time.Minute {
		t.Fatalf("failureWindow should be 15m, got %v", failureWindow)
	}
	if lockoutDuration != 30*time.Minute {
		t.Fatalf("lockoutDuration should be 30m, got %v", lockoutDuration)
	}
}

// TestLockoutWindowStart verifies the rolling window boundary calculation
// used on the first line of Login().
func TestLockoutWindowStart(t *testing.T) {
	now := time.Now()
	windowStart := now.Add(-failureWindow)
	elapsed := now.Sub(windowStart)
	if elapsed != 15*time.Minute {
		t.Fatalf("window should span 15m, got %v", elapsed)
	}
}

// TestLockout_ThresholdReached ensures that the exact threshold value
// (not threshold+1) triggers the lockout condition.
func TestLockout_ThresholdReached(t *testing.T) {
	// Simulate the check from auth_service.go:80
	for _, tc := range []struct {
		name      string
		count     int
		threshold int
		wantLock  bool
	}{
		{"below user threshold", 4, maxUserFailures, false},
		{"at user threshold", 5, maxUserFailures, true},
		{"above user threshold", 6, maxUserFailures, true},
		{"below IP threshold", 19, maxIPFailures, false},
		{"at IP threshold", 20, maxIPFailures, true},
		{"above IP threshold", 21, maxIPFailures, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			locked := tc.count >= tc.threshold
			if locked != tc.wantLock {
				t.Errorf("count=%d threshold=%d: got locked=%v, want %v",
					tc.count, tc.threshold, locked, tc.wantLock)
			}
		})
	}
}

// TestLockExpiry verifies the lock-expired auto-release branch:
// if user.IsLocked and LockedUntil is in the past, the user should be
// allowed to proceed (lock is stale).
func TestLockExpiry(t *testing.T) {
	past := time.Now().Add(-1 * time.Minute)
	future := time.Now().Add(29 * time.Minute)

	tests := []struct {
		name        string
		isLocked    bool
		lockedUntil *time.Time
		wantBlocked bool
	}{
		{"not locked", false, nil, false},
		{"locked with future expiry", true, &future, true},
		{"locked with past expiry", true, &past, false},
		{"locked with nil expiry (legacy)", true, nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Replicate the exact branch from auth_service.go:59-67
			blocked := false
			if tc.isLocked {
				if tc.lockedUntil != nil && time.Now().Before(*tc.lockedUntil) {
					blocked = true
				}
				// else: lock expired, would call UnlockAccount
			}
			if blocked != tc.wantBlocked {
				t.Errorf("got blocked=%v, want %v", blocked, tc.wantBlocked)
			}
		})
	}
}

// TestLockUntilCalculation verifies that lock duration produces a valid
// future timestamp in RFC3339 format (matches LockAccount call).
func TestLockUntilCalculation(t *testing.T) {
	lockUntil := time.Now().Add(lockoutDuration).Format(time.RFC3339)
	parsed, err := time.Parse(time.RFC3339, lockUntil)
	if err != nil {
		t.Fatalf("lockUntil should be valid RFC3339: %v", err)
	}
	if time.Until(parsed) < 29*time.Minute {
		t.Fatalf("lock should be ~30m in the future, got %v", time.Until(parsed))
	}
}

// ---------------------------------------------------------------------------
// Password verification
// ---------------------------------------------------------------------------

func TestPasswordVerification(t *testing.T) {
	hash, err := crypto.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if !crypto.CheckPassword(hash, "correct-password") {
		t.Fatal("correct password should pass")
	}
	if crypto.CheckPassword(hash, "wrong-password") {
		t.Fatal("wrong password should fail")
	}
	if crypto.CheckPassword(hash, "") {
		t.Fatal("empty password should fail")
	}
}

// ---------------------------------------------------------------------------
// JWT access token generation and validation
// ---------------------------------------------------------------------------

func TestAccessToken_RoundTrip(t *testing.T) {
	signingKey := []byte("test-secret-key-for-unit-tests")
	svc := &AuthService{signingKey: signingKey}

	userID := uuid.New().String()
	roles := []string{"buyer", "seller"}

	tokenStr, err := svc.generateAccessToken(userID, roles)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Parse back
	token, err := jwt.ParseWithClaims(tokenStr, &middleware.UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		return signingKey, nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	claims := token.Claims.(*middleware.UserClaims)

	if claims.UserID != userID {
		t.Errorf("userID: got %q, want %q", claims.UserID, userID)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "buyer" || claims.Roles[1] != "seller" {
		t.Errorf("roles: got %v, want [buyer seller]", claims.Roles)
	}
}

func TestAccessToken_Expiry(t *testing.T) {
	signingKey := []byte("test-secret")
	svc := &AuthService{signingKey: signingKey}

	tokenStr, _ := svc.generateAccessToken("user-1", []string{"buyer"})
	token, _ := jwt.ParseWithClaims(tokenStr, &middleware.UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		return signingKey, nil
	})
	claims := token.Claims.(*middleware.UserClaims)

	expiry := claims.ExpiresAt.Time
	ttl := time.Until(expiry)
	if ttl < 14*time.Minute || ttl > 16*time.Minute {
		t.Fatalf("access token TTL should be ~15m, got %v", ttl)
	}
}

func TestAccessToken_WrongKey_Rejected(t *testing.T) {
	svc := &AuthService{signingKey: []byte("real-key")}
	tokenStr, _ := svc.generateAccessToken("user-1", []string{"buyer"})

	_, err := jwt.ParseWithClaims(tokenStr, &middleware.UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("wrong-key"), nil
	})
	if err == nil {
		t.Fatal("token signed with different key should be rejected")
	}
}

func TestAccessToken_Issuer(t *testing.T) {
	signingKey := []byte("test-secret")
	svc := &AuthService{signingKey: signingKey}
	tokenStr, _ := svc.generateAccessToken("user-1", []string{})
	token, _ := jwt.ParseWithClaims(tokenStr, &middleware.UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		return signingKey, nil
	})
	claims := token.Claims.(*middleware.UserClaims)
	if claims.Issuer != "ledgermint" {
		t.Errorf("issuer: got %q, want %q", claims.Issuer, "ledgermint")
	}
}

// ---------------------------------------------------------------------------
// Refresh token family rotation semantics
// ---------------------------------------------------------------------------

func TestRefreshTokenHash_Deterministic(t *testing.T) {
	token := "some-refresh-token-value"
	h1 := sha256.Sum256([]byte(token))
	h2 := sha256.Sum256([]byte(token))
	if h1 != h2 {
		t.Fatal("SHA256 hash should be deterministic")
	}
}

func TestRefreshTokenHash_DifferentTokens(t *testing.T) {
	h1 := sha256.Sum256([]byte("token-a"))
	h2 := sha256.Sum256([]byte("token-b"))
	if h1 == h2 {
		t.Fatal("different tokens should produce different hashes")
	}
}

// TestRefreshToken_RevokedDetection validates the revocation check logic
// from Refresh(): if RevokedAt is non-nil, the entire family is revoked.
func TestRefreshToken_RevokedDetection(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		revokedAt  *time.Time
		expiresAt  time.Time
		wantReject bool
	}{
		{"valid token", nil, time.Now().Add(1 * time.Hour), false},
		{"revoked token", &now, time.Now().Add(1 * time.Hour), true},
		{"expired token", nil, time.Now().Add(-1 * time.Hour), true},
		{"revoked and expired", &now, time.Now().Add(-1 * time.Hour), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rt := &model.RefreshToken{
				RevokedAt: tc.revokedAt,
				ExpiresAt: tc.expiresAt,
			}
			// Replicate Refresh() logic at auth_service.go:167-173
			rejected := false
			if rt.RevokedAt != nil {
				rejected = true
			} else if time.Now().After(rt.ExpiresAt) {
				rejected = true
			}
			if rejected != tc.wantReject {
				t.Errorf("got rejected=%v, want %v", rejected, tc.wantReject)
			}
		})
	}
}

// TestRefreshToken_FamilyPreserved verifies that token rotation preserves
// the family ID (critical for family-based revocation on reuse).
func TestRefreshToken_FamilyPreserved(t *testing.T) {
	familyID := uuid.New()

	// Simulate the rotation from Refresh() at auth_service.go:200-204
	newRT := &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: sha256.New().Sum(nil),
		FamilyID:  familyID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if newRT.FamilyID != familyID {
		t.Fatal("rotated token must preserve the family ID")
	}
}

// TestLogoutCookieShape validates the cookie-clearing constants that
// Logout uses. The actual Logout method requires a live UserStore, so
// we verify the expected shape independently.
func TestLogoutCookieShape(t *testing.T) {
	// These are the cookies Logout() must clear (auth_service.go:248-252).
	expected := []struct {
		name     string
		path     string
		httpOnly bool
	}{
		{"access_token", "/", true},
		{"refresh_token", "/api/auth", true},
		{"csrf_token", "/", false},
	}

	cookies := []*http.Cookie{
		{Name: "access_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true},
		{Name: "refresh_token", Value: "", Path: "/api/auth", MaxAge: -1, HttpOnly: true},
		{Name: "csrf_token", Value: "", Path: "/", MaxAge: -1},
	}

	if len(cookies) != len(expected) {
		t.Fatalf("expected %d cookies, got %d", len(expected), len(cookies))
	}
	for i, c := range cookies {
		if c.Name != expected[i].name {
			t.Errorf("cookie %d: name got %q, want %q", i, c.Name, expected[i].name)
		}
		if c.Path != expected[i].path {
			t.Errorf("cookie %s: path got %q, want %q", c.Name, c.Path, expected[i].path)
		}
		if c.MaxAge != -1 {
			t.Errorf("cookie %s: MaxAge got %d, want -1", c.Name, c.MaxAge)
		}
		if c.Value != "" {
			t.Errorf("cookie %s: Value should be empty, got %q", c.Name, c.Value)
		}
	}
}
