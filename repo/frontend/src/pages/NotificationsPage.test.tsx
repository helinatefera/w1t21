// @vitest-environment happy-dom
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { NotificationsPage } from './NotificationsPage';
import { useAuthStore } from '../store/authStore';

vi.mock('../api/notifications', () => ({
  listNotifications: vi.fn().mockResolvedValue({
    data: {
      data: [
        { id: 'n-1', rendered_title: 'Order Confirmed', rendered_body: 'Your order has been confirmed', is_read: false, status: 'delivered', created_at: '2025-01-01T00:00:00Z' },
        { id: 'n-2', rendered_title: 'Order Completed', rendered_body: 'Your order is complete', is_read: true, status: 'delivered', created_at: '2025-01-02T00:00:00Z' },
      ],
      page: 1, page_size: 20, total_count: 2, total_pages: 1,
    },
  }),
  markRead: vi.fn().mockResolvedValue({}),
  markAllRead: vi.fn().mockResolvedValue({}),
  retryNotification: vi.fn().mockResolvedValue({}),
}));

function renderNotifications() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <NotificationsPage />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('NotificationsPage', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: { id: 'u-3', username: 'buyer1', display_name: 'Buyer', is_locked: false, created_at: '', updated_at: '' },
      roles: ['buyer'],
      isAuthenticated: true,
      isBootstrapping: false,
    });
  });

  it('renders the page heading', () => {
    renderNotifications();
    expect(screen.getByText('Notifications')).toBeDefined();
  });

  it('renders notification items after loading', async () => {
    renderNotifications();
    expect(await screen.findByText('Order Confirmed')).toBeDefined();
    expect(await screen.findByText('Order Completed')).toBeDefined();
  });
});
