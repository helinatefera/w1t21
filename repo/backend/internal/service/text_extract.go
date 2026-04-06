package service

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"io"
	"regexp"
	"strings"
)

// ExtractScannableText returns all human-readable text from an attachment,
// regardless of format. The result is suitable for PII regex scanning.
//
// Supported extraction strategies:
//   - PDF: decompresses FlateDecode streams and extracts text operators
//   - SVG/XML/text: returned as-is (already plaintext)
//   - Binary images: extracts printable ASCII runs (catches EXIF metadata,
//     embedded comments, and any cleartext strings)
func ExtractScannableText(data []byte, mime string, ext string) string {
	lower := strings.ToLower(ext)

	// Text and SVG files: content is already scannable.
	if strings.HasPrefix(mime, "text/") || lower == ".csv" || lower == ".txt" ||
		mime == "image/svg+xml" || lower == ".svg" {
		return string(data)
	}

	// PDF: extract text from compressed and uncompressed streams.
	if mime == "application/pdf" || lower == ".pdf" {
		return extractPDFText(data)
	}

	// All other binary formats: extract printable ASCII runs.
	// This catches EXIF metadata in JPEGs, PNG text chunks, etc.
	return extractASCIIStrings(data, 8)
}

// pdfStreamRe matches the content between "stream" and "endstream" markers.
var pdfStreamRe = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)

// pdfTextOpRe extracts text from PDF text-showing operators:
//   - (text) Tj  — show string
//   - [(text)] TJ — show string array
var pdfTextOpRe = regexp.MustCompile(`\(([^)]*)\)`)

func extractPDFText(data []byte) string {
	var parts []string

	// 1. Collect text from uncompressed content (the raw bytes may
	//    contain plaintext BT...ET blocks or other readable strings).
	parts = append(parts, extractASCIIStrings(data, 8))

	// 2. Find and decompress FlateDecode streams.
	//    Look for /FlateDecode in the object preceding each stream.
	streams := pdfStreamRe.FindAllSubmatchIndex(data, -1)
	for _, loc := range streams {
		if len(loc) < 4 {
			continue
		}
		streamBytes := data[loc[2]:loc[3]]

		// Check if this stream uses FlateDecode by looking backwards
		// for the filter declaration in the same object.
		searchStart := loc[0] - 512
		if searchStart < 0 {
			searchStart = 0
		}
		header := data[searchStart:loc[0]]
		if !bytes.Contains(header, []byte("/FlateDecode")) {
			// Uncompressed stream — scan as-is.
			parts = append(parts, string(streamBytes))
			continue
		}

		// Decompress with zlib (PDF FlateDecode = zlib wrapper around deflate).
		decompressed := decompressFlate(streamBytes)
		if len(decompressed) == 0 {
			continue
		}

		// Extract text from PDF text operators in the decompressed content.
		matches := pdfTextOpRe.FindAllSubmatch(decompressed, -1)
		for _, m := range matches {
			if len(m) >= 2 {
				parts = append(parts, string(m[1]))
			}
		}

		// Also scan the raw decompressed bytes for PII patterns that
		// might appear outside of formal text operators.
		parts = append(parts, extractASCIIStrings(decompressed, 8))
	}

	return strings.Join(parts, " ")
}

// decompressFlate attempts zlib decompression first (standard PDF FlateDecode),
// then falls back to raw deflate.
func decompressFlate(data []byte) []byte {
	// Try zlib (PDF standard for FlateDecode).
	if r, err := zlib.NewReader(bytes.NewReader(data)); err == nil {
		defer r.Close()
		out, err := io.ReadAll(io.LimitReader(r, 10*1024*1024))
		if err == nil && len(out) > 0 {
			return out
		}
	}

	// Fallback: raw deflate.
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()
	out, err := io.ReadAll(io.LimitReader(r, 10*1024*1024))
	if err == nil {
		return out
	}
	return nil
}

// extractASCIIStrings extracts contiguous runs of printable ASCII characters
// that are at least minLen bytes long. This catches text embedded in EXIF
// metadata, PNG tEXt chunks, PDF cleartext, and similar structures.
func extractASCIIStrings(data []byte, minLen int) string {
	var parts []string
	var current []byte

	for _, b := range data {
		if b >= 0x20 && b < 0x7F {
			current = append(current, b)
		} else {
			if len(current) >= minLen {
				parts = append(parts, string(current))
			}
			current = current[:0]
		}
	}
	if len(current) >= minLen {
		parts = append(parts, string(current))
	}
	return strings.Join(parts, " ")
}
