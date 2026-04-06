// Error code tests have been replaced by Go tests that exercise the real
// mapError function, sentinel errors, and error response structure.
//
// See:
//   backend/internal/dto/error_codes_test.go    — sentinel distinctness, code format, 4xx mapping
//   backend/internal/handler/handler_integration_test.go — mapError via httptest
//
// Run with: cd backend && go test ./internal/dto/ ./internal/handler/ -run "Error|MapError" -v
import { describe, it, expect } from 'vitest';

describe('Error code tests (see Go tests)', () => {
  it('error mapping is tested in Go against real handler code — see backend/internal/handler/handler_integration_test.go', () => {
    expect(true).toBe(true);
  });
});
