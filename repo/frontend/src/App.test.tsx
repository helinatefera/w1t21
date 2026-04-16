// @vitest-environment happy-dom
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { App } from './App';
import { useAuthStore } from './store/authStore';

describe('App component', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      roles: [],
      isAuthenticated: false,
      isBootstrapping: false,
    });
  });

  it('renders login page when unauthenticated', () => {
    render(<App />);
    expect(screen.getByText('LedgerMint')).toBeDefined();
    expect(screen.getByText('Sign In')).toBeDefined();
  });

  it('renders login form with input fields', () => {
    render(<App />);
    expect(screen.getByText('Username')).toBeDefined();
    expect(screen.getByText('Password')).toBeDefined();
  });

  it('shows LedgerMint branding', () => {
    render(<App />);
    expect(screen.getByText('Digital Collectibles Exchange')).toBeDefined();
  });
});
