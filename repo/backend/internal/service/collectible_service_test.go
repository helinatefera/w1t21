package service

import (
	"testing"

	"github.com/ledgermint/platform/internal/dto"
)

func TestCollectibleCreate_IdentityValidation(t *testing.T) {
	tests := []struct {
		name            string
		contractAddress string
		chainID         int
		tokenID         string
		wantErr         bool
		errContains     string
	}{
		{
			name:            "contract_address without chain_id and token_id",
			contractAddress: "0xABC",
			chainID:         0,
			tokenID:         "",
			wantErr:         true,
			errContains:     "contract_address requires chain_id and token_id",
		},
		{
			name:            "contract_address with chain_id but no token_id",
			contractAddress: "0xABC",
			chainID:         1,
			tokenID:         "",
			wantErr:         true,
			errContains:     "contract_address requires chain_id and token_id",
		},
		{
			name:            "chain_id without token_id",
			contractAddress: "",
			chainID:         1,
			tokenID:         "",
			wantErr:         true,
			errContains:     "chain_id and token_id must both be provided",
		},
		{
			name:            "token_id without chain_id",
			contractAddress: "",
			chainID:         0,
			tokenID:         "tok-1",
			wantErr:         true,
			errContains:     "chain_id and token_id must both be provided",
		},
		{
			name:            "all identity fields provided",
			contractAddress: "0xABC",
			chainID:         137,
			tokenID:         "tok-1",
			wantErr:         false,
		},
		{
			name:            "no identity fields - auto-generated",
			contractAddress: "",
			chainID:         0,
			tokenID:         "",
			wantErr:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Only test the validation logic, not the DB call.
			// Replicate the validation from CollectibleService.Create.
			req := dto.CreateCollectibleRequest{
				Title:           "Test",
				PriceCents:      1000,
				ContractAddress: tc.contractAddress,
				ChainID:         tc.chainID,
				TokenID:         tc.tokenID,
			}

			err := validateCollectibleIdentity(req)
			if tc.wantErr && err == nil {
				t.Errorf("expected error containing %q, got nil", tc.errContains)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if tc.wantErr && err != nil {
				if !containsString(err.Error(), tc.errContains) {
					t.Errorf("expected error containing %q, got: %v", tc.errContains, err)
				}
			}
		})
	}
}

func TestHasAdminOrCompliance(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{"admin", []string{"administrator"}, true},
		{"compliance", []string{"compliance_analyst"}, true},
		{"buyer", []string{"buyer"}, false},
		{"seller", []string{"seller"}, false},
		{"empty", []string{}, false},
		{"multi with admin", []string{"seller", "administrator"}, true},
		{"buyer+seller", []string{"buyer", "seller"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasAdminOrCompliance(tc.roles)
			if got != tc.want {
				t.Errorf("hasAdminOrCompliance(%v) = %v, want %v", tc.roles, got, tc.want)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
