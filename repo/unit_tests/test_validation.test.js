// Validation tests have been replaced by Go tests that exercise the real
// production validator (go-playground/validator/v10) against actual DTOs.
//
// See: backend/internal/dto/validation_test.go
//
// Run with: cd backend && go test ./internal/dto/ -run TestCreate -v
import { describe, it, expect } from 'vitest';

describe('Validation tests (see Go tests)', () => {
  it('validation logic is tested in Go against real DTOs — see backend/internal/dto/validation_test.go', () => {
    // This placeholder exists so vitest reports a reminder.
    // The real tests run via `go test` in CI and call the production
    // validator with every request DTO, checking required fields, min/max
    // bounds, oneof constraints, and email format validation.
    expect(true).toBe(true);
  });
});
