// @vitest-environment happy-dom
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ProtectedRoute } from './ProtectedRoute';
import { useAuthStore } from '../../store/authStore';

describe('ProtectedRoute component', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      roles: [],
      isAuthenticated: false,
      isBootstrapping: false,
    });
  });

  it('redirects to /login when not authenticated', () => {
    const { container } = render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <ProtectedRoute>
          <p>Protected Content</p>
        </ProtectedRoute>
      </MemoryRouter>
    );
    expect(screen.queryByText('Protected Content')).toBeNull();
  });

  it('renders children when authenticated', () => {
    useAuthStore.setState({
      user: { id: '1', username: 'admin', display_name: 'Admin', is_locked: false, created_at: '', updated_at: '' },
      roles: ['administrator'],
      isAuthenticated: true,
      isBootstrapping: false,
    });

    render(
      <MemoryRouter>
        <ProtectedRoute>
          <p>Protected Content</p>
        </ProtectedRoute>
      </MemoryRouter>
    );
    expect(screen.getByText('Protected Content')).toBeDefined();
  });

  it('renders nothing during bootstrap', () => {
    useAuthStore.setState({
      isBootstrapping: true,
      isAuthenticated: false,
    });

    const { container } = render(
      <MemoryRouter>
        <ProtectedRoute>
          <p>Loading</p>
        </ProtectedRoute>
      </MemoryRouter>
    );
    expect(screen.queryByText('Loading')).toBeNull();
  });

  it('redirects when user lacks required role', () => {
    useAuthStore.setState({
      user: { id: '1', username: 'buyer1', display_name: 'Buyer', is_locked: false, created_at: '', updated_at: '' },
      roles: ['buyer'],
      isAuthenticated: true,
      isBootstrapping: false,
    });

    render(
      <MemoryRouter initialEntries={['/admin']}>
        <ProtectedRoute roles={['administrator']}>
          <p>Admin Only</p>
        </ProtectedRoute>
      </MemoryRouter>
    );
    expect(screen.queryByText('Admin Only')).toBeNull();
  });

  it('renders when user has required role', () => {
    useAuthStore.setState({
      user: { id: '1', username: 'admin', display_name: 'Admin', is_locked: false, created_at: '', updated_at: '' },
      roles: ['administrator'],
      isAuthenticated: true,
      isBootstrapping: false,
    });

    render(
      <MemoryRouter>
        <ProtectedRoute roles={['administrator']}>
          <p>Admin Content</p>
        </ProtectedRoute>
      </MemoryRouter>
    );
    expect(screen.getByText('Admin Content')).toBeDefined();
  });

  it('renders when user has one of multiple allowed roles', () => {
    useAuthStore.setState({
      user: { id: '1', username: 'analyst1', display_name: 'Analyst', is_locked: false, created_at: '', updated_at: '' },
      roles: ['compliance_analyst'],
      isAuthenticated: true,
      isBootstrapping: false,
    });

    render(
      <MemoryRouter>
        <ProtectedRoute roles={['administrator', 'compliance_analyst']}>
          <p>Analytics</p>
        </ProtectedRoute>
      </MemoryRouter>
    );
    expect(screen.getByText('Analytics')).toBeDefined();
  });
});
