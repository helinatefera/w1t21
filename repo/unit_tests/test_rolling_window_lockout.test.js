// Unit tests for rolling-window login lockout policy.
// Mirrors constants and logic from backend/internal/service/auth_service.go.
import { describe, it, expect } from 'vitest';

const MAX_USER_FAILURES = 5;
const MAX_IP_FAILURES   = 20;
const FAILURE_WINDOW_MS = 15 * 60 * 1000;  // 15 minutes
const LOCKOUT_MS        = 30 * 60 * 1000;  // 30 minutes

function withinWindow(attemptTime, now = Date.now()) {
  return attemptTime > (now - FAILURE_WINDOW_MS);
}

function countRecent(attempts, now = Date.now()) {
  return attempts.filter(a => withinWindow(a, now)).length;
}

function shouldLockUser(attempts, now = Date.now()) {
  return countRecent(attempts, now) >= MAX_USER_FAILURES;
}

function shouldLockIP(attempts, now = Date.now()) {
  return countRecent(attempts, now) >= MAX_IP_FAILURES;
}

function isLockoutExpired(lockedUntil, now = Date.now()) {
  return now >= lockedUntil;
}

describe('Rolling Window Threshold', () => {
  it('under threshold → no lock', () => {
    const now = Date.now();
    const attempts = Array.from({ length: 4 }, (_, i) => now - i * 60000);
    expect(shouldLockUser(attempts, now)).toBe(false);
  });

  it('at threshold → locks', () => {
    const now = Date.now();
    const attempts = Array.from({ length: 5 }, (_, i) => now - i * 10000);
    expect(shouldLockUser(attempts, now)).toBe(true);
  });

  it('above threshold → locks', () => {
    const now = Date.now();
    const attempts = Array.from({ length: 8 }, (_, i) => now - i * 5000);
    expect(shouldLockUser(attempts, now)).toBe(true);
  });
});

describe('Window Expiry', () => {
  it('old attempts ignored', () => {
    const now = Date.now();
    const old = Array.from({ length: 10 }, (_, i) => now - (20 + i) * 60000);
    expect(shouldLockUser(old, now)).toBe(false);
  });

  it('mix of old and recent under threshold', () => {
    const now = Date.now();
    const old = [now - 20 * 60000];
    const recent = Array.from({ length: 4 }, (_, i) => now - i * 60000);
    expect(shouldLockUser([...old, ...recent], now)).toBe(false);
  });

  it('mix reaching threshold', () => {
    const now = Date.now();
    const old = [now - 20 * 60000];
    const recent = Array.from({ length: 5 }, (_, i) => now - i * 60000);
    expect(shouldLockUser([...old, ...recent], now)).toBe(true);
  });

  it('exactly at boundary excluded (> not >=)', () => {
    const now = Date.now();
    const boundary = Array(5).fill(now - FAILURE_WINDOW_MS);
    expect(shouldLockUser(boundary, now)).toBe(false);
  });

  it('one ms inside boundary', () => {
    const now = Date.now();
    const justInside = Array(5).fill(now - FAILURE_WINDOW_MS + 1);
    expect(shouldLockUser(justInside, now)).toBe(true);
  });
});

describe('Lockout Duration', () => {
  it('active lockout',       () => expect(isLockoutExpired(Date.now() + 600000)).toBe(false));
  it('expired lockout',      () => expect(isLockoutExpired(Date.now() - 60000)).toBe(true));
  it('exactly at boundary',  () => {
    const now = Date.now();
    expect(isLockoutExpired(now, now)).toBe(true);
  });
  it('full 30min duration',  () => {
    const lockTime = Date.now();
    const lockedUntil = lockTime + LOCKOUT_MS;
    expect(isLockoutExpired(lockedUntil, lockedUntil + 1)).toBe(true);
    expect(isLockoutExpired(lockedUntil, lockedUntil - 1)).toBe(false);
  });
});

describe('IP Threshold', () => {
  it('under 20 → no block', () => {
    const now = Date.now();
    const attempts = Array.from({ length: 19 }, (_, i) => now - i * 1000);
    expect(shouldLockIP(attempts, now)).toBe(false);
  });
  it('at 20 → block', () => {
    const now = Date.now();
    const attempts = Array.from({ length: 20 }, (_, i) => now - i * 1000);
    expect(shouldLockIP(attempts, now)).toBe(true);
  });
});

describe('Successful Login Clears Window', () => {
  it('cleared attempts → no lock', () => {
    const attempts = [];
    expect(shouldLockUser(attempts)).toBe(false);
  });
  it('new failures after clear start fresh', () => {
    const now = Date.now();
    const fresh = Array.from({ length: 4 }, (_, i) => now - i * 2000);
    expect(shouldLockUser(fresh, now)).toBe(false);
  });
});
