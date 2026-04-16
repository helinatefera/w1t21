// @vitest-environment happy-dom
import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MaskedField } from './MaskedField';

describe('MaskedField component', () => {
  it('renders masked value by default', () => {
    render(<MaskedField value="secret@email.com" />);
    expect(screen.queryByText('secret@email.com')).toBeNull();
    expect(screen.getByText('Reveal')).toBeDefined();
  });

  it('reveals value when Reveal is clicked', () => {
    render(<MaskedField value="secret@email.com" />);
    fireEvent.click(screen.getByText('Reveal'));
    expect(screen.getByText('secret@email.com')).toBeDefined();
    expect(screen.getByText('Hide')).toBeDefined();
  });

  it('hides value again when Hide is clicked', () => {
    render(<MaskedField value="secret@email.com" />);
    fireEvent.click(screen.getByText('Reveal'));
    fireEvent.click(screen.getByText('Hide'));
    expect(screen.queryByText('secret@email.com')).toBeNull();
    expect(screen.getByText('Reveal')).toBeDefined();
  });

  it('renders label when provided', () => {
    render(<MaskedField value="test" label="Email" />);
    expect(screen.getByText('Email:')).toBeDefined();
  });

  it('returns null for empty value', () => {
    const { container } = render(<MaskedField value="" />);
    expect(container.innerHTML).toBe('');
  });
});
