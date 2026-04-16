// @vitest-environment happy-dom
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CatalogPage } from './CatalogPage';
import { useAuthStore } from '../store/authStore';
import { useABStore } from '../store/abStore';

vi.mock('../api/collectibles', () => ({
  listCollectibles: vi.fn().mockResolvedValue({
    data: {
      data: [
        { id: 'c-1', title: 'Dragon NFT', description: 'A rare dragon', price_cents: 9990, currency: 'USD', view_count: 42, status: 'published' },
        { id: 'c-2', title: 'Cyber Portrait', description: 'Limited edition', price_cents: 24990, currency: 'USD', view_count: 18, status: 'published' },
      ],
      page: 1, page_size: 20, total_count: 2, total_pages: 1,
    },
  }),
}));

function renderCatalog() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <CatalogPage />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('CatalogPage', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: { id: 'u-1', username: 'buyer1', display_name: 'Buyer', is_locked: false, created_at: '', updated_at: '' },
      roles: ['buyer'],
      isAuthenticated: true,
      isBootstrapping: false,
    });
    useABStore.setState({ assignments: [] });
  });

  it('renders the page heading', () => {
    renderCatalog();
    expect(screen.getByText('Catalog')).toBeDefined();
  });

  it('shows loading state initially', () => {
    renderCatalog();
    expect(screen.getByText('Loading...')).toBeDefined();
  });

  it('buyer does not see Add Collectible button', () => {
    renderCatalog();
    expect(screen.queryByText('+ Add Collectible')).toBeNull();
  });

  it('seller sees Add Collectible button', () => {
    useAuthStore.setState({ roles: ['seller'] });
    renderCatalog();
    expect(screen.getByText('+ Add Collectible')).toBeDefined();
  });

  it('renders collectible items after loading', async () => {
    renderCatalog();
    expect(await screen.findByText('Dragon NFT')).toBeDefined();
    expect(await screen.findByText('Cyber Portrait')).toBeDefined();
  });

  it('displays price formatted correctly', async () => {
    renderCatalog();
    expect(await screen.findByText('$99.90')).toBeDefined();
  });

  it('displays view count', async () => {
    renderCatalog();
    expect(await screen.findByText('42 views')).toBeDefined();
  });
});
