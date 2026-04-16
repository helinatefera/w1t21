// @vitest-environment happy-dom
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Modal } from './Modal';

describe('Modal component', () => {
  it('renders title and children when open', () => {
    render(
      <Modal open={true} onClose={() => {}} title="Test Modal">
        <p>Modal content here</p>
      </Modal>
    );
    expect(screen.getByText('Test Modal')).toBeDefined();
    expect(screen.getByText('Modal content here')).toBeDefined();
  });

  it('does not render content when closed', () => {
    render(
      <Modal open={false} onClose={() => {}} title="Hidden Modal">
        <p>Should not appear</p>
      </Modal>
    );
    expect(screen.queryByText('Hidden Modal')).toBeNull();
    expect(screen.queryByText('Should not appear')).toBeNull();
  });

  it('calls onClose when close button is clicked', () => {
    const onClose = vi.fn();
    render(
      <Modal open={true} onClose={onClose} title="Closable">
        <p>Content</p>
      </Modal>
    );
    const closeButton = screen.getByLabelText('Close');
    fireEvent.click(closeButton);
    expect(onClose).toHaveBeenCalled();
  });
});
