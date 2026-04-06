package service

import "testing"

func TestDetectPII_SSN(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"SSN: 123-45-6789", true},
		{"SSN: 123456789", true},
		{"My number is 123-45-6789 ok", true},
		{"No PII here", false},
	}
	for _, tc := range tests {
		detected, types := DetectPII(tc.input)
		if tc.want && !detected {
			t.Errorf("expected SSN detection in %q", tc.input)
		}
		if tc.want && !sliceContains(types, "SSN") {
			t.Errorf("expected 'SSN' in types for %q, got %v", tc.input, types)
		}
	}
}

func TestDetectPII_Phone(t *testing.T) {
	tests := []string{
		"555-123-4567",
		"(555) 123-4567",
		"+1 555-123-4567",
		"555.123.4567",
	}
	for _, input := range tests {
		_, types := DetectPII(input)
		if !sliceContains(types, "phone number") {
			t.Errorf("expected phone detection in %q, got %v", input, types)
		}
	}
}

func TestDetectPII_Email(t *testing.T) {
	tests := []string{
		"user@example.com",
		"user+tag@example.com",
		"u@sub.example.com",
	}
	for _, input := range tests {
		_, types := DetectPII(input)
		if !sliceContains(types, "email address") {
			t.Errorf("expected email detection in %q, got %v", input, types)
		}
	}
}

func TestDetectPII_Multiple(t *testing.T) {
	_, types := DetectPII("SSN 123-45-6789 phone 555-123-4567 email a@b.com")
	if len(types) != 3 {
		t.Fatalf("expected 3 types, got %v", types)
	}
}

func TestDetectPII_Clean(t *testing.T) {
	detected, _ := DetectPII("Hello world, no PII here")
	if detected {
		t.Error("expected no PII detection in clean text")
	}
}

func sliceContains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
