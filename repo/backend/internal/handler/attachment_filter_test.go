package handler

import (
	"testing"

	"github.com/ledgermint/platform/internal/service"
)

func TestIsTextAttachment(t *testing.T) {
	tests := []struct {
		mime, ext string
		want      bool
	}{
		{"text/plain", ".txt", true},
		{"text/csv", ".csv", true},
		{"text/html", ".html", true},
		{"application/octet-stream", ".csv", true},
		{"application/octet-stream", ".txt", true},
		{"image/png", ".png", false},
		{"application/pdf", ".pdf", false},
	}
	for _, tc := range tests {
		got := isTextAttachment(tc.mime, tc.ext)
		if got != tc.want {
			t.Errorf("isTextAttachment(%q, %q) = %v, want %v", tc.mime, tc.ext, got, tc.want)
		}
	}
}

func TestIsSafeBinaryAttachment(t *testing.T) {
	tests := []struct {
		mime, ext string
		want      bool
	}{
		// Allowed binary types
		{"image/jpeg", ".jpg", true},
		{"image/png", ".png", true},
		{"image/gif", ".gif", true},
		{"image/webp", ".webp", true},
		{"image/svg+xml", ".svg", true},
		{"application/pdf", ".pdf", true},

		// Allowed by extension fallback when MIME is generic
		{"application/octet-stream", ".jpg", true},
		{"application/octet-stream", ".jpeg", true},
		{"application/octet-stream", ".png", true},
		{"application/octet-stream", ".pdf", true},

		// Rejected — unscannable formats that could contain PII
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", ".docx", false},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ".xlsx", false},
		{"application/zip", ".zip", false},
		{"application/octet-stream", ".docx", false},
		{"application/octet-stream", ".exe", false},
		{"application/x-executable", ".bin", false},
	}
	for _, tc := range tests {
		got := isSafeBinaryAttachment(tc.mime, tc.ext)
		if got != tc.want {
			t.Errorf("isSafeBinaryAttachment(%q, %q) = %v, want %v", tc.mime, tc.ext, got, tc.want)
		}
	}
}

func TestAttachmentDecision_AllPathsCovered(t *testing.T) {
	// Verify the two-stage decision: type allowed, then PII-scanned.
	// All accepted types (text OR safeBinary) are PII-scanned uniformly.
	cases := []struct {
		mime, ext string
		text      bool
		safeBin   bool
		accepted  bool
	}{
		{"text/plain", ".txt", true, false, true},
		{"image/png", ".png", false, true, true},
		{"application/pdf", ".pdf", false, true, true},
		{"image/svg+xml", ".svg", false, true, true},
		{"application/zip", ".zip", false, false, false},
		{"application/octet-stream", ".docx", false, false, false},
	}
	for _, tc := range cases {
		text := isTextAttachment(tc.mime, tc.ext)
		safeBin := isSafeBinaryAttachment(tc.mime, tc.ext)
		accepted := text || safeBin
		if text != tc.text {
			t.Errorf("isTextAttachment(%q, %q) = %v, want %v", tc.mime, tc.ext, text, tc.text)
		}
		if safeBin != tc.safeBin {
			t.Errorf("isSafeBinaryAttachment(%q, %q) = %v, want %v", tc.mime, tc.ext, safeBin, tc.safeBin)
		}
		if accepted != tc.accepted {
			t.Errorf("acceptance(%q, %q) = %v, want %v", tc.mime, tc.ext, accepted, tc.accepted)
		}
	}
}

// TestPIIScanViaExtractScannableText verifies the full pipeline that the
// handler uses: ExtractScannableText → DetectPII. This is the integration
// point where format-specific extraction meets PII detection.
func TestPIIScanViaExtractScannableText(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		mime    string
		ext     string
		wantPII bool
	}{
		// Plaintext
		{"text with SSN", []byte("Hello, my SSN is 123-45-6789"), "text/plain", ".txt", true},
		{"clean text", []byte("No sensitive data here"), "text/plain", ".txt", false},

		// SVG
		{
			"SVG with phone",
			[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><text>Contact: 555-123-4567</text></svg>`),
			"image/svg+xml", ".svg", true,
		},
		{
			"SVG clean",
			[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><circle cx="50" cy="50" r="40"/></svg>`),
			"image/svg+xml", ".svg", false,
		},

		// Binary image — no false positives
		{
			"binary image bytes",
			[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0xFF, 0xD8, 0xFF},
			"image/png", ".png", false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text := service.ExtractScannableText(tc.data, tc.mime, tc.ext)
			detected, types := service.DetectPII(text)
			if detected != tc.wantPII {
				t.Errorf("ExtractScannableText+DetectPII: detected=%v (types=%v), want %v", detected, types, tc.wantPII)
			}
		})
	}
}
