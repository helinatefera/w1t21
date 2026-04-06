// Unit tests for PII detection — imports real production code from
// frontend/src/utils/piiDetector.ts (which must match backend/internal/service/pii.go).
import { describe, it, expect } from 'vitest';
import { detectPII } from '@/utils/piiDetector';

describe('SSN Detection', () => {
  it('with dashes',        () => expect(detectPII('SSN: 123-45-6789').types).toContain('SSN'));
  it('without dashes',     () => expect(detectPII('SSN: 123456789').types).toContain('SSN'));
  it('partial dashes',     () => expect(detectPII('12345-6789').types).toContain('SSN'));
  it('in middle of text',  () => expect(detectPII('My number is 123-45-6789 ok').types).toContain('SSN'));
  it('at start',           () => expect(detectPII('123-45-6789 is my SSN').types).toContain('SSN'));
  it('at end',             () => expect(detectPII('SSN 123-45-6789').types).toContain('SSN'));
});

describe('Phone Detection', () => {
  it('with dashes',        () => expect(detectPII('555-123-4567').types).toContain('phone number'));
  it('with dots',          () => expect(detectPII('555.123.4567').types).toContain('phone number'));
  it('with parens',        () => expect(detectPII('(555) 123-4567').types).toContain('phone number'));
  it('with country code',  () => expect(detectPII('+1 555-123-4567').types).toContain('phone number'));
  it('with spaces',        () => expect(detectPII('555 123 4567').types).toContain('phone number'));
});

describe('Email Detection', () => {
  it('basic',              () => expect(detectPII('user@example.com').types).toContain('email address'));
  it('plus addressing',    () => expect(detectPII('user+tag@example.com').types).toContain('email address'));
  it('subdomain',          () => expect(detectPII('u@sub.example.com').types).toContain('email address'));
});

describe('Multiple PII Types', () => {
  it('detects SSN + phone + email together', () => {
    const { types } = detectPII('SSN 123-45-6789 phone 555-123-4567 email a@b.com');
    expect(types).toContain('SSN');
    expect(types).toContain('phone number');
    expect(types).toContain('email address');
  });
});

describe('No PII', () => {
  it('clean text',    () => expect(detectPII('Hello world').detected).toBe(false));
  it('UUID',          () => expect(detectPII('550e8400-e29b-41d4-a716-446655440000').detected).toBe(false));
  it('price',         () => expect(detectPII('$99.99').detected).toBe(false));
});
