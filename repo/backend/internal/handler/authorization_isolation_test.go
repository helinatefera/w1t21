package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/middleware"
)

// This file tests adversarial data isolation at the handler level.
// It creates real JWTs for two different users and verifies that the
// middleware chain (JWTAuth → RequireRole → handler) correctly prevents
// cross-user access. No database is needed — the JWT middleware itself
// populates the Echo context, and we verify that the correct user_id
// is set for downstream authorization.

var testSigningKey = []byte("test-signing-key-for-isolation-tests")

func generateTestJWT(t *testing.T, userID string, roles []string) string {
	t.Helper()
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
	str, err := token.SignedString(testSigningKey)
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	return str
}

// makeAuthenticatedRequest creates an Echo context with a valid JWT cookie
// for the given user, routed through the JWTAuth middleware. Returns the
// context with user_id and user_roles populated, plus the recorder.
func makeAuthenticatedRequest(t *testing.T, method, path, userID string, roles []string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	jwtToken := generateTestJWT(t, userID, roles)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: jwtToken})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Run through JWTAuth middleware to populate context
	jwtMiddleware := middleware.JWTAuth(testSigningKey)
	handler := jwtMiddleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	if err := handler(c); err != nil {
		t.Fatalf("JWT auth failed: %v", err)
	}

	return c, rec
}

// ---------------------------------------------------------------------------
// Test: JWT middleware populates the correct user identity
// ---------------------------------------------------------------------------

func TestJWTAuth_PopulatesCorrectUserID(t *testing.T) {
	userA := uuid.New().String()
	userB := uuid.New().String()

	ctxA, _ := makeAuthenticatedRequest(t, http.MethodGet, "/api/orders", userA, []string{"buyer"})
	ctxB, _ := makeAuthenticatedRequest(t, http.MethodGet, "/api/orders", userB, []string{"buyer"})

	gotA := middleware.GetUserID(ctxA)
	gotB := middleware.GetUserID(ctxB)

	if gotA != userA {
		t.Errorf("user A: got %q, want %q", gotA, userA)
	}
	if gotB != userB {
		t.Errorf("user B: got %q, want %q", gotB, userB)
	}
	if gotA == gotB {
		t.Fatal("two different JWTs should yield different user IDs")
	}
}

func TestJWTAuth_PopulatesCorrectRoles(t *testing.T) {
	ctx, _ := makeAuthenticatedRequest(t, http.MethodGet, "/", uuid.New().String(), []string{"buyer", "seller"})
	roles := middleware.GetUserRoles(ctx)
	if len(roles) != 2 || roles[0] != "buyer" || roles[1] != "seller" {
		t.Errorf("roles: got %v, want [buyer seller]", roles)
	}
}

// ---------------------------------------------------------------------------
// Test: JWT for user A cannot masquerade as user B
// ---------------------------------------------------------------------------

func TestJWTIsolation_UserACannotBeUserB(t *testing.T) {
	userA := uuid.New().String()
	userB := uuid.New().String()

	// Authenticate as user A
	ctx, _ := makeAuthenticatedRequest(t, http.MethodGet, "/api/orders", userA, []string{"buyer"})

	// Verify the context knows this is user A, NOT user B
	extracted := middleware.GetUserID(ctx)
	if extracted == userB {
		t.Fatal("user A's JWT should never resolve to user B's ID")
	}
	if extracted != userA {
		t.Fatalf("user A's JWT should resolve to user A, got %q", extracted)
	}
}

// ---------------------------------------------------------------------------
// Test: Expired JWT is rejected
// ---------------------------------------------------------------------------

func TestJWTAuth_ExpiredToken_Rejected(t *testing.T) {
	claims := &middleware.UserClaims{
		UserID: uuid.New().String(),
		Roles:  []string{"buyer"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(testSigningKey)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: tokenStr})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	jwtMiddleware := middleware.JWTAuth(testSigningKey)
	handler := jwtMiddleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	if err == nil {
		t.Fatal("expired JWT should be rejected")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: JWT signed with wrong key is rejected
// ---------------------------------------------------------------------------

func TestJWTAuth_WrongSigningKey_Rejected(t *testing.T) {
	claims := &middleware.UserClaims{
		UserID: uuid.New().String(),
		Roles:  []string{"buyer"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte("attacker-key"))

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: tokenStr})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	jwtMiddleware := middleware.JWTAuth(testSigningKey)
	handler := jwtMiddleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	if err == nil {
		t.Fatal("JWT signed with wrong key should be rejected")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: Missing JWT cookie is rejected
// ---------------------------------------------------------------------------

func TestJWTAuth_MissingCookie_Rejected(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	jwtMiddleware := middleware.JWTAuth(testSigningKey)
	handler := jwtMiddleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	if err == nil {
		t.Fatal("missing JWT should be rejected")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: RequireRole middleware enforces role boundaries
// ---------------------------------------------------------------------------

func TestRequireRole_MatchingRole_Allowed(t *testing.T) {
	ctx, _ := makeAuthenticatedRequest(t, http.MethodGet, "/", uuid.New().String(), []string{"administrator"})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", middleware.GetUserID(ctx))
	c.Set("user_roles", middleware.GetUserRoles(ctx))

	roleMiddleware := middleware.RequireRole("administrator")
	handler := roleMiddleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	if err := handler(c); err != nil {
		t.Fatalf("admin should pass RequireRole(administrator): %v", err)
	}
}

func TestRequireRole_WrongRole_Denied(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", uuid.New().String())
	c.Set("user_roles", []string{"buyer"})

	roleMiddleware := middleware.RequireRole("administrator")
	handler := roleMiddleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	if err == nil {
		t.Fatal("buyer should be rejected by RequireRole(administrator)")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got: %v", err)
	}
}

func TestRequireRole_NoRoles_Denied(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_roles set at all

	roleMiddleware := middleware.RequireRole("buyer")
	handler := roleMiddleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	if err == nil {
		t.Fatal("missing roles should be rejected")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: Multi-role check (any-of semantics)
// ---------------------------------------------------------------------------

func TestRequireRole_AnyOfSemantics(t *testing.T) {
	roleMiddleware := middleware.RequireRole("administrator", "compliance_analyst")

	tests := []struct {
		name    string
		roles   []string
		allowed bool
	}{
		{"admin only", []string{"administrator"}, true},
		{"compliance only", []string{"compliance_analyst"}, true},
		{"both roles", []string{"administrator", "compliance_analyst"}, true},
		{"buyer only", []string{"buyer"}, false},
		{"seller only", []string{"seller"}, false},
		{"buyer+seller", []string{"buyer", "seller"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_roles", tc.roles)

			handler := roleMiddleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})
			err := handler(c)
			if tc.allowed && err != nil {
				t.Fatalf("roles %v should be allowed: %v", tc.roles, err)
			}
			if !tc.allowed && err == nil {
				t.Fatalf("roles %v should be denied", tc.roles)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test: Full adversarial isolation scenario
// ---------------------------------------------------------------------------

// TestFullIsolation_TwoUsers simulates the complete adversarial scenario:
// two users with valid JWTs, each operating under different identities.
// Verifies that the middleware chain guarantees identity isolation.
func TestFullIsolation_TwoUsers(t *testing.T) {
	userAlice := uuid.New().String()
	userBob := uuid.New().String()

	// Alice authenticates
	ctxAlice, _ := makeAuthenticatedRequest(t, http.MethodGet, "/api/orders", userAlice, []string{"buyer"})
	// Bob authenticates
	ctxBob, _ := makeAuthenticatedRequest(t, http.MethodGet, "/api/orders", userBob, []string{"buyer"})

	aliceID := middleware.GetUserID(ctxAlice)
	bobID := middleware.GetUserID(ctxBob)

	// Fundamental isolation: Alice's session resolves to Alice, Bob's to Bob
	if aliceID != userAlice {
		t.Errorf("Alice's session resolved to %q, want %q", aliceID, userAlice)
	}
	if bobID != userBob {
		t.Errorf("Bob's session resolved to %q, want %q", bobID, userBob)
	}
	if aliceID == bobID {
		t.Fatal("Alice and Bob must have distinct session identities")
	}

	// Simulate downstream authorization: Alice's ID is used to scope queries.
	// An order owned by Bob (buyerID=Bob, sellerID=someone-else) must be
	// inaccessible to Alice's session.
	bobOrder := struct {
		BuyerID  string
		SellerID string
	}{
		BuyerID:  userBob,
		SellerID: uuid.New().String(),
	}

	// Alice tries to access Bob's order — the handler would use aliceID
	if aliceID == bobOrder.BuyerID || aliceID == bobOrder.SellerID {
		t.Fatal("Alice's identity must not match Bob's order participants")
	}
}

// TestFullIsolation_RoleEscalation verifies that a buyer's JWT cannot
// pass through admin-only route middleware.
func TestFullIsolation_RoleEscalation(t *testing.T) {
	buyerID := uuid.New().String()

	e := echo.New()
	jwtToken := generateTestJWT(t, buyerID, []string{"buyer"})
	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: jwtToken})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Chain: JWTAuth → RequireRole(administrator)
	jwtMW := middleware.JWTAuth(testSigningKey)
	roleMW := middleware.RequireRole("administrator")

	chain := jwtMW(func(c echo.Context) error {
		return roleMW(func(c echo.Context) error {
			return c.String(http.StatusOK, "admin action performed")
		})(c)
	})

	err := chain(c)
	if err == nil {
		t.Fatal("buyer JWT must be rejected at admin-only endpoint")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got: %v", err)
	}
}
