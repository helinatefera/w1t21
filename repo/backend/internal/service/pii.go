package service

import (
	"regexp"
	"strings"
)

var (
	ssnRegex   = regexp.MustCompile(`\b\d{3}-?\d{2}-?\d{4}\b`)
	phoneRegex = regexp.MustCompile(`\b(\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`)
	emailRegex = regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)
)

// DetectPII checks text for sensitive data patterns and returns detected types.
func DetectPII(text string) (bool, []string) {
	var types []string
	if ssnRegex.MatchString(text) {
		types = append(types, "SSN")
	}
	if phoneRegex.MatchString(text) {
		types = append(types, "phone number")
	}
	if emailRegex.MatchString(text) {
		types = append(types, "email address")
	}
	return len(types) > 0, types
}

// PIIErrorMessage returns a user-friendly error message for PII detection.
func PIIErrorMessage(types []string) string {
	return "message blocked: contains sensitive personal information (" + strings.Join(types, ", ") + ")"
}
