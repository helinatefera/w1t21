package dto

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Uses the same validator instance as the production handler (handler/helpers.go).
var v = validator.New()

// --- CreateUserRequest ---

func TestCreateUserRequest_Valid(t *testing.T) {
	req := CreateUserRequest{
		Username: "alice", Password: "password123", DisplayName: "Alice", Email: "alice@example.com",
	}
	if err := v.Struct(req); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestCreateUserRequest_UsernameMinBoundary(t *testing.T) {
	req := CreateUserRequest{Username: "abc", Password: "password123", DisplayName: "A"}
	if err := v.Struct(req); err != nil {
		t.Fatalf("3-char username should be valid: %v", err)
	}
}

func TestCreateUserRequest_UsernameBelowMin(t *testing.T) {
	req := CreateUserRequest{Username: "ab", Password: "password123", DisplayName: "A"}
	if err := v.Struct(req); err == nil {
		t.Fatal("2-char username should fail")
	}
}

func TestCreateUserRequest_UsernameAboveMax(t *testing.T) {
	req := CreateUserRequest{Username: strings.Repeat("a", 101), Password: "password123", DisplayName: "A"}
	if err := v.Struct(req); err == nil {
		t.Fatal("101-char username should fail")
	}
}

func TestCreateUserRequest_UsernameRequired(t *testing.T) {
	req := CreateUserRequest{Password: "password123", DisplayName: "A"}
	if err := v.Struct(req); err == nil {
		t.Fatal("empty username should fail")
	}
}

func TestCreateUserRequest_PasswordMinBoundary(t *testing.T) {
	req := CreateUserRequest{Username: "alice", Password: "12345678", DisplayName: "A"}
	if err := v.Struct(req); err != nil {
		t.Fatalf("8-char password should be valid: %v", err)
	}
}

func TestCreateUserRequest_PasswordBelowMin(t *testing.T) {
	req := CreateUserRequest{Username: "alice", Password: "1234567", DisplayName: "A"}
	if err := v.Struct(req); err == nil {
		t.Fatal("7-char password should fail")
	}
}

func TestCreateUserRequest_PasswordAboveMax(t *testing.T) {
	req := CreateUserRequest{Username: "alice", Password: strings.Repeat("a", 129), DisplayName: "A"}
	if err := v.Struct(req); err == nil {
		t.Fatal("129-char password should fail")
	}
}

func TestCreateUserRequest_EmailOptional(t *testing.T) {
	req := CreateUserRequest{Username: "alice", Password: "password123", DisplayName: "A"}
	if err := v.Struct(req); err != nil {
		t.Fatalf("empty email should be valid: %v", err)
	}
}

func TestCreateUserRequest_EmailInvalid(t *testing.T) {
	req := CreateUserRequest{Username: "alice", Password: "password123", DisplayName: "A", Email: "not-an-email"}
	if err := v.Struct(req); err == nil {
		t.Fatal("invalid email should fail")
	}
}

// --- CreateCollectibleRequest ---

func TestCreateCollectibleRequest_Valid(t *testing.T) {
	req := CreateCollectibleRequest{Title: "Dragon NFT", PriceCents: 9900, Currency: "USD"}
	if err := v.Struct(req); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
}

func TestCreateCollectibleRequest_TitleRequired(t *testing.T) {
	req := CreateCollectibleRequest{PriceCents: 100}
	if err := v.Struct(req); err == nil {
		t.Fatal("empty title should fail")
	}
}

func TestCreateCollectibleRequest_TitleAboveMax(t *testing.T) {
	req := CreateCollectibleRequest{Title: strings.Repeat("a", 301), PriceCents: 100}
	if err := v.Struct(req); err == nil {
		t.Fatal("301-char title should fail")
	}
}

func TestCreateCollectibleRequest_DescriptionOptionalAboveMax(t *testing.T) {
	req := CreateCollectibleRequest{Title: "ok", PriceCents: 100, Description: strings.Repeat("x", 5001)}
	if err := v.Struct(req); err == nil {
		t.Fatal("5001-char description should fail")
	}
}

func TestCreateCollectibleRequest_CurrencyExactLen(t *testing.T) {
	good := CreateCollectibleRequest{Title: "ok", PriceCents: 100, Currency: "USD"}
	if err := v.Struct(good); err != nil {
		t.Fatalf("3-char currency should be valid: %v", err)
	}
	bad := CreateCollectibleRequest{Title: "ok", PriceCents: 100, Currency: "US"}
	if err := v.Struct(bad); err == nil {
		t.Fatal("2-char currency should fail")
	}
}

func TestCreateCollectibleRequest_PriceCentsMin(t *testing.T) {
	ok := CreateCollectibleRequest{Title: "ok", PriceCents: 1}
	if err := v.Struct(ok); err != nil {
		t.Fatalf("price 1 should be valid: %v", err)
	}
	bad := CreateCollectibleRequest{Title: "ok", PriceCents: 0}
	if err := v.Struct(bad); err == nil {
		t.Fatal("price 0 should fail")
	}
}

func TestCreateCollectibleRequest_ContractAddressMax(t *testing.T) {
	ok := CreateCollectibleRequest{Title: "ok", PriceCents: 1, ContractAddress: "0x" + strings.Repeat("a", 40)}
	if err := v.Struct(ok); err != nil {
		t.Fatalf("42-char contract address should be valid: %v", err)
	}
	bad := CreateCollectibleRequest{Title: "ok", PriceCents: 1, ContractAddress: "0x" + strings.Repeat("a", 41)}
	if err := v.Struct(bad); err == nil {
		t.Fatal("43-char contract address should fail")
	}
}

// --- AddRoleRequest ---

func TestAddRoleRequest_ValidRoles(t *testing.T) {
	for _, role := range []string{"buyer", "seller", "administrator", "compliance_analyst"} {
		req := AddRoleRequest{RoleName: role}
		if err := v.Struct(req); err != nil {
			t.Errorf("role %q should be valid: %v", role, err)
		}
	}
}

func TestAddRoleRequest_InvalidRole(t *testing.T) {
	req := AddRoleRequest{RoleName: "superadmin"}
	if err := v.Struct(req); err == nil {
		t.Fatal("invalid role should fail")
	}
}

func TestAddRoleRequest_EmptyRole(t *testing.T) {
	req := AddRoleRequest{}
	if err := v.Struct(req); err == nil {
		t.Fatal("empty role should fail")
	}
}

// --- CreateIPRuleRequest ---

func TestCreateIPRuleRequest_ValidActions(t *testing.T) {
	for _, action := range []string{"allow", "deny"} {
		req := CreateIPRuleRequest{CIDR: "10.0.0.0/8", Action: action}
		if err := v.Struct(req); err != nil {
			t.Errorf("action %q should be valid: %v", action, err)
		}
	}
}

func TestCreateIPRuleRequest_InvalidAction(t *testing.T) {
	req := CreateIPRuleRequest{CIDR: "10.0.0.0/8", Action: "block"}
	if err := v.Struct(req); err == nil {
		t.Fatal("action 'block' should fail")
	}
}

// --- CreateABTestRequest ---

func TestCreateABTestRequest_TrafficPctBounds(t *testing.T) {
	base := CreateABTestRequest{
		Name: "test", TrafficPct: 50, StartDate: "2025-01-01", EndDate: "2025-12-31",
		ControlVariant: "a", TestVariant: "b", RollbackThresholdPct: 10,
	}

	base.TrafficPct = 1
	if err := v.Struct(base); err != nil {
		t.Fatalf("traffic 1 should be valid: %v", err)
	}
	base.TrafficPct = 100
	if err := v.Struct(base); err != nil {
		t.Fatalf("traffic 100 should be valid: %v", err)
	}
	base.TrafficPct = 0
	if err := v.Struct(base); err == nil {
		t.Fatal("traffic 0 should fail")
	}
	base.TrafficPct = 101
	if err := v.Struct(base); err == nil {
		t.Fatal("traffic 101 should fail")
	}
}

// --- CancelOrderRequest ---

func TestCancelOrderRequest_ReasonRequired(t *testing.T) {
	req := CancelOrderRequest{}
	if err := v.Struct(req); err == nil {
		t.Fatal("empty reason should fail")
	}
}

func TestCancelOrderRequest_ReasonMax(t *testing.T) {
	req := CancelOrderRequest{Reason: strings.Repeat("x", 501)}
	if err := v.Struct(req); err == nil {
		t.Fatal("501-char reason should fail")
	}
}

// --- SendMessageRequest ---

func TestSendMessageRequest_BodyRequired(t *testing.T) {
	req := SendMessageRequest{}
	if err := v.Struct(req); err == nil {
		t.Fatal("empty body should fail")
	}
}

func TestSendMessageRequest_BodyMax(t *testing.T) {
	req := SendMessageRequest{Body: strings.Repeat("x", 10001)}
	if err := v.Struct(req); err == nil {
		t.Fatal("10001-char body should fail")
	}
}

func TestSendMessageRequest_BodyAtMax(t *testing.T) {
	req := SendMessageRequest{Body: strings.Repeat("x", 10000)}
	if err := v.Struct(req); err != nil {
		t.Fatalf("10000-char body should be valid: %v", err)
	}
}

// --- CreateOrderRequest ---

func TestCreateOrderRequest_CollectibleIDRequired(t *testing.T) {
	req := CreateOrderRequest{}
	if err := v.Struct(req); err == nil {
		t.Fatal("zero collectible ID should fail")
	}
}

func TestCreateOrderRequest_Valid(t *testing.T) {
	req := CreateOrderRequest{CollectibleID: uuid.New()}
	if err := v.Struct(req); err != nil {
		t.Fatalf("valid request should pass: %v", err)
	}
}

// --- HideCollectibleRequest ---

func TestHideCollectibleRequest_ReasonRequired(t *testing.T) {
	req := HideCollectibleRequest{}
	if err := v.Struct(req); err == nil {
		t.Fatal("empty reason should fail")
	}
}

func TestHideCollectibleRequest_ReasonMax(t *testing.T) {
	req := HideCollectibleRequest{Reason: strings.Repeat("x", 501)}
	if err := v.Struct(req); err == nil {
		t.Fatal("501-char reason should fail")
	}
}
