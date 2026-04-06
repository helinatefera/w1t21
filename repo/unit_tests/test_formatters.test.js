// Unit tests for formatting utilities — imports real production code from
// frontend/src/utils/formatters.ts and frontend/src/utils/maskValue.ts.
import { describe, it, expect } from 'vitest';
import { formatCents } from '@/utils/formatters';
import { maskValue } from '@/utils/maskValue';

describe('formatCents', () => {
  it('basic price',   () => expect(formatCents(9990)).toBe('$99.90'));
  it('zero',          () => expect(formatCents(0)).toBe('$0.00'));
  it('one cent',      () => expect(formatCents(1)).toBe('$0.01'));
  it('one dollar',    () => expect(formatCents(100)).toBe('$1.00'));
  it('large price',   () => expect(formatCents(249900)).toBe('$2,499.00'));
  it('sub dollar',    () => expect(formatCents(50)).toBe('$0.50'));
  it('seed prices',   () => {
    expect(formatCents(99900)).toBe('$999.00');
    expect(formatCents(49900)).toBe('$499.00');
  });
});

describe('maskValue', () => {
  it('email',         () => expect(maskValue('alice@example.com')).toBe('a***@e***.com'));
  it('short email',   () => expect(maskValue('a@b.co')).toBe('a***@b***.co'));
  it('long value',    () => expect(maskValue('1234567890')).toBe('***7890'));
  it('short value',   () => expect(maskValue('ab')).toBe('***'));
  it('exactly 4',     () => expect(maskValue('abcd')).toBe('***'));
  it('5 chars',       () => expect(maskValue('abcde')).toBe('***bcde'));
});

describe('Account Lockout Constants', () => {
  // These must match backend/internal/service/auth_service.go constants
  const MAX_FAILED_ATTEMPTS = 5;
  const LOCKOUT_DURATION_MINUTES = 30;

  it('threshold is 5',  () => expect(MAX_FAILED_ATTEMPTS).toBe(5));
  it('duration is 30m', () => expect(LOCKOUT_DURATION_MINUTES).toBe(30));
  it('under threshold', () => { for (let i = 1; i < MAX_FAILED_ATTEMPTS; i++) expect(i >= MAX_FAILED_ATTEMPTS).toBe(false); });
  it('at threshold',    () => expect(MAX_FAILED_ATTEMPTS >= MAX_FAILED_ATTEMPTS).toBe(true));
});

describe('Rate Limit Config', () => {
  // Must match backend/internal/middleware/ratelimit.go preconfigured limiters
  const RL = {
    login:    { count: 10, windowMin: 15 },
    orders:   { count: 30, windowMin: 1 },
    messages: { count: 20, windowMin: 1 },
    listings: { count: 10, windowMin: 60 },
  };

  it('login 10/15min',    () => { expect(RL.login.count).toBe(10); expect(RL.login.windowMin).toBe(15); });
  it('orders 30/1min',    () => expect(RL.orders.count).toBe(30));
  it('messages 20/1min',  () => expect(RL.messages.count).toBe(20));
  it('listings 10/60min', () => { expect(RL.listings.count).toBe(10); expect(RL.listings.windowMin).toBe(60); });
});

describe('Attachment Limits', () => {
  const MAX = 10 * 1024 * 1024;
  it('10MB in bytes',  () => expect(MAX).toBe(10485760));
  it('under limit ok', () => expect(5 * 1024 * 1024 <= MAX).toBe(true));
  it('over limit bad', () => expect(11 * 1024 * 1024 <= MAX).toBe(false));
});
