// @vitest-environment happy-dom
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { LoginPage } from './LoginPage';
import { useAuthStore } from '../store/authStore';

function renderLoginPage() {
  return render(
    <MemoryRouter>
      <LoginPage />
    </MemoryRouter>
  );
}

describe('LoginPage', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      roles: [],
      isAuthenticated: false,
      isBootstrapping: false,
    });
  });

  it('renders the login form', () => {
    renderLoginPage();
    expect(screen.getByText('LedgerMint')).toBeDefined();
    expect(screen.getByText('Digital Collectibles Exchange')).toBeDefined();
    expect(screen.getByText('Username')).toBeDefined();
    expect(screen.getByText('Password')).toBeDefined();
    expect(screen.getByText('Sign In')).toBeDefined();
  });

  it('has a submit button', () => {
    renderLoginPage();
    const button = screen.getByText('Sign In') as HTMLButtonElement;
    expect(button.tagName).toBe('BUTTON');
    expect(button.type).toBe('submit');
  });

  it('has username and password inputs', () => {
    renderLoginPage();
    const inputs = screen.getAllByRole('textbox');
    expect(inputs.length).toBeGreaterThanOrEqual(1);
    // Password input won't be role=textbox, check via querySelector
    const { container } = renderLoginPage();
    const passwordInput = container.querySelector('input[type="password"]');
    expect(passwordInput).not.toBeNull();
  });

  it('password field is type=password', () => {
    const { container } = renderLoginPage();
    const passwordInput = container.querySelector('input[type="password"]') as HTMLInputElement;
    expect(passwordInput).not.toBeNull();
    expect(passwordInput.type).toBe('password');
  });
});
