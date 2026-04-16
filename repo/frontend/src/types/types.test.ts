import { describe, it, expect } from 'vitest';
import type {
  User, AuthResponse, Collectible, Order, OrderStatus,
  Notification, NotificationPreferences, ABTest, ABTestAssignment,
  AnomalyEvent, IPRule, PaginatedResponse, FunnelResponse,
  ErrorResponse,
} from './index';

// These tests verify the TypeScript type contracts compile and match
// the expected structure. They catch drift between frontend types and
// the backend API response shapes.

describe('Type contracts', () => {
  it('User shape matches API response', () => {
    const user: User = {
      id: 'u-1',
      username: 'admin',
      display_name: 'Admin',
      is_locked: false,
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
    };
    expect(user.id).toBe('u-1');
    expect(user.is_locked).toBe(false);
  });

  it('AuthResponse includes user and roles', () => {
    const resp: AuthResponse = {
      user: {
        id: 'u-1', username: 'admin', display_name: 'Admin',
        is_locked: false, created_at: '', updated_at: '',
      },
      roles: ['administrator'],
    };
    expect(resp.roles).toContain('administrator');
  });

  it('Order has all required fields', () => {
    const order: Order = {
      id: 'o-1', idempotency_key: 'k-1', buyer_id: 'b-1',
      collectible_id: 'c-1', seller_id: 's-1', status: 'pending',
      price_snapshot_cents: 9990, created_at: '', updated_at: '',
    };
    expect(order.status).toBe('pending');
  });

  it('OrderStatus allows all valid values', () => {
    const statuses: OrderStatus[] = ['pending', 'confirmed', 'processing', 'completed', 'cancelled'];
    expect(statuses).toHaveLength(5);
  });

  it('NotificationPreferences has subscription_mode', () => {
    const prefs: NotificationPreferences = {
      user_id: 'u-1',
      preferences: { order_confirmed: true },
      subscription_mode: 'all_events',
    };
    expect(prefs.subscription_mode).toBe('all_events');
  });

  it('ABTestAssignment has test_name and variant', () => {
    const a: ABTestAssignment = { test_name: 'catalog_layout', variant: 'grid' };
    expect(a.test_name).toBe('catalog_layout');
    expect(a.variant).toBe('grid');
  });

  it('PaginatedResponse wraps data array', () => {
    const resp: PaginatedResponse<User> = {
      data: [], page: 1, page_size: 20, total_count: 0, total_pages: 0,
    };
    expect(resp.data).toEqual([]);
    expect(resp.page).toBe(1);
  });

  it('FunnelResponse has analytics fields', () => {
    const funnel: FunnelResponse = { views: 100, orders: 25, rate: 0.25, days: 7 };
    expect(funnel.rate).toBe(0.25);
  });

  it('ErrorResponse has error.code and error.message', () => {
    const err: ErrorResponse = {
      error: { code: 'ERR_NOT_FOUND', message: 'not found' },
    };
    expect(err.error.code).toBe('ERR_NOT_FOUND');
  });

  it('AnomalyEvent has acknowledged boolean', () => {
    const anomaly: AnomalyEvent = {
      id: 'a-1', user_id: 'u-1', anomaly_type: 'high_cancel_rate',
      details: {}, acknowledged: false, created_at: '',
    };
    expect(anomaly.acknowledged).toBe(false);
  });

  it('Collectible has price_cents as number', () => {
    const c: Collectible = {
      id: 'c-1', seller_id: 's-1', title: 'NFT', description: '',
      price_cents: 9990, currency: 'USD', status: 'published',
      view_count: 0, created_at: '', updated_at: '',
    };
    expect(typeof c.price_cents).toBe('number');
  });
});
