package service

import "testing"

func TestValidateExperiment_Valid(t *testing.T) {
	msg := ValidateExperiment("catalog_layout", "grid", "list")
	if msg != "" {
		t.Fatalf("expected valid, got: %s", msg)
	}
}

func TestValidateExperiment_UnknownName(t *testing.T) {
	msg := ValidateExperiment("nonexistent_experiment", "a", "b")
	if msg == "" {
		t.Fatal("expected error for unknown experiment")
	}
	if !contains(msg, "unknown experiment") {
		t.Fatalf("expected 'unknown experiment' in message, got: %s", msg)
	}
}

func TestValidateExperiment_BadControlVariant(t *testing.T) {
	msg := ValidateExperiment("catalog_layout", "carousel", "list")
	if msg == "" {
		t.Fatal("expected error for bad control variant")
	}
	if !contains(msg, "control variant") {
		t.Fatalf("expected 'control variant' in message, got: %s", msg)
	}
}

func TestValidateExperiment_BadTestVariant(t *testing.T) {
	msg := ValidateExperiment("catalog_layout", "grid", "carousel")
	if msg == "" {
		t.Fatal("expected error for bad test variant")
	}
	if !contains(msg, "test variant") {
		t.Fatalf("expected 'test variant' in message, got: %s", msg)
	}
}

func TestValidateExperiment_SameVariants(t *testing.T) {
	msg := ValidateExperiment("catalog_layout", "grid", "grid")
	if msg == "" {
		t.Fatal("expected error for same variants")
	}
	if !contains(msg, "must be different") {
		t.Fatalf("expected 'must be different' in message, got: %s", msg)
	}
}

func TestValidateExperiment_CheckoutFlow(t *testing.T) {
	msg := ValidateExperiment("checkout_flow", "standard", "express")
	if msg != "" {
		t.Fatalf("expected valid, got: %s", msg)
	}
}

func TestValidateExperiment_SearchRanking(t *testing.T) {
	msg := ValidateExperiment("search_ranking", "relevance", "popular")
	if msg != "" {
		t.Fatalf("expected valid, got: %s", msg)
	}
}

func TestAssignVariant_Deterministic(t *testing.T) {
	v1 := AssignVariant("user-123", "catalog_layout", 50)
	v2 := AssignVariant("user-123", "catalog_layout", 50)
	if v1 != v2 {
		t.Fatalf("not deterministic: %s != %s", v1, v2)
	}
}

func TestAssignVariant_FullTraffic(t *testing.T) {
	for i := 0; i < 50; i++ {
		v := AssignVariant("user", "test", 100)
		if v != "test" {
			t.Fatalf("100%% traffic should yield test, got: %s", v)
		}
	}
}

func TestAssignVariant_ZeroTraffic(t *testing.T) {
	for i := 0; i < 50; i++ {
		v := AssignVariant("user", "test", 0)
		if v != "control" {
			t.Fatalf("0%% traffic should yield control, got: %s", v)
		}
	}
}

func TestAssignVariant_BothVariantsAppear(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		v := AssignVariant(string(rune('A'+i%26))+string(rune('0'+i/26)), "split", 50)
		seen[v] = true
	}
	if !seen["test"] || !seen["control"] {
		t.Fatalf("expected both test and control, got: %v", seen)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && containsSubstr(s, sub)
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
