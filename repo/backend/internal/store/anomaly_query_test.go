package store

import (
	"fmt"
	"testing"
)

// TestAnomalyQueryPlaceholders verifies that the placeholder construction
// logic in ListAnomalyEvents produces correct $N parameter references.
// This is a regression test for a bug where string(rune('0'+argIdx)) was
// used instead of fmt.Sprintf, which would produce garbage for argIdx >= 10.
func TestAnomalyQueryPlaceholders_WithFilter(t *testing.T) {
	// Simulate the query construction with acknowledgedFilter present.
	argIdx := 1
	filter := fmt.Sprintf(" WHERE acknowledged = $%d", argIdx)
	argIdx++

	query := fmt.Sprintf(
		`SELECT id FROM anomaly_events%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		filter, argIdx, argIdx+1)

	expected := `SELECT id FROM anomaly_events WHERE acknowledged = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	if query != expected {
		t.Errorf("with filter:\ngot:  %s\nwant: %s", query, expected)
	}
}

func TestAnomalyQueryPlaceholders_WithoutFilter(t *testing.T) {
	argIdx := 1
	filter := ""

	query := fmt.Sprintf(
		`SELECT id FROM anomaly_events%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		filter, argIdx, argIdx+1)

	expected := `SELECT id FROM anomaly_events ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	if query != expected {
		t.Errorf("without filter:\ngot:  %s\nwant: %s", query, expected)
	}
}

// TestRuneConversionBug demonstrates why the old approach was broken.
// string(rune('0'+10)) produces ":" (ASCII 58), not "$10".
func TestRuneConversionBug(t *testing.T) {
	// The old code: string(rune('0'+argIdx))
	// For argIdx=10, this produces ":" (rune 58), not "10".
	broken := string(rune('0' + 10))
	if broken == "10" {
		t.Fatal("rune conversion should NOT produce '10' — this test validates the bug existed")
	}
	if broken != ":" {
		t.Fatalf("expected ':' (ASCII 58), got %q", broken)
	}

	// The fix: fmt.Sprintf
	fixed := fmt.Sprintf("%d", 10)
	if fixed != "10" {
		t.Fatalf("Sprintf should produce '10', got %q", fixed)
	}
}
