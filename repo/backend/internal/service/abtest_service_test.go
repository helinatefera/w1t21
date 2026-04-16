package service

import (
	"testing"
	"time"
)

func TestParseFlexibleDate_DatetimeLocal(t *testing.T) {
	result, err := parseFlexibleDate("2025-01-01T00:00")
	if err != nil {
		t.Fatalf("datetime-local format should be accepted: %v", err)
	}
	if result.Year() != 2025 || result.Month() != time.January || result.Day() != 1 {
		t.Errorf("expected 2025-01-01, got %v", result)
	}
}

func TestParseFlexibleDate_RFC3339(t *testing.T) {
	result, err := parseFlexibleDate("2025-06-15T14:30:00Z")
	if err != nil {
		t.Fatalf("RFC3339 format should be accepted: %v", err)
	}
	if result.Year() != 2025 || result.Month() != time.June || result.Day() != 15 {
		t.Errorf("expected 2025-06-15, got %v", result)
	}
}

func TestParseFlexibleDate_ISOWithoutTimezone(t *testing.T) {
	result, err := parseFlexibleDate("2025-03-20T10:45:00")
	if err != nil {
		t.Fatalf("ISO without timezone should be accepted: %v", err)
	}
	if result.Hour() != 10 || result.Minute() != 45 {
		t.Errorf("expected 10:45, got %02d:%02d", result.Hour(), result.Minute())
	}
}

func TestParseFlexibleDate_USFormat(t *testing.T) {
	result, err := parseFlexibleDate("01/15/2025 3:00 PM")
	if err != nil {
		t.Fatalf("US format should be accepted: %v", err)
	}
	if result.Hour() != 15 {
		t.Errorf("expected hour 15, got %d", result.Hour())
	}
}

func TestParseFlexibleDate_InvalidFormat(t *testing.T) {
	_, err := parseFlexibleDate("not-a-date")
	if err == nil {
		t.Error("invalid date format should be rejected")
	}
}

func TestParseFlexibleDate_EmptyString(t *testing.T) {
	_, err := parseFlexibleDate("")
	if err == nil {
		t.Error("empty string should be rejected")
	}
}

func TestAssignVariant_SameInput_Deterministic(t *testing.T) {
	v1 := AssignVariant("user-1", "test-exp", 50)
	v2 := AssignVariant("user-1", "test-exp", 50)
	if v1 != v2 {
		t.Errorf("same user+test should produce same variant: %s vs %s", v1, v2)
	}
}

func TestAssignVariant_100PercentTraffic(t *testing.T) {
	v := AssignVariant("any-user", "test-name", 100)
	if v != "test" {
		t.Errorf("100%% traffic should always assign to test, got %s", v)
	}
}

func TestAssignVariant_ZeroPercentTraffic(t *testing.T) {
	// With 0% traffic pct, hash%100 >= 0 always, so always control
	v := AssignVariant("any-user", "test-name", 0)
	if v != "control" {
		t.Errorf("0%% traffic should always assign to control, got %s", v)
	}
}

func TestAssignVariant_DifferentUsers(t *testing.T) {
	// With enough users at 50%, we should get both variants
	gotTest := false
	gotControl := false
	for i := 0; i < 100; i++ {
		v := AssignVariant("user-"+string(rune('A'+i)), "split-test", 50)
		if v == "test" {
			gotTest = true
		} else {
			gotControl = true
		}
		if gotTest && gotControl {
			break
		}
	}
	if !gotTest || !gotControl {
		t.Error("with 50% traffic and many users, should have both test and control")
	}
}
