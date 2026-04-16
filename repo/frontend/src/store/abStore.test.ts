import { describe, it, expect, beforeEach } from 'vitest';
import { useABStore } from './abStore';

describe('abStore', () => {
  beforeEach(() => {
    useABStore.setState({ assignments: [] });
  });

  it('starts with empty assignments', () => {
    expect(useABStore.getState().assignments).toEqual([]);
  });

  it('setAssignments stores assignments', () => {
    useABStore.getState().setAssignments([
      { test_name: 'catalog_layout', variant: 'grid' },
      { test_name: 'checkout_flow', variant: 'express' },
    ]);

    const state = useABStore.getState();
    expect(state.assignments).toHaveLength(2);
    expect(state.assignments[0].test_name).toBe('catalog_layout');
  });

  it('getVariant returns correct variant for known test', () => {
    useABStore.getState().setAssignments([
      { test_name: 'catalog_layout', variant: 'list' },
    ]);

    expect(useABStore.getState().getVariant('catalog_layout')).toBe('list');
  });

  it('getVariant returns null for unknown test', () => {
    useABStore.getState().setAssignments([
      { test_name: 'catalog_layout', variant: 'grid' },
    ]);

    expect(useABStore.getState().getVariant('nonexistent')).toBeNull();
  });

  it('getVariant returns null when no assignments', () => {
    expect(useABStore.getState().getVariant('catalog_layout')).toBeNull();
  });

  it('setAssignments replaces all previous assignments', () => {
    useABStore.getState().setAssignments([
      { test_name: 'old_test', variant: 'A' },
    ]);
    useABStore.getState().setAssignments([
      { test_name: 'new_test', variant: 'B' },
    ]);

    const state = useABStore.getState();
    expect(state.assignments).toHaveLength(1);
    expect(state.assignments[0].test_name).toBe('new_test');
    expect(state.getVariant('old_test')).toBeNull();
  });
});
