package service

import (
	"bytes"
	"compress/zlib"
	"strings"
	"testing"
)

func TestExtractScannableText_PlainText(t *testing.T) {
	data := []byte("Hello, my SSN is 123-45-6789")
	got := ExtractScannableText(data, "text/plain", ".txt")
	if got != string(data) {
		t.Errorf("expected raw text, got %q", got)
	}
}

func TestExtractScannableText_CSV(t *testing.T) {
	data := []byte("name,ssn\nJohn,123-45-6789")
	got := ExtractScannableText(data, "application/octet-stream", ".csv")
	if got != string(data) {
		t.Errorf("expected raw CSV, got %q", got)
	}
}

func TestExtractScannableText_SVG(t *testing.T) {
	data := []byte(`<svg><text>Call 555-123-4567</text></svg>`)
	got := ExtractScannableText(data, "image/svg+xml", ".svg")
	if got != string(data) {
		t.Errorf("expected raw SVG, got %q", got)
	}
}

func TestExtractScannableText_SVGPIIDetected(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><text>SSN: 123-45-6789</text></svg>`)
	text := ExtractScannableText(data, "image/svg+xml", ".svg")
	detected, types := DetectPII(text)
	if !detected {
		t.Fatal("expected PII detection in SVG")
	}
	if !sliceContains(types, "SSN") {
		t.Errorf("expected SSN in types, got %v", types)
	}
}

// TestExtractScannableText_PDFWithCompressedStream builds a minimal PDF
// with a zlib-compressed text stream containing an SSN, then verifies
// the extractor decompresses and finds it.
func TestExtractScannableText_PDFWithCompressedStream(t *testing.T) {
	// Build a minimal PDF with a FlateDecode stream containing PII.
	textContent := "BT /F1 12 Tf (SSN: 123-45-6789) Tj ET"

	var compressed bytes.Buffer
	w, err := zlib.NewWriterLevel(&compressed, zlib.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte(textContent))
	w.Close()

	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	pdf.WriteString("1 0 obj\n<< /Length " + intStr(compressed.Len()) + " /Filter /FlateDecode >>\n")
	pdf.WriteString("stream\n")
	pdf.Write(compressed.Bytes())
	pdf.WriteString("\nendstream\nendobj\n")

	text := ExtractScannableText(pdf.Bytes(), "application/pdf", ".pdf")
	detected, types := DetectPII(text)
	if !detected {
		t.Fatalf("expected PII detection in compressed PDF stream, extracted text: %q", text)
	}
	if !sliceContains(types, "SSN") {
		t.Errorf("expected SSN in types, got %v", types)
	}
}

// TestExtractScannableText_PDFUncompressed tests a PDF with cleartext content.
func TestExtractScannableText_PDFUncompressed(t *testing.T) {
	pdf := "%PDF-1.4\n1 0 obj\n<< /Length 40 >>\nstream\nBT (Phone: 555-123-4567) Tj ET\nendstream\nendobj\n"
	text := ExtractScannableText([]byte(pdf), "application/pdf", ".pdf")
	detected, types := DetectPII(text)
	if !detected {
		t.Fatalf("expected PII detection in uncompressed PDF, extracted text: %q", text)
	}
	if !sliceContains(types, "phone number") {
		t.Errorf("expected phone number in types, got %v", types)
	}
}

func TestExtractScannableText_PDFClean(t *testing.T) {
	pdf := "%PDF-1.4\n1 0 obj\n<< /Length 50 >>\nstream\nBT (Certificate of Authenticity) Tj ET\nendstream\nendobj\n"
	text := ExtractScannableText([]byte(pdf), "application/pdf", ".pdf")
	detected, _ := DetectPII(text)
	if detected {
		t.Errorf("expected no PII in clean PDF, extracted: %q", text)
	}
}

// TestExtractScannableText_BinaryImage verifies that actual image-like bytes
// don't produce false-positive PII matches.
func TestExtractScannableText_BinaryImage(t *testing.T) {
	// Simulated PNG: header + IDAT-like binary data (non-sequential so the
	// extracted ASCII strings don't accidentally form digit sequences that
	// match SSN/phone patterns).
	data := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG magic
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR
		0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, // binary pixel data
		0xE0, 0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7,
		0xD0, 0xD1, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7,
	}
	text := ExtractScannableText(data, "image/png", ".png")
	detected, _ := DetectPII(text)
	if detected {
		t.Errorf("binary image bytes should not trigger PII detection, extracted: %q", text)
	}
}

// TestExtractScannableText_JPEGWithEXIF verifies that PII embedded in
// EXIF-like ASCII metadata within a JPEG is detected.
func TestExtractScannableText_JPEGWithEXIF(t *testing.T) {
	// Simulate JPEG with EXIF-like metadata containing an email.
	var data []byte
	data = append(data, 0xFF, 0xD8, 0xFF, 0xE1) // JPEG SOI + APP1 marker
	exif := []byte("Exif\x00\x00Author: user@example.com and phone 555-123-4567 end-of-metadata")
	data = append(data, byte(len(exif)>>8), byte(len(exif)&0xFF))
	data = append(data, exif...)
	data = append(data, 0xFF, 0xD9) // JPEG EOI

	text := ExtractScannableText(data, "image/jpeg", ".jpg")
	detected, types := DetectPII(text)
	if !detected {
		t.Fatalf("expected PII detection in JPEG EXIF, extracted: %q", text)
	}
	if !sliceContains(types, "email address") {
		t.Errorf("expected email in types, got %v", types)
	}
}

func TestExtractASCIIStrings(t *testing.T) {
	// Mix of binary and ASCII content.
	data := []byte{0x00, 0x01}
	data = append(data, []byte("Hello World!")...)
	data = append(data, 0x00, 0xFF, 0xFE)
	data = append(data, []byte("short")...) // only 5 chars, below minLen=8
	data = append(data, 0x00)
	data = append(data, []byte("LongerString")...) // 12 chars

	got := extractASCIIStrings(data, 8)
	if !strings.Contains(got, "Hello World!") {
		t.Errorf("expected 'Hello World!' in output, got %q", got)
	}
	if strings.Contains(got, "short") {
		t.Errorf("'short' (5 chars) should be filtered out at minLen=8, got %q", got)
	}
	if !strings.Contains(got, "LongerString") {
		t.Errorf("expected 'LongerString' in output, got %q", got)
	}
}

// intStr is a helper to avoid importing strconv in tests.
func intStr(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
