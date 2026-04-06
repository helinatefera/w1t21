// Unit tests for order state machine transitions.
// Matches backend/internal/model/order.go AllowedTransitions.
import { describe, it, expect } from 'vitest';

const ALLOWED_TRANSITIONS = {
  pending:    ['confirmed', 'cancelled'],
  confirmed:  ['processing', 'cancelled'],
  processing: ['completed'],
  completed:  [],
  cancelled:  [],
};

const ALL_STATUSES = ['pending', 'confirmed', 'processing', 'completed', 'cancelled'];

function canTransition(from, to) {
  return (ALLOWED_TRANSITIONS[from] || []).includes(to);
}

describe('Order State Machine', () => {
  // Valid transitions
  it('pending → confirmed', () => expect(canTransition('pending', 'confirmed')).toBe(true));
  it('pending → cancelled', () => expect(canTransition('pending', 'cancelled')).toBe(true));
  it('confirmed → processing', () => expect(canTransition('confirmed', 'processing')).toBe(true));
  it('confirmed → cancelled', () => expect(canTransition('confirmed', 'cancelled')).toBe(true));
  it('processing → completed', () => expect(canTransition('processing', 'completed')).toBe(true));

  // Invalid transitions
  it('pending cannot skip to processing', () => expect(canTransition('pending', 'processing')).toBe(false));
  it('pending cannot skip to completed', () => expect(canTransition('pending', 'completed')).toBe(false));
  it('processing cannot be cancelled', () => expect(canTransition('processing', 'cancelled')).toBe(false));

  it('completed is terminal', () => {
    for (const s of ALL_STATUSES) expect(canTransition('completed', s)).toBe(false);
  });

  it('cancelled is terminal', () => {
    for (const s of ALL_STATUSES) expect(canTransition('cancelled', s)).toBe(false);
  });

  it('no self-transitions', () => {
    for (const s of ALL_STATUSES) expect(canTransition(s, s)).toBe(false);
  });

  it('no backward transitions', () => {
    expect(canTransition('confirmed', 'pending')).toBe(false);
    expect(canTransition('processing', 'confirmed')).toBe(false);
    expect(canTransition('processing', 'pending')).toBe(false);
    expect(canTransition('completed', 'processing')).toBe(false);
  });

  it('unknown status has no transitions', () => {
    expect(canTransition('unknown', 'pending')).toBe(false);
  });

  it('all statuses have entries', () => {
    for (const s of ALL_STATUSES) expect(ALLOWED_TRANSITIONS).toHaveProperty(s);
  });

  it('happy path: pending → confirmed → processing → completed', () => {
    const path = ['pending', 'confirmed', 'processing', 'completed'];
    for (let i = 0; i < path.length - 1; i++) {
      expect(canTransition(path[i], path[i + 1])).toBe(true);
    }
  });

  it('all transition targets are valid statuses', () => {
    for (const [, targets] of Object.entries(ALLOWED_TRANSITIONS)) {
      for (const t of targets) expect(ALL_STATUSES).toContain(t);
    }
  });
});
