import { describe, it, expect, beforeEach } from 'vitest';
import { useAuthStore } from './authStore';

describe('authStore', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      roles: [],
      isAuthenticated: false,
      isBootstrapping: true,
    });
  });

  it('starts unauthenticated with bootstrapping=true', () => {
    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.isBootstrapping).toBe(true);
    expect(state.user).toBeNull();
    expect(state.roles).toEqual([]);
  });

  it('setAuth stores user and roles', () => {
    const user = {
      id: 'u-1',
      username: 'admin',
      display_name: 'Admin',
      is_locked: false,
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
    };

    useAuthStore.getState().setAuth(user, ['administrator']);

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(state.isBootstrapping).toBe(false);
    expect(state.user?.username).toBe('admin');
    expect(state.roles).toEqual(['administrator']);
  });

  it('clearAuth resets to unauthenticated', () => {
    const user = {
      id: 'u-1',
      username: 'admin',
      display_name: 'Admin',
      is_locked: false,
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
    };

    useAuthStore.getState().setAuth(user, ['administrator']);
    useAuthStore.getState().clearAuth();

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.isBootstrapping).toBe(false);
    expect(state.user).toBeNull();
    expect(state.roles).toEqual([]);
  });

  it('hasRole checks role membership', () => {
    const user = {
      id: 'u-1',
      username: 'seller1',
      display_name: 'Seller',
      is_locked: false,
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
    };

    useAuthStore.getState().setAuth(user, ['seller', 'buyer']);

    expect(useAuthStore.getState().hasRole('seller')).toBe(true);
    expect(useAuthStore.getState().hasRole('buyer')).toBe(true);
    expect(useAuthStore.getState().hasRole('administrator')).toBe(false);
  });

  it('hasRole returns false when unauthenticated', () => {
    expect(useAuthStore.getState().hasRole('administrator')).toBe(false);
  });
});
